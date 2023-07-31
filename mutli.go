package smt

import (
	"crypto/sha256"
	"fmt"
	"hash"
)

var (
	_ MultiStore = (*multi)(nil)
	_ Store      = (*store)(nil)

	// StoreCreator is a function that creates a new Store for the given MultiStore
	StoreCreator = func(name string, ms MultiStore) (Store, MapStore) {
		db := NewSimpleMap()
		return NewStore(name, ms, db, sha256.New()), db
	}
)

type multi struct {
	nodeStore MapStore
	tree      *SMT
	storeMap  map[string]Store
	dbMap     map[string]MapStore
}

// NewMultiStore creates a new instance of the MultiStore
func NewMultiStore(db MapStore, smt *SMT) MultiStore {
	return &multi{
		nodeStore: db,
		tree:      smt,
		storeMap:  make(map[string]Store, 0),
		dbMap:     make(map[string]MapStore, 0),
	}
}

// AddStore adds a store to the MultiStore using the store creator function provided
// this does not update the MutliStore tree to contain the root of the new store
func (m *multi) AddStore(name string, creator func(name string, ms MultiStore) (Store, MapStore)) error {
	if _, ok := m.storeMap[name]; ok {
		return fmt.Errorf("store already exists: %s", name)
	}
	store, db := creator(name, m)
	m.storeMap[name] = store
	m.dbMap[name] = db
	return nil
}

// InsertStore inserts a premade store into the MultiStore and updates the
// MultiStore tree to contain the root of the inserted store
func (m *multi) InsertStore(name string, store Store, db MapStore) error {
	if _, ok := m.storeMap[name]; ok {
		return fmt.Errorf("store already exists: %s", name)
	}
	m.storeMap[name] = store
	m.dbMap[name] = db
	return m.tree.Update([]byte(name), store.Root())
}

// GetStore returns a store from the MultiStore
func (m *multi) GetStore(name string) (Store, error) {
	if store, ok := m.storeMap[name]; ok {
		return store, nil
	}
	return nil, fmt.Errorf("store not found: %s", name)
}

// RemoveStore removes a store from the MultiStore
func (m *multi) RemoveStore(name string) error {
	if _, ok := m.storeMap[name]; !ok {
		return fmt.Errorf("store not found: %s", name)
	}
	delete(m.storeMap, name)
	delete(m.dbMap, name)
	return nil
}

// Commit calls commit on each of the stores tracked by the MultiStore
// which updates the MutliStore to contain their root hashes and then
// commits its own tree to the underlying database
func (m *multi) Commit() error {
	for _, store := range m.storeMap {
		if err := store.Commit(); err != nil {
			return err
		}
	}
	return m.tree.Commit()
}

// Root returns the root hash of the MultiStore
func (m *multi) Root() []byte {
	return m.tree.Root()
}

type store struct {
	*SMT
	name      string
	nodeStore MapStore
	multi     *multi
}

// NewStore creates a new instance of an SMT with the arguments provided and
// returns the Store wrapper around the SMT and the underlying database
func NewStore(name string, ms MultiStore, db MapStore, hasher hash.Hash, options ...Option) Store {
	smt := NewSparseMerkleTree(db, hasher, options...)
	return &store{
		SMT:       smt,
		name:      name,
		nodeStore: db,
		multi:     ms.(*multi),
	}
}

// Commit commits any changes to the underlying database and
// also updates the MultiStore to include its latest root hash
func (s *store) Commit() error {
	if err := s.SMT.Commit(); err != nil {
		return fmt.Errorf("failed to commit store (%s): %w", s.name, err)
	}
	if err := s.multi.tree.Update([]byte(s.name), s.Root()); err != nil {
		return fmt.Errorf("failed to update multi tree (%s): %w", s.name, err)
	}
	return nil
}
