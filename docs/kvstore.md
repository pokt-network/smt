# MapStore

<!-- toc -->

- [Implementations](#implementations)
  * [SimpleMap](#simplemap)
  * [BadgerV4](#badgerv4)

<!-- tocstop -->

The `MapStore` is a simple interface used by the SM(S)T to store, delete and
retrieve key-value pairs. Due to its simplicity and more complex key-value
databases that implement it can be used in its place as the node store for a
trie.

See: [the interface](../kvstore/interfaces.go) for a more detailed description
of the simple interface required by the SM(S)T.

## Implementations

### SimpleMap

The `simplemap` is a simple kv-store shipped with the SM(S)T. But as a
submodule the library (the SMT) can be used without it. As long as the node
store provided to the SMT adheres to the `MapStore` interface **anything** can
be used.

This is recommemded for testing purposes or non-production usage.

### BadgerV4

As documented in the [badger](./badger-store.md) documentation the interface
implements the `MapStore` interface as well as many other methods, and offers
persistence.

See: [badger](../kvstore/badger/) for more details on the implementation of
this submodule.
