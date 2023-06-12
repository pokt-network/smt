package smt

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

// SMSTWithStorage Wraps an SMST with a mapping of value hashes to values with sums (preimages), for use in tests.
// Note: this doesn't delete from preimages, since there could be duplicate stored values.
type SMSTWithStorage struct {
	*SMST
	preimages MapStore
}

func (smst *SMSTWithStorage) Update(key, value []byte, sum uint64) error {
	err := smst.SMST.Update(key, value, sum)
	if err != nil {
		return err
	}
	valueHash := smst.digestValue(value)
	var hexSum [sumLength]byte
	hexBz, err := hex.DecodeString(fmt.Sprintf("%016x", sum))
	if err != nil {
		return err
	}
	copy(hexSum[sumLength-len(hexBz):], hexBz)
	value = append(value, hexSum[:]...)
	err = smst.preimages.Set(valueHash, value)
	if err != nil {
		return err
	}
	return err
}

func (smst *SMSTWithStorage) Delete(key []byte) error {
	err := smst.SMST.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// Get gets the value and sum of a key from the tree.
func (smst *SMSTWithStorage) GetValueSum(key []byte) ([]byte, uint64, error) {
	valueHash, sum, err := smst.Get(key)
	if err != nil {
		return nil, 0, err
	}
	value, err := smst.preimages.Get(valueHash)
	if err != nil {
		var invalidKeyError *InvalidKeyError
		if errors.As(err, &invalidKeyError) {
			// If key isn't found, return default value and sum
			return defaultValue, 0, nil
		} else {
			// Otherwise percolate up any other error
			return nil, 0, err
		}
	}
	var hexSum [sumLength]byte
	copy(hexSum[:], value[len(value)-sumLength:])
	storedSum, err := sumFromHex(hexSum[:])
	if err != nil {
		return nil, 0, err
	}
	if storedSum != sum {
		return nil, 0, fmt.Errorf("sum mismatch for %s: got %d, expected %d", string(key), storedSum, sum)
	}
	return value[:len(value)-sumLength], storedSum, nil
}

// Has returns true if the value at the given key is non-default, false
// otherwise.
func (smst *SMSTWithStorage) Has(key []byte) (bool, error) {
	val, sum, err := smst.GetValueSum(key)
	return !bytes.Equal(defaultValue, val) || sum != 0, err
}

// ProveCompact generates a compacted Merkle proof for a key against the current root.
func ProveSumCompact(key []byte, smst SparseMerkleSumTree) (SparseCompactMerkleSumProof, error) {
	proof, err := smst.Prove(key)
	if err != nil {
		return SparseCompactMerkleSumProof{}, err
	}
	return CompactSumProof(proof, smst.Spec())
}