package smt

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test base case Merkle proof operations.
func TestSMT_ProofsBasic(t *testing.T) {
	var smn, smv KVStore
	var smt *SMTWithStorage
	var proof *SparseMerkleProof
	var result bool
	var root []byte
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smv, err = NewKVStore("")
	require.NoError(t, err)
	smt = NewSMTWithStorage(smn, smv, sha256.New())
	base := smt.Spec()

	// Generate and verify a proof on an empty key.
	proof, err = smt.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifyProof(proof, base.th.placeholder(), []byte("testKey3"), defaultValue, base)
	require.True(t, result)
	result = VerifyProof(proof, root, []byte("testKey3"), []byte("badValue"), base)
	require.False(t, result)

	// Add a key, generate and verify a Merkle proof.
	err = smt.Update([]byte("testKey"), []byte("testValue"))
	require.NoError(t, err)
	root = smt.Root()
	proof, err = smt.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifyProof(proof, root, []byte("testKey"), []byte("testValue"), base)
	require.True(t, result)
	result = VerifyProof(proof, root, []byte("testKey"), []byte("badValue"), base)
	require.False(t, result)

	// Add a key, generate and verify both Merkle proofs.
	err = smt.Update([]byte("testKey2"), []byte("testValue"))
	require.NoError(t, err)
	root = smt.Root()
	proof, err = smt.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifyProof(proof, root, []byte("testKey"), []byte("testValue"), base)
	require.True(t, result)
	result = VerifyProof(proof, root, []byte("testKey"), []byte("badValue"), base)
	require.False(t, result)
	result = VerifyProof(randomiseProof(proof), root, []byte("testKey"), []byte("testValue"), base)
	require.False(t, result)

	proof, err = smt.Prove([]byte("testKey2"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifyProof(proof, root, []byte("testKey2"), []byte("testValue"), base)
	require.True(t, result)
	result = VerifyProof(proof, root, []byte("testKey2"), []byte("badValue"), base)
	require.False(t, result)
	result = VerifyProof(randomiseProof(proof), root, []byte("testKey2"), []byte("testValue"), base)
	require.False(t, result)

	// Try proving a default value for a non-default leaf.
	_, leafData := base.th.digestLeaf(base.ph.Path([]byte("testKey2")), base.digestValue([]byte("testValue")))
	proof = &SparseMerkleProof{
		SideNodes:             proof.SideNodes,
		NonMembershipLeafData: leafData,
	}
	result = VerifyProof(proof, root, []byte("testKey2"), defaultValue, base)
	require.False(t, result)

	// Generate and verify a proof on an empty key.
	proof, err = smt.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifyProof(proof, root, []byte("testKey3"), defaultValue, base)
	require.True(t, result)
	result = VerifyProof(proof, root, []byte("testKey3"), []byte("badValue"), base)
	require.False(t, result)
	result = VerifyProof(randomiseProof(proof), root, []byte("testKey3"), defaultValue, base)
	require.False(t, result)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

// Test sanity check cases for non-compact proofs.
func TestSMT_ProofsSanityCheck(t *testing.T) {
	smn, err := NewKVStore("")
	require.NoError(t, err)
	smv, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSMTWithStorage(smn, smv, sha256.New())
	base := smt.Spec()

	err = smt.Update([]byte("testKey1"), []byte("testValue1"))
	require.NoError(t, err)
	err = smt.Update([]byte("testKey2"), []byte("testValue2"))
	require.NoError(t, err)
	err = smt.Update([]byte("testKey3"), []byte("testValue3"))
	require.NoError(t, err)
	err = smt.Update([]byte("testKey4"), []byte("testValue4"))
	require.NoError(t, err)
	root := smt.Root()

	// Case: invalid number of sidenodes.
	proof, _ := smt.Prove([]byte("testKey1"))
	sideNodes := make([][]byte, smt.Spec().depth()+1)
	for i := range sideNodes {
		sideNodes[i] = proof.SideNodes[0]
	}
	proof.SideNodes = sideNodes
	require.False(t, proof.sanityCheck(base))
	result := VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect size for NonMembershipLeafData.
	proof, _ = smt.Prove([]byte("testKey1"))
	proof.NonMembershipLeafData = make([]byte, 1)
	require.False(t, proof.sanityCheck(base))
	result = VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: unexpected sidenode size.
	proof, _ = smt.Prove([]byte("testKey1"))
	proof.SideNodes[0] = make([]byte, 1)
	require.False(t, proof.sanityCheck(base))
	result = VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect non-nil sibling data
	proof, _ = smt.Prove([]byte("testKey1"))
	proof.SiblingData = base.th.digest(proof.SiblingData)
	require.False(t, proof.sanityCheck(base))

	result = VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

// ProveClosest test against a visual representation of the tree
// See: https://github.com/pokt-network/smt/assets/53987565/2c2ea530-a2e8-49d7-89c2-ca9c615b0c79
func TestSMT_ProveClosest(t *testing.T) {
	var smn KVStore
	var smt *SMT
	var proof *SparseMerkleClosestProof
	var result bool
	var root []byte
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smt = NewSparseMerkleTree(smn, sha256.New(), WithValueHasher(nil))

	// insert some unrelated values to populate the tree
	require.NoError(t, smt.Update([]byte("foo"), []byte("oof")))
	require.NoError(t, smt.Update([]byte("bar"), []byte("rab")))
	require.NoError(t, smt.Update([]byte("baz"), []byte("zab")))
	require.NoError(t, smt.Update([]byte("bin"), []byte("nib")))
	require.NoError(t, smt.Update([]byte("fiz"), []byte("zif")))
	require.NoError(t, smt.Update([]byte("fob"), []byte("bof")))
	require.NoError(t, smt.Update([]byte("testKey"), []byte("testValue")))
	require.NoError(t, smt.Update([]byte("testKey2"), []byte("testValue2")))
	require.NoError(t, smt.Update([]byte("testKey3"), []byte("testValue3")))
	require.NoError(t, smt.Update([]byte("testKey4"), []byte("testValue4")))

	root = smt.Root()

	// `testKey2` is the child of an inner node, which is the child of an extension node.
	// The extension node has the path bounds of [3, 7]. This means any bits between
	// 3-6 can be flipped, and the resulting path would still traverse through the same
	// extension node and lead to testKey2 - the closest key. However, flipping bit 7
	// will lead to testKey4.
	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)
	proof, err = smt.ProveClosest(path[:])
	require.NoError(t, err)
	require.NotEqual(t, proof, &SparseMerkleClosestProof{})

	result = VerifyClosestProof(proof, root, NoPrehashSpec(sha256.New(), false))
	require.True(t, result)
	closestPath := sha256.Sum256([]byte("testKey2"))
	require.Equal(t, closestPath[:], proof.ClosestPath)
	require.Equal(t, []byte("testValue2"), proof.ClosestValueHash)

	// testKey4 is the neighbour of testKey2, by flipping the final bit of the
	// extension node we change the longest common prefix to that of testKey4
	path2 := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path2[:], 3)
	flipPathBit(path2[:], 7)
	proof, err = smt.ProveClosest(path2[:])
	require.NoError(t, err)
	require.NotEqual(t, proof, &SparseMerkleClosestProof{})

	result = VerifyClosestProof(proof, root, NoPrehashSpec(sha256.New(), false))
	require.True(t, result)
	closestPath = sha256.Sum256([]byte("testKey4"))
	require.Equal(t, closestPath[:], proof.ClosestPath)
	require.Equal(t, []byte("testValue4"), proof.ClosestValueHash)

	require.NoError(t, smn.Stop())
}

func TestSMT_ProveClosestEmptyAndOneNode(t *testing.T) {
	var smn KVStore
	var smt *SMT
	var proof *SparseMerkleClosestProof
	var err error

	smn, err = NewKVStore("")
	require.NoError(t, err)
	smt = NewSparseMerkleTree(smn, sha256.New(), WithValueHasher(nil))

	path := sha256.Sum256([]byte("testKey2"))
	flipPathBit(path[:], 3)
	flipPathBit(path[:], 6)
	proof, err = smt.ProveClosest(path[:])
	require.NoError(t, err)
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:         path[:],
		FlippedBits:  []int{0},
		Depth:        0,
		ClosestPath:  placeholder(smt.Spec()),
		ClosestProof: &SparseMerkleProof{},
	})

	result := VerifyClosestProof(proof, smt.Root(), NoPrehashSpec(sha256.New(), false))
	require.True(t, result)

	require.NoError(t, smt.Update([]byte("foo"), []byte("bar")))
	proof, err = smt.ProveClosest(path[:])
	require.NoError(t, err)
	closestPath := sha256.Sum256([]byte("foo"))
	require.Equal(t, proof, &SparseMerkleClosestProof{
		Path:             path[:],
		FlippedBits:      []int{},
		Depth:            0,
		ClosestPath:      closestPath[:],
		ClosestValueHash: []byte("bar"),
		ClosestProof:     &SparseMerkleProof{},
	})

	result = VerifyClosestProof(proof, smt.Root(), NoPrehashSpec(sha256.New(), false))
	require.True(t, result)

	require.NoError(t, smn.Stop())
}
