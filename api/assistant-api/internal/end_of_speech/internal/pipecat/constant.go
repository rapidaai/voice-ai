// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

const (
	// Provider constants configure Pipecat Smart Turn EOS at construction time.
	pipecatEndOfSpeechName = "pipecatSmartTurnEndOfSpeech"
	optPctThreshold        = "microphone.eos.threshold"
	optPctExtendedTimeout  = "microphone.eos.extended_timeout"
	optPctQuickTimeout     = "microphone.eos.quick_timeout"
	optPctFallbackTimeout  = "microphone.eos.fallback_timeout"
	optPctModelPath        = "microphone.eos.pipecat.model_path"

	// Legacy option keys preserve compatibility with older EOS config.
	optPctLegacySilenceTimeout = "microphone.eos.silence_timeout"
	optPctLegacyTimeout        = "microphone.eos.timeout"

	// Timeout defaults are used when runtime config omits Pipecat tuning.
	defaultPctThreshold       = 0.5
	defaultPctQuickTimeout    = 250.0
	defaultPctExtendedTimeout = 2000.0
	defaultPctFallbackTimeout = 500.0

	maxAudioSamples        = whisperMaxSamples
	preSpeechAudioSamples  = whisperSampleRate / 2
	pipecatAudioSampleRate = whisperSampleRate
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
	// Whisper constants define the fixed feature shape expected by Smart Turn.
	whisperSampleRate = 16000
	whisperNFFT       = 400
	whisperHopLength  = 160
	whisperNMels      = 80
	whisperChunkSec   = 8
	whisperMaxSamples = whisperChunkSec * whisperSampleRate
	whisperMaxFrames  = whisperMaxSamples / whisperHopLength
	whisperNFreqBins  = whisperNFFT/2 + 1
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
