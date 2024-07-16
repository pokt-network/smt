package smt_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func TestMerkleSumRoot_SumAndCountSuccess(t *testing.T) {
	tests := []struct {
		desc   string
		hasher hash.Hash
	}{
		{
			desc:   "sha256 hasher",
			hasher: sha256.New(),
		},
		{
			desc:   "sha512 hasher",
			hasher: sha512.New(),
		},
	}

	nodeStore := simplemap.NewSimpleMap()
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, nodeStore.ClearAll())
			})
			trie := smt.NewSparseMerkleSumTrie(nodeStore, test.hasher)
			for i := uint64(0); i < 10; i++ {
				require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)), i))
			}

			sum, sumErr := trie.Sum()
			require.NoError(t, sumErr)

			count, countErr := trie.Count()
			require.NoError(t, countErr)

			require.EqualValues(t, uint64(45), sum)
			require.EqualValues(t, uint64(10), count)
		})
	}
}

func TestMekleRoot_SumAndCountError(t *testing.T) {
	tests := []struct {
		desc   string
		hasher hash.Hash
	}{
		{
			desc:   "sha256 hasher",
			hasher: sha256.New(),
		},
		{
			desc:   "sha512 hasher",
			hasher: sha512.New(),
		},
	}

	nodeStore := simplemap.NewSimpleMap()
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, nodeStore.ClearAll())
			})
			trie := smt.NewSparseMerkleSumTrie(nodeStore, test.hasher)
			for i := uint64(0); i < 10; i++ {
				require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)), i))
			}

			root := trie.Root()

			// Mangle the root bytes.
			root = root[:len(root)-1]

			sum, sumErr := root.Sum()
			require.Error(t, sumErr)
			require.Equal(t, uint64(0), sum)

			count, countErr := root.Count()
			require.Error(t, countErr)
			require.Equal(t, uint64(0), count)
		})
	}
}
