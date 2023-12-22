package simplemap

import (
	"encoding/hex"
	"fmt"
	"io"
	"slices"

	"github.com/pokt-network/smt/kvstore"
)

var _ kvstore.KVStore = &SimpleMap{}

const maxKeySize = 65000

// InvalidKeyError is thrown when a key that does not exist is being accessed.
type InvalidKeyError struct {
	Key []byte
}

func (e *InvalidKeyError) Error() string {
	return fmt.Sprintf("invalid key: %x", e.Key)
}

// SimpleMap is a simple in-memory map.
type SimpleMap struct {
	m map[string][]byte
}

// New creates a new empty SimpleMap.
func New() *SimpleMap {
	return &SimpleMap{
		m: make(map[string][]byte),
	}
}

// Get returns the value for a given key
func (sm *SimpleMap) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, kvstore.ErrKVStoreEmptyKey
	}

	if value, ok := sm.m[string(key)]; ok {
		return value, nil
	}

	return nil, kvstore.ErrKVStoreKeyNotFound
}

// exceedsSize mimics badger's exceedsSize function (why don't they have sentinel errors? :/)
func exceedsSize(prefix string, max int64, key []byte) error {
	return fmt.Errorf("%s with size %d exceeded %d limit. %s:\n%s", prefix, len(key), max, prefix, hex.Dump(key[:1<<10]))
}

// Set sets/updates the value for a given key
func (sm *SimpleMap) Set(key []byte, value []byte) error {
	if len(key) == 0 {
		return kvstore.ErrKVStoreEmptyKey
	}

	// weird kvstore badger compatibility
	if len(key) > maxKeySize {
		return exceedsSize("Key", maxKeySize, key)
	}

	sm.m[string(key)] = value
	return nil
}

// Delete removes a key and its value from the store
func (sm *SimpleMap) Delete(key []byte) error {
	if len(key) == 0 {
		return kvstore.ErrKVStoreEmptyKey
	}

	// weird kvstore badger compatibility
	if len(key) > maxKeySize {
		return exceedsSize("Key", maxKeySize, key)
	}

	_, ok := sm.m[string(key)]
	if ok {
		delete(sm.m, string(key))
		return nil
	}

	return nil
}

// Stop does nothing in SimpleMap.
// It is here to satisfy the KVStore interface.
func (sm *SimpleMap) Stop() error {
	return nil
}

// Backup is not implemented in SimpleMap.
// It is here to satisfy the KVStore interface.
func (sm *SimpleMap) Backup(writer io.Writer, incremental bool) error {
	return fmt.Errorf("backup functionality is not implemented in %T", sm)
}

// Restore is not implemented in SimpleMap.
// It is here to satisfy the KVStore interface.
func (sm *SimpleMap) Restore(io.Reader) error {
	return fmt.Errorf("restore functionality is not implemented in %T", sm)
}

// GetAll returns all keys and values with the given prefix in the specified order
// if the prefix []byte{} is given then all key-value pairs are returned
func (sm *SimpleMap) GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error) {
	matchingKeys := make([]string, 0)

	prefix := string(prefixKey)
	prefixLen := len(prefix)

	for k := range sm.m {
		if prefixLen == 0 || (len(k) >= prefixLen && k[:prefixLen] == prefix) {
			matchingKeys = append(matchingKeys, k)
		}
	}

	slices.Sort(matchingKeys)
	if descending {
		slices.Reverse(matchingKeys)
	}

	keys = make([][]byte, len(matchingKeys))
	values = make([][]byte, len(matchingKeys))
	for i, k := range matchingKeys {
		keys[i] = []byte(k)
		values[i] = sm.m[k]
	}

	return keys, values, nil
}

// Exists checks whether the key exists in the store
func (sm *SimpleMap) Exists(key []byte) (bool, error) {
	if len(key) == 0 {
		return false, kvstore.ErrKVStoreEmptyKey
	}
	value, exists := sm.m[string(key)]
	if !exists {
		return false, kvstore.ErrKVStoreKeyNotFound
	}

	return exists && value != nil, nil
}

// ClearAll deletes all key-value pairs in the store
func (sm *SimpleMap) ClearAll() error {
	sm.m = make(map[string][]byte)
	return nil
}

// Len gives the number of keys in the store
func (sm *SimpleMap) Len() int {
	return len(sm.m)
}
