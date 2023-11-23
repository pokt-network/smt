package smt

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
	"github.com/stretchr/testify/require"
)

type (
	opCounts struct{ ops, inserts, updates, deletes int }
	bulkop   struct{ key, val []byte }
)

// Test all tree operations in bulk.
func TestBulkOperations(t *testing.T) {
	cases := []opCounts{
		// Test more inserts/updates than deletions.
		{200, 100, 100, 50},
		{1000, 100, 100, 50},
		// Test extreme deletions.
		{200, 100, 100, 500},
		{1000, 100, 100, 500},
	}
	for _, tc := range cases {
		bulkOperations(t, tc.ops, tc.inserts, tc.updates, tc.deletes)
	}
}

// Test all tree operations in bulk, with specified ratio probabilities of insert, update and delete.
func bulkOperations(t *testing.T, operations int, insert int, update int, delete int) {
	smn := simplemap.New()
	smv := simplemap.New()

	smt := NewSMTWithStorage(smn, smv, sha256.New())

	max := insert + update + delete
	var kv []bulkop

	r := rand.New(rand.NewSource(1))
	for i := 0; i < operations; i++ {
		n := rand.Intn(max)
		if n < insert { // Insert
			keyLen := 16 + r.Intn(32)
			key := make([]byte, keyLen)
			_, err := crand.Read(key)
			require.NoError(t, err)

			valLen := 1 + r.Intn(64)
			val := make([]byte, valLen)
			_, err = crand.Read(val)
			require.NoError(t, err)

			err = smt.Update(key, val)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			kv = append(kv, bulkop{key, val})
		} else if n > insert && n < insert+update { // Update
			if len(kv) == 0 {
				continue
			}
			ki := rand.Intn(len(kv))
			valLen := 1 + r.Intn(64)
			val := make([]byte, valLen)
			_, err := crand.Read(val)
			require.NoError(t, err)

			err = smt.Update(kv[ki].key, val)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			kv[ki].val = val
		} else { // Delete
			if len(kv) == 0 {
				continue
			}
			ki := r.Intn(len(kv))

			err := smt.Delete(kv[ki].key)
			if err != nil && err != ErrKeyNotPresent {
				t.Fatalf("error: %v", err)
			}
			kv[ki].val = defaultValue
		}
	}

	bulkCheckAll(t, smt, kv)
	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

func bulkCheckAll(t *testing.T, smt *SMTWithStorage, kv []bulkop) {
	for ki := range kv {
		k, v := kv[ki].key, kv[ki].val

		value, err := smt.GetValue([]byte(k))
		if err != nil {
			t.Errorf("error: %v", err)
		}
		if !bytes.Equal([]byte(v), value) {
			t.Errorf("Incorrect value (i=%d)", ki)
		}

		// Generate and verify a Merkle proof for this key.
		proof, err := smt.Prove([]byte(k))
		if err != nil {
			t.Errorf("error: %v", err)
		}
		valid, err := VerifyProof(proof, smt.Root(), []byte(k), []byte(v), smt.Spec())
		if !valid || err != nil {
			t.Fatalf("Merkle proof failed to verify (i=%d): %v", ki, []byte(k))
		}
		compactProof, err := ProveCompact([]byte(k), smt)
		if err != nil {
			t.Errorf("error: %v", err)
		}
		valid, err = VerifyCompactProof(compactProof, smt.Root(), []byte(k), []byte(v), smt.Spec())
		if !valid || err != nil {
			t.Fatalf("Compact Merkle proof failed to verify (i=%d): %v", ki, []byte(k))
		}

		if v == nil {
			continue
		}

		// Check that the key is at the correct height in the tree.
		largestCommonPrefix := 0
		for ki2 := range kv {
			k2, v2 := kv[ki2].key, kv[ki2].val
			if v2 == nil {
				continue
			}

			ph := smt.Spec().ph
			commonPrefix := countCommonPrefixBits(ph.Path([]byte(k)), ph.Path([]byte(k2)), 0)
			if commonPrefix != smt.Spec().depth() && commonPrefix > largestCommonPrefix {
				largestCommonPrefix = commonPrefix
			}
		}
		if len(proof.SideNodes) != largestCommonPrefix+1 &&
			(len(proof.SideNodes) != 0 && largestCommonPrefix != 0) {
			t.Errorf("leaf is at unexpected height (ki=%d)", ki)
		}
	}
}
