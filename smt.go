package smt

import (
	"bytes"
	"hash"

	"github.com/pokt-network/smt/kvstore"
)

// Ensure the `SMT` struct implements the `SparseMerkleTrie` interface
var _ SparseMerkleTrie = (*SMT)(nil)

// SMT is a Sparse Merkle Trie object that implements the SparseMerkleTrie interface
type SMT struct {
	TrieSpec
	// Backing key-value store for the node
	nodes kvstore.MapStore
	// Last persisted root hash
	rootHash []byte
	// The current view of the SMT
	root trieNode
	// Lists of per-operation orphan sets
	orphans []orphanNodes
}

// Hashes of persisted nodes deleted from trie
type orphanNodes = [][]byte

// NewSparseMerkleTrie returns a new pointer to an SMT struct, and applies any
// options provided
func NewSparseMerkleTrie(
	nodes kvstore.MapStore,
	hasher hash.Hash,
	options ...TrieSpecOption,
) *SMT {
	smt := SMT{
		TrieSpec: newTrieSpec(hasher, false),
		nodes:    nodes,
	}
	for _, option := range options {
		option(&smt.TrieSpec)
	}
	return &smt
}

// ImportSparseMerkleTrie returns a pointer to an SMT struct with the provided
// root hash
func ImportSparseMerkleTrie(
	nodes kvstore.MapStore,
	hasher hash.Hash,
	root []byte,
	options ...TrieSpecOption,
) *SMT {
	smt := NewSparseMerkleTrie(nodes, hasher, options...)
	smt.root = &lazyNode{root}
	smt.rootHash = root
	return smt
}

// Root returns the root hash of the trie
func (smt *SMT) Root() MerkleRoot {
	return smt.digest(smt.root)
}

// Get returns the hash (i.e. digest) of the leaf value stored at the given key
func (smt *SMT) Get(key []byte) ([]byte, error) {
	path := smt.ph.Path(key)
	// The leaf node whose value will be returned
	var leaf *leafNode
	var err error

	// Loop throughout the entire trie to find the corresponding leaf for the
	// given key.
	for currNode, depth := &smt.root, 0; ; depth++ {
		*currNode, err = smt.resolveLazy(*currNode)
		if err != nil {
			return nil, err
		}
		if *currNode == nil {
			break
		}
		if n, ok := (*currNode).(*leafNode); ok {
			if bytes.Equal(path, n.path) {
				leaf = n
			}
			break
		}
		if extNode, ok := (*currNode).(*extensionNode); ok {
			if _, fullMatch := extNode.boundsMatch(path, depth); !fullMatch {
				break
			}
			depth += extNode.length()
			currNode = &extNode.child
			*currNode, err = smt.resolveLazy(*currNode)
			if err != nil {
				return nil, err
			}
		}
		inner := (*currNode).(*innerNode)
		if getPathBit(path, depth) == leftChildBit {
			currNode = &inner.leftChild
		} else {
			currNode = &inner.rightChild
		}
	}
	if leaf == nil {
		return defaultEmptyValue, nil
	}
	return leaf.valueHash, nil
}

// Update inserts the `value` for the given `key` into the SMT
func (smt *SMT) Update(key, value []byte) error {
	// Expand the key into a path by computing its digest
	path := smt.ph.Path(key)

	// Convert the value into a hash by computing its digest
	valueHash := smt.valueHash(value)

	// Update the trie with the new key-value pair
	var orphans orphanNodes
	// Compute the new root by inserting (path, valueHash) starting
	newRoot, err := smt.update(smt.root, 0, path, valueHash, &orphans)
	if err != nil {
		return err
	}
	smt.root = newRoot
	if len(orphans) > 0 {
		smt.orphans = append(smt.orphans, orphans)
	}
	return nil
}

// Internal helper to the `Update` method
func (smt *SMT) update(
	node trieNode,
	depth int,
	path, value []byte,
	orphans *orphanNodes,
) (trieNode, error) {
	node, err := smt.resolveLazy(node)
	if err != nil {
		return node, err
	}

	newLeaf := &leafNode{path: path, valueHash: value}
	// Empty subtrie is always replaced by a single leaf
	if node == nil {
		return newLeaf, nil
	}
	if leaf, ok := node.(*leafNode); ok {
		prefixLen := countCommonPrefixBits(path, leaf.path, depth)
		// replace leaf if paths are equal
		if prefixLen == smt.depth() {
			smt.addOrphan(orphans, node)
			return newLeaf, nil
		}
		// We insert an "extension" representing multiple single-branch inner nodes
		last := &node
		if depth < prefixLen {
			// note: this keeps path slice alive - GC inefficiency?
			if depth > 0xff {
				panic("invalid depth")
			}
			ext := extensionNode{
				path: path,
				pathBounds: [2]byte{
					byte(depth),
					byte(prefixLen),
				},
			}
			*last = &ext
			last = &ext.child
		}
		if getPathBit(path, prefixLen) == leftChildBit {
			*last = &innerNode{
				leftChild:  newLeaf,
				rightChild: leaf,
			}
		} else {
			*last = &innerNode{
				leftChild:  leaf,
				rightChild: newLeaf,
			}
		}
		return node, nil
	}

	smt.addOrphan(orphans, node)

	if extNode, ok := node.(*extensionNode); ok {
		var branch *trieNode
		node, branch, depth = extNode.split(path)
		*branch, err = smt.update(*branch, depth, path, value, orphans)
		if err != nil {
			return node, err
		}
		extNode.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child *trieNode
	if getPathBit(path, depth) == leftChildBit {
		child = &inner.leftChild
	} else {
		child = &inner.rightChild
	}
	*child, err = smt.update(*child, depth+1, path, value, orphans)
	if err != nil {
		return node, err
	}
	inner.setDirty()
	return node, nil
}

// Delete removes the node at the path corresponding to the given key
func (smt *SMT) Delete(key []byte) error {
	path := smt.ph.Path(key)
	var orphans orphanNodes
	trie, err := smt.delete(smt.root, 0, path, &orphans)
	if err != nil {
		return err
	}
	smt.root = trie
	if len(orphans) > 0 {
		smt.orphans = append(smt.orphans, orphans)
	}
	return nil
}

func (smt *SMT) delete(node trieNode, depth int, path []byte, orphans *orphanNodes,
) (trieNode, error) {
	node, err := smt.resolveLazy(node)
	if err != nil {
		return node, err
	}

	if node == nil {
		return node, ErrKeyNotFound
	}
	if leaf, ok := node.(*leafNode); ok {
		if !bytes.Equal(path, leaf.path) {
			return node, ErrKeyNotFound
		}
		smt.addOrphan(orphans, node)
		return nil, nil
	}

	smt.addOrphan(orphans, node)

	if extNode, ok := node.(*extensionNode); ok {
		if _, fullMatch := extNode.boundsMatch(path, depth); !fullMatch {
			return node, ErrKeyNotFound
		}
		extNode.child, err = smt.delete(extNode.child, depth+extNode.length(), path, orphans)
		if err != nil {
			return node, err
		}
		switch n := extNode.child.(type) {
		case *leafNode:
			return n, nil
		case *extensionNode:
			// Join this extension with the child
			smt.addOrphan(orphans, n)
			n.pathBounds[0] = extNode.pathBounds[0]
			node = n
		}
		extNode.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child, sib *trieNode
	if getPathBit(path, depth) == leftChildBit {
		child, sib = &inner.leftChild, &inner.rightChild
	} else {
		child, sib = &inner.rightChild, &inner.leftChild
	}
	*child, err = smt.delete(*child, depth+1, path, orphans)
	if err != nil {
		return node, err
	}
	*sib, err = smt.resolveLazy(*sib)
	if err != nil {
		return node, err
	}
	// Handle replacement of this node, depending on the new child states.
	// Note that inner nodes exist at a fixed depth, and can't be moved.
	children := [2]*trieNode{child, sib}
	for i := 0; i < 2; i++ {
		if *children[i] == nil {
			switch n := (*children[1-i]).(type) {
			case *leafNode:
				return n, nil
			case *extensionNode:
				// "Absorb" this node into the extension by prepending
				smt.addOrphan(orphans, n)
				n.pathBounds[0]--
				n.setDirty()
				return n, nil
			}
		}
	}
	inner.setDirty()
	return node, nil
}

// Prove generates a SparseMerkleProof for the given key
func (smt *SMT) Prove(key []byte) (proof *SparseMerkleProof, err error) {
	path := smt.ph.Path(key)
	var siblings []trieNode
	var sib trieNode

	node := smt.root
	for depth := 0; depth < smt.depth(); depth++ {
		node, err = smt.resolveLazy(node)
		if err != nil {
			return nil, err
		}
		if node == nil {
			break
		}
		if _, ok := node.(*leafNode); ok {
			break
		}
		if extNode, ok := node.(*extensionNode); ok {
			matchLen, fullMatch := extNode.boundsMatch(path, depth)
			if fullMatch {
				for i := 0; i < matchLen; i++ {
					siblings = append(siblings, nil)
				}
				depth += matchLen
				node = extNode.child
				node, err = smt.resolveLazy(node)
				if err != nil {
					return nil, err
				}
			} else {
				node = extNode.expand()
			}
		}
		inner := node.(*innerNode)
		if getPathBit(path, depth) == leftChildBit {
			node, sib = inner.leftChild, inner.rightChild
		} else {
			node, sib = inner.rightChild, inner.leftChild
		}
		siblings = append(siblings, sib)
	}

	// Deal with non-membership proofs. If there is no leaf on this path,
	// we do not need to add anything else to the proof.
	var leafData []byte
	if node != nil {
		leaf := node.(*leafNode)
		if !bytes.Equal(leaf.path, path) {
			// This is a non-membership proof that involves showing a different leaf.
			// Add the leaf data to the proof.
			leafData = encodeLeafNode(leaf.path, leaf.valueHash)
		}
	}
	// Hash siblings from bottom up.
	var sideNodes [][]byte
	for i := range siblings {
		var sideNode []byte
		sibling := siblings[len(siblings)-i-1]
		sideNode = smt.digest(sibling)
		sideNodes = append(sideNodes, sideNode)
	}

	proof = &SparseMerkleProof{
		SideNodes:             sideNodes,
		NonMembershipLeafData: leafData,
	}
	if sib != nil {
		sib, err = smt.resolveLazy(sib)
		if err != nil {
			return nil, err
		}
		proof.SiblingData = smt.encode(sib)
	}
	return proof, nil
}

// ProveClosest generates a SparseMerkleProof of inclusion for the first
// key with the most common bits as the path provided.
//
// This method will follow the path provided until it hits a leaf node and then
// exit. If the leaf is along the path it will produce an inclusion proof for
// the key (and return the key-value internal pair) as they share a common
// prefix. If however, during the trie traversal according to the path, a nil
// node is encountered, the traversal backsteps and flips the path bit for that
// depth (ie tries left if it tried right and vice versa). This guarantees that
// a proof of inclusion is found that has the most common bits with the path
// provided, biased to the longest common prefix
func (smt *SMT) ProveClosest(path []byte) (
	proof *SparseMerkleClosestProof, // proof of the key-value pair found
	err error, // the error value encountered
) {
	workingPath := make([]byte, len(path))
	copy(workingPath, path)
	var siblings []trieNode
	var sib trieNode
	var parent trieNode
	// depthDelta is used to track the depth increase when traversing down the trie
	// it is used when back-stepping to go back to the correct depth in the path
	// if we hit a nil node during trie traversal
	var depthDelta int
	proof = &SparseMerkleClosestProof{
		Path:        path,
		FlippedBits: make([]int, 0),
	}

	node := smt.root
	depth := 0
	// continuously traverse the trie until we hit a leaf node
	for depth < smt.depth() {
		// save current node information as "parent" info
		if node != nil {
			parent = node
		}
		// resolve current node
		node, err = smt.resolveLazy(node)
		if err != nil {
			return nil, err
		}
		if node != nil {
			// reset depthDelta if node is non nil
			depthDelta = 0
		} else {
			// if we hit a nil node we backstep to the parent node and flip the
			// path bit at the parent depth and select the other child
			node, err = smt.resolveLazy(parent)
			if err != nil {
				return nil, err
			}
			// trim the last sibling node added as it is no longer relevant
			// due to back-stepping we are now going to traverse to the
			// most recent sibling, including it here would result in an
			// incorrect root hash when calculated
			if len(siblings) > 0 {
				siblings = siblings[:len(siblings)-1]
			}
			depth -= depthDelta
			// flip the path bit at the parent depth
			flipPathBit(workingPath, depth)
			proof.FlippedBits = append(proof.FlippedBits, depth)
		}
		// end traversal when we hit a leaf node
		if _, ok := node.(*leafNode); ok {
			proof.Depth = depth
			break
		}
		if extNode, ok := node.(*extensionNode); ok {
			matchLen, fullMatch := extNode.boundsMatch(workingPath, depth)
			// workingPath from depth to end of extension node's path bounds
			// is a perfect match
			if !fullMatch {
				node = extNode.expand()
			} else {
				// extension nodes represent a singly linked list of inner nodes
				// add nil siblings to represent the empty neighbours
				for i := 0; i < matchLen; i++ {
					siblings = append(siblings, nil)
				}
				depth += matchLen
				depthDelta += matchLen
				node = extNode.child
				node, err = smt.resolveLazy(node)
				if err != nil {
					return nil, err
				}
			}
		}
		inner, ok := node.(*innerNode)
		if !ok { // this can only happen for an empty trie
			proof.Depth = depth
			break
		}
		if getPathBit(workingPath, depth) == leftChildBit {
			node, sib = inner.leftChild, inner.rightChild
		} else {
			node, sib = inner.rightChild, inner.leftChild
		}
		siblings = append(siblings, sib)
		depth++
		depthDelta++
	}

	// Retrieve the closest path and value hash if found
	if node == nil { // trie was empty
		proof.ClosestPath, proof.ClosestValueHash = smt.placeholder(), nil
		proof.ClosestProof = &SparseMerkleProof{}
		return proof, nil
	}
	leaf, ok := node.(*leafNode)
	if !ok {
		// if no leaf was found and the trie is not empty something went wrong
		panic("expected leaf node")
	}
	proof.ClosestPath, proof.ClosestValueHash = leaf.path, leaf.valueHash
	// Hash siblings from bottom up.
	var sideNodes [][]byte
	for i := range siblings {
		var sideNode []byte
		sibling := siblings[len(siblings)-i-1]
		sideNode = smt.digest(sibling)
		sideNodes = append(sideNodes, sideNode)
	}
	proof.ClosestProof = &SparseMerkleProof{
		SideNodes: sideNodes,
	}
	if sib != nil {
		sib, err = smt.resolveLazy(sib)
		if err != nil {
			return nil, err
		}
		proof.ClosestProof.SiblingData = smt.encode(sib)
	}

	return proof, nil
}

// resolveLazy resolves a lazy note into a cached node depending on the tree type
func (smt *SMT) resolveLazy(node trieNode) (trieNode, error) {
	stub, ok := node.(*lazyNode)
	if !ok {
		return node, nil
	}
	if smt.sumTrie {
		return smt.resolveSumNode(stub.digest)
	}
	return smt.resolveNode(stub.digest)
}

// resolveNode returns a trieNode (inner, leaf, or extension) based on what they
// keyHash points to.
func (smt *SMT) resolveNode(digest []byte) (trieNode, error) {
	// Check if the keyHash is the empty zero value of an empty subtree
	if bytes.Equal(smt.placeholder(), digest) {
		return nil, nil
	}

	// Retrieve the encoded noe data
	data, err := smt.nodes.Get(digest)
	if err != nil {
		return nil, err
	}

	// Return the appropriate node type based on the first byte of the data
	if isLeafNode(data) {
		path, valueHash := smt.parseLeafNode(data)
		return &leafNode{
			path:      path,
			valueHash: valueHash,
			persisted: true,
			digest:    digest,
		}, nil
	} else if isExtNode(data) {
		pathBounds, path, childData := smt.parseExtNode(data)
		return &extensionNode{
			path:       path,
			pathBounds: [2]byte(pathBounds),
			child:      &lazyNode{childData},
			persisted:  true,
			digest:     digest,
		}, nil
	} else if isInnerNode(data) {
		leftData, rightData := smt.th.parseInnerNode(data)
		return &innerNode{
			leftChild:  &lazyNode{leftData},
			rightChild: &lazyNode{rightData},
			persisted:  true,
			digest:     digest,
		}, nil
	} else {
		panic("invalid node type")
	}
}

// resolveNode returns a trieNode (inner, leaf, or extension) based on what they
// keyHash points to.
func (smt *SMT) resolveSumNode(digest []byte) (trieNode, error) {
	// Check if the keyHash is the empty zero value of an empty subtree
	if bytes.Equal(smt.placeholder(), digest) {
		return nil, nil
	}

	// Retrieve the encoded noe data
	data, err := smt.nodes.Get(digest)
	if err != nil {
		return nil, err
	}

	// Return the appropriate node type based on the first byte of the data
	if isLeafNode(data) {
		path, valueHash := smt.parseLeafNode(data)
		return &leafNode{
			path:      path,
			valueHash: valueHash,
			persisted: true,
			digest:    digest,
		}, nil
	} else if isExtNode(data) {
		pathBounds, path, childData, _ := smt.parseSumExtNode(data)
		return &extensionNode{
			path:       path,
			pathBounds: [2]byte(pathBounds),
			child:      &lazyNode{childData},
			persisted:  true,
			digest:     digest,
		}, nil
	} else if isInnerNode(data) {
		leftData, rightData, _ := smt.th.parseSumInnerNode(data)
		return &innerNode{
			leftChild:  &lazyNode{leftData},
			rightChild: &lazyNode{rightData},
			persisted:  true,
			digest:     digest,
		}, nil
	} else {
		panic("invalid node type")
	}
}

// Commit persists all dirty nodes in the trie, deletes all orphaned
// nodes from the database and then computes and saves the root hash
func (smt *SMT) Commit() (err error) {
	// All orphans are persisted and have cached digests, so we don't need to check for null
	for _, orphans := range smt.orphans {
		for _, hash := range orphans {
			if err = smt.nodes.Delete(hash); err != nil {
				return
			}
		}
	}
	smt.orphans = nil
	if err = smt.commit(smt.root); err != nil {
		return
	}
	smt.rootHash = smt.Root()
	return
}

func (smt *SMT) commit(node trieNode) error {
	if node != nil && node.Persisted() {
		return nil
	}
	switch n := node.(type) {
	case *leafNode:
		n.persisted = true
	case *innerNode:
		n.persisted = true
		if err := smt.commit(n.leftChild); err != nil {
			return err
		}
		if err := smt.commit(n.rightChild); err != nil {
			return err
		}
	case *extensionNode:
		n.persisted = true
		if err := smt.commit(n.child); err != nil {
			return err
		}
	default:
		return nil
	}
	preimage := smt.encode(node)
	return smt.nodes.Set(smt.digest(node), preimage)
}

func (smt *SMT) addOrphan(orphans *[][]byte, node trieNode) {
	if node.Persisted() {
		*orphans = append(*orphans, node.CachedDigest())
	}
}
