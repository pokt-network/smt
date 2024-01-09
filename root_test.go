package smt_test

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func TestMerkleRoot_SumTrie(t *testing.T) {
	nodeStore := simplemap.NewSimpleMap()
	trie := smt.NewSparseMerkleSumTrie(nodeStore, sha256.New())
	for i := uint64(0); i < 10; i++ {
		require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)), i))
	}
	root := trie.Root()
	require.Equal(t, root.Sum(true), getSumBzHelper(t, root))
}

func TestMerkleRoot_Trie(t *testing.T) {
	nodeStore := simplemap.NewSimpleMap()
	trie := smt.NewSparseMerkleTrie(nodeStore, sha256.New())
	for i := 0; i < 10; i++ {
		require.NoError(t, trie.Update([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i))))
	}
	root := trie.Root()
	require.Equal(t, root.Sum(false), uint64(0))
}

func getSumBzHelper(t *testing.T, r []byte) uint64 {
	var sumbz [8]byte                            // Using sha256
	copy(sumbz[:], []byte(r)[len([]byte(r))-8:]) // Using sha256 so - 8 bytes
	return binary.BigEndian.Uint64(sumbz[:])
}
