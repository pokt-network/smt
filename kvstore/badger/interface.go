package badger

import (
	"io"

	"github.com/pokt-network/smt/kvstore"
)

// Ensure the BadgerKVStore can be used as an SMT node store
var _ kvstore.MapStore = (BadgerKVStore)(nil)

// BadgerKVStore is an interface that defines a key-value store
// that can be used standalone or as the node store for an SMT.
// This is a superset of the MapStore interface that offers more
// features and can be used as a standalone key-value store.
type BadgerKVStore interface {
	// --- Store methods ---

	// Get returns the value for a given key
	Get(key []byte) ([]byte, error)
	// Set sets/updates the value for a given key
	Set(key, value []byte) error
	// Delete removes a key
	Delete(key []byte) error

	// --- Lifecycle methods ---

	// Stop closes the database connection, disabling any access to the store
	Stop() error

	// --- Data methods ---

	// Backup creates a full backup of the store written to the provided writer
	Backup(writer io.Writer, incremental bool) error
	// Restore loads the store from a backup in the reader provided
	Restore(io.Reader) error

	// --- Accessors ---

	// GetAll returns all keys and values with the given prefix in the specified order
	GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error)
	// Exists returns true if the key exists
	Exists(key []byte) (bool, error)
	// Len returns the number of key-value pairs in the store
	Len() (int, error)

	// --- Data management ---

	// ClearAll deletes all key-value pairs in the store
	ClearAll() error
}
