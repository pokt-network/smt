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

// NoPrehashSpec returns a new TrieSpec that has a nil Value Hasher and a nil
// Path Hasher
// NOTE: This should only be used when values are already hashed and a path is
// used instead of a key during proof verification, otherwise these will be
// double hashed and produce an incorrect leaf digest invalidating the proof.
func NoPrehashSpec(hasher hash.Hash, sumTrie bool) *TrieSpec {
	spec := newTrieSpec(hasher, sumTrie)
	opt := WithPathHasher(newNilPathHasher(hasher.Size()))
	opt(&spec)
	opt = WithValueHasher(nil)
	opt(&spec)
	return &spec
}
