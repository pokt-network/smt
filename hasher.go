package smt

import (
	"encoding/binary"
	"hash"
)

// TODO_IMPROVE:: Improve how the `hasher` file is consolidated with
// `node_encoders.go` since the two are very similar.

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
	// ValueHashSize returns the length (in bytes) of digests produced by this hasher.
	ValueHashSize() int
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

// nilPathHasher is a dummy hasher that returns its input - it should not be used outside of the closest proof verification logic
type nilPathHasher struct {
	hashSize int
}

// NewTrieHasher returns a new trie hasher with the given hash function.
func NewTrieHasher(hasher hash.Hash) *trieHasher {
	th := trieHasher{hasher: hasher}
	th.zeroValue = make([]byte, th.hashSize())
	return &th
}

func NewNilPathHasher(hasherSize int) PathHasher {
	return &nilPathHasher{hashSize: hasherSize}
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

// HashValue hashes the produces a digest of the data provided by the value hasher
func (vh *valueHasher) HashValue(data []byte) []byte {
	return vh.digestData(data)
}

// ValueHashSize returns the length (in bytes) of digests produced by the value hasher
func (vh *valueHasher) ValueHashSize() int {
	if vh.hasher == nil {
		return 0
	}
	return vh.hasher.Size()
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

// digestLeafNode returns the encoded leaf data as well as its hash (i.e. digest)
func (th *trieHasher) digestLeafNode(path, data []byte) (digest, value []byte) {
	value = encodeLeafNode(path, data)
	digest = th.digestData(value)
	return
}

func (th *trieHasher) digestInnerNode(leftData, rightData []byte) (digest, value []byte) {
	value = encodeInnerNode(leftData, rightData)
	digest = th.digestData(value)
	return
}

func (th *trieHasher) digestSumLeafNode(path, data []byte) (digest, value []byte) {
	value = encodeLeafNode(path, data)
	digest = th.digestData(value)
	digest = append(digest, value[len(value)-sumSizeBits:]...)
	return
}

func (th *trieHasher) digestSumInnerNode(leftData, rightData []byte) (digest, value []byte) {
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

func (th *trieHasher) parseSumInnerNode(data []byte) (leftData, rightData []byte, sum uint64) {
	// Extract the sum from the encoded node data
	var sumBz [sumSizeBits]byte
	copy(sumBz[:], data[len(data)-sumSizeBits:])
	binary.BigEndian.PutUint64(sumBz[:], sum)

	// Extract the left and right children
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
