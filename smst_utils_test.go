package smt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// SMSTWithStorage wraps an SMST with a mapping of value hashes to values with sums (preimages), for use in tests.
// Note: this doesn't delete from preimages (inputs to hashing functions), since there could be duplicate stored values.
type SMSTWithStorage struct {
	*SMST
	preimages KVStore
}

// Update updates a key with a new value in the tree and adds the value to the preimages KVStore
func (smst *SMSTWithStorage) Update(key, value []byte, sum uint64) error {
	if err := smst.SMST.Update(key, value, sum); err != nil {
		return err
	}
	valueHash := smst.digestValue(value)
	var sumBz [sumSize]byte
	binary.BigEndian.PutUint64(sumBz[:], sum)
	value = append(value, sumBz[:]...)
	if err := smst.preimages.Set(valueHash, value); err != nil {
		return err
	}
	return nil
}

// Delete deletes a key from the tree.
func (smst *SMSTWithStorage) Delete(key []byte) error {
	return smst.SMST.Delete(key)
}

// GetValueSum returns the value and sum of the key stored in the tree, by looking up
// the value hash in the preimages KVStore and extracting the sum
func (smst *SMSTWithStorage) GetValueSum(key []byte) ([]byte, uint64, error) {
	valueHash, sum, err := smst.Get(key)
	if err != nil {
		return nil, 0, err
	}
	if valueHash == nil {
		return nil, 0, nil
	}
	value, err := smst.preimages.Get(valueHash)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			// If key isn't found, return default value and sum
			return defaultValue, 0, nil
		} else {
			// Otherwise percolate up any other error
			return nil, 0, err
		}
	}
	var sumBz [sumSize]byte
	copy(sumBz[:], value[len(value)-sumSize:])
	storedSum := binary.BigEndian.Uint64(sumBz[:])
	if storedSum != sum {
		return nil, 0, fmt.Errorf("sum mismatch for %s: got %d, expected %d", string(key), storedSum, sum)
	}
	return value[:len(value)-sumSize], storedSum, nil
}

// Has returns true if the value at the given key is non-default, false otherwise.
func (smst *SMSTWithStorage) Has(key []byte) (bool, error) {
	val, sum, err := smst.GetValueSum(key)
	return !bytes.Equal(defaultValue, val) || sum != 0, err
}

// ProveSumCompact generates a compacted Merkle proof for a key against the current root.
func ProveSumCompact(key []byte, smst SparseMerkleSumTree) (*SparseCompactMerkleProof, error) {
	proof, err := smst.Prove(key)
	if err != nil {
		return nil, err
	}
	return CompactProof(proof, smst.Spec())
}
