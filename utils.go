package smt

type nilPathHasher struct {
	hashSize int
}

func (n *nilPathHasher) Path(key []byte) []byte { return key[:n.hashSize] }
func (n *nilPathHasher) PathSize() int          { return n.hashSize }

func newNilPathHasher(hashSize int) PathHasher {
	return &nilPathHasher{hashSize: hashSize}
}

// getPathBit gets the bit at an offset (see position) in the data
// provided relative to the most significant bit
func getPathBit(data []byte, position int) int {
	// get the byte at the position and then left shift one by the offset of the position
	// from the leftmost bit in the byte. Check if the bitwise AND is the same
	// Path: []byte{ {0 1 0 1 1 0 1 0}, {0 1 1 0 1 1 0 1}, {1 0 0 1 0 0 1 0} } (length = 24 bits / 3 bytes)
	// Position: 13 - 13/8=1
	// Path[1] = {0 1 1 0 1 1 0 1}
	// uint(13)%8 = 5, 8-1-5=2
	// 00000001 << 2 = 00000100
	//   {0 1 1 0 1 1 0 1}
	// & {0 0 0 0 0 1 0 0}
	// = {0 0 0 0 0 1 0 0}
	// > 0 so Path is on the right at position 13
	if int(data[position/8])&(1<<(8-1-uint(position)%8)) > 0 {
		return 1
	}
	return 0
}

// setPathBit sets the bit at an offset (see position) in the data
// provided relative to the most significant bit
func setPathBit(data []byte, position int) {
	n := int(data[position/8])
	n |= 1 << (8 - 1 - uint(position)%8)
	data[position/8] = byte(n)
}

// flipPathBit flips the bit at an offset (see position) in the data
// provided relative to most significant bit
func flipPathBit(data []byte, position int) {
	n := int(data[position/8])           // get index of byte containing the position
	n ^= 1 << (8 - 1 - uint(position)%8) // XOR the bit within the byte at the position
	data[position/8] = byte(n)
}

func countSetBits(data []byte) int {
	count := 0
	for i := 0; i < len(data)*8; i++ {
		if getPathBit(data, i) == 1 {
			count++
		}
	}
	return count
}

// counts common bits in each path, starting from some position
func countCommonPrefix(data1, data2 []byte, from int) int {
	count := 0
	for i := from; i < len(data1)*8; i++ {
		if getPathBit(data1, i) == getPathBit(data2, i) {
			count++
		} else {
			break
		}
	}
	return count + from
}

// placeholder returns the default placeholder value depending on the tree type
func placeholder(spec *TreeSpec) []byte {
	if spec.sumTree {
		placeholder := spec.th.placeholder()
		placeholder = append(placeholder, defaultSum[:]...)
		return placeholder
	}
	return spec.th.placeholder()
}

// hashSize returns the hash size depending on the tree type
func hashSize(spec *TreeSpec) int {
	if spec.sumTree {
		return spec.th.hashSize() + sumSize
	}
	return spec.th.hashSize()
}

// digestLeaf returns the hash and preimage of a leaf node depending on the tree type
func digestLeaf(spec *TreeSpec, path, value []byte) ([]byte, []byte) {
	if spec.sumTree {
		return spec.th.digestSumLeaf(path, value)
	}
	return spec.th.digestLeaf(path, value)
}

// digestNode returns the hash and preimage of a node depending on the tree type
func digestNode(spec *TreeSpec, left, right []byte) ([]byte, []byte) {
	if spec.sumTree {
		return spec.th.digestSumNode(left, right)
	}
	return spec.th.digestNode(left, right)
}

// hashNode hashes a node depending on the tree type
func hashNode(spec *TreeSpec, node treeNode) []byte {
	if spec.sumTree {
		return spec.hashSumNode(node)
	}
	return spec.hashNode(node)
}

// serialize serializes a node depending on the tree type
func serialize(spec *TreeSpec, node treeNode) []byte {
	if spec.sumTree {
		return spec.sumSerialize(node)
	}
	return spec.serialize(node)
}

// hashPreimage hashes the serialised data provided depending on the tree type
func hashPreimage(spec *TreeSpec, data []byte) []byte {
	if spec.sumTree {
		return hashSumSerialization(spec, data)
	}
	return hashSerialization(spec, data)
}

// Used for verification of serialized proof data
func hashSerialization(smt *TreeSpec, data []byte) []byte {
	if isExtension(data) {
		pathBounds, path, childHash := parseExtension(data, smt.ph)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return smt.hashNode(&ext)
	} else {
		return smt.th.digest(data)
	}
}

// Used for verification of serialized proof data for sum tree nodes
func hashSumSerialization(smt *TreeSpec, data []byte) []byte {
	if isExtension(data) {
		pathBounds, path, childHash, _ := parseSumExtension(data, smt.ph)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return smt.hashSumNode(&ext)
	} else {
		digest := smt.th.digest(data)
		digest = append(digest, data[len(data)-sumSize:]...)
		return digest
	}
}

// resolve resolves a lazy node depending on the tree type
func resolve(smt *SMT, hash []byte, resolver func([]byte) (treeNode, error),
) (treeNode, error) {
	if smt.sumTree {
		return smt.resolveSum(hash, resolver)
	}
	return smt.resolve(hash, resolver)
}
