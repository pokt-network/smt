package tests

import (
	"bytes"
	"errors"

	badger "github.com/dgraph-io/badger/v4"

	"github.com/pokt-network/smt"
)

// SMTWithStorage wraps an SMT with a mapping of value hashes to values (preimages), for use in tests.
// Note: this doesn't delete from preimages (inputs to hashing functions), since there could be duplicate stored values.
type SMTWithStorage struct {
	*smt.SMT
	preimages smt.KVStore
}

// Update updates a key with a new value in the tree and adds the value to the preimages KVStore
func (s *SMTWithStorage) Update(key, value []byte) error {
	if err := s.SMT.Update(key, value); err != nil {
		return err
	}
	valueHash := s.DigestValue(value)
	if err := s.preimages.Set(valueHash, value); err != nil {
		return err
	}
	return nil
}

// Delete deletes a key from the tree.
func (s *SMTWithStorage) Delete(key []byte) error {
	return s.SMT.Delete(key)
}

// Get gets the value of a key from the tree.
func (s *SMTWithStorage) GetValue(key []byte) ([]byte, error) {
	valueHash, err := s.Get(key)
	if err != nil {
		return nil, err
	}
	if valueHash == nil {
		return nil, nil
	}
	value, err := s.preimages.Get(valueHash)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			// If key isn't found, return default value
			value = smt.DefaultValue
		} else {
			// Otherwise percolate up any other error
			return nil, err
		}
	}
	return value, nil
}

// Has returns true if the value at the given key is non-default, false
// otherwise.
func (s *SMTWithStorage) Has(key []byte) (bool, error) {
	val, err := s.GetValue(key)
	return !bytes.Equal(smt.DefaultValue, val), err
}

// ProveCompact generates a compacted Merkle proof for a key against the current root.
func ProveCompact(key []byte, sparseMerkleTree smt.SparseMerkleTree) (*smt.SparseCompactMerkleProof, error) {
	proof, err := sparseMerkleTree.Prove(key)
	if err != nil {
		return nil, err
	}
	return smt.CompactProof(proof, sparseMerkleTree.Spec())
}

// dummyHasher is a dummy hasher for tests, where the digest of keys is equivalent to the preimage.
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
