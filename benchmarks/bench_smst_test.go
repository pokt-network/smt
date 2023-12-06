package smt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
)

func BenchmarkSparseMerkleSumTrie_Fill(b *testing.B) {
	testCases := []struct {
		name       string
		trieSize   int
		commit     bool
		persistent bool
	}{
		{
			name:       "Fill (100000)",
			trieSize:   100000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (100000)",
			trieSize:   100000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (500000)",
			trieSize:   500000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (500000)",
			trieSize:   500000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (1000000)",
			trieSize:   1000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (1000000)",
			trieSize:   1000000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (5000000)",
			trieSize:   5000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (5000000)",
			trieSize:   5000000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (10000000)",
			trieSize:   10000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (10000000)",
			trieSize:   10000000,
			commit:     true,
			persistent: false,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMST(b, tc.persistent, tc.trieSize)
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
		name       string
		trieSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Update (Prefilled: 100000)",
			trieSize:   100000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 100000)",
			trieSize:   100000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 500000)",
			trieSize:   500000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 500000)",
			trieSize:   500000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMST(b, tc.persistent, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Get(b *testing.B) {
	testCases := []struct {
		name       string
		trieSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Get (Prefilled: 100000)",
			trieSize:   100000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 500000)",
			trieSize:   500000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMST(b, tc.persistent, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Prove(b *testing.B) {
	testCases := []struct {
		name       string
		trieSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Prove (Prefilled: 100000)",
			trieSize:   100000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 500000)",
			trieSize:   500000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMST(b, tc.persistent, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTrie_Delete(b *testing.B) {
	testCases := []struct {
		name       string
		trieSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Delete (Prefilled: 100000)",
			trieSize:   100000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 100000)",
			trieSize:   100000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 500000)",
			trieSize:   500000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 500000)",
			trieSize:   500000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 1000000)",
			trieSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 5000000)",
			trieSize:   5000000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 10000000)",
			trieSize:   10000000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			trie := setupSMST(b, tc.persistent, tc.trieSize)
			benchmarkSMST(b, trie, tc.commit, tc.fn)
		})
	}
}
