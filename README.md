# smt <!-- omit in toc -->

[![Tag](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)
[![GoDoc](https://godoc.org/github.com/pokt-network/smt?status.svg)](https://godoc.org/github.com/pokt-network/smt)
[![Go Report Card](https://goreportcard.com/badge/github.com/pokt-network/smt)](https://goreportcard.com/report/github.com/pokt-network/smt)
[![Tests](https://github.com/pokt-network/smt/actions/workflows/test.yml/badge.svg)](https://github.com/pokt-network/smt/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/pokt-network/smt/branch/main/graph/badge.svg)](https://codecov.io/gh/pokt-network/smt)

Note: **Requires Go 1.19+**

- [Overview](#overview)
- [Documentation](#documentation)
- [Benchmarks](#benchmarks)
  - [SMT](#smt)
    - [Fill](#fill)
    - [Operations](#operations)
  - [SMST](#smst)
    - [Fill](#fill-1)
    - [Operations](#operations-1)

## Overview

This is a Go library that implements a Sparse Merkle tree for a key-value map. The tree implements the same optimisations specified in the [Libra whitepaper][libra whitepaper], to reduce the number of hash operations required per tree operation to O(k) where k is the number of non-empty elements in the tree.

## Documentation

Documentation for the different aspects of this library can be found in the [docs](./docs/) directory.

[libra whitepaper]: https://diem-developers-components.netlify.app/papers/the-diem-blockchain/2020-05-26.pdf

## Benchmarks

Benchmarks for the different aspects of this SMT library can be found in [benchmarks](./benchmarks/). In order to run the entire benchmarking suite use the following command:

```sh
go test -benchmem -run=^$ -bench Benchmark ./benchmarks -timeout 0
```

_NOTE: Unless otherwise stated the benchmarks in this document were ran on a 2023 14-inch Macbook Pro M2 Max with 32GB of RAM._

### SMT

In order to run the SMT benchmarks use the following command:

```sh
go test -benchmem -bench=BenchmarkSparseMerkleTree ./benchmarks -timeout 0
```

#### Fill

In order to run the SMT filling benchmarks use the following command:

```sh
go test -benchmem -bench=BenchmarkSparseMerkleTree_Fill ./benchmarks -timeout 0 -benchtime 10x
```

#### Operations

In order to run the SMT operation benchmarks use the following command:

```sh
go test -benchmem -bench='BenchmarkSparseMerkleTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0
```

### SMST

In order to run the SMST benchmarks use the following command:

```sh
go test -benchmem -bench=BenchmarkSparseMerkleSumTree ./benchmarks -timeout 0
```

#### Fill

In order to run the SMST filling benchmarks use the following command:

```sh
go test -benchmem -bench=BenchmarkSparseMerkleSumTree_Fill ./benchmarks -timeout 0 -benchtime 10x
```

#### Operations

In order to run the SMST operation benchmarks use the following command:

```sh
go test -benchmem -bench='BenchmarkSparseMerkleSumTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0
```

| Benchmark Name                    | Iterations | Time (ns/op) | Bytes (B/op) | Allocations (allocs/op) |
| --------------------------------- | ---------- | ------------ | ------------ | ----------------------- |
| Update (Prefilled: 0.1M)          | 913760     | 1477         | 1843         | 25                      |
| Update & Commit (Prefilled: 0.1M) | 20318      | 49705        | 13440        | 256                     |
| Update (Prefilled: 0.5M)          | 687813     | 1506         | 1965         | 27                      |
| Update & Commit (Prefilled: 0.5M) | 14526      | 83295        | 37604        | 428                     |
| Update (Prefilled: 1M)            | 630310     | 1679         | 2076         | 28                      |
| Update & Commit (Prefilled: 1M)   | 11678      | 122568       | 25760        | 501                     |
| Update (Prefilled: 5M)            | 644193     | 1850         | 2378         | 31                      |
| Update & Commit (Prefilled: 5M)   | 6214       | 184533       | 60755        | 723                     |
| Update (Prefilled: 10M)           | 231714     | 4962         | 2616         | 33                      |
| Update & Commit (Prefilled: 10M)  | 4284       | 279893       | 77377        | 830                     |
| Get (Prefilled: 0.1M)             | 3924031    | 281.3        | 40           | 2                       |
| Get (Prefilled: 0.5M)             | 2080167    | 559.6        | 40           | 2                       |
| Get (Prefilled: 1M)               | 1609478    | 718.6        | 40           | 2                       |
| Get (Prefilled: 5M)               | 1015630    | 1105         | 40           | 2                       |
| Get (Prefilled: 10M)              | 352980     | 2949         | 40           | 2                       |
| Prove (Prefilled: 0.1M)           | 717380     | 1692         | 2344         | 18                      |
| Prove (Prefilled: 0.5M)           | 618265     | 1972         | 3040         | 19                      |
| Prove (Prefilled: 1M)             | 567594     | 2117         | 3044         | 19                      |
| Prove (Prefilled: 5M)             | 446062     | 2289         | 3045         | 19                      |
| Prove (Prefilled: 10M)            | 122347     | 11215        | 3046         | 19                      |
| Delete (Prefilled: 0.1M)          | 1000000    | 1022         | 1110         | 7                       |
| Delete & Commit (Prefilled: 0.1M) | 1000000    | 1039         | 1110         | 7                       |
| Delete (Prefilled: 0.5M)          | 1046163    | 1159         | 1548         | 7                       |
| Delete & Commit (Prefilled: 0.5M) | 907071     | 1143         | 1548         | 7                       |
| Delete (Prefilled: 1M)            | 852918     | 1246         | 1552         | 8                       |
| Delete & Commit (Prefilled: 1M)   | 807847     | 1303         | 1552         | 8                       |
| Delete (Prefilled: 5M)            | 625662     | 1604         | 1552         | 8                       |
| Delete & Commit (Prefilled: 5M)   | 864432     | 1382         | 1552         | 8                       |
| Delete (Prefilled: 10M)           | 232544     | 4618         | 1552         | 8                       |
| Delete & Commit (Prefilled: 10M)  | 224767     | 5048         | 1552         | 8                       |
