package fuzz

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
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
			_, err := tree.Get(key())
			if err != smt.ErrKeyNotPresent && err != nil {
				panic(fmt.Sprintf("error getting key: %s (%s)", key(), err.Error()))
			}
		case Update:
			value := make([]byte, 32)
			binary.BigEndian.PutUint64(value, uint64(i))

			err := tree.Update(key(), value)
			if err != smt.ErrKeyNotPresent && err != nil {
				panic(fmt.Sprintf("error updating key: %s (%s)", key(), err.Error()))
			}
		case Delete:
			err := tree.Delete(key())
			if err != smt.ErrKeyNotPresent && err != nil {
				panic(fmt.Sprintf("error deleting key: %s (%s)", key(), err.Error()))
			}
		case Prove:
			_, err := tree.Prove(key())
			if err != smt.ErrKeyNotPresent && err != nil {
				panic(fmt.Sprintf("error proving key: %s (%s)", key(), err.Error()))
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
