package smt

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImmutableTree_CannotUpdate(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())

	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic")
		}
	}()

	ismt.Update([]byte("key"), []byte("value2"))
}

func TestImmutableTree_CannotDelete(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())

	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic")
		}
	}()

	ismt.Delete([]byte("key"))
}

func TestImmutableTree_CannotCommit(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())

	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic")
		}
	}()

	ismt.Commit()
}

func TestImmutableTree_CannotSetInitialVersion(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())

	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic")
		}
	}()

	ismt.SetInitialVersion(2)
}

func TestImmutableTree_CannotSaveVersion(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())

	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic")
		}
	}()

	ismt.SaveVersion()
}

func TestImmutableTree_GetVersion(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())
	require.Equal(t, uint64(1), ismt.Version())
}

func TestImmutableTree_VersionExists(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())
	require.True(t, ismt.VersionExists(1))
	require.False(t, ismt.VersionExists(2))
}

func TestImmutableTree_AvailableVersions(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())
	versions := ismt.AvailableVersions()
	require.Len(t, versions, 1)
	require.Equal(t, []uint64{1}, versions)
	require.Equal(t, uint64(1), versions[0])
}

func TestImmutableTree_GetImmutable(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New())

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())
	require.NoError(t, snm.Stop())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())
	i, err := ismt.GetImmutable(1)
	require.NoError(t, err)
	require.Equal(t, i, ismt)
	require.Equal(t, ismt.Version(), i.Version())

	i, err = ismt.GetImmutable(2)
	require.Error(t, err)
	require.EqualError(t, err, "version 2 does not exist")
	require.Nil(t, i)
}

func TestImmutableTree_GetVersioned(t *testing.T) {
	snm, err := NewKVStore("")
	require.NoError(t, err)
	smt := NewSparseMerkleTree(snm, sha256.New(), WithValueHasher(nil))

	require.NoError(t, smt.Update([]byte("key"), []byte("value")))
	require.NoError(t, smt.Commit())

	ismt := ImportImmutableTree(snm, sha256.New(), 1, smt.Root(), smt.Spec())
	val, err := ismt.GetVersioned([]byte("key"), 1)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), val)

	val, err = ismt.GetVersioned([]byte("key"), 2)
	require.Error(t, err)
	require.EqualError(t, err, "version 2 does not exist")
	require.Nil(t, val)

	require.NoError(t, snm.Stop())
}
