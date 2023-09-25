.SILENT:

.PHONY: test_all
test_all:
	go test -v -p 1 ./ -mod=readonly -race

.PHONY: benchmark_all
bechmark_all:
	go test -benchmem -run=^$ -bench Benchmark ./benchmarks -timeout 0

.PHONY: benchmark_smt
benchmark_smt:
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleTree ./benchmarks -timeout 0

.PHONY: benchmark_smt_fill
benchmark_smt_fill:
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleTree_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smt_ops
benchmark_smt_ops:
	go test -benchmem -run=^$ -bench='BenchmarkSparseMerkleTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: benchmark_smst
benchmark_smst:
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTree ./benchmarks -timeout 0

.PHONY: benchmark_smst_fill
benchmark_smst_fill:
	go test -benchmem -run=^$ -bench=BenchmarkSparseMerkleSumTree_Fill ./benchmarks -timeout 0 -benchtime 10x

.PHONY: benchmark_smst_ops
benchmark_smst_ops:
	go test -benchmem -run=^$ -bench='BenchmarkSparseMerkleSumTree_(Update|Get|Prove|Delete)' ./benchmarks -timeout 0

.PHONY: bechmark_proof_sizes
benchmark_proof_sizes:
	go test -v ./benchmarks -run ProofSizes
