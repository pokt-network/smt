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
func (r MerkleRoot) MustSum() uint64 {
	sum, err := r.Sum()
	if err != nil {
		panic(err)
	}

	return sum
}

// Sum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will return an error.
func (r MerkleRoot) Sum() (uint64, error) {
	if len(r)%SmtRootSizeBytes == 0 {
		return 0, fmt.Errorf("root#sum: not a merkle sum trie")
	}

	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx([]byte(r))

	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], []byte(r)[firstSumByteIdx:firstCountByteIdx])
	return binary.BigEndian.Uint64(sumBz[:]), nil
}

// Count returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree.
func (r MerkleRoot) Count() uint64 {
	if len(r)%SmtRootSizeBytes == 0 {
		panic("root#sum: not a merkle sum trie")
	}

	_, firstCountByteIdx := getFirstMetaByteIdx([]byte(r))

	var countBz [countSizeBytes]byte
	copy(countBz[:], []byte(r)[firstCountByteIdx:])
	return binary.BigEndian.Uint64(countBz[:])
}
