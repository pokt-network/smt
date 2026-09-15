package smt

import (
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// C4: how much resident memory compaction actually reclaims.
//
// What is measured is the memory retained by the TRIE, not by the trie plus
// its node store. A node store is often not in the trie's process at all -- it
// can be an external database -- so charging its bytes to both variants would
// dilute the ratio with a constant that need not exist where the trie runs. The
// store figure is reported separately so the dilution can be seen rather than
// assumed.
//
// The isolation works by difference: build trie + store, GC, read; drop the
// trie while keeping the store alive, GC, read again. What disappeared is the
// trie.
//
// Leaf count comes from SMT_C4_LEAVES so the same test can be walked up a
// staircase of sizes; the structural floor (inner and extension nodes) grows
// with N while the per-leaf value does not, so a single N cannot tell the two
// apart.
func TestCompact_C4_ResidentMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("C4 builds millions of leaves; skipped under -short")
	}

	numLeaves := 200_000
	if raw := os.Getenv("SMT_C4_LEAVES"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("SMT_C4_LEAVES=%q: %v", raw, err)
		}
		numLeaves = parsed
	}
	const valueSize = 900

	plain := measureTrieHeap(t, numLeaves, valueSize, false)
	compacted := measureTrieHeap(t, numLeaves, valueSize, true)

	perLeafPlain := float64(plain.trieBytes) / float64(numLeaves) / 1024
	perLeafCompacted := float64(compacted.trieBytes) / float64(numLeaves) / 1024
	ratio := float64(compacted.trieBytes) / float64(plain.trieBytes) * 100

	t.Logf("C4 leaves=%d value=%dB", numLeaves, valueSize)
	t.Logf("C4 plain     trie=%d B  %.3f KB/leaf   store=%d B  %.3f KB/leaf",
		plain.trieBytes, perLeafPlain, plain.storeBytes,
		float64(plain.storeBytes)/float64(numLeaves)/1024)
	t.Logf("C4 compacted trie=%d B  %.3f KB/leaf   store=%d B  %.3f KB/leaf",
		compacted.trieBytes, perLeafCompacted, compacted.storeBytes,
		float64(compacted.storeBytes)/float64(numLeaves)/1024)
	t.Logf("C4 RATIO compacted/plain = %.1f%% (target <= 30%%)", ratio)
	t.Logf("C4 HeapInuse deltas: plain=%d B (%.3f KB/leaf)  compacted=%d B (%.3f KB/leaf)",
		plain.trieInuse, float64(plain.trieInuse)/float64(numLeaves)/1024,
		compacted.trieInuse, float64(compacted.trieInuse)/float64(numLeaves)/1024)

	// Control: the two variants must have built the same trie. If they did not,
	// the ratio compares two different things.
	if plain.root != compacted.root {
		t.Fatalf("the two variants produced different roots; the comparison is meaningless")
	}
	// Control: compaction moves nothing into the store; it only stops keeping
	// a second copy. A different node count would mean it did something else.
	if plain.storeLen != compacted.storeLen {
		t.Fatalf("store holds %d nodes plain vs %d compacted; compaction changed what is persisted",
			plain.storeLen, compacted.storeLen)
	}

	if ratio > 30 {
		t.Fatalf("C4 FAIL: compacted trie is %.1f%% of plain, above the 30%% target", ratio)
	}
}

type heapMeasurement struct {
	// trieBytes is HeapAlloc retained by the trie alone: live bytes with the
	// trie held, minus live bytes once it is dropped and only the store remains.
	trieBytes uint64
	// trieInuse is the same difference measured with HeapInuse, which the
	// brief asked for. It includes span fragmentation, so it is the noisier of
	// the two and is reported rather than asserted on.
	trieInuse uint64
	// storeBytes is what the node store retains; an out-of-process store would
	// not hold these bytes in this heap.
	storeBytes uint64
	storeLen   int
	root       string
}

// measureTrieHeap builds a trie of numLeaves values of valueSize bytes,
// committing after every update, which is the pattern compaction is meant for,
// and reports how much heap the trie itself retains.
func measureTrieHeap(t *testing.T, numLeaves, valueSize int, compact bool) heapMeasurement {
	t.Helper()

	var before, withTrie, withoutTrie runtime.MemStats

	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&before)

	store := simplemap.NewSimpleMap()
	trie := buildMeasuredTrie(t, store, numLeaves, valueSize, compact)
	root := string(trie.Root())

	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&withTrie)
	// Without this the compiler is free to treat the trie as dead before the
	// read above, which would collapse the two measurements into one.
	runtime.KeepAlive(trie)

	trie = nil
	_ = trie

	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&withoutTrie)
	storeLen, err := store.Len()
	requireNoError(t, err, "store.Len")
	runtime.KeepAlive(store)

	return heapMeasurement{
		trieBytes:  sub(withTrie.HeapAlloc, withoutTrie.HeapAlloc),
		trieInuse:  sub(withTrie.HeapInuse, withoutTrie.HeapInuse),
		storeBytes: sub(withoutTrie.HeapAlloc, before.HeapAlloc),
		storeLen:   storeLen,
		root:       root,
	}
}

// buildMeasuredTrie inserts numLeaves deterministic key/value pairs, calling
// Commit after each Update and optionally compacting after each Commit.
func buildMeasuredTrie(
	t *testing.T,
	store kvstore.MapStore,
	numLeaves, valueSize int,
	compact bool,
) *SMST {
	t.Helper()

	trie := newPoktrollSpecSMST(store)
	rnd := rand.New(rand.NewSource(20260914))

	key := make([]byte, 32)
	for i := 0; i < numLeaves; i++ {
		rnd.Read(key) //nolint:errcheck // math/rand Read never returns an error
		// The value must be a fresh allocation: the leaf retains it.
		value := make([]byte, valueSize)
		rnd.Read(value) //nolint:errcheck // math/rand Read never returns an error

		requireNoError(t, trie.Update(key, value, uint64(i%1000)+1), "Update")
		requireNoError(t, trie.Commit(), "Commit")
		if compact {
			if _, err := trie.CompactPersistedLeaves(); err != nil {
				t.Fatalf("CompactPersistedLeaves: %v", err)
			}
		}
	}

	// Control: assert each variant is in the state it claims to be in, so a
	// silently-not-compacting build cannot be reported as a memory win.
	held, total := residentLeavesHoldingValues(trie.root)
	if total == 0 {
		t.Fatal("built trie has no resident leaves")
	}
	if compact && held != 0 {
		t.Fatalf("compacted variant: %d of %d leaves still hold a value", held, total)
	}
	if !compact && held != total {
		t.Fatalf("plain variant: only %d of %d leaves hold a value", held, total)
	}

	return trie
}

// sub subtracts b from a, clamping at zero: heap readings are not monotonic
// and an underflow on uint64 would report a nonsensical number.
func sub(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}
