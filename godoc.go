// Package smt provides an implementation of a Sparse Merkle tree for a
// key-value map. The tree implements the same optimisations specified in the
// Libra whitepaper to reduce the number of hash operations required per tree
// operation to O(k) where k is the number of non-empty elements in the tree.
// And is implemente in a similar way to the JMT whitepaper, with additional
// features and proof mechanics, such as a Sparse Merkle Sum Tree and new
// ClosestProof mechanics.
package smt
