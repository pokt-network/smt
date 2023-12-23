package badger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/badger"
)

func TestBadger_KVStore_BasicOperations(t *testing.T) {
	store, err := badger.NewKVStore("")
	require.NoError(t, err)
	require.NotNil(t, store)

	invalidKey := [65001]byte{}
	testCases := []struct {
		desc        string
		op          string
		key         []byte
		value       []byte
		fail        bool
		expectedErr error
	}{
		{
			desc:        "Successfully sets a value in the store",
			op:          "set",
			key:         []byte("testKey"),
			value:       []byte("testValue"),
			fail:        false,
			expectedErr: nil,
		},
		{
			desc:        "Successfully updates a value in the store",
			op:          "set",
			key:         []byte("foo"),
			value:       []byte("new value"),
			fail:        false,
			expectedErr: nil,
		},
		{
			desc:        "Fails to set value to nil key",
			op:          "set",
			key:         nil,
			value:       []byte("bar"),
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToSetValue,
		},
		{
			desc:        "Fails to set a value to a key that is too large",
			op:          "set",
			key:         invalidKey[:],
			value:       []byte("bar"),
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToSetValue,
		},
		{
			desc:        "Successfully retrieve a value from the store",
			op:          "get",
			key:         []byte("foo"),
			value:       []byte("bar"),
			fail:        false,
			expectedErr: nil,
		},
		{
			desc:        "Fails to get a value that is not stored",
			op:          "get",
			key:         []byte("bar"),
			value:       nil,
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToGetValue,
		},
		{
			desc:        "Fails when the key is empty",
			op:          "get",
			key:         nil,
			value:       nil,
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToGetValue,
		},
		{
			desc:        "Successfully deletes a value in the store",
			op:          "delete",
			key:         []byte("foo"),
			value:       nil,
			fail:        false,
			expectedErr: nil,
		},
		{
			desc:        "Fails to delete a value not in the store",
			op:          "delete",
			key:         []byte("bar"),
			value:       nil,
			fail:        false,
			expectedErr: nil,
		},
		{
			desc:        "Fails to delete a nil key",
			op:          "delete",
			key:         nil,
			value:       nil,
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToDeleteValue,
		},
		{
			desc:        "Fails to delete a value for a key that is too large",
			op:          "delete",
			key:         invalidKey[:],
			value:       nil,
			fail:        true,
			expectedErr: badger.ErrBadgerUnableToDeleteValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := store.ClearAll()
			require.NoError(t, err)
			setupStore(t, store)
			switch tc.op {
			case "set":
				err := store.Set(tc.key, tc.value)
				if tc.fail {
					require.Error(t, err)
					require.ErrorIs(t, err, tc.expectedErr)
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
					require.ErrorIs(t, err, tc.expectedErr)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.value, got)
				}
			case "delete":
				err := store.Delete(tc.key)
				if tc.fail {
					require.Error(t, err)
					require.ErrorIs(t, err, tc.expectedErr)
				} else {
					require.NoError(t, err)
					_, err := store.Get(tc.key)
					require.ErrorIs(t, err, badger.ErrBadgerUnableToGetValue)
				}
			}
		})
	}

	err = store.Stop()
	require.NoError(t, err)
}

func TestBadger_KVStore_GetAllBasic(t *testing.T) {
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

func TestBadger_KVStore_GetAllPrefixed(t *testing.T) {
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

func TestBadger_KVStore_Exists(t *testing.T) {
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
	require.ErrorIs(t, err, badger.ErrBadgerUnableToGetValue)
	require.False(t, exists)

	err = store.Stop()
	require.NoError(t, err)
}

func TestBadger_KVStore_ClearAll(t *testing.T) {
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

func TestBadger_KVStore_BackupAndRestore(t *testing.T) {
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

func TestBadger_KVStore_Len(t *testing.T) {
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

func setupStore(t *testing.T, store badger.KVStore) {
	t.Helper()
	err := store.Set([]byte("foo"), []byte("bar"))
	require.NoError(t, err)
	err = store.Set([]byte("baz"), []byte("bin"))
	require.NoError(t, err)
}
