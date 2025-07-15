#####################
###   Benchmark   ###
#####################

.PHONY: benchmark_all
benchmark_all:  ## runs all benchmarks
	go test -tags=benchmark -benchmem -run=^$$ -bench Benchmark ./benchmarks -timeout 0

.PHONY: benchmark_smt
benchmark_smt:  ## runs all benchmarks for the SMT
	go test -tags=benchmark -benchmem -run=^$$ -bench=BenchmarkSparseMerkleTrie ./benchmarks -timeout 0

.PHONY: benchmark_smt_fill
benchmark_smt_fill:  ## runs a benchmark on filling the SMT with different amounts of values
	go test -tags=benchmark -benchmem -run=^$$ -bench=BenchmarkSparseMerkleTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smt_ops
benchmark_smt_ops:  ## runs the benchmarks testing different operations on the SMT against different sized tries
	go test -tags=benchmark -benchmem -run=^$$ -bench='BenchmarkSparseMerkleTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_smst
benchmark_smst:  ## runs all benchmarks for the SMST
	go test -tags=benchmark -benchmem -run=^$$ -bench=BenchmarkSparseMerkleSumTrie ./benchmarks -timeout 0

.PHONY: benchmark_smst_fill
benchmark_smst_fill:  ## runs a benchmark on filling the SMST with different amounts of values
	go test -tags=benchmark -benchmem -run=^$$ -bench=BenchmarkSparseMerkleSumTrie_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smst_ops
benchmark_smst_ops:  ## runs the benchmarks test different operations on the SMST against different sized tries
	go test -tags=benchmark -benchmem -run=^$$ -bench='BenchmarkSparseMerkleSumTrie_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_proof_sizes
benchmark_proof_sizes:  ## runs the benchmarks test the proof sizes for different sized tries
	go test -v -tags=benchmark -benchmem -run=^$$ -bench='BenchmarkProofSizes' ./benchmarks -timeout 0

.PHONY: benchmark_resources
benchmark_resources:  ## runs the benchmarks testing different resources
	go test -v -tags=benchmark -benchmem -run=^$$ -bench='BenchmarkResources' ./benchmarks -timeout 0