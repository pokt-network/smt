package smt

import (
	"hash"
)

// TODO_DISCUSS_IN_THIS_PR_IMPROVEMENTS:
// 1. Should we rename all instances of digest to hash?
// 2. Should we introduce a shared interface between SparseMerkleTrie and SparseMerkleSumTrie?
// 3. Should we rename Commit to FlushToDisk?

const (
	// The bit value use to distinguish an inner nodes left child and right child
	leftChildBit = 0
)

var (
	// defaultEmptyValue is the default value for a leaf node
	defaultEmptyValue []byte
	// defaultEmptySum is the default sum value for a leaf node
	defaultEmptySum [sumSizeBits]byte
)

// MerkleRoot is a type alias for a byte slice returned from the Root method
type MerkleRoot []byte

// A high-level interface that captures the behaviour of all types of nodes
type trieNode interface {
	// Persisted returns a boolean to determine whether or not the node
	// has been persisted to disk or only held in memory.
	// It can be used skip unnecessary iops if already persisted
	Persisted() bool

	// The digest of the node, returning a cached value if available.
	CachedDigest() []byte
}

// SparseMerkleTrie represents a Sparse Merkle Trie.
type SparseMerkleTrie interface {
	// Update inserts a value into the SMT.
	Update(key, value []byte) error
	// Delete deletes a value from the SMT. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the trie to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, error)
	// Root computes the Merkle root digest.
	Root() MerkleRoot
	// Prove computes a Merkle proof of inclusion or exclusion of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the trie
	// which is closest to the path provided. It will search for the key with
	// the longest common prefix before finding the key with the most common
	// bits as the path provided.
	ProveClosest([]byte) (*SparseMerkleClosestProof, error)
	// Commit saves the trie's state to its persistent storage.
	Commit() error
	// Spec returns the TrieSpec for the trie
	Spec() *TrieSpec
}

// SparseMerkleSumTrie represents a Sparse Merkle Sum Trie.
type SparseMerkleSumTrie interface {
	// Update inserts a value and its sum into the SMST.
	Update(key, value []byte, sum uint64) error
	// Delete deletes a value from the SMST. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the trie to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, uint64, error)
	// Root computes the Merkle root digest.
	Root() MerkleRoot
	// Sum computes the total sum of the Merkle trie
	Sum() uint64
	// Prove computes a Merkle proof of inclusion or exclusion of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the trie
	// which is closest to the path provided. It will search for the key with
	// the longest common prefix before finding the key with the most common
	// bits as the path provided.
	ProveClosest([]byte) (*SparseMerkleClosestProof, error)
	// Commit saves the trie's state to its persistent storage.
	Commit() error
	// Spec returns the TrieSpec for the trie
	Spec() *TrieSpec
}

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
	spec := TrieSpec{th: *newTrieHasher(hasher)}
	spec.ph = &pathHasher{spec.th}
	spec.vh = &valueHasher{spec.th}
	spec.sumTrie = sumTrie
	return spec
}

// Spec returns the TrieSpec associated with the given trie
func (spec *TrieSpec) Spec() *TrieSpec {
	return spec
}

// depth returns the maximum depth of the trie.
// Since this tree is a binary tree, the depth is the number of bits in the path
func (spec *TrieSpec) depth() int {
	return spec.ph.PathSize() * 8 // path size is in bytes so multiply by 8 to get num bits
}

func (spec *TrieSpec) digestValue(data []byte) []byte {
	if spec.vh == nil {
		return data
	}
	return spec.vh.HashValue(data)
}

func (spec *TrieSpec) serialize(node trieNode) (data []byte) {
	switch n := node.(type) {
	case *lazyNode:
		panic("serialize(lazyNode)")
	case *leafNode:
		return encodeLeafNode(n.path, n.valueHash)
	case *innerNode:
		lchild := spec.hashNode(n.leftChild)
		rchild := spec.hashNode(n.rightChild)
		return encodeInnerNode(lchild, rchild)
	case *extensionNode:
		child := spec.hashNode(n.child)
		return encodeExtensionNode(n.pathBounds, n.path, child)
	}
	return nil
}

func (spec *TrieSpec) hashNode(node trieNode) []byte {
	if node == nil {
		return spec.th.placeholder()
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
			n.digest = spec.hashNode(n.expand())
		}
		return n.digest
	}
	if *cache == nil {
		*cache = spec.th.digest(spec.serialize(node))
	}
	return *cache
}

// sumSerialize serializes a node returning the preimage hash, its sum and any
// errors encountered
func (spec *TrieSpec) sumSerialize(node trieNode) (preImage []byte) {
	switch n := node.(type) {
	case *lazyNode:
		panic("serialize(lazyNode)")
	case *leafNode:
		return encodeLeafNode(n.path, n.valueHash)
	case *innerNode:
		leftChild := spec.hashSumNode(n.leftChild)
		rightChild := spec.hashSumNode(n.rightChild)
		preImage = encodeSumInnerNode(leftChild, rightChild)
		return preImage
	case *extensionNode:
		child := spec.hashSumNode(n.child)
		return encodeSumExtensionNode(n.pathBounds, n.path, child)
	}
	return nil
}

// hashSumNode hashes a node returning its digest in the following form
// digest = [node hash]+[8 byte sum]
func (spec *TrieSpec) hashSumNode(node trieNode) []byte {
	if node == nil {
		return placeholder(spec)
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
			n.digest = spec.hashSumNode(n.expand())
		}
		return n.digest
	}
	if *cache == nil {
		preImage := spec.sumSerialize(node)
		*cache = spec.th.digest(preImage)
		*cache = append(*cache, preImage[len(preImage)-sumSizeBits:]...)
	}
	return *cache
}
