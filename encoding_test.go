package smt

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncoding_StoredTree(t *testing.T) {
	st := newStoredTree(t)
	bz, err := encodeStoredTree(st)
	require.NoError(t, err)
	require.NotNil(t, bz)

	st2, err := decodeStoredTree(bz)
	require.NoError(t, err)
	require.NotNil(t, st2)

	require.Equal(t, st.Db_path, st2.Db_path)
	require.Equal(t, st.Root, st2.Root)
	require.Equal(t, st.Version, st2.Version)
	require.Equal(t, st.Th, st2.Th)
	require.Equal(t, st.Ph, st2.Ph)
	require.Equal(t, st.Vh, st2.Vh)
}

func newStoredTree(t *testing.T) *storedTree {
	t.Helper()
	return &storedTree{
		Db_path: "test",
		Root:    []byte("test"),
		Version: 1,
		Th:      sha256.New(),
		Ph:      sha256.New(),
		Vh:      sha256.New(),
	}
}
