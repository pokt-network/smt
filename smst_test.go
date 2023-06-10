package smt

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMST_SumWorks(t *testing.T) {
	snm := NewSimpleMap()
	smst := NewSparseMerkleSumTree(snm, sha256.New())
	err := smst.Update([]byte("key1"), []byte("value1"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key2"), []byte("value2"), 5)
	require.NoError(t, err)
	err = smst.Update([]byte("key3"), []byte("value3"), 5)
	require.NoError(t, err)
	err = smst.Commit()
	require.NoError(t, err)
	sum, err := smst.Sum()
	require.NoError(t, err)
	require.Equal(t, sum, uint64(15))

	err = smst.Delete([]byte("key1"))
	require.NoError(t, err)
	sum, err = smst.Sum()
	require.NoError(t, err)
	require.Equal(t, sum, uint64(10))
}
