package kvstore

// MapStore defines an interface that represents a key-value store that backs
// the SM(S)T. It is the minimum viable subset of functionality a key-value
// store requires in order to back an SM(S)T.
type MapStore interface {
	// --- Accessors ---
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) error
	Len() int

	// --- Debug ---
	ClearAll() error
}
