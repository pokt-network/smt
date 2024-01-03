// Package smt provides an implementation of a Sparse Merkle Trie for a
// key-value map.
//
// The trie implements the same optimizations specified in the JMT
// whitepaper to account for empty and single-node subtrees. Unlike the
// JMT, it only supports binary trees and does not optimise for RockDB
// on-disk storage.
//
// This package implements novel features that include native in-node
// weight sums, as well as support for ClosestProof mechanics.
package smt
