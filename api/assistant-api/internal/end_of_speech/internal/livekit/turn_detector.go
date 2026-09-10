// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

// #cgo CFLAGS: -Wall -Werror -std=c99
// #cgo LDFLAGS: -lonnxruntime
// #include "ort_bridge.h"
import "C"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

// TurnDetectorConfig holds configuration for the turn detector ONNX model.
type TurnDetectorConfig struct {
	ModelPath     string
	TokenizerPath string
	// ModelType selects the model variant: "en" (default, 66MB) or
	// "multilingual" (378MB, 14 languages). The multilingual model has
	// a different output shape [1, seq_len] vs [1] for English.
	ModelType string
}

// TurnDetector manages the ONNX session for the LiveKit turn detection model.
// It tokenizes conversation text, runs inference, and returns the probability
// that the user has finished their turn (P(im_end)).
//
// NOT safe for concurrent use. The caller must serialize access.
type TurnDetector struct {
	api         *C.OrtApi
	env         *C.OrtEnv
	sessionOpts *C.OrtSessionOptions
	session     *C.OrtSession
	memoryInfo  *C.OrtMemoryInfo

	cStrings map[string]*C.char

	tok *tokenizer

	// multilingual is true when using the multilingual model which outputs
	// [1, seq_len] probabilities (last token = EOU prob) instead of [1].
	multilingual bool
}

// NewTurnDetector loads the ONNX model and tokenizer, initializes the
// inference session, and returns a ready TurnDetector.
func NewTurnDetector(cfg TurnDetectorConfig) (*TurnDetector, error) {
	isMultilingual := cfg.ModelType == "multilingual"
	modelPath := resolveModelPath(cfg.ModelPath, isMultilingual)
	tokenizerPath := resolveTokenizerPath(cfg.TokenizerPath)

	tok, err := newTokenizer(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errTurnDetectorLoadTokenizer, err)
	}

	td := &TurnDetector{
		cStrings:     map[string]*C.char{},
		tok:          tok,
		multilingual: isMultilingual,
	}

	td.api = C.LktOrtGetApi()
	if td.api == nil {
		return nil, errTurnDetectorRuntimeAPIUnavailable
	}

	td.cStrings["loggerName"] = C.CString(turnDetectorName)
	status := C.LktOrtApiCreateEnv(td.api, C.ORT_LOGGING_LEVEL_ERROR, td.cStrings["loggerName"], &td.env)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorCreateEnv, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	status = C.LktOrtApiCreateSessionOptions(td.api, &td.sessionOpts)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorCreateSessionOptions, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	status = C.LktOrtApiSetIntraOpNumThreads(td.api, td.sessionOpts, 1)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorSetIntraThreads, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	status = C.LktOrtApiSetInterOpNumThreads(td.api, td.sessionOpts, 1)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorSetInterThreads, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	status = C.LktOrtApiSetSessionGraphOptimizationLevel(td.api, td.sessionOpts, C.ORT_ENABLE_ALL)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorSetOptimization, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	td.cStrings["modelPath"] = C.CString(modelPath)
	status = C.LktOrtApiCreateSession(td.api, td.env, td.cStrings["modelPath"], td.sessionOpts, &td.session)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorCreateSession, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	status = C.LktOrtApiCreateCpuMemoryInfo(td.api, C.OrtArenaAllocator, C.OrtMemTypeDefault, &td.memoryInfo)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		td.cleanup()
		return nil, fmt.Errorf("%w: %s", errTurnDetectorCreateMemoryInfo, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}

	td.cStrings["input_ids"] = C.CString("input_ids")
	td.cStrings["prob"] = C.CString("prob")

	return td, nil
}

// Predict runs inference on the given text (already formatted via chat template)
// and returns the probability that the user has finished their turn (P(im_end)).
//
// The text should be pre-formatted using formatChatTemplate with the last user
// message left open (no closing <|im_end|>).
func (td *TurnDetector) Predict(text string) (float64, error) {
	return td.PredictContext(context.Background(), text)
}

// PredictContext runs inference synchronously and terminates its native run on cancellation.
// The caller must serialize prediction and Destroy, including cancellation cleanup.
func (td *TurnDetector) PredictContext(ctx context.Context, text string) (prob float64, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if td == nil {
		return 0, errTurnDetectorNil
	}

	tokenIDs := td.tok.Encode(text)
	if len(tokenIDs) == 0 {
		return 0, errTurnDetectorEmptyTokenSequence
	}
	if len(tokenIDs) > maxHistoryTokens {
		tokenIDs = tokenIDs[len(tokenIDs)-maxHistoryTokens:]
	}

	inputIDs := make([]int64, len(tokenIDs))
	for i, id := range tokenIDs {
		inputIDs[i] = int64(id)
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var runOptions *C.OrtRunOptions
	status := C.LktOrtApiCreateRunOptions(td.api, &runOptions)
	defer C.LktOrtApiReleaseStatus(td.api, status)
	if status != nil {
		return 0, fmt.Errorf("%w: %s", errTurnDetectorCreateRunOptions, C.GoString(C.LktOrtApiGetErrorMessage(td.api, status)))
	}
	defer C.LktOrtApiReleaseRunOptions(td.api, runOptions)

	terminated := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(terminated)
		status := C.LktOrtApiRunOptionsSetTerminate(td.api, runOptions)
		C.LktOrtApiReleaseStatus(td.api, status)
	})
	defer func() {
		// Join a started callback before releasing the per-run options.
		if !stop() {
			<-terminated
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			prob, err = 0, ctxErr
		}
	}()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if td.multilingual {
		// Multilingual model outputs [1, seq_len]. Take last token's prob.
		probs, err := td.inferMulti(inputIDs, runOptions)
		if err != nil {
			return 0, err
		}
		if len(probs) == 0 {
			return 0, errTurnDetectorEmptyOutput
		}
		return probs[len(probs)-1], nil
	}

	// English model outputs [1]. Direct probability.
	prob, err = td.infer(inputIDs, runOptions)
	if err != nil {
		return 0, err
	}
	return prob, nil
}

// Destroy releases all ONNX Runtime resources.
func (td *TurnDetector) Destroy() {
	if td == nil {
		return
	}
	td.cleanup()
}

// cleanup releases ONNX Runtime handles in reverse allocation order.
func (td *TurnDetector) cleanup() {
	if td.memoryInfo != nil {
		C.LktOrtApiReleaseMemoryInfo(td.api, td.memoryInfo)
		td.memoryInfo = nil
	}
	if td.session != nil {
		C.LktOrtApiReleaseSession(td.api, td.session)
		td.session = nil
	}
	if td.sessionOpts != nil {
		C.LktOrtApiReleaseSessionOptions(td.api, td.sessionOpts)
		td.sessionOpts = nil
	}
	if td.env != nil {
		C.LktOrtApiReleaseEnv(td.api, td.env)
		td.env = nil
	}
	for k, ptr := range td.cStrings {
		C.free(unsafe.Pointer(ptr))
		delete(td.cStrings, k)
	}
}

func resolveModelPath(configured string, multilingual bool) string {
	if configured != "" {
		return configured
	}
	if multilingual {
		if envPath := os.Getenv(envModelMultiPathKey); envPath != "" {
			return envPath
		}
		_, currentFile, _, _ := runtime.Caller(0)
		return filepath.Join(filepath.Dir(currentFile), defaultModelFileMulti)
	}
	if envPath := os.Getenv(envModelPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultModelFileEn)
}

func resolveTokenizerPath(configured string) string {
	if configured != "" {
		return configured
	}
	if envPath := os.Getenv(envTokenizerPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultTokenizerFile)
}
