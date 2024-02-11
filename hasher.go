package smt

import (
	"hash"
)

// TODO_IN_THIS_PR: Improve how the `hasher` file is consolidated (or not)
// with `node_encoders.go` since the two are very similar.

// Ensure the hasher interfaces are satisfied
var (
	_ PathHasher  = (*pathHasher)(nil)
	_ PathHasher  = (*nilPathHasher)(nil)
	_ ValueHasher = (*valueHasher)(nil)
)

// PathHasher defines how key inputs are hashed to produce trie paths.
type PathHasher interface {
	// Path hashes a key (preimage) and returns a trie path (digest).
	Path([]byte) []byte
	// PathSize returns the length (in bytes) of digests produced by this hasher.
	PathSize() int
}

// ValueHasher defines how value data is hashed to produce leaf data.
type ValueHasher interface {
	// HashValue hashes value data to produce the digest stored in leaf node.
	HashValue([]byte) []byte
}

// trieHasher is a common hasher for all trie hashers (paths & values).
type trieHasher struct {
	hasher    hash.Hash
	zeroValue []byte
}

// pathHasher is a hasher for trie paths.
type pathHasher struct {
	trieHasher
}

// valueHasher is a hasher for leaf values.
type valueHasher struct {
	trieHasher
}

type nilPathHasher struct {
	hashSize int
}

// NewTrieHasher returns a new trie hasher with the given hash function.
func NewTrieHasher(hasher hash.Hash) *trieHasher {
	th := trieHasher{hasher: hasher}
	th.zeroValue = make([]byte, th.hashSize())
	return &th
}

func NewNilPathHasher(hasher hash.Hash) PathHasher {
	return &nilPathHasher{hashSize: hasher.Size()}
}

// Path returns the digest of a key produced by the path hasher
func (ph *pathHasher) Path(key []byte) []byte {
	return ph.digestData(key)[:ph.PathSize()]
}

// PathSize returns the length (in bytes) of digests produced by the path hasher
// which is the length of any path in the trie
func (ph *pathHasher) PathSize() int {
	return ph.hasher.Size()
}

// HashValue hashes the data provided using the value hasher
func (vh *valueHasher) HashValue(data []byte) []byte {
	return vh.digestData(data)
}

// Path satisfies the PathHasher#Path interface
func (n *nilPathHasher) Path(key []byte) []byte {
	return key[:n.hashSize]
}

// PathSize satisfies the PathHasher#PathSize interface
func (n *nilPathHasher) PathSize() int {
	return n.hashSize
}

// digestData returns the hash of the data provided using the trie hasher.
func (th *trieHasher) digestData(data []byte) []byte {
	th.hasher.Write(data)
	digest := th.hasher.Sum(nil)
	th.hasher.Reset()
	return digest
}

// digestLeaf returns the encoded leaf data as well as its hash (i.e. digest)
func (th *trieHasher) digestLeaf(path, data []byte) (digest, value []byte) {
	value = encodeLeafNode(path, data)
	digest = th.digestData(value)
	return
}

func (th *trieHasher) digestNode(leftData, rightData []byte) (digest, value []byte) {
	value = encodeInnerNode(leftData, rightData)
	digest = th.digestData(value)
	return
}

func (th *trieHasher) digestSumLeaf(path, leafData []byte) (digest, value []byte) {
	value = encodeLeafNode(path, leafData)
	digest = th.digestData(value)
	digest = append(digest, value[len(value)-sumSizeBits:]...)
	return
}

func (th *trieHasher) digestSumNode(leftData, rightData []byte) (digest, value []byte) {
	value = encodeSumInnerNode(leftData, rightData)
	digest = th.digestData(value)
	digest = append(digest, value[len(value)-sumSizeBits:]...)
	return
}

func (th *trieHasher) parseInnerNode(data []byte) (leftData, rightData []byte) {
	leftData = data[len(innerNodePrefix) : th.hashSize()+len(innerNodePrefix)]
	rightData = data[len(innerNodePrefix)+th.hashSize():]
	return
}

func (th *trieHasher) parseSumInnerNode(data []byte) (leftData, rightData []byte) {
	dataWithoutSum := data[:len(data)-sumSizeBits]
	leftData = dataWithoutSum[len(innerNodePrefix) : len(innerNodePrefix)+th.hashSize()+sumSizeBits]
	rightData = dataWithoutSum[len(innerNodePrefix)+th.hashSize()+sumSizeBits:]
	return
}

func (th *trieHasher) hashSize() int {
	return th.hasher.Size()
}

func (th *trieHasher) placeholder() []byte {
	return th.zeroValue
}
