// Package smt provides an implementation of a Sparse Merkle Trie for a
// key-value map or engine.
//
// The trie implements the same optimizations specified in the JMT
// whitepaper to account for empty and single-node subtrees.

// Unlike the JMT, it only supports binary trees and does not implemented the
// same RocksDB optimizations as specified in the original JMT library when
// optimizing for disk iops
//
// This package implements additional SMT specific functionality related to
// tree sums and closest proof mechanics.
package smt
