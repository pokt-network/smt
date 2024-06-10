package smt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/pokt-network/smt/kvstore"
)

// SMSTWithStorage wraps an SMST with a mapping of value hashes to values with
// sums (preimages), for use in tests.
// Note: this doesn't delete from preimages (inputs to hashing functions), since
// there could be duplicate stored values.
type SMSTWithStorage struct {
	*SMST
	preimages kvstore.MapStore
}

// Update a key with a new value in the trie and add it to the preimages KVStore.
// Preimages are the values prior to being hashed, used to confirm the values are in the trie.
func (smst *SMSTWithStorage) Update(key, value []byte, sum uint64) error {
	if err := smst.SMST.Update(key, value, sum); err != nil {
		return err
	}
	valueHash := smst.valueHash(value)

	// Append the sum to the value before storing it
	var sumBz [sumSizeBytes]byte
	binary.BigEndian.PutUint64(sumBz[:], sum)
	value = append(value, sumBz[:]...)

	// Append the count to the value before storing it
	var countBz [countSizeBytes]byte
	binary.BigEndian.PutUint64(countBz[:], 1)
	value = append(value, countBz[:]...)

	return smst.preimages.Set(valueHash, value)
}

// Delete deletes a key from the trie.
func (smst *SMSTWithStorage) Delete(key []byte) error {
	return smst.SMST.Delete(key)
}

// GetValueSum returns the value and sum of the key stored in the trie, by
// looking up the value hash in the preimages KVStore and extracting the sum
func (smst *SMSTWithStorage) GetValueSum(key []byte) ([]byte, uint64, error) {
	valueHash, sum, _, err := smst.Get(key)
	if err != nil {
		return nil, 0, err
	}
	if valueHash == nil {
		return nil, 0, nil
	}
	// Extract the value from the preimages KVStore
	value, err := smst.preimages.Get(valueHash)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			// If key isn't found, return default value and sum
			return defaultEmptyValue, 0, nil
		}
		// Otherwise percolate up any other error
		return nil, 0, err
	}

	firstSumByteIdx, firstCountByteIdx := GetFirstMetaByteIdx(value)

	// Extract the sum from the value
	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], value[firstSumByteIdx:firstCountByteIdx])
	storedSum := binary.BigEndian.Uint64(sumBz[:])
	if storedSum != sum {
		return nil, 0, fmt.Errorf("sum mismatch for %s: got %d, expected %d", string(key), storedSum, sum)
	}

	return value[:firstSumByteIdx], storedSum, nil
}

// Has returns true if the value at the given key is non-default, false otherwise.
func (smst *SMSTWithStorage) Has(key []byte) (bool, error) {
	val, sum, err := smst.GetValueSum(key)
	return !bytes.Equal(defaultEmptyValue, val) || sum != 0, err
}
