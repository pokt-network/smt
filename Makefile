SHELL := /bin/sh

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
	go test -v -p 1 -count=1 ./... -mod=readonly -race

.PHONY: test_badger
test_badger: ## runs the badger KVStore submodule's test suite
	go test -v -p 1 -count=1 ./kvstore/badger/... -mod=readonly -race

.PHONY: test_pebble
test_pebble: ## runs the pebble KVStore submodule's test suite
	go test -v -p 1 -count=1 ./kvstore/pebble/... -mod=readonly -race


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

.PHONY: go_lint
go_lint: ## Run all go linters
	golangci-lint run --timeout 5m --build-tags test

###############
### Imports ###
###############

include ./makefiles/colors.mk
include ./makefiles/release.mk
include ./makefiles/benchmarks.mk