// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build !darwin

package internal_pipecat

// #cgo CFLAGS: -Wall -Werror -std=c99
// #cgo LDFLAGS: -lonnxruntime
// #include "ort_bridge.h"
import "C"

import (
	"fmt"
	"unsafe"
)

// inferWithRunOptions runs ONNX on [1, 80, 800] mel features and returns the probability.
// Linux uses C.long for int64 tensor dimensions.
func (pd *PipecatDetector) inferWithRunOptions(features []float32, runOptions *C.OrtRunOptions) (float64, error) {
	// --- Input tensor: input_features [1, 80, 800] ---
	var featValue *C.OrtValue
	featDims := []C.long{1, whisperNMels, whisperMaxFrames}
	status := C.PctOrtApiCreateTensorWithDataAsOrtValue(pd.api, pd.memoryInfo,
		unsafe.Pointer(&features[0]), C.size_t(len(features)*4),
		(*C.int64_t)(unsafe.Pointer(&featDims[0])), C.size_t(len(featDims)),
		C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &featValue)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errPipecatDetectorCreateInputTensor, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}
	defer C.PctOrtApiReleaseValue(pd.api, featValue)

	// --- Run inference ---
	inputs := []*C.OrtValue{featValue}
	outputs := []*C.OrtValue{nil}
	inputNames := []*C.char{pd.cStrings["input_features"]}
	outputNames := []*C.char{pd.cStrings["logits"]}

	status = C.PctOrtApiRun(pd.api, pd.session, runOptions,
		&inputNames[0], &inputs[0], C.size_t(len(inputNames)),
		&outputNames[0], C.size_t(len(outputNames)), &outputs[0])
	if outputs[0] != nil {
		defer C.PctOrtApiReleaseValue(pd.api, outputs[0])
	}
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errPipecatDetectorRunInference, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	// --- Extract output probability ---
	var probPtr unsafe.Pointer
	status = C.PctOrtApiGetTensorMutableData(pd.api, outputs[0], &probPtr)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errPipecatDetectorGetOutputData, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	prob := float64(*(*float32)(probPtr))
	return prob, nil
}
