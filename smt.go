package smt

import (
	"bytes"
	"hash"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

var (
	_ trieNode         = (*innerNode)(nil)
	_ trieNode         = (*leafNode)(nil)
	_ SparseMerkleTrie = (*SMT)(nil)
)

type trieNode interface {
	// when committing a node to disk, skip if already persisted
	Persisted() bool
	CachedDigest() []byte
}

// A branch within the trie
type innerNode struct {
	// Both child nodes are always non-nil
	leftChild, rightChild trieNode
	persisted             bool
	digest                []byte
}

// Stores data and full path
type leafNode struct {
	path      []byte
	valueHash []byte
	persisted bool
	digest    []byte
}

// A compressed chain of singly-linked inner nodes
type extensionNode struct {
	path []byte
	// Offsets into path slice of bounds defining actual path segment.
	// Note: assumes path is <=256 bits
	pathBounds [2]byte
	// Child is always an inner node, or lazy.
	child     trieNode
	persisted bool
	digest    []byte
}

// Represents an uncached, persisted node
type lazyNode struct {
	digest []byte
}

// SMT is a Sparse Merkle Trie object that implements the SparseMerkleTrie interface
type SMT struct {
	TrieSpec
	nodes kvstore.MapStore
	// Last persisted root hash
	savedRoot []byte
	// Current state of trie
	trie trieNode
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
	options ...Option,
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
	options ...Option,
) *SMT {
	smt := NewSparseMerkleTrie(nodes, hasher, options...)
	smt.trie = &lazyNode{root}
	smt.savedRoot = root
	return smt
}

// Get returns the digest of the value stored at the given key
func (smt *SMT) Get(key []byte) ([]byte, error) {
	path := smt.ph.Path(key)
	var leaf *leafNode
	var err error
	for node, depth := &smt.trie, 0; ; depth++ {
		*node, err = smt.resolveLazy(*node)
		if err != nil {
			return nil, err
		}
		if *node == nil {
			break
		}
		if n, ok := (*node).(*leafNode); ok {
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
			*node, err = smt.resolveLazy(*node)
			if err != nil {
				return nil, err
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
		return defaultValue, nil
	}
	return leaf.valueHash, nil
}

// Update sets the value for the given key, to the digest of the provided value
func (smt *SMT) Update(key []byte, value []byte) error {
	path := smt.ph.Path(key)
	valueHash := smt.digestValue(value)
	var orphans orphanNodes
	trie, err := smt.update(smt.trie, 0, path, valueHash, &orphans)
	if err != nil {
		return err
	}
	smt.trie = trie
	if len(orphans) > 0 {
		smt.orphans = append(smt.orphans, orphans)
	}
	return nil
}

func (smt *SMT) update(
	node trieNode, depth int, path, value []byte, orphans *orphanNodes,
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
		prefixlen := countCommonPrefixBits(path, leaf.path, depth)
		if prefixlen == smt.depth() { // replace leaf if paths are equal
			smt.addOrphan(orphans, node)
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

	smt.addOrphan(orphans, node)

	if ext, ok := node.(*extensionNode); ok {
		var branch *trieNode
		node, branch, depth = ext.split(path, depth)
		*branch, err = smt.update(*branch, depth, path, value, orphans)
		if err != nil {
			return node, err
		}
		ext.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child *trieNode
	if getPathBit(path, depth) == left {
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
	trie, err := smt.delete(smt.trie, 0, path, &orphans)
	if err != nil {
		return err
	}
	smt.trie = trie
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
		return node, simplemap.ErrKVStoreKeyNotFound
	}
	if leaf, ok := node.(*leafNode); ok {
		if !bytes.Equal(path, leaf.path) {
			return node, simplemap.ErrKVStoreKeyNotFound
		}
		smt.addOrphan(orphans, node)
		return nil, nil
	}

	smt.addOrphan(orphans, node)

	if ext, ok := node.(*extensionNode); ok {
		if _, match := ext.match(path, depth); !match {
			return node, simplemap.ErrKVStoreKeyNotFound
		}
		ext.child, err = smt.delete(ext.child, depth+ext.length(), path, orphans)
		if err != nil {
			return node, err
		}
		switch n := ext.child.(type) {
		case *leafNode:
			return n, nil
		case *extensionNode:
			// Join this extension with the child
			smt.addOrphan(orphans, n)
			n.pathBounds[0] = ext.pathBounds[0]
			node = n
		}
		ext.setDirty()
		return node, nil
	}

	inner := node.(*innerNode)
	var child, sib *trieNode
	if getPathBit(path, depth) == left {
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

	node := smt.trie
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
		if ext, ok := node.(*extensionNode); ok {
			length, match := ext.match(path, depth)
			if match {
				for i := 0; i < length; i++ {
					siblings = append(siblings, nil)
				}
				depth += length
				node = ext.child
				node, err = smt.resolveLazy(node)
				if err != nil {
					return nil, err
				}
			} else {
				node = ext.expand()
			}
		}
		inner := node.(*innerNode)
		if getPathBit(path, depth) == left {
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
			leafData = encodeLeaf(leaf.path, leaf.valueHash)
		}
	}
	// Hash siblings from bottom up.
	var sideNodes [][]byte
	for i := range siblings {
		var sideNode []byte
		sibling := siblings[len(siblings)-i-1]
		sideNode = hashNode(smt.Spec(), sibling)
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
		proof.SiblingData = serialize(smt.Spec(), sib)
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

	node := smt.trie
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
		if ext, ok := node.(*extensionNode); ok {
			length, match := ext.match(workingPath, depth)
			// workingPath from depth to end of extension node's path bounds
			// is a perfect match
			if !match {
				node = ext.expand()
			} else {
				// extension nodes represent a singly linked list of inner nodes
				// add nil siblings to represent the empty neighbours
				for i := 0; i < length; i++ {
					siblings = append(siblings, nil)
				}
				depth += length
				depthDelta += length
				node = ext.child
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
		if getPathBit(workingPath, depth) == left {
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
		proof.ClosestPath, proof.ClosestValueHash = placeholder(smt.Spec()), nil
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
		sideNode = hashNode(smt.Spec(), sibling)
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
		proof.ClosestProof.SiblingData = serialize(smt.Spec(), sib)
	}

	return proof, nil
}

//nolint:unused
func (smt *SMT) recursiveLoad(hash []byte) (trieNode, error) {
	return smt.resolve(hash, smt.recursiveLoad)
}

// resolves a stub into a cached node
func (smt *SMT) resolveLazy(node trieNode) (trieNode, error) {
	stub, ok := node.(*lazyNode)
	if !ok {
		return node, nil
	}
	resolver := func(hash []byte) (trieNode, error) {
		return &lazyNode{hash}, nil
	}
	ret, err := resolve(smt, stub.digest, resolver)
	if err != nil {
		return node, err
	}
	return ret, nil
}

func (smt *SMT) resolve(hash []byte, resolver func([]byte) (trieNode, error),
) (ret trieNode, err error) {
	if bytes.Equal(smt.th.placeholder(), hash) {
		return
	}
	data, err := smt.nodes.Get(hash)
	if err != nil {
		return
	}
	if isLeaf(data) {
		leaf := leafNode{persisted: true, digest: hash}
		leaf.path, leaf.valueHash = parseLeaf(data, smt.ph)
		return &leaf, nil
	}
	if isExtension(data) {
		ext := extensionNode{persisted: true, digest: hash}
		pathBounds, path, childHash := parseExtension(data, smt.ph)
		ext.path = path
		copy(ext.pathBounds[:], pathBounds)
		ext.child, err = resolver(childHash)
		if err != nil {
			return
		}
		return &ext, nil
	}
	leftHash, rightHash := smt.th.parseNode(data)
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

func (smt *SMT) resolveSum(hash []byte, resolver func([]byte) (trieNode, error),
) (ret trieNode, err error) {
	if bytes.Equal(placeholder(smt.Spec()), hash) {
		return
	}
	data, err := smt.nodes.Get(hash)
	if err != nil {
		return nil, err
	}
	if isLeaf(data) {
		leaf := leafNode{persisted: true, digest: hash}
		leaf.path, leaf.valueHash = parseLeaf(data, smt.ph)
		return &leaf, nil
	}
	if isExtension(data) {
		ext := extensionNode{persisted: true, digest: hash}
		pathBounds, path, childHash, _ := parseSumExtension(data, smt.ph)
		ext.path = path
		copy(ext.pathBounds[:], pathBounds)
		ext.child, err = resolver(childHash)
		if err != nil {
			return
		}
		return &ext, nil
	}
	leftHash, rightHash := smt.th.parseSumNode(data)
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
	if err = smt.commit(smt.trie); err != nil {
		return
	}
	smt.savedRoot = smt.Root()
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
	preimage := serialize(smt.Spec(), node)
	return smt.nodes.Set(hashNode(smt.Spec(), node), preimage)
}

// Root returns the root hash of the trie
func (smt *SMT) Root() []byte {
	return hashNode(smt.Spec(), smt.trie)
}

func (smt *SMT) addOrphan(orphans *[][]byte, node trieNode) {
	if node.Persisted() {
		*orphans = append(*orphans, node.CachedDigest())
	}
}

func (node *leafNode) Persisted() bool      { return node.persisted }
func (node *innerNode) Persisted() bool     { return node.persisted }
func (node *lazyNode) Persisted() bool      { return true }
func (node *extensionNode) Persisted() bool { return node.persisted }

func (node *leafNode) CachedDigest() []byte      { return node.digest }
func (node *innerNode) CachedDigest() []byte     { return node.digest }
func (node *lazyNode) CachedDigest() []byte      { return node.digest }
func (node *extensionNode) CachedDigest() []byte { return node.digest }

func (inner *innerNode) setDirty() {
	inner.persisted = false
	inner.digest = nil
}

func (ext *extensionNode) length() int { return int(ext.pathBounds[1] - ext.pathBounds[0]) }

func (ext *extensionNode) setDirty() {
	ext.persisted = false
	ext.digest = nil
}

// Returns length of matching prefix, and whether it's a full match
func (ext *extensionNode) match(path []byte, depth int) (int, bool) {
	if depth != ext.pathStart() {
		panic("depth != path_begin")
	}
	for i := ext.pathStart(); i < ext.pathEnd(); i++ {
		if getPathBit(ext.path, i) != getPathBit(path, i) {
			return i - ext.pathStart(), false
		}
	}
	return ext.length(), true
}

//nolint:unused
func (ext *extensionNode) commonPrefix(path []byte) int {
	count := 0
	for i := ext.pathStart(); i < ext.pathEnd(); i++ {
		if getPathBit(ext.path, i) != getPathBit(path, i) {
			break
		}
		count++
	}
	return count
}

func (ext *extensionNode) pathStart() int { return int(ext.pathBounds[0]) }
func (ext *extensionNode) pathEnd() int   { return int(ext.pathBounds[1]) }

// Splits the node in-place; returns replacement node, child node at the split, and split depth
func (ext *extensionNode) split(path []byte, depth int) (trieNode, *trieNode, int) {
	if depth != ext.pathStart() {
		panic("depth != path_begin")
	}
	index := ext.pathStart()
	var myBit, branchBit int
	for ; index < ext.pathEnd(); index++ {
		myBit = getPathBit(ext.path, index)
		branchBit = getPathBit(path, index)
		if myBit != branchBit {
			break
		}
	}
	if index == ext.pathEnd() {
		return ext, &ext.child, index
	}

	child := ext.child
	var branch innerNode
	var head trieNode
	var tail *trieNode
	if myBit == left {
		tail = &branch.leftChild
	} else {
		tail = &branch.rightChild
	}

	// Split at first bit: chain starts with new node
	if index == ext.pathStart() {
		head = &branch
		ext.pathBounds[0]++ // Shrink the extension from front
		if ext.length() == 0 {
			*tail = child
		} else {
			*tail = ext
		}
	} else {
		// Split inside: chain ends at index
		head = ext
		ext.child = &branch
		if index == ext.pathEnd()-1 {
			*tail = child
		} else {
			*tail = &extensionNode{
				path:       ext.path,
				pathBounds: [2]byte{byte(index + 1), ext.pathBounds[1]},
				child:      child,
			}
		}
		ext.pathBounds[1] = byte(index)
	}
	var b trieNode = &branch
	return head, &b, index
}

// expand returns the inner node that represents the start of the singly
// linked list that this extension node represents
func (ext *extensionNode) expand() trieNode {
	last := ext.child
	for i := ext.pathEnd() - 1; i >= ext.pathStart(); i-- {
		var next innerNode
		if getPathBit(ext.path, i) == left {
			next.leftChild = last
		} else {
			next.rightChild = last
		}
		last = &next
	}
	return last
}
