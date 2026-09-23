// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
)

const (
	// eosName tags logs, metrics, and executor identity for this provider.
	eosName = "livekitEndOfSpeech"

	// Option keys are read when constructing LiveKit EOS from runtime config.
	optKeyThreshold       = internal_options.MicrophoneEOSOptionThreshold
	optKeyQuickTimeout    = internal_options.MicrophoneEOSOptionQuickTimeout
	optKeyExtendedTimeout = internal_options.MicrophoneEOSOptionExtendedTimeout
	optKeyMaxHistory      = internal_options.MicrophoneEOSOptionMaxHistoryTurns
	optKeyModel           = internal_options.MicrophoneEOSOptionModel
	optKeyModelPath       = internal_options.MicrophoneEOSOptionLivekitModelPath
	optKeyTokenizerPath   = internal_options.MicrophoneEOSOptionLivekitTokenizerPath

	// Predictions at or above this probability use the minimum endpoint delay.
	defaultThreshold = 0.0289
	// Endpoint delays are whole milliseconds measured from speech stop, not from inference completion.
	defaultSilenceTimeout = 3000.0
	defaultQuickTimeout   = 250.0
	// History counts user/assistant messages including current text, before adjacent roles are merged.
	defaultMaxHistory     = 6
	defaultModelType      = "en"
	multilingualModelType = "multilingual"

	// Match LiveKit's inference deadline without changing the configured endpoint delays.
	predictionTimeout = 3 * time.Second
)

const (
	vadStateIdle vadState = iota
	vadStateSpeaking
	vadStateEnded
)

const (
	transcriptStateIdle transcriptState = iota
	transcriptStateInterimPending
	transcriptStateFinalized
	transcriptStateFinalizedWithPendingInterim
	transcriptStateUserText
)

// EOS audio is mono PCM16 at 16 kHz; VAD offsets use the same cumulative audio clock.
const livekitAudioSampleRate = 16000

// ByteLevel uses GPT-2 boundaries before BPE when the tokenizer enables use_regex.
// #nosec G101, this public tokenizer expression is not a credential.
const tokenizerByteLevelPattern = `'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+(?!\S)|\s+`

// The pinned multilingual tokenizer isolates these matches before byte-level encoding.
const tokenizerMultilingualSplitPattern = `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`

const (
	// Turn detector paths select local model assets unless env overrides exist.
	turnDetectorName      = "livekit_turn_detector"
	maxHistoryTokens      = 128
	envModelPathKey       = "LIVEKIT_TURN_MODEL_PATH"
	envModelMultiPathKey  = "LIVEKIT_TURN_MULTI_MODEL_PATH"
	envTokenizerPathKey   = "LIVEKIT_TURN_TOKENIZER_PATH" // #nosec G101, environment variable name, not a credential.
	defaultModelFileEn    = "models/model_q8.onnx"
	defaultModelFileMulti = "models/model_q8_multilingual.onnx"
	defaultTokenizerFile  = "models/tokenizer.json"
)
