//go:build benchmark

package smt

import (
	"crypto/sha256"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

// BenchmarkResources_CreationPatterns tests CPU and memory usage for operations that create different node types
func BenchmarkResources_CreationPatterns(b *testing.B) {
	scenarios := []struct {
		name      string
		valueSize int
	}{
		{"SmallValues_32B", 32},    // Creates ~65 byte nodes (under threshold)
		{"MediumValues_96B", 96},   // Creates ~129 byte nodes (over threshold)
		{"LargeValues_224B", 224},  // Creates ~257 byte nodes (uses buffer pool)
		{"XLargeValues_480B", 480}, // Creates ~513 byte nodes (buffer pool)
	}

	for _, scenario := range scenarios {
		b.Run("LeafCreation_"+scenario.name, func(b *testing.B) {
			nodes := simplemap.NewSimpleMap()
			trie := smt.NewSparseMerkleTrie(nodes, sha256.New())
			value := make([]byte, scenario.valueSize)

			var memBefore, memAfter runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&memBefore)

			b.ReportAllocs()
			b.ResetTimer()

			start := time.Now()
			for i := 0; i < b.N; i++ {
				key := strconv.Itoa(i)
				require.NoError(b, trie.Update([]byte(key), value))
			}
			elapsed := time.Since(start)

			runtime.GC()
			runtime.ReadMemStats(&memAfter)

			// Custom metrics for analysis
			b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N), "ns/op_cpu")
			b.ReportMetric(float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/float64(b.N), "bytes/op_allocated")
			b.ReportMetric(float64(scenario.valueSize), "value_size_bytes")

			b.Cleanup(func() {
				require.NoError(b, nodes.ClearAll())
			})
		})
	}
}

// BenchmarkResources_MemoryPressure tests memory allocation patterns under different buffer pool usage
func BenchmarkResources_MemoryPressure(b *testing.B) {
	scenarios := []struct {
		name      string
		numOps    int
		valueSize int
	}{
		{"SmallValues_1K", 1000, 32},    // ~65 byte nodes (direct allocation)
		{"MediumValues_1K", 1000, 96},   // ~129 byte nodes (buffer pool)
		{"LargeValues_1K", 1000, 224},   // ~257 byte nodes (buffer pool)
		{"SmallValues_10K", 10000, 32},  // Scale test with small nodes
		{"MediumValues_10K", 10000, 96}, // Scale test with medium nodes
		{"LargeValues_10K", 10000, 224}, // Scale test with large nodes
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			var memBefore, memAfter runtime.MemStats
			value := make([]byte, scenario.valueSize)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				nodes := simplemap.NewSimpleMap()
				trie := smt.NewSparseMerkleTrie(nodes, sha256.New())

				runtime.GC()
				runtime.ReadMemStats(&memBefore)

				// Simulate heavy node creation workload
				for j := 0; j < scenario.numOps; j++ {
					key := strconv.Itoa(j)
					require.NoError(b, trie.Update([]byte(key), value))
				}
				require.NoError(b, trie.Commit())

				runtime.GC()
				runtime.ReadMemStats(&memAfter)

				b.ReportMetric(float64(memAfter.TotalAlloc-memBefore.TotalAlloc), "bytes_allocated")
				b.ReportMetric(float64(memAfter.Mallocs-memBefore.Mallocs), "malloc_count")
				b.ReportMetric(float64(scenario.valueSize), "node_size_approx")

				require.NoError(b, nodes.ClearAll())
			}
		})
	}
}

// BenchmarkResources_Trie tests overall trie operations with resource monitoring
func BenchmarkResources_Trie(b *testing.B) {
	nodes := simplemap.NewSimpleMap()
	trie := smt.NewSparseMerkleTrie(nodes, sha256.New())

	// Pre-populate with some data
	for range 1000 {
		key := make([]byte, 32)
		value := make([]byte, 64)
		require.NoError(b, trie.Update(key, value))
	}
	require.NoError(b, trie.Commit())

	b.Run("Mixed_Operations", func(b *testing.B) {
		var cpuStart, cpuEnd time.Time
		var memBefore, memAfter runtime.MemStats

		b.ReportAllocs()

		runtime.GC()
		runtime.ReadMemStats(&memBefore)
		cpuStart = time.Now()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := make([]byte, 32)
			value := make([]byte, 64)

			// Mix of operations that exercise different node types
			_ = trie.Update(key, value) // May create leaf/inner/extension nodes
			_, _ = trie.Get(key)        // Tree traversal
			_, _ = trie.Prove(key)      // Proof generation

			if i%100 == 0 {
				_ = trie.Commit() // Periodic commits
			}
		}
		b.StopTimer()

		cpuEnd = time.Now()
		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		// Custom metrics
		b.ReportMetric(float64(cpuEnd.Sub(cpuStart).Nanoseconds())/float64(b.N), "ns/op_total_cpu")
		b.ReportMetric(float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/float64(b.N), "bytes/op_allocated")
		b.ReportMetric(float64(memAfter.Mallocs-memBefore.Mallocs)/float64(b.N), "mallocs/op")
	})

	b.Cleanup(func() {
		require.NoError(b, nodes.ClearAll())
	})
}

// BenchmarkResources_Contention tests buffer pool performance under concurrent access
func BenchmarkResources_Contention(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, numGoroutines := range concurrencyLevels {
		b.Run("Goroutines_"+strconv.Itoa(numGoroutines), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				// Each goroutine gets its own trie and hasher to avoid thread safety issues
				nodes := simplemap.NewSimpleMap()
				trie := smt.NewSparseMerkleTrie(nodes, sha256.New())
				
				for pb.Next() {
					key := make([]byte, 32)
					value := make([]byte, 128) // Size that uses buffer pool
					_ = trie.Update(key, value)
				}
				
				// Clean up per-goroutine resources
				require.NoError(b, nodes.ClearAll())
			})
		})
	}
}
