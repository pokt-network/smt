package smt

// Ensure extensionNode satisfies the trieNode interface
var _ trieNode = (*extensionNode)(nil)

// A compressed chain of singly-linked inner nodes.
//
// Extension nodes are used to captures a series of inner nodes that only
// have one child in a succinct `pathBounds` for optimization purposes.
//
// Assumption: the path is <=256 bits
type extensionNode struct {
	// The path (starting at the root) to this extension node.
	path []byte
	// The path (starting at pathBounds[0] and ending at pathBounds[1]) of
	// inner nodes that this single extension node replaces.
	pathBounds [2]byte
	// A child node from this extension node.
	// It MUST be either an innerNode or a lazyNode.
	child trieNode
	// Bool whether or not the node has been flushed to disk
	persisted bool
	// The cached digest of the node trie
	digest []byte
}

// Persisted satisfied the trieNode#Persisted interface
func (node *extensionNode) Persisted() bool {
	return node.persisted
}

// Persisted satisfied the trieNode#CachedDigest interface
func (node *extensionNode) CachedDigest() []byte {
	return node.digest
}

// Length returns the length of the path segment represented by this single
// extensionNode. Since the SMT is a binary trie, the length represents both
// the depth and the number of nodes replaced by a single extension node. If
// this SMT were to have k-ary support, the depth would be strictly less than
// the number of nodes replaced.
func (ext *extensionNode) length() int {
	return ext.pathEnd() - ext.pathStart()
}

func (ext *extensionNode) pathStart() int {
	return int(ext.pathBounds[0])
}

func (ext *extensionNode) pathEnd() int {
	return int(ext.pathBounds[1])
}

// setDirty marks the node as dirty (i.e. not flushed to disk) and clears
// its digest
func (ext *extensionNode) setDirty() {
	ext.persisted = false
	ext.digest = nil
}

// boundsMatch returns the length of the matching prefix between `ext.pathBounds`
// and `path` starting at index `depth`, along with a bool if a full match is found.
func (extNode *extensionNode) boundsMatch(path []byte, depth int) (int, bool) {
	if depth != extNode.pathStart() {
		panic("depth != extNode.pathStart")
	}
	for pathIdx := extNode.pathStart(); pathIdx < extNode.pathEnd(); pathIdx++ {
		if getPathBit(extNode.path, pathIdx) != getPathBit(path, pathIdx) {
			return pathIdx - extNode.pathStart(), false
		}
	}
	return extNode.length(), true
}

// split splits the node in-place by returning a new node at the extension node,
// a child node at the split and split depth.
func (extNode *extensionNode) split(path []byte) (trieNode, *trieNode, int) {
	// Start path to extNode.pathBounds until there is no match
	var extNodeBit, pathBit int
	pathIdx := extNode.pathStart()
	for ; pathIdx < extNode.pathEnd(); pathIdx++ {
		extNodeBit = getPathBit(extNode.path, pathIdx)
		pathBit = getPathBit(path, pathIdx)
		if extNodeBit != pathBit {
			break
		}
	}
	// Return the extension node's child if path fully matches extNode.pathBounds
	if pathIdx == extNode.pathEnd() {
		return extNode, &extNode.child, pathIdx
	}

	child := extNode.child
	var branch innerNode
	var head trieNode
	var tail *trieNode
	if extNodeBit == leftChildBit {
		tail = &branch.leftChild
	} else {
		tail = &branch.rightChild
	}

	// Split at first bit: chain starts with new node
	if pathIdx == extNode.pathStart() {
		head = &branch
		extNode.pathBounds[0]++ // Shrink the extension from front
		if extNode.length() == 0 {
			*tail = child
		} else {
			*tail = extNode
		}
	} else {
		// Split inside: chain ends at index
		head = extNode
		extNode.child = &branch
		if pathIdx == extNode.pathEnd()-1 {
			*tail = child
		} else {
			*tail = &extensionNode{
				path: extNode.path,
				pathBounds: [2]byte{
					byte(pathIdx + 1),
					extNode.pathBounds[1],
				},
				child: child,
			}
		}
		extNode.pathBounds[1] = byte(pathIdx)
	}
	var b trieNode = &branch
	return head, &b, pathIdx
}

// expand returns the inner node that represents the end of the singly
// linked list that this extension node represents
func (extNode *extensionNode) expand() trieNode {
	last := extNode.child
	for i := extNode.pathEnd() - 1; i >= extNode.pathStart(); i-- {
		var next innerNode
		if getPathBit(extNode.path, i) == leftChildBit {
			next.leftChild = last
		} else {
			next.rightChild = last
		}
		last = &next
	}
	return last
}
