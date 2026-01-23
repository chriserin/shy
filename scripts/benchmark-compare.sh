#!/bin/bash
# Compare performance between modernc.org/sqlite and zombiezen.com/go/sqlite implementations

set -e

BENCH_PATTERN="${1:-Benchmark}"
BENCH_TIME="${2:-1s}"

echo "=========================================="
echo "Benchmarking modernc.org/sqlite (DB)"
echo "=========================================="
go test -bench="$BENCH_PATTERN" -benchtime="$BENCH_TIME" -benchmem -run=^$ ./internal/db | tee /tmp/bench-modernc.txt

echo ""
echo "=========================================="
echo "Benchmarking zombiezen.com/go/sqlite (ZDB)"
echo "=========================================="
DB_IMPL=zombiezen go test -bench="$BENCH_PATTERN" -benchtime="$BENCH_TIME" -benchmem -run=^$ ./internal/db | tee /tmp/bench-zombiezen.txt

echo ""
echo "=========================================="
echo "Comparison (use benchstat for detailed analysis)"
echo "=========================================="
echo "To compare results in detail, install benchstat:"
echo "  go install golang.org/x/perf/cmd/benchstat@latest"
echo ""
echo "Then run:"
echo "  benchstat /tmp/bench-modernc.txt /tmp/bench-zombiezen.txt"
