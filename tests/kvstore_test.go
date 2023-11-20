package tests

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/badger"
	"github.com/stretchr/testify/require"
)

func TestKVStore_BasicOperations(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	invalidKey := [65001]byte{}
	testCases := []struct {
		name     string
		op       string
		key      []byte
		value    []byte
		fail     bool
		expected error
	}{
		{
			name:     "Successfully sets a value in the store",
			op:       "set",
			key:      []byte("testKey"),
			value:    []byte("testValue"),
			fail:     false,
			expected: nil,
		},
		{
			name:     "Successfully updates a value in the store",
			op:       "set",
			key:      []byte("foo"),
			value:    []byte("new value"),
			fail:     false,
			expected: nil,
		},
		{
			name:     "Fails to set value to nil key",
			op:       "set",
			key:      nil,
			value:    []byte("bar"),
			fail:     true,
			expected: badger.ErrEmptyKey,
		},
		{
			name:     "Fails to set a value to a key that is too large",
			op:       "set",
			key:      invalidKey[:],
			value:    []byte("bar"),
			fail:     true,
			expected: fmt.Errorf("Key with size 65001 exceeded 65000 limit. Key:\n%s", hex.Dump(invalidKey[:1<<10])),
		},
		{
			name:     "Successfully retrieve a value from the store",
			op:       "get",
			key:      []byte("foo"),
			value:    []byte("bar"),
			fail:     false,
			expected: nil,
		},
		{
			name:     "Fails to get a value that is not stored",
			op:       "get",
			key:      []byte("bar"),
			value:    nil,
			fail:     true,
			expected: badger.ErrKeyNotFound,
		},
		{
			name:     "Fails when the key is empty",
			op:       "get",
			key:      nil,
			value:    nil,
			fail:     true,
			expected: badger.ErrEmptyKey,
		},
		{
			name:     "Successfully deletes a value in the store",
			op:       "delete",
			key:      []byte("foo"),
			value:    nil,
			fail:     false,
			expected: nil,
		},
		{
			name:     "Fails to delete a value not in the store",
			op:       "delete",
			key:      []byte("bar"),
			value:    nil,
			fail:     false,
			expected: nil,
		},
		{
			name:     "Fails to set value to nil key",
			op:       "delete",
			key:      nil,
			value:    nil,
			fail:     true,
			expected: badger.ErrEmptyKey,
		},
		{
			name:     "Fails to set a value to a key that is too large",
			op:       "delete",
			key:      invalidKey[:],
			value:    nil,
			fail:     true,
			expected: fmt.Errorf("Key with size 65001 exceeded 65000 limit. Key:\n%s", hex.Dump(invalidKey[:1<<10])),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.ClearAll()
			require.NoError(t, err)
			setupStore(t, store)
			switch tc.op {
			case "set":
				err := store.Set(tc.key, tc.value)
				if tc.fail {
					require.Error(t, err)
					require.EqualError(t, tc.expected, err.Error())
				} else {
					require.NoError(t, err)
					got, err := store.Get(tc.key)
					require.NoError(t, err)
					require.Equal(t, tc.value, got)
				}
			case "get":
				got, err := store.Get(tc.key)
				if tc.fail {
					require.Error(t, err)
					require.EqualError(t, tc.expected, err.Error())
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.value, got)
				}
			case "delete":
				err := store.Delete(tc.key)
				if tc.fail {
					require.Error(t, err)
					require.EqualError(t, tc.expected, err.Error())
				} else {
					require.NoError(t, err)
					_, err := store.Get(tc.key)
					require.EqualError(t, err, badger.ErrKeyNotFound.Error())
				}
			}
		})
	}

	err = store.Stop()
	require.NoError(t, err)
}

func TestKVStore_GetAllBasic(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	keys := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte("bin"),
	}
	values := [][]byte{
		[]byte("oof"),
		[]byte("rab"),
		[]byte("zab"),
		[]byte("nib"),
	}

	for i := 0; i < len(keys); i++ {
		err := store.Set(keys[i], values[i])
		require.NoError(t, err)
	}

	allKeys, allValues, err := store.GetAll([]byte{}, false)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(allKeys))
	require.Equal(t, len(values), len(allValues))

	for i := 0; i < len(keys); i++ {
		require.Contains(t, allKeys, keys[i])
		require.Contains(t, allValues, values[i])
	}

	err = store.Stop()
	require.NoError(t, err)
}

func TestKVStore_GetAllPrefixed(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	keys := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte("bin"),
		[]byte("testKey1"),
		[]byte("testKey2"),
		[]byte("testKey3"),
		[]byte("testKey4"),
	}
	values := [][]byte{
		[]byte("oof"),
		[]byte("rab"),
		[]byte("zab"),
		[]byte("nib"),
		[]byte("testValue1"),
		[]byte("testValue2"),
		[]byte("testValue3"),
		[]byte("testValue4"),
	}

	for i := 0; i < len(keys); i++ {
		err := store.Set(keys[i], values[i])
		require.NoError(t, err)
	}

	allKeys, allValues, err := store.GetAll([]byte("testKey"), false)
	require.NoError(t, err)
	require.Equal(t, 4, len(allKeys))
	require.Equal(t, 4, len(allValues))

	for i := 0; i < len(keys); i++ {
		if strings.HasPrefix(string(keys[i]), "testKey") {
			require.Contains(t, allKeys, keys[i])
			require.Contains(t, allValues, values[i])
		} else {
			require.NotContains(t, allKeys, keys[i])
			require.NotContains(t, allValues, values[i])
		}
	}

	err = store.Stop()
	require.NoError(t, err)
}

func TestKVStore_Exists(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	keys := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte("bin"),
	}
	values := [][]byte{
		[]byte("oof"),
		nil,
		[]byte("zab"),
		[]byte("nib"),
	}

	for i := 0; i < len(keys); i++ {
		err := store.Set(keys[i], values[i])
		require.NoError(t, err)
	}

	// Key exists in store with a value
	exists, err := store.Exists([]byte("foo"))
	require.NoError(t, err)
	require.True(t, exists)

	// Key exists but has nil value
	exists, err = store.Exists([]byte("bar"))
	require.NoError(t, err)
	require.False(t, exists)

	// Key does not exist
	exists, err = store.Exists([]byte("oof"))
	require.EqualError(t, err, badger.ErrKeyNotFound.Error())
	require.False(t, exists)

	err = store.Stop()
	require.NoError(t, err)
}

func TestKVStore_ClearAll(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	keys := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte("bin"),
		[]byte("testKey1"),
		[]byte("testKey2"),
		[]byte("testKey3"),
		[]byte("testKey4"),
	}
	values := [][]byte{
		[]byte("oof"),
		[]byte("rab"),
		[]byte("zab"),
		[]byte("nib"),
		[]byte("testValue1"),
		[]byte("testValue2"),
		[]byte("testValue3"),
		[]byte("testValue4"),
	}

	for i := 0; i < len(keys); i++ {
		err := store.Set(keys[i], values[i])
		require.NoError(t, err)
	}

	allKeys, allValues, err := store.GetAll([]byte{}, false)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(allKeys))
	require.Equal(t, len(values), len(allValues))

	err = store.ClearAll()
	require.NoError(t, err)

	allKeys, allValues, err = store.GetAll([]byte{}, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(allKeys))
	require.Equal(t, 0, len(allValues))

	err = store.Stop()
	require.NoError(t, err)
}

func TestKVStore_BackupAndRestore(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	setupStore(t, store)

	keys, values, err := store.GetAll([]byte{}, false)
	require.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	err = store.Backup(buf, false)
	require.NoError(t, err)

	require.NoError(t, store.ClearAll())
	err = store.Restore(buf)
	require.NoError(t, err)

	newKeys, newValues, err := store.GetAll([]byte{}, false)
	require.NoError(t, err)

	require.Equal(t, keys, newKeys)
	require.Equal(t, values, newValues)
}

func TestKVStore_Len(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	tests := []struct {
		key   []byte
		value []byte
		size  int
	}{
		{
			key:   []byte("foo"),
			value: []byte("bar"),
			size:  1,
		},
		{
			key:   []byte("baz"),
			value: []byte("bin"),
			size:  2,
		},
		{
			key:   []byte("testKey1"),
			value: []byte("testValue1"),
			size:  3,
		},
	}

	for _, tc := range tests {
		require.NoError(t, store.Set(tc.key, tc.value))
		require.Equal(t, tc.size, store.Len())
	}
}

func setupStore(t *testing.T, store smt.KVStore) {
	t.Helper()
	err := store.Set([]byte("foo"), []byte("bar"))
	require.NoError(t, err)
	err = store.Set([]byte("baz"), []byte("bin"))
	require.NoError(t, err)
}
