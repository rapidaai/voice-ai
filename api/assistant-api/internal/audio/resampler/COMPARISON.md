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
high quality when no quality option is supplied.

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
