package smt

import (
	"crypto/sha256"
	"math"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// Commit encodes a big leaf once. It used to encode it twice (once for the
// preimage it stores, once more inside digest to hash it), and each encoding
// built the node in a pooled buffer and copied it out: four allocations the
// size of the leaf per Commit. A relay miner puts megabyte relays in leaves, and
// that garbage outran its garbage collector.
//
// TotalAlloc counts every goroutine, and only ever grows, so whatever else runs
// can only add to a sample. The test commits a fresh trie several times and
// asserts on the smallest sample, which is the closest to Commit's own bytes.
// It counts bytes, not allocations: the cost here is the size of the garbage.
// The stored node is the leaf itself because simplemap keeps the slice it is
// given without copying it; a store that copied would add a leaf's worth.
func TestCommitEncodesABigLeafOnce(t *testing.T) {
	const (
		leafBytes = 1 << 20
		samples   = 5
	)
	key := sha256.Sum256([]byte("big-leaf"))

	allocated := uint64(math.MaxUint64)
	for range samples {
		smst := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
		// A value per trie: with a nil value hasher Update writes the weight
		// into the value's spare capacity, so tries must not share one.
		require.NoError(t, smst.Update(key[:], chainedBytes("big-leaf", leafBytes), 1))

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		require.NoError(t, smst.Commit())
		runtime.ReadMemStats(&after)
		allocated = min(allocated, after.TotalAlloc-before.TotalAlloc)
	}

	require.Less(t, allocated, uint64(leafBytes*3/2),
		"Commit allocated %d bytes for a %d B leaf: the node must be encoded once, into one allocation", allocated, leafBytes)
	require.Greater(t, allocated, uint64(leafBytes),
		"CONTROL: Commit must allocate the stored node itself; less means the measurement missed it")
}
