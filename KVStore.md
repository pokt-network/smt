# KVStore <!-- omit in toc -->

- [Overview](#overview)
- [Implementation](#implementation)
	- [In-Memory and Persistent](#in-memory-and-persistent)
	- [Store methods](#store-methods)
	- [Lifecycle Methods](#lifecycle-methods)
	- [Data Methods](#data-methods)
		- [Backups](#backups)
		- [Restorations](#restorations)
	- [Accessor Methods](#accessor-methods)
		- [Prefixed and Sorted Get All](#prefixed-and-sorted-get-all)
		- [Clear All Key-Value Pairs](#clear-all-key-value-pairs)
		- [Len](#len)

## Overview

The `KVStore` interface is a key-value store that is used by the `SMT` and `SMST` as its underlying database for its nodes. However, it is an independent key-value store that can be used for any purpose.

## Implementation

The `KVStore` is implemented in [`kvstore.go`](./kvstore.go) and is a wrapper around the [BadgerDB](https://github.com/dgraph-io/badger) key-value database.

The interface defines simple key-value store accessor methods as well as other methods desired from a key-value database in general, this can be found in [`kvstore.go`](./kvstore.go).

```go
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
```

_NOTE: The `KVStore` interface can be implemented by any key-value store that satisfies the interface and used as the underlying database store for the `SM(S)T`_

### In-Memory and Persistent

The `KVStore` implementation can be used as an in-memory or persistent key-value store. The `NewKVStore` function takes a `path` argument that can be used to specify a path to a directory to store the database files. If the `path` is an empty string, the database will be stored in-memory.

_NOTE: When providing a path for a persistent database, the directory must exist and be writeable by the user running the application._

### Store methods

As a key-value store the `KVStore` interface defines the simple `Get`, `Set` and `Delete` methods to access and modify the underlying database.

### Lifecycle Methods

The `Stop` method **must** be called when the `KVStore` is no longer needed. This method closes the underlying database and frees up any resources used by the `KVStore`.

For persistent databases, the `Stop` method should be called when the application no longer needs to access the database. For in-memory databases, the `Stop` method should be called when the `KVStore` is no longer needed.

_NOTE: A persistent `KVStore` that is not stopped will stop another `KVStore` from opening the database._

### Data Methods

The `KVStore` interface provides two methods to allow backups and restorations.

#### Backups

The `Backup` method takes an `io.Writer` and a `bool` to indicate whether the backup should be incremental or not. The `io.Writer` is then filled with the contents of the database in an opaque format used by the underlying database for this purpose.

When the `incremental` bool is `false` a full backup will be performed, otherwise an incremental backup will be performed. This is enabled by the `KVStore` keeping the timestamp of its last backup and only backing up data that has been modified since the last backup.

#### Restorations

The `Restore` method takes an `io.Reader` and restores the database from this reader.

The `KVStore` calling the `Restore` method is expected to be initialised and open, otherwise the restore will fail.

_NOTE: Any data contained in the `KVStore` when calling restore will be overwritten._

### Accessor Methods

The accessor methods enable simpler access to the underlying database for certain tasks that are desirable in a key-value store.

#### Prefixed and Sorted Get All

The `GetAll` method supports the retrieval of all keys and values, where the key has a specific prefix. The `descending` bool indicates whether the keys should be returned in descending order or not.

_NOTE: In order to retrieve all keys and values the empty prefix `[]byte{}` should be used to match all keys_

#### Clear All Key-Value Pairs

The `ClearAll` method removes all key-value pairs from the database.

_NOTE: The `ClearAll` method is intended to debug purposes and should not be used in production unless necessary_

#### Len

The `Len` method returns the number of keys in the database, similarly to how the `len` function can return the length of a map.
