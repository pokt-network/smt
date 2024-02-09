package smt

import (
	"bytes"
	"encoding/binary"
)

// TODO_IMPROVE: All of the parsing, encoding and checking functions in this file
// can be abstracted out into the `trieNode` interface.

// NB: In this file, all references to the variable `data` should be treated as `encodedNodeData`.
// It was abbreviated to `data` for brevity.

var (
	leafNodePrefix  = []byte{0}
	innerNodePrefix = []byte{1}
	extNodePrefix   = []byte{2}
)

// isLeafNode returns true if the encoded node data is a leaf node
func isLeafNode(data []byte) bool {
	return bytes.Equal(data[:len(leafNodePrefix)], leafNodePrefix)
}

// isExtNode returns true if the encoded node data is an extension node
func isExtNode(data []byte) bool {
	return bytes.Equal(data[:len(extNodePrefix)], extNodePrefix)
}

func parseLeafNode(data []byte, ph PathHasher) ([]byte, []byte) {
	return data[len(leafNodePrefix) : ph.PathSize()+len(leafNodePrefix)], data[len(leafNodePrefix)+ph.PathSize():]
}

func parseExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte) {
	return data[len(extNodePrefix) : len(extNodePrefix)+2], // +2 represents the length of the pathBounds
		data[len(extNodePrefix)+2 : len(extNodePrefix)+2+ph.PathSize()],
		data[len(extNodePrefix)+2+ph.PathSize():]
}

func parseSumExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumSizeBits]byte) {
	var sumBz [sumSizeBits]byte
	copy(sumBz[:], data[len(data)-sumSizeBits:])
	return data[len(extNodePrefix) : len(extNodePrefix)+2], // +2 represents the length of the pathBounds
		data[len(extNodePrefix)+2 : len(extNodePrefix)+2+ph.PathSize()],
		data[len(extNodePrefix)+2+ph.PathSize() : len(data)-sumSizeBits],
		sumBz
}

// encodeLeafNode encodes leaf nodes. both normal and sum leaves as in the sum leaf the
// sum is appended to the end of the valueHash
func encodeLeafNode(path []byte, leafData []byte) []byte {
	data := make([]byte, 0, len(leafNodePrefix)+len(path)+len(leafData))
	data = append(data, leafNodePrefix...)
	data = append(data, path...)
	data = append(data, leafData...)
	return data
}

func encodeInnerNode(leftData []byte, rightData []byte) []byte {
	data := make([]byte, 0, len(innerNodePrefix)+len(leftData)+len(rightData))
	data = append(data, innerNodePrefix...)
	data = append(data, leftData...)
	data = append(data, rightData...)
	return data
}

func encodeSumInnerNode(leftData []byte, rightData []byte) []byte {
	data := make([]byte, 0, len(innerNodePrefix)+len(leftData)+len(rightData))
	data = append(data, innerNodePrefix...)
	data = append(data, leftData...)
	data = append(data, rightData...)

	var sum [sumSizeBits]byte
	leftSum := uint64(0)
	rightSum := uint64(0)
	leftSumBz := leftData[len(leftData)-sumSizeBits:]
	rightSumBz := rightData[len(rightData)-sumSizeBits:]
	if !bytes.Equal(leftSumBz, defaultEmptySum[:]) {
		leftSum = binary.BigEndian.Uint64(leftSumBz)
	}
	if !bytes.Equal(rightSumBz, defaultEmptySum[:]) {
		rightSum = binary.BigEndian.Uint64(rightSumBz)
	}
	binary.BigEndian.PutUint64(sum[:], leftSum+rightSum)
	data = append(data, sum[:]...)
	return data
}

// encodeExtensionNode encodes the data of an extension nodes
func encodeExtensionNode(pathBounds [2]byte, path []byte, childData []byte) []byte {
	data := []byte{}
	data = append(data, extNodePrefix...)
	data = append(data, pathBounds[:]...)
	data = append(data, path...)
	data = append(data, childData...)
	return data
}

// encodeSumExtensionNode encodes the data of a sum extension nodes
func encodeSumExtensionNode(pathBounds [2]byte, path []byte, childData []byte) []byte {
	data := encodeExtensionNode(pathBounds, path, childData)

	// Append the sum to the end of the data
	var sum [sumSizeBits]byte
	copy(sum[:], childData[len(childData)-sumSizeBits:])
	data = append(data, sum[:]...)

	return data
}
