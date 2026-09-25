package smt

import (
	"crypto/sha256"
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
func TestCommitEncodesABigLeafOnce(t *testing.T) {
	const leafBytes = 1 << 20
	smst := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
	key := sha256.Sum256([]byte("big-leaf"))
	require.NoError(t, smst.Update(key[:], chainedBytes("big-leaf", leafBytes), 1))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	require.NoError(t, smst.Commit())
	runtime.ReadMemStats(&after)

	allocated := after.TotalAlloc - before.TotalAlloc
	require.Less(t, allocated, uint64(leafBytes*3/2),
		"Commit allocated %d bytes for a %d B leaf: the node must be encoded once, into one allocation", allocated, leafBytes)
	require.Greater(t, allocated, uint64(leafBytes),
		"CONTROL: Commit must allocate the stored node itself; less means the measurement missed it")
}
