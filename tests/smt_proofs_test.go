package tests

import (
	"crypto/sha256"
	"crypto/sha512"
	"strconv"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/badger"
	"github.com/stretchr/testify/require"
)

// Test base case Merkle proof operations.
func TestSMT_Proof_Operations(t *testing.T) {
	var smn, smv smt.KVStore
	var smtWithStorage *SMTWithStorage
	var proof *smt.SparseMerkleProof
	var result bool
	var root []byte
	var err error

	smn, err = badger.NewKVStore("")
	require.NoError(t, err)
	smv, err = badger.NewKVStore("")
	require.NoError(t, err)
	smtWithStorage = NewSMTWithStorage(smn, smv, sha256.New())
	base := smtWithStorage.Spec()

	// Generate and verify a proof on an empty key.
	proof, err = smtWithStorage.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = smt.VerifyProof(proof, base.TreeHasher().Placeholder(), []byte("testKey3"), smt.DefaultValue, base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = smt.VerifyProof(proof, root, []byte("testKey3"), []byte("badValue"), base)
	require.NoError(t, err)
	require.False(t, result)

	// Add a key, generate and verify a Merkle proof.
	err = smtWithStorage.Update([]byte("testKey"), []byte("testValue"))
	require.NoError(t, err)
	root = smtWithStorage.Root()
	proof, err = smtWithStorage.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = smt.VerifyProof(proof, root, []byte("testKey"), []byte("testValue"), base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = smt.VerifyProof(proof, root, []byte("testKey"), []byte("badValue"), base)
	require.NoError(t, err)
	require.False(t, result)

	// Add a key, generate and verify both Merkle proofs.
	err = smtWithStorage.Update([]byte("testKey2"), []byte("testValue"))
	require.NoError(t, err)
	root = smtWithStorage.Root()
	proof, err = smtWithStorage.Prove([]byte("testKey"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = smt.VerifyProof(proof, root, []byte("testKey"), []byte("testValue"), base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = smt.VerifyProof(proof, root, []byte("testKey"), []byte("badValue"), base)
	require.NoError(t, err)
	require.False(t, result)
	result, err = smt.VerifyProof(randomiseProof(proof), root, []byte("testKey"), []byte("testValue"), base)
	require.NoError(t, err)
	require.False(t, result)

	proof, err = smtWithStorage.Prove([]byte("testKey2"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = smt.VerifyProof(proof, root, []byte("testKey2"), []byte("testValue"), base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = smt.VerifyProof(proof, root, []byte("testKey2"), []byte("badValue"), base)
	require.NoError(t, err)
	require.False(t, result)
	result, err = smt.VerifyProof(randomiseProof(proof), root, []byte("testKey2"), []byte("testValue"), base)
	require.NoError(t, err)
	require.False(t, result)

	// Try proving a default value for a non-default leaf.
	_, leafData := base.TreeHasher().DigestLeaf(base.PathHasher().Path([]byte("testKey2")), base.DigestValue([]byte("testValue")))
	proof = &smt.SparseMerkleProof{
		SideNodes:             proof.SideNodes,
		NonMembershipLeafData: leafData,
	}
	result, err = smt.VerifyProof(proof, root, []byte("testKey2"), smt.DefaultValue, base)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)

	// Generate and verify a proof on an empty key.
	proof, err = smtWithStorage.Prove([]byte("testKey3"))
	require.NoError(t, err)
	checkCompactEquivalence(t, proof, base)
	result, err = smt.VerifyProof(proof, root, []byte("testKey3"), smt.DefaultValue, base)
	require.NoError(t, err)
	require.True(t, result)
	result, err = smt.VerifyProof(proof, root, []byte("testKey3"), []byte("badValue"), base)
	require.NoError(t, err)
	require.False(t, result)
	result, err = smt.VerifyProof(randomiseProof(proof), root, []byte("testKey3"), smt.DefaultValue, base)
	require.NoError(t, err)
	require.False(t, result)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

// Test sanity check cases for non-compact proofs.
func TestSMT_Proof_ValidateBasic(t *testing.T) {
	smn, err := badger.NewKVStore("")
	require.NoError(t, err)
	smv, err := badger.NewKVStore("")
	require.NoError(t, err)
	smtWithStorage := NewSMTWithStorage(smn, smv, sha256.New())
	base := smtWithStorage.Spec()

	err = smtWithStorage.Update([]byte("testKey1"), []byte("testValue1"))
	require.NoError(t, err)
	err = smtWithStorage.Update([]byte("testKey2"), []byte("testValue2"))
	require.NoError(t, err)
	err = smtWithStorage.Update([]byte("testKey3"), []byte("testValue3"))
	require.NoError(t, err)
	err = smtWithStorage.Update([]byte("testKey4"), []byte("testValue4"))
	require.NoError(t, err)
	root := smtWithStorage.Root()

	// Case: invalid number of sidenodes.
	proof, _ := smtWithStorage.Prove([]byte("testKey1"))
	sideNodes := make([][]byte, smtWithStorage.Spec().Depth()+1)
	for i := range sideNodes {
		sideNodes[i] = proof.SideNodes[0]
	}
	proof.SideNodes = sideNodes
	require.EqualError(t, proof.ValidateBasic(base), "too many side nodes: got 257 but max is 256")
	result, err := smt.VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect size for NonMembershipLeafData.
	proof, _ = smtWithStorage.Prove([]byte("testKey1"))
	proof.NonMembershipLeafData = make([]byte, 1)
	require.EqualError(t, proof.ValidateBasic(base), "invalid non-membership leaf data size: got 1 but min is 33")
	result, err = smt.VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactProof(proof, base)
	require.Error(t, err)

	// Case: unexpected sidenode size.
	proof, _ = smtWithStorage.Prove([]byte("testKey1"))
	proof.SideNodes[0] = make([]byte, 1)
	require.EqualError(t, proof.ValidateBasic(base), "invalid side node size: got 1 but want 32")
	result, err = smt.VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactProof(proof, base)
	require.Error(t, err)

	// Case: incorrect non-nil sibling data
	proof, _ = smtWithStorage.Prove([]byte("testKey1"))
	proof.SiblingData = base.TreeHasher().Digest(proof.SiblingData)
	require.EqualError(t, proof.ValidateBasic(base), "invalid sibling data hash: got 187864587bac133246face60f98b8214407aa314f37dfc9ce8e1f5c80284a866 but want 101cb41e8679c5376da9fb4c1e5ad4772876affb74045574cc7c12e4c38975f9")

	result, err = smt.VerifyProof(proof, root, []byte("testKey1"), []byte("testValue1"), base)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactProof(proof, base)
	require.Error(t, err)

	require.NoError(t, smn.Stop())
	require.NoError(t, smv.Stop())
}

func TestSMT_ClosestProof_ValidateBasic(t *testing.T) {
	smn, err := badger.NewKVStore("")
	require.NoError(t, err)
	s := smt.NewSparseMerkleTree(smn, sha256.New())
	np := smt.NoPrehashSpec(sha256.New(), false)
	base := s.Spec()
	path := sha256.Sum256([]byte("testKey2"))
	smt.FlipPathBit(path[:], 3)
	smt.FlipPathBit(path[:], 6)

	// insert some unrelated values to populate the tree
	require.NoError(t, s.Update([]byte("foo"), []byte("oof")))
	require.NoError(t, s.Update([]byte("bar"), []byte("rab")))
	require.NoError(t, s.Update([]byte("baz"), []byte("zab")))
	require.NoError(t, s.Update([]byte("bin"), []byte("nib")))
	require.NoError(t, s.Update([]byte("fiz"), []byte("zif")))
	require.NoError(t, s.Update([]byte("fob"), []byte("bof")))
	require.NoError(t, s.Update([]byte("testKey"), []byte("testValue")))
	require.NoError(t, s.Update([]byte("testKey2"), []byte("testValue2")))
	require.NoError(t, s.Update([]byte("testKey3"), []byte("testValue3")))
	require.NoError(t, s.Update([]byte("testKey4"), []byte("testValue4")))
	root := s.Root()

	proof, err := s.ProveClosest(path[:])
	require.NoError(t, err)
	proof.Depth = -1
	require.EqualError(t, proof.ValidateBasic(base), "invalid depth: got -1, outside of [0, 256]")
	result, err := smt.VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactClosestProof(proof, base)
	require.Error(t, err)
	proof.Depth = 257
	require.EqualError(t, proof.ValidateBasic(base), "invalid depth: got 257, outside of [0, 256]")
	result, err = smt.VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactClosestProof(proof, base)
	require.Error(t, err)

	proof, err = s.ProveClosest(path[:])
	require.NoError(t, err)
	proof.FlippedBits[0] = -1
	require.EqualError(t, proof.ValidateBasic(base), "invalid flipped bit index 0: got -1, outside of [0, 8]")
	result, err = smt.VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactClosestProof(proof, base)
	require.Error(t, err)
	proof.FlippedBits[0] = 9
	require.EqualError(t, proof.ValidateBasic(base), "invalid flipped bit index 0: got 9, outside of [0, 8]")
	result, err = smt.VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactClosestProof(proof, base)
	require.Error(t, err)

	proof, err = s.ProveClosest(path[:])
	require.NoError(t, err)
	smt.FlipPathBit(proof.Path, 3)
	require.EqualError(t, proof.ValidateBasic(base), "invalid closest path: 8d13809f932d0296b88c1913231ab4b403f05c88363575476204fef6930f22ae (not equal at bit: 3)")
	result, err = smt.VerifyClosestProof(proof, root, np)
	require.ErrorIs(t, err, smt.ErrBadProof)
	require.False(t, result)
	_, err = smt.CompactClosestProof(proof, base)
	require.Error(t, err)
}

// ProveClosest test against a visual representation of the tree
// See: https://github.com/pokt-network/smt/assets/53987565/2c2ea530-a2e8-49d7-89c2-ca9c615b0c79
func TestSMT_ProveClosest(t *testing.T) {
	var smn smt.KVStore
	var s *smt.SMT
	var proof *smt.SparseMerkleClosestProof
	var result bool
	var root []byte
	var err error

	smn, err = badger.NewKVStore("")
	require.NoError(t, err)
	s = smt.NewSparseMerkleTree(smn, sha256.New(), smt.WithValueHasher(nil))

	// insert some unrelated values to populate the tree
	require.NoError(t, s.Update([]byte("foo"), []byte("oof")))
	require.NoError(t, s.Update([]byte("bar"), []byte("rab")))
	require.NoError(t, s.Update([]byte("baz"), []byte("zab")))
	require.NoError(t, s.Update([]byte("bin"), []byte("nib")))
	require.NoError(t, s.Update([]byte("fiz"), []byte("zif")))
	require.NoError(t, s.Update([]byte("fob"), []byte("bof")))
	require.NoError(t, s.Update([]byte("testKey"), []byte("testValue")))
	require.NoError(t, s.Update([]byte("testKey2"), []byte("testValue2")))
	require.NoError(t, s.Update([]byte("testKey3"), []byte("testValue3")))
	require.NoError(t, s.Update([]byte("testKey4"), []byte("testValue4")))

	root = s.Root()

	// `testKey2` is the child of an inner node, which is the child of an extension node.
	// The extension node has the path bounds of [3, 7]. This means any bits between
	// 3-6 can be flipped, and the resulting path would still traverse through the same
	// extension node and lead to testKey2 - the closest key. However, flipping bit 7
	// will lead to testKey4.
	path := sha256.Sum256([]byte("testKey2"))
	smt.FlipPathBit(path[:], 3)
	smt.FlipPathBit(path[:], 6)
	proof, err = s.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, s.Spec())
	require.NotEqual(t, proof, &smt.SparseMerkleClosestProof{})

	result, err = smt.VerifyClosestProof(proof, root, smt.NoPrehashSpec(sha256.New(), false))
	require.NoError(t, err)
	require.True(t, result)
	closestPath := sha256.Sum256([]byte("testKey2"))
	require.Equal(t, closestPath[:], proof.ClosestPath)
	require.Equal(t, []byte("testValue2"), proof.ClosestValueHash)

	// testKey4 is the neighbour of testKey2, by flipping the final bit of the
	// extension node we change the longest common prefix to that of testKey4
	path2 := sha256.Sum256([]byte("testKey2"))
	smt.FlipPathBit(path2[:], 3)
	smt.FlipPathBit(path2[:], 7)
	proof, err = s.ProveClosest(path2[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, s.Spec())
	require.NotEqual(t, proof, &smt.SparseMerkleClosestProof{})

	result, err = smt.VerifyClosestProof(proof, root, smt.NoPrehashSpec(sha256.New(), false))
	require.NoError(t, err)
	require.True(t, result)
	closestPath = sha256.Sum256([]byte("testKey4"))
	require.Equal(t, closestPath[:], proof.ClosestPath)
	require.Equal(t, []byte("testValue4"), proof.ClosestValueHash)

	require.NoError(t, smn.Stop())
}

func TestSMT_ProveClosest_Empty(t *testing.T) {
	var smn smt.KVStore
	var s *smt.SMT
	var proof *smt.SparseMerkleClosestProof
	var err error

	smn, err = badger.NewKVStore("")
	require.NoError(t, err)
	s = smt.NewSparseMerkleTree(smn, sha256.New(), smt.WithValueHasher(nil))

	path := sha256.Sum256([]byte("testKey2"))
	smt.FlipPathBit(path[:], 3)
	smt.FlipPathBit(path[:], 6)
	proof, err = s.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, s.Spec())
	require.Equal(t, proof, &smt.SparseMerkleClosestProof{
		Path:         path[:],
		FlippedBits:  []int{0},
		Depth:        0,
		ClosestPath:  smt.Placeholder(s.Spec()),
		ClosestProof: &smt.SparseMerkleProof{},
	})

	result, err := smt.VerifyClosestProof(proof, s.Root(), smt.NoPrehashSpec(sha256.New(), false))
	require.NoError(t, err)
	require.True(t, result)

	require.NoError(t, smn.Stop())
}

func TestSMT_ProveClosest_OneNode(t *testing.T) {
	var smn smt.KVStore
	var s *smt.SMT
	var proof *smt.SparseMerkleClosestProof
	var err error

	smn, err = badger.NewKVStore("")
	require.NoError(t, err)
	s = smt.NewSparseMerkleTree(smn, sha256.New(), smt.WithValueHasher(nil))
	require.NoError(t, s.Update([]byte("foo"), []byte("bar")))

	path := sha256.Sum256([]byte("testKey2"))
	smt.FlipPathBit(path[:], 3)
	smt.FlipPathBit(path[:], 6)

	proof, err = s.ProveClosest(path[:])
	require.NoError(t, err)
	checkClosestCompactEquivalence(t, proof, s.Spec())
	closestPath := sha256.Sum256([]byte("foo"))
	require.Equal(t, proof, &smt.SparseMerkleClosestProof{
		Path:             path[:],
		FlippedBits:      []int{},
		Depth:            0,
		ClosestPath:      closestPath[:],
		ClosestValueHash: []byte("bar"),
		ClosestProof:     &smt.SparseMerkleProof{},
	})

	result, err := smt.VerifyClosestProof(proof, s.Root(), smt.NoPrehashSpec(sha256.New(), false))
	require.NoError(t, err)
	require.True(t, result)

	require.NoError(t, smn.Stop())
}

func TestSMT_ProveClosest_Proof(t *testing.T) {
	var smn smt.KVStore
	var smt256 *smt.SMT
	var smt512 *smt.SMT
	var proof256 *smt.SparseMerkleClosestProof
	var proof512 *smt.SparseMerkleClosestProof
	var err error

	// setup tree (256+512 path hasher) and nodestore
	smn, err = badger.NewKVStore("")
	require.NoError(t, err)
	smt256 = smt.NewSparseMerkleTree(smn, sha256.New())
	smt512 = smt.NewSparseMerkleTree(smn, sha512.New())

	// insert 100000 key-value-sum triples
	for i := 0; i < 100000; i++ {
		s := strconv.Itoa(i)
		require.NoError(t, smt256.Update([]byte(s), []byte(s)))
		require.NoError(t, smt512.Update([]byte(s), []byte(s)))
		// generate proofs for each key in the tree
		path256 := sha256.Sum256([]byte(s))
		path512 := sha512.Sum512([]byte(s))
		proof256, err = smt256.ProveClosest(path256[:])
		require.NoError(t, err)
		proof512, err = smt512.ProveClosest(path512[:])
		require.NoError(t, err)
		// ensure proof is same after compression and decompression
		checkClosestCompactEquivalence(t, proof256, smt256.Spec())
		checkClosestCompactEquivalence(t, proof512, smt512.Spec())
	}

	require.NoError(t, smn.Stop())
}
