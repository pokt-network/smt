package smt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzSMT(f *testing.F) {
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
		t.Log(input)
		smn := NewSimpleMap()
		tree := NewSparseMerkleTree(smn, sha256.New())

		r := bytes.NewReader(input)
		var keys [][]byte
		key := func() []byte {
			if readByte(r) < math.MaxUint8/2 {
				k := make([]byte, readByte(r)/2)
				if _, err := r.Read(k); err != nil {
					return nil
				}
				keys = append(keys, k)
				return k
			}

			if len(keys) == 0 {
				return nil
			}

			return keys[int(readByte(r))%len(keys)]
		}

		for i := 0; r.Len() != 0; i++ {
			b, err := r.ReadByte()
			if err != nil {
				continue
			}

			op := op(int(b) % int(Noop))
			switch op {
			case Get:
				_, err := tree.Get(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while getting")
				}
			case Update:
				value := make([]byte, 32)
				binary.BigEndian.PutUint64(value, uint64(i))
				err := tree.Update(key(), value)
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while updating")
				}
			case Delete:
				err := tree.Delete(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while deleting")
				}
			case Prove:
				_, err := tree.Prove(key())
				if err != nil {
					require.ErrorIsf(t, err, ErrKeyNotPresent, "unknown error occured while proving")
				}
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
	Noop
)

func readByte(r *bytes.Reader) byte {
	b, err := r.ReadByte()
	if err != nil {
		return 0
	}
	return b
}
