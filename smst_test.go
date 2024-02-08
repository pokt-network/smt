package smt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func NewSMSTWithStorage(
	nodes, preimages kvstore.MapStore,
	hasher hash.Hash,
	options ...Option,
) *SMSTWithStorage {
	return &SMSTWithStorage{
		SMST:      NewSparseMerkleSumTrie(nodes, hasher, options...),
		preimages: preimages,
	}
}

func TestSMST_TrieUpdateBasic(t *testing.T) {
	smn := simplemap.NewSimpleMap()
	smv := simplemap.NewSimpleMap()
	lazy := NewSparseMerkleSumTrie(smn, sha256.New())
	smst := &SMSTWithStorage{SMST: lazy, preimages: smv}
	var value []byte
	var sum uint64
	var has bool
	var err error

	// Test getting an empty key.
	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, defaultEmptyValue, value)
	require.Equal(t, uint64(0), sum)

	has, err = smst.Has([]byte("testKey"))
	require.NoError(t, err)
	require.False(t, has)

	// Test updating the empty key.
	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue"), value)
	require.Equal(t, uint64(5), sum)

	has, err = smst.Has([]byte("testKey"))
	require.NoError(t, err)
	require.True(t, has)

	// Test updating the non-empty key.
	err = smst.Update([]byte("testKey"), []byte("testValue2"), 10)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue2"), value)
	require.Equal(t, uint64(10), sum)

	// Test updating a second empty key where the path for both keys share the
	// first 2 bits (when using SHA256).
	err = smst.Update([]byte("foo"), []byte("bar"), 5)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, []byte("bar"), value)
	require.Equal(t, uint64(5), sum)

	// Test updating a third empty key.
	err = smst.Update([]byte("testKey2"), []byte("testValue3"), 5)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey2"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue3"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue2"), value)
	require.Equal(t, uint64(10), sum)

	require.NoError(t, lazy.Commit())

	// Test that a trie can be imported from a KVStore.
	lazy = ImportSparseMerkleSumTrie(smn, sha256.New(), smst.Root())
	require.NoError(t, err)
	smst = &SMSTWithStorage{SMST: lazy, preimages: smv}

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue2"), value)
	require.Equal(t, uint64(10), sum)

	value, sum, err = smst.GetValueSum([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, []byte("bar"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.GetValueSum([]byte("testKey2"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue3"), value)
	require.Equal(t, uint64(5), sum)
}

// Test base case trie delete operations with a few keys.
func TestSMST_TrieDeleteBasic(t *testing.T) {
	smn := simplemap.NewSimpleMap()
	smv := simplemap.NewSimpleMap()
	lazy := NewSparseMerkleSumTrie(smn, sha256.New())
	smst := &SMSTWithStorage{SMST: lazy, preimages: smv}
	rootEmpty := smst.Root()

	// Testing inserting, deleting a key, and inserting it again.
	err := smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)

	root1 := smst.Root()
	err = smst.Delete([]byte("testKey"))
	require.NoError(t, err)

	value, sum, err := smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, defaultEmptyValue, value, "getting deleted key")
	require.Equal(t, uint64(0), sum, "getting deleted key")

	has, err := smst.Has([]byte("testKey"))
	require.NoError(t, err)
	require.False(t, has, "checking existence of deleted key")

	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue"), value)
	require.Equal(t, uint64(5), sum)
	require.Equal(t, root1, smst.Root(), "re-inserting key after deletion")

	// Test inserting and deleting a second key.
	err = smst.Update([]byte("testKey2"), []byte("testValue2"), 10)
	require.NoError(t, err)

	err = smst.Delete([]byte("testKey2"))
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey2"))
	require.NoError(t, err)
	require.Equal(t, defaultEmptyValue, value, "getting deleted key")
	require.Equal(t, uint64(0), sum, "getting deleted key")

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue"), value)
	require.Equal(t, uint64(5), sum)
	require.Equal(t, root1, smst.Root(), "after deleting second key")

	// Test inserting and deleting a different second key, when the the first 2
	// bits of the path for the two keys in the trie are the same (when using SHA256).
	err = smst.Update([]byte("foo"), []byte("bar"), 5)
	require.NoError(t, err)

	_, _, err = smst.GetValueSum([]byte("foo"))
	require.NoError(t, err)

	err = smst.Delete([]byte("foo"))
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, defaultEmptyValue, value, "getting deleted key")
	require.Equal(t, uint64(0), sum, "getting deleted key")

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue"), value)
	require.Equal(t, uint64(5), sum)
	require.Equal(t, root1, smst.Root(), "after deleting second key")

	// Testing inserting, deleting a key, and inserting it again
	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)

	root1 = smst.Root()
	err = smst.Delete([]byte("testKey"))
	require.NoError(t, err)

	// Fail to delete an absent key, but leave trie in a valid state
	err = smst.Delete([]byte("testKey"))
	require.Error(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, defaultEmptyValue, value, "getting deleted key")
	require.Equal(t, uint64(0), sum, "getting deleted key")

	has, err = smst.Has([]byte("testKey"))
	require.NoError(t, err)
	require.False(t, has, "checking existence of deleted key")
	require.Equal(t, rootEmpty, smst.Root())

	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum([]byte("testKey"))
	require.NoError(t, err)
	require.Equal(t, []byte("testValue"), value)
	require.Equal(t, uint64(5), sum)
	require.Equal(t, root1, smst.Root(), "re-inserting key after deletion")
}

// Test trie ops with known paths
func TestSMST_TrieKnownPath(t *testing.T) {
	ph := dummyPathHasher{32}
	smn := simplemap.NewSimpleMap()
	smv := simplemap.NewSimpleMap()
	smst := NewSMSTWithStorage(smn, smv, sha256.New(), WithPathHasher(ph))
	var value []byte
	var sum uint64

	baseKey := make([]byte, ph.PathSize())
	keys := make([][]byte, 7)
	for i := range keys {
		keys[i] = make([]byte, ph.PathSize())
		copy(keys[i], baseKey)
	}
	keys[0][0] = byte(0b00000000)
	keys[1][0] = byte(0b00100000)
	keys[2][0] = byte(0b10000000)
	keys[3][0] = byte(0b11000000)
	keys[4][0] = byte(0b11010000)
	keys[5][0] = byte(0b11100000)
	keys[6][0] = byte(0b11110000)

	err := smst.Update(keys[0], []byte("testValue1"), 1)
	require.NoError(t, err)
	err = smst.Update(keys[1], []byte("testValue2"), 2)
	require.NoError(t, err)
	err = smst.Update(keys[2], []byte("testValue3"), 3)
	require.NoError(t, err)
	err = smst.Update(keys[3], []byte("testValue4"), 4)
	require.NoError(t, err)
	err = smst.Update(keys[4], []byte("testValue5"), 5)
	require.NoError(t, err)
	err = smst.Update(keys[5], []byte("testValue6"), 6)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum(keys[0])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue1"), value)
	require.Equal(t, uint64(1), sum)

	value, sum, err = smst.GetValueSum(keys[1])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue2"), value)
	require.Equal(t, uint64(2), sum)

	value, sum, err = smst.GetValueSum(keys[2])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue3"), value)
	require.Equal(t, uint64(3), sum)

	value, sum, err = smst.GetValueSum(keys[3])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue4"), value)
	require.Equal(t, uint64(4), sum)

	err = smst.Delete(keys[3])
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum(keys[4])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue5"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.GetValueSum(keys[5])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue6"), value)
	require.Equal(t, uint64(6), sum)

	// Fail to delete an absent key with a leaf where it would be
	err = smst.Delete(keys[6])
	require.Error(t, err)
	// Key at would-be position is still accessible
	value, sum, err = smst.GetValueSum(keys[5])
	require.NoError(t, err)
	require.Equal(t, []byte("testValue6"), value)
	require.Equal(t, uint64(6), sum)
}

// Test trie operations when two leafs are immediate neighbors.
func TestSMST_TrieMaxHeightCase(t *testing.T) {
	ph := dummyPathHasher{32}
	smn := simplemap.NewSimpleMap()
	smv := simplemap.NewSimpleMap()
	smst := NewSMSTWithStorage(smn, smv, sha256.New(), WithPathHasher(ph))
	var value []byte
	var sum uint64

	// Make two neighboring keys.
	// The dummy hash function will return the preimage itself as the digest.
	key1 := make([]byte, ph.PathSize())
	key2 := make([]byte, ph.PathSize())
	_, err := rand.Read(key1)
	require.NoError(t, err)
	copy(key2, key1)
	// We make key2's least significant bit different than key1's
	key1[ph.PathSize()-1] = byte(0)
	key2[ph.PathSize()-1] = byte(1)

	err = smst.Update(key1, []byte("testValue1"), 1)
	require.NoError(t, err)

	err = smst.Update(key2, []byte("testValue2"), 2)
	require.NoError(t, err)

	value, sum, err = smst.GetValueSum(key1)
	require.NoError(t, err)
	require.Equal(t, []byte("testValue1"), value)
	require.Equal(t, uint64(1), sum)

	value, sum, err = smst.GetValueSum(key2)
	require.NoError(t, err)
	require.Equal(t, []byte("testValue2"), value)
	require.Equal(t, uint64(2), sum)

	proof, err := smst.Prove(key1)
	require.NoError(t, err)
	require.Equal(t, 256, len(proof.SideNodes), "unexpected proof size")
}

func TestSMST_OrphanRemoval(t *testing.T) {
	var smn, smv kvstore.MapStore
	var impl *SMST
	var smst *SMSTWithStorage
	var err error

	nodeCount := func(t *testing.T) int {
		require.NoError(t, impl.Commit())
		return smn.Len()
	}
	setup := func() {
		smn = simplemap.NewSimpleMap()
		smv = simplemap.NewSimpleMap()
		impl = NewSparseMerkleSumTrie(smn, sha256.New())
		smst = &SMSTWithStorage{SMST: impl, preimages: smv}

		err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
		require.NoError(t, err)
		require.Equal(t, 1, nodeCount(t)) // only root node
	}

	t.Run("delete 1", func(t *testing.T) {
		setup()
		err = smst.Delete([]byte("testKey"))
		require.NoError(t, err)
		require.Equal(t, 0, nodeCount(t))
	})

	t.Run("overwrite 1", func(t *testing.T) {
		setup()
		err = smst.Update([]byte("testKey"), []byte("testValue2"), 10)
		require.NoError(t, err)
		require.Equal(t, 1, nodeCount(t))
	})

	type testCase struct {
		keys  []string
		count int
	}
	// sha256(testKey)  = 0001...
	// sha256(testKey2) = 1000... common prefix len 0; 3 nodes (root + 2 leaf)
	// sha256(foo)      = 0010... common prefix len 2; 5 nodes (3 inner + 2 leaf)
	cases := []testCase{
		{[]string{"testKey2"}, 3},
		{[]string{"foo"}, 4},
		{[]string{"testKey2", "foo"}, 6},
		{[]string{"a", "b", "c", "d", "e"}, 14},
	}

	t.Run("overwrite and delete", func(t *testing.T) {
		setup()
		err = smst.Update([]byte("testKey"), []byte("testValue2"), 2)
		require.NoError(t, err)
		require.Equal(t, 1, nodeCount(t))

		err = smst.Delete([]byte("testKey"))
		require.NoError(t, err)
		require.Equal(t, 0, nodeCount(t))

		for tci, tc := range cases {
			setup()
			for _, key := range tc.keys {
				err = smst.Update([]byte(key), []byte("testValue2"), 10)
				require.NoError(t, err, tci)
			}
			require.Equal(t, tc.count, nodeCount(t), tci)

			// Overwrite doesn't change count
			for _, key := range tc.keys {
				err = smst.Update([]byte(key), []byte("testValue3"), 10)
				require.NoError(t, err, tci)
			}
			require.Equal(t, tc.count, nodeCount(t), tci)

			// Deletion removes all nodes except root
			for _, key := range tc.keys {
				err = smst.Delete([]byte(key))
				require.NoError(t, err, tci)
			}
			require.Equal(t, 1, nodeCount(t), tci)

			// Deleting and re-inserting a persisted node doesn't change count
			require.NoError(t, smst.Delete([]byte("testKey")))
			require.NoError(t, smst.Update([]byte("testKey"), []byte("testValue"), 10))
			require.Equal(t, 1, nodeCount(t), tci)
		}
	})
}

func TestSMST_TotalSum(t *testing.T) {
	snm := simplemap.NewSimpleMap()
	smst := NewSparseMerkleSumTrie(snm, sha256.New())
	err := smst.Update([]byte("key1"), []byte("value1"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key2"), []byte("value2"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key3"), []byte("value3"), 5)
	require.NoError(t, err)

	// Check root hash contains the correct hex sum
	root1 := smst.Root()
	sumBz := root1[len(root1)-sumSizeBits:]
	rootSum := binary.BigEndian.Uint64(sumBz)
	require.NoError(t, err)

	// Calculate total sum of the trie
	sum := smst.Sum()
	require.Equal(t, sum, uint64(15))
	require.Equal(t, sum, rootSum)

	// Prove inclusion
	proof, err := smst.Prove([]byte("key1"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, smst.Spec())
	valid, err := VerifySumProof(proof, root1, []byte("key1"), []byte("value1"), 5, smst.Spec())
	require.NoError(t, err)
	require.True(t, valid)

	// Check that the sum is correct after deleting a key
	err = smst.Delete([]byte("key1"))
	require.NoError(t, err)
	sum = smst.Sum()
	require.Equal(t, sum, uint64(10))

	// Check that the sum is correct after importing the trie
	require.NoError(t, smst.Commit())
	root2 := smst.Root()
	smst = ImportSparseMerkleSumTrie(snm, sha256.New(), root2)
	sum = smst.Sum()
	require.Equal(t, sum, uint64(10))

	// Calculate the total sum of a larger trie
	snm = simplemap.NewSimpleMap()
	smst = NewSparseMerkleSumTrie(snm, sha256.New())
	for i := 1; i < 10000; i++ {
		err := smst.Update([]byte(fmt.Sprintf("testKey%d", i)), []byte(fmt.Sprintf("testValue%d", i)), uint64(i))
		require.NoError(t, err)
	}
	require.NoError(t, smst.Commit())
	sum = smst.Sum()
	require.Equal(t, sum, uint64(49995000))
}

func TestSMST_Retrieval(t *testing.T) {
	snm := simplemap.NewSimpleMap()
	smst := NewSparseMerkleSumTrie(snm, sha256.New(), WithValueHasher(nil))

	err := smst.Update([]byte("key1"), []byte("value1"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key2"), []byte("value2"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key3"), []byte("value3"), 5)
	require.NoError(t, err)

	value, sum, err := smst.Get([]byte("key1"))
	require.NoError(t, err)
	require.Equal(t, []byte("value1"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.Get([]byte("key2"))
	require.NoError(t, err)
	require.Equal(t, []byte("value2"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.Get([]byte("key3"))
	require.NoError(t, err)
	require.Equal(t, []byte("value3"), value)
	require.Equal(t, uint64(5), sum)

	require.NoError(t, smst.Commit())

	value, sum, err = smst.Get([]byte("key1"))
	require.NoError(t, err)
	require.Equal(t, []byte("value1"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.Get([]byte("key2"))
	require.NoError(t, err)
	require.Equal(t, []byte("value2"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = smst.Get([]byte("key3"))
	require.NoError(t, err)
	require.Equal(t, []byte("value3"), value)
	require.Equal(t, uint64(5), sum)

	root := smst.Root()
	sum = smst.Sum()
	require.Equal(t, sum, uint64(15))

	lazy := ImportSparseMerkleSumTrie(snm, sha256.New(), root, WithValueHasher(nil))

	value, sum, err = lazy.Get([]byte("key1"))
	require.NoError(t, err)
	require.Equal(t, []byte("value1"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = lazy.Get([]byte("key2"))
	require.NoError(t, err)
	require.Equal(t, []byte("value2"), value)
	require.Equal(t, uint64(5), sum)

	value, sum, err = lazy.Get([]byte("key3"))
	require.NoError(t, err)
	require.Equal(t, []byte("value3"), value)
	require.Equal(t, uint64(5), sum)

	sum = lazy.Sum()
	require.Equal(t, sum, uint64(15))
}
