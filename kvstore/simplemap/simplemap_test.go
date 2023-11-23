package simplemap

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestSimpleMap(t *testing.T) {
	sm := New()
	h := sha256.New()
	var value []byte
	var err error

	h.Write([]byte("test"))

	// Tests for Get.
	_, err = sm.Get(h.Sum(nil))
	if err == nil {
		t.Error("did not return an error when getting a non-existent key")
	}

	// Tests for Put.
	err = sm.Set(h.Sum(nil), []byte("hello"))
	if err != nil {
		t.Error("updating a key returned an error")
	}
	value, err = sm.Get(h.Sum(nil))
	if err != nil {
		t.Error("getting a key returned an error")
	}
	if !bytes.Equal(value, []byte("hello")) {
		t.Error("failed to update key")
	}

	// Tests for Exists.
	exists, err := sm.Exists(h.Sum(nil))
	if err != nil {
		t.Error("failed to check if key exists")
	}
	if !exists {
		t.Error("failed to check if key exists")
	}

	// Tests for Del.
	err = sm.Delete(h.Sum(nil))
	if err != nil {
		t.Error("deleting a key returned an error")
	}
	_, err = sm.Get(h.Sum(nil))
	if err == nil {
		t.Error("failed to delete key")
	}

	err = sm.Delete([]byte("nonexistent"))
	if err != nil {
		t.Error("deleting a nonexistent key returned an error")
	}

	// Tests for ClearAll.
	err = sm.Set(h.Sum(nil), []byte("hello"))
	if err != nil {
		t.Error("updating a key returned an error")
	}
	exists, err = sm.Exists(h.Sum(nil))
	if err != nil {
		t.Error("failed to check if key exists")
	}
	if !exists {
		t.Error("failed to check if key exists")
	}
	err = sm.ClearAll()
	if err != nil {
		t.Error("failed to clear all keys")
	}
	exists, err = sm.Exists(h.Sum(nil))
	if err != nil {
		t.Error("failed to check if key exists")
	}
	if exists {
		t.Error("failed to clear all keys")
	}

	// Tests for GetAll.
	if err := sm.Set([]byte("key1"), []byte("value1")); err != nil {
		t.Error("setting a key returned an error")
	}
	if err := sm.Set([]byte("key2"), []byte("value2")); err != nil {
		t.Error("setting a key returned an error")
	}
	if err := sm.Set([]byte("key3"), []byte("value3")); err != nil {
		t.Error("setting a key returned an error")
	}
	if err := sm.Set([]byte("key4"), []byte("value4")); err != nil {
		t.Error("setting a key returned an error")
	}
	if err := sm.Set([]byte("key5"), []byte("value5")); err != nil {
		t.Error("setting a key returned an error")
	}

	keys, values, err := sm.GetAll([]byte("key"), false)
	if err != nil {
		t.Error("failed to get all keys")
	}
	if len(keys) != 5 {
		t.Error("failed to get all keys, wrong length")
	}

	for i, key := range keys {
		if string(key) != fmt.Sprintf("key%d", i+1) {
			t.Error("failed to get all keys, wrong key")
		}
		if string(values[i]) != fmt.Sprintf("value%d", i+1) {
			t.Error("failed to get all keys, wrong value")
		}
	}

	keys, values, err = sm.GetAll([]byte("key"), true)
	if err != nil {
		t.Error("failed to get all keys")
	}
	if len(keys) != 5 {
		t.Error("failed to get all keys, wrong length")
	}

	for i, key := range keys {
		if string(key) != fmt.Sprintf("key%d", 5-i) {
			t.Error("failed to get all keys, wrong key")
		}
		if string(values[i]) != fmt.Sprintf("value%d", 5-i) {
			t.Error("failed to get all keys, wrong value")
		}
	}
}
