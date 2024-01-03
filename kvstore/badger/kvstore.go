package badger

import (
	"errors"
	"io"

	badgerv4 "github.com/dgraph-io/badger/v4"
)

const (
	maxPendingWrites = 16 // used in backup restoration
)

var _ BadgerKVStore = &badgerKVStore{}

type badgerKVStore struct {
	db         *badgerv4.DB
	lastBackup uint64 // timestamp of the most recent backup
}

// NewKVStore creates a new BadgerKVStore using badger as the underlying database
// if no path for a persistence database is provided it will create one in-memory
func NewKVStore(path string) (BadgerKVStore, error) {
	var db *badgerv4.DB
	var err error
	if path == "" {
		db, err = badgerv4.Open(badgerOptions("").WithInMemory(true))
	} else {
		db, err = badgerv4.Open(badgerOptions(path))
	}
	if err != nil {
		return nil, errors.Join(ErrBadgerOpeningStore, err)
	}
	return &badgerKVStore{db: db}, nil
}

// Set sets/updates the value for a given key
func (store *badgerKVStore) Set(key, value []byte) error {
	err := store.db.Update(func(tx *badgerv4.Txn) error {
		return tx.Set(key, value)
	})
	if err != nil {
		return errors.Join(ErrBadgerUnableToSetValue, err)
	}
	return nil
}

// Get returns the value for a given key
func (store *badgerKVStore) Get(key []byte) ([]byte, error) {
	var val []byte
	if err := store.db.View(func(tx *badgerv4.Txn) error {
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
		return nil, errors.Join(ErrBadgerUnableToGetValue, err)
	}
	return val, nil
}

// Delete removes a key and its value from the store
func (store *badgerKVStore) Delete(key []byte) error {
	err := store.db.Update(func(tx *badgerv4.Txn) error {
		return tx.Delete(key)
	})
	if err != nil {
		return errors.Join(ErrBadgerUnableToDeleteValue, err)
	}
	return nil
}

// GetAll returns all keys and values with the given prefix in the specified order
// if the prefix []byte{} is given then all key-value pairs are returned
func (store *badgerKVStore) GetAll(prefix []byte, descending bool) (keys, values [][]byte, err error) {
	if err := store.db.View(func(tx *badgerv4.Txn) error {
		opt := badgerv4.DefaultIteratorOptions
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
		return nil, nil, errors.Join(ErrBadgerIteratingStore, err)
	}
	return keys, values, nil
}

// Exists checks whether the key exists in the store
func (store *badgerKVStore) Exists(key []byte) (bool, error) {
	val, err := store.Get(key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// ClearAll deletes all key-value pairs in the store
func (store *badgerKVStore) ClearAll() error {
	if err := store.db.DropAll(); err != nil {
		return errors.Join(ErrBadgerClearingStore, err)
	}
	return nil
}

// Backup creates a full backup of the store written to the provided writer
// if incremental is true then only the changes since the last backup are written
func (store *badgerKVStore) Backup(w io.Writer, incremental bool) error {
	version := uint64(0)
	if incremental {
		version = store.lastBackup
	}
	timestamp, err := store.db.Backup(w, version)
	if err != nil {
		return errors.Join(ErrBadgerUnableToBackup, err)
	}
	store.lastBackup = timestamp
	return nil
}

// Restore loads the store from a backup in the reader provided
// NOTE: Do not call on a database that is running other concurrent transactions
func (store *badgerKVStore) Restore(r io.Reader) error {
	if err := store.db.Load(r, maxPendingWrites); err != nil {
		return errors.Join(ErrBadgerUnableToRestore, err)
	}
	return nil
}

// Stop closes the database connection, disabling any access to the store
func (store *badgerKVStore) Stop() error {
	if err := store.db.Close(); err != nil {
		return errors.Join(ErrBadgerClosingStore, err)
	}
	return nil
}

// Len gives the number of keys in the store
func (store *badgerKVStore) Len() int {
	count := 0
	if err := store.db.View(func(tx *badgerv4.Txn) error {
		opt := badgerv4.DefaultIteratorOptions
		opt.Prefix = []byte{}
		opt.Reverse = false
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Seek(nil); it.Valid(); it.Next() {
			count++
		}
		return nil
	}); err != nil {
		panic(errors.Join(ErrBadgerGettingStoreLength, err))
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
func badgerOptions(path string) badgerv4.Options {
	opts := badgerv4.DefaultOptions(path)
	opts.Logger = nil // disable badger's logger since it's very noisy
	return opts
}
