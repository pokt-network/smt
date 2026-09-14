package smt

import (
	"testing"
)

// C5: what compaction costs at the node store.
//
// The claim the design rests on is that the hot path is untouched: inserting
// into a compacted trie never has to read a value back, because update and
// delete only ever look at a leaf's path and its cached digest. The reads move
// to the proof and Get paths, which run once per session rather than once per
// relay.
func TestCompact_C5_StoreAccessCost(t *testing.T) {
	const (
		warmupLeaves = 5000
		hotLeaves    = 1000
	)

	store := newCountingStore()
	trie := newPoktrollSpecSMST(store)

	keys := make([][]byte, 0, warmupLeaves)
	for _, o := range generateOps(53, warmupLeaves) {
		requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
		requireNoError(t, trie.Commit(), "Commit")
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("CompactPersistedLeaves: %v", err)
		}
		keys = append(keys, o.key)
	}

	// Control: the reads below only mean something on a trie that is actually
	// compacted.
	held, total := residentLeavesHoldingValues(trie.root)
	if total == 0 {
		t.Fatal("no resident leaves")
	}
	if held != 0 {
		t.Fatalf("%d of %d leaves still hold a value", held, total)
	}

	// --- the hot path: insert into an already-compacted trie ---
	store.reset()
	for _, o := range generateOps(59, hotLeaves) {
		requireNoError(t, trie.Update(o.key, o.value, o.weight), "hot Update")
		requireNoError(t, trie.Commit(), "hot Commit")
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("hot CompactPersistedLeaves: %v", err)
		}
	}
	hotGets, hotSets := store.gets, store.sets
	t.Logf("C5 hot path: %d Update+Commit+Compact -> %d Gets, %d Sets", hotLeaves, hotGets, hotSets)
	if hotGets != 0 {
		t.Fatalf("C5 FAIL: inserting into a compacted trie made %d store Gets, want 0", hotGets)
	}
	// Control: a run that wrote nothing would also report zero reads.
	if hotSets == 0 {
		t.Fatal("the hot path wrote nothing to the store; the zero-read result is vacuous")
	}

	// --- Get on a compacted leaf ---
	store.reset()
	if _, _, err := trie.Get(keys[0]); err != nil {
		t.Fatalf("Get: %v", err)
	}
	getsForOneGet := store.gets
	t.Logf("C5 Get on a compacted leaf: %d store Gets", getsForOneGet)
	if getsForOneGet == 0 {
		t.Fatal("Get on a compacted leaf read nothing from the store; it cannot have resolved a value")
	}

	// A second Get of the same key costs nothing: the leaf stays hydrated.
	store.reset()
	if _, _, err := trie.Get(keys[0]); err != nil {
		t.Fatalf("second Get: %v", err)
	}
	t.Logf("C5 second Get of the same key: %d store Gets", store.gets)
	if store.gets != 0 {
		t.Fatalf("re-reading a hydrated leaf cost %d Gets, want 0", store.gets)
	}

	// --- ProveClosest on a compacted trie ---
	if _, err := trie.CompactPersistedLeaves(); err != nil {
		t.Fatalf("re-compaction before proving: %v", err)
	}
	spec := trie.Spec()
	totalProofGets, proofs := 0, 0
	for _, key := range keys[:100] {
		store.reset()
		proof, err := trie.ProveClosest(spec.ph.Path(key))
		requireNoError(t, err, "ProveClosest")
		totalProofGets += store.gets
		proofs++

		valid, err := VerifyClosestProof(proof, trie.Root(), spec)
		requireNoError(t, err, "VerifyClosestProof")
		if !valid {
			t.Fatalf("key %x: proof did not verify", key)
		}
		if _, err := trie.CompactPersistedLeaves(); err != nil {
			t.Fatalf("re-compaction between proofs: %v", err)
		}
	}
	t.Logf("C5 ProveClosest on a compacted trie: %d Gets over %d proofs (%.2f per proof)",
		totalProofGets, proofs, float64(totalProofGets)/float64(proofs))
	if totalProofGets == 0 {
		t.Fatal("proofs read nothing from the store; they cannot have resolved any compacted value")
	}
}
