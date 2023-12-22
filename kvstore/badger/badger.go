package badger

import (
	"errors"
	"fmt"
	"io"

	"github.com/dgraph-io/badger/v4"
	"github.com/pokt-network/smt/kvstore"
)

const (
	maxPendingWrites = 16 // used in backup restoration
)

var _ kvstore.KVStore = &badgerKVStore{}

type badgerKVStore struct {
	db          *badger.DB
	last_backup uint64 // timestamp of the most recent backup
}

// badgerToSMTErrorsMap maps badger errors to smt kvstore errors.
// This is necessary to achieve a consistent error interface
// across all kvstore implementations.
var badgerToSMTErrorsMap = map[error]error{
	badger.ErrKeyNotFound: kvstore.ErrKVStoreKeyNotFound,
	badger.ErrEmptyKey:    kvstore.ErrKVStoreEmptyKey,
}

// badgerToKVStoreError converts a badger error to a kvstore error
func badgerToKVStoreError(err error) error {
	for badgerError, smtError := range badgerToSMTErrorsMap {
		if errors.Is(err, badgerError) {
			return smtError
		}
	}

	return err
}

// NewKVStore creates a new KVStore using badger as the underlying database
// if no path for a peristence database is provided it will create one in-memory
func NewKVStore(path string) (kvstore.KVStore, error) {
	var db *badger.DB
	var err error
	if path == "" {
		db, err = badger.Open(badgerOptions("").WithInMemory(true))
	} else {
		db, err = badger.Open(badgerOptions(path))
	}
	if err != nil {
		return nil, badgerToKVStoreError(err)
	}

	return &badgerKVStore{db: db}, nil
}

// Set sets/updates the value for a given key
func (store *badgerKVStore) Set(key, value []byte) error {
	return badgerToKVStoreError(store.db.Update(func(tx *badger.Txn) error {
		return tx.Set(key, value)
	}))
}

// Get returns the value for a given key
func (store *badgerKVStore) Get(key []byte) ([]byte, error) {
	var val []byte
	if err := store.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, badgerToKVStoreError(err)
	}
	return val, nil
}

// Delete removes a key and its value from the store
func (store *badgerKVStore) Delete(key []byte) error {
	return badgerToKVStoreError(store.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(key)
	}))
}

// GetAll returns all keys and values with the given prefix in the specified order
// if the prefix []byte{} is given then all key-value pairs are returned
func (store *badgerKVStore) GetAll(prefix []byte, descending bool) (keys, values [][]byte, err error) {
	if err := store.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = prefix
		opt.Reverse = descending
		if descending {
			prefix = prefixEndBytes(prefix)
		}
		it := tx.NewIterator(opt)
		defer it.Close()
		keys = make([][]byte, 0)
		values = make([][]byte, 0)
		for it.Seek(prefix); it.Valid(); it.Next() {
			item := it.Item()
			err = item.Value(func(v []byte) error {
				b := make([]byte, len(v))
				copy(b, v)
				keys = append(keys, item.Key())
				values = append(values, b)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, nil, badgerToKVStoreError(err)
	}
	return keys, values, nil
}

// Exists checks whether the key exists in the store
func (store *badgerKVStore) Exists(key []byte) (bool, error) {
	val, err := store.Get(key)
	if err != nil {
		return false, badgerToKVStoreError(err)
	}
	return val != nil, nil
}

// ClearAll deletes all key-value pairs in the store
func (store *badgerKVStore) ClearAll() error {
	return badgerToKVStoreError(store.db.DropAll())
}

// Backup creates a full backup of the store written to the provided writer
// if incremental is true then only the changes since the last backup are written
func (store *badgerKVStore) Backup(w io.Writer, incremental bool) error {
	version := uint64(0)
	if incremental {
		version = store.last_backup
	}
	timestamp, err := store.db.Backup(w, version)
	if err != nil {
		return badgerToKVStoreError(err)
	}
	store.last_backup = timestamp
	return nil
}

// Restore loads the store from a backup in the reader provided
// NOTE: Do not call on a database that is running other concurrent transactions
func (store *badgerKVStore) Restore(r io.Reader) error {
	return badgerToKVStoreError(store.db.Load(r, maxPendingWrites))
}

// Stop closes the database connection, disabling any access to the store
func (store *badgerKVStore) Stop() error {
	return badgerToKVStoreError(store.db.Close())
}

// Len gives the number of keys in the store
func (store *badgerKVStore) Len() int {
	count := 0
	if err := store.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte{}
		opt.Reverse = false
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Seek(nil); it.Valid(); it.Next() {
			count++
		}
		return nil
	}); err != nil {
		panic(fmt.Sprintf("error getting key count: %v", err))
	}
	return count
}

// PrefixEndBytes returns the end byteslice for a noninclusive range
// that would include all byte slices for which the input is the prefix
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	if prefix[len(prefix)-1] == byte(255) {
		return prefixEndBytes(prefix[:len(prefix)-1])
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	end[len(end)-1]++
	return end
}

// badgerOptions returns the badger options for the store being created
func badgerOptions(path string) badger.Options {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // disable badger's logger since it's very noisy
	return opts
}
