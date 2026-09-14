package smt

import (
	"bytes"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// Confirms the spec the C1-C5 tests run against is the one poktroll uses for
// the claim/proof trie. Asserted rather than described, so a change to
// NewSparseMerkleSumTrie cannot silently move the tests off the real spec.
//
// Ref: github.com/pokt-network/poktroll@v0.1.35 pkg/crypto/protocol/hasher.go
//
//	NewTrieHasher    = sha256.New
//	SMTValueHasher() = smt.WithValueHasher(nil)
func TestCompact_SpecMatchesPoktroll(t *testing.T) {
	trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())

	spec := trie.Spec()
	if !spec.sumTrie {
		t.Fatal("spec is not a sum trie")
	}
	if spec.vh != nil {
		t.Fatalf("spec has a value hasher (%T); poktroll uses WithValueHasher(nil), "+
			"which is what makes the leaf carry the raw value", spec.vh)
	}
	if got := spec.th.hashSize(); got != 32 {
		t.Fatalf("trie hasher size is %d, want 32 (sha256)", got)
	}
	// The inner SMT the SMST wraps must carry the nil value hasher too,
	// otherwise the value would be hashed on the way in.
	if trie.SMT.vh != nil {
		t.Fatalf("inner SMT has a value hasher (%T), want nil", trie.SMT.vh)
	}
}

// C3: after compaction, Get must return the value that was inserted, resolved
// back from the node store.
func TestCompact_C3_GetAfterCompaction(t *testing.T) {
	ops := generateOps(31, 1000)

	trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	latest := map[string]trieOp{}
	for _, o := range ops {
		requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
		requireNoError(t, trie.Commit(), "Commit")
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("CompactPersistedLeaves: %v", err)
		}
		latest[string(o.key)] = o
	}

	// Control: without this the test would pass on a trie that was never
	// compacted, which is the state it is supposed to be reading through.
	held, total := residentLeavesHoldingValues(trie.root)
	if total == 0 {
		t.Fatal("no resident leaves; the test reads nothing")
	}
	if held != 0 {
		t.Fatalf("%d of %d leaves still hold their value; the reads below would "+
			"not exercise the store fallback", held, total)
	}

	for key, o := range latest {
		gotValue, gotWeight, err := trie.Get([]byte(key))
		requireNoError(t, err, "Get")

		if !bytes.Equal(gotValue, o.value) {
			t.Fatalf("key %x: Get returned %d bytes, want the %d inserted bytes",
				key, len(gotValue), len(o.value))
		}
		if gotWeight != o.weight {
			t.Fatalf("key %x: Get returned weight %d, want %d", key, gotWeight, o.weight)
		}
	}

	// A key that was never inserted still reports absent.
	absent, weight, err := trie.Get([]byte("this key was never inserted"))
	requireNoError(t, err, "Get(absent)")
	if len(absent) != 0 || weight != 0 {
		t.Fatalf("absent key returned value of %d bytes and weight %d, want empty",
			len(absent), weight)
	}

	t.Logf("read back %d keys through the store fallback", len(latest))
}

// C3 (cont.): resolving a value must leave the trie in a state where a further
// compaction pass drops it again, and where the root never moves.
func TestCompact_C3_ResolveThenRecompact(t *testing.T) {
	trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
	ops := generateOps(37, 200)
	for _, o := range ops {
		requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
	}
	requireNoError(t, trie.Commit(), "Commit")

	rootBefore := append([]byte(nil), trie.Root()...)

	first, err := trie.CompactPersistedLeaves()
	requireNoError(t, err, "first compaction")
	if first == 0 {
		t.Fatal("first compaction pass compacted nothing")
	}

	// Read every key, which hydrates every leaf.
	for _, o := range ops {
		if _, _, err := trie.Get(o.key); err != nil {
			t.Fatalf("Get: %v", err)
		}
	}
	held, total := residentLeavesHoldingValues(trie.root)
	if held != total {
		t.Fatalf("after reading every key only %d of %d leaves hold a value", held, total)
	}

	second, err := trie.CompactPersistedLeaves()
	requireNoError(t, err, "second compaction")
	if second != first {
		t.Fatalf("second compaction dropped %d leaves, want the same %d as the first", second, first)
	}

	if !bytes.Equal(rootBefore, trie.Root()) {
		t.Fatalf("root moved across compact/resolve/compact:\n before: %x\n after:  %x",
			rootBefore, trie.Root())
	}
}
