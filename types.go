package smt

import (
	"errors"
	"hash"
)

const (
	left    = 0
	sumSize = 8
)

var (
	defaultValue []byte = nil
	defaultSum   [sumSize]byte

	// ErrKeyNotPresent is returned when a key is not present in the tree.
	ErrKeyNotPresent = errors.New("key already empty")
)

// SparseMerkleTree represents a Sparse Merkle tree.
type SparseMerkleTree interface {
	// Update inserts a value into the SMT.
	Update(key, value []byte) error
	// Delete deletes a value from the SMT. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the tree to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, error)
	// Root computes the Merkle root digest.
	Root() []byte
	// Prove computes a Merkle proof of membership or non-membership of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the tree which is
	// closest to the path provided
	ProveClosest([]byte) (closestPath, closestValueHash []byte, proof *SparseMerkleProof, err error)
	// Commit saves the tree's state to its persistent storage.
	Commit() error
	// Spec returns the TreeSpec for the tree
	Spec() *TreeSpec
}

// SparseMerkleSumTree represents a Sparse Merkle sum tree.
type SparseMerkleSumTree interface {
	// Update inserts a value and its sum into the SMST.
	Update(key, value []byte, sum uint64) error
	// Delete deletes a value from the SMST. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the tree to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, uint64, error)
	// Root computes the Merkle root digest.
	Root() []byte
	// Sum computes the total sum of the Merkle tree
	Sum() uint64
	// Prove computes a Merkle proof of membership or non-membership of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the tree which is
	// closest to the path provided
	ProveClosest([]byte) (closestPath, closestValueHash []byte, closestSum uint64, proof *SparseMerkleProof, err error)
	// Commit saves the tree's state to its persistent storage.
	Commit() error
	// Spec returns the TreeSpec for the tree
	Spec() *TreeSpec
}

// TreeSpec specifies the hashing functions used by a tree instance to encode leaf paths
// and stored values, and the corresponding maximum tree depth.
type TreeSpec struct {
	th      treeHasher
	ph      PathHasher
	vh      ValueHasher
	sumTree bool
}

func newTreeSpec(hasher hash.Hash, sumTree bool) TreeSpec {
	spec := TreeSpec{th: *newTreeHasher(hasher)}
	spec.ph = &pathHasher{spec.th}
	spec.vh = &valueHasher{spec.th}
	spec.sumTree = sumTree
	return spec
}

func (spec *TreeSpec) Spec() *TreeSpec { return spec }

func (spec *TreeSpec) depth() int { return spec.ph.PathSize() * 8 }
func (spec *TreeSpec) digestValue(data []byte) []byte {
	if spec.vh == nil {
		return data
	}
	return spec.vh.HashValue(data)
}

func (spec *TreeSpec) serialize(node treeNode) (data []byte) {
	switch n := node.(type) {
	case *lazyNode:
		panic("serialize(lazyNode)")
	case *leafNode:
		return encodeLeaf(n.path, n.valueHash)
	case *innerNode:
		lchild := spec.hashNode(n.leftChild)
		rchild := spec.hashNode(n.rightChild)
		return encodeInner(lchild, rchild)
	case *extensionNode:
		child := spec.hashNode(n.child)
		return encodeExtension(n.pathBounds, n.path, child)
	}
	return nil
}

func (spec *TreeSpec) hashNode(node treeNode) []byte {
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

// sumSerialize serializes a node returning the preimage hash, its sum and any errors encountered
func (spec *TreeSpec) sumSerialize(node treeNode) (preimage []byte) {
	switch n := node.(type) {
	case *lazyNode:
		panic("serialize(lazyNode)")
	case *leafNode:
		return encodeLeaf(n.path, n.valueHash)
	case *innerNode:
		lchild := spec.hashSumNode(n.leftChild)
		rchild := spec.hashSumNode(n.rightChild)
		preimage = encodeSumInner(lchild, rchild)
		return preimage
	case *extensionNode:
		child := spec.hashSumNode(n.child)
		return encodeSumExtension(n.pathBounds, n.path, child)
	}
	return nil
}

// hashSumNode hashes a node returning its digest in the following form
// digest = [node hash]+[8 byte sum]
func (spec *TreeSpec) hashSumNode(node treeNode) []byte {
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
		preimage := spec.sumSerialize(node)
		*cache = spec.th.digest(preimage)
		*cache = append(*cache, preimage[len(preimage)-sumSize:]...)
	}
	return *cache
}
