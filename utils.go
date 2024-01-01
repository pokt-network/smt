package smt

import (
	"encoding/binary"
)

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

// countSetBits counts the number of bits set in the data provided (ie the number of 1s)
func countSetBits(data []byte) int {
	count := 0
	for _, b := range data {
		// Kernighan’s Method of counting set bits in a byte
		for b != 0 {
			b = b & (b - 1) // unset the rightmost set bit
			count++
		}
	}
	return count
}

// countCommonPrefixBits counts common bits in each path, starting from some position
func countCommonPrefixBits(data1, data2 []byte, from int) int {
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

// equalPrefixBits checks if the bits from n to m (inclusive) in the two paths are equal
func equalPrefixBits(data1, data2 []byte, n, m int) (bool, int) {
	for i := n; i < m; i++ {
		if getPathBit(data1, i) != getPathBit(data2, i) {
			return false, i
		}
	}
	return true, -1
}

// minBytes calculates the minimum number of bytes required to store an int
func minBytes(i int) int {
	if i == 0 {
		return 1
	}
	bytes := 0
	for i > 0 {
		bytes++
		i >>= 8
	}
	return bytes
}

// intToBytes converts an int to a byte slice
func intToBytes(i int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	d := minBytes(i)
	return b[8-d:]
}

// bytesToInt converts a byte slice to an int
func bytesToInt(bz []byte) int {
	b := make([]byte, 8) // allocate space for a 64-bit unsigned integer
	d := 8 - len(bz)     // determine how much padding is necessary
	copy(b[d:], bz)      // copy over the non-zero bytes
	u := binary.BigEndian.Uint64(b)
	return int(u)
}

// placeholder returns the default placeholder value depending on the trie type
func placeholder(spec *TrieSpec) []byte {
	if spec.sumTrie {
		placeholder := spec.th.placeholder()
		placeholder = append(placeholder, defaultSum[:]...)
		return placeholder
	}
	return spec.th.placeholder()
}

// hashSize returns the hash size depending on the trie type
func hashSize(spec *TrieSpec) int {
	if spec.sumTrie {
		return spec.th.hashSize() + sumSize
	}
	return spec.th.hashSize()
}

// digestLeaf returns the hash and preimage of a leaf node depending on the trie type
func digestLeaf(spec *TrieSpec, path, value []byte) ([]byte, []byte) {
	if spec.sumTrie {
		return spec.th.digestSumLeaf(path, value)
	}
	return spec.th.digestLeaf(path, value)
}

// digestNode returns the hash and preimage of a node depending on the trie type
func digestNode(spec *TrieSpec, left, right []byte) ([]byte, []byte) {
	if spec.sumTrie {
		return spec.th.digestSumNode(left, right)
	}
	return spec.th.digestNode(left, right)
}

// hashNode hashes a node depending on the trie type
func hashNode(spec *TrieSpec, node trieNode) []byte {
	if spec.sumTrie {
		return spec.hashSumNode(node)
	}
	return spec.hashNode(node)
}

// serialize serializes a node depending on the trie type
func serialize(spec *TrieSpec, node trieNode) []byte {
	if spec.sumTrie {
		return spec.sumSerialize(node)
	}
	return spec.serialize(node)
}

// hashPreimage hashes the serialised data provided depending on the trie type
func hashPreimage(spec *TrieSpec, data []byte) []byte {
	if spec.sumTrie {
		return hashSumSerialization(spec, data)
	}
	return hashSerialization(spec, data)
}

// Used for verification of serialized proof data
func hashSerialization(smt *TrieSpec, data []byte) []byte {
	if isExtension(data) {
		pathBounds, path, childHash := parseExtension(data, smt.ph)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return smt.hashNode(&ext)
	}
	return smt.th.digest(data)
}

// Used for verification of serialized proof data for sum trie nodes
func hashSumSerialization(smt *TrieSpec, data []byte) []byte {
	if isExtension(data) {
		pathBounds, path, childHash, _ := parseSumExtension(data, smt.ph)
		ext := extensionNode{path: path, child: &lazyNode{childHash}}
		copy(ext.pathBounds[:], pathBounds)
		return smt.hashSumNode(&ext)
	}
	digest := smt.th.digest(data)
	digest = append(digest, data[len(data)-sumSize:]...)
	return digest
}

// resolve resolves a lazy node depending on the trie type
func resolve(smt *SMT, hash []byte, resolver func([]byte) (trieNode, error),
) (trieNode, error) {
	if smt.sumTrie {
		return smt.resolveSum(hash, resolver)
	}
	return smt.resolve(hash, resolver)
}
