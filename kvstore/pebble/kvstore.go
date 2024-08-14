package pebble

import (
	"errors"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

var _ PebbleKVStore = &pebbleKVStore{}

type pebbleKVStore struct {
	db *pebble.DB
}

// NewKVStore creates a new PebbleKVStore instance.
// If path is empty, it creates an in-memory store.
func NewKVStore(path string) (PebbleKVStore, error) {
	store := &pebbleKVStore{}

	opts := &pebble.Options{}
	if path == "" {
		opts.FS = vfs.NewMem()
	}

	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, errors.Join(ErrPebbleOpeningStore, err)
	}
	store.db = db
	return store, nil
}

// Set stores a key-value pair in the database.
func (store *pebbleKVStore) Set(key, value []byte) error {
	if key == nil {
		return ErrPebbleUnableToSetValue
	}
	err := store.db.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Join(ErrPebbleUnableToSetValue, err)
	}
	return nil
}

// Get retrieves the value associated with the given key.
func (store *pebbleKVStore) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, ErrPebbleUnableToGetValue
	}
	value, closer, err := store.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrPebbleUnableToGetValue
		}
		return nil, errors.Join(ErrPebbleUnableToGetValue, err)
	}
	defer closer.Close()
	return append([]byte{}, value...), nil
}

// Delete removes the key-value pair associated with the given key.
func (store *pebbleKVStore) Delete(key []byte) error {
	if key == nil {
		return ErrPebbleUnableToDeleteValue
	}
	err := store.db.Delete(key, pebble.Sync)
	if err != nil {
		return errors.Join(ErrPebbleUnableToDeleteValue, err)
	}
	return nil
}

// GetAll retrieves all key-value pairs with keys starting with the given prefix.
// If descending is true, it returns the results in reverse lexicographical order.
func (store *pebbleKVStore) GetAll(prefix []byte, descending bool) (keys, values [][]byte, err error) {
	iter, _ := store.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: prefixEndBytes(prefix),
	})
	defer iter.Close()

	if descending {
		for valid := iter.Last(); valid; valid = iter.Prev() {
			keys = append(keys, append([]byte{}, iter.Key()...))
			values = append(values, append([]byte{}, iter.Value()...))
		}
	} else {
		for iter.First(); iter.Valid(); iter.Next() {
			keys = append(keys, append([]byte{}, iter.Key()...))
			values = append(values, append([]byte{}, iter.Value()...))
		}
	}

	if err := iter.Error(); err != nil {
		return nil, nil, errors.Join(ErrPebbleIteratingStore, err)
	}

	return keys, values, nil
}

// Exists checks if a key exists in the store and has a non-empty value.
func (store *pebbleKVStore) Exists(key []byte) (bool, error) {
	value, closer, err := store.db.Get(key)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, errors.Join(ErrPebbleUnableToGetValue, err)
	}
	defer closer.Close()
	return len(value) > 0, nil
}

// ClearAll removes all key-value pairs from the store.
func (store *pebbleKVStore) ClearAll() error {
	iter, _ := store.db.NewIter(nil)
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		if err := store.db.Delete(iter.Key(), pebble.Sync); err != nil {
			return errors.Join(ErrPebbleClearingStore, err)
		}
	}
	if err := iter.Error(); err != nil {
		return errors.Join(ErrPebbleClearingStore, err)
	}
	return nil
}

// Stop closes the database connection.
func (store *pebbleKVStore) Stop() error {
	return store.db.Close()
}

// Len returns the number of key-value pairs in the store.
func (store *pebbleKVStore) Len() (int, error) {
	count := 0
	iter, _ := store.db.NewIter(nil)
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}
	if err := iter.Error(); err != nil {
		return 0, errors.Join(ErrPebbleGettingStoreLength, err)
	}
	return count, nil
}

// prefixEndBytes returns the end key for prefix scans.
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // when all bytes are 0xff
}
