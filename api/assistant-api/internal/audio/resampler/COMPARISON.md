# Audio Resampler Selection

## SOXR

Use `soxr.New(...)` for continuous audio streams. Each logical stream must own
its own resampler because the implementation preserves filter state between
calls.

```go
resampler := soxr.New(
	soxr.WithLogger(logger),
	soxr.WithQuickQuality(),
)
```

Use `WithQuickQuality()` for latency-sensitive speech. The constructor uses
native libsoxr with direct PCM16 buffers for this mode. When CGO or libsoxr is
unavailable, it falls back to the Go polyphase implementation. The constructor
uses the higher-quality Go implementation when no quality option is supplied.

Native development requires `libsoxr` and `pkg-config`. Install `libsoxr` with
Homebrew on macOS or install `libsoxr-dev` on Debian-based systems. The
assistant API production image installs the build and runtime packages.

## Linear

Use `linear.New(...)` for stateless, bounded conversions where preserving
streaming filter state is not required.

```go
resampler := linear.New(
	linear.WithLogger(logger),
)
```

Use `linear.NewConverter(...)` when only byte and float sample conversion is
required.

## Ownership

Do not share a stateful SOXR instance between unrelated audio streams. Create
one instance for each independently ordered stream.
