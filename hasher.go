package smt

import (
	"bytes"
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

func (th *treeHasher) digestNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeInner(leftData, rightData)
	return th.digest(value), value
}

func (th *treeHasher) parseNode(data []byte) ([]byte, []byte) {
	return data[len(innerPrefix) : th.hashSize()+len(innerPrefix)], data[len(innerPrefix)+th.hashSize():]
}

func (th *treeHasher) parseSumNode(data []byte) ([]byte, []byte) {
	return data[len(innerPrefix) : th.hashSize()+len(innerPrefix)+16], data[len(innerPrefix)+th.hashSize()+16:]
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

// parseSumLeaf returns the path, value hash and hex sum of the leaf node
func parseSumLeaf(data []byte, ph PathHasher) ([]byte, []byte, [16]byte) {
	var sum [16]byte
	copy(sum[:], data[len(data)-16:])
	return data[len(leafPrefix) : ph.PathSize()+len(leafPrefix)], data[len(leafPrefix)+ph.PathSize() : len(data)-16], sum
}

func parseExtension(data []byte, ph PathHasher) (pathBounds, path, childData []byte) {
	return data[len(extPrefix) : len(extPrefix)+2],
		data[len(extPrefix)+2 : len(extPrefix)+2+ph.PathSize()],
		data[len(extPrefix)+2+ph.PathSize():]
}

func encodeLeaf(path []byte, leafData []byte) []byte {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)
	return value
}

// encodeSumLeaf returns the pre-image digest of a sum leaf node
func encodeSumLeaf(path []byte, leafData []byte, sum [16]byte) []byte {
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

// encodeSumInner returns the pre-image digest of the innernode, as well as the sum of its children
func encodeSumInner(leftData []byte, rightData []byte) ([]byte, [16]byte, error) {
	value := make([]byte, 0, len(innerPrefix)+len(leftData)+len(rightData))
	value = append(value, innerPrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)
	var sum [16]byte
	leftSum, err := strconv.ParseUint(string(leftData[len(leftData)-16:]), 16, 64)
	if err != nil {
		return nil, sum, err
	}
	rightSum, err := strconv.ParseUint(string(rightData[len(rightData)-16:]), 16, 64)
	if err != nil {
		return nil, sum, err
	}
	copy(sum[:], []byte(fmt.Sprintf("%016x", leftSum+rightSum)))
	return value, sum, nil
}

func encodeExtension(pathBounds [2]byte, path []byte, childData []byte) []byte {
	value := make([]byte, 0, len(extPrefix)+len(path)+2+len(childData))
	value = append(value, extPrefix...)
	value = append(value, pathBounds[:]...)
	value = append(value, path...)
	value = append(value, childData...)
	return value
}

// encodeSumExtension returns the pre-image digest of the extension node and the sum of its child
func encodeSumExtension(pathBounds [2]byte, path []byte, childData []byte) ([]byte, [16]byte) {
	value := make([]byte, 0, len(extPrefix)+len(path)+2+len(childData))
	value = append(value, extPrefix...)
	value = append(value, pathBounds[:]...)
	value = append(value, path...)
	value = append(value, childData...)
	var sum [16]byte
	copy(sum[:], childData[len(childData)-16:])
	return value, sum
}
