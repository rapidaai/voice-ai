# Linear Audio Resampler

This package provides stateless audio conversion using linear interpolation.

Use `New(WithLogger(logger))` when an `AudioResampler` is required. Use
`NewConverter(WithLogger(logger))` when only sample conversion is required.

Realtime transport pipelines should use the stateful SOXR implementation.
