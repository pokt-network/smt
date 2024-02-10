package smt

import (
	"hash"
)

// Option is a function that configures SparseMerkleTrie.
type Option func(*TrieSpec)

// WithPathHasher returns an Option that sets the PathHasher to the one provided
func WithPathHasher(ph PathHasher) Option {
	return func(ts *TrieSpec) { ts.ph = ph }
}

// WithValueHasher returns an Option that sets the ValueHasher to the one provided
func WithValueHasher(vh ValueHasher) Option {
	return func(ts *TrieSpec) { ts.vh = vh }
}

// NoHasherSpec returns a new TrieSpec that has nil ValueHasher & PathHasher specs.
// NB: This should only be used when values are already hashed and a path is
// used instead of a key during proof verification. Otherwise, this will lead
// double hashing and product an incorrect leaf digest, thereby invalidating
// the proof.
// TODO_IN_THIS_PR: Need to understand this part more.
func NoHasherSpec(hasher hash.Hash, sumTrie bool) *TrieSpec {
	spec := newTrieSpec(hasher, sumTrie)

	// Set a nil path hasher
	opt := WithPathHasher(NewNilPathHasher(hasher))
	opt(&spec)

	// Set a nil value hasher
	opt = WithValueHasher(nil)
	opt(&spec)

	// Return the spec
	return &spec
}
