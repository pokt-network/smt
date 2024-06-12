package smt

import (
	"bytes"
	"encoding/binary"
	"hash"

	"github.com/pokt-network/smt/kvstore"
)

const (
	// The number of bytes used to represent the sum of a node
	sumSizeBytes = 8

	// The number of bytes used to track the count of non-empty nodes in the trie.
	// TODO_TECHDEBT: Since we are using sha256, we could theoretically have
	// 2^256 leaves. This would require 32 bytes, and would not fit in a uint64.
	// For now, we are assuming that we will not have more than 2^64 leaves.
	countSizeBytes = 8
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
	options ...TrieSpecOption,
) *SMST {
	trieSpec := newTrieSpec(hasher, true)
	for _, option := range options {
		option(&trieSpec)
	}

	// Initialize a non-sum SMT and modify it to have a nil value hasher.
	// NB: We are using a nil value hasher because the SMST pre-hashes its paths.
	//     This results in double path hashing because the SMST is a wrapper
	//     around the SMT. The reason the SMST uses its own path hashing logic is
	//     to account for the additional sum in the encoding/decoding process.
	//     Therefore, the underlying SMT underneath needs a nil path hasher, while
	//     the outer SMST does all the (non nil) path hashing itself.
	// TODO_TECHDEBT(@Olshansk): Look for ways to simplify / cleanup the above.
	smt := &SMT{
		TrieSpec: trieSpec,
		nodes:    nodes,
	}
	nilValueHasher := WithValueHasher(nil)
	nilValueHasher(&smt.TrieSpec)

	return &SMST{
		TrieSpec: trieSpec,
		SMT:      smt,
	}
}

// ImportSparseMerkleSumTrie returns a pointer to an SMST struct with the root hash provided
func ImportSparseMerkleSumTrie(
	nodes kvstore.MapStore,
	hasher hash.Hash,
	root []byte,
	options ...TrieSpecOption,
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

// Get retrieves the value digest for the given key, along with its weight assuming
// the node exists, otherwise the default placeholder values are returned 
func (smst *SMST) Get(key []byte) (valueDigest []byte, weight uint64, err error) {
	// Retrieve the value digest from the trie for the given key
	value, err := smst.SMT.Get(key)
	if err != nil {
		return nil, 0, err
	}

	// Check if it is an empty branch
	if bytes.Equal(value, defaultEmptyValue) {
		return defaultEmptyValue, 0, nil
	}

	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(value)

	// Extract the value digest only
	valueDigest = value[:firstSumByteIdx]

	// Retrieve the node weight
	var weightBz [sumSizeBytes]byte
	copy(weightBz[:], value[firstSumByteIdx:firstCountByteIdx])
	weight = binary.BigEndian.Uint64(weightBz[:])

	// Retrieve the number of non-empty nodes in the sub trie
	var countBz [countSizeBytes]byte
	copy(countBz[:], value[firstCountByteIdx:])
	count := binary.BigEndian.Uint64(countBz[:])

	// TODO_IN_THIS_PR: Consider not retuning count here because it should always be 1
	if count != 1 {
		panic("count for leaf node should always be 1")
	}

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
	var weightBz [sumSizeBytes]byte
	binary.BigEndian.PutUint64(weightBz[:], weight)

	// Convert the node count (1 for a single leaf) to a byte slice
	var countBz [countSizeBytes]byte
	binary.BigEndian.PutUint64(countBz[:], 1)

	// Compute the digest of the value and append the weight to it
	valueDigest := smst.valueHash(value)
	valueDigest = append(valueDigest, weightBz[:]...)
	valueDigest = append(valueDigest, countBz[:]...)

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
	rootDigest := []byte(smst.Root())

	if !smst.Spec().sumTrie {
		panic("SMST: not a merkle sum trie")
	}

	firstSumByteIdx, firstCountByteIdx := getFirstMetaByteIdx(rootDigest)

	var sumBz [sumSizeBytes]byte
	copy(sumBz[:], rootDigest[firstSumByteIdx:firstCountByteIdx])
	return binary.BigEndian.Uint64(sumBz[:])
}

// Count returns the number of non-empty nodes in the entire trie stored in the root.
func (smst *SMST) Count() uint64 {
	rootDigest := []byte(smst.Root())

	if !smst.Spec().sumTrie {
		panic("SMST: not a merkle sum trie")
	}

	_, firstCountByteIdx := getFirstMetaByteIdx(rootDigest)

	var countBz [countSizeBytes]byte
	copy(countBz[:], rootDigest[firstCountByteIdx:])
	return binary.BigEndian.Uint64(countBz[:])
}

// getFirstMetaByteIdx returns the index of the first count byte and the first sum byte
// in the data slice provided. This is useful metadata when parsing the data
// of any node in the trie.
func getFirstMetaByteIdx(data []byte) (firstSumByteIdx, firstCountByteIdx int) {
	firstCountByteIdx = len(data) - countSizeBytes
	firstSumByteIdx = firstCountByteIdx - sumSizeBytes
	return
}
