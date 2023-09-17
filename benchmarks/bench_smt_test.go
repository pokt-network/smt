package smt

import (
	"strconv"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
)

func BenchmarkSparseMerkleTree_Fill(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
	}{
		{
			name:       "Fill (100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (100000)",
			treeSize:   100000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (500000)",
			treeSize:   500000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (1000000)",
			treeSize:   1000000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (5000000)",
			treeSize:   5000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (5000000)",
			treeSize:   5000000,
			commit:     true,
			persistent: false,
		},
		{
			name:       "Fill (10000000)",
			treeSize:   10000000,
			commit:     false,
			persistent: false,
		},
		{
			name:       "Fill & Commit (10000000)",
			treeSize:   10000000,
			commit:     true,
			persistent: false,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			tree := setupSMT(b, tc.persistent, tc.treeSize)
			b.ResetTimer()
			b.StartTimer()
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				for i := 0; i < tc.treeSize; i++ {
					s := strconv.Itoa(i)
					require.NoError(b, tree.Update([]byte(s), []byte(s)))
				}
				if tc.commit {
					require.NoError(b, tree.Commit())
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkSparseMerkleTree_Update(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMT, []byte) error
	}{
		{
			name:       "Update (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update & Commit (Prefilled: 100000)",
			treeSize:   100000,
			commit:     true,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update & Commit (Prefilled: 500000)",
			treeSize:   500000,
			commit:     true,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update & Commit (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update & Commit (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     true,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         updSMT,
		},
		{
			name:       "Update & Commit (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     true,
			persistent: false,
			fn:         updSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			tree := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, tree, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTree_Get(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMT, []byte) error
	}{
		{
			name:       "Get (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         getSMT,
		},
		{
			name:       "Get (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         getSMT,
		},
		{
			name:       "Get (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         getSMT,
		},
		{
			name:       "Get (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         getSMT,
		},
		{
			name:       "Get (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         getSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			tree := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, tree, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTree_Prove(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMT, []byte) error
	}{
		{
			name:       "Prove (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         proSMT,
		},
		{
			name:       "Prove (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         proSMT,
		},
		{
			name:       "Prove (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         proSMT,
		},
		{
			name:       "Prove (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         proSMT,
		},
		{
			name:       "Prove (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         proSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			tree := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, tree, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleTree_Delete(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMT, []byte) error
	}{
		{
			name:       "Delete (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete & Commit (Prefilled: 100000)",
			treeSize:   100000,
			commit:     true,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete & Commit (Prefilled: 500000)",
			treeSize:   500000,
			commit:     true,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete & Commit (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     false,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete & Commit (Prefilled: 5000000)",
			treeSize:   5000000,
			commit:     true,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     false,
			persistent: false,
			fn:         delSMT,
		},
		{
			name:       "Delete & Commit (Prefilled: 10000000)",
			treeSize:   10000000,
			commit:     true,
			persistent: false,
			fn:         delSMT,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			tree := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, tree, tc.commit, tc.fn)
		})
	}
}
