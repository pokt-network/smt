package smt

// TODO_DISCUSS_IN_THE_FUTURE:
// 1. Should we rename all instances of digest to hash?
// 2. Should we introduce a shared interface between SparseMerkleTrie and SparseMerkleSumTrie?
// 3. Should we rename Commit to FlushToDisk?

const (
	// The bit value use to distinguish an inner nodes left child and right child
	leftChildBit = 0
)

var (
	// defaultEmptyValue is the default value for a leaf node
	defaultEmptyValue []byte
	// defaultEmptySum is the default sum value for a leaf node
	defaultEmptySum [sumSizeBits]byte
)

// MerkleRoot is a type alias for a byte slice returned from the Root method
type MerkleRoot []byte

// A high-level interface that captures the behaviour of all types of nodes
type trieNode interface {
	// Persisted returns a boolean to determine whether or not the node
	// has been persisted to disk or only held in memory.
	// It can be used skip unnecessary iops if already persisted
	Persisted() bool

	// The digest of the node, returning a cached value if available.
	CachedDigest() []byte
}

// SparseMerkleTrie represents a Sparse Merkle Trie.
type SparseMerkleTrie interface {
	// Update inserts a value into the SMT.
	Update(key, value []byte) error
	// Delete deletes a value from the SMT. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the trie to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, error)
	// Root computes the Merkle root digest.
	Root() MerkleRoot
	// Prove computes a Merkle proof of inclusion or exclusion of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the trie
	// which is closest to the path provided. It will search for the key with
	// the longest common prefix before finding the key with the most common
	// bits as the path provided.
	ProveClosest([]byte) (*SparseMerkleClosestProof, error)
	// Commit saves the trie's state to its persistent storage.
	Commit() error
	// Spec returns the TrieSpec for the trie
	Spec() *TrieSpec
}

// SparseMerkleSumTrie represents a Sparse Merkle Sum Trie.
type SparseMerkleSumTrie interface {
	// Update inserts a value and its sum into the SMST.
	Update(key, value []byte, sum uint64) error
	// Delete deletes a value from the SMST. Raises an error if the key is not present.
	Delete(key []byte) error
	// Get descends the trie to access a value. Returns nil if key is not present.
	Get(key []byte) ([]byte, uint64, error)
	// Root computes the Merkle root digest.
	Root() MerkleRoot
	// Sum computes the total sum of the Merkle trie
	Sum() uint64
	// Prove computes a Merkle proof of inclusion or exclusion of a key.
	Prove(key []byte) (*SparseMerkleProof, error)
	// ProveClosest computes a Merkle proof of inclusion for a key in the trie
	// which is closest to the path provided. It will search for the key with
	// the longest common prefix before finding the key with the most common
	// bits as the path provided.
	ProveClosest([]byte) (*SparseMerkleClosestProof, error)
	// Commit saves the trie's state to its persistent storage.
	Commit() error
	// Spec returns the TrieSpec for the trie
	Spec() *TrieSpec
}
