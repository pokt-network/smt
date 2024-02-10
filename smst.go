package smt

import (
	"bytes"
	"encoding/binary"
	"hash"

	"github.com/pokt-network/smt/kvstore"
)

const (
	// The number of bits used to represent the sum of a node
	sumSizeBits = 8
)

var _ SparseMerkleSumTrie = (*SMST)(nil)

// SMST is an object wrapping a Sparse Merkle Trie for custom encoding
type SMST struct {
	TrieSpec
	*SMT
}

// NewSparseMerkleSumTrie returns a pointer to an SMST struct
func NewSparseMerkleSumTrie(
	nodes kvstore.MapStore,
	hasher hash.Hash,
	options ...Option,
) *SMST {
	smt := &SMT{
		TrieSpec: newTrieSpec(hasher, true),
		nodes:    nodes,
	}
	for _, option := range options {
		option(&smt.TrieSpec)
	}
	nvh := WithValueHasher(nil)
	nvh(&smt.TrieSpec)
	smst := &SMST{
		TrieSpec: newTrieSpec(hasher, true),
		SMT:      smt,
	}
	for _, option := range options {
		option(&smst.TrieSpec)
	}
	return smst
}

// ImportSparseMerkleSumTrie returns a pointer to an SMST struct with the root hash provided
func ImportSparseMerkleSumTrie(
	nodes kvstore.MapStore,
	hasher hash.Hash,
	root []byte,
	options ...Option,
) *SMST {
	smst := NewSparseMerkleSumTrie(nodes, hasher, options...)
	smst.root = &lazyNode{root}
	smst.rootHash = root
	return smst
}

// Spec returns the SMST TrieSpec
func (smst *SMST) Spec() *TrieSpec {
	return &smst.TrieSpec
}

// Get retrieves the value digest for the given key and the digest of the value
// along with its weight provided a leaf node exists.
func (smst *SMST) Get(key []byte) (valueDigest []byte, weight uint64, err error) {
	// Retrieve the value digest from the trie for the given key
	valueDigest, err = smst.SMT.Get(key)
	if err != nil {
		return nil, 0, err
	}

	// Check if it ias an empty branch
	if bytes.Equal(valueDigest, defaultEmptyValue) {
		return defaultEmptyValue, 0, nil
	}

	// Retrieve the node weight
	var weightBz [sumSizeBits]byte
	copy(weightBz[:], valueDigest[len(valueDigest)-sumSizeBits:])
	weight = binary.BigEndian.Uint64(weightBz[:])

	// Remove the weight from the value digest
	valueDigest = valueDigest[:len(valueDigest)-sumSizeBits]

	// Return the value digest and weight
	return valueDigest, weight, nil
}

// Update inserts the value and weight into the trie for the given key.
//
// The a digest (i.e. hash) of the value is computed and appended with the byte
// representation of the weight integer provided.

// The weight is used to compute the interim sum of the node which then percolates
// up to the total sum of the trie.
func (smst *SMST) Update(key, value []byte, weight uint64) error {
	// Convert the node weight to a byte slice
	var weightBz [sumSizeBits]byte
	binary.BigEndian.PutUint64(weightBz[:], weight)

	// Compute the digest of the value and append the weight to it
	valueDigest := smst.valueHash(value)
	valueDigest = append(valueDigest, weightBz[:]...)

	// Return the result of the trie update
	return smst.SMT.Update(key, valueDigest)
}

// Delete removes the node at the path corresponding to the given key
func (smst *SMST) Delete(key []byte) error {
	return smst.SMT.Delete(key)
}

// Prove generates a SparseMerkleProof for the given key
func (smst *SMST) Prove(key []byte) (*SparseMerkleProof, error) {
	return smst.SMT.Prove(key)
}

// ProveClosest generates a SparseMerkleProof of inclusion for the key
// with the most common bits as the path provided
func (smst *SMST) ProveClosest(path []byte) (
	proof *SparseMerkleClosestProof,
	err error,
) {
	return smst.SMT.ProveClosest(path)
}

// Commit persists all dirty nodes in the trie, deletes all orphaned
// nodes from the database and then computes and saves the root hash
func (smst *SMST) Commit() error {
	return smst.SMT.Commit()
}

// Root returns the root hash of the trie with the total sum bytes appended
func (smst *SMST) Root() MerkleRoot {
	return smst.SMT.Root() // [digest]+[binary sum]
}

// Sum returns the sum of the entire trie stored in the root.
// If the tree is not a sum tree, it will panic.
func (smst *SMST) Sum() uint64 {
	rootDigest := smst.Root()
	if len(rootDigest) != smst.th.hashSize()+sumSizeBits {
		panic("roo#sum: not a merkle sum trie")
	}
	var sumbz [sumSizeBits]byte
	copy(sumbz[:], []byte(rootDigest)[len([]byte(rootDigest))-sumSizeBits:])
	return binary.BigEndian.Uint64(sumbz[:])
}
