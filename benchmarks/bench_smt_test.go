package smt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
)

func BenchmarkSparseMerkleTrie_Fill(b *testing.B) {
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
		{
			desc:     "Fill (5000000)",
			trieSize: 5000000,
			commit:   false,
		},
		{
			desc:     "Fill & Commit (5000000)",
			trieSize: 5000000,
			commit:   true,
		},
		{
			desc:     "Fill (10000000)",
			trieSize: 10000000,
			commit:   false,
		},
		{
			desc:     "Fill & Commit (10000000)",
			trieSize: 10000000,
			commit:   true,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMT(b, tc.trieSize)
			b.ResetTimer()
			b.StartTimer()
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				for i := 0; i < tc.trieSize; i++ {
					s := strconv.Itoa(i)
					require.NoError(b, trie.Update([]byte(s), []byte(s)))
				}
				if tc.commit {
					require.NoError(b, trie.Commit())
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkSparseMerkleTrie_Update(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMT, []byte) error
	}{
		{
			desc:     "Update (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       updSMT,
		},
		{
			desc:     "Update & Commit (Prefilled: 100000)",
			trieSize: 100000,
			commit:   true,
			fn:       updSMT,
		},
		{
			desc:     "Update (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       updSMT,
		},
		{
			desc:     "Update & Commit (Prefilled: 500000)",
			trieSize: 500000,
			commit:   true,
			fn:       updSMT,
		},
		{
			desc:     "Update (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       updSMT,
		},
		{
			desc:     "Update & Commit (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   true,
			fn:       updSMT,
		},
		{
			desc:     "Update (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   false,
			fn:       updSMT,
		},
		{
			desc:     "Update & Commit (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   true,
			fn:       updSMT,
		},
		{
			desc:     "Update (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   false,
			fn:       updSMT,
		},
		{
			desc:     "Update & Commit (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   true,
			fn:       updSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMT(b, tc.trieSize)
			benchmarkSMT(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTrie_Get(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMT, []byte) error
	}{
		{
			desc:     "Get (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       getSMT,
		},
		{
			desc:     "Get (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       getSMT,
		},
		{
			desc:     "Get (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       getSMT,
		},
		{
			desc:     "Get (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   false,
			fn:       getSMT,
		},
		{
			desc:     "Get (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   false,
			fn:       getSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.desc, func(b *testing.B) {
			trie := setupSMT(b, tc.trieSize)
			benchmarkSMT(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTrie_Prove(b *testing.B) {
	testCases := []struct {
		desc     string
		trieSize int
		commit   bool
		fn       func(*smt.SMT, []byte) error
	}{
		{
			desc:     "Prove (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       proSMT,
		},
		{
			desc:     "Prove (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       proSMT,
		},
		{
			name:     "Prove (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       proSMT,
		},
		{
			name:     "Prove (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   false,
			fn:       proSMT,
		},
		{
			name:     "Prove (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   false,
			fn:       proSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMT(b, tc.trieSize)
			benchmarkSMT(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTrie_Delete(b *testing.B) {
	testCases := []struct {
		name     string
		trieSize int
		commit   bool
		fn       func(*smt.SMT, []byte) error
	}{
		{
			name:     "Delete (Prefilled: 100000)",
			trieSize: 100000,
			commit:   false,
			fn:       delSMT,
		},
		{
			name:     "Delete & Commit (Prefilled: 100000)",
			trieSize: 100000,
			commit:   true,
			fn:       delSMT,
		},
		{
			name:     "Delete (Prefilled: 500000)",
			trieSize: 500000,
			commit:   false,
			fn:       delSMT,
		},
		{
			name:     "Delete & Commit (Prefilled: 500000)",
			trieSize: 500000,
			commit:   true,
			fn:       delSMT,
		},
		{
			name:     "Delete (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   false,
			fn:       delSMT,
		},
		{
			name:     "Delete & Commit (Prefilled: 1000000)",
			trieSize: 1000000,
			commit:   true,
			fn:       delSMT,
		},
		{
			name:     "Delete (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   false,
			fn:       delSMT,
		},
		{
			name:     "Delete & Commit (Prefilled: 5000000)",
			trieSize: 5000000,
			commit:   true,
			fn:       delSMT,
		},
		{
			name:     "Delete (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   false,
			fn:       delSMT,
		},
		{
			name:     "Delete & Commit (Prefilled: 10000000)",
			trieSize: 10000000,
			commit:   true,
			fn:       delSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMT(b, tc.trieSize)
			benchmarkSMT(b, trie, tc.commit, tc.fn)
		})
	}
}
