package smt

import (
	"testing"

	"github.com/pokt-network/smt"
)

func BenchmarkSparseMerkleSumTree_Update(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Update (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 100000)",
			treeSize:   100000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 500000)",
			treeSize:   500000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         updSMST,
		},
		{
			name:       "Update & Commit (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         updSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMST(b, tc.persistent, tc.treeSize)
			benchmarkSMST(b, smt, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTree_Get(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Get (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
		{
			name:       "Get (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         getSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMST(b, tc.persistent, tc.treeSize)
			benchmarkSMST(b, smt, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTree_Prove(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Prove (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
		{
			name:       "Prove (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         proSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMST(b, tc.persistent, tc.treeSize)
			benchmarkSMST(b, smt, tc.commit, tc.fn)
		})
	}
}

func BenchmarkSparseMerkleSumTree_Delete(b *testing.B) {
	testCases := []struct {
		name       string
		treeSize   int
		commit     bool
		persistent bool
		fn         func(*smt.SMST, uint64) error
	}{
		{
			name:       "Delete (Prefilled: 100000)",
			treeSize:   100000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 100000)",
			treeSize:   100000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 500000)",
			treeSize:   500000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 500000)",
			treeSize:   500000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     false,
			persistent: false,
			fn:         delSMST,
		},
		{
			name:       "Delete & Commit (Prefilled: 1000000)",
			treeSize:   1000000,
			commit:     true,
			persistent: false,
			fn:         delSMST,
		},
	}

	for _, tc := range testCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			smt := setupSMST(b, tc.persistent, tc.treeSize)
			benchmarkSMST(b, smt, tc.commit, tc.fn)
		})
	}
}
