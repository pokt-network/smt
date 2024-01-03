// Package kvstore provides a series of sub-modules that (if they comply with
// the MapStore interface) can be used as the underlying nodestore for the trie.
// Ultimatetly the trie's reqeuire only the methods exposed by the MapStore,
// and as such any more advanced wrappers or DBs can be used IFF they implement
// this interface.
package kvstore
