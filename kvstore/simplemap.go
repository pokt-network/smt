package kvstore

// MapStore defines an interface that represents a key-value store that backs
// the SM(S)T. It is a subset of the full functionality of a key-value store
// needed for the SM(S)T to function. By using a simplified interface any
// key-value store that implements these methods can be used with the SM(S)T.
type MapStore interface {
	// Accessors
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) error
	Len() int

	// Debug
	ClearAll() error
}

// simpleMap is a simple in-memory map.
type simpleMap struct {
	m map[string][]byte
}

// NewSimpleMap creates a new SimpleMap instance.
func NewSimpleMap() MapStore {
	return &simpleMap{
		m: make(map[string][]byte),
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
