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
	kvstore.MapStore

	// --- Lifecycle methods ---
	Stop() error
	// --- Accessors ---
	GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error)
	Exists(key []byte) (bool, error)
}
