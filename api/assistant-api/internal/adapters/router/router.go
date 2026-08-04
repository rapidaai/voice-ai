// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package router

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// Route identifies which dispatcher channel should receive a packet.
type Route uint8

const (
	RouteControl Route = iota + 1
	RouteBootstrap
	RouteIngress
	RouteEgress
	RouteData
	RouteBackground
)

// Classify maps a packet to its dispatcher route.
// Unknown packets default to RouteBackground.
func Classify(p internal_type.Packet) Route {
	return ClassifyName(p.PacketName())
}

func ClassifyName(name internal_type.PacketName) Route {
	switch name {
	// Critical — interrupts, tool lifecycle
	case internal_type.PacketNameInterruptionDetected,
		internal_type.PacketNameTextToSpeechInterrupt,
		internal_type.PacketNameLLMInterrupt,
		internal_type.PacketNameDispatchPolicy,
		internal_type.PacketNameSpeechToTextEnd,
		internal_type.PacketNameSpeechToTextStart,
		internal_type.PacketNameEndOfSpeechInterruption,
		internal_type.PacketNameTurnChange:
		return RouteControl

	// Bootstrap — connect/session initialization pipeline
	case internal_type.PacketNameInitializeAssistant,
		internal_type.PacketNameInitializeConversation,
		internal_type.PacketNameInitializeSessionRuntime,
		internal_type.PacketNameInitializeConversationRecordingExecutor,
		internal_type.PacketNameInitializeArtifactPushExecutor,
		internal_type.PacketNameInitializeAnalysisExecutor,
		internal_type.PacketNameInitializeAuthentication,
		internal_type.PacketNameExecuteAuthentication,
		internal_type.PacketNameSessionAuthenticationSucceeded,
		internal_type.PacketNameSessionAuthenticationFailed,
		internal_type.PacketNameInitializeSpeechToText,
		internal_type.PacketNameInitializeAssistantExecutor,
		internal_type.PacketNameInitializeTextToSpeech,
		internal_type.PacketNameInitializeVoiceActivityDetection,
		internal_type.PacketNameInitializeEndOfSpeech,
		internal_type.PacketNameInitializeDenoise,
		internal_type.PacketNameInitializeBehavior,
		internal_type.PacketNameInitializationCompleted,
		internal_type.PacketNameInitializationFailed,
		internal_type.PacketNameInitializeInboundDispatcher,
		internal_type.PacketNameFinalizeInboundDispatcher,
		internal_type.PacketNameModeSwitchRequested,
		internal_type.PacketNameModeSwitchCompleted,
		internal_type.PacketNameModeSwitchInitializeSpeechToText,
		internal_type.PacketNameModeSwitchInitializeTextToSpeech,
		internal_type.PacketNameModeSwitchInitializeVoiceActivityDetection,
		internal_type.PacketNameModeSwitchInitializeEndOfSpeech,
		internal_type.PacketNameModeSwitchFinalizeEndOfSpeech,
		internal_type.PacketNameModeSwitchInitializeDenoise,
		internal_type.PacketNameModeSwitchFinalizeDenoise,
		internal_type.PacketNameModeSwitchFinalizeVoiceActivityDetection,
		internal_type.PacketNameModeSwitchFinalizeTextToSpeech,
		internal_type.PacketNameModeSwitchFinalizeSpeechToText:
		return RouteBootstrap

	// Input — inbound audio pipeline, VAD, STT, EOS
	case internal_type.PacketNameUserAudioReceived,
		internal_type.PacketNameUserTextReceived,
		internal_type.PacketNameDenoiseAudio,
		internal_type.PacketNameDenoisedAudio,
		internal_type.PacketNameVadAudio,
		internal_type.PacketNameVadSpeechActivity,
		internal_type.PacketNameSpeechToText,
		internal_type.PacketNameSpeechToTextAudio,
		internal_type.PacketNameEndOfSpeechAudio,
		internal_type.PacketNameEndOfSpeech,
		internal_type.PacketNameInterimEndOfSpeech,
		internal_type.PacketNameUserInput,
		internal_type.PacketNameLLMToolResult:
		return RouteIngress

	// Output — LLM generation, TTS, outbound pipeline
	case internal_type.PacketNameLLMResponseDelta,
		internal_type.PacketNameLLMResponseDone,
		internal_type.PacketNameSpeechToTextError,
		internal_type.PacketNameLLMError,
		internal_type.PacketNameTextToSpeechError,
		internal_type.PacketNameModeSwitchError,
		internal_type.PacketNameInjectMessage,
		internal_type.PacketNameStartIdleTimeout,
		internal_type.PacketNameStopIdleTimeout,
		internal_type.PacketNameIdleTimeoutExpired,
		internal_type.PacketNameUnclearInputExpired,
		internal_type.PacketNameMaxSessionExpired,
		internal_type.PacketNameTextToSpeechText,
		internal_type.PacketNameTextToSpeechDone,
		internal_type.PacketNameTextToSpeechAudio,
		internal_type.PacketNameTextToSpeechEnd,
		internal_type.PacketNameLLMToolCall:
		return RouteEgress

	// Data — DB writes, recording, lifecycle orchestration. No observer dependency,
	// dispatcher starts at NewGenericRequestor.
	case internal_type.PacketNameRecordUserAudio,
		internal_type.PacketNameRecordAssistantAudio,
		internal_type.PacketNameConversationRecordingCompleted,
		internal_type.PacketNameMessageCreate,
		internal_type.PacketNameToolLogCreate,
		internal_type.PacketNameToolLogUpdate,
		internal_type.PacketNameHTTPLogCreate,
		internal_type.PacketNameFinalizeBehavior,
		internal_type.PacketNameFinalizeEndOfSpeech,
		internal_type.PacketNameFinalizeVoiceActivityDetection,
		internal_type.PacketNameFinalizeTextToSpeech,
		internal_type.PacketNameFinalizeSpeechToText,
		internal_type.PacketNameFinalizeAuthentication,
		internal_type.PacketNameFinalizeConversationRecordingExecutor,
		internal_type.PacketNameFinalizeSessionRuntime,
		internal_type.PacketNameFinalizeArtifactPushExecutor,
		internal_type.PacketNameExecuteAnalysis,
		internal_type.PacketNameFinalizeConversation,
		internal_type.PacketNameFinalizeAnalysisExecutor,
		internal_type.PacketNameFinalizeAssistant,
		internal_type.PacketNameFinalizationCompleted:
		return RouteData

	// Background — observer-touching telemetry. Dispatcher starts after telemetry init.
	case internal_type.PacketNameObservabilityLogRecord,
		internal_type.PacketNameObservabilityEventRecord,
		internal_type.PacketNameObservabilityMetricRecord,
		internal_type.PacketNameObservabilityMetadataRecord,
		internal_type.PacketNameObservabilityUsageRecord,
		internal_type.PacketNameObservabilityWebhookRecord:
		return RouteBackground
	default:
		return RouteBackground
	}
}
