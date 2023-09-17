package smt

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"strconv"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
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

func setupSMT(b *testing.B, persistent bool, num int) *smt.SMT {
	b.Helper()
	path := ""
	if persistent {
		path = b.TempDir()
	}
	nodes, err := smt.NewKVStore(path)
	require.NoError(b, err)
	tree := smt.NewSparseMerkleTree(nodes, sha256.New())
	for i := 0; i < num; i++ {
		s := strconv.Itoa(i)
		require.NoError(b, tree.Update([]byte(s), []byte(s)))
	}
	require.NoError(b, tree.Commit())
	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
		require.NoError(b, nodes.Stop())
		if path != "" {
			require.NoError(b, os.RemoveAll(path))
		}
	})
	return tree
}

func benchmarkSMT(b *testing.B, tree *smt.SMT, commit bool, fn func(*smt.SMT, []byte) error) {
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		_ = fn(tree, []byte(s))
	}
	if commit {
		require.NoError(b, tree.Commit())
	}
	b.StopTimer()
}

func setupSMST(b *testing.B, persistent bool, num int) *smt.SMST {
	b.Helper()
	path := ""
	if persistent {
		path = b.TempDir()
	}
	nodes, err := smt.NewKVStore(path)
	require.NoError(b, err)
	tree := smt.NewSparseMerkleSumTree(nodes, sha256.New())
	for i := 0; i < num; i++ {
		s := strconv.Itoa(i)
		require.NoError(b, tree.Update([]byte(s), []byte(s), uint64(i)))
	}
	require.NoError(b, tree.Commit())
	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
		require.NoError(b, nodes.Stop())
		if path != "" {
			require.NoError(b, os.RemoveAll(path))
		}
	})
	return tree
}

func benchmarkSMST(b *testing.B, tree *smt.SMST, commit bool, fn func(*smt.SMST, uint64) error) {
	b.ResetTimer()
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fn(tree, uint64(i))
	}
	if commit {
		require.NoError(b, tree.Commit())
	}
	b.StopTimer()
}
