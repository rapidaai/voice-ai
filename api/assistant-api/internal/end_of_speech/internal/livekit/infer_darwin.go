// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build darwin

package internal_livekit

// #cgo CFLAGS: -Wall -Werror -std=c99
// #cgo LDFLAGS: -lonnxruntime
// #include "ort_bridge.h"
import "C"

import (
	"fmt"
	"unsafe"
)

// infer runs ONNX inference with input_ids tensor only.
// Returns the turn-end probability directly (model output is [1] float32).
//
// darwin: uses C.longlong for int64 tensor dimensions.
func (td *TurnDetector) infer(inputIDs []int64, runOptions *C.OrtRunOptions) (float64, error) {
	seqLen := len(inputIDs)

	// --- Input tensor: input_ids [1, seq_len] ---
	var idsValue *C.OrtValue
	idsDims := []C.longlong{1, C.longlong(seqLen)}
	status := C.LktOrtApiCreateTensorWithDataAsOrtValue(td.api, td.memoryInfo,
		unsafe.Pointer(&inputIDs[0]), C.size_t(seqLen*8),
		(*C.int64_t)(unsafe.Pointer(&idsDims[0])), C.size_t(len(idsDims)),
		C.ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &idsValue)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errTurnDetectorCreateInputIDsTensor, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}
	defer C.LktOrtApiReleaseValue(td.api, idsValue)

	// --- Run inference ---
	inputs := []*C.OrtValue{idsValue}
	outputs := []*C.OrtValue{nil}
	inputNames := []*C.char{td.cStrings["input_ids"]}
	outputNames := []*C.char{td.cStrings["prob"]}

	status = C.LktOrtApiRun(td.api, td.session, runOptions,
		&inputNames[0], &inputs[0], C.size_t(len(inputNames)),
		&outputNames[0], C.size_t(len(outputNames)), &outputs[0])
	if outputs[0] != nil {
		defer C.LktOrtApiReleaseValue(td.api, outputs[0])
	}
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errTurnDetectorRunInference, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	// --- Extract output probability [1] ---
	var probPtr unsafe.Pointer
	status = C.LktOrtApiGetTensorMutableData(td.api, outputs[0], &probPtr)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errTurnDetectorGetOutputData, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	prob := float64(*(*float32)(probPtr))
	return prob, nil
}

// inferMulti runs ONNX inference for the multilingual model.
// Output shape: [1, seq_len] float32. Returns all token probabilities.
func (td *TurnDetector) inferMulti(inputIDs []int64, runOptions *C.OrtRunOptions) ([]float64, error) {
	seqLen := len(inputIDs)

	var idsValue *C.OrtValue
	idsDims := []C.longlong{1, C.longlong(seqLen)}
	status := C.LktOrtApiCreateTensorWithDataAsOrtValue(td.api, td.memoryInfo,
		unsafe.Pointer(&inputIDs[0]), C.size_t(seqLen*8),
		(*C.int64_t)(unsafe.Pointer(&idsDims[0])), C.size_t(len(idsDims)),
		C.ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &idsValue)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return nil, fmt.Errorf("%w: %s", errTurnDetectorCreateInputIDsTensor, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}
	defer C.LktOrtApiReleaseValue(td.api, idsValue)

	inputs := []*C.OrtValue{idsValue}
	outputs := []*C.OrtValue{nil}
	inputNames := []*C.char{td.cStrings["input_ids"]}
	outputNames := []*C.char{td.cStrings["prob"]}

	status = C.LktOrtApiRun(td.api, td.session, runOptions,
		&inputNames[0], &inputs[0], C.size_t(len(inputNames)),
		&outputNames[0], C.size_t(len(outputNames)), &outputs[0])
	if outputs[0] != nil {
		defer C.LktOrtApiReleaseValue(td.api, outputs[0])
	}
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return nil, fmt.Errorf("%w: %s", errTurnDetectorRunInference, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	var dataPtr unsafe.Pointer
	status = C.LktOrtApiGetTensorMutableData(td.api, outputs[0], &dataPtr)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return nil, fmt.Errorf("%w: %s", errTurnDetectorGetOutputData, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	// Copy [1, seq_len] float32 → []float64
	probs := make([]float64, seqLen)
	f32Ptr := (*[1 << 30]float32)(dataPtr)
	for i := 0; i < seqLen; i++ {
		probs[i] = float64(f32Ptr[i])
	}

	return probs, nil
}
