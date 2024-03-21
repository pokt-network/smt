package smt

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
