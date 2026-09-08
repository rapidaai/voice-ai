// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

// #cgo CFLAGS: -Wall -Werror -std=c99
// #cgo LDFLAGS: -lonnxruntime
// #include "ort_bridge.h"
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

// PipecatDetectorConfig holds configuration for the Pipecat Smart Turn ONNX model.
type PipecatDetectorConfig struct {
	ModelPath string
}

// PipecatDetector manages the ONNX session for the Pipecat Smart Turn model.
// It takes mel spectrogram features and returns the probability that the
// user's turn is complete.
//
// NOT safe for concurrent use — the caller must serialize access.
type PipecatDetector struct {
	api         *C.OrtApi
	env         *C.OrtEnv
	sessionOpts *C.OrtSessionOptions
	session     *C.OrtSession
	memoryInfo  *C.OrtMemoryInfo

	cStrings map[string]*C.char

	features *whisperFeatures
	scratch  *whisperFeatureScratch
}

// NewPipecatDetector loads the ONNX model, initializes the inference session,
// and pre-computes the Whisper mel filterbank.
func NewPipecatDetector(cfg PipecatDetectorConfig) (*PipecatDetector, error) {
	modelPath := resolvePctModelPath(cfg.ModelPath)

	pd := &PipecatDetector{
		cStrings: map[string]*C.char{},
		features: newWhisperFeatures(),
		scratch:  newWhisperFeatureScratch(),
	}

	pd.api = C.PctOrtGetApi()
	if pd.api == nil {
		return nil, errPipecatDetectorRuntimeAPIUnavailable
	}

	pd.cStrings["loggerName"] = C.CString(pctDetectorName)
	status := C.PctOrtApiCreateEnv(pd.api, C.ORT_LOGGING_LEVEL_ERROR, pd.cStrings["loggerName"], &pd.env)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorCreateEnv, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	status = C.PctOrtApiCreateSessionOptions(pd.api, &pd.sessionOpts)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorCreateSessionOptions, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	status = C.PctOrtApiSetIntraOpNumThreads(pd.api, pd.sessionOpts, 1)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorSetIntraThreads, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	status = C.PctOrtApiSetInterOpNumThreads(pd.api, pd.sessionOpts, 1)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorSetInterThreads, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	status = C.PctOrtApiSetSessionGraphOptimizationLevel(pd.api, pd.sessionOpts, C.ORT_ENABLE_ALL)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorSetOptimization, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	pd.cStrings["modelPath"] = C.CString(modelPath)
	status = C.PctOrtApiCreateSession(pd.api, pd.env, pd.cStrings["modelPath"], pd.sessionOpts, &pd.session)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorCreateSession, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	status = C.PctOrtApiCreateCpuMemoryInfo(pd.api, C.OrtArenaAllocator, C.OrtMemTypeDefault, &pd.memoryInfo)
	defer C.PctOrtApiReleaseStatus(pd.api, status)
	if status != nil {
		pd.cleanup()
		return nil, fmt.Errorf("%w: %s", errPipecatDetectorCreateMemoryInfo, C.GoString(C.PctOrtApiGetErrorMessage(pd.api, status)))
	}

	pd.cStrings["input_features"] = C.CString("input_features")
	pd.cStrings["logits"] = C.CString("logits")

	return pd, nil
}

// Predict computes mel features from raw audio and returns the probability
// that the user's turn is complete.
//
// audio must be float32 PCM samples at 16kHz.
func (pd *PipecatDetector) Predict(audio []float32) (float64, error) {
	if pd == nil {
		return 0, errPipecatDetectorNil
	}
	if len(audio) == 0 {
		return 0, errPipecatDetectorEmptyAudio
	}

	// Extract mel spectrogram features [80 * 800]
	features := pd.features.extractInto(audio, pd.scratch.output[:], pd.scratch)

	// Run ONNX inference
	prob, err := pd.infer(features)
	if err != nil {
		return 0, err
	}

	return prob, nil
}

// Destroy releases all ONNX Runtime resources.
func (pd *PipecatDetector) Destroy() {
	if pd == nil {
		return
	}
	pd.cleanup()
}

func (pd *PipecatDetector) cleanup() {
	if pd.memoryInfo != nil {
		C.PctOrtApiReleaseMemoryInfo(pd.api, pd.memoryInfo)
		pd.memoryInfo = nil
	}
	if pd.session != nil {
		C.PctOrtApiReleaseSession(pd.api, pd.session)
		pd.session = nil
	}
	if pd.sessionOpts != nil {
		C.PctOrtApiReleaseSessionOptions(pd.api, pd.sessionOpts)
		pd.sessionOpts = nil
	}
	if pd.env != nil {
		C.PctOrtApiReleaseEnv(pd.api, pd.env)
		pd.env = nil
	}
	for k, ptr := range pd.cStrings {
		C.free(unsafe.Pointer(ptr))
		delete(pd.cStrings, k)
	}
}

func resolvePctModelPath(configured string) string {
	if configured != "" {
		return configured
	}
	if envPath := os.Getenv(envPctModelPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultPctModel)
}
