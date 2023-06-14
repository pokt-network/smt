package smt

import (
	"bytes"
	"encoding/binary"
)

func (th *treeHasher) digestSumLeaf(path []byte, leafData []byte, sum [sumLength]byte) ([]byte, []byte) {
	value := encodeSumLeaf(path, leafData, sum)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumLength:]...)
	return digest, value
}

func (th *treeHasher) digestSumNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeSumInner(leftData, rightData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumLength:]...)
	return digest, value
}

func (th *treeHasher) parseSumNode(data []byte) ([]byte, []byte) {
	sumless := data[:len(data)-sumLength]
	return sumless[len(innerPrefix) : th.hashSize()+len(innerPrefix)+sumLength], sumless[len(innerPrefix)+th.hashSize()+sumLength:]
}

func (th *treeHasher) sumPlaceholder() []byte {
	placeholder := th.zeroValue
	placeholder = append(placeholder, defaultSum[:]...)
	return placeholder
}

// parseSumLeaf returns the path, value hash and hex sum of the leaf node
func parseSumLeaf(data []byte, ph PathHasher) ([]byte, []byte, [sumLength]byte) {
	var sum [sumLength]byte
	copy(sum[:], data[len(data)-sumLength:])
	return data[len(leafPrefix) : ph.PathSize()+len(leafPrefix)], data[len(leafPrefix)+ph.PathSize() : len(data)-sumLength], sum
}

func parseSumExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumLength]byte) {
	var sumBz [sumLength]byte
	copy(sumBz[:], data[len(data)-sumLength:])
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize() : len(data)-sumLength],
		sumBz
}

func encodeSumLeaf(path []byte, leafData []byte, sum [sumLength]byte) []byte {
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
	var sum [sumLength]byte
	leftSum := uint64(0)
	rightSum := uint64(0)
	if !bytes.Equal(leftData[len(leftData)-sumLength:], defaultSum[:]) {
		leftSum = binary.BigEndian.Uint64(leftData[len(leftData)-sumLength:])
	}
	if !bytes.Equal(rightData[len(rightData)-sumLength:], defaultSum[:]) {
		rightSum = binary.BigEndian.Uint64(rightData[len(rightData)-sumLength:])
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
	var sum [sumLength]byte
	copy(sum[:], childData[len(childData)-sumLength:])
	value = append(value, sum[:]...)
	return value
}
