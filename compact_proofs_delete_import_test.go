package smt

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// These tests cover the compaction paths the C1-C5 suite does not reach: Prove
// (non-membership and sibling data), Delete, the plain trie without sums, and a
// trie imported from a root whose nodes start out lazy. Each asserts a control
// that the path under test was actually taken, so a green result cannot come
// from never reaching it.

// distinctKeysInOrder returns each key once, in the order it first appears, so
// iteration over the keys is deterministic.
func distinctKeysInOrder(ops []trieOp) [][]byte {
	seen := map[string]struct{}{}
	keys := make([][]byte, 0, len(ops))
	for _, o := range ops {
		if _, ok := seen[string(o.key)]; ok {
			continue
		}
		seen[string(o.key)] = struct{}{}
		keys = append(keys, o.key)
	}
	return keys
}

// latestByKey returns the last operation applied to each key.
func latestByKey(ops []trieOp) map[string]trieOp {
	latest := make(map[string]trieOp, len(ops))
	for _, o := range ops {
		latest[string(o.key)] = o
	}
	return latest
}

// A non-membership proof carries the encoding of the unrelated leaf found on
// the proven path. When that leaf is compacted, Prove has to restore its value
// before encoding it, or the proof carries prefix+path with no value.
func TestCompact_ProveNonMembershipOverCompactedLeaf(t *testing.T) {
	compacted := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	plain := newPoktrollSpecSMST(simplemap.NewSimpleMap())

	for _, o := range generateOps(71, 300) {
		requireNoError(t, compacted.Update(o.key, o.value, o.weight), "compacted.Update")
		requireNoError(t, plain.Update(o.key, o.value, o.weight), "plain.Update")
	}
	requireNoError(t, compacted.Commit(), "compacted.Commit")
	requireNoError(t, plain.Commit(), "plain.Commit")
	compacted.CompactPersistedLeaves()

	held, total := residentLeavesHoldingValues(compacted.root)
	if total == 0 || held != 0 {
		t.Fatalf("CONTROL: want every resident leaf compacted, got %d of %d holding a value", held, total)
	}

	root := compacted.Root()
	if !bytes.Equal(root, plain.Root()) {
		t.Fatalf("roots diverged before proving")
	}
	spec := compacted.Spec()

	rnd := rand.New(rand.NewSource(73))
	withLeafData := 0
	for i := 0; i < 1000 && withLeafData < 50; i++ {
		absent := make([]byte, 32)
		rnd.Read(absent) //nolint:errcheck // math/rand Read never fails

		proofP, err := plain.Prove(absent)
		requireNoError(t, err, "plain.Prove")
		// Decided on the plain trie, so the choice of which proofs count cannot
		// depend on the behaviour under test.
		if proofP.NonMembershipLeafData == nil {
			continue
		}
		withLeafData++

		proofC, err := compacted.Prove(absent)
		requireNoError(t, err, "compacted.Prove")

		bzC, err := proofC.Marshal()
		requireNoError(t, err, "compacted proof Marshal")
		bzP, err := proofP.Marshal()
		requireNoError(t, err, "plain proof Marshal")
		if !bytes.Equal(bzC, bzP) {
			t.Fatalf("non-membership proof for %x diverged from the plain trie's: the unrelated "+
				"leaf was encoded without restoring its compacted value", absent)
		}

		valid, err := VerifySumProof(proofC, root, absent, defaultEmptyValue, 0, 0, spec)
		requireNoError(t, err, "VerifySumProof")
		if !valid {
			t.Fatalf("non-membership proof for %x from the compacted trie did not verify", absent)
		}

		// Prove restored one leaf; compact again so the next proof also starts
		// from a fully compacted trie.
		compacted.CompactPersistedLeaves()
	}

	if withLeafData == 0 {
		t.Fatal("CONTROL: no probed key produced a non-membership proof with leaf data; " +
			"the branch under test was never reached")
	}
	t.Logf("verified %d non-membership proofs carrying a compacted leaf", withLeafData)
}

// Prove re-serialises the proven leaf's sibling. When that sibling is a
// compacted leaf, Prove has to restore its value first; encoding it as it is
// trips the compacted-leaf guard.
func TestCompact_ProveWithCompactedSibling(t *testing.T) {
	compacted := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	plain := newPoktrollSpecSMST(simplemap.NewSimpleMap())

	keys := make([][]byte, 0, 64)
	for i := 0; i < 64; i++ {
		key := []byte{byte(i)}
		requireNoError(t, compacted.Update(key, valueFor(key), uint64(i+1)), "compacted.Update")
		requireNoError(t, plain.Update(key, valueFor(key), uint64(i+1)), "plain.Update")
		keys = append(keys, key)
	}
	requireNoError(t, compacted.Commit(), "compacted.Commit")
	requireNoError(t, plain.Commit(), "plain.Commit")
	compacted.CompactPersistedLeaves()

	root := compacted.Root()
	spec := compacted.Spec()
	leafSiblings := 0
	for i, key := range keys {
		proofP, err := plain.Prove(key)
		requireNoError(t, err, "plain.Prove")
		if proofP.SiblingData != nil && isLeafNode(proofP.SiblingData) {
			leafSiblings++
		}

		proofC, err := compacted.Prove(key)
		requireNoError(t, err, "compacted.Prove")

		bzC, err := proofC.Marshal()
		requireNoError(t, err, "compacted proof Marshal")
		bzP, err := proofP.Marshal()
		requireNoError(t, err, "plain proof Marshal")
		if !bytes.Equal(bzC, bzP) {
			t.Fatalf("proof for key %x diverged from the plain trie's", key)
		}

		valid, err := VerifySumProof(proofC, root, key, valueFor(key), uint64(i+1), 1, spec)
		requireNoError(t, err, "VerifySumProof")
		if !valid {
			t.Fatalf("proof for key %x from the compacted trie did not verify", key)
		}

		compacted.CompactPersistedLeaves()
	}

	if leafSiblings == 0 {
		t.Fatal("CONTROL: no proof had a leaf as its sibling; the branch under test was never reached")
	}
	t.Logf("%d of %d proofs re-serialised a compacted leaf sibling", leafSiblings, len(keys))
}

// Delete on a compacted trie must leave the same root and the same store as on
// a plain one, and must not leave any resident leaf holding a value.
func TestCompact_DeleteOnCompactedTrie(t *testing.T) {
	// A deleted leaf whose sibling is an extension node makes delete absorb that
	// extension into the parent's path. The absorbed node is mutated in place,
	// so it must be marked dirty; built so its parent is an inner node, because
	// under an extension parent the join that follows marks it dirty anyway.
	//
	// Paths are the keys themselves (dummyPathHasher):
	//
	//	bit 0      splits r from the rest            -> root inner node
	//	bit 1      splits lone from the pair         -> inner node at depth 1
	//	bits 2-253 shared by the pair                -> extension under it
	t.Run("absorb under an inner node", func(t *testing.T) {
		r, lone, pairA, pairB := absorbShapedKeys()

		trie := newDummyPathSMST(simplemap.NewSimpleMap())
		for _, key := range [][]byte{lone, r, pairA, pairB} {
			requireNoError(t, trie.Update(key, valueFor(key), 1), "Update")
		}
		requireNoError(t, trie.Commit(), "Commit")
		trie.CompactPersistedLeaves()
		assertAbsorbShape(t, trie)

		requireNoError(t, trie.Delete(lone), "Delete")
		requireNoError(t, trie.Commit(), "Commit after delete")
		trie.CompactPersistedLeaves()

		ref := newDummyPathSMST(simplemap.NewSimpleMap())
		for _, key := range [][]byte{r, pairA, pairB} {
			requireNoError(t, ref.Update(key, valueFor(key), 1), "ref.Update")
		}
		requireNoError(t, ref.Commit(), "ref.Commit")

		if !bytes.Equal(trie.Root(), ref.Root()) {
			t.Fatalf("root after deleting the lone key does not match a trie built without it: "+
				"the absorbed extension node kept the digest it had before its bounds moved\n"+
				" after delete: %x\n reference:    %x", trie.Root(), ref.Root())
		}
		if held, total := residentLeavesHoldingValues(trie.root); held != 0 {
			t.Fatalf("%d of %d resident leaves hold a value after compaction", held, total)
		}
		for _, key := range [][]byte{r, pairA, pairB} {
			value, _, err := trie.Get(key)
			requireNoError(t, err, "Get")
			if !bytes.Equal(value, valueFor(key)) {
				t.Fatalf("key %x: Get returned the wrong value after delete", key)
			}
		}
	})

	for _, seed := range []int64{3, 5, 8} {
		t.Run(fmt.Sprintf("mixed updates and deletes/seed=%d", seed), func(t *testing.T) {
			mapA, mapB := map[string][]byte{}, map[string][]byte{}
			trieA := newPoktrollSpecSMST(simplemap.NewSimpleMapWithMap(mapA)) // compacted
			trieB := newPoktrollSpecSMST(simplemap.NewSimpleMapWithMap(mapB)) // plain

			rnd := rand.New(rand.NewSource(seed))
			var alive [][]byte
			latest := map[string]trieOp{}
			updates, deletes := 0, 0

			for step := 0; step < 600; step++ {
				op := "update"
				if len(alive) > 0 && rnd.Intn(3) == 0 {
					op = "delete"
					idx := rnd.Intn(len(alive))
					key := alive[idx]
					alive = append(alive[:idx], alive[idx+1:]...)
					requireNoError(t, trieA.Delete(key), "A.Delete")
					requireNoError(t, trieB.Delete(key), "B.Delete")
					delete(latest, string(key))
					deletes++
				} else {
					key := make([]byte, 32)
					rnd.Read(key) //nolint:errcheck // math/rand Read never fails
					value := make([]byte, 900)
					rnd.Read(value) //nolint:errcheck // math/rand Read never fails
					weight := uint64(rnd.Intn(1000) + 1)
					requireNoError(t, trieA.Update(key, value, weight), "A.Update")
					requireNoError(t, trieB.Update(key, value, weight), "B.Update")
					alive = append(alive, key)
					latest[string(key)] = trieOp{key: key, value: value, weight: weight}
					updates++
				}

				requireNoError(t, trieA.Commit(), "A.Commit")
				requireNoError(t, trieB.Commit(), "B.Commit")
				trieA.CompactPersistedLeaves()
				if !bytes.Equal(trieA.Root(), trieB.Root()) {
					t.Fatalf("step %d (%s): root diverged between the compacted and the plain trie", step, op)
				}
			}

			if updates == 0 || deletes == 0 {
				t.Fatalf("CONTROL: want both updates and deletes, got %d and %d", updates, deletes)
			}
			if held, total := residentLeavesHoldingValues(trieA.root); held != 0 {
				t.Fatalf("%d of %d resident leaves still hold a value after compaction: a pass "+
					"stopped at a subtree still marked compacted after it changed", held, total)
			}
			if len(mapA) != len(mapB) {
				t.Fatalf("store size diverged: compacted=%d plain=%d", len(mapA), len(mapB))
			}
			for k, vB := range mapB {
				if vA, ok := mapA[k]; !ok || !bytes.Equal(vA, vB) {
					t.Fatalf("store entry %x diverged between the compacted and the plain trie", k)
				}
			}
			for _, o := range latest {
				value, weight, err := trieA.Get(o.key)
				requireNoError(t, err, "Get")
				if !bytes.Equal(value, o.value) || weight != o.weight {
					t.Fatalf("key %x: Get returned the wrong value or weight after deletes", o.key)
				}
			}
			t.Logf("%d updates, %d deletes, roots equal at every step", updates, deletes)
		})
	}
}

// absorbShapedKeys returns four 32-byte keys whose paths build
// root inner -> {inner -> {lone, extension -> {pairA, pairB}}, r}.
func absorbShapedKeys() (r, lone, pairA, pairB []byte) {
	r = make([]byte, 32)
	r[0] = 0x80 // bit 0 is 1
	r[31] = 0x01

	lone = make([]byte, 32)
	lone[31] = 0x01 // bits 0 and 1 are 0

	pairA = make([]byte, 32)
	pairA[0] = 0x40 // bit 0 is 0, bit 1 is 1
	pairA[31] = 0x01

	pairB = make([]byte, 32)
	pairB[0] = 0x40
	pairB[31] = 0x02 // diverges from pairA only inside the last byte

	return r, lone, pairA, pairB
}

// assertAbsorbShape fails unless deleting lone would absorb an extension node
// into an inner node's child slot.
func assertAbsorbShape(t *testing.T, trie *SMST) {
	t.Helper()

	root, ok := trie.root.(*innerNode)
	if !ok {
		t.Fatalf("CONTROL: root is %T, want *innerNode", trie.root)
	}
	left, ok := root.leftChild.(*innerNode)
	if !ok {
		t.Fatalf("CONTROL: root's left child is %T, want *innerNode", root.leftChild)
	}
	if _, ok := left.leftChild.(*leafNode); !ok {
		t.Fatalf("CONTROL: the depth-1 inner node's left child is %T, want *leafNode", left.leftChild)
	}
	if _, ok := left.rightChild.(*extensionNode); !ok {
		t.Fatalf("CONTROL: the depth-1 inner node's right child is %T, want *extensionNode; "+
			"deleting lone would not absorb an extension", left.rightChild)
	}
}

// The plain trie has no sums, so it digests and encodes through a different
// pair of functions. Its proofs must match an uncompacted plain trie byte for
// byte, including when a compacted leaf is the sibling ProveClosest encodes.
func TestCompact_PlainTrie(t *testing.T) {
	compacted := NewSparseMerkleTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
	plain := NewSparseMerkleTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))

	ops := generateOps(83, 400)
	for i, o := range ops {
		requireNoError(t, compacted.Update(o.key, o.value), "compacted.Update")
		requireNoError(t, plain.Update(o.key, o.value), "plain.Update")
		requireNoError(t, compacted.Commit(), "compacted.Commit")
		requireNoError(t, plain.Commit(), "plain.Commit")
		compacted.CompactPersistedLeaves()
		if !bytes.Equal(compacted.Root(), plain.Root()) {
			t.Fatalf("op %d: root diverged between the compacted and the plain trie", i)
		}
	}

	held, total := residentLeavesHoldingValues(compacted.root)
	if total == 0 || held != 0 {
		t.Fatalf("CONTROL: want every resident leaf compacted, got %d of %d holding a value", held, total)
	}

	root := compacted.Root()
	spec := compacted.Spec()
	latest := latestByKey(ops)
	leafSiblings := 0
	for _, key := range distinctKeysInOrder(ops) {
		path := spec.ph.Path(key)

		proofP, err := plain.ProveClosest(path)
		requireNoError(t, err, "plain.ProveClosest")
		if proofP.ClosestProof.SiblingData != nil && isLeafNode(proofP.ClosestProof.SiblingData) {
			leafSiblings++
		}

		proofC, err := compacted.ProveClosest(path)
		requireNoError(t, err, "compacted.ProveClosest")

		bzC, err := proofC.Marshal()
		requireNoError(t, err, "compacted proof Marshal")
		bzP, err := proofP.Marshal()
		requireNoError(t, err, "plain proof Marshal")
		if !bytes.Equal(bzC, bzP) {
			t.Fatalf("closest proof for key %x diverged from the plain trie's", key)
		}
		valid, err := VerifyClosestProof(proofC, root, spec)
		requireNoError(t, err, "VerifyClosestProof")
		if !valid {
			t.Fatalf("closest proof for key %x from the compacted trie did not verify", key)
		}

		compacted.CompactPersistedLeaves()
		value, err := compacted.Get(key)
		requireNoError(t, err, "Get")
		if !bytes.Equal(value, latest[string(key)].value) {
			t.Fatalf("key %x: Get returned the wrong value", key)
		}
		compacted.CompactPersistedLeaves()
	}

	if leafSiblings == 0 {
		t.Fatal("CONTROL: no closest proof had a leaf as its sibling; the branch under test was never reached")
	}
	t.Logf("%d closest proofs re-serialised a compacted leaf sibling", leafSiblings)
}

// A trie imported from a root starts with every node lazy, and reads pull
// subtrees in from the store without going through setDirty. A compaction pass
// that trusted the marks it left on resident ancestors before those reads would
// stop above the newly resident leaves and leave their values in memory.
func TestCompact_ImportedTrieWithLazyNodes(t *testing.T) {
	t.Run("lazy nodes resolved under a marked ancestor", func(t *testing.T) {
		store := simplemap.NewSimpleMap()
		src := newPoktrollSpecSMST(store)
		ops := generateOps(89, 400)
		for _, o := range ops {
			requireNoError(t, src.Update(o.key, o.value, o.weight), "src.Update")
		}
		requireNoError(t, src.Commit(), "src.Commit")
		root := append([]byte(nil), src.Root()...)

		imported := ImportSparseMerkleSumTrie(store, sha256.New(), root, WithValueHasher(nil))
		keys := distinctKeysInOrder(ops)
		latest := latestByKey(ops)

		// Resolve one path and compact it. Ancestors on that path get marked,
		// with their other children still lazy.
		if _, _, err := imported.Get(keys[0]); err != nil {
			t.Fatalf("Get: %v", err)
		}
		imported.CompactPersistedLeaves()
		held, total := residentLeavesHoldingValues(imported.root)
		if total == 0 || held != 0 {
			t.Fatalf("CONTROL: after resolving one path want its leaf resident and compacted, got %d of %d holding a value",
				held, total)
		}

		// Resolve more paths through those marked ancestors.
		for _, key := range keys[1:40] {
			value, _, err := imported.Get(key)
			requireNoError(t, err, "Get")
			if !bytes.Equal(value, latest[string(key)].value) {
				t.Fatalf("key %x: Get on the imported trie returned the wrong value", key)
			}
		}
		held, total = residentLeavesHoldingValues(imported.root)
		if held == 0 {
			t.Fatalf("CONTROL: the reads pulled no leaf values into memory (%d resident leaves); nothing to compact", total)
		}

		imported.CompactPersistedLeaves()
		if held, total := residentLeavesHoldingValues(imported.root); held != 0 {
			t.Fatalf("%d of %d leaves resolved from the store still hold a value after compaction: "+
				"the pass stopped at an ancestor marked compacted before those leaves were resolved into it",
				held, total)
		}
		if !bytes.Equal(imported.Root(), root) {
			t.Fatalf("the imported trie's root moved across reads and compaction")
		}
	})

	t.Run("restored leaf under a marked ancestor", func(t *testing.T) {
		trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
		ops := generateOps(97, 400)
		for _, o := range ops {
			requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
		}
		requireNoError(t, trie.Commit(), "Commit")
		trie.CompactPersistedLeaves()
		if held, total := residentLeavesHoldingValues(trie.root); total == 0 || held != 0 {
			t.Fatalf("CONTROL: want every resident leaf compacted, got %d of %d holding a value", held, total)
		}

		// Fully resident, so every value these reads bring back comes through
		// the compacted-leaf restore and nothing else.
		for _, key := range distinctKeysInOrder(ops)[:40] {
			if _, _, err := trie.Get(key); err != nil {
				t.Fatalf("Get: %v", err)
			}
		}
		if held, _ := residentLeavesHoldingValues(trie.root); held == 0 {
			t.Fatal("CONTROL: the reads restored no leaf value; nothing to compact")
		}

		trie.CompactPersistedLeaves()
		if held, total := residentLeavesHoldingValues(trie.root); held != 0 {
			t.Fatalf("%d of %d restored leaves still hold a value after compaction: the pass "+
				"stopped at an ancestor still marked compacted from before the restore", held, total)
		}
	})
}
