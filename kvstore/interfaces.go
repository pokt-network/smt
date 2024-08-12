package kvstore

// MapStore defines an interface that represents a key-value store that backs
// the SM(S)T. It is the minimum viable subset of functionality a key-value
// store requires in order to back an SM(S)T.
type MapStore interface {
	// --- Accessors ---

	// Get returns the value for a given key
	Get(key []byte) ([]byte, error)
	// Set sets/updates the value for a given key
	Set(key, value []byte) error
	// Delete removes a key
	Delete(key []byte) error
	// Len returns the number of key-value pairs in the store
	Len() (int, error)

	// --- Debug ---

	// ClearAll deletes all key-value pairs in the store
	ClearAll() error
}
