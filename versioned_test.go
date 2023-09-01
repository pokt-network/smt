package smt

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionedTree_Create(t *testing.T) {
	store_path := setup(t)
	_ = createVersionedTree(t, store_path, 3)
}

func TestVersionedTree_Import(t *testing.T) {
	store_path := setup(t)
	db, err := NewKVStore(filepath.Join(store_path, "db"))
	require.NoError(t, err)

	tree, err := NewVersionedTree(db, sha256.New(), store_path, 3, WithValueHasher(nil))
	require.NoError(t, err)
	require.NotNil(t, tree)
	t.Cleanup(func() {
		err := tree.Stop()
		require.NoError(t, err)
	})

	require.NoError(t, tree.Update([]byte("foo"), []byte("bar")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar1")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar2")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar3")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar4")))
	require.NoError(t, tree.Commit())

	require.NoError(t, tree.Stop())
	db, err = NewKVStore(filepath.Join(store_path, "db"))
	require.NoError(t, err)

	tree2, err := ImportVersionedTree(db, sha256.New(), tree.Root(), store_path, 3, WithValueHasher(nil))
	require.NoError(t, err)
	require.NotNil(t, tree2)
	t.Cleanup(func() {
		err := tree2.Stop()
		require.NoError(t, err)
	})

	val, err := tree2.Get([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar4"))
}

func TestVersionedTree_Version(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	version := tree.Version()
	require.Equal(t, uint64(0), version)

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(1), version)
}

func TestVersionedTree_SetInitialVersion(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	version := tree.Version()
	require.Equal(t, uint64(0), version)

	require.NoError(t, tree.SetInitialVersion(1))
	version = tree.Version()
	require.Equal(t, uint64(1), version)

	require.EqualError(t, tree.SetInitialVersion(2), "tree already at version: 1")
}

func TestVersionedTree_SaveVersion(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	version := tree.Version()
	require.Equal(t, uint64(0), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar")))
	require.Equal(t, tree.AvailableVersions(), []uint64{})

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(1), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar1")))
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0})

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(2), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar2")))
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1})

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(3), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar3")))
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1, 0x2})

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(4), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar4")))
	require.Equal(t, tree.AvailableVersions(), []uint64{0x1, 0x2, 0x3})
}

func TestVersionedTree_VersionExists(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	require.False(t, tree.VersionExists(0))

	require.NoError(t, tree.SaveVersion())
	require.True(t, tree.VersionExists(0))

	require.NoError(t, tree.SaveVersion())
	require.True(t, tree.VersionExists(1))

	require.NoError(t, tree.SaveVersion())
	require.True(t, tree.VersionExists(2))

	require.NoError(t, tree.SaveVersion())
	require.True(t, tree.VersionExists(3))
	require.False(t, tree.VersionExists(0))
}

func TestVersionedTree_AvailableVersions(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 0)

	require.Equal(t, tree.AvailableVersions(), []uint64{})
	require.NoError(t, tree.SaveVersion())
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0})
	require.NoError(t, tree.SaveVersion())
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1})
	require.NoError(t, tree.SaveVersion())
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1, 0x2})
	require.NoError(t, tree.SaveVersion())
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1, 0x2, 0x3})
	require.NoError(t, tree.SaveVersion())
	require.Equal(t, tree.AvailableVersions(), []uint64{0x0, 0x1, 0x2, 0x3, 0x4})
}

func TestVersionedTree_GetVersioned(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()

	version := tree.Version()
	require.Equal(t, uint64(0), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar")))
	val, err := tree.GetVersioned([]byte("foo"), 0)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar"))

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(1), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar1")))
	val, err = tree.GetVersioned([]byte("foo"), 0)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar"))
	val, err = tree.GetVersioned([]byte("foo"), 1)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar1"))

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(2), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar2")))
	val, err = tree.GetVersioned([]byte("foo"), 0)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar"))
	val, err = tree.GetVersioned([]byte("foo"), 1)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar1"))
	val, err = tree.GetVersioned([]byte("foo"), 2)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar2"))

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(3), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar3")))
	val, err = tree.GetVersioned([]byte("foo"), 0)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar"))
	val, err = tree.GetVersioned([]byte("foo"), 1)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar1"))
	val, err = tree.GetVersioned([]byte("foo"), 2)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar2"))
	val, err = tree.GetVersioned([]byte("foo"), 3)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar3"))

	require.NoError(t, tree.SaveVersion())
	version = tree.Version()
	require.Equal(t, uint64(4), version)
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar4")))
	val, err = tree.GetVersioned([]byte("foo"), 0)
	require.EqualError(t, err, "version 0 does not exist")
	require.Nil(t, val)
	val, err = tree.GetVersioned([]byte("foo"), 1)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar1"))
	val, err = tree.GetVersioned([]byte("foo"), 2)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar2"))
	val, err = tree.GetVersioned([]byte("foo"), 3)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar3"))
	val, err = tree.GetVersioned([]byte("foo"), 4)
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar4"))
}

func TestVersionedTree_GetImmutable(t *testing.T) {
	store_path := setup(t)
	tree := createVersionedTree(t, store_path, 3)

	require.NoError(t, tree.Update([]byte("foo"), []byte("bar")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar1")))
	require.NoError(t, tree.SaveVersion())
	require.NoError(t, tree.Update([]byte("foo"), []byte("bar2")))

	val, err := tree.Get([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar2"))

	itree, err := tree.GetImmutable(0)
	require.NoError(t, err)
	val, err = itree.Get([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar"))

	itree, err = tree.GetImmutable(1)
	require.NoError(t, err)
	val, err = itree.Get([]byte("foo"))
	require.NoError(t, err)
	require.Equal(t, val, []byte("bar1"))

	itree, err = tree.GetImmutable(2) // current version
	require.EqualError(t, err, "version 2 does not exist")
	require.Nil(t, itree)
}

func setup(t *testing.T) string {
	t.Helper()
	store_path, err := os.MkdirTemp("", "versioned_test")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(store_path)
		require.NoError(t, err)
	})
	return store_path
}

func createVersionedTree(t *testing.T, store_path string, max_versions uint64) VersionedSMT {
	t.Helper()
	db, err := NewKVStore(filepath.Join(store_path, "db"))
	require.NoError(t, err)

	tree, err := NewVersionedTree(db, sha256.New(), store_path, max_versions, WithValueHasher(nil))
	require.NoError(t, err)
	require.NotNil(t, tree)

	t.Cleanup(func() {
		err := tree.Stop()
		require.NoError(t, err)
	})

	return tree
}
