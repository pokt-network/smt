package smt

import (
	"bytes"
	"encoding/binary"
)

func (th *treeHasher) digestSumLeaf(path []byte, leafData []byte, sum [sumSize]byte) ([]byte, []byte) {
	value := encodeSumLeaf(path, leafData, sum)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSize:]...)
	return digest, value
}

func (th *treeHasher) digestSumNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeSumInner(leftData, rightData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSize:]...)
	return digest, value
}

func (th *treeHasher) parseSumNode(data []byte) ([]byte, []byte) {
	sumless := data[:len(data)-sumSize]
	return sumless[len(innerPrefix) : th.hashSize()+len(innerPrefix)+sumSize], sumless[len(innerPrefix)+th.hashSize()+sumSize:]
}

func (th *treeHasher) sumPlaceholder() []byte {
	placeholder := th.zeroValue
	placeholder = append(placeholder, defaultSum[:]...)
	return placeholder
}

// parseSumLeaf returns the path, value hash and hex sum of the leaf node
func parseSumLeaf(data []byte, ph PathHasher) ([]byte, []byte, [sumSize]byte) {
	var sum [sumSize]byte
	copy(sum[:], data[len(data)-sumSize:])
	return data[len(leafPrefix) : ph.PathSize()+len(leafPrefix)], data[len(leafPrefix)+ph.PathSize() : len(data)-sumSize], sum
}

func parseSumExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumSize]byte) {
	var sumBz [sumSize]byte
	copy(sumBz[:], data[len(data)-sumSize:])
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize() : len(data)-sumSize],
		sumBz
}

func encodeSumLeaf(path []byte, leafData []byte, sum [sumSize]byte) []byte {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)
	value = append(value, sum[:]...)
	return value
}

func encodeSumInner(leftData []byte, rightData []byte) []byte {
	value := make([]byte, 0, len(innerPrefix)+len(leftData)+len(rightData))
	value = append(value, innerPrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)
	var sum [sumSize]byte
	leftSum := uint64(0)
	rightSum := uint64(0)
	if !bytes.Equal(leftData[len(leftData)-sumSize:], defaultSum[:]) {
		leftSum = binary.BigEndian.Uint64(leftData[len(leftData)-sumSize:])
	}
	if !bytes.Equal(rightData[len(rightData)-sumSize:], defaultSum[:]) {
		rightSum = binary.BigEndian.Uint64(rightData[len(rightData)-sumSize:])
	}
	binary.BigEndian.PutUint64(sum[:], leftSum+rightSum)
	value = append(value, sum[:]...)
	return value
}

func encodeSumExtension(pathBounds [2]byte, path []byte, childData []byte) []byte {
	value := make([]byte, 0, len(extPrefix)+len(path)+2+len(childData))
	value = append(value, extPrefix...)
	value = append(value, pathBounds[:]...)
	value = append(value, path...)
	value = append(value, childData...)
	var sum [sumSize]byte
	copy(sum[:], childData[len(childData)-sumSize:])
	value = append(value, sum[:]...)
	return value
}
