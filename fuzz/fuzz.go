package fuzz

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/pokt-network/smt"
)

func Fuzz(input []byte) int {
	if len(input) < 100 {
		return 0
	}

	smn := smt.NewSimpleMap()
	tree := smt.NewSparseMerkleTree(smn, sha256.New())

	r := bytes.NewReader(input)
	var keys [][]byte
	key := func() []byte {
		if readByte(r) < math.MaxUint8/2 {
			k := make([]byte, readByte(r)/2)
			if _, err := r.Read(k); err != nil {
				panic(err)
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
			if _, err := tree.Get(key()); err != nil {
				return 0
			}
		case Update:
			value := make([]byte, 32)
			binary.BigEndian.PutUint64(value, uint64(i))
			if err := tree.Update(key(), value); err != nil {
				return 0
			}
		case Delete:
			if err := tree.Delete(key()); err != nil {
				return 0
			}
		case Prove:
			if _, err := tree.Prove(key()); err != nil {
				return 0
			}
		}
	}

	return 1
}

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
