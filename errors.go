package smt

import (
	"errors"
)

// ErrBadProof is returned when an invalid Merkle proof is supplied.
var ErrBadProof = errors.New("bad proof")
