package smt

// Ensure lazyNode satisfies the trieNode interface
var _ trieNode = (*lazyNode)(nil)

// lazyNode represents an uncached persisted node
type lazyNode struct {
	digest []byte
}

// Persisted satisfied the trieNode#Persisted interface
func (node *lazyNode) Persisted() bool { return true }

// Persisted satisfied the trieNode#CachedDigest interface
func (node *lazyNode) CachedDigest() []byte { return node.digest }
