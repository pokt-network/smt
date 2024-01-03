# smt

[![Tag](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)
[![GoDoc](https://godoc.org/github.com/pokt-network/smt?status.svg)](https://godoc.org/github.com/pokt-network/smt)
![Go Version](https://img.shields.io/github/go-mod/go-version/pokt-network/smt)
[![Go Report Card](https://goreportcard.com/badge/github.com/pokt-network/smt)](https://goreportcard.com/report/github.com/pokt-network/smt)
[![Tests](https://github.com/pokt-network/smt/actions/workflows/test.yml/badge.svg)](https://github.com/pokt-network/smt/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/pokt-network/smt/branch/main/graph/badge.svg)](https://codecov.io/gh/pokt-network/smt)

<!-- toc -->

- [Overview](#overview)
- [Documentation](#documentation)
- [Tests](#tests)
- [Benchmarks](#benchmarks)

<!-- tocstop -->

**NOTE: Requires Go 1.20.12+**

## Overview

This is a Go library that implements a Sparse Merkle Trie for a key-value map.
The trie implements the same optimisations specified in the [Libra whitepaper],
to reduce the number of hash operations required per trie operation to $O(k)$
where $k$ is the number of non-empty elements in the trie. And is implemented
in a similar way to the [JMT whitepaper], with additional features and proof
mechanics.

## Documentation

Documentation for the different aspects of this library, the trie, proofs and
all its different components can be found in the [docs](./docs/) directory.

## Tests

To run all tests (excluding benchmarks) run the following command:

```sh
make test_all
```

To test the `badger` submodule that provides a more fully featured key-value
store run the following command:

```sh
make test_badger
```

## Benchmarks

To run the full suite of benchmarks simply run the following command:

```sh
make benchmark_all
```

To view pre-ran results of the entire benchmarking suite see
[benchmarks](./docs/benchmarks.md)

[jmt whitepaper]: https://developers.diem.com/papers/jellyfish-merkle-tree/2021-01-14.pdf
[libra whitepaper]: https://diem-developers-components.netlify.app/papers/the-diem-blockchain/2020-05-26.pdf
