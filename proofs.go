package smt

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"math"
)

func init() {
	gob.Register(SparseMerkleProof{})
	gob.Register(SparseCompactMerkleProof{})
	gob.Register(SparseMerkleClosestProof{})
	gob.Register(SparseCompactMerkleClosestProof{})
}

// ErrBadProof is returned when an invalid Merkle proof is supplied.
var ErrBadProof = errors.New("bad proof")

// SparseMerkleProof is a Merkle proof for an element in a SparseMerkleTree.
// TODO: Research whether the SiblingData is required and remove it if not
type SparseMerkleProof struct {
	// SideNodes is an array of the sibling nodes leading up to the leaf of the proof.
	SideNodes [][]byte

	// NonMembershipLeafData is the data of the unrelated leaf at the position
	// of the key being proven, in the case of a non-membership proof. For
	// membership proofs, is nil.
	NonMembershipLeafData []byte

	// SiblingData is the data of the sibling node to the leaf being proven,
	// required for updatable proofs. For unupdatable proofs, is nil.
	SiblingData []byte
}

// Marshal serialises the SparseMerkleProof to bytes
func (proof *SparseMerkleProof) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(proof); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserialises the SparseMerkleProof from bytes
func (proof *SparseMerkleProof) Unmarshal(bz []byte) error {
	buf := bytes.NewBuffer(bz)
	dec := gob.NewDecoder(buf)
	return dec.Decode(proof)
}

func (proof *SparseMerkleProof) sanityCheck(spec *TreeSpec) bool {
	// Do a basic sanity check on the proof, so that a malicious proof cannot
	// cause the verifier to fatally exit (e.g. due to an index out-of-range
	// error) or cause a CPU DoS attack.

	// Check that the number of supplied sidenodes does not exceed the maximum possible.
	if len(proof.SideNodes) > spec.ph.PathSize()*8 ||

		// Check that leaf data for non-membership proofs is a valid size.
		(proof.NonMembershipLeafData != nil && len(proof.NonMembershipLeafData) < len(leafPrefix)+spec.ph.PathSize()) {
		return false
	}

	// Check that all supplied sidenodes are the correct size.
	for _, v := range proof.SideNodes {
		if len(v) != hashSize(spec) {
			return false
		}
	}

	// Check that the sibling data hashes to the first side node if not nil
	if proof.SiblingData == nil || len(proof.SideNodes) == 0 {
		return true
	}

	siblingHash := hashPreimage(spec, proof.SiblingData)
	return bytes.Equal(proof.SideNodes[0], siblingHash)
}

// SparseCompactMerkleProof is a compact Merkle proof for an element in a SparseMerkleTree.
type SparseCompactMerkleProof struct {
	// SideNodes is an array of the sibling nodes leading up to the leaf of the proof.
	SideNodes [][]byte

	// NonMembershipLeafData is the data of the unrelated leaf at the position
	// of the key being proven, in the case of a non-membership proof. For
	// membership proofs, is nil.
	NonMembershipLeafData []byte

	// BitMask, in the case of a compact proof, is a bit mask of the sidenodes
	// of the proof where an on-bit indicates that the sidenode at the bit's
	// index is a placeholder. This is only set if the proof is compact.
	BitMask []byte

	// NumSideNodes, in the case of a compact proof, indicates the number of
	// sidenodes in the proof when decompacted. This is only set if the proof is compact.
	NumSideNodes int

	// SiblingData is the data of the sibling node to the leaf being proven,
	// required for updatable proofs. For unupdatable proofs, is nil.
	SiblingData []byte
}

// Marshal serialises the SparseCompactMerkleProof to bytes
func (proof *SparseCompactMerkleProof) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(proof); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserialises the SparseCompactMerkleProof from bytes
func (proof *SparseCompactMerkleProof) Unmarshal(bz []byte) error {
	buf := bytes.NewBuffer(bz)
	dec := gob.NewDecoder(buf)
	return dec.Decode(proof)
}

func (proof *SparseCompactMerkleProof) sanityCheck(spec *TreeSpec) bool {
	// Do a basic sanity check on the proof on the fields of the proof specific to
	// the compact proof only.
	//
	// When the proof is de-compacted and verified, the sanity check for the
	// de-compacted proof should be executed.

	// Compact proofs: check that NumSideNodes is within the right range.
	if proof.NumSideNodes < 0 || proof.NumSideNodes > spec.ph.PathSize()*8 ||

		// Compact proofs: check that the length of the bit mask is as expected
		// according to NumSideNodes.
		len(proof.BitMask) != int(math.Ceil(float64(proof.NumSideNodes)/float64(8))) ||

		// Compact proofs: check that the correct number of sidenodes have been
		// supplied according to the bit mask.
		(proof.NumSideNodes > 0 && len(proof.SideNodes) != proof.NumSideNodes-countSetBits(proof.BitMask)) {
		return false
	}

	return true
}

// SparseMerkleClosestProof is a wrapper around a SparseMerkleProof that
// represents the proof of the leaf with the closest path to the one provided.
type SparseMerkleClosestProof struct {
	Path             []byte             // the path provided to the ProveClosest method
	FlippedBits      []int              // the index of the bits flipped in the path during tree traversal
	Depth            int                // the depth of the tree when tree traversal stopped
	ClosestPath      []byte             // the path of the leaf closest to the path provided
	ClosestValueHash []byte             // the value hash of the leaf closest to the path provided
	ClosestProof     *SparseMerkleProof // the proof of the leaf closest to the path provided
}

// Marshal serialises the SparseMerkleClosestProof to bytes
func (proof *SparseMerkleClosestProof) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(proof); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserialises the SparseMerkleClosestProof from bytes
func (proof *SparseMerkleClosestProof) Unmarshal(bz []byte) error {
	buf := bytes.NewBuffer(bz)
	dec := gob.NewDecoder(buf)
	return dec.Decode(proof)
}

func (proof *SparseMerkleClosestProof) sanityCheck(spec *TreeSpec) bool {
	if proof.Depth > spec.ph.PathSize()*8 {
		return false
	}
	for _, i := range proof.FlippedBits {
		if i > proof.Depth {
			return false
		}
		if i > spec.ph.PathSize()*8 {
			return false
		}
	}
	workingPath := proof.Path
	for _, i := range proof.FlippedBits {
		flipPathBit(workingPath, i)
	}
	if prefix := countCommonPrefix(
		workingPath[:proof.Depth/8],
		proof.ClosestPath[:proof.Depth/8],
		0,
	); prefix != proof.Depth {
		return false
	}
	if !proof.ClosestProof.sanityCheck(spec) {
		return false
	}
	return true
}

// SparseCompactMerkleClosestProof is a compressed representation of the SparseMerkleClosestProof
// NOTE: This compact proof assumes the path hasher is 256 bits
// TODO: Generalise this compact proof to support other path hasher sizes
type SparseCompactMerkleClosestProof struct {
	Path             []byte                    // the path provided to the ProveClosest method
	FlippedBits      []byte                    // the index of the bits flipped in the path during tree traversal
	Depth            byte                      // the depth of the tree when tree traversal stopped
	ClosestPath      []byte                    // the path of the leaf closest to the path provided
	ClosestValueHash []byte                    // the value hash of the leaf closest to the path provided
	ClosestProof     *SparseCompactMerkleProof // the proof of the leaf closest to the path provided
}

func (proof *SparseCompactMerkleClosestProof) sanityCheck(spec *TreeSpec) bool {
	if int(proof.Depth) > spec.ph.PathSize()*8 {
		return false
	}
	for _, i := range proof.FlippedBits {
		if i > proof.Depth {
			return false
		}
		if int(i) > spec.ph.PathSize()*8 {
			return false
		}
	}
	workingPath := proof.Path
	for _, i := range proof.FlippedBits {
		flipPathBit(workingPath, int(i))
	}
	if prefix := countCommonPrefix(
		workingPath[:proof.Depth/8],
		proof.ClosestPath[:proof.Depth/8],
		0,
	); prefix != int(proof.Depth) {
		return false
	}
	if !proof.ClosestProof.sanityCheck(spec) {
		return false
	}
	return true
}

// Marshal serialises the SparseCompactMerkleClosestProof to bytes
func (proof *SparseCompactMerkleClosestProof) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(proof); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserialises the SparseCompactMerkleClosestProof from bytes
func (proof *SparseCompactMerkleClosestProof) Unmarshal(bz []byte) error {
	buf := bytes.NewBuffer(bz)
	dec := gob.NewDecoder(buf)
	return dec.Decode(proof)
}

// VerifyProof verifies a Merkle proof.
func VerifyProof(proof *SparseMerkleProof, root, key, value []byte, spec *TreeSpec) bool {
	result, _ := verifyProofWithUpdates(proof, root, key, value, spec)
	return result
}

// VerifySumProof verifies a Merkle proof for a sum tree.
func VerifySumProof(proof *SparseMerkleProof, root, key, value []byte, sum uint64, spec *TreeSpec) bool {
	var sumBz [sumSize]byte
	binary.BigEndian.PutUint64(sumBz[:], sum)
	valueHash := spec.digestValue(value)
	valueHash = append(valueHash, sumBz[:]...)
	if bytes.Equal(value, defaultValue) && sum == 0 {
		valueHash = defaultValue
	}
	smtSpec := &TreeSpec{
		th:      spec.th,
		ph:      spec.ph,
		vh:      spec.vh,
		sumTree: spec.sumTree,
	}
	nvh := WithValueHasher(nil)
	nvh(smtSpec)
	return VerifyProof(proof, root, key, valueHash, smtSpec)
}

// VerifyClosestProof verifies a Merkle proof for a proof of a leaf found to
// have the closest path to the one provided to the proof function
func VerifyClosestProof(proof *SparseMerkleClosestProof, root []byte, spec *TreeSpec) bool {
	if !proof.sanityCheck(spec) {
		return false
	}
	if !spec.sumTree {
		return VerifyProof(proof.ClosestProof, root, proof.ClosestPath, proof.ClosestValueHash, spec)
	}
	if proof.ClosestValueHash == nil {
		return VerifySumProof(proof.ClosestProof, root, proof.ClosestPath, nil, 0, spec)
	}
	sumBz := proof.ClosestValueHash[len(proof.ClosestValueHash)-sumSize:]
	sum := binary.BigEndian.Uint64(sumBz)
	return VerifySumProof(proof.ClosestProof, root, proof.ClosestPath, proof.ClosestValueHash[:len(proof.ClosestValueHash)-sumSize], sum, spec)
}

func verifyProofWithUpdates(proof *SparseMerkleProof, root []byte, key []byte, value []byte, spec *TreeSpec) (bool, [][][]byte) {
	path := spec.ph.Path(key)

	if !proof.sanityCheck(spec) {
		return false, nil
	}

	var updates [][][]byte

	// Determine what the leaf hash should be.
	var currentHash, currentData []byte
	if bytes.Equal(value, defaultValue) { // Non-membership proof.
		if proof.NonMembershipLeafData == nil { // Leaf is a placeholder value.
			currentHash = placeholder(spec)
		} else { // Leaf is an unrelated leaf.
			var actualPath, valueHash []byte
			actualPath, valueHash = parseLeaf(proof.NonMembershipLeafData, spec.ph)
			if bytes.Equal(actualPath, path) {
				// This is not an unrelated leaf; non-membership proof failed.
				return false, nil
			}
			currentHash, currentData = digestLeaf(spec, actualPath, valueHash)

			update := make([][]byte, 2)
			update[0], update[1] = currentHash, currentData
			updates = append(updates, update)
		}
	} else { // Membership proof.
		valueHash := spec.digestValue(value)
		currentHash, currentData = digestLeaf(spec, path, valueHash)
		update := make([][]byte, 2)
		update[0], update[1] = currentHash, currentData
		updates = append(updates, update)
	}

	// Recompute root.
	for i := 0; i < len(proof.SideNodes); i++ {
		node := make([]byte, hashSize(spec))
		copy(node, proof.SideNodes[i])

		if getPathBit(path, len(proof.SideNodes)-1-i) == left {
			currentHash, currentData = digestNode(spec, currentHash, node)
		} else {
			currentHash, currentData = digestNode(spec, node, currentHash)
		}

		update := make([][]byte, 2)
		update[0], update[1] = currentHash, currentData
		updates = append(updates, update)
	}

	return bytes.Equal(currentHash, root), updates
}

// VerifyCompactProof verifies a compacted Merkle proof.
func VerifyCompactProof(proof *SparseCompactMerkleProof, root []byte, key, value []byte, spec *TreeSpec) bool {
	decompactedProof, err := DecompactProof(proof, spec)
	if err != nil {
		return false
	}
	return VerifyProof(decompactedProof, root, key, value, spec)
}

// VerifyCompactSumProof verifies a compacted Merkle proof.
func VerifyCompactSumProof(proof *SparseCompactMerkleProof, root []byte, key, value []byte, sum uint64, spec *TreeSpec) bool {
	decompactedProof, err := DecompactProof(proof, spec)
	if err != nil {
		return false
	}
	return VerifySumProof(decompactedProof, root, key, value, sum, spec)
}

// VerifyCompactClosestProof verifies a compacted Merkle proof
func VerifyCompactClosestProof(proof *SparseCompactMerkleClosestProof, root []byte, spec *TreeSpec) bool {
	decompactedProof, err := DecompactClosestProof(proof, spec)
	if err != nil {
		return false
	}
	return VerifyClosestProof(decompactedProof, root, spec)
}

// CompactProof compacts a proof, to reduce its size.
func CompactProof(proof *SparseMerkleProof, spec *TreeSpec) (*SparseCompactMerkleProof, error) {
	if !proof.sanityCheck(spec) {
		return nil, ErrBadProof
	}

	bitMask := make([]byte, int(math.Ceil(float64(len(proof.SideNodes))/float64(8))))
	var compactedSideNodes [][]byte
	for i := 0; i < len(proof.SideNodes); i++ {
		node := make([]byte, hashSize(spec))
		copy(node, proof.SideNodes[i])
		if bytes.Equal(node, placeholder(spec)) {
			setPathBit(bitMask, i)
		} else {
			compactedSideNodes = append(compactedSideNodes, node)
		}
	}

	return &SparseCompactMerkleProof{
		SideNodes:             compactedSideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
		BitMask:               bitMask,
		NumSideNodes:          len(proof.SideNodes),
		SiblingData:           proof.SiblingData,
	}, nil
}

// DecompactProof decompacts a proof, so that it can be used for VerifyProof.
func DecompactProof(proof *SparseCompactMerkleProof, spec *TreeSpec) (*SparseMerkleProof, error) {
	if !proof.sanityCheck(spec) {
		return nil, ErrBadProof
	}

	decompactedSideNodes := make([][]byte, proof.NumSideNodes)
	position := 0
	for i := 0; i < proof.NumSideNodes; i++ {
		if getPathBit(proof.BitMask, i) == 1 {
			decompactedSideNodes[i] = placeholder(spec)
		} else {
			decompactedSideNodes[i] = proof.SideNodes[position]
			position++
		}
	}

	return &SparseMerkleProof{
		SideNodes:             decompactedSideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
		SiblingData:           proof.SiblingData,
	}, nil
}

// CompactClosestProof compacts a proof, to reduce its size.
func CompactClosestProof(proof *SparseMerkleClosestProof, spec *TreeSpec) (*SparseCompactMerkleClosestProof, error) {
	if !proof.sanityCheck(spec) {
		return nil, ErrBadProof
	}
	compactedProof, err := CompactProof(proof.ClosestProof, spec)
	if err != nil {
		return nil, err
	}
	flippedBits := make([]byte, len(proof.FlippedBits))
	for i, v := range proof.FlippedBits {
		flippedBits[i] = byte(v)
	}
	return &SparseCompactMerkleClosestProof{
		Path:             proof.Path,
		FlippedBits:      flippedBits,
		Depth:            byte(proof.Depth),
		ClosestPath:      proof.ClosestPath,
		ClosestValueHash: proof.ClosestValueHash,
		ClosestProof:     compactedProof,
	}, nil
}

// DecompactClosestProof decompacts a proof, so that it can be used for VerifyClosestProof.
func DecompactClosestProof(proof *SparseCompactMerkleClosestProof, spec *TreeSpec) (*SparseMerkleClosestProof, error) {
	if !proof.sanityCheck(spec) {
		return nil, ErrBadProof
	}
	decompactedProof, err := DecompactProof(proof.ClosestProof, spec)
	if err != nil {
		return nil, err
	}
	flippedBits := make([]int, len(proof.FlippedBits))
	for i, v := range proof.FlippedBits {
		flippedBits[i] = int(v)
	}
	return &SparseMerkleClosestProof{
		Path:             proof.Path,
		FlippedBits:      flippedBits,
		Depth:            int(proof.Depth),
		ClosestPath:      proof.ClosestPath,
		ClosestValueHash: proof.ClosestValueHash,
		ClosestProof:     decompactedProof,
	}, nil
}
