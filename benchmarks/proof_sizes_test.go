package benchmarks

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
	"github.com/stretchr/testify/require"
)

func TestSMT_ProofSizes(t *testing.T) {
	nodes := simplemap.New()

	testCases := []struct {
		name     string
		treeSize int
	}{
		{
			name:     "Proof Size (Prefilled: 100000)",
			treeSize: 100000,
		},
		{
			name:     "Proof Size (Prefilled: 500000)",
			treeSize: 500000,
		},
		{
			name:     "Proof Size (Prefilled: 1000000)",
			treeSize: 1000000,
		},
		{
			name:     "Proof Size (Prefilled: 5000000)",
			treeSize: 5000000,
		},
		{
			name:     "Proof Size (Prefilled: 10000000)",
			treeSize: 10000000,
		},
	}
	for _, tc := range testCases {
		tree := smt.NewSparseMerkleTree(nodes, sha256.New())
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < tc.treeSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				require.NoError(t, tree.Update(b, b))
			}
			require.NoError(t, tree.Commit())
			avgProof := uint64(0)
			maxProof := uint64(0)
			minProof := uint64(0)
			avgCompact := uint64(0)
			maxCompact := uint64(0)
			minCompact := uint64(0)
			for i := 0; i < tc.treeSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				proof, err := tree.Prove(b)
				require.NoError(t, err)
				require.NotNil(t, proof)
				compacted, err := smt.CompactProof(proof, tree.Spec())
				require.NoError(t, err)
				require.NotNil(t, compacted)
				proofBz, err := proof.Marshal()
				require.NoError(t, err)
				require.NotNil(t, proofBz)
				compactedBz, err := compacted.Marshal()
				require.NoError(t, err)
				require.NotNil(t, compactedBz)
				avgProof += uint64(len(proofBz))
				if uint64(len(proofBz)) > maxProof {
					maxProof = uint64(len(proofBz))
				}
				if uint64(len(proofBz)) < minProof || i == 0 {
					minProof = uint64(len(proofBz))
				}
				avgCompact += uint64(len(compactedBz))
				if uint64(len(compactedBz)) > maxCompact {
					maxCompact = uint64(len(compactedBz))
				}
				if uint64(len(compactedBz)) < minCompact || i == 0 {
					minCompact = uint64(len(compactedBz))
				}
			}
			avgProof /= uint64(tc.treeSize)
			avgCompact /= uint64(tc.treeSize)
			t.Logf("Average Serialised Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgProof, minProof, maxProof, tc.treeSize)
			t.Logf("Average Serialised Compacted Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgCompact, minCompact, maxCompact, tc.treeSize)
		})
		require.NoError(t, nodes.ClearAll())
	}
	require.NoError(t, nodes.Stop())
}

func TestSMST_ProofSizes(t *testing.T) {
	nodes := simplemap.New()

	testCases := []struct {
		name     string
		treeSize int
	}{
		{
			name:     "Proof Size (Prefilled: 100000)",
			treeSize: 100000,
		},
		{
			name:     "Proof Size (Prefilled: 500000)",
			treeSize: 500000,
		},
		{
			name:     "Proof Size (Prefilled: 1000000)",
			treeSize: 1000000,
		},
		{
			name:     "Proof Size (Prefilled: 5000000)",
			treeSize: 5000000,
		},
		{
			name:     "Proof Size (Prefilled: 10000000)",
			treeSize: 10000000,
		},
	}
	for _, tc := range testCases {
		tree := smt.NewSparseMerkleSumTree(nodes, sha256.New())
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < tc.treeSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				require.NoError(t, tree.Update(b, b, uint64(i)))
			}
			require.NoError(t, tree.Commit())
			avgProof := uint64(0)
			maxProof := uint64(0)
			minProof := uint64(0)
			avgCompact := uint64(0)
			maxCompact := uint64(0)
			minCompact := uint64(0)
			for i := 0; i < tc.treeSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				proof, err := tree.Prove(b)
				require.NoError(t, err)
				require.NotNil(t, proof)
				compacted, err := smt.CompactProof(proof, tree.Spec())
				require.NoError(t, err)
				require.NotNil(t, compacted)
				proofBz, err := proof.Marshal()
				require.NoError(t, err)
				require.NotNil(t, proofBz)
				compactedBz, err := compacted.Marshal()
				require.NoError(t, err)
				require.NotNil(t, compactedBz)
				avgProof += uint64(len(proofBz))
				if uint64(len(proofBz)) > maxProof {
					maxProof = uint64(len(proofBz))
				}
				if uint64(len(proofBz)) < minProof || i == 0 {
					minProof = uint64(len(proofBz))
				}
				avgCompact += uint64(len(compactedBz))
				if uint64(len(compactedBz)) > maxCompact {
					maxCompact = uint64(len(compactedBz))
				}
				if uint64(len(compactedBz)) < minCompact || i == 0 {
					minCompact = uint64(len(compactedBz))
				}
			}
			avgProof /= uint64(tc.treeSize)
			avgCompact /= uint64(tc.treeSize)
			t.Logf("Average Serialised Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgProof, minProof, maxProof, tc.treeSize)
			t.Logf("Average Serialised Compacted Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgCompact, minCompact, maxCompact, tc.treeSize)
		})
		require.NoError(t, nodes.ClearAll())
	}
	require.NoError(t, nodes.Stop())
}
