package simplemap

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt/kvstore"
)

func TestSimpleMap_Get(t *testing.T) {
	store := NewSimpleMap()
	key := []byte("key1")
	value := []byte("value1")

	// Set a value to retrieve
	require.NoError(t, store.Set(key, value))

	tests := []struct {
		desc        string
		key         []byte
		want        []byte
		expectedErr error
	}{
		{
			desc:        "Get existing key",
			key:         key,
			want:        value,
			expectedErr: nil,
		},
		{
			desc:        "Get non-existing key",
			key:         []byte("nonexistent"),
			want:        nil,
			expectedErr: ErrKVStoreKeyNotFound,
		},
		{
			desc:        "Get with empty key",
			key:         []byte(""),
			want:        nil,
			expectedErr: ErrKVStoreEmptyKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := store.Get(tt.key)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSimpleMap_Set(t *testing.T) {
	store := NewSimpleMap()

	tests := []struct {
		desc        string
		key         []byte
		value       []byte
		expectedErr error
	}{
		{
			desc:        "Set valid key-value",
			key:         []byte("key1"),
			value:       []byte("value1"),
			expectedErr: nil,
		},
		{
			desc:        "Set with empty key",
			key:         []byte(""),
			value:       []byte("value1"),
			expectedErr: ErrKVStoreEmptyKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := store.Set(tt.key, tt.value)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSimpleMap_Delete(t *testing.T) {
	store := NewSimpleMap()
	key := []byte("key1")
	value := []byte("value1")

	// Set a value to delete
	require.NoError(t, store.Set(key, value))

	tests := []struct {
		desc        string
		key         []byte
		expectedErr error
	}{
		{
			desc:        "Delete existing key",
			key:         key,
			expectedErr: nil,
		},
		{
			desc:        "Delete non-existing key",
			key:         []byte("nonexistent"),
			expectedErr: nil,
		},
		{
			desc:        "Delete with empty key",
			key:         []byte(""),
			expectedErr: ErrKVStoreEmptyKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := store.Delete(tt.key)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSimpleMap_Len(t *testing.T) {
	store := NewSimpleMap()

	tests := []struct {
		desc        string
		setup       func(kvstore.MapStore)
		expectedLen int
	}{
		{
			desc:        "Length of empty map",
			setup:       func(store kvstore.MapStore) {},
			expectedLen: 0,
		},
		{
			desc: "Length after adding items",
			setup: func(sm kvstore.MapStore) {
				require.NoError(t, sm.Set([]byte("key1"), []byte("value1")))
				require.NoError(t, sm.Set([]byte("key2"), []byte("value2")))
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.setup(store)
			require.Equal(t, tt.expectedLen, store.Len())
		})
	}
}

func TestSimpleMap_ClearAll(t *testing.T) {
	store := NewSimpleMap()

	// Add some elements
	require.NoError(t, store.Set([]byte("key1"), []byte("value1")))
	require.NoError(t, store.Set([]byte("key2"), []byte("value2")))

	// Clear all elements
	require.NoError(t, store.ClearAll())

	require.Equal(t, 0, store.Len())
}
