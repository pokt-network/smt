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

var errInjectedGet = errors.New("injected Get failure")

// failNthGetStore fails exactly the nth call to Get after it is armed and
// passes every other call through.
type failNthGetStore struct {
	kvstore.MapStore
	failAt int
	calls  int
}

func (s *failNthGetStore) Get(key []byte) ([]byte, error) {
	s.calls++
	if s.calls == s.failAt {
		return nil, errInjectedGet
	}
	return s.MapStore.Get(key)
}

// A store read that fails while an operation resolves a lazy node must leave
// the trie as it was, so that the same operation succeeds once the store
// recovers. The lazy node used to be replaced in place by the nil the resolve
// returned, so the subtree below it vanished from memory: a Get then read an
// existing key as absent, and an Update or Delete committed a root missing
// every leaf of that subtree, all without an error.
//
// Each operation runs on a trie imported from its root with one path already
// resident, and fails at its first read, then its second, and so on until it
// needs no more reads. After each failure the store recovers, the operation is
// retried, the trie is committed, and a trie imported from the new root must
// hold exactly the expected keys and values.
func TestStoreReadFailure_LeavesTheTrieIntact(t *testing.T) {
	rnd := rand.New(rand.NewSource(20260918))
	keys := make([][]byte, 40)
	values := make(map[string][]byte, len(keys))
	for i := range keys {
		keys[i] = make([]byte, 32)
		value := make([]byte, 64)
		rnd.Read(keys[i]) //nolint:errcheck // math/rand Read never returns an error
		rnd.Read(value)   //nolint:errcheck // math/rand Read never returns an error
		values[string(keys[i])] = value
	}
	newValue := bytes.Repeat([]byte{7}, 64)

	ops := []struct {
		name string
		run  func(trie *SMST) error
		want func() map[string][]byte
	}{
		{"get", func(trie *SMST) error {
			got, _, err := trie.Get(keys[1])
			if err == nil && !bytes.Equal(got, values[string(keys[1])]) {
				return fmt.Errorf("Get returned %d bytes, want the stored value", len(got))
			}
			return err
		}, func() map[string][]byte { return values }},
		{"update", func(trie *SMST) error {
			return trie.Update(keys[1], newValue, 1)
		}, func() map[string][]byte {
			want := cloneValues(values)
			want[string(keys[1])] = newValue
			return want
		}},
		{"delete", func(trie *SMST) error {
			return trie.Delete(keys[1])
		}, func() map[string][]byte {
			want := cloneValues(values)
			delete(want, string(keys[1]))
			return want
		}},
	}

	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			failures := 0
			for n := 1; ; n++ {
				inner := simplemap.NewSimpleMap()
				src := newPoktrollSpecSMST(inner)
				for _, key := range keys {
					requireNoError(t, src.Update(key, values[string(key)], 1), "Update")
				}
				requireNoError(t, src.Commit(), "Commit")

				store := &failNthGetStore{MapStore: inner}
				trie := ImportSparseMerkleSumTrie(store, sha256.New(), src.Root(), WithValueHasher(nil))
				_, _, err := trie.Get(keys[0])
				requireNoError(t, err, "Get resolving one path")

				store.failAt = store.calls + n
				err = op.run(trie)
				if !errors.Is(err, errInjectedGet) {
					requireNoError(t, err, fmt.Sprintf("%s needing fewer than %d reads", op.name, n))
					break
				}
				failures++

				store.failAt = 0
				requireNoError(t, op.run(trie), fmt.Sprintf("%s retried after read %d failed", op.name, n))
				requireNoError(t, trie.Commit(), "Commit")

				want := op.want()
				imported := ImportSparseMerkleSumTrie(inner, sha256.New(), trie.Root(), WithValueHasher(nil))
				count, err := imported.Count()
				requireNoError(t, err, "Count")
				if count != uint64(len(want)) {
					t.Fatalf("read %d failed: the committed root counts %d leaves, want %d", n, count, len(want))
				}
				for k, v := range want {
					got, _, err := imported.Get([]byte(k))
					requireNoError(t, err, "Get from the committed root")
					if !bytes.Equal(got, v) {
						t.Fatalf("read %d failed: key %x reads %d bytes from the committed root, want %d",
							n, k, len(got), len(v))
					}
				}
			}
			if failures < 2 {
				t.Fatalf("CONTROL: %s failed at only %d reads; it never resolved more than one lazy node", op.name, failures)
			}
			t.Logf("%s: failed and recovered at each of its %d store reads", op.name, failures)
		})
	}
}

func cloneValues(in map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
