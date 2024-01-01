package smt

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test base case Merkle proof operations.
func TestSMST_Proof_Operations(t *testing.T) {
	var smn, smv KVStore
	var smst *SMSTWithStorage
	var proof *SparseMerkleProof
	var result bool
	var root []byte
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smv, err = NewKVStore("")
	require.NoError(t, err)
	smst = NewSMSTWithStorage(smn, smv, sha256.New())
	base := smst.Spec()

	// Generate and verify a proof on an empty key.
	proof, err = smst.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = VerifySumProof(proof, placeholder(base), []byte("testKey3"), defaultValue, 0, base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey3"), []byte("badValue"), 5, base)
	require.NoError(t, err)
	require.False(t, result)

	// Add a key, generate and verify a Merkle proof.
	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)
	root = smst.Root()
	proof, err = smst.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 5, base) // valid
	require.NoError(t, err)
	require.True(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 5, base) // wrong value
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 10, base) // wrong sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 10, base) // wrong value and sum
	require.NoError(t, err)
	require.False(t, result)

	// Add a key, generate and verify both Merkle proofs.
	err = smst.Update([]byte("testKey2"), []byte("testValue"), 5)
	require.NoError(t, err)
	root = smst.Root()
	proof, err = smst.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 5, base) // valid
	require.NoError(t, err)
	require.True(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 5, base) // wrong value
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 10, base) // wrong sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 10, base) // wrong value and sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey"), []byte("testValue"), 5, base) // invalid proof
	require.NoError(t, err)
	require.False(t, result)

	proof, err = smst.Prove([]byte("testKey2"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = VerifySumProof(proof, root, []byte("testKey2"), []byte("testValue"), 5, base) // valid
	require.NoError(t, err)
	require.True(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey2"), []byte("badValue"), 5, base) // wrong value
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey2"), []byte("testValue"), 10, base) // wrong sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey2"), []byte("badValue"), 10, base) // wrong value and sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey2"), []byte("testValue"), 5, base) // invalid proof
	require.NoError(t, err)
	require.False(t, result)

	// Try proving a default value for a non-default leaf.
	var sum [sumSize]byte
	binary.LittleEndian.PutUint64(sum[:], 5)
	tval := base.digestValue([]byte("testValue"))
	tval = append(tval, sum[:]...)
	_, leafData := base.th.digestSumLeaf(base.ph.Path([]byte("testKey2")), tval)
	proof = &SparseMerkleProof{
		SideNodes:             proof.SideNodes,
		NonMembershipLeafData: leafData,
	}
	result, err = VerifySumProof(proof, root, []byte("testKey2"), defaultValue, 0, base)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)

	// Generate and verify a proof on an empty key.
	proof, err = smst.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = VerifySumProof(proof, root, []byte("testKey3"), defaultValue, 0, base) // valid
	require.NoError(t, err)
	require.True(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey3"), []byte("badValue"), 0, base) // wrong value
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(proof, root, []byte("testKey3"), defaultValue, 5, base) // wrong sum
	require.NoError(t, err)
	require.False(t, result)
	result, err = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey3"), defaultValue, 0, base) // invalid proof
	require.NoError(t, err)
	require.False(t, result)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

// Test sanity check cases for non-compact proofs.
func TestSMST_Proof_ValidateBasic(t *testing.T) {
	smn, err := NewKVStore("")
	require.NoError(t, err)
	smv, err := NewKVStore("")
	require.NoError(t, err)
	smst := NewSMSTWithStorage(smn, smv, sha256.New())
	base := smst.Spec()

	err = smst.Update([]byte("testKey1"), []byte("testValue1"), 1)
	require.NoError(t, err)
	err = smst.Update([]byte("testKey2"), []byte("testValue2"), 2)
	require.NoError(t, err)
	err = smst.Update([]byte("testKey3"), []byte("testValue3"), 3)
	require.NoError(t, err)
	err = smst.Update([]byte("testKey4"), []byte("testValue4"), 4)
	require.NoError(t, err)
	root := smst.Root()

	// Case: invalid number of sidenodes.
	proof, _ := smst.Prove([]byte("testKey1"))
	sideNodes := make([][]byte, smst.Spec().depth()+1)
	for i := range sideNodes {
		sideNodes[i] = proof.SideNodes[0]
	}
	proof.SideNodes = sideNodes
	require.EqualError(t, proof.validateBasic(base), "too many side nodes: got 257 but max is 256")
	result, err := VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect size for NonMembershipLeafData.
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.NonMembershipLeafData = make([]byte, 1)
	require.EqualError(t, proof.validateBasic(base), "invalid non-membership leaf data size: got 1 but min is 33")
	result, err = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: unexpected sidenode size.
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.SideNodes[0] = make([]byte, 1)
	require.EqualError(t, proof.validateBasic(base), "invalid side node size: got 1 but want 40")
	result, err = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect non-nil sibling data
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.SiblingData = base.th.digest(proof.SiblingData)
	require.EqualError(
		t,
		proof.validateBasic(base),
		"invalid sibling data hash: got 437437455c0f5ca33597b9dd2a307bdfcc6833d3c272e101f30ed6358783fc247f0b9966865746c1 but want 1dc9a3da748c53b22c9e54dcafe9e872341babda9b3e50577f0b9966865746c10000000000000009",
	)

	result, err = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

func TestSMST_ClosestProof_ValidateBasic(t *testing.T) {
	smn, err := NewKVStore("")
	require.NoError(t, err)
	smst := NewSparseMerkleSumTrie(smn, sha256.New())
	np := NoPrehashSpec(sha256.New(), true)
	base := smst.Spec()
	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)

	// insert some unrelated values to populate the trie
	require.NoError(t, smst.Update([]byte("foo"), []byte("oof"), 3))
	require.NoError(t, smst.Update([]byte("bar"), []byte("rab"), 6))
	require.NoError(t, smst.Update([]byte("baz"), []byte("zab"), 9))
	require.NoError(t, smst.Update([]byte("bin"), []byte("nib"), 12))
	require.NoError(t, smst.Update([]byte("fiz"), []byte("zif"), 15))
	require.NoError(t, smst.Update([]byte("fob"), []byte("bof"), 18))
	require.NoError(t, smst.Update([]byte("testKey"), []byte("testValue"), 21))
	require.NoError(t, smst.Update([]byte("testKey2"), []byte("testValue2"), 24))
	require.NoError(t, smst.Update([]byte("testKey3"), []byte("testValue3"), 27))
	require.NoError(t, smst.Update([]byte("testKey4"), []byte("testValue4"), 30))
	root := smst.Root()

	proof, err := smst.ProveClosest(path[:])
	require.NoError(t, err)
	proof.Depth = -1
	require.EqualError(t, proof.validateBasic(base), "invalid depth: got -1, outside of [0, 256]")
	result, err := VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactClosestProof(proof, base)
	require.Error(t, err)
	proof.Depth = 257
	require.EqualError(t, proof.validateBasic(base), "invalid depth: got 257, outside of [0, 256]")
	result, err = VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactClosestProof(proof, base)
	require.Error(t, err)

	proof, err = smst.ProveClosest(path[:])
	require.NoError(t, err)
	proof.FlippedBits[0] = -1
	require.EqualError(t, proof.validateBasic(base), "invalid flipped bit index 0: got -1, outside of [0, 8]")
	result, err = VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactClosestProof(proof, base)
	require.Error(t, err)
	proof.FlippedBits[0] = 9
	require.EqualError(t, proof.validateBasic(base), "invalid flipped bit index 0: got 9, outside of [0, 8]")
	result, err = VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactClosestProof(proof, base)
	require.Error(t, err)

	proof, err = smst.ProveClosest(path[:])
	require.NoError(t, err)
	flipPathBit(proof.Path, 3)
	require.EqualError(
		t,
		proof.validateBasic(base),
		"invalid closest path: 8d13809f932d0296b88c1913231ab4b403f05c88363575476204fef6930f22ae (not equal at bit: 3)",
	)
	result, err = VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, result)
	_, err = CompactClosestProof(proof, base)
	require.Error(t, err)
}

// ProveClosest test against a visual representation of the trie
// See: https://github.com/pokt-network/smt/assets/53987565/2a2f33e0-f81f-41c5-bd76-af0cd1cd8f15
func TestSMST_ProveClosest(t *testing.T) {
	var smn KVStore
	var smst *SMST
	var proof *SparseMerkleClosestProof
	var result bool
	var root []byte
	var err error
	var sumBz [sumSize]byte

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smst = NewSparseMerkleSumTrie(smn, sha256.New(), WithValueHasher(nil))

	// insert some unrelated values to populate the trie
	require.NoError(t, smst.Update([]byte("foo"), []byte("oof"), 3))
	require.NoError(t, smst.Update([]byte("bar"), []byte("rab"), 6))
	require.NoError(t, smst.Update([]byte("baz"), []byte("zab"), 9))
	require.NoError(t, smst.Update([]byte("bin"), []byte("nib"), 12))
	require.NoError(t, smst.Update([]byte("fiz"), []byte("zif"), 15))
	require.NoError(t, smst.Update([]byte("fob"), []byte("bof"), 18))
	require.NoError(t, smst.Update([]byte("testKey"), []byte("testValue"), 21))
	require.NoError(t, smst.Update([]byte("testKey2"), []byte("testValue2"), 24))
	require.NoError(t, smst.Update([]byte("testKey3"), []byte("testValue3"), 27))
	require.NoError(t, smst.Update([]byte("testKey4"), []byte("testValue4"), 30))

	root = smst.Root()

	// `testKey2` is the child of an inner node, which is the child of an extension node.
	// The extension node has the path bounds of [3, 7]. This means any bits between
	// 3-6 can be flipped, and the resulting path would still traverse through the same
	// extension node and lead to testKey2 - the closest key. However, flipping bit 7
	// will lead to testKey4.
	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)
	proof, err = smst.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, smst.Spec())
	require.NotEqual(t, proof, &SparseMerkleClosestProof{})
	closestPath := sha256.Sum256([]byte("testKey2"))
	closestValueHash := []byte("testValue2")
	binary.LittleEndian.PutUint64(sumBz[:], 24)
	closestValueHash = append(closestValueHash, sumBz[:]...)
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:             path[:],
		FlippedBits:      []int{3, 6},
		Depth:            8,
		ClosestPath:      closestPath[:],
		ClosestValueHash: closestValueHash,
		ClosestProof:     proof.ClosestProof, // copy of proof as we are checking equality of other fields
	})

	result, err = VerifyClosestProof(proof, root, NoPrehashSpec(sha256.New(), true))
	require.NoError(t, err)
	require.True(t, result)

	// testKey4 is the neighbour of testKey2, by flipping the final bit of the
	// extension node we change the longest common prefix to that of testKey4
	path2 := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path2[:], 3)
	flipPathBit(path2[:], 7)
	proof, err = smst.ProveClosest(path2[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, smst.Spec())
	require.NotEqual(t, proof, &SparseMerkleClosestProof{})
	closestPath = sha256.Sum256([]byte("testKey4"))
	closestValueHash = []byte("testValue4")
	binary.LittleEndian.PutUint64(sumBz[:], 30)
	closestValueHash = append(closestValueHash, sumBz[:]...)
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:             path2[:],
		FlippedBits:      []int{3},
		Depth:            8,
		ClosestPath:      closestPath[:],
		ClosestValueHash: closestValueHash,
		ClosestProof:     proof.ClosestProof, // copy of proof as we are checking equality of other fields
	})

	result, err = VerifyClosestProof(proof, root, NoPrehashSpec(sha256.New(), true))
	require.NoError(t, err)
	require.True(t, result)

	require.NoError(t, smn.Stop())
}

func TestSMST_ProveClosest_Empty(t *testing.T) {
	var smn KVStore
	var smst *SMST
	var proof *SparseMerkleClosestProof
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smst = NewSparseMerkleSumTrie(smn, sha256.New(), WithValueHasher(nil))

	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)
	proof, err = smst.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, smst.Spec())
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:         path[:],
		FlippedBits:  []int{0},
		Depth:        0,
		ClosestPath:  placeholder(smst.Spec()),
		ClosestProof: &SparseMerkleProof{},
	})

	result, err := VerifyClosestProof(proof, smst.Root(), NoPrehashSpec(sha256.New(), true))
	require.NoError(t, err)
	require.True(t, result)

	require.NoError(t, smn.Stop())
}

func TestSMST_ProveClosest_OneNode(t *testing.T) {
	var smn KVStore
	var smst *SMST
	var proof *SparseMerkleClosestProof
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smst = NewSparseMerkleSumTrie(smn, sha256.New(), WithValueHasher(nil))

	require.NoError(t, smst.Update([]byte("foo"), []byte("bar"), 5))

	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)
	proof, err = smst.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, smst.Spec())

	closestPath := sha256.Sum256([]byte("foo"))
	closestValueHash := []byte("bar")
	var sumBz [sumSize]byte
	binary.LittleEndian.PutUint64(sumBz[:], 5)
	closestValueHash = append(closestValueHash, sumBz[:]...)
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:             path[:],
		FlippedBits:      []int{},
		Depth:            0,
		ClosestPath:      closestPath[:],
		ClosestValueHash: closestValueHash,
		ClosestProof:     &SparseMerkleProof{},
	})

	result, err := VerifyClosestProof(proof, smst.Root(), NoPrehashSpec(sha256.New(), true))
	require.NoError(t, err)
	require.True(t, result)

	require.NoError(t, smn.Stop())
}

func TestSMST_ProveClosest_Proof(t *testing.T) {
	var smn KVStore
	var smst256 *SMST
	var smst512 *SMST
	var proof256 *SparseMerkleClosestProof
	var proof512 *SparseMerkleClosestProof
	var err error

	// setup trie (256+512 path hasher) and nodestore
	smn, err = NewKVStore("")
	require.NoError(t, err)
	smst256 = NewSparseMerkleSumTrie(smn, sha256.New())
	smst512 = NewSparseMerkleSumTrie(smn, sha512.New())

	// insert 100000 key-value-sum triples
	for i := 0; i < 100000; i++ {
		s := strconv.Itoa(i)
		require.NoError(t, smst256.Update([]byte(s), []byte(s), uint64(i)))
		require.NoError(t, smst512.Update([]byte(s), []byte(s), uint64(i)))
		// generate proofs for each key in the trie
		path256 := sha256.Sum256([]byte(s))
		path512 := sha512.Sum512([]byte(s))
		proof256, err = smst256.ProveClosest(path256[:])
		require.NoError(t, err)
		proof512, err = smst512.ProveClosest(path512[:])
		require.NoError(t, err)
		// ensure proof is same after compression and decompression
		checkClosestCompactEquivalence(t, proof256, smst256.Spec())
		checkClosestCompactEquivalence(t, proof512, smst512.Spec())
	}

	require.NoError(t, smn.Stop())
}
