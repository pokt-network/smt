package smt

import (
	"hash"
)

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
	return ph.digest(key)[:ph.PathSize()]
}

// PathSize returns the length (in bytes) of digests produced by the path hasher
// which is the length of any path in the trie
func (ph *pathHasher) PathSize() int {
	return ph.hasher.Size()
}

// HashValue hashes the data provided using the value hasher
func (vh *valueHasher) HashValue(data []byte) []byte {
	return vh.digest(data)
}

// Path satisfies the PathHasher#Path interface
func (n *nilPathHasher) Path(key []byte) []byte {
	return key[:n.hashSize]
}

// PathSize satisfies the PathHasher#PathSize interface
func (n *nilPathHasher) PathSize() int {
	return n.hashSize
}

// digest returns the hash of the data provided using the trie hasher.
func (th *trieHasher) digest(data []byte) []byte {
	th.hasher.Write(data)
	sum := th.hasher.Sum(nil)
	th.hasher.Reset()
	return sum
}

// digestLeaf returns the hash of the leaf data & pathprovided using the trie hasher.
func (th *trieHasher) digestLeaf(path, data []byte) ([]byte, []byte) {
	value := encodeLeafNode(path, data)
	return th.digest(value), value
}

func (th *trieHasher) digestSumLeaf(path []byte, leafData []byte) ([]byte, []byte) {
	value := encodeLeafNode(path, leafData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSizeBits:]...)
	return digest, value
}

func (th *trieHasher) digestNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeInnerNode(leftData, rightData)
	return th.digest(value), value
}

func (th *trieHasher) digestSumNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := encodeSumInnerNode(leftData, rightData)
	digest := th.digest(value)
	digest = append(digest, value[len(value)-sumSizeBits:]...)
	return digest, value
}

func (th *trieHasher) parseNode(data []byte) ([]byte, []byte) {
	return data[len(innerNodePrefix) : th.hashSize()+len(innerNodePrefix)], data[len(innerNodePrefix)+th.hashSize():]
}

func (th *trieHasher) parseSumNode(data []byte) ([]byte, []byte) {
	sumless := data[:len(data)-sumSizeBits]
	return sumless[len(innerNodePrefix) : th.hashSize()+sumSizeBits+len(innerNodePrefix)], sumless[len(innerNodePrefix)+th.hashSize()+sumSizeBits:]
}

func (th *trieHasher) hashSize() int {
	return th.hasher.Size()
}

func (th *trieHasher) placeholder() []byte {
	return th.zeroValue
}
