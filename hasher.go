package smt

import (
	"encoding/binary"
	"hash"
	"sync"
)

// TODO_IMPROVE:: Improve how the `hasher` file is consolidated with
// `node_encoders.go` since the two are very similar.

// Ensure the hasher interfaces are satisfied
var (
	_ PathHasher  = (*pathHasher)(nil)
	_ PathHasher  = (*nilPathHasher)(nil)
	_ ValueHasher = (*valueHasher)(nil)
)

const (
	// Reasonable size to pre-allocate for the buffer pool
	bufferPoolPreallocationSize = 512

	// Maximum size for the buffer pool to avoid pooling very large buffers
	maxSizeForBufferPool = 1024
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

// bufferPool provides reusable byte slices to reduce allocations
var bufferPool = sync.Pool{
	New: func() any {
		// Pre-allocate reasonable capacity
		buf := make([]byte, 0, bufferPoolPreallocationSize)
		return &buf
	},
}

// getBuffer returns a reusable byte slice from the pool
func getBuffer() []byte {
	buf := bufferPool.Get().(*[]byte)
	*buf = (*buf)[:0] // Reset length but keep capacity
	return *buf
}

// putBuffer returns a byte slice to the pool for reuse
func putBuffer(buf []byte) {
	// Don't pool very large buffers
	if cap(buf) < maxSizeForBufferPool {
		buf = buf[:0] // Reset length but keep capacity
		bufferPool.Put(&buf)
	}
}

// trieHasher is a common hasher for all trie hashers (paths & values).
type trieHasher struct {
	hasher    hash.Hash
	zeroValue []byte
	hashBuf   []byte     // Reusable buffer for hash computations
	hashBufMu sync.Mutex // Protects concurrent access to hasher
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
	th.hashBuf = make([]byte, 0, th.hashSize()) // Pre-allocate hash buffer
	return &th
}

// newNilPathHasher returns a new nil path hasher with the given hash size.
// It is not exported as the validation logic for the ClosestProof automatically handles this case.
func newNilPathHasher(hasherSize int) PathHasher {
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
	th.hashBufMu.Lock()
	defer th.hashBufMu.Unlock()

	th.hasher.Write(data)
	th.hashBuf = th.hasher.Sum(th.hashBuf[:0]) // Reuse buffer, reset length to 0
	th.hasher.Reset()

	// Return a copy to avoid buffer reuse issues
	digest := make([]byte, len(th.hashBuf))
	copy(digest, th.hashBuf)
	return digest
}

// digestLeafNode returns the encoded leaf data as well as its hash (i.e. digest)
func (th *trieHasher) digestLeafNode(path, data []byte) (digest, value []byte) {
	value = encodeLeafNode(path, data)
	digest = th.digestData(value)
	return
}

// digestInnerNode returns the encoded inner node data as well as its hash (i.e. digest)
func (th *trieHasher) digestInnerNode(leftData, rightData []byte) (digest, value []byte) {
	value = encodeInnerNode(leftData, rightData)
	digest = th.digestData(value)
	return
}

// digestSumNode returns the encoded leaf node data as well as its hash (i.e. digest)
func (th *trieHasher) digestSumLeafNode(path, data []byte) (digest, value []byte) {
	value = encodeLeafNode(path, data)
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(value)

	digest = th.digestData(value)
	digest = append(digest, value[firstSumByteIdx:firstCountByteIdx]...)
	digest = append(digest, value[firstCountByteIdx:]...)

	return
}

// digestSumInnerNode returns the encoded inner node data as well as its hash (i.e. digest)
func (th *trieHasher) digestSumInnerNode(leftData, rightData []byte) (digest, value []byte) {
	value = encodeSumInnerNode(leftData, rightData)
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(value)

	digest = th.digestData(value)
	digest = append(digest, value[firstSumByteIdx:firstCountByteIdx]...)
	digest = append(digest, value[firstCountByteIdx:]...)

	return
}

// parseInnerNode returns the encoded left and right nodes
func (th *trieHasher) parseInnerNode(data []byte) (leftData, rightData []byte) {
	leftData = data[len(innerNodePrefix) : th.hashSize()+len(innerNodePrefix)]
	rightData = data[len(innerNodePrefix)+th.hashSize():]
	return
}

// parseSumInnerNode returns the encoded left & right nodes, as well as the sum
// and non-empty leaf count in the sub-trie of the current node.
func (th *trieHasher) parseSumInnerNode(data []byte) (leftData, rightData []byte, sum, count uint64) {
	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(data)

	// Extract the sum from the encoded node data
	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], data[firstSumByteIdx:firstCountByteIdx])
	binary.BigEndian.PutUint64(sumBz[:], sum)

	// Extract the count from the encoded node data
	var countBz [countSizeBytes]byte
	copy(countBz[:], data[firstCountByteIdx:])
	binary.BigEndian.PutUint64(countBz[:], count)

	// Extract the left and right children
	leftIdxLastByte := len(innerNodePrefix) + th.hashSize() + sumSizeBytes + countSizeBytes
	dataValue := data[:firstSumByteIdx]
	leftData = dataValue[len(innerNodePrefix):leftIdxLastByte]
	rightData = dataValue[leftIdxLastByte:]

	return
}

func (th *trieHasher) hashSize() int {
	return th.hasher.Size()
}

func (th *trieHasher) placeholder() []byte {
	return th.zeroValue
}
