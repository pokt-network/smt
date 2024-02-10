package simplemap

import (
	"github.com/pokt-network/smt/kvstore"
)

// Ensure that the SimpleMap can be used as an SMT node store
var _ kvstore.MapStore = (*simpleMap)(nil)

// simpleMap is a simple in-memory map.
type simpleMap struct {
	m map[string][]byte
}

// NewSimpleMap creates a new SimpleMap instance.
func NewSimpleMap() kvstore.MapStore {
	return &simpleMap{
		m: make(map[string][]byte),
	}
}

// NewSimpleMap creates a new SimpleMap instance using the map provided.
// This is useful for testing & debugging purposes.
func NewSimpleMapWithMap(m map[string][]byte) kvstore.MapStore {
	return &simpleMap{
		m: m,
	}
}

// Get gets the value for a key.
func (sm *simpleMap) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrKVStoreEmptyKey
	}

	if value, ok := sm.m[string(key)]; ok {
		return value, nil
	}

	return nil, ErrKVStoreKeyNotFound
}

// Set updates the value for a key.
func (sm *simpleMap) Set(key []byte, value []byte) error {
	if len(key) == 0 {
		return ErrKVStoreEmptyKey
	}
	sm.m[string(key)] = value
	return nil
}

// Delete deletes a key.
func (sm *simpleMap) Delete(key []byte) error {
	if len(key) == 0 {
		return ErrKVStoreEmptyKey
	}
	_, ok := sm.m[string(key)]
	if ok {
		delete(sm.m, string(key))
		return nil
	}
	return nil
}

// Len returns the number of key-value pairs in the store.
func (sm *simpleMap) Len() int {
	return len(sm.m)
}

// ClearAll clears all key-value pairs
// NB: This should only be used for testing purposes.
func (sm *simpleMap) ClearAll() error {
	sm.m = make(map[string][]byte)
	return nil
}
