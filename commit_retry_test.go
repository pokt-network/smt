package smt

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

var errInjectedSet = errors.New("injected Set failure")

// failNthSetStore fails exactly the nth call to Set and passes every other
// call through, so a Commit can be made to stop part-way.
type failNthSetStore struct {
	kvstore.MapStore
	failAt int
	calls  int
}

func (s *failNthSetStore) Set(key, value []byte) error {
	s.calls++
	if s.calls == s.failAt {
		return errInjectedSet
	}
	return s.MapStore.Set(key, value)
}

// A Commit that fails part-way must be retryable. commit marks a node
// persisted and skips persisted nodes, so a node marked before its Set
// succeeded is never written by the retry, and neither is anything above it:
// before the fix, failing the very first Set left a retried Commit returning
// nil over an empty store.
//
// The failure is injected at the first Set (the deepest node, so every
// ancestor was already marked), at one in the middle, and at the last (the
// root). After the retry, the store must hold exactly what a trie committed
// without failures holds, and a trie imported from the root must read every
// key back.
func TestCommit_RetryAfterFailedSetWritesEveryNode(t *testing.T) {
	const numKeys = 50
	rnd := rand.New(rand.NewSource(20260916))
	keys := make([][]byte, numKeys)
	values := make([][]byte, numKeys)
	for i := range keys {
		keys[i] = make([]byte, 32)
		values[i] = make([]byte, 64)
		rnd.Read(keys[i])   //nolint:errcheck // math/rand Read never returns an error
		rnd.Read(values[i]) //nolint:errcheck // math/rand Read never returns an error
	}

	ref := simplemap.NewSimpleMap()
	refTrie := newPoktrollSpecSMST(ref)
	for i := range keys {
		requireNoError(t, refTrie.Update(keys[i], values[i], uint64(i)+1), "ref Update")
	}
	requireNoError(t, refTrie.Commit(), "ref Commit")
	refLen, err := ref.Len()
	requireNoError(t, err, "ref Len")

	for _, failAt := range []int{1, refLen / 2, refLen} {
		t.Run(fmt.Sprintf("fail_set_%d_of_%d", failAt, refLen), func(t *testing.T) {
			inner := simplemap.NewSimpleMap()
			store := &failNthSetStore{MapStore: inner, failAt: failAt}
			trie := newPoktrollSpecSMST(store)
			for i := range keys {
				requireNoError(t, trie.Update(keys[i], values[i], uint64(i)+1), "Update")
			}

			if err := trie.Commit(); !errors.Is(err, errInjectedSet) {
				t.Fatalf("first Commit: got %v, want the injected Set failure", err)
			}
			requireNoError(t, trie.Commit(), "retried Commit")

			gotLen, err := inner.Len()
			requireNoError(t, err, "Len")
			if gotLen != refLen {
				t.Fatalf("store holds %d nodes after the retried Commit, want %d: "+
					"nodes marked persisted before a failed Set were never written", gotLen, refLen)
			}
			if !bytes.Equal(trie.Root(), refTrie.Root()) {
				t.Fatalf("root after retry %x, want %x", trie.Root(), refTrie.Root())
			}

			reimported := ImportSparseMerkleSumTrie(inner, sha256.New(), trie.Root(), WithValueHasher(nil))
			for i := range keys {
				got, weight, err := reimported.Get(keys[i])
				requireNoError(t, err, "Get from a trie imported after the retry")
				if !bytes.Equal(got, values[i]) || weight != uint64(i)+1 {
					t.Fatalf("key %d: imported trie returned value %x weight %d, want %x weight %d",
						i, got, weight, values[i], uint64(i)+1)
				}
			}
		})
	}
}
