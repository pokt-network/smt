package smt

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
)

var (
	leafPrefix  = []byte{0}
	innerPrefix = []byte{1}
	extPrefix   = []byte{2}
)

var (
	_ PathHasher  = (*pathHasher)(nil)
	_ ValueHasher = (*valueHasher)(nil)
)

// PathHasher defines how key inputs are hashed to produce tree paths.
type PathHasher interface {
	// Path hashes a key (preimage) and returns a tree path (digest).
	Path([]byte) []byte
	// PathSize returns the length (in bytes) of digests produced by this hasher.
	PathSize() int
}

// ValueHasher defines how value data is hashed to produce leaf data.
type ValueHasher interface {
	// HashValue hashes value data to produce the digest stored in leaf node.
	HashValue([]byte) []byte
}

type treeHasher struct {
	hasher    hash.Hash
	zeroValue []byte
}
type pathHasher struct {
	treeHasher
}
type valueHasher struct {
	treeHasher
}

func newTreeHasher(hasher hash.Hash) *treeHasher {
	th := treeHasher{hasher: hasher}
	th.zeroValue = make([]byte, th.hashSize())
	return &th
}

// Path returns the digest of a key produced by the path hasher
func (ph *pathHasher) Path(key []byte) []byte {
	return ph.digest(key)[:ph.PathSize()]
}

// PathSize returns the length (in bytes) of digests produced by the path hasher
// which is the length of any path in the tree
func (ph *pathHasher) PathSize() int {
	return ph.hasher.Size()
}

func (vh *valueHasher) HashValue(data []byte) []byte {
	return vh.digest(data)
}

func (th *treeHasher) digest(data []byte) []byte {
	th.hasher.Write(data)
	sum := th.hasher.Sum(nil)
	th.hasher.Reset()
	return sum
}

func (th *treeHasher) digestLeaf(path []byte, leafData []byte) ([]byte, []byte) {
	value := encodeLeaf(path, leafData)
	return th.digest(value), value
}

func (th *treeHasher) digestSumLeaf(path []byte, leafData []byte, sum [sumLength]byte) ([]byte, []byte) {
	value := encodeSumLeaf(path, leafData, sum)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumLength:]...)
	return digest, value
}

func (th *treeHasher) digestNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeInner(leftData, rightData)
	return th.digest(value), value
}

func (th *treeHasher) digestSumNode(leftData []byte, rightData []byte) ([]byte, []byte, error) {
	value, err := encodeSumInner(leftData, rightData)
	if err != nil {
		return nil, nil, err
	}
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumLength:]...)
	return digest, value, nil
}

func (th *treeHasher) parseNode(data []byte) ([]byte, []byte) {
	return data[len(innerPrefix) : th.hashSize()+len(innerPrefix)], data[len(innerPrefix)+th.hashSize():]
}

func (th *treeHasher) parseSumNode(data []byte) ([]byte, []byte) {
	sumless := data[:len(data)-sumLength]
	return sumless[len(innerPrefix) : th.hashSize()+len(innerPrefix)+sumLength], sumless[len(innerPrefix)+th.hashSize()+sumLength:]
}

func (th *treeHasher) hashSize() int {
	return th.hasher.Size()
}

func (th *treeHasher) placeholder() []byte {
	return th.zeroValue
}

func (th *treeHasher) sumPlaceholder() []byte {
	placeholder := th.zeroValue
	placeholder = append(placeholder, defaultSum[:]...)
	return placeholder
}

func isLeaf(data []byte) bool {
	return bytes.Equal(data[:len(leafPrefix)], leafPrefix)
}

func isExtension(data []byte) bool {
	return bytes.Equal(data[:len(extPrefix)], extPrefix)
}

func parseLeaf(data []byte, ph PathHasher) ([]byte, []byte) {
	return data[len(leafPrefix) : ph.PathSize()+len(leafPrefix)], data[len(leafPrefix)+ph.PathSize():]
}

// parseSumLeaf returns the path, value hash and hex sum of the leaf node
func parseSumLeaf(data []byte, ph PathHasher) ([]byte, []byte, [sumLength]byte) {
	var sum [sumLength]byte
	copy(sum[:], data[len(data)-sumLength:])
	return data[len(leafPrefix) : ph.PathSize()+len(leafPrefix)], data[len(leafPrefix)+ph.PathSize() : len(data)-sumLength], sum
}

func parseExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte) {
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize():]
}

func parseSumExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumLength]byte) {
	var hexSum [sumLength]byte
	copy(hexSum[:], data[len(data)-sumLength:])
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize() : len(data)-sumLength],
		hexSum
}

func encodeLeaf(path []byte, leafData []byte) []byte {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)
	return value
}

func encodeSumLeaf(path []byte, leafData []byte, sum [sumLength]byte) []byte {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)
	value = append(value, sum[:]...)
	return value
}

func encodeInner(leftData []byte, rightData []byte) []byte {
	value := make([]byte, 0, len(innerPrefix)+len(leftData)+len(rightData))
	value = append(value, innerPrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)
	return value
}

func encodeSumInner(leftData []byte, rightData []byte) ([]byte, error) {
	value := make([]byte, 0, len(innerPrefix)+len(leftData)+len(rightData))
	value = append(value, innerPrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)
	var sum [sumLength]byte
	var err error
	leftSum := uint64(0)
	rightSum := uint64(0)
	if !bytes.Equal(leftData[len(leftData)-sumLength:], defaultSum[:]) {
		leftSum, err = strconv.ParseUint(hex.EncodeToString(leftData[len(leftData)-sumLength:]), 16, 64)
		if err != nil {
			return nil, err
		}
	}
	if !bytes.Equal(rightData[len(rightData)-sumLength:], defaultSum[:]) {
		rightSum, err = strconv.ParseUint(hex.EncodeToString(rightData[len(rightData)-sumLength:]), 16, 64)
		if err != nil {
			return nil, err
		}
	}
	totalBz, err := hex.DecodeString(fmt.Sprintf("%016x", leftSum+rightSum))
	if err != nil {
		return nil, err
	}
	copy(sum[sumLength-len(totalBz):], totalBz)
	value = append(value, sum[:]...)
	return value, nil
}

func encodeExtension(pathBounds [2]byte, path []byte, childData []byte) []byte {
	value := make([]byte, 0, len(extPrefix)+len(path)+2+len(childData))
	value = append(value, extPrefix...)
	value = append(value, pathBounds[:]...)
	value = append(value, path...)
	value = append(value, childData...)
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
