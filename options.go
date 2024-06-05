package smt

// TrieSpecOption is a function that configures SparseMerkleTrie.
type TrieSpecOption func(*TrieSpec)

// WithPathHasher returns an Option that sets the PathHasher to the one provided
// this MUST not be nil or unknown behaviour will occur.
func WithPathHasher(ph PathHasher) TrieSpecOption {
	return func(ts *TrieSpec) { ts.ph = ph }
}

// WithValueHasher returns an Option that sets the ValueHasher to the one provided
func WithValueHasher(vh ValueHasher) TrieSpecOption {
	return func(ts *TrieSpec) { ts.vh = vh }
}
