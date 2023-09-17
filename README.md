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

_NOTE: Unless otherwise stated the benchmarks in this document were ran on a 2023 14-inch Macbook Pro M2 Max with 32GB of RAM. The trees tested are using the `sha256.New()` hasher._

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

| Benchmark Name       | Iterations | Time (ns/op)    | Bytes (B/op)    | Allocations (allocs/op) |
| -------------------- | ---------- | --------------- | --------------- | ----------------------- |
| Fill (0.1M)          | 10         | 162,967,196     | 159,479,499     | 2,371,598               |
| Fill & Commit (0.1M) | 10         | 2,877,307,858   | 972,961,486     | 15,992,605              |
| Fill (0.5M)          | 10         | 926,864,771     | 890,408,326     | 13,021,258              |
| Fill & Commit (0.5M) | 10         | 16,043,430,012  | 5,640,034,396   | 82,075,720              |
| Fill (1M)            | 10         | 2,033,616,088   | 1,860,523,968   | 27,041,639              |
| Fill & Commit (1M)   | 10         | 32,617,249,642  | 12,655,347,004  | 166,879,661             |
| Fill (5M)            | 10         | 12,502,309,738  | 10,229,139,731  | 146,821,675             |
| Fill & Commit (5M)   | 10         | 175,421,250,979 | 78,981,342,709  | 870,235,579             |
| Fill (10M)           | 10         | 29,718,092,496  | 21,255,245,031  | 303,637,210             |
| Fill & Commit (10M)  | 10         | 396,142,675,962 | 173,053,933,624 | 1,775,304,977           |

#### Operations

In order to run the SMT operation benchmarks use the following command:

```sh
go test -benchmem -bench='BenchmarkSparseMerkleTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0
```

| Benchmark Name                    | Iterations | Time (ns/op) | Bytes (B/op) | Allocations (allocs/op) |
| --------------------------------- | ---------- | ------------ | ------------ | ----------------------- |
| Update (Prefilled: 0.1M)          | 740,618    | 1,350        | 1,753        | 25                      |
| Update & Commit (Prefilled: 0.1M) | 21,022     | 54,665       | 13,110       | 281                     |
| Update (Prefilled: 0.5M)          | 605,348    | 1,682        | 1,957        | 26                      |
| Update & Commit (Prefilled: 0.5M) | 11,697     | 91,028       | 21,501       | 468                     |
| Update (Prefilled: 1M)            | 545,701    | 1,890        | 2,112        | 28                      |
| Update & Commit (Prefilled: 1M)   | 9,540      | 119,347      | 24,983       | 545                     |
| Update (Prefilled: 5M)            | 466,688    | 2,226        | 2,453        | 31                      |
| Update & Commit (Prefilled: 5M)   | 7,906      | 186,026      | 52,621       | 722                     |
| Update (Prefilled: 10M)           | 284,580    | 5,263        | 2,658        | 33                      |
| Update & Commit (Prefilled: 10M)  | 4,484      | 298,376      | 117,923      | 844                     |
| Get (Prefilled: 0.1M)             | 3,923,601  | 303.2        | 48           | 3                       |
| Get (Prefilled: 0.5M)             | 2,209,981  | 577.7        | 48           | 3                       |
| Get (Prefilled: 1M)               | 1,844,431  | 661.6        | 48           | 3                       |
| Get (Prefilled: 5M)               | 1,196,467  | 1,030        | 48           | 3                       |
| Get (Prefilled: 10M)              | 970,195    | 2,667        | 48           | 3                       |
| Prove (Prefilled: 0.1M)           | 829,801    | 1,496        | 2,177        | 17                      |
| Prove (Prefilled: 0.5M)           | 610,402    | 1,835        | 2,747        | 17                      |
| Prove (Prefilled: 1M)             | 605,799    | 1,905        | 2,728        | 17                      |
| Prove (Prefilled: 5M)             | 566,930    | 2,129        | 2,731        | 17                      |
| Prove (Prefilled: 10M)            | 458,472    | 7,113        | 2,735        | 17                      |
| Delete (Prefilled: 0.1M)          | 12,081,112 | 96.18        | 50           | 3                       |
| Delete & Commit (Prefilled: 0.1M) | 26,490     | 39,568       | 7,835        | 177                     |
| Delete (Prefilled: 0.5M)          | 7,253,522  | 140.3        | 64           | 3                       |
| Delete & Commit (Prefilled: 0.5M) | 12,766     | 80,518       | 16,696       | 376                     |
| Delete (Prefilled: 1M)            | 1,624,569  | 629.6        | 196          | 4                       |
| Delete & Commit (Prefilled: 1M)   | 9,811      | 135,606      | 20,254       | 456                     |
| Delete (Prefilled: 5M)            | 856,424    | 1,400        | 443          | 6                       |
| Delete & Commit (Prefilled: 5M)   | 8,431      | 151,107      | 74,133       | 626                     |
| Delete (Prefilled: 10M)           | 545,876    | 4,173        | 556          | 6                       |
| Delete & Commit (Prefilled: 10M)  | 3,916      | 271,332      | 108,396      | 772                     |

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

| Benchmark Name       | Iterations | Time (ns/op)    | Bytes (B/op)    | Allocations (allocs/op) |
| -------------------- | ---------- | --------------- | --------------- | ----------------------- |
| Fill (0.1M)          | 10         | 157,951,888     | 165,878,234     | 2,471,593               |
| Fill & Commit (0.1M) | 10         | 3,011,097,462   | 1,058,069,050   | 16,664,811              |
| Fill (0.5M)          | 10         | 927,521,862     | 922,408,350     | 13,521,259              |
| Fill & Commit (0.5M) | 10         | 15,338,199,979  | 6,533,439,773   | 85,880,046              |
| Fill (1M)            | 10         | 1,982,756,162   | 1,924,516,467   | 28,041,610              |
| Fill & Commit (1M)   | 10         | 31,197,517,821  | 14,874,342,889  | 175,474,251             |
| Fill (5M)            | 10         | 12,054,370,871  | 10,549,075,488  | 151,821,423             |
| Fill & Commit (5M)   | 10         | 176,912,009,238 | 89,667,234,678  | 914,653,740             |
| Fill (10M)           | 10         | 26,859,672,362  | 21,894,837,504  | 313,635,611             |
| Fill & Commit (10M)  | 10         | 490,805,535,617 | 197,997,807,905 | 1,865,882,489           |

#### Operations

In order to run the SMST operation benchmarks use the following command:

```sh
go test -benchmem -bench='BenchmarkSparseMerkleSumTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0
```

| Benchmark Name                    | Iterations | Time (ns/op) | Bytes (B/op) | Allocations (allocs/op) |
| --------------------------------- | ---------- | ------------ | ------------ | ----------------------- |
| Update (Prefilled: 0.1M)          | 913,760    | 1,477        | 1,843        | 25                      |
| Update & Commit (Prefilled: 0.1M) | 20,318     | 49,705       | 13,440       | 256                     |
| Update (Prefilled: 0.5M)          | 687,813    | 1,506        | 1,965        | 27                      |
| Update & Commit (Prefilled: 0.5M) | 14,526     | 83,295       | 37,604       | 428                     |
| Update (Prefilled: 1M)            | 630,310    | 1,679        | 2,076        | 28                      |
| Update & Commit (Prefilled: 1M)   | 11,678     | 122,568      | 25,760       | 501                     |
| Update (Prefilled: 5M)            | 644,193    | 1,850        | 2,378        | 31                      |
| Update & Commit (Prefilled: 5M)   | 6,214      | 184,533      | 60,755       | 723                     |
| Update (Prefilled: 10M)           | 231,714    | 4,962        | 2,616        | 33                      |
| Update & Commit (Prefilled: 10M)  | 4,284      | 279,893      | 77,377       | 830                     |
| Get (Prefilled: 0.1M)             | 3,924,031  | 281.3        | 40           | 2                       |
| Get (Prefilled: 0.5M)             | 2,080,167  | 559.6        | 40           | 2                       |
| Get (Prefilled: 1M)               | 1,609,478  | 718.6        | 40           | 2                       |
| Get (Prefilled: 5M)               | 1,015,630  | 1,105        | 40           | 2                       |
| Get (Prefilled: 10M)              | 352,980    | 2,949        | 40           | 2                       |
| Prove (Prefilled: 0.1M)           | 717,380    | 1,692        | 2,344        | 18                      |
| Prove (Prefilled: 0.5M)           | 618,265    | 1,972        | 3,040        | 19                      |
| Prove (Prefilled: 1M)             | 567,594    | 2,117        | 3,044        | 19                      |
| Prove (Prefilled: 5M)             | 446,062    | 2,289        | 3,045        | 19                      |
| Prove (Prefilled: 10M)            | 122,347    | 11,215       | 3,046        | 19                      |
| Delete (Prefilled: 0.1M)          | 1,000,000  | 1,022        | 1,110        | 7                       |
| Delete & Commit (Prefilled: 0.1M) | 1,000,000  | 1,039        | 1,110        | 7                       |
| Delete (Prefilled: 0.5M)          | 1,046,163  | 1,159        | 1,548        | 7                       |
| Delete & Commit (Prefilled: 0.5M) | 907,071    | 1,143        | 1,548        | 7                       |
| Delete (Prefilled: 1M)            | 852,918    | 1,246        | 1,552        | 8                       |
| Delete & Commit (Prefilled: 1M)   | 807,847    | 1,303        | 1,552        | 8                       |
| Delete (Prefilled: 5M)            | 625,662    | 1,604        | 1,552        | 8                       |
| Delete & Commit (Prefilled: 5M)   | 864,432    | 1,382        | 1,552        | 8                       |
| Delete (Prefilled: 10M)           | 232,544    | 4,618        | 1,552        | 8                       |
| Delete & Commit (Prefilled: 10M)  | 224,767    | 5,048        | 1,552        | 8                       |

### Proofs

To run the tests to average the proof size for numerous prefilled trees use the following command:

```sh
go test -v ./benchmarks -run ProofSizes
```

#### SMT

| Prefilled Size | Average Serialised Proof Size (bytes) | Min (bytes) | Max (bytes) | Average Serialised Compacted Proof Size (bytes) | Min (bytes) | Max (bytes) |
|----------------|---------------------------------------|-------------|-------------|-------------------------------------------------|-------------|-------------|
| 100,000        | 780                                   | 650         | 1310        | 790                                             | 692         | 925         |
| 500,000        | 856                                   | 716         | 1475        | 866                                             | 758         | 1024        |
| 1,000,000      | 890                                   | 716         | 1475        | 900                                             | 758         | 1057        |
| 5,000,000      | 966                                   | 815         | 1739        | 976                                             | 858         | 1156        |
| 10,000,000     | 999                                   | 848         | 1739        | 1010                                            | 891         | 1189        |

#### SMST

| Prefilled Size | Average Serialised Proof Size (bytes) | Min (bytes) | Max (bytes) | Average Serialised Compacted Proof Size (bytes) | Min (bytes) | Max (bytes) |
|----------------|---------------------------------------|-------------|-------------|-------------------------------------------------|-------------|-------------|
| 100,000        | 935                                   | 780         | 1590        | 937                                             | 822         | 1101        |
| 500,000        | 1030                                  | 862         | 1795        | 1032                                            | 904         | 1224        |
| 1,000,000      | 1071                                  | 868         | 1795        | 1073                                            | 910         | 1265        |
| 5,000,000      | 1166                                  | 975         | 2123        | 1169                                            | 1018        | 1388        |
| 10,000,000     | 1207                                  | 1026        | 2123        | 1210                                            | 1059        | 1429        |
