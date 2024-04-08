# MapStore <!-- omit in toc -->

- [Introduction](#introduction)
- [Implementations](#implementations)
  - [SimpleMap](#simplemap)
  - [BadgerV4](#badgerv4)
- [Note On External Writability](#note-on-external-writability)

## Introduction

The `MapStore` is a simple interface used by the SM(S)T to store, delete and
retrieve key-value pairs. It is intentionally simple and minimalistic so as to
enable different key-value engines to implement and back the trie database.

See: [the interface](../kvstore/interfaces.go) for a more detailed description
of the simple interface required by the SM(S)T.

## Implementations

### SimpleMap

`simplemap` is a simple kv-store shipped with the SM(S)T. The SMT library that
can be used without it as long as the selected node store adheres to the
`Mapstore` interface.

This library is recommended for development, testing and exploration purposes.

This library **SHOULD NOT** be used in production.

See [simplemap.go](../kvstore/simplemap/simplemap.go) for more details.

### BadgerV4

This library provides a wrapper around [dgraph-io/badger][https://github.com/dgraph-io/badger] to adhere to
the `MapStore` interface. See the [full documentation](./badger-store.md) for
additional functionality and implementation details.

See: [badger](../kvstore/badger/) for more details on the implementation of this
submodule.

## Note On External Writability

Any key-value store used by the tries should **not** be able to be externally
writeable in production. This opens the possibility to attacks where the writer
can modify the trie database and prove values that were not inserted.
