package smt_test

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// TestExampleSMT is a test that aims to act as an example of how to use the SMST.
func TestExampleSMT(t *testing.T) {
	// Initialize a new in-memory key-value store to store the nodes of the trie
	// (Note: the trie only stores hashed values, not raw value data)
	nodeStore := simplemap.NewSimpleMap()

	// Initialize the trie
	trie := smt.NewSparseMerkleTrie(nodeStore, sha256.New())

	// Update the key "foo" with the value "bar"
	_ = trie.Update([]byte("foo"), []byte("bar"))

	// Commit the changes to the node store
	_ = trie.Commit()

	// Generate a Merkle proof for "foo"
	proof, _ := trie.Prove([]byte("foo"))
	root := trie.Root() // We also need the current trie root for the proof

	// Verify the Merkle proof for "foo"="bar"
	valid, _ := smt.VerifyProof(proof, root, []byte("foo"), []byte("bar"), trie.Spec())
	require.True(t, valid)
	// Attempt to verify the Merkle proof for "foo"="baz"
	invalid, _ := smt.VerifyProof(proof, root, []byte("foo"), []byte("baz"), trie.Spec())
	require.False(t, invalid)
}
