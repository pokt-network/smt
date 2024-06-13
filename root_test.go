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

func TestMerkleRoot_TrieTypes(t *testing.T) {
	tests := []struct {
		desc          string
		sumTree       bool
		hasher        hash.Hash
		expectedPanic string
	}{
		{
			desc:          "successfully: gets sum of sha256 hasher SMST",
			sumTree:       true,
			hasher:        sha256.New(),
			expectedPanic: "",
		},
		{
			desc:          "successfully: gets sum of sha512 hasher SMST",
			sumTree:       true,
			hasher:        sha512.New(),
			expectedPanic: "",
		},
		{
			desc:          "failure: panics for sha256 hasher SMT",
			sumTree:       false,
			hasher:        sha256.New(),
			expectedPanic: "roo#sum: not a merkle sum trie",
		},
		{
			desc:          "failure: panics for sha512 hasher SMT",
			sumTree:       false,
			hasher:        sha512.New(),
			expectedPanic: "roo#sum: not a merkle sum trie",
		},
	}

	nodeStore := simplemap.NewSimpleMap()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, nodeStore.ClearAll())
			})
			if tt.sumTree {
				trie := smt.NewSparseMerkleSumTrie(nodeStore, tt.hasher)
				for i := uint64(0); i < 10; i++ {
					require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)), i))
				}
				require.NotNil(t, trie.Sum())
				require.EqualValues(t, 45, trie.Sum())
				require.EqualValues(t, 10, trie.Count())

				return
			}
			trie := smt.NewSparseMerkleTrie(nodeStore, tt.hasher)
			for i := 0; i < 10; i++ {
				require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i))))
			}
			if panicStr := recover(); panicStr != nil {
				require.Equal(t, tt.expectedPanic, panicStr)
			}
		})
	}
}
