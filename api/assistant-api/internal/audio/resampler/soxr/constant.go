// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resampler_soxr

import resampling "github.com/tphakala/go-audio-resampler"

// Default quality favors fidelity for callers that do not select realtime mode.
const defaultQuality = resampling.QualityHigh

// PCM16 constants define sample storage and conversion boundaries.
const (
	pcm16BytesPerSample = 2
	pcm16Scale          = 32768.0
	pcm16PositiveLimit  = 32767.0
)

// Channel constants identify the supported conversion layouts.
const (
	monoChannelCount   = 1
	stereoChannelCount = 2
)
