package smt

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// C1: compacting persisted leaves must not change a single byte of the root,
// at EVERY step, for any insertion order, including repeated keys (which
// produce orphans) and the weights that feed the sum and count.
//
// Two tries are driven with the identical operation sequence over identical
// stores. Trie A compacts after each Commit; trie B never compacts. Their
// roots are compared after every operation.
func TestCompact_C1_RootIsIdentical(t *testing.T) {
	for _, numLeaves := range []int{1, 2, 10, 1000, 10000} {
		if numLeaves >= 1000 && testing.Short() {
			continue
		}
		for _, seed := range []int64{1, 42, 1337} {
			name := fmt.Sprintf("n=%d/seed=%d", numLeaves, seed)
			t.Run(name, func(t *testing.T) {
				ops := generateOps(seed, numLeaves)

				trieA := newPoktrollSpecSMST(simplemap.NewSimpleMap())
				trieB := newPoktrollSpecSMST(simplemap.NewSimpleMap())

				for i, o := range ops {
					requireNoError(t, trieA.Update(o.key, o.value, o.weight), "A.Update")
					requireNoError(t, trieB.Update(o.key, o.value, o.weight), "B.Update")

					requireNoError(t, trieA.Commit(), "A.Commit")
					requireNoError(t, trieB.Commit(), "B.Commit")

					// Only trie A compacts.
					if _, err := trieA.CompactPersistedLeaves(); err != nil {
						t.Fatalf("op %d: CompactPersistedLeaves: %v", i, err)
					}

					if !bytes.Equal(trieA.Root(), trieB.Root()) {
						t.Fatalf("op %d: root diverged after compaction\n compacted: %x\n plain:     %x",
							i, trieA.Root(), trieB.Root())
					}
					if trieA.MustSum() != trieB.MustSum() {
						t.Fatalf("op %d: sum diverged: compacted=%d plain=%d",
							i, trieA.MustSum(), trieB.MustSum())
					}
					if trieA.MustCount() != trieB.MustCount() {
						t.Fatalf("op %d: count diverged: compacted=%d plain=%d",
							i, trieA.MustCount(), trieB.MustCount())
					}
				}
			})
		}
	}
}

// C1 (cont.): compacting must also be invisible to the bytes held by the
// store. If compaction ever caused a leaf to be re-encoded and re-persisted
// differently, or left an orphan behind, the two stores would diverge.
func TestCompact_C1_StoreContentsIdentical(t *testing.T) {
	ops := generateOps(7, 200)

	mapA, mapB := map[string][]byte{}, map[string][]byte{}
	trieA := newPoktrollSpecSMST(simplemap.NewSimpleMapWithMap(mapA))
	trieB := newPoktrollSpecSMST(simplemap.NewSimpleMapWithMap(mapB))

	for _, o := range ops {
		requireNoError(t, trieA.Update(o.key, o.value, o.weight), "A.Update")
		requireNoError(t, trieB.Update(o.key, o.value, o.weight), "B.Update")
		requireNoError(t, trieA.Commit(), "A.Commit")
		requireNoError(t, trieB.Commit(), "B.Commit")
		if _, err := trieA.CompactPersistedLeaves(); err != nil {
			t.Fatalf("CompactPersistedLeaves: %v", err)
		}
	}

	if len(mapA) != len(mapB) {
		t.Fatalf("store size diverged: compacted=%d plain=%d", len(mapA), len(mapB))
	}
	for k, vB := range mapB {
		vA, ok := mapA[k]
		if !ok {
			t.Fatalf("key %x present in plain store, missing from compacted store", k)
		}
		if !bytes.Equal(vA, vB) {
			t.Fatalf("key %x: stored bytes diverged\n compacted: %x\n plain:     %x", k, vA, vB)
		}
	}
}

// Control for C1: the two tests above would pass just as happily if
// CompactPersistedLeaves were a no-op. This one asserts it actually drops
// values, and that after a pass no resident persisted leaf still holds one.
func TestCompact_C1_ActuallyCompacts(t *testing.T) {
	const numLeaves = 500

	trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	distinct := map[string]struct{}{}
	for _, o := range generateOps(11, numLeaves) {
		requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
		distinct[string(o.key)] = struct{}{}
	}
	requireNoError(t, trie.Commit(), "Commit")

	compacted, err := trie.CompactPersistedLeaves()
	requireNoError(t, err, "CompactPersistedLeaves")
	if compacted != len(distinct) {
		t.Fatalf("compacted %d leaves, want %d (one per distinct key)", compacted, len(distinct))
	}

	held, total := residentLeavesHoldingValues(trie.root)
	if total != len(distinct) {
		t.Fatalf("walked %d resident leaves, want %d", total, len(distinct))
	}
	if held != 0 {
		t.Fatalf("%d of %d resident leaves still hold their value after compaction", held, total)
	}

	// A second pass has nothing left to do.
	again, err := trie.CompactPersistedLeaves()
	requireNoError(t, err, "second CompactPersistedLeaves")
	if again != 0 {
		t.Fatalf("second compaction pass compacted %d leaves, want 0", again)
	}
}

// C1 (cont.): a leaf that has not been committed yet must survive compaction
// untouched. Its value is in neither the store nor anywhere else, so dropping
// it would leave a leaf whose digest gets computed over a truncated preimage —
// and commit would then write that corruption to the store.
//
// This is the only test that reaches the "not persisted" guard in compactNode:
// everywhere else compaction runs immediately after Commit, when every leaf is
// already persisted.
func TestCompact_C1_UncommittedLeafIsNotCompacted(t *testing.T) {
	trieA := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	trieB := newPoktrollSpecSMST(simplemap.NewSimpleMap())

	ops := generateOps(97, 100)
	committed := ops[:60]
	pending := ops[60:]

	for _, o := range committed {
		requireNoError(t, trieA.Update(o.key, o.value, o.weight), "A.Update")
		requireNoError(t, trieB.Update(o.key, o.value, o.weight), "B.Update")
	}
	requireNoError(t, trieA.Commit(), "A.Commit")
	requireNoError(t, trieB.Commit(), "B.Commit")

	// Leave these uncommitted, then compact anyway.
	for _, o := range pending {
		requireNoError(t, trieA.Update(o.key, o.value, o.weight), "A.Update pending")
		requireNoError(t, trieB.Update(o.key, o.value, o.weight), "B.Update pending")
	}

	compacted, err := trieA.CompactPersistedLeaves()
	requireNoError(t, err, "CompactPersistedLeaves")

	// Control: the trie must really hold uncommitted leaves at this point,
	// otherwise the guard is not being reached and this test proves nothing.
	_, total := residentLeavesHoldingValues(trieA.root)
	uncommitted := total - compacted
	if uncommitted <= 0 {
		t.Fatalf("every resident leaf was compacted (total=%d compacted=%d): either the "+
			"not-persisted guard did not hold and a leaf whose value is in neither memory "+
			"nor the store was compacted, or this test failed to leave any leaf uncommitted",
			total, compacted)
	}
	t.Logf("%d leaves compacted, %d left alone because they are not persisted",
		compacted, uncommitted)

	if !bytes.Equal(trieA.Root(), trieB.Root()) {
		t.Fatalf("root diverged after compacting a trie with uncommitted leaves\n compacted: %x\n plain:     %x",
			trieA.Root(), trieB.Root())
	}

	// Committing afterwards must still produce the same root.
	requireNoError(t, trieA.Commit(), "A.Commit after compaction")
	requireNoError(t, trieB.Commit(), "B.Commit after compaction")
	if !bytes.Equal(trieA.Root(), trieB.Root()) {
		t.Fatalf("root diverged after committing the previously-uncommitted leaves\n compacted: %x\n plain:     %x",
			trieA.Root(), trieB.Root())
	}
}

// residentLeavesHoldingValues walks the resident trie and reports how many
// leaves still hold a value, and how many leaves were seen in total.
func residentLeavesHoldingValues(node trieNode) (held, total int) {
	switch n := node.(type) {
	case *leafNode:
		total = 1
		if n.valueHash != nil {
			held = 1
		}
	case *innerNode:
		lh, lt := residentLeavesHoldingValues(n.leftChild)
		rh, rt := residentLeavesHoldingValues(n.rightChild)
		held, total = lh+rh, lt+rt
	case *extensionNode:
		held, total = residentLeavesHoldingValues(n.child)
	}
	return held, total
}

// trieOp is a single trie mutation used by the property tests.
type trieOp struct {
	key    []byte
	value  []byte
	weight uint64
}

// generateOps builds a deterministic operation sequence for the given seed.
// Roughly one in five operations reuses an earlier key, which is what makes
// the trie orphan a persisted (and, in trie A, compacted) leaf.
func generateOps(seed int64, numLeaves int) []trieOp {
	rnd := rand.New(rand.NewSource(seed))
	ops := make([]trieOp, 0, numLeaves)
	keys := make([][]byte, 0, numLeaves)

	for i := 0; i < numLeaves; i++ {
		var key []byte
		if len(keys) > 0 && rnd.Intn(5) == 0 {
			key = keys[rnd.Intn(len(keys))]
		} else {
			key = make([]byte, 32)
			rnd.Read(key) //nolint:errcheck // math/rand Read never returns an error
			keys = append(keys, key)
		}

		// ~900 B stands in for a serialised relay, the production shape.
		value := make([]byte, 850+rnd.Intn(100))
		rnd.Read(value) //nolint:errcheck // math/rand Read never returns an error

		ops = append(ops, trieOp{key: key, value: value, weight: uint64(rnd.Intn(1000) + 1)})
	}
	return ops
}
