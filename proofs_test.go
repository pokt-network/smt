package smt

import (
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

func TestSparseMerkleProof_Marshal(t *testing.T) {
	trie := setupTrie(t)

	proof, err := trie.Prove([]byte("key"))
	require.NoError(t, err)
	bz, err := proof.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz)
	require.Greater(t, len(bz), 0)

	proof2, err := trie.Prove([]byte("key2"))
	require.NoError(t, err)
	bz2, err := proof2.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz2)
	require.Greater(t, len(bz2), 0)
	require.NotEqual(t, bz, bz2)

	proof3 := randomizeProof(proof)
	bz3, err := proof3.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz3)
	require.Greater(t, len(bz3), 0)
	require.NotEqual(t, bz, bz3)
}

func TestSparseMerkleProof_Unmarshal(t *testing.T) {
	trie := setupTrie(t)

	proof, err := trie.Prove([]byte("key"))
	require.NoError(t, err)
	bz, err := proof.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz)
	require.Greater(t, len(bz), 0)
	uproof := new(SparseMerkleProof)
	require.NoError(t, uproof.Unmarshal(bz))
	require.Equal(t, proof, uproof)

	proof2, err := trie.Prove([]byte("key2"))
	require.NoError(t, err)
	bz2, err := proof2.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz2)
	require.Greater(t, len(bz2), 0)
	uproof2 := new(SparseMerkleProof)
	require.NoError(t, uproof2.Unmarshal(bz2))
	require.Equal(t, proof2, uproof2)

	proof3 := randomizeProof(proof)
	bz3, err := proof3.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz3)
	require.Greater(t, len(bz3), 0)
	uproof3 := new(SparseMerkleProof)
	require.NoError(t, uproof3.Unmarshal(bz3))
	require.Equal(t, proof3, uproof3)
}

func TestSparseCompactMerkleProof_Marshal(t *testing.T) {
	trie := setupTrie(t)

	proof, err := trie.Prove([]byte("key"))
	require.NoError(t, err)
	compactProof, err := CompactProof(proof, trie.Spec())
	require.NoError(t, err)
	bz, err := compactProof.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz)
	require.Greater(t, len(bz), 0)

	proof2, err := trie.Prove([]byte("key2"))
	require.NoError(t, err)
	compactProof2, err := CompactProof(proof2, trie.Spec())
	require.NoError(t, err)
	bz2, err := compactProof2.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz2)
	require.Greater(t, len(bz2), 0)
	require.NotEqual(t, bz, bz2)

	proof3 := randomizeProof(proof)
	compactProof3, err := CompactProof(proof3, trie.Spec())
	require.NoError(t, err)
	bz3, err := compactProof3.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz3)
	require.Greater(t, len(bz3), 0)
	require.NotEqual(t, bz, bz3)
}

func TestSparseCompactMerkleProof_Unmarshal(t *testing.T) {
	trie := setupTrie(t)

	proof, err := trie.Prove([]byte("key"))
	require.NoError(t, err)
	compactProof, err := CompactProof(proof, trie.Spec())
	require.NoError(t, err)
	bz, err := compactProof.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz)
	require.Greater(t, len(bz), 0)
	uCproof := new(SparseCompactMerkleProof)
	require.NoError(t, uCproof.Unmarshal(bz))
	require.Equal(t, compactProof, uCproof)
	uproof, err := DecompactProof(uCproof, trie.Spec())
	require.NoError(t, err)
	require.Equal(t, proof, uproof)

	proof2, err := trie.Prove([]byte("key2"))
	require.NoError(t, err)
	compactProof2, err := CompactProof(proof2, trie.Spec())
	require.NoError(t, err)
	bz2, err := compactProof2.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz2)
	require.Greater(t, len(bz2), 0)
	uCproof2 := new(SparseCompactMerkleProof)
	require.NoError(t, uCproof2.Unmarshal(bz2))
	require.Equal(t, compactProof2, uCproof2)
	uproof2, err := DecompactProof(uCproof2, trie.Spec())
	require.NoError(t, err)
	require.Equal(t, proof2, uproof2)

	proof3 := randomizeProof(proof)
	compactProof3, err := CompactProof(proof3, trie.Spec())
	require.NoError(t, err)
	bz3, err := compactProof3.Marshal()
	require.NoError(t, err)
	require.NotNil(t, bz3)
	require.Greater(t, len(bz3), 0)
	uCproof3 := new(SparseCompactMerkleProof)
	require.NoError(t, uCproof3.Unmarshal(bz3))
	require.Equal(t, compactProof3, uCproof3)
	uproof3, err := DecompactProof(uCproof3, trie.Spec())
	require.NoError(t, err)
	require.Equal(t, proof3, uproof3)
}

func setupTrie(t *testing.T) *SMT {
	t.Helper()

	db := simplemap.NewSimpleMap()

	trie := NewSparseMerkleTrie(db, sha256.New())
	require.NoError(t, trie.Update([]byte("key"), []byte("value")))
	require.NoError(t, trie.Update([]byte("key2"), []byte("value2")))
	require.NoError(t, trie.Update([]byte("key3"), []byte("value3")))

	return trie
}

func randomizeProof(proof *SparseMerkleProof) *SparseMerkleProof {
	sideNodes := make([][]byte, len(proof.SideNodes))
	for i := range sideNodes {
		sideNodes[i] = make([]byte, len(proof.SideNodes[i]))
		rand.Read(sideNodes[i]) // nolint: errcheck
	}
	return &SparseMerkleProof{
		SideNodes:             sideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
	}
}

func randomizeSumProof(proof *SparseMerkleProof) *SparseMerkleProof {
	sideNodes := make([][]byte, len(proof.SideNodes))
	for i := range sideNodes {
		sideNodes[i] = make([]byte, len(proof.SideNodes[i])-sumSizeBytes)
		rand.Read(sideNodes[i]) // nolint: errcheck
		sideNodes[i] = append(sideNodes[i], proof.SideNodes[i][len(proof.SideNodes[i])-sumSizeBytes:]...)
	}
	return &SparseMerkleProof{
		SideNodes:             sideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
	}
}

// Check that a non-compact proof is equivalent to the proof returned when it is compacted and de-compacted.
func checkCompactEquivalence(t *testing.T, proof *SparseMerkleProof, base *TrieSpec) {
	t.Helper()
	compactedProof, err := CompactProof(proof, base)
	require.NoErrorf(t, err, "failed to compact proof: %v", err)
	decompactedProof, err := DecompactProof(compactedProof, base)
	require.NoErrorf(t, err, "failed to decompact proof: %v", err)
	require.Equal(t, proof, decompactedProof)
}

// Check that a non-compact proof is equivalent to the proof returned when it is compacted and de-compacted.
func checkClosestCompactEquivalence(t *testing.T, proof *SparseMerkleClosestProof, spec *TrieSpec) {
	t.Helper()
	compactedProof, err := CompactClosestProof(proof, spec)
	require.NoErrorf(t, err, "failed to compact proof: %v", err)
	decompactedProof, err := DecompactClosestProof(compactedProof, spec)
	require.NoErrorf(t, err, "failed to decompact proof: %v", err)
	require.Equal(t, proof, decompactedProof)
}
