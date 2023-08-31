package smt

import (
	"fmt"
	"hash"
)

var _ VersionedSMT = (*ImmutableTree)(nil)

type ImmutableTree struct {
	*SMT
	db      KVStore
	version uint64
}

func ImportImmutableTree(nodes KVStore, hasher hash.Hash, version uint64, root []byte, spec *TreeSpec) *ImmutableTree {
	smt := ImportSparseMerkleTree(nodes, hasher, root)
	smt.TreeSpec = *spec
	return &ImmutableTree{
		SMT:     smt,
		db:      nodes,
		version: version,
	}
}

func (i *ImmutableTree) Update(key, value []byte) error {
	panic("immutable SMT cannot be updated")
}

func (i *ImmutableTree) Delete(key []byte) error {
	panic("immutable SMT cannot delete entries")
}

func (i *ImmutableTree) Commit() error {
	panic("immutable SMT cannot be committed")
}

func (i *ImmutableTree) SetInitialVersion(version uint64) error {
	panic("immutable SMT cannot set initial version")
}

func (i *ImmutableTree) Version() uint64 {
	return i.version
}

func (i *ImmutableTree) AvailableVersions() []uint64 {
	return []uint64{i.version}
}

func (i *ImmutableTree) SaveVersion() error {
	panic("immutable SMT cannot save versions")
}

func (i *ImmutableTree) VersionExists(version uint64) bool {
	return i.version == version
}

func (i *ImmutableTree) GetVersioned(key []byte, version uint64) ([]byte, error) {
	if version != i.version {
		return nil, fmt.Errorf("version %d does not exist", version)
	}
	return i.Get(key)
}

func (i *ImmutableTree) GetImmutable(version uint64) (*ImmutableTree, error) {
	if version != i.version {
		return nil, fmt.Errorf("version %d does not exist", version)
	}
	return i, nil
}
