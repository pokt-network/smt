package smt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzSMT uses fuzzing to attempt to break the SMT implementation
// in its current state this fuzzing test does not confirm the SMT
// functions correctly, it only trys to detect when it fails unexpectedly
func FuzzSMT_DetectUnexpectedFailures(f *testing.F) {
	seeds := [][]byte{
		[]byte(""),
		[]byte("foo"),
		{1, 2, 3, 4},
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		nil,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		smn := NewSimpleMap()
		tree := NewSparseMerkleTree(smn, sha256.New())

		r := bytes.NewReader(input)
		var keys [][]byte

		// key returns a random byte to be used as a key, either generating a new
		// one or using a previously generated one with a 50/50 chance of either
		key := func() []byte {
			b := readByte(r)
			if b < math.MaxUint8/2 {
				k := make([]byte, b/2)
				if _, err := r.Read(k); err != nil {
					return nil
				}
				keys = append(keys, k)
				return k
			}

			if len(keys) == 0 {
				return nil
			}

			return keys[int(b)%len(keys)]
		}

		for i := 0; r.Len() != 0; i++ {
			originalRoot := tree.Root()
			b, err := r.ReadByte()
			if err != nil {
				continue
			}

			// Randomly select an operation to perform
			op := op(int(b) % int(NumOps))
			switch op {
			case Get:
				_, err := tree.Get(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while getting")
				}
				newRoot := tree.Root()
				require.Equal(t, originalRoot, newRoot, "root changed while getting")
			case Update:
				value := make([]byte, 32)
				binary.BigEndian.PutUint64(value, uint64(i))
				err := tree.Update(key(), value)
				require.NoErrorf(t, err, "unknown error occured while updating")
				newRoot := tree.Root()
				require.NotEqual(t, originalRoot, newRoot, "root unchanged while updating")
			case Delete:
				err := tree.Delete(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while deleting")
					continue
				}
				// If the key was present check root has changed
				newRoot := tree.Root()
				require.NotEqual(t, originalRoot, newRoot, "root unchanged while deleting")
			case Prove:
				_, err := tree.Prove(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while proving")
				}
				newRoot := tree.Root()
				require.Equal(t, originalRoot, newRoot, "root changed while proving")
			}

			newRoot := tree.Root()
			require.Greater(t, len(newRoot), 0, "new root is empty while err is nil")
		}
	})
}

// Fuzzing helpers
type op int

const (
	Get op = iota
	Update
	Delete
	Prove
	NumOps
)

func readByte(r *bytes.Reader) byte {
	b, err := r.ReadByte()
	if err != nil {
		return 0
	}
	return b
}
