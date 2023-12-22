package kvstore

import (
	"errors"
)

var (
	// ErrKVStoreKeyNotFound is returned when a key is not present in the trie.
	ErrKVStoreKeyNotFound = errors.New("key already empty")
	// ErrKVStoreEmptyKey is returned when the given key is empty.
	ErrKVStoreEmptyKey = errors.New("key is empty")
)
