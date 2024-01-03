//go:build benchmark

package smt

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func BenchmarkSMTLeafSizes_Fill(b *testing.B) {
	trieSizes := []int{100000, 500000, 1000000, 5000000, 10000000} // number of leaves
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}    // number of bytes per leaf
	nodes := simplemap.NewSimpleMap()
	for _, trieSize := range trieSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			for _, operation := range []string{"Fill", "Fill & Commit"} {
				trie := smt.NewSparseMerkleTrie(nodes, sha256.New(), smt.WithValueHasher(nil))
				b.ResetTimer()
				b.Run(
					fmt.Sprintf("%s [Leaf Size: %d bytes] (%d)", operation, leafSize, trieSize),
					func(b *testing.B) {
						b.ResetTimer()
						b.ReportAllocs()
						for i := 0; i < trieSize; i++ {
							require.NoError(b, trie.Update([]byte(strconv.Itoa(i)), leaf))
						}
						if operation == "Fill & Commit" {
							require.NoError(b, trie.Commit())
						}
					},
				)
				require.NoError(b, nodes.ClearAll())
			}
		}
	}
}

func BenchmarkSMSTLeafSizes_Fill(b *testing.B) {
	trieSizes := []int{100000, 500000, 1000000, 5000000, 10000000} // number of leaves
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}    // number of bytes per leaf
	nodes := simplemap.NewSimpleMap()
	for _, trieSize := range trieSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			for _, operation := range []string{"Fill", "Fill & Commit"} {
				trie := smt.NewSparseMerkleSumTrie(nodes, sha256.New(), smt.WithValueHasher(nil))
				b.ResetTimer()
				b.Run(
					fmt.Sprintf("%s [Leaf Size: %d bytes] (%d)", operation, leafSize, trieSize),
					func(b *testing.B) {
						b.ResetTimer()
						b.ReportAllocs()
						for i := 0; i < trieSize; i++ {
							require.NoError(b, trie.Update([]byte(strconv.Itoa(i)), leaf, uint64(i)))
						}
						if operation == "Fill & Commit" {
							require.NoError(b, trie.Commit())
						}
					},
				)
				require.NoError(b, nodes.ClearAll())
			}
		}
	}
}
