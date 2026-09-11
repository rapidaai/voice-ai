// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"math"
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
)

const (
	// Provider constants configure Pipecat Smart Turn EOS at construction time.
	pipecatEndOfSpeechName = "pipecatSmartTurnEndOfSpeech"
	optPctThreshold        = internal_options.MicrophoneEOSOptionThreshold
	optPctExtendedTimeout  = internal_options.MicrophoneEOSOptionExtendedTimeout
	optPctFallbackTimeout  = internal_options.MicrophoneEOSOptionFallbackTimeout
	optPctModelPath        = internal_options.MicrophoneEOSOptionPipecatModelPath

	// The transcript safety budget and Smart Turn silence limit run independently after VAD stop.
	defaultPctThreshold       = 0.85
	defaultPctExtendedTimeout = 4000.0
	defaultPctFallbackTimeout = 1000.0

	maxAudioSamples        = whisperMaxSamples
	preSpeechAudioSamples  = whisperSampleRate / 2
	pipecatAudioSampleRate = whisperSampleRate
	// Convert samples directly to nanoseconds without overflowing an intermediate product.
	pipecatAudioSampleDuration = time.Second / pipecatAudioSampleRate
	maxAudioDurationSamples    = math.MaxInt64 / uint64(pipecatAudioSampleDuration)
)

const (
	vadStateIdle vadState = iota
	vadStateSpeaking
	vadStateEnded
)

const (
	turnStatePending turnState = iota
	turnStateIncomplete
	turnStateComplete
)

const (
	transcriptStateIdle transcriptState = iota
	transcriptStateInterimPending
	transcriptStateFinalized
	transcriptStateFinalizedWithPendingInterim
	transcriptStateUserText
)

const (
	// Whisper constants define the fixed feature shape expected by Smart Turn.
	whisperSampleRate = 16000
	whisperNFFT       = 400
	whisperHopLength  = 160
	whisperNMels      = 80
	whisperChunkSec   = 8
	whisperMaxSamples = whisperChunkSec * whisperSampleRate
	whisperMaxFrames  = whisperMaxSamples / whisperHopLength
	whisperNFreqBins  = whisperNFFT/2 + 1

	// NumPy's buffered float32 reductions fix the rounding order before feature extraction.
	whisperReductionChunkSamples = 8192
	whisperReductionLeafSamples  = 128
	whisperReductionLanes        = 8
	// Uneven chunks need one extra split after the aligned 128-sample leaves.
	whisperReductionTreeNodes = 4 * whisperReductionChunkSamples / whisperReductionLeafSamples
	// Pipecat adds this scalar before taking the waveform variance's square root.
	whisperVarianceEpsilon = 1e-7
)

const (
	// Slaney constants are used only while building the mel filterbank.
	melFSP      = 200.0 / 3.0
	melMinLogHz = 1000.0
	melMinLogM  = melMinLogHz / melFSP
	melLogStep  = 0.06875177742094912
)

const (
	// Pipecat model constants select the bundled Smart Turn model by default.
	pctDetectorName    = "pipecat_smart_turn"
	envPctModelPathKey = "PIPECAT_TURN_MODEL_PATH"
	defaultPctModel    = "models/smart-turn-v3.2-cpu.onnx"
)
