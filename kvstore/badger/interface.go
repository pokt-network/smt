package badger

import (
	"io"

	"github.com/pokt-network/smt/kvstore"
)

// Ensure the KVStore can be used as an SMT node store
var _ kvstore.MapStore = (KVStore)(nil)

// KVStore is an interface that defines a key-value store
// that can be used standalone or as the node store for an SMT.
type KVStore interface {
	// Store methods
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) error

	// Lifecycle methods
	Stop() error

	// Data methods
	Backup(writer io.Writer, incremental bool) error
	Restore(io.Reader) error

	// Accessors
	GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error)
	Exists(key []byte) (bool, error)
	ClearAll() error
	Len() int
}
