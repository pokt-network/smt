package smt

import (
	"bytes"
	"encoding/binary"
	"hash"
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

func (th *treeHasher) digestSumLeaf(path []byte, leafData []byte) ([]byte, []byte) {
	value := encodeLeaf(path, leafData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSize:]...)
	return digest, value
}

func (th *treeHasher) digestNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeInner(leftData, rightData)
	return th.digest(value), value
}

func (th *treeHasher) digestSumNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeSumInner(leftData, rightData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSize:]...)
	return digest, value
}

func (th *treeHasher) parseNode(data []byte) ([]byte, []byte) {
	return data[len(innerPrefix) : th.hashSize()+len(innerPrefix)], data[len(innerPrefix)+th.hashSize():]
}

func (th *treeHasher) parseSumNode(data []byte) ([]byte, []byte) {
	sumless := data[:len(data)-sumSize]
	return sumless[len(innerPrefix) : th.hashSize()+len(innerPrefix)+sumSize], sumless[len(innerPrefix)+th.hashSize()+sumSize:]
}

func (th *treeHasher) hashSize() int {
	return th.hasher.Size()
}

func (th *treeHasher) placeholder() []byte {
	return th.zeroValue
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

func parseExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte) {
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize():]
}

func parseSumExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte, sum [sumSize]byte) {
	var sumBz [sumSize]byte
	copy(sumBz[:], data[len(data)-sumSize:])
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize() : len(data)-sumSize],
		sumBz
}

// encodeLeaf encodes both normal and sum leaves as in the sum leaf the
// sum is appended to the end of the valueHash
func encodeLeaf(path []byte, leafData []byte) []byte {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)
	return value
}

func encodeInner(leftData []byte, rightData []byte) []byte {
	value := make([]byte, 0, len(innerPrefix)+len(leftData)+len(rightData))
	value = append(value, innerPrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)
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
	var sum [sumSize]byte
	copy(sum[:], childData[len(childData)-sumSize:])
	value = append(value, sum[:]...)
	return value
}
