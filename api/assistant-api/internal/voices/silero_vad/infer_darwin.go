//go:build darwin

package internal_voice_silero_vad

// #cgo CFLAGS: -Wall -Werror -std=c99
// #include "ort_bridge.h"
import "C"

import (
	"fmt"
	"unsafe"
)

func (sd *sileroDetector) infer(pcm []float32) (float32, error) {
	// Create tensors
	var pcmValue *C.OrtValue
	pcmInputDims := []C.longlong{
		1,
		C.longlong(len(pcm)),
	}
	status := C.OrtApiCreateTensorWithDataAsOrtValue(sd.api, sd.memoryInfo, unsafe.Pointer(&pcm[0]), C.size_t(len(pcm)*4), &pcmInputDims[0], C.size_t(len(pcmInputDims)), C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &pcmValue)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to create value: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}
	defer C.OrtApiReleaseValue(sd.api, pcmValue)

	var stateValue *C.OrtValue
	stateNodeInputDims := []C.longlong{2, 1, 128}
	status = C.OrtApiCreateTensorWithDataAsOrtValue(sd.api, sd.memoryInfo, unsafe.Pointer(&sd.state[0]), C.size_t(stateLen*4), &stateNodeInputDims[0], C.size_t(len(stateNodeInputDims)), C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &stateValue)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to create value: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}
	defer C.OrtApiReleaseValue(sd.api, stateValue)

	var rateValue *C.OrtValue
	rateInputDims := []C.longlong{1}
	rate := []C.int64_t{C.int64_t(sd.cfg.SampleRate)}
	status = C.OrtApiCreateTensorWithDataAsOrtValue(sd.api, sd.memoryInfo, unsafe.Pointer(&rate[0]), C.size_t(8), &rateInputDims[0], C.size_t(len(rateInputDims)), C.ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &rateValue)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to create value: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}
	defer C.OrtApiReleaseValue(sd.api, rateValue)

	// Run inference
	inputs := []*C.OrtValue{pcmValue, stateValue, rateValue}
	outputs := []*C.OrtValue{nil, nil}

	inputNames := []*C.char{
		sd.cStrings["input"],
		sd.cStrings["state"],
		sd.cStrings["sr"],
	}
	outputNames := []*C.char{
		sd.cStrings["output"],
		sd.cStrings["stateN"],
	}
	status = C.OrtApiRun(sd.api, sd.session, nil, &inputNames[0], &inputs[0], C.size_t(len(inputNames)), &outputNames[0], C.size_t(len(outputNames)), &outputs[0])
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to run: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	// Get output values from tensor data
	var prob unsafe.Pointer
	var stateN unsafe.Pointer

	status = C.OrtApiGetTensorMutableData(sd.api, outputs[0], &prob)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to get tensor data: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiGetTensorMutableData(sd.api, outputs[1], &stateN)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return 0, fmt.Errorf("failed to get tensor data: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	C.memcpy(unsafe.Pointer(&sd.state[0]), stateN, stateLen*4)

	C.OrtApiReleaseValue(sd.api, outputs[0])
	C.OrtApiReleaseValue(sd.api, outputs[1])

	// Return speech probability
	return *(*float32)(prob), nil
}
