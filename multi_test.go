package smt

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiStore_AddStore(t *testing.T) {
	multi := setupMultiStore(t)

	require.NoError(t, multi.AddStore("test", StoreCreator))

	require.EqualError(t, multi.AddStore("test", StoreCreator), "store already exists: test")
}

func TestMultiStore_InsertStore(t *testing.T) {
	multi := setupMultiStore(t)

	store, db := StoreCreator("test", multi)
	require.NotNil(t, store)
	require.NotNil(t, db)

	require.NoError(t, multi.InsertStore("test", store, db))

	require.EqualError(t, multi.InsertStore("test", store, db), "store already exists: test")
}

func TestMultiStore_GetStore(t *testing.T) {
	multi := setupMultiStore(t)

	require.NoError(t, multi.AddStore("test", StoreCreator))

	store, err := multi.GetStore("test")
	require.NoError(t, err)
	require.NotNil(t, store)

	store, err = multi.GetStore("test2")
	require.EqualError(t, err, "store not found: test2")
	require.Nil(t, store)
}

func TestMultiStore_RemoveStore(t *testing.T) {
	multi := setupMultiStore(t)

	require.NoError(t, multi.AddStore("test", StoreCreator))

	require.NoError(t, multi.RemoveStore("test"))

	require.EqualError(t, multi.RemoveStore("test"), "store not found: test")
}

func TestMultiStore_StoreOperations(t *testing.T) {
	db := NewSimpleMap()
	multi := setupMultiStore(t)
	store, _ := customStoreCreator(t, "test", db, multi)

	// check multi tree empty
	multiRoot1 := multi.Root()
	require.Equal(t, hex.EncodeToString(multiRoot1), "0000000000000000000000000000000000000000000000000000000000000000")

	require.NoError(t, store.Update([]byte("foo"), []byte("bar")))

	// check store root updates
	root := store.Root()
	require.Equal(t, hex.EncodeToString(root), "ace64ee83ecf596655deac72c646a30ae7bd71635992cd4c1a5a10350fcc1c52")

	// check insert updates multi tree root
	require.NoError(t, multi.InsertStore("test", store, db))
	multiRoot2 := multi.Root()
	require.Equal(t, hex.EncodeToString(multiRoot2), "726e2a2b5497e9472b6e6ff5cb5cec0fa145359a130e0e969adf8ada8173c1e4")

	store, err := multi.GetStore("test")
	require.NoError(t, err)

	require.NoError(t, store.Update([]byte("foo"), []byte("bar2")))

	// check store root updates
	root2 := store.Root()
	require.Equal(t, hex.EncodeToString(root2), "956e81f5c0bb44396fd79c3120c0aef2c2ad0009f3974c0ab7105e01c8ed094f")

	// check multi tree root doesnt update
	multiRoot3 := multi.Root()
	require.Equal(t, multiRoot3, multiRoot2)

	// check multi tree root updates after store commits
	require.NoError(t, store.Commit())
	multiRoot4 := multi.Root()
	require.Equal(t, hex.EncodeToString(multiRoot4), "0175c22dd60ad4db9b1e0bbfd0f7dd8235e4c66b74e04c5282dfbcc895f9085a")
}

func TestMultiStore_Commit(t *testing.T) {
	multi := setupMultiStore(t)

	require.NoError(t, multi.AddStore("test", StoreCreator))
	require.NoError(t, multi.AddStore("test2", StoreCreator))

	store, err := multi.GetStore("test")
	require.NoError(t, err)
	store2, err := multi.GetStore("test2")
	require.NoError(t, err)

	// update the stores
	require.NoError(t, store.Update([]byte("foo"), []byte("bar")))
	require.NoError(t, store2.Update([]byte("foo2"), []byte("bar2")))

	// check store roots update
	root1 := store.Root()
	require.Equal(t, hex.EncodeToString(root1), "ace64ee83ecf596655deac72c646a30ae7bd71635992cd4c1a5a10350fcc1c52")
	root2 := store2.Root()
	require.Equal(t, hex.EncodeToString(root2), "c8eec74eb4db3fae8caae0e308025fd8027e2303e47c2b9a5cfe63d083e7b689")

	// check multi tree root doesnt update
	multiRoot1 := multi.Root()
	require.Equal(t, hex.EncodeToString(multiRoot1), "0000000000000000000000000000000000000000000000000000000000000000")

	// check multi tree root updates after commit
	require.NoError(t, multi.Commit())
	multiRoot2 := multi.Root()
	require.Equal(t, hex.EncodeToString(multiRoot2), "3b93279b1113300f2d2009a3287b35845fce522c8d3f3f3a81d97388efa54db5")
}

func TestMultiStore_Prove(t *testing.T) {
	multi := setupMultiStore(t)

	require.NoError(t, multi.AddStore("test", StoreCreator))

	store, err := multi.GetStore("test")
	require.NoError(t, err)

	// update the store
	require.NoError(t, store.Update([]byte("foo"), []byte("bar")))
	root1 := store.Root()
	require.Equal(t, hex.EncodeToString(root1), "ace64ee83ecf596655deac72c646a30ae7bd71635992cd4c1a5a10350fcc1c52")
	require.NoError(t, store.Commit())

	// generate proof
	proof, err := multi.Prove([]byte("test"))
	require.NoError(t, err)
	require.NotNil(t, proof)

	// verify proof
	valid := VerifyProof(proof, multi.Root(), []byte("test"), root1, multi.Spec())
	require.True(t, valid)
	invalid := VerifyProof(proof, multi.Root(), []byte("test"), []byte("foo"), multi.Spec())
	require.False(t, invalid)
	invalid = VerifyProof(proof, multi.Root(), []byte("test"), nil, multi.Spec())
	require.False(t, invalid)
}

func setupMultiStore(t *testing.T) MultiStore {
	t.Helper()
	db := NewSimpleMap()
	smt := NewSparseMerkleTree(db, sha256.New())
	multi := NewMultiStore(db, smt)
	require.NotNil(t, multi)
	require.Implements(t, (*MultiStore)(nil), multi)
	return multi
}

func customStoreCreator(t *testing.T, name string, db MapStore, multi MultiStore) (Store, MapStore) {
	t.Helper()
	store := NewStore(name, multi, db, sha256.New())
	require.NotNil(t, store)
	require.Implements(t, (*Store)(nil), store)
	return store, db
}
