package smt

import (
	"encoding/binary"
	"fmt"
)

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

// MustCount returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It panics if the root length
// is invalid.
func (root MerkleSumRoot) MustCount() uint64 {
	count, err := root.Count()
	if err != nil {
		panic(err)
	}

	return count
}

// Count returns the uint64 count of the merkle root, a cryptographically secure
// count of the number of non-empty leafs in the tree. It returns an error if the
// root length is invalid.
func (root MerkleSumRoot) Count() (uint64, error) {
	if err := root.validateBasic(); err != nil {
		return 0, err
	}

	return root.count(), nil
}

// HasDigestSize returns true if the root hash (digest) length is the same as
// that of the size of the given hasher.
func (root MerkleSumRoot) HasDigestSize(size int) bool {
	return root.length() == size
}

// validateBasic returns an error if the root (digest) length is not a power of two.
func (root MerkleSumRoot) validateBasic() error {
	if !isPowerOfTwo(root.length()) {
		return fmt.Errorf("MerkleSumRoot#validateBasic: invalid root length")
	}

	return nil
}

// length returns the length of the digest portion of the root.
func (root MerkleSumRoot) length() int {
	return len(root) - countSizeBytes - sumSizeBytes
}

// sum returns the sum of the node stored in the root.
func (root MerkleSumRoot) sum() uint64 {
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(root)

	return binary.BigEndian.Uint64(root[firstSumByteIdx:firstCountByteIdx])
}

// count returns the count of the node stored in the root.
func (root MerkleSumRoot) count() uint64 {
	_, firstCountByteIdx := getFirstMetaByteIdx(root)

	return binary.BigEndian.Uint64(root[firstCountByteIdx:])
}

// isPowerOfTwo function returns true if the input n is a power of 2
func isPowerOfTwo(n int) bool {
	// A power of 2 has only one bit set in its binary representation
	if n <= 0 {
		return false
	}
	return (n & (n - 1)) == 0
}
