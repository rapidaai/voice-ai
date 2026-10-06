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
	handle               C.soxr_t
	closed               *atomic.Bool
	sourceRate           uint32
	targetRate           uint32
	frameBuffered        bool
	maxInputSampleCount  int
	outputSamples        []int16
	pendingOutputSamples []int16
}

func newNativePCM16Resampler(sourceRate, targetRate uint32, quality nativeSOXRQuality) (*nativePCM16Resampler, error) {
	ioSpec := C.soxr_io_spec(C.SOXR_INT16_I, C.SOXR_INT16_I)
	ioSpec.flags = C.SOXR_NO_DITHER
	var nativeError C.soxr_error_t
	qualityRecipe := C.ulong(C.SOXR_QQ)
	if quality == nativeSOXRQualityLiveKit {
		qualityRecipe = C.ulong(C.SOXR_LQ)
	}
	qualitySpec := C.soxr_quality_spec(qualityRecipe, 0)
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
		message := C.GoString(nativeError)
		C.free(unsafe.Pointer(nativeError))
		return nil, fmt.Errorf("%w: %s", ErrNativeSOXRFailed, message)
	}

	closed := new(atomic.Bool)
	resampler := &nativePCM16Resampler{
		handle:        nativeHandle,
		closed:        closed,
		sourceRate:    sourceRate,
		targetRate:    targetRate,
		frameBuffered: quality == nativeSOXRQualityLiveKit,
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
	if inputSampleCount > resampler.maxInputSampleCount {
		resampler.maxInputSampleCount = inputSampleCount
	}
	producedSamples, err := resampler.process(data, false)
	if err != nil {
		return nil, err
	}
	if !resampler.frameBuffered {
		return encodePCM16Samples(producedSamples), nil
	}

	resampler.pendingOutputSamples = append(resampler.pendingOutputSamples, producedSamples...)
	return resampler.drainCompleteOutputFrames(resampler.outputFrameSampleCount(inputSampleCount)), nil
}

func (resampler *nativePCM16Resampler) WriteTo(data []byte, sink func([]byte) error) error {
	if resampler == nil || resampler.handle == nil || resampler.closed.Load() {
		return ErrResamplerClosed
	}
	if sink == nil {
		return ErrResampleSinkRequired
	}
	if len(data)%pcm16BytesPerSample != 0 {
		return fmt.Errorf("%w: %d", ErrInvalidPCM16InputLength, len(data))
	}
	if len(data) == 0 {
		return nil
	}

	inputSampleCount := len(data) / pcm16BytesPerSample
	if inputSampleCount > resampler.maxInputSampleCount {
		resampler.maxInputSampleCount = inputSampleCount
	}
	producedSamples, err := resampler.process(data, false)
	if err != nil {
		return err
	}
	if !resampler.frameBuffered {
		return resampler.writePCM16Samples(producedSamples, sink)
	}

	resampler.pendingOutputSamples = append(resampler.pendingOutputSamples, producedSamples...)
	return resampler.writeCompleteOutputFrames(resampler.outputFrameSampleCount(inputSampleCount), sink)
}

func (resampler *nativePCM16Resampler) FlushTo(sink func([]byte) error) error {
	if resampler == nil || resampler.handle == nil || resampler.closed.Load() {
		return ErrResamplerClosed
	}
	if sink == nil {
		return ErrResampleSinkRequired
	}
	producedSamples, err := resampler.process(nil, true)
	if err != nil {
		return err
	}
	if !resampler.frameBuffered {
		return resampler.writePCM16Samples(producedSamples, sink)
	}
	resampler.pendingOutputSamples = append(resampler.pendingOutputSamples, producedSamples...)
	err = resampler.writePCM16Samples(resampler.pendingOutputSamples, sink)
	resampler.pendingOutputSamples = resampler.pendingOutputSamples[:0]
	return err
}

func (resampler *nativePCM16Resampler) process(input []byte, flush bool) ([]int16, error) {
	outputSampleCount := resampler.outputCapacitySampleCount(len(input)/pcm16BytesPerSample, flush)
	if outputSampleCount < minimumOutputSampleCount {
		outputSampleCount = minimumOutputSampleCount
	}
	if cap(resampler.outputSamples) < outputSampleCount {
		resampler.outputSamples = make([]int16, outputSampleCount)
	} else {
		resampler.outputSamples = resampler.outputSamples[:outputSampleCount]
	}

	remainingInput := input
	producedSampleCount := 0
	for {
		if producedSampleCount == len(resampler.outputSamples) {
			resampler.outputSamples = append(resampler.outputSamples, make([]int16, outputSampleCount)...)
		}
		outputSamples := resampler.outputSamples[producedSampleCount:]
		var inputSamplesConsumed C.size_t
		var outputSamplesProduced C.size_t
		var nativeError C.soxr_error_t
		if len(remainingInput) > 0 {
			nativeError = C.soxr_process(
				resampler.handle,
				C.soxr_in_t(unsafe.Pointer(unsafe.SliceData(remainingInput))),
				C.size_t(len(remainingInput)/pcm16BytesPerSample),
				&inputSamplesConsumed,
				C.soxr_out_t(unsafe.Pointer(unsafe.SliceData(outputSamples))),
				C.size_t(len(outputSamples)),
				&outputSamplesProduced,
			)
		} else if flush {
			nativeError = C.soxr_process(
				resampler.handle,
				nil,
				0,
				nil,
				C.soxr_out_t(unsafe.Pointer(unsafe.SliceData(outputSamples))),
				C.size_t(len(outputSamples)),
				&outputSamplesProduced,
			)
		} else {
			break
		}
		if nativeError != nil {
			message := C.GoString(nativeError)
			C.free(unsafe.Pointer(nativeError))
			return nil, fmt.Errorf("%w: %s", ErrNativeSOXRFailed, message)
		}

		producedSampleCount += int(outputSamplesProduced)
		if len(remainingInput) == 0 {
			if !flush || int(outputSamplesProduced) == 0 || int(outputSamplesProduced) < len(outputSamples) {
				break
			}
			continue
		}
		if int(inputSamplesConsumed) == 0 && int(outputSamplesProduced) == 0 {
			return nil, fmt.Errorf("%w: consumed 0 of %d", ErrIncompleteSOXRInput, len(remainingInput)/pcm16BytesPerSample)
		}
		remainingInput = remainingInput[int(inputSamplesConsumed)*pcm16BytesPerSample:]
		if len(remainingInput) == 0 {
			break
		}
	}

	return resampler.outputSamples[:producedSampleCount], nil
}

func (resampler *nativePCM16Resampler) outputCapacitySampleCount(inputSampleCount int, flush bool) int {
	if flush && inputSampleCount == 0 {
		inputSampleCount = resampler.maxInputSampleCount
	}
	outputSampleCount := int(
		(uint64(inputSampleCount)*uint64(resampler.targetRate) + uint64(resampler.sourceRate)/2) /
			uint64(resampler.sourceRate),
	)
	return outputSampleCount + nativeOutputExtraSamples
}

func (resampler *nativePCM16Resampler) outputFrameSampleCount(inputSampleCount int) int {
	outputFrameSampleCount := int(uint64(inputSampleCount) * uint64(resampler.targetRate) / uint64(resampler.sourceRate))
	if outputFrameSampleCount < minimumOutputSampleCount {
		return minimumOutputSampleCount
	}
	return outputFrameSampleCount
}

func (resampler *nativePCM16Resampler) drainCompleteOutputFrames(frameSampleCount int) []byte {
	completeSampleCount := len(resampler.pendingOutputSamples) / frameSampleCount * frameSampleCount
	if completeSampleCount == 0 {
		return []byte{}
	}
	output := encodePCM16Samples(resampler.pendingOutputSamples[:completeSampleCount])
	copy(resampler.pendingOutputSamples, resampler.pendingOutputSamples[completeSampleCount:])
	resampler.pendingOutputSamples = resampler.pendingOutputSamples[:len(resampler.pendingOutputSamples)-completeSampleCount]
	return output
}

func (resampler *nativePCM16Resampler) writeCompleteOutputFrames(frameSampleCount int, sink func([]byte) error) error {
	completeSampleCount := len(resampler.pendingOutputSamples) / frameSampleCount * frameSampleCount
	if completeSampleCount == 0 {
		return nil
	}
	if err := resampler.writePCM16Samples(resampler.pendingOutputSamples[:completeSampleCount], sink); err != nil {
		return err
	}
	copy(resampler.pendingOutputSamples, resampler.pendingOutputSamples[completeSampleCount:])
	resampler.pendingOutputSamples = resampler.pendingOutputSamples[:len(resampler.pendingOutputSamples)-completeSampleCount]
	return nil
}

func (resampler *nativePCM16Resampler) writePCM16Samples(samples []int16, sink func([]byte) error) error {
	if len(samples) == 0 {
		return nil
	}
	byteCount := len(samples) * pcm16BytesPerSample
	return sink(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(samples))), byteCount))
}

func encodePCM16Samples(samples []int16) []byte {
	output := make([]byte, len(samples)*pcm16BytesPerSample)
	for sampleIndex, sample := range samples {
		// #nosec G115, PCM16 encoding preserves the signed sample bits.
		binary.LittleEndian.PutUint16(output[sampleIndex*pcm16BytesPerSample:], uint16(sample))
	}
	return output
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
