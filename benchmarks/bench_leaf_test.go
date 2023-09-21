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
	treeSizes := []int{100000, 500000, 1000000, 5000000, 10000000}
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}
	nodes, err := smt.NewKVStore("")
	require.NoError(b, err)
	for _, treeSize := range treeSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			tree := smt.NewSparseMerkleTree(nodes, sha256.New(), smt.WithValueHasher(nil))
			b.ResetTimer()
			b.Run(
				fmt.Sprintf("Fill [Leaf Size: %d bytes] (%d)", leafSize, treeSize),
				func(b *testing.B) {
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < treeSize; i++ {
						require.NoError(b, tree.Update([]byte(strconv.Itoa(i)), leaf))
					}
				},
			)
			b.ResetTimer()
			b.Run(
				fmt.Sprintf("Fill & Commit [Leaf Size: %d bytes] (%d)", leafSize, treeSize),
				func(b *testing.B) {
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < treeSize; i++ {
						require.NoError(b, tree.Update([]byte(strconv.Itoa(i)), leaf))
					}
					require.NoError(b, tree.Commit())
				},
			)
			require.NoError(b, nodes.ClearAll())
		}
	}
	require.NoError(b, nodes.Stop())
}

func BenchmarkSMSTLeafSizes_Fill(b *testing.B) {
	treeSizes := []int{100000, 500000, 1000000, 5000000, 10000000}
	leafSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}
	nodes, err := smt.NewKVStore("")
	require.NoError(b, err)
	for _, treeSize := range treeSizes {
		for _, leafSize := range leafSizes {
			leaf := make([]byte, leafSize)
			for i := 0; i < 2; i++ {
				commit := i == 1
				name := "Fill"
				if commit {
					name = "Fill & Commit"
				}
				tree := smt.NewSparseMerkleSumTree(nodes, sha256.New(), smt.WithValueHasher(nil))
				b.ResetTimer()
				b.Run(
					fmt.Sprintf("%s [Leaf Size: %d bytes] (%d)", name, leafSize, treeSize),
					func(b *testing.B) {
						b.ResetTimer()
						b.ReportAllocs()
						for i := 0; i < treeSize; i++ {
							require.NoError(b, tree.Update([]byte(strconv.Itoa(i)), leaf, uint64(i)))
						}
						if commit {
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
