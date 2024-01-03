package smt_test

import (
	"crypto/sha256"
	"fmt"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
)

func ExampleSMST() {
	// Initialise a new in-memory key-value store to store the nodes of the trie
	// (Note: the trie only stores hashed values, not raw value data)
	nodeStore := kvstore.NewSimpleMap()

	// Initialise the trie
	trie := smt.NewSparseMerkleSumTrie(nodeStore, sha256.New())

	// Update trie with keys, values and their sums
	_ = trie.Update([]byte("foo"), []byte("oof"), 10)
	_ = trie.Update([]byte("baz"), []byte("zab"), 7)
	_ = trie.Update([]byte("bin"), []byte("nib"), 3)

	// Commit the changes to the nodeStore
	_ = trie.Commit()

	sum := trie.Sum()
	fmt.Println(sum == 20) // true

	// Generate a Merkle proof for "foo"
	proof, _ := trie.Prove([]byte("foo"))
	root := trie.Root() // We also need the current trie root for the proof
	// Verify the Merkle proof for "foo"="oof" where "foo" has a sum of 10
	valid_true1, _ := smt.VerifySumProof(proof, root, []byte("foo"), []byte("oof"), 10, trie.Spec())
	// Verify the Merkle proof for "baz"="zab" where "baz" has a sum of 7
	valid_true2, _ := smt.VerifySumProof(proof, root, []byte("baz"), []byte("zab"), 7, trie.Spec())
	// Verify the Merkle proof for "bin"="nib" where "bin" has a sum of 3
	valid_true3, _ := smt.VerifySumProof(proof, root, []byte("bin"), []byte("nib"), 3, trie.Spec())
	// Fail to verify the Merkle proof for "foo"="oof" where "foo" has a sum of 11
	valid_false1, _ := smt.VerifySumProof(proof, root, []byte("foo"), []byte("oof"), 11, trie.Spec())
	fmt.Println(valid_true1, valid_true2, valid_true3, valid_false1)
	// Output: true true true false
}
