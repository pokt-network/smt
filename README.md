# smt <!-- omit in toc -->

[![Tag](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)](https://img.shields.io/github/v/tag/pokt-network/smt.svg?sort=semver)
[![GoDoc](https://godoc.org/github.com/pokt-network/smt?status.svg)](https://godoc.org/github.com/pokt-network/smt)
[![Go Report Card](https://goreportcard.com/badge/github.com/pokt-network/smt)](https://goreportcard.com/report/github.com/pokt-network/smt)
[![Tests](https://github.com/pokt-network/smt/actions/workflows/test.yml/badge.svg)](https://github.com/pokt-network/smt/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/pokt-network/smt/branch/main/graph/badge.svg)](https://codecov.io/gh/pokt-network/smt)

Note: **Requires Go 1.19+**

- [Overview](#overview)
- [Documentation](#documentation)

## Overview

This is a Go library that implements a Sparse Merkle tree for a key-value map. The tree implements the same optimisations specified in the [Libra whitepaper][libra whitepaper], to reduce the number of hash operations required per tree operation to O(k) where k is the number of non-empty elements in the tree.

## Documentation

Documentation for the different aspects of this library can be found in the [docs](./docs/) directory.

[libra whitepaper]: https://diem-developers-components.netlify.app/papers/the-diem-blockchain/2020-05-26.pdf
