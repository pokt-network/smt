SHELL := /bin/bash

.SILENT:

#####################
###    General    ###
#####################

.PHONY: help
.DEFAULT_GOAL := help
help:  ## Prints all the targets in all the Makefiles
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list:  ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

## Ensure godoc is installed
.PHONY: check_godoc
# Internal helper target - check if godoc is installed
check_godoc:
	{ \
	if ( ! ( command -v godoc >/dev/null )); then \
		echo "Seems like you don't have godoc installed. Make sure you install it via 'go install golang.org/x/tools/cmd/godoc@latest' before continuing"; \
		exit 1; \
	fi; \
	}

#####################
### Documentation ###
#####################

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/smt/"
	godoc -http=:6060

#####################
####   Testing   ####
#####################

.PHONY: test_all
test_all:  ## runs the test suite
	go test -v -p 1 ./ -mod=readonly -race

#####################
###   Benchmark   ###
#####################

.PHONY: benchmark_all
bechmark_all:  ## runs all benchmarks
	go test -benchmem -run=^$ -bench Benchmark ./benchmarks -timeout 0

.PHONY: benchmark_smt
benchmark_smt:  ## runs all benchmarks for the SMT
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleTrie ./benchmarks -timeout 0

.PHONY: benchmark_smt_fill
benchmark_smt_fill:  ## runs a benchmark on filling the SMT with different amounts of values
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smt_ops
benchmark_smt_ops:  ## runs the benchmarks testing different operations on the SMT against different sized tries
	go test -benchmem -run=^$ -bench='BenchmarkSparseMerkleTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_smst
benchmark_smst:  ## runs all benchmarks for the SMST
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTrie ./benchmarks -timeout 0

.PHONY: benchmark_smst_fill
benchmark_smst_fill:  ## runs a benchmark on filling the SMST with different amounts of values
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smst_ops
benchmark_smst_ops:  ## runs the benchmarks testing different operations on the SMST against different sized tries
	go test -benchmem -run=^$ -bench='BenchmarkSparseMerkleSumTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: bechmark_proof_sizes
benchmark_proof_sizes:  ## runs the benchmarks testing the proof sizes for different sized tries
	go test -v ./benchmarks -run ProofSizes
