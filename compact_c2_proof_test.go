package smt

import (
	"bytes"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// C2: a proof taken from a compacted trie must verify, survive the
// compact/decompact round trip, and carry back exactly the bytes that were
// inserted.
//
// The strongest form of "nothing observable changed" is used: the proof from
// the compacted trie is compared byte for byte against the proof the plain
// trie produces for the same path.
func TestCompact_C2_ProofFromCompactedTrie(t *testing.T) {
	ops := generateOps(23, 500)

	trieA := newPoktrollSpecSMST(simplemap.NewSimpleMap()) // compacted
	trieB := newPoktrollSpecSMST(simplemap.NewSimpleMap()) // plain

	latest := map[string]trieOp{}
	for _, o := range ops {
		requireNoError(t, trieA.Update(o.key, o.value, o.weight), "A.Update")
		requireNoError(t, trieB.Update(o.key, o.value, o.weight), "B.Update")
		requireNoError(t, trieA.Commit(), "A.Commit")
		requireNoError(t, trieB.Commit(), "B.Commit")
		if _, err := trieA.CompactPersistedLeaves(); err != nil {
			t.Fatalf("CompactPersistedLeaves: %v", err)
		}
		latest[string(o.key)] = o
	}

	rootA, rootB := trieA.Root(), trieB.Root()
	if !bytes.Equal(rootA, rootB) {
		t.Fatalf("roots diverged before proving: %x vs %x", rootA, rootB)
	}

	spec := trieA.Spec()
	checked := 0
	for _, o := range latest {
		path := spec.ph.Path(o.key)

		proofA, err := trieA.ProveClosest(path)
		requireNoError(t, err, "A.ProveClosest")
		proofB, err := trieB.ProveClosest(path)
		requireNoError(t, err, "B.ProveClosest")

		// Every field of the proof must match the plain trie's proof.
		bzA, err := proofA.Marshal()
		requireNoError(t, err, "A.proof.Marshal")
		bzB, err := proofB.Marshal()
		requireNoError(t, err, "B.proof.Marshal")
		if !bytes.Equal(bzA, bzB) {
			t.Fatalf("proof for key %x diverged between compacted and plain trie", o.key)
		}

		// The proof verifies against the root.
		valid, err := VerifyClosestProof(proofA, rootA, spec)
		requireNoError(t, err, "VerifyClosestProof")
		if !valid {
			t.Fatalf("proof for key %x did not verify against the compacted trie root", o.key)
		}

		// The value carried back is the one that was inserted. In a sum trie
		// the leaf value is [value][8B weight][8B count], so strip the 16
		// trailing metadata bytes before comparing.
		firstSumByteIdx, _ := getFirstMetaByteIdx(proofA.ClosestValueHash)
		gotValue := proofA.ClosestValueHash[:firstSumByteIdx]
		want := latest[string(o.key)].value
		if !bytes.Equal(gotValue, want) {
			t.Fatalf("key %x: proof value has %d bytes, want the %d inserted bytes",
				o.key, len(gotValue), len(want))
		}

		// Compact / decompact round trip.
		compacted, err := CompactClosestProof(proofA, spec)
		requireNoError(t, err, "CompactClosestProof")
		decompacted, err := DecompactClosestProof(compacted, spec)
		requireNoError(t, err, "DecompactClosestProof")

		bzRT, err := decompacted.Marshal()
		requireNoError(t, err, "decompacted.Marshal")
		if !bytes.Equal(bzA, bzRT) {
			t.Fatalf("key %x: proof changed across the compact/decompact round trip", o.key)
		}
		valid, err = VerifyClosestProof(decompacted, rootA, spec)
		requireNoError(t, err, "VerifyClosestProof(decompacted)")
		if !valid {
			t.Fatalf("key %x: round-tripped proof did not verify", o.key)
		}

		checked++
	}

	if checked == 0 {
		t.Fatal("no proofs were checked; the test proves nothing")
	}
	t.Logf("verified %d proofs taken from a compacted trie", checked)
}

// C2 (cont.): the sibling of the proven leaf is the one node the proof has to
// re-serialise from the live trie (SiblingData). When that sibling is itself a
// compacted leaf, encoding it without resolving the value first would produce a
// truncated preimage whose hash no longer matches SideNodes[0].
//
// A dense trie built from short sequential keys makes leaf siblings likely.
func TestCompact_C2_CompactedLeafAsSibling(t *testing.T) {
	trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())

	keys := make([][]byte, 0, 64)
	for i := 0; i < 64; i++ {
		key := []byte{byte(i)}
		value := bytes.Repeat([]byte{byte(i + 1)}, 900)
		requireNoError(t, trie.Update(key, value, uint64(i+1)), "Update")
		keys = append(keys, key)
	}
	requireNoError(t, trie.Commit(), "Commit")
	compacted, err := trie.CompactPersistedLeaves()
	requireNoError(t, err, "CompactPersistedLeaves")
	if compacted != len(keys) {
		t.Fatalf("compacted %d leaves, want %d", compacted, len(keys))
	}

	spec := trie.Spec()
	root := trie.Root()
	withSibling := 0
	for _, key := range keys {
		proof, err := trie.ProveClosest(spec.ph.Path(key))
		requireNoError(t, err, "ProveClosest")
		if proof.ClosestProof.SiblingData != nil {
			withSibling++
		}
		valid, err := VerifyClosestProof(proof, root, spec)
		requireNoError(t, err, "VerifyClosestProof")
		if !valid {
			t.Fatalf("key %x: proof did not verify", key)
		}
	}

	if withSibling == 0 {
		t.Fatal("no proof carried SiblingData; this test never exercised the sibling path")
	}
	t.Logf("%d of %d proofs carried SiblingData", withSibling, len(keys))
}
