package smt

// Option is a function that configures SparseMerkleTree.
type Option func(*SMT)

// WithPathHasher returns an Option that sets the PathHasher to the one provided
func WithPathHasher(ph PathHasher) Option {
	return func(smt *SMT) { smt.ph = ph }
}

// WithValueHasher returns an Option that sets the ValueHasher to the one provided
func WithValueHasher(vh ValueHasher) Option {
	return func(smt *SMT) { smt.vh = vh }
}
