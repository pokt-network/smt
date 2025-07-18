//go:build benchmark

package smt

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func TestSMT_ProofSizes(t *testing.T) {
	nodes := simplemap.NewSimpleMap()
	testCases := []struct {
		desc     string
		trieSize int
	}{
		{
			desc:     "Proof Size (Prefilled: 100000)",
			trieSize: 100000,
		},
		{
			desc:     "Proof Size (Prefilled: 50000)",
			trieSize: 50000,
		},
	}
	for _, tc := range testCases {
		trie := smt.NewSparseMerkleTrie(nodes, sha256.New())
		t.Run(tc.desc, func(t *testing.T) {
			for i := 0; i < tc.trieSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				require.NoError(t, trie.Update(b, b))
			}
			require.NoError(t, trie.Commit())
			avgProof := uint64(0)
			maxProof := uint64(0)
			minProof := uint64(0)
			avgCompact := uint64(0)
			maxCompact := uint64(0)
			minCompact := uint64(0)
			for i := 0; i < tc.trieSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				proof, err := trie.Prove(b)
				require.NoError(t, err)
				require.NotNil(t, proof)
				compacted, err := smt.CompactProof(proof, trie.Spec())
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
			avgProof /= uint64(tc.trieSize)
			avgCompact /= uint64(tc.trieSize)
			t.Logf("Average Serialised Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgProof, minProof, maxProof, tc.trieSize)
			t.Logf(
				"Average Serialised Compacted Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)",
				avgCompact,
				minCompact,
				maxCompact,
				tc.trieSize,
			)
		})
		require.NoError(t, nodes.ClearAll())
	}
}

func TestSMST_ProofSizes(t *testing.T) {
	nodes := simplemap.NewSimpleMap()
	testCases := []struct {
		desc     string
		trieSize int
	}{
		{
			desc:     "Proof Size (Prefilled: 100000)",
			trieSize: 100000,
		},
		{
			desc:     "Proof Size (Prefilled: 50000)",
			trieSize: 50000,
		},
	}
	for _, tc := range testCases {
		trie := smt.NewSparseMerkleSumTrie(nodes, sha256.New())
		t.Run(tc.desc, func(t *testing.T) {
			for i := 0; i < tc.trieSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				require.NoError(t, trie.Update(b, b, uint64(i)))
			}
			require.NoError(t, trie.Commit())
			avgProof := uint64(0)
			maxProof := uint64(0)
			minProof := uint64(0)
			avgCompact := uint64(0)
			maxCompact := uint64(0)
			minCompact := uint64(0)
			for i := 0; i < tc.trieSize; i++ {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(i))
				proof, err := trie.Prove(b)
				require.NoError(t, err)
				require.NotNil(t, proof)
				compacted, err := smt.CompactProof(proof, trie.Spec())
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
			avgProof /= uint64(tc.trieSize)
			avgCompact /= uint64(tc.trieSize)
			t.Logf("Average Serialised Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)", avgProof, minProof, maxProof, tc.trieSize)
			t.Logf(
				"Average Serialised Compacted Proof Size: %d bytes [Min: %d || Max: %d] (Prefilled: %d)",
				avgCompact,
				minCompact,
				maxCompact,
				tc.trieSize,
			)
		})
		require.NoError(t, nodes.ClearAll())
	}
}
