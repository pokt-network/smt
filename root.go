package smt

import "encoding/binary"

const (
	nonSumRootSizeBytes = 32
)

// Sum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will panic.
func (r MerkleRoot) Sum() uint64 {
	if len(r)%nonSumRootSizeBytes == 0 {
		panic("roo#sum: not a merkle sum trie")
	}

	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx([]byte(r))

	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], []byte(r)[firstSumByteIdx:firstCountByteIdx])
	return binary.BigEndian.Uint64(sumBz[:])
}

// Count returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree.
func (r MerkleRoot) Count() uint64 {
	if len(r)%nonSumRootSizeBytes == 0 {
		panic("roo#sum: not a merkle sum trie")
	}

	_, firstCountByteIdx := getFirstMetaByteIdx([]byte(r))

	var countBz [countSizeBytes]byte
	copy(countBz[:], []byte(r)[firstCountByteIdx:])
	return binary.BigEndian.Uint64(countBz[:])
}
