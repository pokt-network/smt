package smt

import (
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkSparseMerkleTree_Update(b *testing.B) {
	smn, err := NewKVStore("")
	require.NoError(b, err)
	smv, err := NewKVStore("")
	require.NoError(b, err)
	smt := NewSMTWithStorage(smn, smv, sha256.New())

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		_ = smt.Update([]byte(s), []byte(s))
	}

	require.NoError(b, smn.Stop())
	require.NoError(b, smv.Stop())
}

func BenchmarkSparseMerkleTree_Delete(b *testing.B) {
	smn, err := NewKVStore("")
	require.NoError(b, err)
	smv, err := NewKVStore("")
	require.NoError(b, err)
	smt := NewSMTWithStorage(smn, smv, sha256.New())

	for i := 0; i < 100000; i++ {
		s := strconv.Itoa(i)
		_ = smt.Update([]byte(s), []byte(s))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		_ = smt.Delete([]byte(s))
	}

	require.NoError(b, smn.Stop())
	require.NoError(b, smv.Stop())
}
