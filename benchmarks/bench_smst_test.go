//go:build benchmark

package smt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
)

func BenchmarkSparseMerkleSumTrie_Fill(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
	}{
		{
			desc:     "Fill (100000)",
			trieSize: 100000,
			commit:   false,
		},
		{
			desc:     "Fill & Commit (100000)",
			trieSize: 100000,
			commit:   true,
		},
		{
			desc:     "Fill (500000)",
			trieSize: 500000,
			commit:   false,
		},
		{
			desc:     "Fill & Commit (500000)",
			trieSize: 500000,
			commit:   true,
		},
		{
			desc:     "Fill (1000000)",
			trieSize: 1000000,
			commit:   false,
		},
		{
			desc:     "Fill & Commit (1000000)",
			trieSize: 1000000,
			commit:   true,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMST(b, tc.trieSize)
			b.ResetTimer()
			b.StartTimer()
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				for i := 0; i < tc.trieSize; i++ {
					s := strconv.Itoa(i)
					require.NoError(b, trie.Update([]byte(s), []byte(s), uint64(i)))
				}
				if tc.commit {
					require.NoError(b, trie.Commit())
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Update(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMST, uint64) error
	}{
		{
			desc:     "Update (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       updSMST,
		},
		{
			desc:     "Update & Commit (Prefilled: 100000)",
			trieSize: 100000,
			commit:   true,
			fn:       updSMST,
		},
		{
			desc:     "Update (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       updSMST,
		},
		{
			desc:     "Update & Commit (Prefilled: 500000)",
			trieSize: 500000,
			commit:   true,
			fn:       updSMST,
		},
		{
			desc:     "Update (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       updSMST,
		},
		{
			desc:     "Update & Commit (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   true,
			fn:       updSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMST(b, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Get(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMST, uint64) error
	}{
		{
			desc:     "Get (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       getSMST,
		},
		{
			desc:     "Get (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       getSMST,
		},
		{
			desc:     "Get (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       getSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMST(b, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Prove(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMST, uint64) error
	}{
		{
			desc:     "Prove (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       proSMST,
		},
		{
			desc:     "Prove (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       proSMST,
		},
		{
			desc:     "Prove (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       proSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMST(b, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Delete(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMST, uint64) error
	}{
		{
			desc:     "Delete (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       delSMST,
		},
		{
			desc:     "Delete & Commit (Prefilled: 100000)",
			trieSize: 100000,
			commit:   true,
			fn:       delSMST,
		},
		{
			desc:     "Delete (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       delSMST,
		},
		{
			desc:     "Delete & Commit (Prefilled: 500000)",
			trieSize: 500000,
			commit:   true,
			fn:       delSMST,
		},
		{
			desc:     "Delete (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       delSMST,
		},
		{
			desc:     "Delete & Commit (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   true,
			fn:       delSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMST(b, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}
