package smt

import (
	"bytes"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// envInt reads a positive integer from the environment, falling back to def.
func envInt(t *testing.T, name string, def int) int {
	t.Helper()
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s=%q: %v", name, raw, err)
	}
	return parsed
}

// C1 at the scale the protocol actually permits.
//
// target_num_relays on mainnet is 100000: that is the leaf target per service
// per session across the whole network, held there by the mining difficulty.
// With num_suppliers_per_session=50 a typical claim carries ~2000 leaves, and
// the ceiling for a single claim — one supplier taking a service's entire
// traffic — is of the order of 100k leaves. So 100k is the default here, and
// the runs that back the report also cover 500k as a 5x safety margin.
//
// Millions of leaves are deliberately NOT the default: that regime only shows
// up on a localnet running at base difficulty, without the brake Pocket
// applies in production, and testing it costs hours to exercise a state the
// network does not allow.
//
// Roots are sampled rather than compared on every insert, which would be O(N)
// root computations on top of O(N) inserts. The final comparison is mandatory.
// On divergence the exact insert index is reported.
//
// Memory: this holds TWO tries and TWO node stores at once — roughly 2.6 GB at
// 500k leaves. Override with SMT_SCALE_LEAVES.
func TestCompactScale_C1_RootIdenticalAtProtocolMaximum(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test; skipped under -short")
	}

	numLeaves := envInt(t, "SMT_SCALE_LEAVES", 100_000)
	sampleEvery := envInt(t, "SMT_SCALE_SAMPLE", 10_000)

	for _, seed := range []int64{1, 1337} {
		t.Run("seed="+strconv.FormatInt(seed, 10), func(t *testing.T) {
			trieA := newPoktrollSpecSMST(simplemap.NewSimpleMap()) // compacted
			trieB := newPoktrollSpecSMST(simplemap.NewSimpleMap()) // plain

			rnd := rand.New(rand.NewSource(seed))
			keys := make([][]byte, 0, numLeaves/4)
			samples := 0

			for i := 0; i < numLeaves; i++ {
				// One in five operations reuses an earlier key, which orphans a
				// persisted (and, in trie A, compacted) leaf.
				var key []byte
				if len(keys) > 0 && rnd.Intn(5) == 0 {
					key = keys[rnd.Intn(len(keys))]
				} else {
					key = make([]byte, 32)
					rnd.Read(key) //nolint:errcheck // math/rand Read never fails
					if len(keys) < cap(keys) {
						keys = append(keys, key)
					}
				}
				value := make([]byte, 900)
				rnd.Read(value) //nolint:errcheck // math/rand Read never fails
				weight := uint64(rnd.Intn(1000) + 1)

				requireNoError(t, trieA.Update(key, value, weight), "A.Update")
				requireNoError(t, trieB.Update(key, value, weight), "B.Update")
				requireNoError(t, trieA.Commit(), "A.Commit")
				requireNoError(t, trieB.Commit(), "B.Commit")
				if _, err := trieA.CompactPersistedLeaves(); err != nil {
					t.Fatalf("insert %d: CompactPersistedLeaves: %v", i, err)
				}

				if i%sampleEvery == 0 {
					samples++
					if !bytes.Equal(trieA.Root(), trieB.Root()) {
						t.Fatalf("root diverged at insert %d (N=%d)\n compacted: %x\n plain:     %x",
							i, i+1, trieA.Root(), trieB.Root())
					}
				}
			}

			// Mandatory final comparison.
			if !bytes.Equal(trieA.Root(), trieB.Root()) {
				t.Fatalf("root diverged at the final insert (N=%d)\n compacted: %x\n plain:     %x",
					numLeaves, trieA.Root(), trieB.Root())
			}
			if trieA.MustSum() != trieB.MustSum() || trieA.MustCount() != trieB.MustCount() {
				t.Fatalf("sum/count diverged: compacted=(%d,%d) plain=(%d,%d)",
					trieA.MustSum(), trieA.MustCount(), trieB.MustSum(), trieB.MustCount())
			}

			// Control: a compacted trie that quietly stopped compacting would
			// pass every assertion above.
			held, total := residentLeavesHoldingValues(trieA.root)
			if held != 0 {
				t.Fatalf("%d of %d leaves in the compacted trie still hold a value", held, total)
			}
			t.Logf("N=%d seed=%d: %d root samples compared, final root equal, %d resident leaves compacted",
				numLeaves, seed, samples, total)
		})
	}
}

// C1 for the fleet shape: the miner runs many session tries at once, not one
// enormous one. The measured fleet carries roughly 350 claims in parallel with
// num_suppliers_per_session=50, so a thousand concurrent tries is the shape
// that matters.
//
// Only the compacted tries stay resident. Each plain twin is built, its root
// recorded, and then dropped, because keeping a thousand plain twins alive
// would measure nothing except the machine running out of memory.
//
// Leaves per trie default to 1k-3k rather than the 1k-10k asked for: at 10k the
// thousand IN-PROCESS node stores alone come to roughly 8 GB, and those bytes
// live in Redis in production, not in the miner. Raise with SMT_FLEET_MAXLEAVES.
func TestCompactScale_C1_ThousandConcurrentTries(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test; skipped under -short")
	}

	numTries := envInt(t, "SMT_FLEET_TRIES", 1000)
	minLeaves := envInt(t, "SMT_FLEET_MINLEAVES", 1000)
	maxLeaves := envInt(t, "SMT_FLEET_MAXLEAVES", 3000)

	type result struct {
		leaves    int
		compacted int
		held      int
		total     int
	}

	resident := make([]*SMST, numTries)
	results := make([]result, numTries)
	errs := make([]error, numTries)

	// Bound how many tries are built at once. Every builder holds a plain twin
	// and its store while it runs, so a thousand unbounded builders peak at
	// tens of GB even though what stays resident afterwards is far smaller.
	// The fleet property under test is that a thousand compacted tries COEXIST,
	// not that a thousand are built simultaneously.
	buildSlots := make(chan struct{}, runtime.NumCPU())

	var wg sync.WaitGroup
	for i := 0; i < numTries; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			buildSlots <- struct{}{}
			defer func() { <-buildSlots }()

			rnd := rand.New(rand.NewSource(int64(idx) + 1))
			leaves := minLeaves
			if maxLeaves > minLeaves {
				leaves += rnd.Intn(maxLeaves - minLeaves)
			}

			ops := make([]trieOp, leaves)
			for j := range ops {
				key := make([]byte, 32)
				rnd.Read(key) //nolint:errcheck // math/rand Read never fails
				value := make([]byte, 900)
				rnd.Read(value) //nolint:errcheck // math/rand Read never fails
				ops[j] = trieOp{key: key, value: value, weight: uint64(rnd.Intn(1000) + 1)}
			}

			// Plain twin: built, root recorded, dropped.
			plain := newPoktrollSpecSMST(simplemap.NewSimpleMap())
			for _, o := range ops {
				if err := plain.Update(o.key, o.value, o.weight); err != nil {
					errs[idx] = err
					return
				}
			}
			if err := plain.Commit(); err != nil {
				errs[idx] = err
				return
			}
			plainRoot := append([]byte(nil), plain.Root()...)
			plain = nil
			_ = plain

			// Compacted trie: stays resident, the way a session trie does.
			trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
			compactedTotal := 0
			for _, o := range ops {
				if err := trie.Update(o.key, o.value, o.weight); err != nil {
					errs[idx] = err
					return
				}
				if err := trie.Commit(); err != nil {
					errs[idx] = err
					return
				}
				n, err := trie.CompactPersistedLeaves()
				if err != nil {
					errs[idx] = err
					return
				}
				compactedTotal += n
			}

			if !bytes.Equal(plainRoot, trie.Root()) {
				errs[idx] = &rootMismatchError{trie: idx, leaves: leaves}
				return
			}

			held, total := residentLeavesHoldingValues(trie.root)
			resident[idx] = trie
			results[idx] = result{leaves: leaves, compacted: compactedTotal, held: held, total: total}
		}(i)
	}
	wg.Wait()

	totalLeaves, totalCompacted, totalHeld := 0, 0, 0
	for i, err := range errs {
		if err != nil {
			t.Fatalf("trie %d: %v", i, err)
		}
		totalLeaves += results[i].leaves
		totalCompacted += results[i].compacted
		totalHeld += results[i].held
	}

	if totalHeld != 0 {
		t.Fatalf("%d leaves across the fleet still hold a value after compaction", totalHeld)
	}
	if totalCompacted == 0 {
		t.Fatal("the fleet compacted nothing; the test proves nothing")
	}

	// Keep every trie alive to the end: the point is that a thousand compacted
	// tries coexist, not that they were built and collected one at a time.
	for _, trie := range resident {
		if trie == nil {
			t.Fatal("a trie in the fleet was not retained")
		}
	}

	t.Logf("fleet: %d tries, %d leaves total, %d leaf compactions, all roots equal to their plain twin",
		numTries, totalLeaves, totalCompacted)
}

type rootMismatchError struct {
	trie   int
	leaves int
}

func (e *rootMismatchError) Error() string {
	return "root mismatch: trie " + strconv.Itoa(e.trie) +
		" with " + strconv.Itoa(e.leaves) + " leaves"
}

// C2 at scale: a proof taken from a deep trie. What the miner pays is the
// number of store reads per proof, which this reports rather than assumes.
func TestCompactScale_C2_ProofsFromDeepTrie(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test; skipped under -short")
	}

	numLeaves := envInt(t, "SMT_SCALE_PROOF_LEAVES", 100_000)
	numProofs := envInt(t, "SMT_SCALE_PROOFS", 200)

	store := newCountingStore()
	trie := newPoktrollSpecSMST(store)

	rnd := rand.New(rand.NewSource(99))
	sampledKeys := make([][]byte, 0, numProofs)
	sampleEvery := numLeaves / numProofs

	for i := 0; i < numLeaves; i++ {
		key := make([]byte, 32)
		rnd.Read(key) //nolint:errcheck // math/rand Read never fails
		value := make([]byte, 900)
		rnd.Read(value) //nolint:errcheck // math/rand Read never fails

		requireNoError(t, trie.Update(key, value, uint64(rnd.Intn(1000)+1)), "Update")
		requireNoError(t, trie.Commit(), "Commit")
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("insert %d: CompactPersistedLeaves: %v", i, err)
		}
		if i%sampleEvery == 0 && len(sampledKeys) < numProofs {
			sampledKeys = append(sampledKeys, key)
		}
	}

	held, total := residentLeavesHoldingValues(trie.root)
	if held != 0 {
		t.Fatalf("%d of %d leaves still hold a value; proofs would not hit the store", held, total)
	}

	spec := trie.Spec()
	root := trie.Root()
	store.reset()
	withSibling := 0

	for _, key := range sampledKeys {
		proof, err := trie.ProveClosest(spec.ph.Path(key))
		requireNoError(t, err, "ProveClosest")
		if proof.ClosestProof.SiblingData != nil {
			withSibling++
		}

		valid, err := VerifyClosestProof(proof, root, spec)
		requireNoError(t, err, "VerifyClosestProof")
		if !valid {
			t.Fatalf("key %x: proof did not verify against a %d-leaf compacted trie", key, numLeaves)
		}

		compactProof, err := CompactClosestProof(proof, spec)
		requireNoError(t, err, "CompactClosestProof")
		decompacted, err := DecompactClosestProof(compactProof, spec)
		requireNoError(t, err, "DecompactClosestProof")
		valid, err = VerifyClosestProof(decompacted, root, spec)
		requireNoError(t, err, "VerifyClosestProof(decompacted)")
		if !valid {
			t.Fatalf("key %x: round-tripped proof did not verify", key)
		}

		// Re-compact so the next proof starts from the compacted state too;
		// otherwise only the first proof measures the store cost.
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("re-compaction: %v", err)
		}
	}

	t.Logf("C2 scale: leaves=%d proofs=%d verified, %d carried SiblingData",
		numLeaves, len(sampledKeys), withSibling)
	t.Logf("C2 scale: store Gets during %d proofs = %d (%.2f per proof)",
		len(sampledKeys), store.gets, float64(store.gets)/float64(len(sampledKeys)))
}
