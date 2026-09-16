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

var errInjectedDelete = errors.New("injected Delete failure")

// flakyStore fails the next Set or Delete when armed, and passes everything
// else through, so a Commit can be made to fail at a chosen step.
type flakyStore struct {
	kvstore.MapStore
	failSet, failDelete bool
}

func (s *flakyStore) Set(key, value []byte) error {
	if s.failSet {
		s.failSet = false
		return errInjectedSet
	}
	return s.MapStore.Set(key, value)
}

func (s *flakyStore) Delete(key []byte) error {
	if s.failDelete {
		s.failDelete = false
		return errInjectedDelete
	}
	return s.MapStore.Delete(key)
}

// commitScenario builds a committed trie over store, then applies a second
// batch that orphans persisted nodes (an overwrite and a delete) and adds a key,
// without committing it. It returns the first root, the values it held, the
// second batch's final values, and a reference store holding exactly what a
// successful second Commit leaves behind.
func commitScenario(t *testing.T, store kvstore.MapStore) (
	trie *SMST, firstRoot []byte, first, second map[string][]byte, ref kvstore.MapStore,
) {
	t.Helper()
	rnd := rand.New(rand.NewSource(20260917))
	first = make(map[string][]byte)
	var keys [][]byte
	for i := 0; i < 50; i++ {
		key, value := make([]byte, 32), make([]byte, 64)
		rnd.Read(key)   //nolint:errcheck // math/rand Read never returns an error
		rnd.Read(value) //nolint:errcheck // math/rand Read never returns an error
		keys = append(keys, key)
		first[string(key)] = value
	}
	build := func(s kvstore.MapStore) *SMST {
		tr := newPoktrollSpecSMST(s)
		for _, key := range keys {
			requireNoError(t, tr.Update(key, first[string(key)], 1), "Update")
		}
		requireNoError(t, tr.Commit(), "first Commit")
		return tr
	}
	second = make(map[string][]byte, len(first))
	for k, v := range first {
		second[k] = v
	}
	newKey, newValue := make([]byte, 32), make([]byte, 64)
	rnd.Read(newKey)   //nolint:errcheck // math/rand Read never returns an error
	rnd.Read(newValue) //nolint:errcheck // math/rand Read never returns an error
	overwritten := append([]byte(nil), first[string(keys[0])]...)
	overwritten[0] ^= 0xff
	secondBatch := func(tr *SMST) {
		requireNoError(t, tr.Update(keys[0], overwritten, 1), "overwrite")
		requireNoError(t, tr.Delete(keys[1]), "Delete")
		requireNoError(t, tr.Update(newKey, newValue, 1), "Update new key")
	}
	second[string(keys[0])] = overwritten
	delete(second, string(keys[1]))
	second[string(newKey)] = newValue

	ref = simplemap.NewSimpleMap()
	refTrie := build(ref)
	secondBatch(refTrie)
	requireNoError(t, refTrie.Commit(), "ref second Commit")

	trie = build(store)
	firstRoot = append([]byte(nil), trie.Root()...)
	secondBatch(trie)
	return trie, firstRoot, first, second, ref
}

// requireStoreServes imports root from store and requires it to read exactly
// the given values.
func requireStoreServes(t *testing.T, store kvstore.MapStore, root []byte, values map[string][]byte, what string) {
	t.Helper()
	imported := ImportSparseMerkleSumTrie(store, sha256.New(), root, WithValueHasher(nil))
	for k, want := range values {
		got, _, err := imported.Get([]byte(k))
		if err != nil {
			t.Fatalf("%s: cannot read key %x: %v", what, k, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: key %x reads %d bytes, want %d", what, k, len(got), len(want))
		}
	}
}

// A Commit whose write fails must leave the store serving the last committed
// root: nothing has replaced it yet. Deleting the orphans before writing broke
// that, because the orphans are nodes of that root.
func TestCommit_FailedWriteKeepsTheLastCommittedRoot(t *testing.T) {
	store := &flakyStore{MapStore: simplemap.NewSimpleMap()}
	trie, firstRoot, first, second, ref := commitScenario(t, store)

	store.failSet = true
	if err := trie.Commit(); !errors.Is(err, errInjectedSet) {
		t.Fatalf("Commit: got %v, want the injected Set failure", err)
	}
	requireStoreServes(t, store, firstRoot, first, "after a failed Commit, the last committed root")

	// And the orphans are still owed: the retry must leave exactly what a
	// Commit that never failed leaves, with no node of the first root behind.
	requireNoError(t, trie.Commit(), "retried Commit")
	requireStoreServes(t, store, trie.Root(), second, "after the retry, the new root")
	requireSameStoreSize(t, store, ref)
}

// A Commit whose orphan delete fails has already written the new root; the
// orphans left undeleted must be deleted by the next Commit.
func TestCommit_FailedDeleteKeepsOrphansForTheNextCommit(t *testing.T) {
	store := &flakyStore{MapStore: simplemap.NewSimpleMap()}
	trie, _, _, second, ref := commitScenario(t, store)

	store.failDelete = true
	if err := trie.Commit(); !errors.Is(err, errInjectedDelete) {
		t.Fatalf("Commit: got %v, want the injected Delete failure", err)
	}
	requireStoreServes(t, store, trie.Root(), second, "after a failed orphan delete, the new root")

	requireNoError(t, trie.Commit(), "retried Commit")
	requireSameStoreSize(t, store, ref)
}

// Setting a key back to a value it held gives the new leaf the digest of the
// leaf it orphans. That digest must survive the Commit: writing the new nodes
// first and then deleting every orphan would delete the leaf just written.
func TestCommit_KeySetBackToItsValueKeepsItsNode(t *testing.T) {
	key, value, other := bytes.Repeat([]byte{1}, 32), []byte("value"), []byte("other")
	cases := map[string]func(tr *SMST){
		"same value again": func(tr *SMST) {
			requireNoError(t, tr.Update(key, value, 1), "Update same value")
		},
		"changed and changed back in one batch": func(tr *SMST) {
			requireNoError(t, tr.Update(key, other, 1), "Update other")
			requireNoError(t, tr.Update(key, value, 1), "Update back")
		},
		"deleted and set again in one batch": func(tr *SMST) {
			requireNoError(t, tr.Delete(key), "Delete")
			requireNoError(t, tr.Update(key, value, 1), "Update again")
		},
	}
	for name, batch := range cases {
		t.Run(name, func(t *testing.T) {
			store := simplemap.NewSimpleMap()
			trie := newPoktrollSpecSMST(store)
			requireNoError(t, trie.Update(bytes.Repeat([]byte{2}, 32), other, 1), "Update neighbour")
			requireNoError(t, trie.Update(key, value, 1), "Update")
			requireNoError(t, trie.Commit(), "Commit")

			batch(trie)
			requireNoError(t, trie.Commit(), "Commit after the batch")
			requireStoreServes(t, store, trie.Root(), map[string][]byte{string(key): value}, name)
		})
	}
}

func requireSameStoreSize(t *testing.T, got, want kvstore.MapStore) {
	t.Helper()
	gotLen, err := got.Len()
	requireNoError(t, err, "Len")
	wantLen, err := want.Len()
	requireNoError(t, err, "reference Len")
	if gotLen != wantLen {
		t.Fatalf("store holds %d nodes, want %d: orphans were leaked or live nodes deleted", gotLen, wantLen)
	}
}
