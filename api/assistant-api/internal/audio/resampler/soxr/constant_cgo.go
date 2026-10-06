//go:build cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resampler_soxr

// Native SOXR uses one worker to keep realtime frame latency predictable.
const (
	nativeSOXRThreadCount    = 1
	minimumOutputSampleCount = 1
	nativeOutputExtraSamples = 1024
)
