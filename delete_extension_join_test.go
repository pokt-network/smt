package smt

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// Deleting a key can leave an extension node joined onto its parent's path:
// the node is mutated in place, its pathBounds start moving to the parent's.
// This test pins the result against a trie built without the deleted key: the
// same root, and a store complete enough to re-import it. What it catches is
// the join itself (measured: dropping the pathBounds assignment fails it).
//
// It does not catch the setDirty the join also calls, and cannot: an extension
// node's child is always an inner node, and delete returns an extension node
// from an inner node only through the inner-node branch's absorb, which has
// already dirtied it, so the joined node is dirty either way. That is read from
// the code and measured, not proven: instrumented randomized deletes reached
// the join 977 times, never with a clean node. The call stays, symmetric with
// the absorb, as a guard should that ever change.
//
// Paths are taken straight from the keys via dummyPathHasher so the shape can
// be built deliberately rather than searched for:
//
//	bits 0..127   all zero, shared by every key  -> outer extension
//	bit  128      splits lone from the pair      -> inner node
//	bits 129..254 zero, shared by the pair       -> inner extension
//
// Deleting `lone` empties one side of the inner node, so its sibling — the
// inner extension — is returned upward and joined onto the outer extension's
// path. That join is the branch under test.
func TestDelete_JoinedExtensionMatchesReference(t *testing.T) {
	lone, pairA, pairB := joinShapedKeys()

	store := simplemap.NewSimpleMap()
	trie := newDummyPathSMST(store)

	for _, key := range [][]byte{lone, pairA, pairB} {
		requireNoError(t, trie.Update(key, valueFor(key), 1), "Update")
	}
	requireNoError(t, trie.Commit(), "Commit")

	// Control: the trie must have the shape the branch needs, or the delete
	// below exercises some other path and this test proves nothing.
	assertJoinShape(t, trie)

	requireNoError(t, trie.Delete(lone), "Delete")
	requireNoError(t, trie.Commit(), "Commit after delete")

	// Reference: the same two keys, inserted into a fresh trie.
	ref := newDummyPathSMST(simplemap.NewSimpleMap())
	for _, key := range [][]byte{pairA, pairB} {
		requireNoError(t, ref.Update(key, valueFor(key), 1), "ref.Update")
	}
	requireNoError(t, ref.Commit(), "ref.Commit")

	if !bytes.Equal(trie.Root(), ref.Root()) {
		t.Fatalf("root after deleting the lone key does not match a trie built without it:\n"+
			" after delete: %x\n reference:    %x",
			trie.Root(), ref.Root())
	}

	// The node the join produced must have been written to the store under its
	// new digest. Re-importing from the root proves the store is complete.
	reimported := ImportSparseMerkleSumTrie(store, sha256.New(), trie.Root(),
		WithPathHasher(dummyPathHasher{32}), WithValueHasher(nil))
	for _, key := range [][]byte{pairA, pairB} {
		got, _, err := reimported.Get(key)
		if err != nil {
			t.Fatalf("re-importing from the post-delete root cannot read key %x: %v", key, err)
		}
		if !bytes.Equal(got, valueFor(key)) {
			t.Fatalf("key %x: re-imported trie returned %d bytes, want %d",
				key, len(got), len(valueFor(key)))
		}
	}
}

// newDummyPathSMST builds a sum trie with poktroll's nil value hasher and a
// path hasher that returns the key unchanged, so paths are chosen, not hashed.
func newDummyPathSMST(nodes kvstore.MapStore) *SMST {
	return NewSparseMerkleSumTrie(nodes, sha256.New(),
		WithPathHasher(dummyPathHasher{32}), WithValueHasher(nil))
}

// joinShapedKeys returns three 32-byte keys whose paths produce
// outer-extension -> inner -> {leaf, inner-extension}.
func joinShapedKeys() (lone, pairA, pairB []byte) {
	lone = make([]byte, 32)
	lone[31] = 0x01 // bit 128 is 0: the lone side of the inner node

	pairA = make([]byte, 32)
	pairA[16] = 0x80 // bit 128 is 1
	pairA[31] = 0x01

	pairB = make([]byte, 32)
	pairB[16] = 0x80
	pairB[31] = 0x02 // diverges from pairA only inside the last byte

	return lone, pairA, pairB
}

func valueFor(key []byte) []byte {
	value := make([]byte, 900)
	copy(value, key)
	return value
}

// assertJoinShape fails unless the trie is the outer-extension -> inner ->
// {leaf, extension} shape the join branch needs.
func assertJoinShape(t *testing.T, trie *SMST) {
	t.Helper()

	outer, ok := trie.root.(*extensionNode)
	if !ok {
		t.Fatalf("root is %T, want *extensionNode", trie.root)
	}
	inner, ok := outer.child.(*innerNode)
	if !ok {
		t.Fatalf("the outer extension's child is %T, want *innerNode", outer.child)
	}
	if _, ok := inner.leftChild.(*leafNode); !ok {
		t.Fatalf("the inner node's left child is %T, want *leafNode", inner.leftChild)
	}
	if _, ok := inner.rightChild.(*extensionNode); !ok {
		t.Fatalf("the inner node's right child is %T, want *extensionNode; "+
			"deleting the lone key would not produce an extension join", inner.rightChild)
	}
}
