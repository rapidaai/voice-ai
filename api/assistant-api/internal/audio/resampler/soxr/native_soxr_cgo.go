//go:build cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

/*
#cgo pkg-config: soxr
#include <stdlib.h>
#include <soxr.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"
)

type nativePCM16Resampler struct {
	handle            C.soxr_t
	closed            *atomic.Bool
	sourceRate        uint32
	targetRate        uint32
	inputSamples      []int16
	outputSamples     []int16
	hasProcessedFrame bool
}

func newNativePCM16Resampler(sourceRate, targetRate uint32) (*nativePCM16Resampler, error) {
	ioSpec := C.soxr_io_spec(C.SOXR_INT16_I, C.SOXR_INT16_I)
	ioSpec.flags = C.SOXR_NO_DITHER
	var nativeError C.soxr_error_t
	qualitySpec := C.soxr_quality_spec(C.SOXR_QQ, 0)
	runtimeSpec := C.soxr_runtime_spec(nativeSOXRThreadCount)
	nativeHandle := C.soxr_create(
		C.double(sourceRate),
		C.double(targetRate),
		monoChannelCount,
		&nativeError,
		&ioSpec,
		&qualitySpec,
		&runtimeSpec,
	)
	if nativeError != nil {
		return nil, fmt.Errorf("%w: %s", ErrNativeSOXRFailed, C.GoString(nativeError))
	}

	closed := new(atomic.Bool)
	resampler := &nativePCM16Resampler{
		handle:     nativeHandle,
		closed:     closed,
		sourceRate: sourceRate,
		targetRate: targetRate,
	}
	runtime.AddCleanup(resampler, func(handle C.soxr_t) {
		if closed.CompareAndSwap(false, true) {
			C.soxr_delete(handle)
		}
	}, nativeHandle)
	return resampler, nil
}

func (resampler *nativePCM16Resampler) Resample(data []byte) ([]byte, error) {
	if resampler == nil || resampler.handle == nil || resampler.closed.Load() {
		return nil, ErrResamplerClosed
	}
	if len(data)%pcm16BytesPerSample != 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidPCM16InputLength, len(data))
	}
	if len(data) == 0 {
		return []byte{}, nil
	}

	inputSampleCount := len(data) / pcm16BytesPerSample
	if cap(resampler.inputSamples) < inputSampleCount {
		resampler.inputSamples = make([]int16, inputSampleCount)
	} else {
		resampler.inputSamples = resampler.inputSamples[:inputSampleCount]
	}
	for sampleIndex := range resampler.inputSamples {
		// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
		resampler.inputSamples[sampleIndex] = int16(binary.LittleEndian.Uint16(data[sampleIndex*pcm16BytesPerSample:]))
	}

	outputSampleCount := int(
		(uint64(inputSampleCount)*uint64(resampler.targetRate) + uint64(resampler.sourceRate)/2) /
			uint64(resampler.sourceRate),
	)
	if outputSampleCount < minimumOutputSampleCount {
		outputSampleCount = minimumOutputSampleCount
	}
	if cap(resampler.outputSamples) < outputSampleCount {
		resampler.outputSamples = make([]int16, outputSampleCount)
	} else {
		resampler.outputSamples = resampler.outputSamples[:outputSampleCount]
	}

	var inputSamplesConsumed C.size_t
	var outputSamplesProduced C.size_t
	nativeError := C.soxr_process(
		resampler.handle,
		C.soxr_in_t(unsafe.Pointer(unsafe.SliceData(resampler.inputSamples))),
		C.size_t(len(resampler.inputSamples)),
		&inputSamplesConsumed,
		C.soxr_out_t(unsafe.Pointer(unsafe.SliceData(resampler.outputSamples))),
		C.size_t(len(resampler.outputSamples)),
		&outputSamplesProduced,
	)
	if nativeError != nil {
		return nil, fmt.Errorf("%w: %s", ErrNativeSOXRFailed, C.GoString(nativeError))
	}
	// #nosec G115, the native count cannot exceed the provided input length.
	if int(inputSamplesConsumed) != len(resampler.inputSamples) {
		return nil, fmt.Errorf(
			"%w: consumed %d of %d",
			ErrIncompleteSOXRInput,
			inputSamplesConsumed,
			len(resampler.inputSamples),
		)
	}

	// #nosec G115, the native count cannot exceed the provided output length.
	producedSampleCount := int(outputSamplesProduced)
	output := make([]byte, outputSampleCount*pcm16BytesPerSample)
	outputSampleOffset := 0
	if !resampler.hasProcessedFrame && producedSampleCount < outputSampleCount {
		outputSampleOffset = outputSampleCount - producedSampleCount
	}
	for sampleIndex := range producedSampleCount {
		// #nosec G115, PCM16 encoding preserves the signed sample bits.
		binary.LittleEndian.PutUint16(
			output[(sampleIndex+outputSampleOffset)*pcm16BytesPerSample:],
			uint16(resampler.outputSamples[sampleIndex]),
		)
	}
	resampler.hasProcessedFrame = true
	return output, nil
}

func (resampler *nativePCM16Resampler) Close() {
	if resampler == nil || resampler.handle == nil {
		return
	}
	if resampler.closed.CompareAndSwap(false, true) {
		C.soxr_delete(resampler.handle)
	}
	resampler.handle = nil
}
