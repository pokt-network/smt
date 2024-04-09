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

#####################
### Documentation ###
#####################

## Ensure godoc is installed
.PHONY:
# Internal helper target - check if godoc is installed
check_godoc:
	{ \
		if ( ! ( command -v godoc >/dev/null )); then \
		echo "Seems like you don't have godoc installed. Make sure you install it via 'go install golang.org/x/tools/cmd/godoc@latest' before continuing"; \
		exit 1; \
		fi; \
	}

#####################
####   Testing   ####
#####################

.PHONY: test_all
test_all:  ## runs the test suite
	go test -v -p 1 ./... -mod=readonly -race

.PHONY: test_badger
test_badger: ## runs the badger KVStore submodule's test suite
	go test -v -p 1 ./kvstore/badger/... -mod=readonly -race


#####################
###   go helpers  ###
#####################
.PHONY: mod_tidy
mod_tidy: ## runs go mod tidy for all (sub)modules
	go mod tidy
	cd kvstore/simplemap && go mod tidy
	cd kvstore/badger && go mod tidy

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/smt/"
	godoc -http=:6060

#####################
###   Benchmark   ###
#####################

.PHONY: benchmark_all
benchmark_all:  ## runs all benchmarks
	go test -tags=benchmark -benchmem -run=^$ -bench Benchmark ./benchmarks -timeout 0

.PHONY: benchmark_smt
benchmark_smt:  ## runs all benchmarks for the SMT
	go test -tags=benchmark -benchmem -run=^$ -bench=BenchmarkSparseMerkleTrie ./benchmarks -timeout 0

.PHONY: benchmark_smt_fill
benchmark_smt_fill:  ## runs a benchmark on filling the SMT with different amounts of values
	go test -tags=benchmark -benchmem -run=^$ -bench=BenchmarkSparseMerkleTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smt_ops
benchmark_smt_ops:  ## runs the benchmarks testing different operations on the SMT against different sized tries
	go test -tags=benchmark -benchmem -run=^$ -bench='BenchmarkSparseMerkleTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_smst
benchmark_smst:  ## runs all benchmarks for the SMST
	go test -tags=benchmark -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTrie ./benchmarks -timeout 0

.PHONY: benchmark_smst_fill
benchmark_smst_fill:  ## runs a benchmark on filling the SMST with different amounts of values
	go test -tags=benchmark -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smst_ops
benchmark_smst_ops:  ## runs the benchmarks test different operations on the SMST against different sized tries
	go test -tags=benchmark -benchmem -run=^$ -bench='BenchmarkSparseMerkleSumTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_proof_sizes
benchmark_proof_sizes:  ## runs the benchmarks test the proof sizes for different sized tries
	go test -tags=benchmark -v ./benchmarks -run ProofSizes

###########################
###   Release Helpers   ###
###########################

.PHONY: tag_minor_release
tag_minor_release: ## Tag a new minor release (e.g. v1.0.0 -> v1.1.0)
	@$(eval LATEST_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. '{$$2 += 1; $$3 = 2; print $$1 "." $$2 "." $$3}'))
	@git tag $(NEW_TAG)
	@echo "New minor release version: $(NEW_TAG)\nRun `git push origin $(NEW_TAG)` and draft a new release at https://github.com/pokt-network/smt/releases/new"

.PHONY: tag_bug_fix
tag_bug_fix: ## Tag a new bug fix release (e.g. v1.0.1 -> v1.0.2)
	@$(eval LATEST_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. '{$$3 += 1; print $$1 "." $$2 "." $$3}'))
	@git tag $(NEW_TAG)
	@echo "New bug fix version: $(NEW_TAG)\nRun `git push origin $(NEW_TAG)` and draft a new release at https://github.com/pokt-network/smt/releases/new"
