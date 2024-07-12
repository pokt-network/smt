package smt

import (
	"encoding/binary"
	"fmt"
)

const (
	// These are intentionally exposed to allow for for testing and custom
	// implementations of downstream applications.
	SmtRootSizeBytes  = 32
	SmstRootSizeBytes = SmtRootSizeBytes + sumSizeBytes + countSizeBytes
)

// MustSum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will panic.
func (r MerkleSumRoot) MustSum() uint64 {
	sum, err := r.Sum()
	if err != nil {
		panic(err)
	}

	return sum
}

// Sum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will return an error.
func (r MerkleSumRoot) Sum() (uint64, error) {
	if len(r) != SmstRootSizeBytes {
		return 0, fmt.Errorf("MerkleSumRoot#Sum: not a merkle sum trie")
	}

	return getSum(r), nil
}

// MustCount returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It panics if the root hash length
// does not match that of the SMST hasher.
func (r MerkleSumRoot) MustCount() uint64 {
	count, err := r.Count()
	if err != nil {
		panic(err)
	}

	return count
}

// Count returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It returns an error if the root hash length
// does not match that of the SMST hasher.
func (r MerkleSumRoot) Count() (uint64, error) {
	if len(r) != SmstRootSizeBytes {
		return 0, fmt.Errorf("MerkleSumRoot#Count: not a merkle sum trie")
	}

	return getCount(r), nil
}

// getSum returns the sum of the node stored in the root.
func getSum(root []byte) uint64 {
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(root)

	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], root[firstSumByteIdx:firstCountByteIdx])
	return binary.BigEndian.Uint64(sumBz[:])
}

// getCount returns the count of the node stored in the root.
func getCount(root []byte) uint64 {
	_, firstCountByteIdx := getFirstMetaByteIdx(root)

	var countBz [countSizeBytes]byte
	copy(countBz[:], root[firstCountByteIdx:])
	return binary.BigEndian.Uint64(countBz[:])
}
