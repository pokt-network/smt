// TODO: replace with protobuf definitions for serialisation
package smt

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
)

// TODO: only currently works for sha256 Hasher
func init() {
	gob.Register(storedTree{})
	gob.Register(sha256.New())
}

func encodeStoredTree(st *storedTree) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(st)
	if err != nil {
		return nil, fmt.Errorf("error encoding: %v", err)
	}
	return buf.Bytes(), nil
}

func decodeStoredTree(b []byte) (*storedTree, error) {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	var st storedTree
	err := dec.Decode(&st)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %v", err)
	}
	return &st, nil
}
