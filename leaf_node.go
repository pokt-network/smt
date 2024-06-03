package smt

// Ensure leafNode satisfies the trieNode interface
var _ trieNode = (*leafNode)(nil)

// leafNode stores a full key-value pair in the trie
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
