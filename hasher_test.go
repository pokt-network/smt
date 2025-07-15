package smt

import (
	"crypto/sha256"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTrieHasher_ConcurrentAccess tests that trieHasher.digestData() is safe for concurrent access.
func TestTrieHasher_ConcurrentAccess(t *testing.T) {
	// Create a trie hasher instance
	hasher := NewTrieHasher(sha256.New())

	// Number of goroutines to run concurrently
	numGoroutines := 10
	// Number of operations per goroutine
	operationsPerGoroutine := 100

	// Use a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect any failures
	failureChan := make(chan string, numGoroutines)

	// Start multiple goroutines that concurrently call digestData
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Perform multiple hashing operations
			for j := 0; j < operationsPerGoroutine; j++ {
				// Create unique data for each operation
				data := []byte("test-data-" + string(rune(goroutineID)) + "-" + string(rune(j)))

				// This call should be safe for concurrent access
				digest := hasher.digestData(data)

				// Verify the digest is not empty
				if len(digest) == 0 {
					failureChan <- "digest is empty"
					return
				}

				// Verify the digest has the expected length
				if len(digest) != sha256.Size {
					failureChan <- "digest has wrong length"
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(failureChan)

	// Check if any failures occurred
	for failure := range failureChan {
		t.Errorf("Concurrent access test failed: %s", failure)
	}
}

// TestTrieHasher_ConcurrentAccessWithRaceDetection runs the concurrent test with race detection.
func TestTrieHasher_ConcurrentAccessWithRaceDetection(t *testing.T) {
	// Skip this test if race detection is not enabled
	if !testing.Short() {
		t.Skip("This test is designed to be run with -race flag")
	}

	// Force more aggressive scheduling to increase chance of race detection
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create a trie hasher instance
	hasher := NewTrieHasher(sha256.New())

	// Number of goroutines to run concurrently (higher number to stress test)
	numGoroutines := 50
	// Number of operations per goroutine
	operationsPerGoroutine := 200

	// Use a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines that concurrently call digestData
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Perform multiple hashing operations
			for j := 0; j < operationsPerGoroutine; j++ {
				// Create unique data for each operation
				data := []byte("race-test-data-" + string(rune(goroutineID)) + "-" + string(rune(j)))

				// This call should be safe for concurrent access
				digest := hasher.digestData(data)

				// Verify the digest is valid
				require.NotEmpty(t, digest)
				require.Equal(t, sha256.Size, len(digest))

				// Yield to scheduler to increase chance of race conditions
				runtime.Gosched()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// TestTrieHasher_DigestConsistency verifies that the same input always produces the same output
// across multiple concurrent calls.
func TestTrieHasher_DigestConsistency(t *testing.T) {
	hasher := NewTrieHasher(sha256.New())
	testData := []byte("consistent-test-data")

	// Hash the data once to get the expected result
	expectedDigest := hasher.digestData(testData)

	// Number of goroutines to run concurrently
	numGoroutines := 20
	// Number of operations per goroutine
	operationsPerGoroutine := 50

	// Use a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect results
	resultChan := make(chan []byte, numGoroutines*operationsPerGoroutine)

	// Start multiple goroutines that concurrently call digestData with the same input
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				digest := hasher.digestData(testData)
				resultChan <- digest
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultChan)

	// Verify all results are consistent
	for digest := range resultChan {
		require.Equal(t, expectedDigest, digest, "Digest should be consistent across concurrent calls")
	}
}
