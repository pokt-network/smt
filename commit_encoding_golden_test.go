package smt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// The bytes a trie writes and the roots and proofs it produces are consensus
// data: a relay miner signs the root and the chain verifies the proof. Any
// change to how nodes are encoded or hashed must reproduce them byte for byte.
// This test pins them to the output of v0.15.0 for both trie kinds, with the
// nil value hasher (what Pocket uses) and with the default one, across node
// sizes on both sides of reasonableNodeSize (256 B) and the buffer pool's cap
// (1 KiB), up to a 1 MiB leaf, with a last round on a trie imported from its
// committed root. It pins the root after every Commit and every stored node;
// on the final, imported trie, membership proofs of live keys and
// non-membership proofs of deleted keys (plain and compact, all verified); and
// closest proofs (plain and compact) on the trie before the import.
//
// Regenerate only from a version known to be correct:
//
//	SMT_UPDATE_GOLDEN=1 go test -run TestCommitEncodingGolden ./
const commitEncodingGoldenFile = "testdata/commit_encoding_golden.json"

type commitEncodingGolden struct {
	Roots          []string          `json:"roots"`
	StoreDigest    []string          `json:"store_digest"`
	StoredNodeSize map[string]int    `json:"stored_node_sizes"`
	Proofs         map[string]string `json:"proofs"`
	CompactProofs  map[string]string `json:"compact_proofs"`
	ClosestProofs  map[string]string `json:"closest_proofs"`
	CompactClosest map[string]string `json:"compact_closest_proofs"`
}

// chainedBytes is deterministic, incompressible filler: chained SHA-256.
func chainedBytes(seed string, n int) []byte {
	out := make([]byte, 0, n+sha256.Size)
	prev := sha256.Sum256([]byte(seed))
	for len(out) < n {
		out = append(out, prev[:]...)
		prev = sha256.Sum256(prev[:])
	}
	return out[:n]
}

var commitEncodingSizes = []int{1, 32, 200, 256, 257, 600, 1023, 1024, 1025, 4096, 1 << 20}

// storeDigest hashes every (key, value) the store holds, in key order: two
// runs that wrote the same nodes produce the same digest.
func storeDigest(m map[string][]byte) (string, map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	sizes := map[string]int{}
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(m[k])
		switch {
		case len(m[k]) > 1<<20:
			sizes["over_1MiB"]++
		case len(m[k]) > 1024:
			sizes["over_1KiB"]++
		case len(m[k]) > 256:
			sizes["over_256B"]++
		default:
			sizes["small"]++
		}
	}
	return hex.EncodeToString(h.Sum(nil)), sizes
}

// goldenTrie is what the scenario needs from either trie kind.
type goldenTrie struct {
	update       func(key, value []byte, weight uint64) error
	delete       func(key []byte) error
	commit       func() error
	root         func() []byte
	prove        func(key []byte) (*SparseMerkleProof, error)
	proveClosest func(path []byte) (*SparseMerkleClosestProof, error)
	spec         *TrieSpec
	// verify checks a (non-)membership proof; value nil and weight 0 prove absence.
	verify        func(proof *SparseMerkleProof, root, key, value []byte, weight uint64) (bool, error)
	verifyCompact func(proof *SparseCompactMerkleProof, root, key, value []byte, weight uint64) (bool, error)
}

func newGoldenTrie(sum bool, store *map[string][]byte, root []byte, opts ...TrieSpecOption) goldenTrie {
	nodes := simplemap.NewSimpleMapWithMap(*store)
	if sum {
		var s *SMST
		if root == nil {
			s = NewSparseMerkleSumTrie(nodes, sha256.New(), opts...)
		} else {
			s = ImportSparseMerkleSumTrie(nodes, sha256.New(), root, opts...)
		}
		count := func(value []byte) uint64 {
			if value == nil {
				return 0
			}
			return 1
		}
		return goldenTrie{
			update: s.Update, delete: s.Delete, commit: s.Commit,
			root: func() []byte { return s.Root() }, prove: s.Prove, proveClosest: s.ProveClosest, spec: s.Spec(),
			verify: func(p *SparseMerkleProof, root, key, value []byte, weight uint64) (bool, error) {
				return VerifySumProof(p, root, key, value, weight, count(value), s.Spec())
			},
			verifyCompact: func(p *SparseCompactMerkleProof, root, key, value []byte, weight uint64) (bool, error) {
				return VerifyCompactSumProof(p, root, key, value, weight, count(value), s.Spec())
			},
		}
	}
	var t *SMT
	if root == nil {
		t = NewSparseMerkleTrie(nodes, sha256.New(), opts...)
	} else {
		t = ImportSparseMerkleTrie(nodes, sha256.New(), root, opts...)
	}
	return goldenTrie{
		update: func(key, value []byte, _ uint64) error { return t.Update(key, value) },
		delete: t.Delete, commit: t.Commit, root: func() []byte { return t.Root() }, prove: t.Prove, proveClosest: t.ProveClosest, spec: t.Spec(),
		verify: func(p *SparseMerkleProof, root, key, value []byte, _ uint64) (bool, error) {
			return VerifyProof(p, root, key, value, t.Spec())
		},
		verifyCompact: func(p *SparseCompactMerkleProof, root, key, value []byte, _ uint64) (bool, error) {
			return VerifyCompactProof(p, root, key, value, t.Spec())
		},
	}
}

func runCommitEncodingScenario(t *testing.T, sum, nilValueHasher bool) commitEncodingGolden {
	t.Helper()
	var opts []TrieSpecOption
	if nilValueHasher {
		opts = append(opts, WithValueHasher(nil))
	}
	nodes := map[string][]byte{}
	trie := newGoldenTrie(sum, &nodes, nil, opts...)

	out := commitEncodingGolden{
		Proofs: map[string]string{}, CompactProofs: map[string]string{},
		ClosestProofs: map[string]string{}, CompactClosest: map[string]string{},
	}
	record := func() {
		require.NoError(t, trie.commit())
		out.Roots = append(out.Roots, hex.EncodeToString(trie.root()))
		d, sizes := storeDigest(nodes)
		out.StoreDigest = append(out.StoreDigest, d)
		out.StoredNodeSize = sizes
	}

	key := func(label string) []byte { k := sha256.Sum256([]byte(label)); return k[:] }
	type entry struct {
		value  []byte
		weight uint64
	}
	live := map[string]entry{}
	var deleted []string
	put := func(label string, value []byte, weight uint64) {
		live[label] = entry{value, weight}
		require.NoError(t, trie.update(key(label), bytesCopy(value), weight))
	}
	remove := func(label string) {
		require.NoError(t, trie.delete(key(label)))
		delete(live, label)
		deleted = append(deleted, label)
	}

	// Round 1: every size, plus enough small leaves to build inner and
	// extension nodes at several depths.
	for i, n := range commitEncodingSizes {
		l := fmt.Sprintf("size-%d", n)
		put(l, chainedBytes(l, n), uint64(i+1))
	}
	for i := 0; i < 200; i++ {
		l := fmt.Sprintf("small-%d", i)
		put(l, chainedBytes(l, 64), uint64(i%7+1))
	}
	record()

	// Round 2: overwrite big and small leaves, delete some, add new ones.
	for _, n := range []int{257, 1024, 1 << 20} {
		l := fmt.Sprintf("size-%d", n)
		put(l, chainedBytes(l+"-v2", n), 99)
	}
	for i := 0; i < 200; i += 3 {
		remove(fmt.Sprintf("small-%d", i))
	}
	for i := 0; i < 50; i++ {
		l := fmt.Sprintf("late-%d", i)
		put(l, chainedBytes(l, 300+i*37), 3)
	}
	record()

	// Closest proofs are pinned on this trie, before the import below. On an
	// imported trie, ProveClosest in v0.15.0 (and v0.14.1) returns an empty
	// proof for some paths: an empty child of an imported trie is a lazy
	// placeholder, not nil, and the walk takes it as the parent. That is a
	// separate issue, not what this test is about.
	closestRoot := trie.root()
	for i := 0; i < 64; i++ {
		path := sha256.Sum256([]byte(fmt.Sprintf("closest-%d", i)))
		proof, err := trie.proveClosest(path[:])
		require.NoError(t, err)
		if nilValueHasher {
			// Only checked with the nil value hasher, which is what closest
			// proofs are used with; with a value hasher VerifyClosestProof
			// returns false in v0.15.0 as well.
			valid, err := VerifyClosestProof(proof, closestRoot, trie.spec)
			require.NoError(t, err)
			require.True(t, valid, "closest proof %d must verify against the root", i)
		}
		bz, err := proof.Marshal()
		require.NoError(t, err)
		out.ClosestProofs[fmt.Sprint(i)] = hexSum(bz)

		compact, err := CompactClosestProof(proof, trie.spec)
		require.NoError(t, err)
		bz, err = compact.Marshal()
		require.NoError(t, err)
		out.CompactClosest[fmt.Sprint(i)] = hexSum(bz)
	}

	// Round 3: on a trie imported from the committed root, over the same
	// store, as a restarted process reloads it.
	trie = newGoldenTrie(sum, &nodes, trie.root(), opts...)
	put("size-1025", chainedBytes("size-1025-v3", 1025), 5)
	put("imported-big", chainedBytes("imported-big", 1<<20), 8)
	remove("late-7")
	remove("small-1")
	record()

	root := trie.root()
	pin := func(label string, k, value []byte, weight uint64) {
		proof, err := trie.prove(k)
		require.NoError(t, err)
		valid, err := trie.verify(proof, root, k, value, weight)
		require.NoError(t, err)
		require.True(t, valid, "proof of %q must verify against the root", label)
		bz, err := proof.Marshal()
		require.NoError(t, err)
		out.Proofs[label] = hexSum(bz)

		compact, err := CompactProof(proof, trie.spec)
		require.NoError(t, err)
		valid, err = trie.verifyCompact(compact, root, k, value, weight)
		require.NoError(t, err)
		require.True(t, valid, "compact proof of %q must verify against the root", label)
		bz, err = compact.Marshal()
		require.NoError(t, err)
		out.CompactProofs[label] = hexSum(bz)
	}
	labels := make([]string, 0, len(live))
	for l := range live {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		pin(l, key(l), live[l].value, live[l].weight)
	}
	for _, l := range deleted {
		pin("absent/"+l, key(l), nil, 0)
	}

	return out
}

func hexSum(bz []byte) string { s := sha256.Sum256(bz); return hex.EncodeToString(s[:]) }

func bytesCopy(b []byte) []byte { return append([]byte(nil), b...) }

func TestCommitEncodingGolden(t *testing.T) {
	variants := []struct {
		kind           string
		sum            bool
		nilValueHasher bool
	}{
		{"smst/nil-value-hasher", true, true},
		{"smst/default-value-hasher", true, false},
		{"smt/nil-value-hasher", false, true},
		{"smt/default-value-hasher", false, false},
	}
	got := map[string]commitEncodingGolden{}
	for _, v := range variants {
		t.Run(v.kind, func(t *testing.T) { got[v.kind] = runCommitEncodingScenario(t, v.sum, v.nilValueHasher) })
	}
	if t.Failed() {
		return
	}
	require.Equal(t, 2, got["smst/nil-value-hasher"].StoredNodeSize["over_1MiB"],
		"premise: both 1 MiB leaves' nodes are among the stored nodes")

	if os.Getenv("SMT_UPDATE_GOLDEN") == "1" {
		bz, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(commitEncodingGoldenFile, append(bz, '\n'), 0o644))
		t.Logf("wrote %s", commitEncodingGoldenFile)
		return
	}
	bz, err := os.ReadFile(commitEncodingGoldenFile)
	require.NoError(t, err)
	var want map[string]commitEncodingGolden
	require.NoError(t, json.Unmarshal(bz, &want))
	require.Len(t, want, len(got), "the golden must cover every trie variant the test runs")
	for kind := range want {
		require.Equal(t, want[kind].Roots, got[kind].Roots, "%s: roots differ from v0.15.0", kind)
		require.Equal(t, want[kind].StoreDigest, got[kind].StoreDigest, "%s: the nodes written differ from v0.15.0", kind)
		require.Equal(t, want[kind].StoredNodeSize, got[kind].StoredNodeSize, "%s: stored node sizes differ from v0.15.0", kind)
		require.Equal(t, want[kind].Proofs, got[kind].Proofs, "%s: proofs differ from v0.15.0", kind)
		require.Equal(t, want[kind].CompactProofs, got[kind].CompactProofs, "%s: compact proofs differ from v0.15.0", kind)
		require.Equal(t, want[kind].ClosestProofs, got[kind].ClosestProofs, "%s: closest proofs differ from v0.15.0", kind)
		require.Equal(t, want[kind].CompactClosest, got[kind].CompactClosest, "%s: compact closest proofs differ from v0.15.0", kind)
	}
}
