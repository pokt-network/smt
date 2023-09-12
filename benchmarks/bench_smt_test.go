package smt

import (
	"testing"

	"github.com/pokt-network/smt"
)

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
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, smt, tc.commit, tc.fn)
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
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, smt, tc.commit, tc.fn)
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
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, smt, tc.commit, tc.fn)
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
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMT(b, tc.persistent, tc.treeSize)
			benchmarkSMT(b, smt, tc.commit, tc.fn)
		})
	}
}
