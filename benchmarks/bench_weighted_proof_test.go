//go:build benchmark

package smt

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// numBenchSeeds is the number of distinct seeds, and so proofs, cycled
// through by each benchmark.
const numBenchSeeds = 1024

// BenchmarkSparseMerkleSumTrie_ClosestVsWeighted compares closest and weighted
// proofs on the same trie: proving on a resident trie, proving on a freshly
// imported trie (every node read from the store), and verifying the proof in
// full and compact form. Tries are built as poktroll builds them, with a nil
// value hasher, which closest proof verification requires.
func BenchmarkSparseMerkleSumTrie_ClosestVsWeighted(b *testing.B) {
	for _, trieSize := range []int{10_000, 100_000, 1_000_000} {
		nodes, trie := setupWeightedBenchSMST(b, trieSize)
		root, spec := trie.Root(), trie.Spec()

		seeds := make([][]byte, numBenchSeeds)
		for i := range seeds {
			seed := sha256.Sum256(binary.BigEndian.AppendUint64(nil, uint64(i)))
			seeds[i] = seed[:]
		}

		provers := []struct {
			name  string
			prove func(*smt.SMST, []byte) (*smt.SparseMerkleClosestProof, error)
			// verify must match the proof type produced by prove.
			verify        func(*smt.SparseMerkleClosestProof, []byte) (bool, error)
			verifyCompact func(*smt.SparseCompactMerkleClosestProof, []byte) (bool, error)
		}{
			{
				name:  "Closest",
				prove: (*smt.SMST).ProveClosest,
				verify: func(p *smt.SparseMerkleClosestProof, _ []byte) (bool, error) {
					return smt.VerifyClosestProof(p, root, spec)
				},
				verifyCompact: func(p *smt.SparseCompactMerkleClosestProof, _ []byte) (bool, error) {
					return smt.VerifyCompactClosestProof(p, root, spec)
				},
			},
			{
				name:  "Weighted",
				prove: (*smt.SMST).ProveWeighted,
				verify: func(p *smt.SparseMerkleClosestProof, seed []byte) (bool, error) {
					return smt.VerifyWeightedProof(p, root, seed, spec)
				},
				verifyCompact: func(p *smt.SparseCompactMerkleClosestProof, seed []byte) (bool, error) {
					return smt.VerifyCompactWeightedProof(p, root, seed, spec)
				},
			},
		}

		for _, p := range provers {
			proofs := make([]*smt.SparseMerkleClosestProof, numBenchSeeds)
			compactProofs := make([]*smt.SparseCompactMerkleClosestProof, numBenchSeeds)
			compactBytes := 0
			for i, seed := range seeds {
				proof, err := p.prove(trie, seed)
				require.NoError(b, err)
				valid, err := p.verify(proof, seed)
				require.NoError(b, err)
				require.True(b, valid)
				proofs[i] = proof
				compactProofs[i], err = smt.CompactClosestProof(proof, spec)
				require.NoError(b, err)
				bz, err := compactProofs[i].Marshal()
				require.NoError(b, err)
				compactBytes += len(bz)
			}
			avgCompactBytes := float64(compactBytes) / numBenchSeeds

			prefix := fmt.Sprintf("%s/Prefilled_%d", p.name, trieSize)

			b.Run(prefix+"/Prove", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := p.prove(trie, seeds[i%numBenchSeeds]); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run(prefix+"/ProveFromStore", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					imported := smt.ImportSparseMerkleSumTrie(nodes, sha256.New(), root, smt.WithValueHasher(nil))
					if _, err := p.prove(imported, seeds[i%numBenchSeeds]); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run(prefix+"/Verify", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					j := i % numBenchSeeds
					if valid, err := p.verify(proofs[j], seeds[j]); !valid || err != nil {
						b.Fatal("proof did not verify", err)
					}
				}
			})

			b.Run(prefix+"/VerifyCompact", func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(avgCompactBytes, "proof-bytes")
				for i := 0; i < b.N; i++ {
					j := i % numBenchSeeds
					if valid, err := p.verifyCompact(compactProofs[j], seeds[j]); !valid || err != nil {
						b.Fatal("proof did not verify", err)
					}
				}
			})
		}
	}
}

func setupWeightedBenchSMST(b *testing.B, numLeaves int) (kvstore.MapStore, *smt.SMST) {
	b.Helper()
	nodes := simplemap.NewSimpleMap()
	trie := smt.NewSparseMerkleSumTrie(nodes, sha256.New(), smt.WithValueHasher(nil))
	for i := 0; i < numLeaves; i++ {
		key := binary.BigEndian.AppendUint64(nil, uint64(i))
		require.NoError(b, trie.Update(key, key, 1))
	}
	require.NoError(b, trie.Commit())
	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
	})
	return nodes, trie
}
