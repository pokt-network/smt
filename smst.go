package smt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"

	"github.com/pokt-network/smt/kvstore"
)

const (
	// The number of bytes used to represent the sum of a node
	sumSizeBytes = 8

	// The number of bytes used to track the count of non-empty nodes in the trie.
	//
	// TODO_TECHDEBT: Since we are using sha256, we could theoretically have
	// 2^256 leaves. This would require 32 bytes, and would not fit in a uint64.
	// For now, we are assuming that we will not have more than 2^64 - 1 leaves.
	//
	// This need for this variable could be removed, but is kept around to enable
	// a simpler transition to little endian encoding if/when necessary.
	// Ref: https://github.com/pokt-network/smt/pull/46#discussion_r1636975124
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
	trieSpec := NewTrieSpec(hasher, true)
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
	smtSpec := TrieSpec{
		th:      NewTrieHasher(trieSpec.th.hasher),
		ph:      trieSpec.ph,
		vh:      trieSpec.vh,
		sumTrie: trieSpec.sumTrie,
	}
	smt := &SMT{
		TrieSpec: smtSpec,
		nodes:    nodes,
	}
	nilValueHasher := WithValueHasher(nil)
	nilValueHasher(&smt.TrieSpec)

	smstSpec := TrieSpec{
		th:      NewTrieHasher(trieSpec.th.hasher),
		ph:      trieSpec.ph,
		vh:      trieSpec.vh,
		sumTrie: trieSpec.sumTrie,
	}
	return &SMST{
		TrieSpec: smstSpec,
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

// ProveWeighted generates a proof of inclusion for a leaf selected in
// proportion to leaf count: every leaf in the trie is equally likely to be
// selected, whatever its position in the keyspace.
//
// The seed selects a target index t = BigEndian(seed[:8]) mod count. The trie
// is descended from the root, going left when t is less than the count of
// the left child, and otherwise subtracting that count and going right.
//
// The proof returned is the proof ProveClosest returns for the selected leaf's
// own path, built in the same descent that selects the leaf, so it can be
// compacted, marshaled and verified as a closest proof. Use
// VerifyWeightedProof to also check that the leaf is the one the seed selects.
func (smst *SMST) ProveWeighted(seed []byte) (*SparseMerkleClosestProof, error) {
	t, err := weightedTarget(seed, smst.Root())
	if err != nil {
		return nil, err
	}

	// siblings are collected top-down; lastSibling is the sibling of the
	// deepest inner node, whose encoding is returned as SiblingData, as
	// ProveClosest does.
	var siblings []trieNode
	var lastSibling trieNode
	depth := 0
	node := smst.root
	for {
		if node, err = smst.resolveLazy(node); err != nil {
			return nil, err
		}
		if leaf, ok := node.(*leafNode); ok {
			value, err := smst.leafValue(leaf)
			if err != nil {
				return nil, err
			}
			// Left nil for a root leaf, as ProveClosest leaves it.
			var sideNodes [][]byte
			if len(siblings) > 0 {
				sideNodes = make([][]byte, len(siblings))
			}
			for i, sibling := range siblings {
				sideNodes[len(siblings)-1-i] = smst.digest(sibling)
			}
			proof := &SparseMerkleClosestProof{
				Path:             leaf.path,
				FlippedBits:      make([]int, 0),
				Depth:            depth,
				ClosestPath:      leaf.path,
				ClosestValueHash: value,
				ClosestProof:     &SparseMerkleProof{SideNodes: sideNodes},
			}
			if lastSibling != nil {
				if lastSibling, err = smst.resolveLazy(lastSibling); err != nil {
					return nil, err
				}
				if proof.ClosestProof.SiblingData, err = smst.encodeWithValue(lastSibling); err != nil {
					return nil, err
				}
			}
			return proof, nil
		}
		switch n := node.(type) {
		case *extensionNode:
			// Each level an extension node replaces has an empty sibling.
			for i := 0; i < n.length(); i++ {
				siblings = append(siblings, nil)
			}
			depth += n.length()
			node = n.child
		case *innerNode:
			_, leftCount := parseSumAndCount(smst.digest(n.leftChild))
			if t < leftCount {
				node, lastSibling = n.leftChild, n.rightChild
			} else {
				t -= leftCount
				node, lastSibling = n.rightChild, n.leftChild
			}
			siblings = append(siblings, lastSibling)
			depth++
		default:
			return nil, fmt.Errorf("ProveWeighted: unexpected node %T", node)
		}
	}
}

// Commit persists all dirty nodes in the trie, deletes all orphaned
// nodes from the database and then computes and saves the root hash
func (smst *SMST) Commit() error {
	return smst.SMT.Commit()
}

// Root returns the root hash of the trie with the total sum bytes appended
func (smst *SMST) Root() MerkleSumRoot {
	return MerkleSumRoot(smst.SMT.Root()) // [digest]+[binary sum]+[binary count]
}

// MustSum returns the sum of the entire trie stored in the root.
// If the tree is not a sum tree, it will panic.
func (smst *SMST) MustSum() uint64 {
	sum, err := smst.Sum()
	if err != nil {
		panic(err)
	}
	return sum
}

// Sum returns the sum of the entire trie stored in the root.
// If the tree is not a sum tree, it will return an error.
func (smst *SMST) Sum() (uint64, error) {
	if !smst.Spec().sumTrie {
		return 0, fmt.Errorf("SMST: not a merkle sum trie")
	}

	return smst.Root().Sum()
}

// MustCount returns the number of non-empty nodes in the entire trie stored in the root.
// If the tree is not a sum tree, it will panic.
func (smst *SMST) MustCount() uint64 {
	count, err := smst.Count()
	if err != nil {
		panic(err)
	}
	return count
}

// Count returns the number of non-empty nodes in the entire trie stored in the root.
// If the tree is not a sum tree, it will return an error.
func (smst *SMST) Count() (uint64, error) {
	if !smst.Spec().sumTrie {
		return 0, fmt.Errorf("SMST: not a merkle sum trie")
	}

	return smst.Root().Count()
}

// getFirstMetaByteIdx returns the index of the first count byte and the first sum byte
// in the data slice provided. This is useful metadata when parsing the data
// of any node in the trie.
func getFirstMetaByteIdx(data []byte) (firstSumByteIdx, firstCountByteIdx int) {
	firstCountByteIdx = len(data) - countSizeBytes
	firstSumByteIdx = firstCountByteIdx - sumSizeBytes
	return firstSumByteIdx, firstCountByteIdx
}
