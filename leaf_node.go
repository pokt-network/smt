package smt

var _ trieNode = (*leafNode)(nil)

// A leaf node storing a key-value pair for a full path.
type leafNode struct {
	path      []byte
	valueHash []byte
	persisted bool
	digest    []byte
}

// Persisted satisfied the trieNode#Persisted interface
func (node *leafNode) Persisted() bool { return node.persisted }

// Persisted satisfied the trieNode#CachedDigest interface
func (node *leafNode) CachedDigest() []byte { return node.digest }
