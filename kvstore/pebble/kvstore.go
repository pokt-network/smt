package pebble

import (
	"errors"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

const maxKeySize = 65000 // Pebble's maximum key size

var _ PebbleKVStore = &pebbleKVStore{}

type pebbleKVStore struct {
	db      *pebble.DB
	tempDir string
}

func NewKVStore(path string, tempDir string) (PebbleKVStore, error) {
	opts := &pebble.Options{}
	if path == "" {
		opts.FS = vfs.NewMem()
	}
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, errors.Join(ErrPebbleOpeningStore, err)
	}
	return &pebbleKVStore{db: db, tempDir: tempDir}, nil
}

func (store *pebbleKVStore) Set(key, value []byte) error {
	if key == nil || len(key) > maxKeySize {
		return ErrPebbleUnableToSetValue
	}
	err := store.db.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Join(ErrPebbleUnableToSetValue, err)
	}
	return nil
}

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

func (store *pebbleKVStore) Delete(key []byte) error {
	if key == nil || len(key) > maxKeySize {
		return ErrPebbleUnableToDeleteValue
	}
	err := store.db.Delete(key, pebble.Sync)
	if err != nil {
		return errors.Join(ErrPebbleUnableToDeleteValue, err)
	}
	return nil
}

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

func (store *pebbleKVStore) ClearAll() error {
	iter, _ := store.db.NewIter(nil)
	for iter.First(); iter.Valid(); iter.Next() {
		if err := store.db.Delete(iter.Key(), pebble.Sync); err != nil {
			iter.Close()
			return errors.Join(ErrPebbleClearingStore, err)
		}
	}
	if err := iter.Error(); err != nil {
		return errors.Join(ErrPebbleClearingStore, err)
	}
	iter.Close()
	return nil
}

func (store *pebbleKVStore) Stop() error {
	return store.db.Close()
}

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
