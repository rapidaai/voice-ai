// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import "errors"

var (
	errLivekitOnPacketRequired = errors.New("onPacket is required")
	errLivekitInitTurnDetector = errors.New("livekit_eos: init turn detector")

	errTokenizerReadFile        = errors.New("tokenizer: read file")
	errTokenizerUnmarshal       = errors.New("tokenizer: unmarshal")
	errTokenizerPreTokenizer    = errors.New("tokenizer: unsupported pretokenizer")
	errTokenizerTextPreparation = errors.New("tokenizer: unsupported text preparation")

	errTurnDetectorLoadTokenizer         = errors.New("turn_detector: load tokenizer")
	errTurnDetectorRuntimeAPIUnavailable = errors.New("turn_detector: failed to get ONNX Runtime API")
	errTurnDetectorCreateEnv             = errors.New("turn_detector: create env")
	errTurnDetectorCreateSessionOptions  = errors.New("turn_detector: create session options")
	errTurnDetectorCreateRunOptions      = errors.New("turn_detector: create run options")
	errTurnDetectorSetIntraThreads       = errors.New("turn_detector: set intra threads")
	errTurnDetectorSetInterThreads       = errors.New("turn_detector: set inter threads")
	errTurnDetectorSetOptimization       = errors.New("turn_detector: set optimization")
	errTurnDetectorCreateSession         = errors.New("turn_detector: create session")
	errTurnDetectorCreateMemoryInfo      = errors.New("turn_detector: create memory info")
	errTurnDetectorCreateInputIDsTensor  = errors.New("turn_detector: create input_ids tensor")
	errTurnDetectorRunInference          = errors.New("turn_detector: run inference")
	errTurnDetectorGetOutputData         = errors.New("turn_detector: get output data")
	errTurnDetectorNil                   = errors.New("turn_detector: nil detector")
	errTurnDetectorEmptyTokenSequence    = errors.New("turn_detector: empty token sequence")
	errTurnDetectorEmptyOutput           = errors.New("turn_detector: empty output")
)
