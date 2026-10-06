// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import "errors"

var (
	// Construction errors preserve their underlying cause for errors.Is and errors.As.
	errPipecatOnPacketRequired = errors.New("pipecat_eos: packet callback is required")
	errPipecatInvalidOption    = errors.New("pipecat_eos: invalid option")
	errPipecatInitDetector     = errors.New("pipecat_eos: initialize detector")

	// Native detector errors identify the failing operation; runtime details are wrapped at the call site.
	errPipecatDetectorRuntimeAPIUnavailable = errors.New("pipecat_detector: failed to get ONNX Runtime API")
	errPipecatDetectorCreateEnv             = errors.New("pipecat_detector: create env")
	errPipecatDetectorCreateSessionOptions  = errors.New("pipecat_detector: create session options")
	errPipecatDetectorSetIntraThreads       = errors.New("pipecat_detector: set intra threads")
	errPipecatDetectorSetInterThreads       = errors.New("pipecat_detector: set inter threads")
	errPipecatDetectorSetOptimization       = errors.New("pipecat_detector: set optimization")
	errPipecatDetectorCreateSession         = errors.New("pipecat_detector: create session")
	errPipecatDetectorCreateMemoryInfo      = errors.New("pipecat_detector: create memory info")
	errPipecatDetectorCreateRunOptions      = errors.New("pipecat_detector: create run options")
	errPipecatDetectorCreateInputTensor     = errors.New("pipecat_detector: create input tensor")
	errPipecatDetectorRunInference          = errors.New("pipecat_detector: run inference")
	errPipecatDetectorGetOutputData         = errors.New("pipecat_detector: get output data")
	errPipecatDetectorNil                   = errors.New("pipecat_detector: nil detector")
	errPipecatDetectorEmptyAudio            = errors.New("pipecat_detector: empty audio")
)
