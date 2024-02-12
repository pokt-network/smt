package smt

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// TestExampleSMT is a test that aims to act as an example of how to use the SMST.
func TestExampleSMST(t *testing.T) {
	dataMap := make(map[string]string)

	// Initialize a new in-memory key-value store to store the nodes of the trie.
	// NB: The trie only stores hashed values and not raw value data.
	nodeStore := simplemap.NewSimpleMap()

	// Initialize the smst
	smst := NewSparseMerkleSumTrie(nodeStore, sha256.New())

	// Update trie with keys, values and their sums
	err := smst.Update([]byte("foo"), []byte("oof"), 10)
	require.NoError(t, err)
	dataMap["foo"] = "oof"
	err = smst.Update([]byte("baz"), []byte("zab"), 7)
	require.NoError(t, err)
	dataMap["baz"] = "zab"
	err = smst.Update([]byte("bin"), []byte("nib"), 3)
	require.NoError(t, err)
	dataMap["bin"] = "nib"

	// Commit the changes to the nodeStore
	err = smst.Commit()
	require.NoError(t, err)

	// Calculate the total sum of the trie
	sum := smst.Sum()
	require.Equal(t, uint64(20), sum)

	// Generate a Merkle proof for "foo"
	proof1, err := smst.Prove([]byte("foo"))
	require.NoError(t, err)
	proof2, err := smst.Prove([]byte("baz"))
	require.NoError(t, err)
	proof3, err := smst.Prove([]byte("bin"))
	require.NoError(t, err)

	// We also need the current trie root for the proof
	root := smst.Root()

	// Verify the Merkle proof for "foo"="oof" where "foo" has a sum of 10
	valid_true1, err := VerifySumProof(proof1, root, []byte("foo"), []byte("oof"), 10, smst.Spec())
	require.NoError(t, err)
	require.True(t, valid_true1)

	// Verify the Merkle proof for "baz"="zab" where "baz" has a sum of 7
	valid_true2, err := VerifySumProof(proof2, root, []byte("baz"), []byte("zab"), 7, smst.Spec())
	require.NoError(t, err)
	require.True(t, valid_true2)

	// Verify the Merkle proof for "bin"="nib" where "bin" has a sum of 3
	valid_true3, err := VerifySumProof(proof3, root, []byte("bin"), []byte("nib"), 3, smst.Spec())
	require.NoError(t, err)
	require.True(t, valid_true3)

	// Fail to verify the Merkle proof for "foo"="oof" where "foo" has a sum of 11
	valid_false1, err := VerifySumProof(proof1, root, []byte("foo"), []byte("oof"), 11, smst.Spec())
	require.NoError(t, err)
	require.False(t, valid_false1)

	exportToCSV(t, smst, dataMap, nodeStore)
}

func exportToCSV(
	t *testing.T,
	smst SparseMerkleSumTrie,
	innerMap map[string]string,
	nodeStore kvstore.MapStore,
) {
	t.Helper()
	/*
		rootHash := smst.Root()
		rootNode, err := nodeStore.Get(rootHash)
		require.NoError(t, err)
		leftData, rightData := smst.Spec().th.parseSumInnerNode(rootNode)
		leftChild, err := nodeStore.Get(leftData)
		require.NoError(t, err)
		rightChild, err := nodeStore.Get(rightData)
		require.NoError(t, err)
		fmt.Println("Prefix", "isExt", "isLeaf", "isInner")
		// false false true
		fmt.Println("root", isExtNode(rootNode), isLeafNode(rootNode), isInnerNode(rootNode), rootNode)
		fmt.Println()
		// false false false
		fmt.Println("left", isExtNode(leftChild), isLeafNode(leftChild), isInnerNode(leftChild), leftChild)
		fmt.Println()
		// false false false
		fmt.Println("right", isExtNode(rightChild), isLeafNode(rightChild), isInnerNode(rightChild), rightChild)
		fmt.Println()
	*/

	/*
		for key, value := range innerMap {
			v, s, err := smst.Get([]byte(key))
			require.NoError(t, err)
			fmt.Println(v, s, []byte(key))
			fmt.Println(value)
			fmt.Println("")
			fmt.Println("")
		}
	*/

	helper(t, smst, nodeStore, smst.Root())
}

func helper(t *testing.T, smst SparseMerkleSumTrie, nodeStore kvstore.MapStore, nodeDigest []byte) {
	t.Helper()

	node, err := nodeStore.Get(nodeDigest)
	require.NoError(t, err)

	fmt.Println()
	if isExtNode(node) {
		pathBounds, path, childData, sum := smst.Spec().parseSumExtNode(node)
		fmt.Println("ext node sum", sum)
		fmt.Println(pathBounds, path)
		helper(t, smst, nodeStore, childData)
		return
	} else if isLeafNode(node) {
		path, value := smst.Spec().parseLeafNode(node)
		fmt.Println("leaf node sum", 0)
		fmt.Println(path, value)
	} else if isInnerNode(node) {
		leftData, rightData, sum := smst.Spec().th.parseSumInnerNode(node)
		fmt.Println("inner node sum", sum)
		helper(t, smst, nodeStore, leftData)
		helper(t, smst, nodeStore, rightData)
	}

	// v, s, err := smst.Get([]byte(key))
	// require.NoError(t, err)
	// require.Equal(t, []byte(value), v)
	// require.Equal(t, sum, s)
}
