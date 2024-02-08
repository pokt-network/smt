package smt

var _ trieNode = (*extensionNode)(nil)

// A compressed chain of singly-linked inner nodes
type extensionNode struct {
	path []byte
	// Offsets into path slice of bounds defining actual path segment.
	// NOTE: assumes path is <=256 bits
	pathBounds [2]byte
	// Child is always an inner node, or lazy.
	child trieNode
	// Bool whether or not the node has been flushed to disk
	persisted bool
	// The cached digest of the node trie
	digest []byte
}

func (node *extensionNode) Persisted() bool {
	return node.persisted
}

func (node *extensionNode) CachedDigest() []byte {
	return node.digest
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

func (ext *extensionNode) pathStart() int {
	return int(ext.pathBounds[0])
}

func (ext *extensionNode) pathEnd() int {
	return int(ext.pathBounds[1])
}

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
				path: ext.path,
				pathBounds: [2]byte{
					byte(index + 1),
					ext.pathBounds[1],
				},
				child: child,
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
