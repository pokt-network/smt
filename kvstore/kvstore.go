package kvstore

import (
	"errors"
	"io"
)

var (
	ErrKVStoreExists    = errors.New("kvstore already exists")
	ErrKVStoreNotExists = errors.New("kvstore does not exist")

	// ErrKVStoreKeyNotFound is returned when key isn't found.
	ErrKVStoreKeyNotFound = errors.New("key not found")

	// ErrKVStoreEmptyKey is returned if an empty key is passed on an update function.
	ErrKVStoreEmptyKey = errors.New("key cannot be empty")
)

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
