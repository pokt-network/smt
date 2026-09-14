package smt

import (
	"crypto/sha256"
	"testing"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// countingStore wraps a MapStore and counts the calls made against it, so a
// test can assert WHERE the store is touched rather than only that it works.
// C5 uses it to pin that the hot insert path stays free of reads.
type countingStore struct {
	inner kvstore.MapStore
	gets  int
	sets  int
	dels  int
}

func newCountingStore() *countingStore {
	return &countingStore{inner: simplemap.NewSimpleMap()}
}

func (s *countingStore) Get(key []byte) ([]byte, error) {
	s.gets++
	return s.inner.Get(key)
}

func (s *countingStore) Set(key, value []byte) error {
	s.sets++
	return s.inner.Set(key, value)
}

func (s *countingStore) Delete(key []byte) error {
	s.dels++
	return s.inner.Delete(key)
}

func (s *countingStore) Len() (int, error) { return s.inner.Len() }

func (s *countingStore) ClearAll() error { return s.inner.ClearAll() }

func (s *countingStore) reset() { s.gets, s.sets, s.dels = 0, 0, 0 }

// newPoktrollSpecSMST builds an SMST with the same spec poktroll uses for the
// claim/proof trie: a sum trie over sha256 with a NIL value hasher, so the leaf
// carries the RAW value that the chain deserialises at proof validation time.
//
// Mirrors github.com/pokt-network/poktroll@v0.1.35
// pkg/crypto/protocol/hasher.go:
//
//	NewTrieHasher     = sha256.New
//	SMTValueHasher()  = smt.WithValueHasher(nil)
//
// poktroll is NOT imported here: this repository must not depend on it. The
// spec is replicated, and any divergence would show up as a different root.
func newPoktrollSpecSMST(nodes kvstore.MapStore) *SMST {
	return NewSparseMerkleSumTrie(nodes, sha256.New(), WithValueHasher(nil))
}

// requireNoError fails the test with the given message when err is non-nil.
func requireNoError(t testing.TB, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}
