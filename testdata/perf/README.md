# Performance Testing

This directory contains test databases and benchmark results for performance testing shy.

## Quick Start

```bash
# 1. Create test databases
./scripts/create-test-databases.sh

# 2. Run benchmarks
./scripts/run-benchmarks.sh quick

# 3. View results
cat testdata/perf/results/bench-*.txt
```

## Test Databases

Test databases are generated with realistic command history data:

| Database | Size | Commands | Description |
|----------|------|----------|-------------|
| `history-medium.db` | ~500KB | 10,000 | Medium workload (1-2 weeks of usage) |
| `history-large.db` | ~5MB | 100,000 | Large workload (3-6 months of usage) |
| `history-xlarge.db` | ~50MB | 1,000,000 | Extra large workload (1+ years of heavy usage) |

### Test Data Characteristics

- **Commands**: Realistic mix of git, npm, docker, system commands
- **Directories**: Multiple working directories (~7 different paths)
- **Git Context**: Mix of repos and branches (~4 repos, ~5 branches)
- **Sessions**: Multiple sessions (2 shells × 5 PIDs = 10 sessions)
- **Exit Codes**: 10% failure rate (realistic error distribution)
- **Timestamps**: Sequential with 1-60 second gaps
- **Durations**: Random 100ms to 10s execution times

### Creating Test Databases

Three methods are available, listed from fastest to slowest:

#### Method 1: SQL Script (Fastest - Recommended for xlarge)

```bash
# Generate using raw SQL (fastest method)
./scripts/generate-testdata-sql.sh

# Performance:
# - medium (10k): ~1 second
# - large (100k): ~10 seconds
# - xlarge (1M): ~2-3 minutes
```

To include xlarge, edit `scripts/generate-testdata-sql.sh` and uncomment the xlarge line.

#### Method 2: Go Command (Fast)

```bash
# Generate using Go command
./scripts/create-test-databases.sh

# OR directly:
go run . generate-testdata

# Performance:
# - medium (10k): ~2-3 seconds
# - large (100k): ~20-30 seconds
# - xlarge (1M): ~5-8 minutes
```

To include xlarge, edit `cmd/generate_testdata.go` and uncomment the xlarge line.

#### Method 3: CLI Inserts (Slow - Not Recommended)

The old method using `shy insert` CLI commands one-by-one is very slow and not recommended for large datasets. See the archived code in `scripts/create-test-databases.sh` for reference.

## Benchmarks

### Critical Path Benchmarks

These operations affect user experience directly:

| Benchmark | Operation | Target | Used By |
|-----------|-----------|--------|---------|
| `BenchmarkInsertCommand` | Command insertion | <10ms | Shell hook (every command) |
| `BenchmarkLikeRecent` | Prefix search | <50ms | Autosuggestions |
| `BenchmarkGetRecentCommandsWithoutConsecutiveDuplicates` | Last command | <100ms | History navigation |

### Full Benchmark Suite

| Benchmark | What It Measures |
|-----------|------------------|
| `BenchmarkInsertCommand` | Single command insertion across DB sizes |
| `BenchmarkLikeRecent` | Prefix search without filters |
| `BenchmarkLikeRecentWithFilters` | Prefix search with pwd/session filters |
| `BenchmarkGetRecentCommandsWithoutConsecutiveDuplicates` | Last-command with various limits |
| `BenchmarkGetRecentCommandsWithSession` | Last-command with session+pwd filter |
| `BenchmarkListCommands` | List command with various limits |
| `BenchmarkGetCommandsForFzf` | FZF data source (with deduplication) |
| `BenchmarkGetCommandsByRange` | FC command range queries |
| `BenchmarkConcurrentInserts` | Multiple sessions inserting simultaneously |

### Running Benchmarks

```bash
# Quick benchmarks (critical paths only, fast)
./scripts/run-benchmarks.sh quick

# All benchmarks (comprehensive, ~5-10 minutes)
./scripts/run-benchmarks.sh all

# Specific benchmarks
./scripts/run-benchmarks.sh insert
./scripts/run-benchmarks.sh like-recent
./scripts/run-benchmarks.sh last-command
./scripts/run-benchmarks.sh list
./scripts/run-benchmarks.sh fzf
./scripts/run-benchmarks.sh fc
./scripts/run-benchmarks.sh concurrent

# Custom benchmark duration/iterations
BENCH_TIME=5s BENCH_COUNT=10 ./scripts/run-benchmarks.sh all
```

### Benchmark Options

Environment variables:
- `BENCH_TIME` - Duration per benchmark (default: 3s)
- `BENCH_COUNT` - Number of iterations (default: 5)

Example:
```bash
# Run longer benchmarks for more accurate results
BENCH_TIME=10s BENCH_COUNT=10 ./scripts/run-benchmarks.sh insert
```

## Analyzing Results

### Reading Benchmark Output

```
BenchmarkInsertCommand/medium-8    5000    234567 ns/op    1024 B/op    15 allocs/op
```

- `5000` - Number of iterations completed
- `234567 ns/op` - Average time per operation (234μs)
- `1024 B/op` - Bytes allocated per operation
- `15 allocs/op` - Memory allocations per operation

### Using benchstat

Compare results across runs:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run baseline
./scripts/run-benchmarks.sh all
mv testdata/perf/results/bench-*.txt testdata/perf/results/baseline.txt

# Make optimizations...

# Run again
./scripts/run-benchmarks.sh all

# Compare
benchstat testdata/perf/results/baseline.txt testdata/perf/results/bench-*.txt
```

Output example:
```
name                       old time/op  new time/op  delta
InsertCommand/medium-8     234μs ± 2%   189μs ± 1%  -19.23%  (p=0.000 n=5+5)
LikeRecent/medium-8        1.23ms ± 3%  0.98ms ± 2%  -20.33%  (p=0.000 n=5+5)
```

### Profiling

CPU and memory profiles are automatically generated:

```bash
# View CPU profile in browser
go tool pprof -http=:8080 testdata/perf/results/cpu-*.prof

# View memory profile in browser
go tool pprof -http=:8080 testdata/perf/results/mem-*.prof

# Command-line analysis
go tool pprof -top testdata/perf/results/cpu-*.prof
go tool pprof -top testdata/perf/results/mem-*.prof
```

## Performance Targets

Based on user experience requirements:

| Operation | Target Latency | Rationale |
|-----------|---------------|-----------|
| Insert | <10ms | Called on every shell command; must be imperceptible |
| Like-recent | <50ms | Used for autosuggestions; should feel instant |
| Last-command | <100ms | History navigation; acceptable for interactive use |
| List (20 items) | <200ms | Interactive listing; should feel responsive |
| FZF data | <500ms | Initial load for interactive search |

## Optimization Workflow

1. **Establish baseline**: Run benchmarks and save results
2. **Identify bottleneck**: Use profiling to find hot paths
3. **Optimize**: Make targeted improvements
4. **Verify**: Re-run benchmarks and compare with benchstat
5. **Test**: Ensure functionality still works (run tests)
6. **Iterate**: Repeat for next bottleneck

Example:
```bash
# Baseline
./scripts/run-benchmarks.sh all > testdata/perf/results/baseline.txt

# Profile to identify bottleneck
go tool pprof -http=:8080 testdata/perf/results/cpu-*.prof

# Make changes...

# Verify improvement
./scripts/run-benchmarks.sh all
benchstat testdata/perf/results/baseline.txt testdata/perf/results/bench-*.txt

# Run tests to ensure correctness
go test ./...
```

## Common Optimizations

### Database Level
- **Add indexes**: Most impactful for query performance
- **Prepared statements**: Reduce parsing overhead
- **Connection pooling**: Improve concurrent performance
- **SQLite pragmas**: WAL mode, cache size, mmap

### Query Level
- **Reduce data fetching**: Only SELECT needed columns
- **Push filtering to SQL**: Use WHERE clauses vs Go filtering
- **Use LIMIT effectively**: Don't fetch more than needed
- **Optimize window functions**: Can be slow on large datasets

### Application Level
- **Reduce allocations**: Reuse buffers, avoid unnecessary copies
- **Batch operations**: Group multiple inserts when possible
- **Cache results**: For frequently accessed data
- **Lazy evaluation**: Only compute when needed

## Continuous Performance Monitoring

Save baseline results after optimizations:

```bash
# After verified optimizations
./scripts/run-benchmarks.sh all | tee testdata/perf/results/baseline-$(git rev-parse --short HEAD).txt
```

This allows comparing performance across commits:

```bash
benchstat \
  testdata/perf/results/baseline-abc123.txt \
  testdata/perf/results/baseline-def456.txt
```
