package smt

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// var _ VersionedSMT = (*VersionedTree)(nil)
type storedTree struct {
	db      KVStore
	root    []byte
	version uint64
	spec    *TreeSpec
}

type VersionedTree struct {
	*SMT
	db                  KVStore
	version             *uint64
	stored_versions     []storedTree
	max_stored_versions uint64 // 0 == all
}

func (v *VersionedTree) SetInitialVersion(version uint64) error {
	if v.version != nil {
		return fmt.Errorf("tree already at version: %d", *v.version)
	}
	v.version = &version
	return nil
}

func (v *VersionedTree) Version() uint64 {
	return *v.version
}

func (v *VersionedTree) AvailableVersions() []uint64 {
	versions := make([]uint64, len(v.stored_versions))
	for _, st := range v.stored_versions {
		versions = append(versions, st.version)
	}
	return versions
}

func (v *VersionedTree) SaveVersion() error {
	if err := v.Commit(); err != nil {
		return fmt.Errorf("failed to commit tree: %w", err)
	}
	if v.max_stored_versions > 0 &&
		uint64(len(v.stored_versions)) >= v.max_stored_versions {
		// remove oldest version
		v.stored_versions = v.stored_versions[1:]
	}
	// add new version
	v.stored_versions = append(v.stored_versions, storedTree{
		v.db, // TODO: need to clone
		v.Root(),
		*v.version,
		v.Spec().clone(),
	})
	// increment version
	*v.version += 1
	return nil
}

func (v *VersionedTree) VersionExists(version uint64) bool {
	idx := slices.IndexFunc(v.stored_versions, func(st storedTree) bool {
		return st.version == version
	})
	return idx != -1
}

func (v *VersionedTree) GetVersioned(key []byte, version uint64) ([]byte, error) {
	if !v.VersionExists(version) {
		return nil, fmt.Errorf("version %d does not exist", version)
	}
	imt, err := v.GetImmutable(version)
	if err != nil {
		return nil, err
	}
	return imt.Get(key)
}

func (v *VersionedTree) GetImmutable(version uint64) (*ImmutableTree, error) {
	if !v.VersionExists(version) {
		return nil, fmt.Errorf("version %d does not exist", version)
	}
	idx := slices.IndexFunc(v.stored_versions, func(st storedTree) bool {
		return st.version == version
	})
	st := v.stored_versions[idx]
	return ImportImmutableTree(st.db, st.spec.th.hasher, version, st.root, st.spec), nil
}
