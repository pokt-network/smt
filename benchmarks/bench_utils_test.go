package smt

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
)

var (
	updSMT = func(s *smt.SMT, b []byte) error {
		return s.Update(b, b)
	}
	getSMT = func(s *smt.SMT, b []byte) error {
		_, err := s.Get(b)
		return err
	}
	proSMT = func(s *smt.SMT, b []byte) error {
		_, err := s.Prove(b)
		return err
	}
	delSMT = func(s *smt.SMT, b []byte) error {
		return s.Delete(b)
	}

	updSMST = func(s *smt.SMST, i uint64) error {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, i)
		return s.Update(b, b, i)
	}
	getSMST = func(s *smt.SMST, i uint64) error {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, i)
		_, _, err := s.Get(b)
		return err
	}
	proSMST = func(s *smt.SMST, i uint64) error {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, i)
		_, err := s.Prove(b)
		return err
	}
	delSMST = func(s *smt.SMST, i uint64) error {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, i)
		return s.Delete(b)
	}
)

func setupSMT(b *testing.B, persistent bool, numLeaves int) *smt.SMT {
	b.Helper()
	path := ""
	if persistent {
		path = b.TempDir()
	}
	nodes, err := smt.NewKVStore(path)
	require.NoError(b, err)
	trie := smt.NewSparseMerkleTrie(nodes, sha256.New())
	for i := 0; i < numLeaves; i++ {
		s := strconv.Itoa(i)
		require.NoError(b, trie.Update([]byte(s), []byte(s)))
	}
	require.NoError(b, trie.Commit())
	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
		require.NoError(b, nodes.Stop())
		if path != "" {
			require.NoError(b, os.RemoveAll(path))
		}
	})
	return trie
}

func benchmarkSMT(b *testing.B, trie *smt.SMT, commit bool, fn func(*smt.SMT, []byte) error) {
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		_ = fn(trie, []byte(s))
	}
	if commit {
		require.NoError(b, trie.Commit())
	}
	b.StopTimer()
}

func setupSMST(b *testing.B, persistent bool, numLeaves int) *smt.SMST {
	b.Helper()
	path := ""
	if persistent {
		path = b.TempDir()
	}
	nodes, err := smt.NewKVStore(path)
	require.NoError(b, err)
	trie := smt.NewSparseMerkleSumTrie(nodes, sha256.New())
	for i := 0; i < numLeaves; i++ {
		s := strconv.Itoa(i)
		require.NoError(b, trie.Update([]byte(s), []byte(s), uint64(i)))
	}
	require.NoError(b, trie.Commit())
	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
		require.NoError(b, nodes.Stop())
		if path != "" {
			require.NoError(b, os.RemoveAll(path))
		}
	})
	return trie
}

func benchmarkSMST(b *testing.B, trie *smt.SMST, commit bool, fn func(*smt.SMST, uint64) error) {
	b.ResetTimer()
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fn(trie, uint64(i))
	}
	if commit {
		require.NoError(b, trie.Commit())
	}
	b.StopTimer()
}
