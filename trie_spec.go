package smt

import (
	"encoding/binary"
	"hash"
)

// TrieSpec specifies the hashing functions used by a trie instance to encode
// leaf paths and stored values, and the corresponding maximum trie depth.
type TrieSpec struct {
	th      trieHasher
	ph      PathHasher
	vh      ValueHasher
	sumTrie bool
}

// newTrieSpec returns a new TrieSpec with the given hasher and sumTrie flag
func newTrieSpec(hasher hash.Hash, sumTrie bool) TrieSpec {
	spec := TrieSpec{th: *NewTrieHasher(hasher)}
	spec.ph = &pathHasher{spec.th}
	spec.vh = &valueHasher{spec.th}
	spec.sumTrie = sumTrie
	return spec
}

// Spec returns the TrieSpec associated with the given trie
func (spec *TrieSpec) Spec() *TrieSpec {
	return spec
}

// placeholder returns the default placeholder value depending on the trie type
func (spec *TrieSpec) placeholder() []byte {
	if spec.sumTrie {
		placeholder := spec.th.placeholder()
		placeholder = append(placeholder, defaultEmptySum[:]...)
		placeholder = append(placeholder, defaultEmptyCount[:]...)
		return placeholder
	}
	return spec.th.placeholder()
}

// hashSize returns the hash size depending on the trie type
func (spec *TrieSpec) hashSize() int {
	if spec.sumTrie {
		return spec.th.hashSize() + sumSizeBytes + countSizeBytes
	}
	return spec.th.hashSize()
}

// digestLeaf returns the hash and preimage of a leaf node depending on the trie type
func (spec *TrieSpec) digestLeaf(path, value []byte) ([]byte, []byte) {
	if spec.sumTrie {
		return spec.th.digestSumLeafNode(path, value)
	}
	return spec.th.digestLeafNode(path, value)
}

// digestNode returns the hash and preimage of a node depending on the trie type
func (spec *TrieSpec) digestInnerNode(left, right []byte) ([]byte, []byte) {
	if spec.sumTrie {
		return spec.th.digestSumInnerNode(left, right)
	}
	return spec.th.digestInnerNode(left, right)
}

// digest hashes a node depending on the trie type
func (spec *TrieSpec) digest(node trieNode) []byte {
	if spec.sumTrie {
		return spec.digestSumNode(node)
	}
	return spec.digestNode(node)
}

// encode serializes a node depending on the trie type
func (spec *TrieSpec) encode(node trieNode) []byte {
	if spec.sumTrie {
		return spec.encodeSumNode(node)
	}
	return spec.encodeNode(node)
}

// hashPreimage hashes the serialised data provided depending on the trie type
func (spec *TrieSpec) hashPreimage(data []byte) []byte {
	if spec.sumTrie {
		return spec.hashSumSerialization(data)
	}
	return spec.hashSerialization(data)
}

// Used for verification of serialized proof data
func (spec *TrieSpec) hashSerialization(data []byte) []byte {
	if isExtNode(data) {
		pathBounds, path, childHash := spec.parseExtNode(data)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return spec.digestNode(&ext)
	}
	return spec.th.digestData(data)
}

// Used for verification of serialized proof data for sum trie nodes
func (spec *TrieSpec) hashSumSerialization(data []byte) []byte {
	if isExtNode(data) {
		pathBounds, path, childHash, _, _ := spec.parseSumExtNode(data)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return spec.digestSumNode(&ext)
	}

	firstSumByteIdx, firstCountByteIdx := GetFirstMetaByteIdx(data)

	digest := spec.th.digestData(data)
	digest = append(digest, data[firstSumByteIdx:firstCountByteIdx]...)
	digest = append(digest, data[firstCountByteIdx:]...)
	return digest
}

// depth returns the maximum depth of the trie.
// Since this tree is a binary tree, the depth is the number of bits in the path
// TODO_UPNEXT(@Olshansk):: Try to understand why we're not taking the log of the output
func (spec *TrieSpec) depth() int {
	return spec.ph.PathSize() * 8 // path size is in bytes so multiply by 8 to get num bits
}

// valueHash returns the hash of a value, or the value itself if no value hasher is specified.
func (spec *TrieSpec) valueHash(value []byte) []byte {
	if spec.vh == nil {
		return value
	}
	return spec.vh.HashValue(value)
}

// encodeNode serializes a node into a byte slice
func (spec *TrieSpec) encodeNode(node trieNode) []byte {
	switch n := node.(type) {
	case *lazyNode:
		panic("Encoding a lazyNode is not supported")
	case *leafNode:
		return encodeLeafNode(n.path, n.valueHash)
	case *innerNode:
		leftChild := spec.digestNode(n.leftChild)
		rightChild := spec.digestNode(n.rightChild)
		return encodeInnerNode(leftChild, rightChild)
	case *extensionNode:
		child := spec.digestNode(n.child)
		return encodeExtensionNode(n.pathBounds, n.path, child)
	default:
		panic("Unknown node type")
	}
}

// digestNode hashes a node and returns its digest
func (spec *TrieSpec) digestNode(node trieNode) []byte {
	if node == nil {
		return spec.th.placeholder()
	}

	var cachedDigest *[]byte
	switch n := node.(type) {
	case *lazyNode:
		return n.digest
	case *leafNode:
		cachedDigest = &n.digest
	case *innerNode:
		cachedDigest = &n.digest
	case *extensionNode:
		if n.digest == nil {
			n.digest = spec.digestNode(n.expand())
		}
		return n.digest
	}
	if *cachedDigest == nil {
		*cachedDigest = spec.th.digestData(spec.encodeNode(node))
	}
	return *cachedDigest
}

// encodeSumNode serializes a sum node and returns the preImage hash.
func (spec *TrieSpec) encodeSumNode(node trieNode) (preImage []byte) {
	switch n := node.(type) {
	case *lazyNode:
		panic("encodeSumNode(lazyNode)")
	case *leafNode:
		return encodeLeafNode(n.path, n.valueHash)
	case *innerNode:
		leftChild := spec.digestSumNode(n.leftChild)
		rightChild := spec.digestSumNode(n.rightChild)
		return encodeSumInnerNode(leftChild, rightChild)
	case *extensionNode:
		child := spec.digestSumNode(n.child)
		return encodeSumExtensionNode(n.pathBounds, n.path, child)
	}
	return nil
}

// digestSumNode hashes a sum node returning its digest in the following form: [node hash]+[8 byte sum]+[8 byte count]
func (spec *TrieSpec) digestSumNode(node trieNode) []byte {
	if node == nil {
		return spec.placeholder()
	}
	var cache *[]byte
	switch n := node.(type) {
	case *lazyNode:
		return n.digest
	case *leafNode:
		cache = &n.digest
	case *innerNode:
		cache = &n.digest
	case *extensionNode:
		if n.digest == nil {
			n.digest = spec.digestSumNode(n.expand())
		}
		return n.digest
	}
	if *cache == nil {
		preImage := spec.encodeSumNode(node)
		firstSumByteIdx, firstCountByteIdx := GetFirstMetaByteIdx(preImage)
		*cache = spec.th.digestData(preImage)
		*cache = append(*cache, preImage[firstSumByteIdx:firstCountByteIdx]...)
		*cache = append(*cache, preImage[firstCountByteIdx:]...)
	}
	return *cache
}

// parseLeafNode parses a leafNode into its components
func (spec *TrieSpec) parseLeafNode(data []byte) (path, value []byte) {
	// panics if not a leaf node
	checkPrefix(data, leafNodePrefix)

	path = data[prefixLen : prefixLen+spec.ph.PathSize()]
	value = data[prefixLen+spec.ph.PathSize():]
	return
}

// parseExtNode parses an extNode into its components
func (spec *TrieSpec) parseExtNode(data []byte) (pathBounds, path, childData []byte) {
	// panics if not an extension node
	checkPrefix(data, extNodePrefix)

	// +2 represents the length of the pathBounds
	pathBounds = data[prefixLen : prefixLen+2]
	path = data[prefixLen+2 : prefixLen+2+spec.ph.PathSize()]
	childData = data[prefixLen+2+spec.ph.PathSize():]
	return
}

// parseSumLeafNode parses a leafNode and returns its weight as well
// // nolint: unused
func (spec *TrieSpec) parseSumLeafNode(data []byte) (path, value []byte, weight, count uint64) {
	// panics if not a leaf node
	checkPrefix(data, leafNodePrefix)

	path = data[prefixLen : prefixLen+spec.ph.PathSize()]
	value = data[prefixLen+spec.ph.PathSize():]

	firstSumByteIdx, firstCountByteIdx := GetFirstMetaByteIdx(data)

	// Extract the sum from the encoded node data
	var weightBz [sumSizeBytes]byte
	copy(weightBz[:], data[firstSumByteIdx:firstCountByteIdx])
	binary.BigEndian.PutUint64(weightBz[:], weight)

	// Extract the count from the encoded node data
	var countBz [countSizeBytes]byte
	copy(countBz[:], value[firstCountByteIdx:])
	binary.BigEndian.PutUint64(countBz[:], count)
	if count != 1 {
		panic("count for leaf node should always be 1")
	}

	return
}

// parseSumExtNode parses the pathBounds, path, child data and sum from the encoded extension node data
func (spec *TrieSpec) parseSumExtNode(data []byte) (pathBounds, path, childData []byte, sum, count uint64) {
	// panics if not an extension node
	checkPrefix(data, extNodePrefix)

	firstSumByteIdx, firstCountByteIdx := GetFirstMetaByteIdx(data)

	// Extract the sum from the encoded node data
	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], data[firstSumByteIdx:firstCountByteIdx])
	binary.BigEndian.PutUint64(sumBz[:], sum)

	// Extract the count from the encoded node data
	var countBz [countSizeBytes]byte
	copy(countBz[:], data[firstCountByteIdx:])
	binary.BigEndian.PutUint64(countBz[:], count)

	// +2 represents the length of the pathBounds
	pathBounds = data[prefixLen : prefixLen+2]
	path = data[prefixLen+2 : prefixLen+2+spec.ph.PathSize()]
	childData = data[prefixLen+2+spec.ph.PathSize() : firstSumByteIdx]
	return
}
