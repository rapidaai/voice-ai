# Benchmark Quick Reference

## Most Common Commands

### Run all benchmarks with memory stats

```bash
go test -bench=. -benchmem ./api/assistant-api/internal/audio/resampler/soxr
```

### Run specific benchmark

```bash
go test -bench=BenchmarkResample ./api/assistant-api/internal/audio/resampler/soxr
```

### Profile CPU usage

```bash
go test -bench=BenchmarkResample -cpuprofile=cpu.prof ./api/assistant-api/internal/audio/resampler/soxr
go tool pprof -http=:8080 cpu.prof
```

### Compare before/after performance

```bash
go test -bench=. -benchmem -count=10 ./api/assistant-api/internal/audio/resampler/soxr > before.txt
# (make your changes)
go test -bench=. -benchmem -count=10 ./api/assistant-api/internal/audio/resampler/soxr > after.txt
benchstat before.txt after.txt
```

## Benchmark Categories

### Sequential Baseline

```bash
go test -bench=BenchmarkResampleSequential ./api/assistant-api/internal/audio/resampler/soxr
```

### Parallel Scaling

```bash
go test -bench=BenchmarkResampleParallel2Cores ./api/assistant-api/internal/audio/resampler/soxr
go test -bench=BenchmarkResampleParallel4Cores ./api/assistant-api/internal/audio/resampler/soxr
go test -bench=BenchmarkResampleParallel8Cores ./api/assistant-api/internal/audio/resampler/soxr
go test -bench=BenchmarkResampleParallel16Cores ./api/assistant-api/internal/audio/resampler/soxr
```

### Data Size Impact

```bash
go test -bench=BenchmarkSmallDataParallel ./api/assistant-api/internal/audio/resampler/soxr   # 10k samples
go test -bench=BenchmarkMediumDataParallel ./api/assistant-api/internal/audio/resampler/soxr  # 500k samples
go test -bench=BenchmarkLargeDataParallel ./api/assistant-api/internal/audio/resampler/soxr   # 1M samples
```

### Stress Testing

```bash
go test -bench=BenchmarkHighConcurrencyResampling ./api/assistant-api/internal/audio/resampler/soxr  # 100 goroutines
go test -bench=BenchmarkStressTest ./api/assistant-api/internal/audio/resampler/soxr                 # 256 goroutines
```

## Command Options Cheat Sheet

| Flag               | Description             | Example                                 |
| ------------------ | ----------------------- | --------------------------------------- |
| `-bench=.`         | Run all benchmarks      | `go test -bench=.`                      |
| `-bench=Name`      | Run specific benchmark  | `go test -bench=BenchmarkResample`      |
| `-benchmem`        | Show memory allocations | `go test -bench=. -benchmem`            |
| `-benchtime=Xs`    | Run for X seconds       | `go test -bench=. -benchtime=10s`       |
| `-benchtime=Nx`    | Run N iterations        | `go test -bench=. -benchtime=1000x`     |
| `-count=N`         | Repeat N times          | `go test -bench=. -count=5`             |
| `-cpuprofile=file` | CPU profile output      | `go test -bench=. -cpuprofile=cpu.prof` |
| `-memprofile=file` | Memory profile output   | `go test -bench=. -memprofile=mem.prof` |
| `-trace=file`      | Execution trace         | `go test -bench=. -trace=trace.out`     |
| `-v`               | Verbose output          | `go test -bench=. -v`                   |
| `-race`            | Race detection          | `go test -bench=. -race`                |

## Performance Targets

### Expected Performance Ranges

- **Sequential Resampling**: ~600-800 µs per 100k samples
- **Parallel 8-Core**: ~300-400 µs per 100k samples (should see 2x improvement)
- **Float32 Conversion**: ~100-200 µs per 100k samples
- **MuLaw Encoding**: ~150-250 µs per 100k samples

### Scaling Expectations

- 2 cores: ~1.5-1.8x speedup
- 4 cores: ~2.5-3.0x speedup
- 8 cores: ~3.5-4.5x speedup
- 16 cores: ~4.0-6.0x speedup (depends on CPU)

## Quick Profiling Workflow

```bash
# 1. Identify bottleneck
go test -bench=BenchmarkResample -cpuprofile=cpu.prof ./api/assistant-api/internal/audio/resampler/soxr

# 2. View profile in browser
go tool pprof -http=:8080 cpu.prof

# 3. Check memory allocations
go test -bench=BenchmarkResample -memprofile=mem.prof -benchmem ./api/assistant-api/internal/audio/resampler/soxr
go tool pprof -http=:8081 mem.prof

# 4. Look for goroutine issues
go test -bench=BenchmarkConcurrent -trace=trace.out ./api/assistant-api/internal/audio/resampler/soxr
go tool trace trace.out
```

## Interpreting Results

### Benchmark Output Format

```
BenchmarkResample-8    1711   632416 ns/op   145234 B/op   4 allocs/op
                  │     │         │             │            │
                  │     │         │             │            └─ Allocations per operation
                  │     │         │             └────────────── Bytes allocated per operation
                  │     │         └──────────────────────────── Nanoseconds per operation
                  │     └────────────────────────────────────── Number of iterations
                  └──────────────────────────────────────────── CPU cores used
```

### What to Look For

- **ns/op**: Lower is better (time per operation)
- **B/op**: Lower is better (memory per operation)
- **allocs/op**: Lower is better (number of allocations)
- **Iterations**: Higher means more reliable results

## Common Issues and Solutions

### Issue: Inconsistent results

```bash
# Solution: Increase count and benchmark time
go test -bench=. -count=10 -benchtime=5s ./api/assistant-api/internal/audio/resampler/soxr
```

### Issue: Want to see memory details

```bash
# Solution: Use -benchmem flag
go test -bench=. -benchmem ./api/assistant-api/internal/audio/resampler/soxr
```

### Issue: Benchmark too slow

```bash
# Solution: Reduce benchmark time or run specific benchmark
go test -bench=BenchmarkResample -benchtime=1s ./api/assistant-api/internal/audio/resampler/soxr
```

### Issue: Need statistical comparison

```bash
# Solution: Use benchstat
go test -bench=. -count=10 ./api/assistant-api/internal/audio/resampler/soxr > results.txt
benchstat results.txt
```

## Installation Requirements

### Install benchstat for comparison

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## Environment Variables

```bash
# Set CPU count
GOMAXPROCS=4 go test -bench=. ./api/assistant-api/internal/audio/resampler/soxr

# Disable GC during benchmark
GOGC=off go test -bench=. ./api/assistant-api/internal/audio/resampler/soxr
```
