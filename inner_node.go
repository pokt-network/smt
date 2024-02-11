package smt

// Ensure innerNode satisfies the trieNode interface
var _ trieNode = (*innerNode)(nil)

// A branch within the binary trie pointing to a left & right child.
type innerNode struct {
	// Left and right child nodes.
	// Both child nodes are always expected to be non-nil.
	leftChild, rightChild trieNode
	persisted             bool
	digest                []byte
}

// Persisted satisfied the trieNode#Persisted interface
func (node *innerNode) Persisted() bool { return node.persisted }

// Persisted satisfied the trieNode#CachedDigest interface
func (node *innerNode) CachedDigest() []byte { return node.digest }

// setDirty marks the node as dirty (i.e. not flushed to disk) and clears the cached digest
func (node *innerNode) setDirty() {
	node.persisted = false
	node.digest = nil
}
