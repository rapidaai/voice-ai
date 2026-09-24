# SOXR Realtime Benchmark

Measured on Apple M1 Pro with Go 1.25.13 on September 8, 2026.

Command:

```bash
go test -run '^$' -bench BenchmarkRealtimeResample20ms -benchmem -count=5 ./api/assistant-api/internal/audio/resampler/soxr
```

The benchmark converts one 20 ms mono PCM16 frame from 8 kHz to 16 kHz.

| Implementation | Time per frame | Bytes per operation | Allocations per operation |
| --- | ---: | ---: | ---: |
| Go polyphase baseline | approximately 15.1 µs | 7,169 | 322 |
| Native SOXR quick quality | approximately 1.6 µs | 656 | 2 |

The realtime path uses native SOXR when CGO is available. A Go implementation
remains as the non-CGO fallback so static analysis and unsupported development
targets continue to compile.
