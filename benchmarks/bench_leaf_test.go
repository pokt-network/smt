package smt

import (
	"crypto/sha256"
	"fmt"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func BenchmarkSMTLeafSizes_Fill(b *testing.B) {
	treeSizes := []int{100000, 500000, 1000000, 5000000, 10000000} // number of leaves
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}    // number of bytes per leaf
	nodes, err := smt.NewKVStore("")
	require.NoError(b, err)
	for _, treeSize := range treeSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			for _, operation := range []string{"Fill", "Fill & Commit"} {
				tree := smt.NewSparseMerkleTree(nodes, sha256.New(), smt.WithValueHasher(nil))
				b.ResetTimer()
				b.Run(
					fmt.Sprintf("%s [Leaf Size: %d bytes] (%d)", operation, leafSize, treeSize),
					func(b *testing.B) {
						b.ResetTimer()
						b.ReportAllocs()
						for i := 0; i < treeSize; i++ {
							require.NoError(b, tree.Update([]byte(strconv.Itoa(i)), leaf))
						}
						if operation == "Fill & Commit" {
							require.NoError(b, tree.Commit())
						}
					},
				)
				require.NoError(b, nodes.ClearAll())
			}
		}
	}
	require.NoError(b, nodes.Stop())
}

func BenchmarkSMSTLeafSizes_Fill(b *testing.B) {
	treeSizes := []int{100000, 500000, 1000000, 5000000, 10000000} // number of leaves
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}    // number of bytes per leaf
	nodes, err := smt.NewKVStore("")
	require.NoError(b, err)
	for _, treeSize := range treeSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			for _, operation := range []string{"Fill", "Fill & Commit"} {
				tree := smt.NewSparseMerkleSumTree(nodes, sha256.New(), smt.WithValueHasher(nil))
				b.ResetTimer()
				b.Run(
					fmt.Sprintf("%s [Leaf Size: %d bytes] (%d)", operation, leafSize, treeSize),
					func(b *testing.B) {
						b.ResetTimer()
						b.ReportAllocs()
						for i := 0; i < treeSize; i++ {
							require.NoError(b, tree.Update([]byte(strconv.Itoa(i)), leaf, uint64(i)))
						}
						if operation == "Fill & Commit" {
							require.NoError(b, tree.Commit())
						}
					},
				)
				require.NoError(b, nodes.ClearAll())
			}
		}
	}
	require.NoError(b, nodes.Stop())
}
