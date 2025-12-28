// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser_rnnoise

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L./models -lrnnoise
#include <rnnoise.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

var frameSize int

func init() {
	frameSize = int(C.rnnoise_get_frame_size())
}

// RNNoise wraps the RNNoise C library with thread-safe processing
type RNNoise struct {
	mu           sync.Mutex
	denoiseState *C.DenoiseState
	frameCount   int
}

// NewRNNoise creates a new RNNoise instance
func NewRNNoise() (*RNNoise, error) {
	state := C.rnnoise_create(nil)
	if state == nil {
		return nil, fmt.Errorf("failed to create rnnoise state")
	}

	return &RNNoise{
		denoiseState: state,
		frameCount:   0,
	}, nil
}

// SuppressNoise processes a single frame of audio and returns confidence score and cleaned audio
// Input must be exactly frameSize samples (typically 480 at 48kHz)
// Audio must be at 48kHz sample rate for proper noise suppression
func (st *RNNoise) SuppressNoise(input []float32) (float64, []float32, error) {
	if st.denoiseState == nil {
		return 0, nil, fmt.Errorf("rnnoise state is not initialized")
	}

	if len(input) != frameSize {
		return 0, nil, fmt.Errorf("input must be exactly %d samples, got %d", frameSize, len(input))
	}

	output := make([]float32, frameSize)
	copy(output, input) // Copy input to output for in-place processing

	st.mu.Lock()
	defer st.mu.Unlock()

	// Process frame - rnnoise_process_frame modifies the output buffer in-place
	// and returns the VAD probability (0.0 = noise, 1.0 = speech)
	inputPtr := (*C.float)(unsafe.Pointer(&input[0]))
	outputPtr := (*C.float)(unsafe.Pointer(&output[0]))

	vad := C.rnnoise_process_frame(st.denoiseState, outputPtr, inputPtr)

	st.frameCount++

	return float64(vad), output, nil
}

// ProcessAudio processes multiple frames and returns combined confidence and cleaned audio
func (st *RNNoise) ProcessAudio(input []float32) (float64, []float32, error) {
	if st.denoiseState == nil {
		return 0, nil, fmt.Errorf("rnnoise state is not initialized")
	}

	if len(input) == 0 {
		return 0, nil, fmt.Errorf("input audio is empty")
	}

	// Pre-allocate output buffer
	frameCount := (len(input) + frameSize - 1) / frameSize
	cleanedAudio := make([]float32, 0, frameCount*frameSize)
	var maxConfidence float64

	st.mu.Lock()
	defer st.mu.Unlock()

	for i := 0; i < len(input); i += frameSize {
		end := i + frameSize
		if end > len(input) {
			end = len(input)
		}

		// Extract chunk
		chunk := input[i:end]

		// Create padded buffer if necessary
		processBuffer := make([]float32, frameSize)
		copy(processBuffer, chunk)
		// Rest is zeros (padding)

		// Process frame
		output := make([]float32, frameSize)
		copy(output, processBuffer)

		inputPtr := (*C.float)(unsafe.Pointer(&processBuffer[0]))
		outputPtr := (*C.float)(unsafe.Pointer(&output[0]))

		vad := C.rnnoise_process_frame(st.denoiseState, outputPtr, inputPtr)
		confidence := float64(vad)

		if confidence > maxConfidence {
			maxConfidence = confidence
		}

		// Append to result (only original length, not padding)
		if i+frameSize > len(input) {
			// Last frame - only append the original samples
			cleanedAudio = append(cleanedAudio, output[:len(input)-i]...)
		} else {
			cleanedAudio = append(cleanedAudio, output...)
		}

		st.frameCount++
	}

	return maxConfidence, cleanedAudio, nil
}

// Close cleans up resources
func (st *RNNoise) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.denoiseState == nil {
		return fmt.Errorf("double-free attempt")
	}

	C.rnnoise_destroy(st.denoiseState)
	st.denoiseState = nil

	return nil
}
