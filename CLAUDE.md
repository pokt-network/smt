# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go library implementing Sparse Merkle Tries (SMT) and Sparse Merkle Sum Tries (SMST) with optimizations from the Libra whitepaper. Provides cryptographic data structures for key-value storage with efficient proof generation and verification.

**NOTE: Requires Go 1.20.12+**

## Development Commands

### Testing
```bash
make test_all          # Run all tests excluding benchmarks
make test_badger       # Test badger KVStore submodule
make test_pebble       # Test pebble KVStore submodule
```

### Benchmarking
```bash
make benchmark_all     # Run complete benchmark suite
make benchmark_smt     # SMT-specific benchmarks
make benchmark_smst    # SMST-specific benchmarks
make benchmark_proof_sizes  # Proof size analysis
```

### Go Management
```bash
make mod_tidy          # Run go mod tidy for all submodules
make go_docs           # Start godoc server on localhost:6060
```

### Release Management
```bash
make tag_bug_fix       # Tag new patch release (e.g., v1.0.1 -> v1.0.2)
make tag_minor_release # Tag new minor release (e.g., v1.0.0 -> v1.1.0)
```

## Architecture

### Core Components

**Main Trie Types:**
- **SMT** (`smt.go`) - Sparse Merkle Trie implementation with 4 node types: leaf, inner, extension, and lazy nodes
- **SMST** (`smst.go`) - Sparse Merkle Sum Trie wrapper around SMT with sum tracking capabilities

**Node Types** (defined in `types.go`):
- **Leaf Nodes**: Store full path and value hash with prefix `[]byte{0}`
- **Inner Nodes**: Branch nodes with two non-nil children, prefix `[]byte{1}`
- **Extension Nodes**: Optimize single-child chains, prefix `[]byte{2}`
- **Lazy Nodes**: Uncached persisted nodes loaded on demand

**Key Storage Systems** (`kvstore/`):
- **SimpleMap**: In-memory implementation for testing
- **Badger**: BadgerDB v4 wrapper with persistent storage
- **Pebble**: PebbleDB implementation for production use

### Key Patterns

**Lazy Loading**: Nodes are cached and only read from/written to storage on `Commit()` calls for performance optimization.

**Sum Encoding**: SMST encodes sums as 8-byte binary values appended to digests, enabling efficient total sum calculations.

**Proof Systems**:
- Standard inclusion/exclusion proofs via `Prove(key []byte)`
- Novel closest proof mechanism via `ProveClosest()` for commit-and-reveal schemes
- Proof compression available for efficient storage

### File Organization

- **Core files**: `smt.go`, `smst.go`, `types.go`, `hasher.go`, `proofs.go`
- **Node implementations**: `leaf_node.go`, `inner_node.go`, `extension_node.go`, `lazy_node.go`
- **Storage backends**: `kvstore/badger/`, `kvstore/pebble/`, `kvstore/simplemap/`
- **Documentation**: `docs/smt.md`, `docs/merkle-sum-trie.md`, `docs/benchmarks.md`
- **Testing**: `*_test.go` files, `benchmarks/` directory

## Development Workflow

1. **Running Tests**: Use `make test_all` for comprehensive testing across all components
2. **Benchmarking**: Run `make benchmark_all` to evaluate performance characteristics
3. **Documentation**: Local docs available via `make go_docs` at http://localhost:6060
4. **Modules**: Run `make mod_tidy` to sync dependencies across all submodules

## Release Process

Tag releases with semantic versioning using `make tag_bug_fix` or `make tag_minor_release`, then push tags and create GitHub releases manually.