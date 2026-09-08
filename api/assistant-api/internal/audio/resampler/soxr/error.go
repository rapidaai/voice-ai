// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resampler_soxr

import "errors"

var (
	ErrResamplerClosed               = errors.New("resampler is closed")
	ErrAudioConfigRequired           = errors.New("source and target configs are required")
	ErrResamplingFailed              = errors.New("resample failed")
	ErrResamplerInitializationFailed = errors.New("resampler init failed")
	ErrUnsupportedInputFormat        = errors.New("unsupported input format")
	ErrUnsupportedOutputFormat       = errors.New("unsupported output format")
	ErrNativeSOXRUnavailable         = errors.New("native soxr requires cgo")
	ErrNativeSOXRFailed              = errors.New("native soxr failed")
	ErrInvalidPCM16InputLength       = errors.New("pcm16 input length must be even")
	ErrInvalidPCM16ChannelFrame      = errors.New("pcm16 data does not contain complete channel frames")
	ErrUnsupportedChannelConversion  = errors.New("unsupported channel conversion")
	ErrIncompleteSOXRInput           = errors.New("soxr did not consume all input samples")
)
