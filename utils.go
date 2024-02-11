package smt

import (
	"encoding/binary"
)

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
