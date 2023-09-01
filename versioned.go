package smt

import (
	"errors"
	"fmt"
	"hash"
	"path/filepath"

	"golang.org/x/exp/slices"
)

var _ VersionedSMT = (*VersionedTree)(nil)

// storedTree is a serialisable struct representing a stored VersionedTree
// it is stored in a KVStore for later retrieval, whereby an ImmutableTree
// will be imported using the contents of the struct.
type storedTree struct {
	Db_path string
	Root    []byte
	Version uint64
	Th      hash.Hash
	Ph      hash.Hash
	Vh      hash.Hash
}

// VersionedTree represents a Sparse Merkle tree that supports versioning
// where the previous version of the tree is kept open and accessible and
// all other versions are stored in a KVStore for later retrieval
//
// If max_stored_versions is 0, then all versions are stored, otherwise
// when the VersionedTree's latest version is Saved if the number of
// stored versions exceeds max_stored_versions, then the oldest version
// is pruned
type VersionedTree struct {
	*SMT
	db      KVStore
	version uint64

	store_path          string
	previous_stored     *ImmutableTree
	stored_versions     KVStore
	available_versions  []uint64
	max_stored_versions uint64 // 0 == all
}

// NewVersionedTree creates a new versioned sparse merkle tree that stores its
// previous versions in a key-value store, up to max_stored_versions or all if 0
//
// If the desired initial version is not 0, then the user must call SetInitialVersion
// on the VersionedTree before using it, in order to start at the correct version
//
// The store_path is used as a base directory for the previous version to be kept
// open and for all other versions to be accessible.
// NOTE: The store_path directory is expected to already exist
func NewVersionedTree(
	nodes KVStore,
	hasher hash.Hash,
	store_path string,
	max_stored_versions uint64,
	options ...Option,
) (*VersionedTree, error) {
	if store_path == "" {
		return nil, fmt.Errorf("store path cannot be empty")
	}
	if exists := dirExists(store_path); !exists {
		return nil, fmt.Errorf("store path does not exist: %s", store_path)
	}
	tree := NewSparseMerkleTree(nodes, hasher, options...)
	versions_store_path := filepath.Join(store_path, "versions")
	store_db, err := NewKVStore(versions_store_path)
	if err != nil {
		return nil, fmt.Errorf("failed to create version store db: %v", err)
	}
	return &VersionedTree{
		SMT:                 tree,
		db:                  nodes,
		store_path:          store_path,
		stored_versions:     store_db,
		available_versions:  make([]uint64, 0, max_stored_versions),
		max_stored_versions: max_stored_versions,
	}, nil
}

// ImportVersionedTree imports a versioned tree from the KVStore provided, and
// determines the available versions stored from the store_path provided. It will
// also load the previous stored ImmutableTree if it exists.
func ImportVersionedTree(
	nodes KVStore,
	hasher hash.Hash,
	root []byte,
	store_path string,
	max_stored_versions uint64,
	options ...Option,
) (*VersionedTree, error) {
	if store_path == "" {
		return nil, fmt.Errorf("store path cannot be empty")
	}
	if exists := dirExists(store_path); !exists {
		return nil, fmt.Errorf("store path does not exist: %s", store_path)
	}
	tree := ImportSparseMerkleTree(nodes, hasher, root, options...)
	versions_store_path := filepath.Join(store_path, "versions")
	store_db, err := NewKVStore(versions_store_path)
	if err != nil {
		return nil, fmt.Errorf("failed to create version store db: %v", err)
	}
	available_versions, err := getNumericDirs(store_path)
	if err != nil {
		return nil, fmt.Errorf("failed to get available versions: %v", err)
	}
	if max_stored_versions > 0 && uint64(len(available_versions)) > max_stored_versions {
		return nil, fmt.Errorf("too many versions found: %d > %d", len(available_versions), max_stored_versions)
	}
	vt := &VersionedTree{
		SMT:                 tree,
		db:                  nodes,
		store_path:          store_path,
		stored_versions:     store_db,
		available_versions:  available_versions,
		max_stored_versions: max_stored_versions,
	}
	if exists := dirExists(filepath.Join(store_path, "previous")); exists {
		prev_db, err := NewKVStore(filepath.Join(store_path, "previous"))
		if err != nil {
			return nil, fmt.Errorf("error opening previous stored tree node store: %v", err)
		}
		version := available_versions[len(available_versions)-1]
		versionBz := uint64ToBytes(version)
		encoded, err := store_db.Get(versionBz)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous version %d: %w", version, err)
		}
		st, err := decodeStoredTree(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode stored tree: %w", err)
		}
		prev := ImportImmutableTree(prev_db, st.Th, version, st.Root, specFromHashers(st.Th, st.Ph, st.Vh, false))
		vt.previous_stored = prev
	}
	return vt, nil
}

// SetInitialVersion sets the initial version of the tree
// NOTE: This call only works before the first save when the tree is at version 0
func (v *VersionedTree) SetInitialVersion(version uint64) error {
	if v.version != 0 {
		return fmt.Errorf("tree already at version: %d", v.version)
	}
	v.version = version
	return nil
}

// Version returns the current version of the tree
func (v *VersionedTree) Version() uint64 {
	return v.version
}

// AvailableVersions returns the list of versions stored
func (v *VersionedTree) AvailableVersions() []uint64 {
	return v.available_versions
}

// SaveVersion clones the current tree into an ImmutableTree keeping the node store
// open for access. It also backs up and stores the current version in the stored
// versions KVStore. When called any changes will be committed to the underlying
// node store to ensure recovery is possible.
//
// If max_stored_versions is non-zero it only keeps the last max_stored_versions
//
// Versions are stored under the store_path directory as subdirectorys named by
// the version number and the previous version is stored open at the subdirectory
// {store_path}/previous
func (v *VersionedTree) SaveVersion() error {
	if err := v.Commit(); err != nil {
		return fmt.Errorf("failed to commit tree: %w", err)
	}
	if v.max_stored_versions > 0 &&
		uint64(len(v.available_versions)) >= v.max_stored_versions {
		// remove oldest version
		versionBz := uint64ToBytes(v.available_versions[0])
		if err := v.stored_versions.Delete(versionBz); err != nil {
			return fmt.Errorf("failed to delete version %d", v.available_versions[0])
		}
		v.available_versions = v.available_versions[1:]
	}
	// save and keep open as previous saved
	if v.previous_stored != nil {
		v.previous_stored.db.Stop()
	}
	store_path := filepath.Join(v.store_path, fmt.Sprintf("%d", v.version))
	clonedNodes, err := v.db.Clone(store_path)
	if err != nil {
		return err
	}
	v.previous_stored = ImportImmutableTree(
		clonedNodes,
		v.th.hasher,
		v.version,
		v.Root(),
		specFromHashers(v.th.hasher, v.ph.Hasher(), v.vh.Hasher(), false),
	)
	// store current version
	sv := &storedTree{
		Db_path: filepath.Join(v.store_path, fmt.Sprintf("%d", v.version)),
		Root:    v.Root(),
		Version: v.version,
		Th:      v.th.hasher,
		Ph:      v.ph.Hasher(),
		Vh:      v.vh.Hasher(),
	}
	encoded, err := encodeStoredTree(sv)
	if err != nil {
		return fmt.Errorf("failed to encode stored tree: %v", err)
	}
	versionBz := uint64ToBytes(v.version)
	v.stored_versions.Set(versionBz, encoded)
	// increment version
	v.version += 1
	return nil
}

// VersionExists checks whether a version is stored and available
func (v *VersionedTree) VersionExists(version uint64) bool {
	idx := slices.IndexFunc(v.available_versions, func(sv uint64) bool {
		return sv == version
	})
	return idx != -1
}

// GetVersioned returns the value for a given key from the desired version
// of the VersionedTree
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

// GetImmutable returns an ImmutableTree for a given version, either returning
// the open previous stored tree if it is the correct version or else it loads
// an encoded stored tree from the stored_versions KVStore and returns an
// imported instance of an ImmutableTree from the correct KVStore
func (v *VersionedTree) GetImmutable(version uint64) (*ImmutableTree, error) {
	if !v.VersionExists(version) {
		return nil, fmt.Errorf("version %d does not exist", version)
	}
	if v.previous_stored != nil && v.previous_stored.Version() == version {
		return v.previous_stored, nil
	}
	versionBz := uint64ToBytes(version)
	encoded, err := v.stored_versions.Get(versionBz)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", version, err)
	}
	st, err := decodeStoredTree(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stored tree: %w", err)
	}
	db, err := NewKVStore(st.Db_path)
	if err != nil {
		return nil, err
	}
	return ImportImmutableTree(
		db,
		v.th.Hasher(),
		st.Version,
		st.Root,
		specFromHashers(st.Th, st.Ph, st.Vh, false),
	), nil
}

// Stop closes the node store, previous tree, and stored versions KVStores
func (v *VersionedTree) Stop() error {
	multierr := error(nil)
	if err := v.db.Stop(); err != nil {
		errors.Join(multierr, fmt.Errorf("failed to stop node store: %w", err))
	}
	if v.previous_stored != nil {
		if err := v.previous_stored.Stop(); err != nil {
			errors.Join(multierr, fmt.Errorf("failed to stop previous stored store: %w", err))
		}
	}
	if err := v.stored_versions.Stop(); err != nil {
		errors.Join(multierr, fmt.Errorf("failed to stop stored versions store: %w", err))
	}
	return multierr
}
