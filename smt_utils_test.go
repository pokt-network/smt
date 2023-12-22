package smt

import (
	"bytes"
	"errors"

	"github.com/pokt-network/smt/kvstore"
)

// SMTWithStorage wraps an SMT with a mapping of value hashes to values
// (preimages), for use in tests.
// Note: this doesn't delete from preimages (inputs to hashing functions),
// since there could be duplicate stored values.
type SMTWithStorage struct {
	*SMT
	preimages kvstore.MapStore
}

// Update updates a key with a new value in the trie and adds the value to
// the preimages KVStore
func (smt *SMTWithStorage) Update(key, value []byte) error {
	if err := smt.SMT.Update(key, value); err != nil {
		return err
	}
	valueHash := smt.digestValue(value)
	return smt.preimages.Set(valueHash, value)
}

// Delete deletes a key from the trie.
func (smt *SMTWithStorage) Delete(key []byte) error {
	return smt.SMT.Delete(key)
}

// Get gets the value of a key from the trie.
func (smt *SMTWithStorage) GetValue(key []byte) ([]byte, error) {
	valueHash, err := smt.Get(key)
	if err != nil {
		return nil, err
	}
	if valueHash == nil {
		return nil, nil
	}
	value, err := smt.preimages.Get(valueHash)
	if err != nil {
		if errors.Is(err, kvstore.ErrKVStoreKeyNotFound) {
			// If key isn't found, return default value
			value = defaultValue
		} else {
			// Otherwise percolate up any other error
			return nil, err
		}
	}
	return value, nil
}

// Has returns true if the value at the given key is non-default, false
// otherwise.
func (smt *SMTWithStorage) Has(key []byte) (bool, error) {
	val, err := smt.GetValue(key)
	return !bytes.Equal(defaultValue, val), err
}

// ProveCompact generates a compacted Merkle proof for a key against the
// current root.
func ProveCompact(key []byte, smt SparseMerkleTrie) (*SparseCompactMerkleProof, error) {
	proof, err := smt.Prove(key)
	if err != nil {
		return nil, err
	}
	return CompactProof(proof, smt.Spec())
}

// dummyHasher is a dummy hasher for tests, where the digest of keys is
// equivalent to the preimage.
type dummyPathHasher struct {
	size int
}

func (h dummyPathHasher) Path(key []byte) []byte {
	if len(key) != h.size {
		panic("len(key) must equal path size")
	}
	return key
}

func (h dummyPathHasher) PathSize() int { return h.size }
