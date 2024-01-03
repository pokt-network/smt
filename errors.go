package smt

import (
	"errors"
)

var (
	// ErrBadProof is returned when an invalid Merkle proof is supplied.
	ErrBadProof = errors.New("bad proof")
	// ErrKeyNotFound is returned when a key is not found in the tree.
	ErrKeyNotFound = errors.New("key not found")
)
