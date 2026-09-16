package smt

import (
	"crypto/sha256"
	"errors"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// A compaction pass stops at a node marked compactedSubtree, so every read that
// can leave a value in the resident trie below such a node has to clear the
// marks on its way down. Mutations clear them through setDirty; Get and a
// delete that misses do not dirty anything, so each is pinned here: after it,
// the next pass must reach and drop every value it left behind. Proofs are
// pinned the other way: they must leave the trie untouched.

// importedWithOneCompactedPath returns a trie imported from the root of a
// committed trie, with one path resolved and compacted: its ancestors are
// marked while their other children are still lazy.
func importedWithOneCompactedPath(t *testing.T, seed int64) *SMST {
	t.Helper()
	store := simplemap.NewSimpleMap()
	src := newPoktrollSpecSMST(store)
	ops := generateOps(seed, 400)
	for _, o := range ops {
		requireNoError(t, src.Update(o.key, o.value, o.weight), "src.Update")
	}
	requireNoError(t, src.Commit(), "src.Commit")

	imported := ImportSparseMerkleSumTrie(store, sha256.New(), src.Root(), WithValueHasher(nil))
	_, _, err := imported.Get(ops[0].key)
	requireNoError(t, err, "Get")
	imported.CompactPersistedLeaves()
	if held, total := residentLeavesHoldingValues(imported.root); total == 0 || held != 0 {
		t.Fatalf("CONTROL: want one resolved path with its leaf compacted, got %d of %d holding a value", held, total)
	}
	if unmarkedResidentInternalNodes(imported.root) != 0 {
		t.Fatal("CONTROL: the compacted path's ancestors are not marked, so nothing below can be skipped")
	}
	return imported
}

// A miss still resolves the nodes on its path, and a leaf resolved from the
// store carries its value, but a miss dirties nothing.
func TestCompact_MissesUnderMarkedAncestors(t *testing.T) {
	misses := map[string]func(trie *SMST, key []byte) error{
		"delete of an absent key": func(trie *SMST, key []byte) error {
			if err := trie.Delete(key); !errors.Is(err, ErrKeyNotFound) {
				return err
			}
			return nil
		},
		"get of an absent key": func(trie *SMST, key []byte) error {
			_, _, err := trie.Get(key)
			return err
		},
	}
	for name, miss := range misses {
		t.Run(name, func(t *testing.T) {
			trie := importedWithOneCompactedPath(t, 101)
			rnd := rand.New(rand.NewSource(103))
			key := make([]byte, 32)
			for i := 0; i < 200; i++ {
				rnd.Read(key) //nolint:errcheck // math/rand Read never returns an error
				requireNoError(t, miss(trie, key), name)
			}
			held, total := residentLeavesHoldingValues(trie.root)
			if held == 0 {
				t.Fatalf("CONTROL: 200 misses left no leaf value resident (%d resident leaves); "+
					"none of them landed on a lazy leaf", total)
			}

			if compacted := trie.CompactPersistedLeaves(); compacted != held {
				t.Fatalf("pass compacted %d leaves, want the %d the misses left holding a value", compacted, held)
			}
			if held, total := residentLeavesHoldingValues(trie.root); held != 0 {
				t.Fatalf("%d of %d leaves still hold a value: the pass stopped at an ancestor the misses left marked",
					held, total)
			}
		})
	}
}

// Prove and ProveClosest read the value of a compacted leaf from the store into
// the proof and leave the trie exactly as they found it: no leaf is hydrated,
// no mark is cleared. That is what keeps them pure reads, as they are on a trie
// that was never compacted.
func TestCompact_ProofsLeaveTheTrieUntouched(t *testing.T) {
	ops := generateOps(107, 400)
	keys := distinctKeysInOrder(ops)
	proofs := map[string]func(trie *SMST, i int, absent []byte) error{
		"prove of a present key": func(trie *SMST, i int, _ []byte) error {
			_, err := trie.Prove(keys[i])
			return err
		},
		"prove of an absent key": func(trie *SMST, _ int, absent []byte) error {
			_, err := trie.Prove(absent)
			return err
		},
		"prove closest": func(trie *SMST, _ int, absent []byte) error {
			_, err := trie.ProveClosest(absent)
			return err
		},
	}
	for name, prove := range proofs {
		t.Run(name, func(t *testing.T) {
			trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
			for _, o := range ops {
				requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
			}
			requireNoError(t, trie.Commit(), "Commit")
			trie.CompactPersistedLeaves()
			if held, total := residentLeavesHoldingValues(trie.root); total == 0 || held != 0 {
				t.Fatalf("CONTROL: want every resident leaf compacted, got %d of %d holding a value", held, total)
			}
			before := snapshotResidentNodes(trie.root)
			if unmarkedResidentInternalNodes(trie.root) != 0 {
				t.Fatal("CONTROL: a compacted trie must have every internal node marked")
			}

			rnd := rand.New(rand.NewSource(109))
			absent := make([]byte, 32)
			for i := 0; i < 40; i++ {
				rnd.Read(absent) //nolint:errcheck // math/rand Read never returns an error
				requireNoError(t, prove(trie, i, absent), name)
			}

			after := snapshotResidentNodes(trie.root)
			if len(after) != len(before) {
				t.Fatalf("proofs changed the resident trie from %d to %d nodes", len(before), len(after))
			}
			for node, was := range before {
				if now := after[node]; now != was {
					t.Fatalf("proofs changed a resident %T: before %+v, after %+v", node, was, now)
				}
			}
		})
	}
}

// Two ProveClosest calls at once on the same trie must not race, compacted or
// not: neither writes to a node. Run under -race, which CI does. (Prove is not
// covered: it shares the path hasher, which races on its own.)
func TestCompact_ConcurrentProofsDoNotRace(t *testing.T) {
	ops := generateOps(127, 300)
	for _, compact := range []bool{false, true} {
		name := "never compacted"
		if compact {
			name = "compacted"
		}
		t.Run(name, func(t *testing.T) {
			trie := newPoktrollSpecSMST(simplemap.NewSimpleMap())
			for _, o := range ops {
				requireNoError(t, trie.Update(o.key, o.value, o.weight), "Update")
			}
			requireNoError(t, trie.Commit(), "Commit")
			if compact {
				if trie.CompactPersistedLeaves() == 0 {
					t.Fatal("CONTROL: compaction compacted nothing")
				}
			}

			const workers = 4
			errs := make(chan error, workers)
			for w := 0; w < workers; w++ {
				go func(w int) {
					path := make([]byte, 32)
					rnd := rand.New(rand.NewSource(int64(131 + w)))
					for i := 0; i < 100; i++ {
						rnd.Read(path) //nolint:errcheck // math/rand Read never returns an error
						if _, err := trie.ProveClosest(path); err != nil {
							errs <- err
							return
						}
					}
					errs <- nil
				}(w)
			}
			for w := 0; w < workers; w++ {
				requireNoError(t, <-errs, "ProveClosest")
			}
		})
	}
}

// nodeState is what a read could change on a resident node.
type nodeState struct {
	compactedSubtree bool
	compacted        bool
	holdsValue       bool
	persisted        bool
	digest           string
}

// snapshotResidentNodes records the state of every resident node, keyed by the
// node itself, so a before/after comparison sees any in-place change.
func snapshotResidentNodes(root trieNode) map[trieNode]nodeState {
	out := make(map[trieNode]nodeState)
	var walk func(node trieNode)
	walk = func(node trieNode) {
		switch n := node.(type) {
		case *leafNode:
			out[n] = nodeState{compacted: n.compacted, holdsValue: n.valueHash != nil,
				persisted: n.persisted, digest: string(n.digest)}
		case *innerNode:
			out[n] = nodeState{compactedSubtree: n.compactedSubtree, persisted: n.persisted, digest: string(n.digest)}
			walk(n.leftChild)
			walk(n.rightChild)
		case *extensionNode:
			out[n] = nodeState{compactedSubtree: n.compactedSubtree, persisted: n.persisted, digest: string(n.digest)}
			walk(n.child)
		}
	}
	walk(root)
	return out
}

// What a pass costs is the set of resident inner and extension nodes without a
// mark: a marked node returns at once. On a trie imported from the store, that
// set must stay the path an update changed, and every node must be marked again
// once the pass is done. Were reads or resolves to fall back on a walk of the
// whole trie, or were marks not restored, this set would grow with the trie.
func TestCompact_PassCostStaysOnTheChangedPath(t *testing.T) {
	const (
		initialLeaves = 2000
		updates       = 2000
		// Far above the depth of a 4k-leaf trie over sha256 paths, far below
		// the thousands of internal nodes it holds.
		maxUnmarked = 64
	)
	store := simplemap.NewSimpleMap()
	src := newPoktrollSpecSMST(store)
	rnd := rand.New(rand.NewSource(113))
	put := func(trie *SMST) {
		key := make([]byte, 32)
		value := make([]byte, 64)
		rnd.Read(key)   //nolint:errcheck // math/rand Read never returns an error
		rnd.Read(value) //nolint:errcheck // math/rand Read never returns an error
		requireNoError(t, trie.Update(key, value, 1), "Update")
		requireNoError(t, trie.Commit(), "Commit")
	}
	for i := 0; i < initialLeaves; i++ {
		put(src)
	}

	trie := ImportSparseMerkleSumTrie(store, sha256.New(), src.Root(), WithValueHasher(nil))
	worst := 0
	for i := 0; i < updates; i++ {
		put(trie)
		unmarked := unmarkedResidentInternalNodes(trie.root)
		if unmarked > worst {
			worst = unmarked
		}
		if unmarked > maxUnmarked {
			t.Fatalf("update %d: the next pass would visit %d unmarked internal nodes, want at most %d",
				i, unmarked, maxUnmarked)
		}
		trie.CompactPersistedLeaves()
		if left := unmarkedResidentInternalNodes(trie.root); left != 0 {
			t.Fatalf("update %d: %d internal nodes are still unmarked after the pass, so every later pass visits them",
				i, left)
		}
	}
	held, total := residentLeavesHoldingValues(trie.root)
	if total < updates || held != 0 {
		t.Fatalf("CONTROL: want at least %d resident leaves all compacted, got %d of %d holding a value",
			updates, held, total)
	}
	t.Logf("worst unmarked internal nodes before a pass: %d over %d resident leaves", worst, total)
}

// unmarkedResidentInternalNodes counts the resident inner and extension nodes
// that a compaction pass would descend into.
func unmarkedResidentInternalNodes(node trieNode) int {
	switch n := node.(type) {
	case *innerNode:
		count := unmarkedResidentInternalNodes(n.leftChild) + unmarkedResidentInternalNodes(n.rightChild)
		if !n.compactedSubtree {
			count++
		}
		return count
	case *extensionNode:
		count := unmarkedResidentInternalNodes(n.child)
		if !n.compactedSubtree {
			count++
		}
		return count
	}
	return 0
}
