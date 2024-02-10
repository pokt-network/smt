package smt

import (
	"bytes"
	"encoding/binary"
)

// TODO_IMPROVE: All of the parsing, encoding and checking functions in this file
// can be abstracted out into the `trieNode` interface.

// TODO_IMPROVE: We should create well-defined types & structs for every type of node
// (e.g. protobufs) to streamline the process of encoding & encoding and to improve
// readability.

// NB: In this file, all references to the variable `data` should be treated as `encodedNodeData`.
// It was abbreviated to `data` for brevity.

var (
	leafNodePrefix  = []byte{0}
	innerNodePrefix = []byte{1}
	extNodePrefix   = []byte{2}
	prefixLen       = 1
)

// NB: We use `prefixLen` a lot through this file, so to make the code more readable, we
// define it as a constant but need to assert on its length just in case the code evolves
// in the future.
func init() {
	if len(leafNodePrefix) != prefixLen ||
		len(innerNodePrefix) != prefixLen ||
		len(extNodePrefix) != prefixLen {
		panic("invalid prefix length")
	}
}

// isLeafNode returns true if the encoded node data is a leaf node
func isLeafNode(data []byte) bool {
	return bytes.Equal(data[:prefixLen], leafNodePrefix)
}

// isExtNode returns true if the encoded node data is an extension node
func isExtNode(data []byte) bool {
	return bytes.Equal(data[:prefixLen], extNodePrefix)
}

// parseLeafNode parses a leafNode into its components
func parseLeafNode(data []byte, ph PathHasher) (path, value []byte) {
	path = data[prefixLen : prefixLen+ph.PathSize()]
	value = data[prefixLen+ph.PathSize():]
	return
}

// parseExtNode parses an extNode into its components
func parseExtNode(data []byte, ph PathHasher) (pathBounds, path, childData []byte) {
	// +2 represents the length of the pathBounds
	pathBounds = data[prefixLen : prefixLen+2]
	path = data[prefixLen+2 : prefixLen+2+ph.PathSize()]
	childData = data[prefixLen+2+ph.PathSize():]
	return
}

// parseSumExtNode parses the pathBounds, path, child data and sum from the encoded extension node data
func parseSumExtNode(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumSizeBits]byte) {
	// Extract the sum from the encoded node data
	var sumBz [sumSizeBits]byte
	copy(sumBz[:], data[len(data)-sumSizeBits:])

	// +2 represents the length of the pathBounds
	pathBounds = data[prefixLen : prefixLen+2]
	path = data[prefixLen+2 : prefixLen+2+ph.PathSize()]
	childData = data[prefixLen+2+ph.PathSize() : len(data)-sumSizeBits]
	return
}

// encodeLeafNode encodes leaf nodes. both normal and sum leaves as in the sum leaf the
// sum is appended to the end of the valueHash
func encodeLeafNode(path, leafData []byte) (data []byte) {
	data = append(data, leafNodePrefix...)
	data = append(data, path...)
	data = append(data, leafData...)
	return
}

// encodeInnerNode encodes inner node given the data for both children
func encodeInnerNode(leftData, rightData []byte) (data []byte) {
	data = append(data, innerNodePrefix...)
	data = append(data, leftData...)
	data = append(data, rightData...)
	return
}

// encodeSumInnerNode encodes an inner node for an smst given the data for both children
func encodeSumInnerNode(leftData, rightData []byte) (data []byte) {
	// Retrieve the sum of the left subtree
	leftSum := uint64(0)
	leftSumBz := leftData[len(leftData)-sumSizeBits:]
	if !bytes.Equal(leftSumBz, defaultEmptySum[:]) {
		leftSum = binary.BigEndian.Uint64(leftSumBz)
	}

	// Retrieve the sum of the right subtree
	rightSum := uint64(0)
	rightSumBz := rightData[len(rightData)-sumSizeBits:]
	if !bytes.Equal(rightSumBz, defaultEmptySum[:]) {
		rightSum = binary.BigEndian.Uint64(rightSumBz)
	}

	// Compute the sum of the current node
	var sum [sumSizeBits]byte
	binary.BigEndian.PutUint64(sum[:], leftSum+rightSum)

	// Prepare and return the encoded inner node data
	data = encodeInnerNode(leftData, rightData)
	data = append(data, sum[:]...)
	return
}

// encodeExtensionNode encodes the data of an extension nodes
func encodeExtensionNode(pathBounds [2]byte, path, childData []byte) (data []byte) {
	data = append(data, extNodePrefix...)
	data = append(data, pathBounds[:]...)
	data = append(data, path...)
	data = append(data, childData...)
	return
}

// encodeSumExtensionNode encodes the data of a sum extension nodes
func encodeSumExtensionNode(pathBounds [2]byte, path, childData []byte) (data []byte) {

	// Compute the sum of the current node
	var sum [sumSizeBits]byte
	copy(sum[:], childData[len(childData)-sumSizeBits:])

	// Prepare and return the encoded inner node data
	data = encodeExtensionNode(pathBounds, path, childData)
	data = append(data, sum[:]...)
	return
}
