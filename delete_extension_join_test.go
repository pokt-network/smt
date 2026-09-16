package smt

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// Deleting a key can leave an extension node joined onto its parent's path.
// The joined node is mutated in place (its pathBounds start moves), and the
// join now calls setDirty on it, like the symmetric join in the inner-node
// branch of delete.
//
// That call is defensive: both tests below pass without it. An extension
// node's child is always an inner node (smt.go builds it that way and
// extensionNode.split keeps it), and delete returns an extension node from an
// inner node only through the inner-node branch's absorb, which has already
// called setDirty on it, so the node this branch joins is already dirty. That
// is read from the code and measured, not proven: instrumented randomized
// deletes reached this branch 99 times and never with a clean node. The tests
// pin what the call protects, should that ever change: the root matches a
// trie built without the deleted key, the store holds the joined node, and the
// node carries no persisted flag, stale digest or compaction mark.
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
func TestDelete_JoinedExtensionIsMarkedDirty(t *testing.T) {
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
			" after delete: %x\n reference:    %x\n"+
			"the joined extension node kept the digest it had before its path was changed",
			trie.Root(), ref.Root())
	}

	// The node the join produced must have been written to the store under its
	// new digest. Re-importing from the root proves the store is complete.
	reimported := ImportSparseMerkleSumTrie(store, sha256.New(), trie.Root(),
		WithPathHasher(dummyPathHasher{32}), WithValueHasher(nil))
	for _, key := range [][]byte{pairA, pairB} {
		got, _, err := reimported.Get(key)
		if err != nil {
			t.Fatalf("re-importing from the post-delete root cannot read key %x: %v\n"+
				"commit skipped the joined extension node because it still claimed to be persisted",
				key, err)
		}
		if !bytes.Equal(got, valueFor(key)) {
			t.Fatalf("key %x: re-imported trie returned %d bytes, want %d",
				key, len(got), len(valueFor(key)))
		}
	}
}

// The compaction mark rides on the same setDirty: a joined extension node that
// kept it would claim its subtree is fully compacted, and a later pass would
// stop there. Defensive for the same reason as the test above.
func TestDelete_JoinedExtensionClearsCompactionMark(t *testing.T) {
	lone, pairA, pairB := joinShapedKeys()

	trie := newDummyPathSMST(simplemap.NewSimpleMap())
	for _, key := range [][]byte{lone, pairA, pairB} {
		requireNoError(t, trie.Update(key, valueFor(key), 1), "Update")
	}
	requireNoError(t, trie.Commit(), "Commit")
	trie.CompactPersistedLeaves()
	assertJoinShape(t, trie)

	requireNoError(t, trie.Delete(lone), "Delete")

	// The node returned upward by the join is the new root.
	joined, ok := trie.root.(*extensionNode)
	if !ok {
		t.Fatalf("after the delete the root is %T, want *extensionNode; "+
			"the join branch was not reached", trie.root)
	}
	if joined.compactedSubtree {
		t.Fatal("the joined extension node still claims its subtree is fully compacted, " +
			"so a later compaction pass would stop there even though the node changed")
	}
	if joined.persisted {
		t.Fatal("the joined extension node still claims to be persisted, so commit will " +
			"not re-write it under its new digest")
	}
	if joined.digest != nil {
		t.Fatal("the joined extension node kept the digest it had before its path changed")
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
