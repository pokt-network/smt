package smt

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test base case Merkle proof operations.
func TestSMST_ProofsBasic(t *testing.T) {
	var smn, smv *SimpleMap
	var smst *SMSTWithStorage
	var proof SparseMerkleProof
	var result bool
	var root []byte
	var err error

	smn, smv = NewSimpleMap(), NewSimpleMap()
	smst = NewSMSTWithStorage(smn, smv, sha256.New())
	base := smst.Spec()

	// Generate and verify a proof on an empty key.
	proof, err = smst.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifySumProof(proof, base.th.sumPlaceholder(), []byte("testKey3"), defaultValue, 0, base)
	require.True(t, result)
	result = VerifySumProof(proof, root, []byte("testKey3"), []byte("badValue"), 5, base)
	require.False(t, result)

	// Add a key, generate and verify a Merkle proof.
	err = smst.Update([]byte("testKey"), []byte("testValue"), 5)
	require.NoError(t, err)
	root = smst.Root()
	proof, err = smst.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 5, base)
	require.True(t, result)
	result = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 10, base)
	require.False(t, result)

	// Add a key, generate and verify both Merkle proofs.
	err = smst.Update([]byte("testKey2"), []byte("testValue"), 5)
	require.NoError(t, err)
	root = smst.Root()
	proof, err = smst.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifySumProof(proof, root, []byte("testKey"), []byte("testValue"), 5, base)
	require.True(t, result)
	result = VerifySumProof(proof, root, []byte("testKey"), []byte("badValue"), 10, base)
	require.False(t, result)
	result = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey"), []byte("testValue"), 5, base)
	require.False(t, result)

	proof, err = smst.Prove([]byte("testKey2"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifySumProof(proof, root, []byte("testKey2"), []byte("testValue"), 5, base)
	require.True(t, result)
	result = VerifySumProof(proof, root, []byte("testKey2"), []byte("badValue"), 10, base)
	require.False(t, result)
	result = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey2"), []byte("testValue"), 5, base)
	require.False(t, result)

	// Try proving a default value for a non-default leaf.
	var sum [sumSize]byte
	binary.BigEndian.PutUint64(sum[:], 5)
	tval := base.digestValue([]byte("testValue"))
	tval = append(tval, sum[:]...)
	_, leafData := base.th.digestSumLeaf(base.ph.Path([]byte("testKey2")), tval)
	proof = SparseMerkleProof{
		SideNodes:             proof.SideNodes,
		NonMembershipLeafData: leafData,
	}
	result = VerifySumProof(proof, root, []byte("testKey2"), defaultValue, 0, base)
	require.False(t, result)

	// Generate and verify a proof on an empty key.
	proof, err = smst.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result = VerifySumProof(proof, root, []byte("testKey3"), defaultValue, 0, base)
	require.True(t, result)
	result = VerifySumProof(proof, root, []byte("testKey3"), []byte("badValue"), 5, base)
	require.False(t, result)
	result = VerifySumProof(randomiseSumProof(proof), root, []byte("testKey3"), defaultValue, 0, base)
	require.False(t, result)
}

// Test sanity check cases for non-compact proofs.
func TestSMST_ProofsSanityCheck(t *testing.T) {
	smn, smv := NewSimpleMap(), NewSimpleMap()
	smst := NewSMSTWithStorage(smn, smv, sha256.New())
	base := smst.Spec()

	err := smst.Update([]byte("testKey1"), []byte("testValue1"), 1)
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
	require.False(t, proof.sanityCheck(base))
	result := VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect size for NonMembershipLeafData.
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.NonMembershipLeafData = make([]byte, 1)
	require.False(t, proof.sanityCheck(base))
	result = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: unexpected sidenode size.
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.SideNodes[0] = make([]byte, 1)
	require.False(t, proof.sanityCheck(base))
	result = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect non-nil sibling data
	proof, _ = smst.Prove([]byte("testKey1"))
	proof.SiblingData = base.th.digest(proof.SiblingData)
	require.False(t, proof.sanityCheck(base))

	result = VerifySumProof(proof, root, []byte("testKey1"), []byte("testValue1"), 1, base)
	require.False(t, result)
	_, err = CompactProof(proof, base)
	require.Error(t, err)
}
