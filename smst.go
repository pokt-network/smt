package smt

import (
	"bytes"
	"fmt"
	"hash"
	"strconv"
)

var _ treeNode = (*sumLeafNode)(nil)

// sumLeafNode stores data and a full path as well as a hex sum
type sumLeafNode struct {
	path      []byte
	valueHash []byte
	sum       [16]byte // 16 byte hex string representing a uint64
	persisted bool
	digest    []byte
}

// Sparse Merkle Sum Tree object
type SMST struct {
	TreeSpec
	nodes MapStore
	// Last persisted root hash: hash = [32 byte digest]+[16 byte hex sum]
	savedRoot []byte
	// Current state of the tree
	tree treeNode
	// Lists of per-operation orphan sets
	orphans []orphanNodes
}

// TODO: Figure out options
// NewSparseMerkleSumTree returns a pointer to an SMST struct
func NewSparseMerkleSumTree(nodes MapStore, hasher hash.Hash) *SMST {
	return &SMST{
		TreeSpec: newTreeSpec(hasher),
		nodes:    nodes,
	}
}

// ImportSparseMerkleSumTree returns a pointer to an SMST struct with the root hash provided
func ImportSparseMerkleSumTree(nodes MapStore, hasher hash.Hash, root []byte) *SMST {
	smst := NewSparseMerkleSumTree(nodes, hasher)
	smst.tree = &lazyNode{root}
	smst.savedRoot = root
	return smst
}

// Get returns the digest of the value stored at the given key and the sum of the leaf node
func (smst *SMST) Get(key []byte) ([]byte, uint64, error) {
	path := smst.ph.Path(key)
	var leaf *sumLeafNode
	var err error

	for node, depth := &smst.tree, 0; ; depth++ {
		*node, err = smst.resolveLazy(*node)
		if err != nil {
			return nil, 0, err
		}
		if *node == nil {
			break
		}

		if n, ok := (*node).(*sumLeafNode); ok {
			if bytes.Equal(path, n.path) {
				leaf = n
			}
			break
		}

		if ext, ok := (*node).(*extensionNode); ok {
			if _, match := ext.match(path, depth); !match {
				break
			}
			depth += ext.length()
			node = &ext.child
			*node, err = smst.resolveLazy(*node)
			if err != nil {
				return nil, 0, err
			}
		}

		inner := (*node).(*innerNode)
		if getPathBit(path, depth) == left {
			node = &inner.leftChild
		} else {
			node = &inner.rightChild
		}
	}

	if leaf == nil {
		return defaultValue, 0, nil
	}

	sum, err := strconv.ParseUint(string(leaf.sum[:]), 16, 64)
	if err != nil {
		return nil, 0, err
	}

	return leaf.valueHash, uint64(sum), nil
}

// Update sets the value for the given key, to the digest of the provided value
func (smst *SMST) Update(key []byte, value []byte, sum uint64) error {
	path := smst.ph.Path(key)
	valueHash := smst.digestValue(value)
	var hexSum [16]byte
	copy(hexSum[:], fmt.Sprintf("%016x", sum))

	var orphans orphanNodes
	tree, err := smst.update(smst.tree, 0, path, valueHash, hexSum, &orphans)
	if err != nil {
		return err
	}

	smst.tree = tree
	if len(orphans) > 0 {
		smst.orphans = append(smst.orphans, orphans)
	}

	return nil
}

func (smst *SMST) update(
	node treeNode, depth int, path, value []byte, sum [16]byte, orphans *orphanNodes,
) (treeNode, error) {
	node, err := smst.resolveLazy(node)
	if err != nil {
		return node, err
	}

	newLeaf := &sumLeafNode{path: path, valueHash: value, sum: sum}
	// Empty subtree is always replaced by a single leaf
	if node == nil {
		return newLeaf, nil
	}
	if leaf, ok := node.(*leafNode); ok {
		prefixlen := countCommonPrefix(path, leaf.path, depth)
		if prefixlen == smst.depth() { // replace leaf if paths are equal
			smst.addOrphan(orphans, node)
			return newLeaf, nil
		}
		// We insert an "extension" representing multiple single-branch inner nodes
		last := &node
		if depth < prefixlen {
			// note: this keeps path slice alive - GC inefficiency?
			if depth > 0xff {
				panic("invalid depth")
			}
			ext := extensionNode{path: path, pathBounds: [2]byte{byte(depth), byte(prefixlen)}}
			*last = &ext
			last = &ext.child
		}
		if getPathBit(path, prefixlen) == left {
			*last = &innerNode{leftChild: newLeaf, rightChild: leaf}
		} else {
			*last = &innerNode{leftChild: leaf, rightChild: newLeaf}
		}
		return node, nil
	}

	smst.addOrphan(orphans, node)

	if ext, ok := node.(*extensionNode); ok {
		var branch *treeNode
		node, branch, depth = ext.split(path, depth)
		*branch, err = smst.update(*branch, depth, path, value, sum, orphans)
		if err != nil {
			return node, err
		}
		ext.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child *treeNode
	if getPathBit(path, depth) == left {
		child = &inner.leftChild
	} else {
		child = &inner.rightChild
	}

	*child, err = smst.update(*child, depth+1, path, value, sum, orphans)
	if err != nil {
		return node, err
	}

	inner.setDirty()
	return node, nil
}

// Delete removes the node at the path corresponding to the given key
func (smst *SMST) Delete(key []byte) error {
	path := smst.ph.Path(key)
	var orphans orphanNodes

	tree, err := smst.delete(smst.tree, 0, path, &orphans)
	if err != nil {
		return err
	}
	smst.tree = tree

	if len(orphans) > 0 {
		smst.orphans = append(smst.orphans, orphans)
	}

	return nil
}

func (smst *SMST) delete(node treeNode, depth int, path []byte, orphans *orphanNodes,
) (treeNode, error) {
	node, err := smst.resolveLazy(node)
	if err != nil {
		return node, err
	}
	if node == nil {
		return node, ErrKeyNotPresent
	}

	if leaf, ok := node.(*sumLeafNode); ok {
		if !bytes.Equal(path, leaf.path) {
			return node, ErrKeyNotPresent
		}
		smst.addOrphan(orphans, node)
		return nil, nil
	}
	smst.addOrphan(orphans, node)

	if ext, ok := node.(*extensionNode); ok {
		if _, match := ext.match(path, depth); !match {
			return node, ErrKeyNotPresent
		}
		ext.child, err = smst.delete(ext.child, depth+ext.length(), path, orphans)
		if err != nil {
			return node, err
		}
		switch n := ext.child.(type) {
		case *sumLeafNode:
			return n, nil
		case *extensionNode:
			// Join this extension with the child
			smst.addOrphan(orphans, n)
			n.pathBounds[0] = ext.pathBounds[0]
			node = n
		}
		ext.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child, sib *treeNode
	if getPathBit(path, depth) == left {
		child, sib = &inner.leftChild, &inner.rightChild
	} else {
		child, sib = &inner.rightChild, &inner.leftChild
	}

	*child, err = smst.delete(*child, depth+1, path, orphans)
	if err != nil {
		return node, err
	}

	*sib, err = smst.resolveLazy(*sib)
	if err != nil {
		return node, err
	}

	// Handle replacement of this node, depending on the new child states.
	// Note that inner nodes exist at a fixed depth, and can't be moved.
	children := [2]*treeNode{child, sib}
	for i := 0; i < 2; i++ {
		if *children[i] == nil {
			switch n := (*children[1-i]).(type) {
			case *leafNode:
				return n, nil
			case *extensionNode:
				// "Absorb" this node into the extension by prepending
				smst.addOrphan(orphans, n)
				n.pathBounds[0]--
				n.setDirty()
				return n, nil
			}
		}
	}

	inner.setDirty()

	return node, nil
}

// Commit persists all dirty nodes in the tree, deletes all orphaned
// nodes from the database and then computes and saves the root hash
func (smst *SMST) Commit() (err error) {
	// All orphans are persisted and have cached digests, so we don't need to check for null
	for _, orphans := range smst.orphans {
		for _, hash := range orphans {
			if err = smst.nodes.Delete(hash); err != nil {
				return
			}
		}
	}
	smst.orphans = nil
	if err = smst.commit(smst.tree); err != nil {
		return
	}
	smst.savedRoot = smst.Root()
	return
}

func (smst *SMST) commit(node treeNode) error {
	if node != nil && node.Persisted() {
		return nil
	}

	switch n := node.(type) {
	case *sumLeafNode:
		n.persisted = true
	case *innerNode:
		n.persisted = true
		if err := smst.commit(n.leftChild); err != nil {
			return err
		}
		if err := smst.commit(n.rightChild); err != nil {
			return err
		}
	case *extensionNode:
		n.persisted = true
		if err := smst.commit(n.child); err != nil {
			return err
		}
	default:
		return nil
	}

	preimage, err := smst.sumSerialize(node)
	if err != nil {
		return err
	}

	return smst.nodes.Set(smst.hashSumNode(node), preimage)
}

func (smst *SMST) Root() []byte {
	return smst.hashSumNode(smst.tree)
}

// Sum returns the uint64 sum of the entire tree
func (smst *SMST) Sum() (uint64, error) {
	var hexSum [16]byte
	digest := smst.hashSumNode(smst.tree)
	copy(hexSum[:], digest[len(digest)-16:])
	sum, err := strconv.ParseUint(string(hexSum[:]), 16, 64)
	if err != nil {
		return 0, err
	}
	return sum, nil
}

// resolves a stub into a cached node
func (smst *SMST) resolveLazy(node treeNode) (treeNode, error) {
	stub, ok := node.(*lazyNode)
	if !ok {
		return node, nil
	}
	resolver := func(hash []byte) (treeNode, error) {
		return &lazyNode{hash}, nil
	}
	ret, err := smst.resolve(stub.digest, resolver)
	if err != nil {
		return node, err
	}
	return ret, nil
}

func (smst *SMST) resolve(hash []byte, resolver func([]byte) (treeNode, error),
) (ret treeNode, err error) {
	if bytes.Equal(smst.th.placeholder(), hash) {
		return
	}
	data, err := smst.nodes.Get(hash)
	if err != nil {
		return nil, err
	}
	if isLeaf(data) {
		var sum [16]byte
		copy(sum[:], data[len(data)-16:])
		leaf := sumLeafNode{persisted: true, digest: hash, sum: sum}
		leaf.path, leaf.valueHash, leaf.sum = parseSumLeaf(data, smst.ph)
		return &leaf, nil
	}
	if isExtension(data) {
		ext := extensionNode{persisted: true, digest: hash}
		pathBounds, path, childHash := parseExtension(data, smst.ph)
		ext.path = path
		copy(ext.pathBounds[:], pathBounds)
		ext.child, err = resolver(childHash)
		if err != nil {
			return
		}
		return &ext, nil
	}
	leftHash, rightHash := smst.th.parseSumNode(data)
	inner := innerNode{persisted: true, digest: hash}
	inner.leftChild, err = resolver(leftHash)
	if err != nil {
		return
	}
	inner.rightChild, err = resolver(rightHash)
	if err != nil {
		return
	}
	return &inner, nil
}

func (smst *SMST) addOrphan(orphans *[][]byte, node treeNode) {
	if node.Persisted() {
		*orphans = append(*orphans, node.CachedDigest())
	}
}

func (node *sumLeafNode) Persisted() bool      { return node.persisted }
func (node *sumLeafNode) CachedDigest() []byte { return node.digest }
