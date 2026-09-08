// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

const (
	// eosName tags logs, metrics, and executor identity for this provider.
	eosName = "livekitEndOfSpeech"

	// Option keys are read when constructing LiveKit EOS from runtime config.
	optKeyThreshold       = "microphone.eos.threshold"
	optKeyQuickTimeout    = "microphone.eos.quick_timeout"
	optKeyExtendedTimeout = "microphone.eos.extended_timeout"
	optKeyFallbackTimeout = "microphone.eos.fallback_timeout"
	optKeyMaxHistory      = "microphone.eos.max_history_turns"
	optKeyModel           = "microphone.eos.model"
	optKeyModelPath       = "microphone.eos.livekit.model_path"
	optKeyTokenizerPath   = "microphone.eos.livekit.tokenizer_path"

	// Legacy option keys preserve older EOS config compatibility.
	optKeyLegacySilenceTimeout = "microphone.eos.silence_timeout"
	optKeyLegacyTimeout        = "microphone.eos.timeout"

	// Defaults mirror LiveKit turn detector behavior when options are absent.
	defaultThreshold       = 0.0289
	defaultSilenceTimeout  = 3000.0
	defaultQuickTimeout    = 250.0
	defaultMaxHistory      = 6.0
	defaultFallbackTimeout = 500.0
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
)

const (
	// Turn detector paths select local model assets unless env overrides exist.
	turnDetectorName      = "livekit_turn_detector"
	maxHistoryTokens      = 128
	envModelPathKey       = "LIVEKIT_TURN_MODEL_PATH"
	envModelMultiPathKey  = "LIVEKIT_TURN_MULTI_MODEL_PATH"
	envTokenizerPathKey   = "LIVEKIT_TURN_TOKENIZER_PATH"
	defaultModelFileEn    = "models/model_q8.onnx"
	defaultModelFileMulti = "models/model_q8_multilingual.onnx"
	defaultTokenizerFile  = "models/tokenizer.json"
)
