package badger

import (
	"errors"
	"io"
	"time"

	badgerv4 "github.com/dgraph-io/badger/v4"
	badgerv4opts "github.com/dgraph-io/badger/v4/options"
)

const (
	maxPendingWrites  = 16 // used in backup restoration
	gcIntervalMinutes = 5 * time.Minute
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

	store := &badgerKVStore{db: db}

	// Start value log GC in a separate goroutine
	go store.runValueLogGC()

	return store, nil
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

// TODO: Add/use BatchSet when multiple KV pairs need to be updated for performance benefits
// BatchSet sets/updates multiple key-value pairs in a single transaction
// func (store *badgerKVStore) BatchSet(keys, values [][]byte) error {
// 	if len(keys) != len(values) {
// 		return errors.New("mismatched number of keys and values")
// 	}
//
// 	wb := store.db.NewWriteBatch()
// 	defer wb.Cancel()
//
// 	for i := range keys {
// 		if err := wb.Set(keys[i], values[i]); err != nil {
// 			return errors.Join(ErrBadgerUnableToSetValue, err)
// 		}
// 	}
//
// 	if err := wb.Flush(); err != nil {
// 		return errors.Join(ErrBadgerUnableToSetValue, err)
// 	}
// 	return nil
// }

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
	var exists bool
	err := store.db.View(func(tx *badgerv4.Txn) error {
		_, err := tx.Get(key)
		if err == badgerv4.ErrKeyNotFound {
			exists = false
			return nil
		}
		if err != nil {
			return err
		}
		exists = true
		return nil
	})
	if err != nil {
		return false, errors.Join(ErrBadgerUnableToCheckExistence, err)
	}
	return exists, nil
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
func (store *badgerKVStore) Len() (int, error) {
	var count int
	err := store.db.View(func(tx *badgerv4.Txn) error {
		opt := badgerv4.DefaultIteratorOptions
		opt.Prefix = []byte{}
		opt.Reverse = false
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Seek(nil); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, errors.Join(ErrBadgerGettingStoreLength, err)
	}
	return count, nil
}

// runValueLogGC runs the value log garbage collection process periodically
func (store *badgerKVStore) runValueLogGC() {
	ticker := time.NewTicker(gcIntervalMinutes)
	defer ticker.Stop()

	for range ticker.C {
		err := store.db.RunValueLogGC(0.5)
		if err != nil && err != badgerv4.ErrNoRewrite {
			// Log the error, but don't stop the process
			// We might want to use a proper logging mechanism here
			println("Error during value log GC:", err)
		}
	}
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
	// Parameters should be adjusted carefully, depending on the type of load. We need to experiment more to find the best
	// values, and even then they might need further adjustments as the type of load/environment (e.g. memory dedicated
	// to the process) changes.
	//
	// Good links to read about options:
	// - https://github.com/dgraph-io/badger/issues/1304#issuecomment-630078745
	// - https://github.com/dgraph-io/badger/blob/master/options.go#L37
	// - https://github.com/open-policy-agent/opa/issues/4014#issuecomment-1003700744
	opts := badgerv4.DefaultOptions(path)

	// Disable badger's logger since it's very noisy
	opts.Logger = nil

	// Reduce MemTableSize from default 64MB to 32MB
	// This reduces memory usage but may increase disk I/O due to more frequent flushes
	opts.MemTableSize = 32 << 20

	// Decrease NumMemtables from default 5 to 3
	// This reduces memory usage but may slow down writes if set too low
	opts.NumMemtables = 3

	// Lower ValueThreshold from default 1MB to 256 bytes
	// This stores more data in LSM trees, reducing memory usage for small values
	// but may impact write performance for larger values
	opts.ValueThreshold = 256

	// Reduce BlockCacheSize from default 256MB to 32MB
	// This reduces memory usage but may slow down read operations
	opts.BlockCacheSize = 32 << 20

	// Adjust NumLevelZeroTables from default 5 to 3
	// This triggers compaction more frequently, potentially reducing memory usage
	// but at the cost of more disk I/O
	opts.NumLevelZeroTables = 3

	// Adjust NumLevelZeroTablesStall from default 15 to 8
	// This also helps in triggering compaction more frequently
	opts.NumLevelZeroTablesStall = 8

	// Change Compression from default Snappy to ZSTD
	// This can reduce memory and disk usage at the cost of higher CPU usage
	opts.Compression = badgerv4opts.ZSTD

	// Set ZSTDCompressionLevel to 3 (default is 1)
	// This provides better compression but increases CPU usage
	opts.ZSTDCompressionLevel = 3

	// Reduce BaseTableSize from default 2MB to 1MB
	// This might help in reducing memory usage, especially for many small keys
	opts.BaseTableSize = 1 << 20

	// Enable dynamic thresholding for value log (default is 0.0, which is disabled)
	// This might help in optimizing memory usage for specific workloads
	opts.VLogPercentile = 0.9

	return opts
}
