// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser

import "context"

// Denoiser is an interface that defines the contract for audio denoising operations.
// Implementations of this interface are expected to provide methods for removing
// noise from audio data and flushing any internal state.
type Denoiser interface {
	// Denoise takes a slice of float32 audio samples and returns a new slice
	// with noise reduction applied. The input audio data is typically in the
	// range of [-1, 1]. The method should process the input samples and return
	// the denoised version, maintaining the same length as the input.
	//
	Denoise(ctx context.Context, input []byte) ([]byte, float64, error)
	// Flush clears any internal state of the denoiser. This method should be
	// called when processing of a stream of audio data is complete or when
	// switching between different audio streams. It ensures that any buffered
	// data or state information is reset, preparing the denoiser for processing
	// new, unrelated audio data.
	Flush()
}
