package smt

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
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

func (proof *SparseMerkleProof) validateBasic(spec *TreeSpec) error {
	// Do a basic sanity check on the proof, so that a malicious proof cannot
	// cause the verifier to fatally exit (e.g. due to an index out-of-range
	// error) or cause a CPU DoS attack.

	// Check that the number of supplied sidenodes does not exceed the maximum possible.
	if len(proof.SideNodes) > spec.ph.PathSize()*8 {
		return fmt.Errorf("too many side nodes: %d", len(proof.SideNodes))
	}
	// Check that leaf data for non-membership proofs is a valid size.
	if proof.NonMembershipLeafData != nil && len(proof.NonMembershipLeafData) < len(leafPrefix)+spec.ph.PathSize() {
		return fmt.Errorf("invalid non-membership leaf data size: %d", len(proof.NonMembershipLeafData))
	}

	// Check that all supplied sidenodes are the correct size.
	for _, v := range proof.SideNodes {
		if len(v) != hashSize(spec) {
			return fmt.Errorf("invalid side node size: %d", len(v))
		}
	}

	// Check that the sibling data hashes to the first side node if not nil
	if proof.SiblingData == nil || len(proof.SideNodes) == 0 {
		return nil
	}
	siblingHash := hashPreimage(spec, proof.SiblingData)
	if eq := bytes.Equal(proof.SideNodes[0], siblingHash); !eq {
		return fmt.Errorf("invalid sibling data hash: %x", siblingHash)
	}

	return nil
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

func (proof *SparseCompactMerkleProof) validateBasic(spec *TreeSpec) error {
	// Do a basic sanity check on the proof on the fields of the proof specific to
	// the compact proof only.
	//
	// When the proof is de-compacted and verified, the sanity check for the
	// de-compacted proof should be executed.

	// Compact proofs: check that NumSideNodes is within the right range.
	if proof.NumSideNodes < 0 || proof.NumSideNodes > spec.ph.PathSize()*8 {
		return fmt.Errorf("invalid number of side nodes: %d", proof.NumSideNodes)
	}

	// Compact proofs: check that the length of the bit mask is as expected
	// according to NumSideNodes.
	if len(proof.BitMask) != int(math.Ceil(float64(proof.NumSideNodes)/float64(8))) {
		return fmt.Errorf("invalid bit mask length: %d", len(proof.BitMask))
	}

	// Compact proofs: check that the correct number of sidenodes have been
	// supplied according to the bit mask.
	if proof.NumSideNodes > 0 && len(proof.SideNodes) != proof.NumSideNodes-countSetBits(proof.BitMask) {
		return fmt.Errorf("invalid number of side nodes: %d", len(proof.SideNodes))
	}

	return nil
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

func (proof *SparseMerkleClosestProof) validateBasic(spec *TreeSpec) error {
	// ensure the depth of the leaf node being proven is within the path size
	if proof.Depth < 0 || proof.Depth > spec.ph.PathSize()*8 {
		return fmt.Errorf("invalid depth: %d", proof.Depth)
	}
	// for each of the bits flipped ensure that they are within the path size
	// and that they are not greater than the depth of the leaf node being proven
	for _, i := range proof.FlippedBits {
		// as proof.Depth <= spec.ph.PathSize()*8, i <= proof.Depth
		if i < 0 || i > proof.Depth {
			return fmt.Errorf("invalid flipped bit index: %d", i)
		}
	}
	// create the path of the leaf node using the flipped bits metadata
	workingPath := proof.Path
	for _, i := range proof.FlippedBits {
		flipPathBit(workingPath, i)
	}
	// ensure that the path of the leaf node being proven has a prefix
	// of length depth as the path provided (with bits flipped)
	if equal, failed := equalPrefixBits(
		workingPath,
		proof.ClosestPath,
		0, proof.Depth,
	); !equal {
		return fmt.Errorf("invalid closest path: %x (not equal at bit: %d)", proof.ClosestPath, failed)
	}
	// validate the proof itself
	if err := proof.ClosestProof.validateBasic(spec); err != nil {
		return fmt.Errorf("invalid closest proof: %w", err)
	}
	return nil
}

// SparseCompactMerkleClosestProof is a compressed representation of the SparseMerkleClosestProof
type SparseCompactMerkleClosestProof struct {
	Path             []byte                    // the path provided to the ProveClosest method
	FlippedBits      [][]byte                  // the index of the bits flipped in the path during tree traversal
	Depth            []byte                    // the depth of the tree when tree traversal stopped
	ClosestPath      []byte                    // the path of the leaf closest to the path provided
	ClosestValueHash []byte                    // the value hash of the leaf closest to the path provided
	ClosestProof     *SparseCompactMerkleProof // the proof of the leaf closest to the path provided
}

func (proof *SparseCompactMerkleClosestProof) validateBasic(spec *TreeSpec) error {
	// Do a basic sanity check on the proof on the fields of the proof specific to
	// the compact proof only.
	//
	// When the proof is de-compacted and verified, the sanity check for the
	// de-compacted proof should be executed.

	// ensure no compressed fields are larger than the path size
	maxSliceLen := minBytes(spec.ph.PathSize() * 8)
	if len(proof.Depth) > maxSliceLen {
		return fmt.Errorf("invalid depth: %d", proof.Depth)
	}
	for _, i := range proof.FlippedBits {
		if len(i) > maxSliceLen {
			return fmt.Errorf("invalid flipped bit index: %d", bytesToInt(i))
		}
	}
	// perform a sanity check on the closest proof
	if err := proof.ClosestProof.validateBasic(spec); err != nil {
		return fmt.Errorf("invalid closest proof: %w", err)
	}
	return nil
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
func VerifyProof(proof *SparseMerkleProof, root, key, value []byte, spec *TreeSpec) (bool, error) {
	result, _, err := verifyProofWithUpdates(proof, root, key, value, spec)
	return result, err
}

// VerifySumProof verifies a Merkle proof for a sum tree.
func VerifySumProof(proof *SparseMerkleProof, root, key, value []byte, sum uint64, spec *TreeSpec) (bool, error) {
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

// VerifyClosestProof verifies a Merkle proof for a proof of inclusion for a leaf
// found to have the closest path to the one provided to the proof structure
func VerifyClosestProof(proof *SparseMerkleClosestProof, root []byte, spec *TreeSpec) (bool, error) {
	if err := proof.validateBasic(spec); err != nil {
		return false, errors.Join(ErrBadProof, err)
	}
	if !spec.sumTree {
		return VerifyProof(proof.ClosestProof, root, proof.ClosestPath, proof.ClosestValueHash, spec)
	}
	if proof.ClosestValueHash == nil {
		return VerifySumProof(proof.ClosestProof, root, proof.ClosestPath, nil, 0, spec)
	}
	sumBz := proof.ClosestValueHash[len(proof.ClosestValueHash)-sumSize:]
	sum := binary.BigEndian.Uint64(sumBz)
	valueHash := proof.ClosestValueHash[:len(proof.ClosestValueHash)-sumSize]
	return VerifySumProof(proof.ClosestProof, root, proof.ClosestPath, valueHash, sum, spec)
}

func verifyProofWithUpdates(proof *SparseMerkleProof, root []byte, key []byte, value []byte, spec *TreeSpec) (bool, [][][]byte, error) {
	path := spec.ph.Path(key)

	if err := proof.validateBasic(spec); err != nil {
		return false, nil, errors.Join(ErrBadProof, err)
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
				return false, nil, errors.Join(ErrBadProof, errors.New("non-membership proof on related leaf"))
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

	return bytes.Equal(currentHash, root), updates, nil
}

// VerifyCompactProof is similar to VerifyProof but for a compacted Merkle proof.
func VerifyCompactProof(proof *SparseCompactMerkleProof, root []byte, key, value []byte, spec *TreeSpec) (bool, error) {
	decompactedProof, err := DecompactProof(proof, spec)
	if err != nil {
		return false, errors.Join(ErrBadProof, err)
	}
	return VerifyProof(decompactedProof, root, key, value, spec)
}

// VerifyCompactSumProof is similar to VerifySumProof but for a compacted Merkle proof.
func VerifyCompactSumProof(proof *SparseCompactMerkleProof, root []byte, key, value []byte, sum uint64, spec *TreeSpec) (bool, error) {
	decompactedProof, err := DecompactProof(proof, spec)
	if err != nil {
		return false, errors.Join(ErrBadProof, err)
	}
	return VerifySumProof(decompactedProof, root, key, value, sum, spec)
}

// VerifyCompactClosestProof is similar to VerifyClosestProof but for a compacted merkle proof
func VerifyCompactClosestProof(proof *SparseCompactMerkleClosestProof, root []byte, spec *TreeSpec) (bool, error) {
	decompactedProof, err := DecompactClosestProof(proof, spec)
	if err != nil {
		return false, errors.Join(ErrBadProof, err)
	}
	return VerifyClosestProof(decompactedProof, root, spec)
}

// CompactProof compacts a proof, to reduce its size.
func CompactProof(proof *SparseMerkleProof, spec *TreeSpec) (*SparseCompactMerkleProof, error) {
	if err := proof.validateBasic(spec); err != nil {
		return nil, errors.Join(ErrBadProof, err)
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
	if err := proof.validateBasic(spec); err != nil {
		return nil, errors.Join(ErrBadProof, err)
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

	if len(decompactedSideNodes) == 0 {
		decompactedSideNodes = nil
	}
	return &SparseMerkleProof{
		SideNodes:             decompactedSideNodes,
		NonMembershipLeafData: proof.NonMembershipLeafData,
		SiblingData:           proof.SiblingData,
	}, nil
}

// CompactClosestProof compacts a proof, to reduce its size.
func CompactClosestProof(proof *SparseMerkleClosestProof, spec *TreeSpec) (*SparseCompactMerkleClosestProof, error) {
	if err := proof.validateBasic(spec); err != nil {
		return nil, errors.Join(ErrBadProof, err)
	}
	compactedProof, err := CompactProof(proof.ClosestProof, spec)
	if err != nil {
		return nil, err
	}
	flippedBits := make([][]byte, len(proof.FlippedBits))
	for i, v := range proof.FlippedBits {
		flippedBits[i] = intToBytes(v)
	}
	return &SparseCompactMerkleClosestProof{
		Path:             proof.Path,
		FlippedBits:      flippedBits,
		Depth:            intToBytes(proof.Depth),
		ClosestPath:      proof.ClosestPath,
		ClosestValueHash: proof.ClosestValueHash,
		ClosestProof:     compactedProof,
	}, nil
}

// DecompactClosestProof decompacts a proof, so that it can be used for VerifyClosestProof.
func DecompactClosestProof(proof *SparseCompactMerkleClosestProof, spec *TreeSpec) (*SparseMerkleClosestProof, error) {
	if err := proof.validateBasic(spec); err != nil {
		return nil, errors.Join(ErrBadProof, err)
	}
	decompactedProof, err := DecompactProof(proof.ClosestProof, spec)
	if err != nil {
		return nil, err
	}
	flippedBits := make([]int, len(proof.FlippedBits))
	for i, v := range proof.FlippedBits {
		flippedBits[i] = bytesToInt(v)
	}
	return &SparseMerkleClosestProof{
		Path:             proof.Path,
		FlippedBits:      flippedBits,
		Depth:            bytesToInt(proof.Depth),
		ClosestPath:      proof.ClosestPath,
		ClosestValueHash: proof.ClosestValueHash,
		ClosestProof:     decompactedProof,
	}, nil
}
