# Versioned Trees <!-- omit in toc -->

- [Overview](#overview)
- [Implementation](#implementation)
  - [SMT Methods](#smt-methods)
  - [Versioning](#versioning)
    - [Set Initial Version](#set-initial-version)
    - [Version Accessors](#version-accessors)
    - [Save Version](#save-version)
    - [Get Versioned Keys](#get-versioned-keys)
    - [Get Immutable Tree](#get-immutable-tree)
  - [Database Lifecycle](#database-lifecycle)

## Overview

The `VersionedSMT` interface defines an SMT that supports versioning. Where the versioned tree stores its previous versions allowing for historical queries and proofs to be generated. The interface is implemented in two separate types: `ImmutableTree` and `VersionedTree`. A `VersionedTree` is mutable and can be changed, and saved. Once saved a tree becomes immutable and the `Update`, `Delete`, `Commit`, `SetInitialVersion` and `SaveVersion` methods will cause a panic.

A `VersionedTree` maintains its own internal state, keeping track of its available versions and keeping a `KVStore` which it can use to retrieve the data for previous versions. The `KVStore` contains the information needed to import an `ImmutableTree` from its specific database path.

_NOTE: If the user deletes the database for a previous version the tree will not be able to import it and will cause a panic if use is attempted_

See: [immutable.go](./immutable.go) and [versioned.go](./versioned.go) for more details on the implementation specifics.

## Implementation

The `VersionedSMT` interface is as follows

```go
type VersionedSMT interface {
	SparseMerkleTree

	// --- Versioning ---
	SetInitialVersion(uint64) error
	Version() uint64
	AvailableVersions() []uint64
	SaveVersion() error
	VersionExists(uint64) bool
	GetVersioned(key []byte, version uint64) ([]byte, error)
	GetImmutable(uint64) (*ImmutableTree, error)

	// --- Database ---
	Stop() error
}
```

Both the `VersionedTree` and `ImmutableTree` types implement this interface where the `ImmutableTree` panics when any modifying method is called, due to it being immutable.

The `VersionedTree` implementation keeps track of the available versions, and keeps a `KVStore` to restore these versions when needed. The last saved version is not only stored with the others in the `KVStore` but is also kept open and accessible as an `ImmutableTree` embedded within the `VersionedTree` struct. This allows for the `VersionedTree` to easily access its most recent version without having to restore it from the `KVStore`.

When a new version is saved the previous version is closed and the current version (the one being saved) is imported as an `ImmutableTree`, as well as being saved in the `KVStore`, replacing the older previous version in the struct.

_NOTE: This does not over right the previous saved versions database at all only a pointer to an open, in memory `ImmutableTree`_

### SMT Methods

The inclusion of the `SparseMerkleTree` interface within the `VersionedSMT` interface enables the use of all the regular SMT methods.

See: [smt.go](./smt.go) and [types.go](./types.go) for more details on the SMT implementation.

### Versioning

The `VersionedSMT` interface naturally defines methods relevant to storing previous versions of the tree.

#### Set Initial Version

Upon creation a `VersionedTree` will by default have a version of 0, this can be overridden once, if and only if the tree has not been saved and its version incremented.

#### Version Accessors

The following version accessors have simple functionalities:

- `Version` method returns the current version of the `VersionedSMT`.
- `AvailableVersions` method returns a slice (`[]uint64`) of all the available versions of the `VersionedSMT`.
- `VersionExists` method returns a boolean indicating if the given version exists.

#### Save Version

The `SaveVersion` method is used to save the current version of the `VersionedTree`. As detailed above ([Implementation](#implementation)) this will keep the most recently saved version open and embedded in the `VersionedTree` struct, as well as saving the current version in the `KVStore`. This in memory previous stored version is used for easy access to the most recently saved version without having to open its database from the data retrieved from the `KVStore`.

A `VersionedTree` can be saved by encoding it into the following struct:

```go
type storedTree struct {
	Db_path string
	Root    []byte
	Version uint64
	Th      hash.Hash
	Ph      hash.Hash
	Vh      hash.Hash
}
```

This struct is what is serialised and stored in the `KVStore` with the key corresponding to the version (`binary.BigEndian.PutUint64(version)`).

If `max_stored_versions` was set during the creation/importing of a `VersionedTree` when saving if the number of saved versions exceeds `max_stored_versions` the oldest version will be deleted from the `KVStore` and the database will be deleted from the file system.

#### Get Versioned Keys

The `GetVersioned` method allows you to retrieve the value for a given key at the version specified, it does so by either using the current or previously stored versions which are open in memory (if the version is correct), or by opening the database using the data from the `KVStore` and retrieving the value from an imported `ImmutableTree`.

#### Get Immutable Tree

The `GetImmutable` method returns an `ImmutableTree` for the given version. It can only do so if the version has already been stored. If the version has not been stored it will return an error.

The `ImmutableTree` returned cannot be modified without directly writing to the underlying database, which **should never** be done.

_NOTE: If the user retrieves an `ImmutableTree` they are responsible for closing the connection to its underlying database, if the version is that of the previous stored version it **should not** be closed as this is expected to remain open_

### Database Lifecycle

As the `VersionedTree` requires a persistent `KVStore` the database must be closed properly once the tree is no longer in use. When calling the `SaveVersion` method this is handled automatically, however if the user wishes to stop using the tree before saving a version they must call the `Stop` method to close the database connection.

The `Stop` method will close the `VersionedTree`'s node store, as well as that of the previously stored `ImmutableTree` and also the `KVStore` that stores the saved versions.
