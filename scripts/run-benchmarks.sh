#!/bin/bash
# run-benchmarks.sh - Run performance benchmarks and generate reports

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RESULTS_DIR="$PROJECT_ROOT/testdata/perf/results"

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Create results directory
mkdir -p "$RESULTS_DIR"

# Default benchmark options
BENCH_TIME="${BENCH_TIME:-1s}"
BENCH_COUNT="${BENCH_COUNT:-1}"

echo -e "${BLUE}Running shy performance benchmarks...${NC}"
echo ""
echo "Configuration:"
echo "  Benchmark time: $BENCH_TIME"
echo "  Iterations: $BENCH_COUNT"
echo "  Results dir: $RESULTS_DIR"
echo ""

# Check if test databases exist
TEST_DATA_DIR="$PROJECT_ROOT/testdata/perf"
if [ ! -f "$TEST_DATA_DIR/history-medium.db" ]; then
    echo -e "${YELLOW}Warning: Test databases not found. Run scripts/create-test-databases.sh first.${NC}"
    echo ""
fi

# Generate timestamp for this run
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULT_FILE="$RESULTS_DIR/bench-$TIMESTAMP.txt"
MEMPROFILE="$RESULTS_DIR/mem-$TIMESTAMP.prof"
CPUPROFILE="$RESULTS_DIR/cpu-$TIMESTAMP.prof"

# Run benchmarks based on argument
case "${1:-all}" in
"insert")
    echo -e "${BLUE}Benchmarking: InsertCommand${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkInsertCommand -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"like-recent")
    echo -e "${BLUE}Benchmarking: LikeRecent${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkLikeRecent -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"last-command")
    echo -e "${BLUE}Benchmarking: GetRecentCommandsWithoutConsecutiveDuplicates${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkGetRecentCommands -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"list")
    echo -e "${BLUE}Benchmarking: ListCommands${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkListCommands -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"fzf")
    echo -e "${BLUE}Benchmarking: GetCommandsForFzf${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkGetCommandsForFzf -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"fc")
    echo -e "${BLUE}Benchmarking: GetCommandsByRange${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkGetCommandsByRange -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"concurrent")
    echo -e "${BLUE}Benchmarking: Concurrent Inserts${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkConcurrentInserts -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"all")
    echo -e "${BLUE}Running all benchmarks...${NC}"
    cd "$PROJECT_ROOT"
    go test -bench=. -benchtime="$BENCH_TIME" -count="$BENCH_COUNT" \
        -benchmem -memprofile="$MEMPROFILE" -cpuprofile="$CPUPROFILE" \
        ./internal/db | tee "$RESULT_FILE"
    ;;

"quick")
    echo -e "${BLUE}Running quick benchmarks (critical paths only)...${NC}"
    cd "$PROJECT_ROOT"
    go test -bench='BenchmarkInsertCommand|BenchmarkLikeRecent[^W]|BenchmarkGetRecentCommandsWithoutConsecutiveDuplicates' \
        -benchtime=1s -count=3 -benchmem \
        ./internal/db | tee "$RESULT_FILE"
    ;;

*)
    echo "Usage: $0 [insert|like-recent|last-command|list|fzf|fc|concurrent|all|quick]"
    echo ""
    echo "Options:"
    echo "  insert       - Benchmark command insertion"
    echo "  like-recent  - Benchmark prefix search (autosuggestions)"
    echo "  last-command - Benchmark last-command (history navigation)"
    echo "  list         - Benchmark list command"
    echo "  fzf          - Benchmark fzf data source"
    echo "  fc           - Benchmark fc command (range queries)"
    echo "  concurrent   - Benchmark concurrent inserts"
    echo "  all          - Run all benchmarks (default)"
    echo "  quick        - Run only critical path benchmarks"
    echo ""
    echo "Environment variables:"
    echo "  BENCH_TIME   - Time per benchmark (default: 3s)"
    echo "  BENCH_COUNT  - Number of iterations (default: 5)"
    exit 1
    ;;
esac

echo ""
echo -e "${GREEN}✓ Benchmark complete${NC}"
echo "  Results: $RESULT_FILE"
echo "  Memory profile: $MEMPROFILE"
echo "  CPU profile: $CPUPROFILE"
echo ""
echo "To analyze profiles:"
echo "  go tool pprof -http=:8080 $CPUPROFILE"
echo "  go tool pprof -http=:8080 $MEMPROFILE"
echo ""
echo "To compare with previous run:"
echo "  benchstat <old-result-file> $RESULT_FILE"
