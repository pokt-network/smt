package smt

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomiseSumProof(proof SparseMerkleSumProof) SparseMerkleSumProof {
	sideNodes := make([][]byte, len(proof.SideNodes))
	for i := range sideNodes {
		sideNodes[i] = make([]byte, len(proof.SideNodes[i])-sumLength)
		rand.Read(sideNodes[i]) //nolint: errcheck
		sideNodes[i] = append(sideNodes[i], proof.SideNodes[i][len(proof.SideNodes[i])-sumLength:]...)
	}
	return SparseMerkleSumProof{
		SideNodes:             sideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
	}
}

// Check that a non-compact proof is equivalent to the proof returned when it is compacted and de-compacted.
func checkSumCompactEquivalence(t *testing.T, proof SparseMerkleSumProof, base *TreeSpec) {
	t.Helper()
	compactedProof, err := CompactSumProof(proof, base)
	if err != nil {
		t.Fatalf("failed to compact proof: %v", err)
	}
	decompactedProof, err := DecompactSumProof(compactedProof, base)
	if err != nil {
		t.Fatalf("failed to decompact proof: %v", err)
	}

	for i, sideNode := range proof.SideNodes {
		if !bytes.Equal(decompactedProof.SideNodes[i], sideNode) {
			t.Fatalf("de-compacted proof does not match original proof")
		}
	}

	if !bytes.Equal(proof.NonMembershipLeafData, decompactedProof.NonMembershipLeafData) {
		t.Fatalf("de-compacted proof does not match original proof")
	}
}
