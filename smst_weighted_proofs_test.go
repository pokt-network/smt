package smt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore/simplemap"
)

// seedFor returns a seed that selects target index i in any trie with a count
// greater than i.
func seedFor(i uint64) []byte {
	seed := make([]byte, 32)
	binary.BigEndian.PutUint64(seed, i)
	return seed
}

// Closest proofs, and so weighted proofs, verify only in tries without a
// value hasher, as poktroll configures them.
func newWeightedTestTrie(t *testing.T, numLeaves int) (*SMST, [][]byte) {
	t.Helper()
	trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
	var paths [][]byte
	for i := 0; i < numLeaves; i++ {
		key := []byte{byte(i >> 8), byte(i)}
		require.NoError(t, trie.Update(key, key, uint64(i+1)))
		paths = append(paths, trie.Spec().ph.Path(key))
	}
	sort.Slice(paths, func(i, j int) bool { return bytes.Compare(paths[i], paths[j]) < 0 })
	return trie, paths
}

func requireWeightedValid(t *testing.T, trie *SMST, seed []byte) *SparseMerkleClosestProof {
	t.Helper()
	proof, err := trie.ProveWeighted(seed)
	require.NoError(t, err)
	valid, err := VerifyWeightedProof(proof, trie.Root(), seed, trie.Spec())
	require.NoError(t, err)
	require.True(t, valid)
	return proof
}

// Target index i selects the i-th leaf in path order, so every leaf is
// selected by exactly one index in [0, count).
func TestSMST_ProveWeighted_SelectsLeafByIndex(t *testing.T) {
	trie, paths := newWeightedTestTrie(t, 300)
	require.NoError(t, trie.Commit())

	// Import the committed trie so the descent also resolves lazy nodes.
	trie = ImportSparseMerkleSumTrie(trie.nodes, sha256.New(), trie.Root(), WithValueHasher(nil))
	count := trie.MustCount()
	require.Equal(t, uint64(len(paths)), count)

	for i := uint64(0); i < count; i++ {
		proof := requireWeightedValid(t, trie, seedFor(i))
		require.Equal(t, paths[i], proof.ClosestPath, "index %d", i)
	}
	// Seeds are reduced modulo the count.
	proof := requireWeightedValid(t, trie, seedFor(count+5))
	require.Equal(t, paths[5], proof.ClosestPath)
}

func TestSMST_ProveWeighted_RandomSeedsAreUniform(t *testing.T) {
	const numLeaves, samples = 32, 32 * 400
	trie, _ := newWeightedTestTrie(t, numLeaves)
	rng := rand.New(rand.NewSource(1))
	hits := make(map[string]int)
	for i := 0; i < samples; i++ {
		seed := make([]byte, 32)
		rng.Read(seed)
		hits[string(requireWeightedValid(t, trie, seed).ClosestPath)]++
	}
	require.Len(t, hits, numLeaves)
	for _, n := range hits {
		// Expected 400 per leaf; the standard deviation is about 20.
		require.InDelta(t, 400, n, 120)
	}
}

// A subtree holding M of the trie's count is selected with probability
// M/(n+M), wherever its leaves sit in the keyspace. This rebuilds a trie with
// n leaves at random paths and M leaves sharing a 10-bit path prefix with one
// of them.
func TestSMST_ProveWeighted_DenseSubtree(t *testing.T) {
	const n, m, prefixBits = 8, 10000, 10
	rng := rand.New(rand.NewSource(2))
	randBytes := func() []byte { b := make([]byte, 32); rng.Read(b); return b }

	trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
	var prefixPath []byte
	for i := 0; i < n; i++ {
		key := randBytes()
		if i == 0 {
			prefixPath = trie.Spec().ph.Path(key)
		}
		require.NoError(t, trie.Update(key, append([]byte("A"), key...), 1))
	}
	for j := 0; j < m; j++ {
		for {
			key := randBytes()
			if equal, _ := equalPrefixBits(trie.Spec().ph.Path(key), prefixPath, 0, prefixBits); equal {
				require.NoError(t, trie.Update(key, append([]byte("B"), key...), 1))
				break
			}
		}
	}
	require.NoError(t, trie.Commit())
	root := trie.Root()
	require.Equal(t, uint64(n+m), root.MustCount())

	// Exhaustively, every index in [0, count) selects a distinct leaf, so the
	// dense subtree is selected by exactly m of the n+m indices.
	inSubtree := 0
	for i := uint64(0); i < n+m; i++ {
		proof := requireWeightedValid(t, trie, seedFor(i))
		if proof.ClosestValueHash[0] == 'B' {
			inSubtree++
		}
	}
	require.Equal(t, m, inSubtree)

	const samples = 20000
	inSubtree = 0
	for i := 0; i < samples; i++ {
		seed := randBytes()
		proof := requireWeightedValid(t, trie, seed)
		if proof.ClosestValueHash[0] == 'B' {
			inSubtree++
		}
	}
	freq := float64(inSubtree) / samples
	t.Logf("n=%d m=%d: dense subtree selected with frequency %.4f (expected %.4f)", n, m, freq, float64(m)/float64(n+m))
	require.Greater(t, freq, 0.998)
}

func TestSMST_VerifyWeightedProof_Rejects(t *testing.T) {
	trie, paths := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	seed := seedFor(17)
	proof := requireWeightedValid(t, trie, seed)
	require.Equal(t, paths[17], proof.ClosestPath)

	t.Run("proof of another leaf", func(t *testing.T) {
		for _, i := range []int{0, 16, 18, 49} {
			other, err := trie.ProveClosest(paths[i])
			require.NoError(t, err)
			// It is a valid closest proof, but not of the selected leaf.
			valid, err := VerifyClosestProof(other, root, spec)
			require.NoError(t, err)
			require.True(t, valid)
			valid, err = VerifyWeightedProof(other, root, seed, spec)
			require.NoError(t, err)
			require.False(t, valid)
		}
	})

	t.Run("wrong seed", func(t *testing.T) {
		valid, err := VerifyWeightedProof(proof, root, seedFor(18), spec)
		require.NoError(t, err)
		require.False(t, valid)
	})

	t.Run("edited sibling count", func(t *testing.T) {
		for i := range proof.ClosestProof.SideNodes {
			tampered := copyClosestProof(proof)
			sideNode := tampered.ClosestProof.SideNodes[i]
			_, count := parseSumAndCount(sideNode)
			binary.BigEndian.PutUint64(sideNode[len(sideNode)-countSizeBytes:], count+1)
			valid, _ := VerifyWeightedProof(tampered, root, seed, spec)
			require.False(t, valid, "side node %d", i)
		}
	})

	t.Run("mismatched root count", func(t *testing.T) {
		tamperedRoot := append([]byte{}, root...)
		binary.BigEndian.PutUint64(tamperedRoot[len(tamperedRoot)-countSizeBytes:], root.MustCount()+1)
		valid, _ := VerifyWeightedProof(proof, tamperedRoot, seed, spec)
		require.False(t, valid)
	})

	t.Run("not in weighted form", func(t *testing.T) {
		flipped := copyClosestProof(proof)
		flipped.Path = append([]byte{}, proof.Path...)
		flipPathBit(flipped.Path, 0)
		flipped.FlippedBits = []int{0}
		valid, err := VerifyClosestProof(flipped, root, spec)
		require.NoError(t, err)
		require.True(t, valid)
		valid, err = VerifyWeightedProof(flipped, root, seed, spec)
		require.ErrorIs(t, err, ErrBadProof)
		require.False(t, valid)
	})

	t.Run("short seed", func(t *testing.T) {
		_, err := trie.ProveWeighted(make([]byte, 7))
		require.Error(t, err)
		_, err = VerifyWeightedProof(proof, root, make([]byte, 7), spec)
		require.Error(t, err)
	})

	t.Run("plain trie spec", func(t *testing.T) {
		plainSpec := NewTrieSpec(sha256.New(), false)
		_, err := VerifyWeightedProof(proof, root, seed, &plainSpec)
		require.Error(t, err)
	})
}

// Leaves written directly into the underlying trie can carry any count. The
// verifier accepts only leaves with a count of 1, and rejects counts that
// wrap around when summed.
func TestSMST_VerifyWeightedProof_RejectsLeafCounts(t *testing.T) {
	updateWithCount := func(trie *SMST, key []byte, count uint64) {
		valueHash := trie.valueHash(key)
		valueHash = binary.BigEndian.AppendUint64(valueHash, 1)
		valueHash = binary.BigEndian.AppendUint64(valueHash, count)
		require.NoError(t, trie.SMT.Update(key, valueHash))
	}

	t.Run("leaf count above 1", func(t *testing.T) {
		trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
		require.NoError(t, trie.Update([]byte("a"), []byte("a"), 1))
		updateWithCount(trie, []byte("b"), 5)
		require.Equal(t, uint64(6), trie.MustCount())
		for i := uint64(0); i < 6; i++ {
			seed := seedFor(i)
			proof, err := trie.ProveWeighted(seed)
			require.NoError(t, err)
			valid, err := VerifyWeightedProof(proof, trie.Root(), seed, trie.Spec())
			require.NoError(t, err)
			_, leafCount := parseSumAndCount(proof.ClosestValueHash)
			require.Equal(t, leafCount == 1, valid, "index %d", i)
		}
	})

	t.Run("counts that wrap around", func(t *testing.T) {
		trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
		require.NoError(t, trie.Update([]byte("a"), []byte("a"), 1))
		updateWithCount(trie, []byte("b"), math.MaxUint64)
		updateWithCount(trie, []byte("c"), 2)
		// 1 + (2^64 - 1) + 2 wraps around to 2.
		require.Equal(t, uint64(2), trie.MustCount())
		// The subtree holding b and c is sampled by the count it claims, and
		// no leaf in it can be proven, so at most one index verifies, for a.
		numValid := 0
		for _, key := range []string{"a", "b", "c"} {
			proof, err := trie.ProveClosest(trie.Spec().ph.Path([]byte(key)))
			require.NoError(t, err)
			for i := uint64(0); i < 2; i++ {
				valid, err := VerifyWeightedProof(proof, trie.Root(), seedFor(i), trie.Spec())
				require.NoError(t, err)
				if valid {
					require.Equal(t, "a", key, "index %d", i)
					numValid++
				}
			}
		}
		require.LessOrEqual(t, numValid, 1)
	})
}

func TestSMST_ProveWeighted_EdgeCases(t *testing.T) {
	t.Run("empty trie", func(t *testing.T) {
		trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
		_, err := trie.ProveWeighted(seedFor(0))
		require.Error(t, err)
		closest, err := trie.ProveClosest(seedFor(0))
		require.NoError(t, err)
		_, err = VerifyWeightedProof(closest, trie.Root(), seedFor(0), trie.Spec())
		require.Error(t, err)
	})

	t.Run("single leaf", func(t *testing.T) {
		trie, paths := newWeightedTestTrie(t, 1)
		for _, i := range []uint64{0, 1, math.MaxUint64} {
			proof := requireWeightedValid(t, trie, seedFor(i))
			require.Equal(t, paths[0], proof.ClosestPath)
			require.Empty(t, proof.ClosestProof.SideNodes)
		}
	})

	t.Run("last index selects rightmost leaf", func(t *testing.T) {
		trie, paths := newWeightedTestTrie(t, 77)
		proof := requireWeightedValid(t, trie, seedFor(trie.MustCount()-1))
		require.Equal(t, paths[len(paths)-1], proof.ClosestPath)
	})
}

func TestSMST_WeightedProof_Compact(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 40)
	root, spec := trie.Root(), trie.Spec()
	for i := uint64(0); i < 40; i++ {
		seed := seedFor(i)
		proof := requireWeightedValid(t, trie, seed)

		compact, err := CompactClosestProof(proof, spec)
		require.NoError(t, err)
		bz, err := compact.Marshal()
		require.NoError(t, err)
		unmarshaled := new(SparseCompactMerkleClosestProof)
		require.NoError(t, unmarshaled.Unmarshal(bz))

		valid, err := VerifyCompactWeightedProof(unmarshaled, root, seed, spec)
		require.NoError(t, err)
		require.True(t, valid)
		valid, err = VerifyCompactWeightedProof(unmarshaled, root, seedFor(i+1), spec)
		require.NoError(t, err)
		require.False(t, valid)

		decompacted, err := DecompactClosestProof(unmarshaled, spec)
		require.NoError(t, err)
		valid, err = VerifyWeightedProof(decompacted, root, seed, spec)
		require.NoError(t, err)
		require.True(t, valid)
	}
}

func copyClosestProof(proof *SparseMerkleClosestProof) *SparseMerkleClosestProof {
	sideNodes := make([][]byte, len(proof.ClosestProof.SideNodes))
	for i, sideNode := range proof.ClosestProof.SideNodes {
		sideNodes[i] = append([]byte{}, sideNode...)
	}
	return &SparseMerkleClosestProof{
		Path:             proof.Path,
		FlippedBits:      proof.FlippedBits,
		Depth:            proof.Depth,
		ClosestPath:      proof.ClosestPath,
		ClosestValueHash: proof.ClosestValueHash,
		ClosestProof: &SparseMerkleProof{
			SideNodes:             sideNodes,
			NonMembershipLeafData: proof.ClosestProof.NonMembershipLeafData,
			SiblingData:           proof.ClosestProof.SiblingData,
		},
	}
}

// ProveWeighted returns exactly the proof ProveClosest returns for the
// selected leaf's path.
func TestSMST_ProveWeighted_MatchesProveClosest(t *testing.T) {
	uncommitted, _ := newWeightedTestTrie(t, 500)
	committed, _ := newWeightedTestTrie(t, 500)
	require.NoError(t, committed.Commit())
	imported := ImportSparseMerkleSumTrie(committed.nodes, sha256.New(), committed.Root(), WithValueHasher(nil))
	single, _ := newWeightedTestTrie(t, 1)

	for name, trie := range map[string]*SMST{
		"uncommitted": uncommitted, "committed": committed, "imported": imported, "single leaf": single,
	} {
		for i := uint64(0); i < trie.MustCount(); i++ {
			got, err := trie.ProveWeighted(seedFor(i))
			require.NoError(t, err)
			want, err := trie.ProveClosest(got.ClosestPath)
			require.NoError(t, err)
			require.Equal(t, want, got, "%s, index %d", name, i)
		}
	}
}
