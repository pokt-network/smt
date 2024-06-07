package smt

import (
	"bytes"
	"encoding/binary"
)

// TODO_TECHDEBT: All of the parsing, encoding and checking functions in this file
// can be abstracted out into the `trieNode` interface.

// TODO_IMPROVE: We should create well-defined structs for every type of node
// to streamline the process of encoding & encoding and to improve readability.
// If decoding needs to be language agnostic (to implement POKT clients), in other
// languages, protobufs should be considered. If decoding does not need to be
// language agnostic, we can use Go's gob package for more efficient serialization.

// NB: In this file, all references to the variable `data` should be treated as `encodedNodeData`.
// It was abbreviated to `data` for brevity.

// TODO_TECHDEBT: We can easily use `iota` and ENUMS to create a wait to have
// more expressive code, and leverage switches statements throughout.
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

// isInnerNode returns true if the encoded node data is an inner node
func isInnerNode(data []byte) bool {
	return bytes.Equal(data[:prefixLen], innerNodePrefix)
}

// encodeLeafNode encodes leaf nodes. This function applies to both the SMT and
// SMST since the weight of the node is appended to the end of the valueHash.
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

// encodeExtensionNode encodes the data of an extension nodes
func encodeExtensionNode(pathBounds [2]byte, path, childData []byte) (data []byte) {
	data = append(data, extNodePrefix...)
	data = append(data, pathBounds[:]...)
	data = append(data, path...)
	data = append(data, childData...)
	return
}

// encodeSumInnerNode encodes an inner node for an smst given the data for both children
func encodeSumInnerNode(leftData, rightData []byte) (data []byte) {
	// Compute the sum of the current node
	var sum [sumSizeBytes]byte
	leftSum := parseSum(leftData)
	rightSum := parseSum(rightData)
	// TODO_CONSIDERATION: ` I chose BigEndian for readability but most computers
	// now are optimized for LittleEndian encoding could be a micro optimization one day.`
	binary.BigEndian.PutUint64(sum[:], leftSum+rightSum)
	var count [countSizeBytes]byte
	binary.BigEndian.PutUint64(count[:], parseCount(leftData)+parseCount(rightData))

	// Prepare and return the encoded inner node data
	data = encodeInnerNode(leftData, rightData)
	data = append(data, sum[:]...)
	return
}

// encodeSumExtensionNode encodes the data of a sum extension nodes
func encodeSumExtensionNode(pathBounds [2]byte, path, childData []byte) (data []byte) {
	// Compute the sum of the current node
	var sum [sumSizeBytes]byte
	copy(sum[:], childData[len(childData)-sumSizeBytes:])

	// Compute the count of the current node
	var count [countSizeBytes]byte
	binary.BigEndian.PutUint64(count[:], parseCount(childData))

	// Prepare and return the encoded inner node data
	data = encodeExtensionNode(pathBounds, path, childData)
	data = append(data, sum[:]...)
	data = append(data, count[:]...)
	return
}

// checkPrefix panics if the prefix of the data does not match the expected prefix
func checkPrefix(data, prefix []byte) {
	if !bytes.Equal(data[:prefixLen], prefix) {
		panic("invalid prefix")
	}
}

// parseSum parses the sum from the encoded node data
func parseSum(data []byte) uint64 {
	sum := uint64(0)
	sumBz := data[len(data)-sumSizeBytes:]
	if !bytes.Equal(sumBz, defaultEmptySum[:]) {
		sum = binary.BigEndian.Uint64(sumBz)
	}
	return sum
}

func parseCount(data []byte) uint64 {
	count := uint64(0)
	countBz := data[len(data)-sumSizeBytes-countSizeBytes : len(data)-sumSizeBytes]
	if !bytes.Equal(countBz, defaultEmptyCount[:]) {
		count = binary.BigEndian.Uint64(countBz)
	}
	return count
}
