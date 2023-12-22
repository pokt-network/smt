// Package smt provides an implementation of a Sparse Merkle trie for a
// key-value map. The trie implements the same optimisations specified in the
// Libra whitepaper to reduce the number of hash operations required per trie
// operation to O(k) where k is the number of non-empty elements in the trie.
// And is implemente in a similar way to the JMT whitepaper, with additional
// features and proof mechanics, such as a Sparse Merkle Sum Trie and new
// ClosestProof mechanics.
package smt
