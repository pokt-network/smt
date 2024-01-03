// Package kvstore provides a simple interface for a key-value store, the
// `MapStore` and a simple implementation of this interface. These are provided
// for convenience and for simple in-memory use cases. The KVStore package also
// contains submodules with more complex implementations of key-value stores
// that can be used independently, or as the backend for the SM(S)T.
// These submodules satisfy the simple `MapStore` interface as well as also
// providing their own more complex interfaces, which can provide more features
// and better performance - such as persistence, backups and restores for example.
// These are included as submodules as they are not required for the SM(S)T and
// any key-value store that satisfies the `MapStore` interface can be used with
// the SM(S)T.
package simplemap
