package smt

import (
	"encoding/binary"
	"fmt"
)

// MustCount returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It panics if the root length
// is invalid.
func (root MerkleRoot) MustCount() uint64 {
	count, err := root.Count()
	if err != nil {
		panic(err)
	}

	return count
}

// Count returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It returns an error if the
// root length is invalid.
func (root MerkleRoot) Count() (uint64, error) {
	if err := root.validateBasic(); err != nil {
		return 0, err
	}

	return root.count(), nil
}

// DigestSize returns the length of the digest portion of the root.
func (root MerkleRoot) DigestSize() int {
	return len(root) - countSizeBytes - sumSizeBytes
}

// HasDigestSize returns true if the root digest size is the same as
// that of the size of the given hasher.
func (root MerkleRoot) HasDigestSize(size int) bool {
	return root.DigestSize() == size
}

// MustSum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will panic.
func (root MerkleSumRoot) MustSum() uint64 {
	sum, err := root.Sum()
	if err != nil {
		panic(err)
	}

	return sum
}

// Sum returns the uint64 sum of the merkle root, it checks the length of the
// merkle root and if it is no the same as the size of the SMST's expected
// root hash it will return an error.
func (root MerkleSumRoot) Sum() (uint64, error) {
	if err := root.validateBasic(); err != nil {
		return 0, err
	}

	return root.sum(), nil
}

// validateBasic returns an error if the root digest size is not a power of two.
func (root MerkleRoot) validateBasic() error {
	if !isPowerOfTwo(root.DigestSize()) {
		return fmt.Errorf("MerkleSumRoot#validateBasic: invalid root length")
	}

	return nil
}

// count returns the count of the node stored in the root.
func (root MerkleRoot) count() uint64 {
	_, firstCountByteIdx := getFirstMetaByteIdx(root)

	return binary.BigEndian.Uint64(root[firstCountByteIdx:])
}

// sum returns the sum of the node stored in the root.
func (root MerkleSumRoot) sum() uint64 {
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(root)

	return binary.BigEndian.Uint64(root[firstSumByteIdx:firstCountByteIdx])
}

// isPowerOfTwo function returns true if the input n is a power of 2
func isPowerOfTwo(n int) bool {
	// A power of 2 has only one bit set in its binary representation
	if n <= 0 {
		return false
	}
	return (n & (n - 1)) == 0
}
