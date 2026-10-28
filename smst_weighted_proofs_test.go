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

// Side nodes must be exactly hashSize() bytes for every proof form, including
// proofs without SiblingData.
func TestSMST_VerifyWeightedProof_SideNodeSize(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	seed := seedFor(17)
	proof := requireWeightedValid(t, trie, seed)
	require.NotEmpty(t, proof.ClosestProof.SideNodes)

	tampered := copyClosestProof(proof)
	tampered.ClosestProof.SiblingData = nil
	i := len(tampered.ClosestProof.SideNodes) - 1
	extra := make([]byte, sumSizeBytes+countSizeBytes)
	tampered.ClosestProof.SideNodes[i] = append(tampered.ClosestProof.SideNodes[i], extra...)

	valid, err := VerifyWeightedProof(tampered, root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)
}

// compactRoundTrip marshals and unmarshals a compact closest proof, as a
// verifier receives it over the wire.
func compactRoundTrip(t *testing.T, compact *SparseCompactMerkleClosestProof) *SparseCompactMerkleClosestProof {
	t.Helper()
	bz, err := compact.Marshal()
	require.NoError(t, err)
	out := new(SparseCompactMerkleClosestProof)
	require.NoError(t, out.Unmarshal(bz))
	return out
}

// tail builds a side node sum-and-count suffix.
func tail(sum, count uint64) []byte {
	bz := make([]byte, sumSizeBytes+countSizeBytes)
	binary.BigEndian.PutUint64(bz[:sumSizeBytes], sum)
	binary.BigEndian.PutUint64(bz[sumSizeBytes:], count)
	return bz
}

// The side node size check applies to proofs that reach the verifier through
// the compact wire form too. A hand-built compact proof with no SiblingData
// and one side node of hashSize()+16 bytes (its trailing sum and count
// repeated, so only the size differs from a well-formed node) must be rejected
// once decompacted and verified.
func TestSMST_VerifyWeightedProof_SideNodeSize_Compact(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	seed := seedFor(17)
	honest := requireWeightedValid(t, trie, seed)

	compact, err := CompactClosestProof(honest, spec)
	require.NoError(t, err)
	require.NotEmpty(t, compact.ClosestProof.SideNodes)
	compact.ClosestProof.SiblingData = nil
	k := len(compact.ClosestProof.SideNodes) - 1
	node := compact.ClosestProof.SideNodes[k]
	compact.ClosestProof.SideNodes[k] = append(append([]byte{}, node...), node[len(node)-sumSizeBytes-countSizeBytes:]...)

	wire := compactRoundTrip(t, compact)

	// Decompaction keeps the node as supplied; the size is checked at verification.
	decompacted, err := DecompactClosestProof(wire, spec)
	require.NoError(t, err)
	foundWrongSize := false
	for _, sn := range decompacted.ClosestProof.SideNodes {
		if len(sn) != spec.hashSize() {
			foundWrongSize = true
		}
	}
	require.True(t, foundWrongSize, "decompacted proof keeps the out-of-size node")

	valid, err := VerifyWeightedProof(decompacted, root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)

	valid, err = VerifyCompactWeightedProof(wire, root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)
}

// A side node whose extra tail bytes change the weighted descent must not
// verify after the compact round trip.
func TestSMST_VerifyWeightedProof_CompactRoundTrip_SideNodeTail(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	rootCount := MerkleSumRoot(root).count()
	const seedIdx = uint64(3)
	seed := seedFor(seedIdx)
	selected, err := trie.ProveWeighted(seed)
	require.NoError(t, err)

	withTailCounts := func(x uint64) (*SparseCompactMerkleClosestProof, bool) {
		proofX, err := trie.ProveWeighted(seedFor(x))
		require.NoError(t, err)
		if string(proofX.ClosestPath) == string(selected.ClosestPath) {
			return nil, false
		}
		compact, err := CompactClosestProof(proofX, spec)
		require.NoError(t, err)
		n := compact.ClosestProof.NumSideNodes
		if n == 0 {
			return nil, false
		}
		isPlaceholder := func(i int) bool { return getPathBit(compact.ClosestProof.BitMask, i) == 1 }
		// tailCounts[i] is the count to append to decompacted side node i.
		tailCounts := make([]uint64, n)
		count, tt := rootCount, seedIdx%rootCount
		for depth := 0; depth < n; depth++ {
			i := n - 1 - depth
			var s uint64
			if !isPlaceholder(i) {
				if getPathBit(proofX.ClosestPath, depth) == leftChildBit {
					s = count - (tt + 1)
				} else {
					s = tt
				}
			}
			if getPathBit(proofX.ClosestPath, depth) == leftChildBit {
				if tt >= count-s {
					return nil, false
				}
			} else {
				if tt < s {
					return nil, false
				}
				tt -= s
			}
			tailCounts[i] = s
			count -= s
		}
		if count != 1 || tt != 0 {
			return nil, false
		}
		pos := 0
		for i := 0; i < n; i++ {
			if isPlaceholder(i) {
				continue
			}
			node := compact.ClosestProof.SideNodes[pos]
			compact.ClosestProof.SideNodes[pos] = append(append([]byte{}, node...), tail(0, tailCounts[i])...)
			pos++
		}
		compact.ClosestProof.SiblingData = nil
		return compact, true
	}

	var tampered *SparseCompactMerkleClosestProof
	for x := uint64(0); x < rootCount && tampered == nil; x++ {
		if c, ok := withTailCounts(x); ok {
			tampered = c
		}
	}
	require.NotNil(t, tampered, "no candidate leaf found")

	wire := compactRoundTrip(t, tampered)
	valid, err := VerifyCompactWeightedProof(wire, root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)
}

// A mutated weighted proof may only verify if all its side nodes have the
// canonical size and it selects the same leaf as the honest proof for that seed,
// both directly and through the compact encoding.
func FuzzSMST_VerifyWeightedProof_Mutations(f *testing.F) {
	trie := NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), sha256.New(), WithValueHasher(nil))
	for i := 0; i < 50; i++ {
		key := []byte{byte(i >> 8), byte(i)}
		require.NoError(f, trie.Update(key, key, uint64(i+1)))
	}
	root, spec, count := trie.Root(), trie.Spec(), trie.MustCount()

	// mode: bit 0 clears SiblingData; bits 1-2 pick the side node edit.
	seedProof, err := trie.ProveWeighted(seedFor(17))
	require.NoError(f, err)
	lastIdx := len(seedProof.ClosestProof.SideNodes) - 1
	last := seedProof.ClosestProof.SideNodes[lastIdx]
	f.Add(uint16(17), uint8(lastIdx), append([]byte{}, last[len(last)-sumSizeBytes-countSizeBytes:]...), uint8(1))
	f.Add(uint16(3), uint8(0), []byte{0xff}, uint8(2))
	f.Add(uint16(40), uint8(1), []byte{0x01, 0x02}, uint8(5))
	f.Add(uint16(9), uint8(2), []byte{}, uint8(7))

	f.Fuzz(func(t *testing.T, seedIdx uint16, nodeIdx uint8, data []byte, mode uint8) {
		seed := seedFor(uint64(seedIdx) % count)
		honest, err := trie.ProveWeighted(seed)
		require.NoError(t, err)
		if len(honest.ClosestProof.SideNodes) == 0 {
			return
		}

		mutated := copyClosestProof(honest)
		if mode&1 == 1 {
			mutated.ClosestProof.SiblingData = nil
		}
		nodes := mutated.ClosestProof.SideNodes
		i := int(nodeIdx) % len(nodes)
		switch (mode >> 1) & 3 {
		case 0: // append
			nodes[i] = append(nodes[i], data...)
		case 1: // overwrite the tail
			n := min(len(data), len(nodes[i]))
			copy(nodes[i][len(nodes[i])-n:], data[:n])
		case 2: // truncate
			nodes[i] = nodes[i][:len(nodes[i])-len(data)%(len(nodes[i])+1)]
		case 3: // replace
			nodes[i] = append([]byte{}, data...)
		}

		requireHonestIfValid := func(proof *SparseMerkleClosestProof, valid bool, err error) {
			if err != nil || !valid {
				return
			}
			for _, sideNode := range proof.ClosestProof.SideNodes {
				require.Len(t, sideNode, spec.hashSize())
			}
			require.Equal(t, honest.ClosestPath, proof.ClosestPath)
			require.Equal(t, honest.ClosestValueHash, proof.ClosestValueHash)
		}

		valid, err := VerifyWeightedProof(mutated, root, seed, spec)
		requireHonestIfValid(mutated, valid, err)

		// Apply the same edit to the compacted proof's side nodes, after
		// compaction rather than before: compacting the mutated proof would
		// reject an out-of-size node before it is ever encoded, so the edit is
		// made here to carry it through the wire form to the verifier.
		compact, err := CompactClosestProof(honest, spec)
		require.NoError(t, err)
		if mode&1 == 1 {
			compact.ClosestProof.SiblingData = nil
		}
		cnodes := compact.ClosestProof.SideNodes
		if len(cnodes) == 0 {
			return
		}
		j := int(nodeIdx) % len(cnodes)
		switch (mode >> 1) & 3 {
		case 0: // append
			cnodes[j] = append(cnodes[j], data...)
		case 1: // overwrite the tail
			n := min(len(data), len(cnodes[j]))
			copy(cnodes[j][len(cnodes[j])-n:], data[:n])
		case 2: // truncate
			cnodes[j] = cnodes[j][:len(cnodes[j])-len(data)%(len(cnodes[j])+1)]
		case 3: // replace
			cnodes[j] = append([]byte{}, data...)
		}

		bz, err := compact.Marshal()
		require.NoError(t, err)
		unmarshaled := new(SparseCompactMerkleClosestProof)
		require.NoError(t, unmarshaled.Unmarshal(bz))
		decompacted, err := DecompactClosestProof(unmarshaled, spec)
		if err != nil {
			return
		}
		valid, err = VerifyCompactWeightedProof(unmarshaled, root, seed, spec)
		requireHonestIfValid(decompacted, valid, err)
	})
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

// Malformed proofs return errors: each shape below is rejected with
// ErrBadProof by the verifier that receives it.
func TestSMST_MalformedProofsReturnErrors(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	seed := seedFor(17)
	honest := requireWeightedValid(t, trie, seed)
	require.NotEmpty(t, honest.ClosestProof.SideNodes)
	require.NotNil(t, honest.ClosestProof.SiblingData)
	path := make([]byte, spec.ph.PathSize())

	withSiblingData := func(siblingData []byte) func() (bool, error) {
		return func() (bool, error) {
			p := copyClosestProof(honest)
			p.ClosestProof.SiblingData = siblingData
			return VerifyWeightedProof(p, root, seed, spec)
		}
	}
	// prefixed returns size bytes of node data that start with prefix.
	prefixed := func(prefix []byte, size int) []byte {
		data := make([]byte, size)
		copy(data, prefix)
		return data
	}
	extSize := prefixLen + 2 + spec.ph.PathSize() + spec.hashSize() + sumSizeBytes + countSizeBytes

	plain := NewSparseMerkleTrie(simplemap.NewSimpleMap(), sha256.New())
	for i := 0; i < 10; i++ {
		require.NoError(t, plain.Update([]byte{byte(i)}, []byte{byte(i)}))
	}
	plainProof, err := plain.Prove([]byte{3})
	require.NoError(t, err)
	require.NotEmpty(t, plainProof.SideNodes)
	require.NotNil(t, plainProof.SiblingData)

	tests := []struct {
		name   string
		verify func() (bool, error)
	}{
		{"compact proof without closest proof", func() (bool, error) {
			wire := compactRoundTrip(t, &SparseCompactMerkleClosestProof{
				Path: path, Depth: []byte{0}, ClosestPath: path, ClosestValueHash: tail(1, 1),
			})
			require.Nil(t, wire.ClosestProof)
			return VerifyCompactWeightedProof(wire, root, seed, spec)
		}},
		{"closest proof without closest proof", func() (bool, error) {
			return VerifyClosestProof(&SparseMerkleClosestProof{
				Path: path, ClosestPath: path, ClosestValueHash: tail(1, 1),
			}, root, spec)
		}},
		{"compacting a closest proof without closest proof", func() (bool, error) {
			_, err := CompactClosestProof(&SparseMerkleClosestProof{
				Path: path, ClosestPath: path, ClosestValueHash: tail(1, 1),
			}, spec)
			return false, err
		}},
		{"bit mask bits set past the number of side nodes", func() (bool, error) {
			wire := compactRoundTrip(t, &SparseCompactMerkleClosestProof{
				Path: path, Depth: []byte{9}, ClosestPath: path, ClosestValueHash: tail(1, 1),
				ClosestProof: &SparseCompactMerkleProof{
					NumSideNodes: 9,
					BitMask:      []byte{0x00, 0x7F}, // bits 9 to 15 set; only bit 8 is a side node
					SideNodes:    [][]byte{make([]byte, spec.hashSize()), make([]byte, spec.hashSize())},
				},
			})
			return VerifyCompactWeightedProof(wire, root, seed, spec)
		}},
		{"empty sibling data", withSiblingData([]byte{})},
		{"sibling data with an extension prefix only", withSiblingData(extNodePrefix)},
		{"sibling data shorter than a leaf", withSiblingData([]byte{0, 1, 2})},
		{"sibling data shorter than an inner node", withSiblingData(prefixed(innerNodePrefix, sumSizeBytes+countSizeBytes-1))},
		{"sibling data one byte shorter than an extension node", withSiblingData(prefixed(extNodePrefix, extSize-1))},
		{"sibling data of extension node size that is not the sibling", withSiblingData(prefixed(extNodePrefix, extSize))},
		{"sibling data of leaf size that is not the sibling", withSiblingData(
			prefixed(leafNodePrefix, prefixLen+spec.ph.PathSize()+sumSizeBytes+countSizeBytes))},
		{"plain trie sibling data with an extension prefix only", func() (bool, error) {
			p := *plainProof
			p.SiblingData = extNodePrefix
			return VerifyProof(&p, plain.Root(), []byte{3}, []byte{3}, plain.Spec())
		}},
		{"closest value hash shorter than the sum and count", func() (bool, error) {
			p := copyClosestProof(honest)
			p.ClosestValueHash = []byte{1, 2, 3}
			return VerifyWeightedProof(p, root, seed, spec)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var valid bool
			var err error
			require.NotPanics(t, func() { valid, err = test.verify() })
			require.ErrorIs(t, err, ErrBadProof)
			require.False(t, valid)
		})
	}

	t.Run("value hash of a closest value hash shorter than the sum and count", func(t *testing.T) {
		p := copyClosestProof(honest)
		p.ClosestValueHash = []byte{1, 2, 3}
		require.NotPanics(t, func() { require.Nil(t, p.GetValueHash(spec)) })
	})
}

// A weighted proof proves a leaf, not its absence: a proof carrying
// non-membership leaf data is rejected even when it is a valid closest proof.
func TestSMST_VerifyWeightedProof_RejectsNonMembershipLeafData(t *testing.T) {
	trie, _ := newWeightedTestTrie(t, 50)
	root, spec := trie.Root(), trie.Spec()
	seed := seedFor(17)
	honest := requireWeightedValid(t, trie, seed)
	require.Less(t, len(honest.ClosestProof.SideNodes), 255)

	// A path that shares the honest leaf's descent but is not in the trie, with
	// an empty value and the honest leaf as the non-membership leaf.
	otherPath := append([]byte{}, honest.ClosestPath...)
	flipPathBit(otherPath, 255)
	proof := copyClosestProof(honest)
	proof.Path, proof.ClosestPath = otherPath, otherPath
	proof.ClosestValueHash = tail(0, 1)
	proof.ClosestProof.NonMembershipLeafData = encodeLeafNode(honest.ClosestPath, honest.ClosestValueHash)

	valid, err := VerifyClosestProof(proof, root, spec)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = VerifyWeightedProof(proof, root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)

	compact, err := CompactClosestProof(proof, spec)
	require.NoError(t, err)
	valid, err = VerifyCompactWeightedProof(compactRoundTrip(t, compact), root, seed, spec)
	require.ErrorIs(t, err, ErrBadProof)
	require.False(t, valid)
}
