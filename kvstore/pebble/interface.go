package pebble

import (
	"github.com/pokt-network/smt/kvstore"
)

// Ensure the PebbleKVStore can be used as an SMT node store
var _ kvstore.MapStore = (PebbleKVStore)(nil)

// PebbleKVStore is an interface that defines a key-value store
// that can be used standalone or as the node store for an SMT.
// This is a superset of the MapStore interface that offers more
// features and can be used as a standalone key-value store.
type PebbleKVStore interface {
	// --- Store methods ---
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) error
	// --- Lifecycle methods ---
	Stop() error
	// --- Data methods ---
	// Backup(writer io.Writer, incremental bool) error
	// Restore(io.Reader) error
	// --- Accessors ---
	GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error)
	Exists(key []byte) (bool, error)
	// Len returns the number of key-value pairs in the store
	Len() (int, error)
	// --- Data management ---
	ClearAll() error
}
