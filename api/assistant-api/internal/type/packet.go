// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

// =============================================================================
// Packet Interfaces
// =============================================================================

// Packet represents a generic request packet handled by the adapter layer.
// Concrete packet types signal specific actions or events within a given context.
//
// Naming convention:
//   - Commands (trigger an action): verb-first — ExecuteLLM, DenoiseAudio, InterruptTTS, RecordUserAudio, SaveMessage, SpeakText
//   - Events  (something happened): past-tense/noun — LLMResponseDelta, DenoisedAudio, SpeechToText, EndOfSpeech, TextToSpeechAudio
type Packet interface {
	ContextId() string
	PacketName() PacketName
}

type PacketName string

const (
	PacketNameDispatchPolicy                             PacketName = "DispatchPolicyPacket"
	PacketNameUserTextReceived                           PacketName = "UserTextReceivedPacket"
	PacketNameUserAudioReceived                          PacketName = "UserAudioReceivedPacket"
	PacketNameSpeechToTextAudio                          PacketName = "SpeechToTextAudioPacket"
	PacketNameDenoiseAudio                               PacketName = "DenoiseAudioPacket"
	PacketNameDenoisedAudio                              PacketName = "DenoisedAudioPacket"
	PacketNameVadAudio                                   PacketName = "VadAudioPacket"
	PacketNameSpeechToText                               PacketName = "SpeechToTextPacket"
	PacketNameEndOfSpeechAudio                           PacketName = "EndOfSpeechAudioPacket"
	PacketNameEndOfSpeechInterruption                    PacketName = "EndOfSpeechInterruptionPacket"
	PacketNameEndOfSpeech                                PacketName = "EndOfSpeechPacket"
	PacketNameInterimEndOfSpeech                         PacketName = "InterimEndOfSpeechPacket"
	PacketNameUserInput                                  PacketName = "UserInputPacket"
	PacketNameInterruptionDetected                       PacketName = "InterruptionDetectedPacket"
	PacketNameTextToSpeechInterrupt                      PacketName = "TextToSpeechInterruptPacket"
	PacketNameSpeechToTextError                          PacketName = "SpeechToTextErrorPacket"
	PacketNameSpeechToTextEnd                            PacketName = "SpeechToTextEndPacket"
	PacketNameSpeechToTextStart                          PacketName = "SpeechToTextStartPacket"
	PacketNameLLMInterrupt                               PacketName = "LLMInterruptPacket"
	PacketNameTurnChange                                 PacketName = "TurnChangePacket"
	PacketNameInjectMessage                              PacketName = "InjectMessagePacket"
	PacketNameInitializeAssistant                        PacketName = "InitializeAssistantPacket"
	PacketNameInitializeConversation                     PacketName = "InitializeConversationPacket"
	PacketNameInitializeSessionRuntime                   PacketName = "InitializeSessionRuntimePacket"
	PacketNameInitializeConversationRecordingExecutor    PacketName = "InitializeConversationRecordingExecutorPacket"
	PacketNameInitializeArtifactPushExecutor             PacketName = "InitializeArtifactPushExecutorPacket"
	PacketNameInitializeAnalysisExecutor                 PacketName = "InitializeAnalysisExecutorPacket"
	PacketNameInitializeAuthentication                   PacketName = "InitializeAuthenticationPacket"
	PacketNameExecuteAuthentication                      PacketName = "ExecuteAuthenticationPacket"
	PacketNameSessionAuthenticationSucceeded             PacketName = "SessionAuthenticationSucceededPacket"
	PacketNameSessionAuthenticationFailed                PacketName = "SessionAuthenticationFailedPacket"
	PacketNameInitializeSpeechToText                     PacketName = "InitializeSpeechToTextPacket"
	PacketNameInitializeAssistantExecutor                PacketName = "InitializeAssistantExecutorPacket"
	PacketNameInitializeTextToSpeech                     PacketName = "InitializeTextToSpeechPacket"
	PacketNameInitializeVoiceActivityDetection           PacketName = "InitializeVoiceActivityDetectionPacket"
	PacketNameInitializeEndOfSpeech                      PacketName = "InitializeEndOfSpeechPacket"
	PacketNameInitializeDenoise                          PacketName = "InitializeDenoisePacket"
	PacketNameInitializeBehavior                         PacketName = "InitializeBehaviorPacket"
	PacketNameInitializationCompleted                    PacketName = "InitializationCompletedPacket"
	PacketNameInitializeInboundDispatcher                PacketName = "InitializeInboundDispatcherPacket"
	PacketNameFinalizeInboundDispatcher                  PacketName = "FinalizeInboundDispatcherPacket"
	PacketNameInitializationFailed                       PacketName = "InitializationFailedPacket"
	PacketNameModeSwitchRequested                        PacketName = "ModeSwitchRequestedPacket"
	PacketNameModeSwitchCompleted                        PacketName = "ModeSwitchCompletedPacket"
	PacketNameModeSwitchError                            PacketName = "ModeSwitchErrorPacket"
	PacketNameModeSwitchInitializeSpeechToText           PacketName = "ModeSwitchInitializeSpeechToTextPacket"
	PacketNameModeSwitchInitializeTextToSpeech           PacketName = "ModeSwitchInitializeTextToSpeechPacket"
	PacketNameModeSwitchInitializeVoiceActivityDetection PacketName = "ModeSwitchInitializeVoiceActivityDetectionPacket"
	PacketNameModeSwitchInitializeEndOfSpeech            PacketName = "ModeSwitchInitializeEndOfSpeechPacket"
	PacketNameModeSwitchInitializeDenoise                PacketName = "ModeSwitchInitializeDenoisePacket"
	PacketNameModeSwitchFinalizeSpeechToText             PacketName = "ModeSwitchFinalizeSpeechToTextPacket"
	PacketNameModeSwitchFinalizeTextToSpeech             PacketName = "ModeSwitchFinalizeTextToSpeechPacket"
	PacketNameModeSwitchFinalizeVoiceActivityDetection   PacketName = "ModeSwitchFinalizeVoiceActivityDetectionPacket"
	PacketNameModeSwitchFinalizeEndOfSpeech              PacketName = "ModeSwitchFinalizeEndOfSpeechPacket"
	PacketNameModeSwitchFinalizeDenoise                  PacketName = "ModeSwitchFinalizeDenoisePacket"
	PacketNameFinalizeBehavior                           PacketName = "FinalizeBehaviorPacket"
	PacketNameFinalizeEndOfSpeech                        PacketName = "FinalizeEndOfSpeechPacket"
	PacketNameFinalizeVoiceActivityDetection             PacketName = "FinalizeVoiceActivityDetectionPacket"
	PacketNameFinalizeTextToSpeech                       PacketName = "FinalizeTextToSpeechPacket"
	PacketNameFinalizeSpeechToText                       PacketName = "FinalizeSpeechToTextPacket"
	PacketNameFinalizeAuthentication                     PacketName = "FinalizeAuthenticationPacket"
	PacketNameFinalizeConversationRecordingExecutor      PacketName = "FinalizeConversationRecordingExecutorPacket"
	PacketNameFinalizeSessionRuntime                     PacketName = "FinalizeSessionRuntimePacket"
	PacketNameFinalizeArtifactPushExecutor               PacketName = "FinalizeArtifactPushExecutorPacket"
	PacketNameExecuteAnalysis                            PacketName = "ExecuteAnalysisPacket"
	PacketNameFinalizeConversation                       PacketName = "FinalizeConversationPacket"
	PacketNameFinalizeAnalysisExecutor                   PacketName = "FinalizeAnalysisExecutorPacket"
	PacketNameFinalizeAssistant                          PacketName = "FinalizeAssistantPacket"
	PacketNameFinalizationCompleted                      PacketName = "FinalizationCompletedPacket"
	PacketNameStartIdleTimeout                           PacketName = "StartIdleTimeoutPacket"
	PacketNameStopIdleTimeout                            PacketName = "StopIdleTimeoutPacket"
	PacketNameIdleTimeoutExpired                         PacketName = "IdleTimeoutExpiredPacket"
	PacketNameUnclearInputExpired                        PacketName = "UnclearInputExpiredPacket"
	PacketNameMaxSessionExpired                          PacketName = "MaxSessionExpiredPacket"
	PacketNameLLMResponseDelta                           PacketName = "LLMResponseDeltaPacket"
	PacketNameLLMResponseDone                            PacketName = "LLMResponseDonePacket"
	PacketNameLLMError                                   PacketName = "LLMErrorPacket"
	PacketNameLLMToolCall                                PacketName = "LLMToolCallPacket"
	PacketNameLLMToolResult                              PacketName = "LLMToolResultPacket"
	PacketNameTextToSpeechError                          PacketName = "TextToSpeechErrorPacket"
	PacketNameTextToSpeechText                           PacketName = "TextToSpeechTextPacket"
	PacketNameTextToSpeechDone                           PacketName = "TextToSpeechDonePacket"
	PacketNameTextToSpeechAudio                          PacketName = "TextToSpeechAudioPacket"
	PacketNameTextToSpeechEnd                            PacketName = "TextToSpeechEndPacket"
	PacketNameRecordUserAudio                            PacketName = "RecordUserAudioPacket"
	PacketNameRecordAssistantAudio                       PacketName = "RecordAssistantAudioPacket"
	PacketNameConversationRecordingCompleted             PacketName = "ConversationRecordingCompletedPacket"
	PacketNameMessageCreate                              PacketName = "MessageCreatePacket"
	PacketNameToolLogCreate                              PacketName = "ToolLogCreatePacket"
	PacketNameToolLogUpdate                              PacketName = "ToolLogUpdatePacket"
	PacketNameHTTPLogCreate                              PacketName = "HTTPLogCreatePacket"
	PacketNameObservabilityLogRecord                     PacketName = "ObservabilityLogRecordPacket"
	PacketNameObservabilityEventRecord                   PacketName = "ObservabilityEventRecordPacket"
	PacketNameObservabilityMetricRecord                  PacketName = "ObservabilityMetricRecordPacket"
	PacketNameObservabilityMetadataRecord                PacketName = "ObservabilityMetadataRecordPacket"
	PacketNameObservabilityUsageRecord                   PacketName = "ObservabilityUsageRecordPacket"
	PacketNameObservabilityWebhookRecord                 PacketName = "ObservabilityWebhookRecordPacket"
)

// DispatchAction defines what the dispatcher should do with packets matching a
// dispatch policy target.
type DispatchAction string

const (
	// DispatchActionPassthrough is the default behavior. Matching packets are
	// dispatched to their handlers normally. Setting passthrough removes any
	// stored policy for the target.
	DispatchActionPassthrough DispatchAction = "passthrough"

	// DispatchActionIgnore drops matching packets before handler execution.
	DispatchActionIgnore DispatchAction = "ignore"
)

// DispatchPolicy defines dispatcher behavior for packets matching Target.
type DispatchPolicy struct {
	Target PacketName
	Action DispatchAction
}

// DispatchPolicyPacket sets the active dispatch policy for its target.
type DispatchPolicyPacket struct {
	ContextID string
	Policy    DispatchPolicy
}

func (p DispatchPolicyPacket) ContextId() string      { return p.ContextID }
func (p DispatchPolicyPacket) PacketName() PacketName { return PacketNameDispatchPolicy }

// MessagePacket wraps a Packet with role and text content.
type MessagePacket interface {
	Packet
	Role() string
	Content() string
}

// AudioPacket wraps a Packet with raw audio bytes.
type AudioPacket interface {
	Packet
	Content() []byte
}

// LLMPacket is a marker interface for LLM pipeline packets.
type LLMPacket interface {
	Packet
	ContextId() string
}

// LLMToolPacket is a marker interface for LLM tool-related packets.
type LLMToolPacket interface {
	ToolId() string
}

type ErrorPacket interface {
	Packet
	IsRecoverable() bool
	Err() error
	ErrMessage() string
}

type MessageEntry struct {
	Role    string
	Content string
}

// =============================================================================
// Input Pipeline — user -> denoise -> VAD -> STT -> EOS -> normalize
// =============================================================================

// UserTextReceivedPacket carries text input from the user (e.g. via WebSocket/HTTP).
type UserTextReceivedPacket struct {
	ContextID string
	Text      string

	// Language detected by STT for this turn (may be empty for text-mode input).
	Language string
}

func (f UserTextReceivedPacket) ContextId() string      { return f.ContextID }
func (f UserTextReceivedPacket) PacketName() PacketName { return PacketNameUserTextReceived }
func (f UserTextReceivedPacket) Content() string        { return f.Text }
func (f UserTextReceivedPacket) Role() string           { return "user" }

// UserAudioReceivedPacket carries raw audio input from the user (e.g. via WebRTC).
type UserAudioReceivedPacket struct {
	ContextID string
	Audio     []byte
}

func (f UserAudioReceivedPacket) ContextId() string      { return f.ContextID }
func (f UserAudioReceivedPacket) PacketName() PacketName { return PacketNameUserAudioReceived }
func (f UserAudioReceivedPacket) Content() []byte        { return f.Audio }
func (f UserAudioReceivedPacket) Role() string           { return "user" }

type SpeechToTextAudioPacket struct {
	ContextID string
	Audio     []byte
}

func (f SpeechToTextAudioPacket) ContextId() string      { return f.ContextID }
func (f SpeechToTextAudioPacket) PacketName() PacketName { return PacketNameSpeechToTextAudio }
func (f SpeechToTextAudioPacket) Content() []byte        { return f.Audio }
func (f SpeechToTextAudioPacket) IsAsync() bool          { return true }

// DenoiseAudioPacket carries raw user audio to be denoised before entering the pipeline.
type DenoiseAudioPacket struct {
	ContextID string
	Audio     []byte
}

func (f DenoiseAudioPacket) ContextId() string      { return f.ContextID }
func (f DenoiseAudioPacket) PacketName() PacketName { return PacketNameDenoiseAudio }

// DenoisedAudioPacket carries the result of the denoiser stage.
// The denoiser pushes this via onPacket instead of returning bytes to the caller.
// On error the denoiser falls back to the original audio with NoiseReduced=false.
type DenoisedAudioPacket struct {
	ContextID  string
	Audio      []byte
	Confidence float64
}

func (f DenoisedAudioPacket) ContextId() string      { return f.ContextID }
func (f DenoisedAudioPacket) PacketName() PacketName { return PacketNameDenoisedAudio }

// VadAudioPacket carries a processed audio chunk to submit to the VAD processor.
type VadAudioPacket struct {
	ContextID string
	Audio     []byte
}

func (f VadAudioPacket) ContextId() string      { return f.ContextID }
func (f VadAudioPacket) PacketName() PacketName { return PacketNameVadAudio }

// SpeechToTextPacket carries a transcript result from the STT provider.
type SpeechToTextPacket struct {
	ContextID  string
	Script     string
	Concat     *string
	Confidence float64
	Language   string
	Interim    bool
	Latency    time.Duration
}

func (f SpeechToTextPacket) ContextId() string      { return f.ContextID }
func (f SpeechToTextPacket) PacketName() PacketName { return PacketNameSpeechToText }

func (f SpeechToTextPacket) GetConcat() string {
	if f.Concat == nil {
		return " "
	}
	return *f.Concat
}

// EndOfSpeechPacket signals that the EOS detector determined the user's turn is complete.
type EndOfSpeechAudioPacket struct {
	ContextID string
	Audio     []byte
}

func (f EndOfSpeechAudioPacket) ContextId() string      { return f.ContextID }
func (f EndOfSpeechAudioPacket) PacketName() PacketName { return PacketNameEndOfSpeechAudio }
func (f EndOfSpeechAudioPacket) IsAsync() bool          { return true }

type EndOfSpeechInterruptionPacket struct {
	ContextID string
	Source    InterruptionSource
}

func (f EndOfSpeechInterruptionPacket) ContextId() string { return f.ContextID }
func (f EndOfSpeechInterruptionPacket) PacketName() PacketName {
	return PacketNameEndOfSpeechInterruption
}
func (f EndOfSpeechInterruptionPacket) IsAsync() bool { return true }

type EndOfSpeechPacket struct {
	ContextID string
	Speech    string
	Speechs   []SpeechToTextPacket // accumulated transcript chunks
}

func (f EndOfSpeechPacket) ContextId() string      { return f.ContextID }
func (f EndOfSpeechPacket) PacketName() PacketName { return PacketNameEndOfSpeech }

// InterimEndOfSpeechPacket carries a partial EOS result (in-progress transcript).
type InterimEndOfSpeechPacket struct {
	ContextID string
	Speech    string
}

func (p InterimEndOfSpeechPacket) ContextId() string      { return p.ContextID }
func (p InterimEndOfSpeechPacket) PacketName() PacketName { return PacketNameInterimEndOfSpeech }

// UserInputPacket carries the processed user text after input preprocessing (language detection, etc.).
type UserInputPacket struct {
	ContextID string
	Text      string
	Language  types.Language
}

func (f UserInputPacket) ContextId() string      { return f.ContextID }
func (f UserInputPacket) PacketName() PacketName { return PacketNameUserInput }

// =============================================================================
// Control — interrupts, directives, injected messages
// =============================================================================

type InterruptionSource string

const (
	InterruptionSourceWord InterruptionSource = "word"
	InterruptionSourceVad  InterruptionSource = "vad"
)

type InterruptionEvent string

const (
	InterruptionEventStart InterruptionEvent = "start"
	InterruptionEventEnd   InterruptionEvent = "end"
)

// InterruptionDetectedPacket signals that an interruption was detected (by VAD or word-level STT).
// Dispatch handles this event by emitting InterruptTTSPacket and InterruptLLMPacket commands.
type InterruptionDetectedPacket struct {
	ContextID string
	Source    InterruptionSource
	Event     InterruptionEvent
	StartAt   float64
	EndAt     float64
}

func (f InterruptionDetectedPacket) ContextId() string      { return f.ContextID }
func (f InterruptionDetectedPacket) PacketName() PacketName { return PacketNameInterruptionDetected }

// TextToSpeechInterruptPacket signals the TTS transformer to stop current playback.
type TextToSpeechInterruptPacket struct {
	ContextID string
	StartAt   float64
	EndAt     float64
}

func (f TextToSpeechInterruptPacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechInterruptPacket) PacketName() PacketName { return PacketNameTextToSpeechInterrupt }
func (f TextToSpeechInterruptPacket) IsAsync() bool          { return true }

type STTErrorType int

const (
	STTRateLimit = 1
	STTNetworkTimeout

	// Non-Recoverable STT errors (e.g., bad API keys, invalid audio formats)
	STTAuthentication
	STTInvalidInput
	STTSystemPanic
)

// When IsRecoverable is true, the conversation should be gracefully terminated.
type SpeechToTextErrorPacket struct {
	ContextID string
	Error     error
	Type      STTErrorType
}

func (f SpeechToTextErrorPacket) ContextId() string      { return f.ContextID }
func (f SpeechToTextErrorPacket) PacketName() PacketName { return PacketNameSpeechToTextError }
func (f SpeechToTextErrorPacket) IsRecoverable() bool {
	return f.Type == STTRateLimit || f.Type == STTNetworkTimeout
}
func (f SpeechToTextErrorPacket) Err() error         { return f.Error }
func (f SpeechToTextErrorPacket) ErrMessage() string { return fmt.Sprintf("stt: %s", f.Error.Error()) }

type SpeechToTextEndPacket struct {
	ContextID string
	StartAt   float64
	EndAt     float64
}

func (f SpeechToTextEndPacket) ContextId() string      { return f.ContextID }
func (f SpeechToTextEndPacket) PacketName() PacketName { return PacketNameSpeechToTextEnd }

type SpeechToTextStartPacket struct {
	ContextID string
	StartAt   float64
	EndAt     float64
}

func (f SpeechToTextStartPacket) ContextId() string      { return f.ContextID }
func (f SpeechToTextStartPacket) PacketName() PacketName { return PacketNameSpeechToTextStart }

// InterruptLLMPacket signals the LLM executor to cancel current generation.
type LLMInterruptPacket struct {
	ContextID string
}

func (f LLMInterruptPacket) ContextId() string      { return f.ContextID }
func (f LLMInterruptPacket) PacketName() PacketName { return PacketNameLLMInterrupt }

// TurnChangePacket notifies components that active context changed to a new turn.
type TurnChangePacket struct {
	ContextID         string
	PreviousContextID string
	Reason            string
	Source            string
	Time              time.Time
}

func (f TurnChangePacket) ContextId() string      { return f.ContextID }
func (f TurnChangePacket) PacketName() PacketName { return PacketNameTurnChange }

// InjectMessagePacket injects a pre-written message (greeting, error, idle timeout) into the pipeline.
type InjectMessagePacket struct {
	ContextID string
	Text      string
}

func (f InjectMessagePacket) ContextId() string      { return f.ContextID }
func (f InjectMessagePacket) PacketName() PacketName { return PacketNameInjectMessage }
func (f InjectMessagePacket) Content() string        { return f.Text }
func (f InjectMessagePacket) Role() string           { return "rapida" }

// =============================================================================
// Initialization chain:
// InitializeAssistant → InitializeConversation → InitializeSessionRuntime
// → InitializeAuthentication → InitializeSpeechToText
// → InitializeTextToSpeech → InitializeVoiceActivityDetection
// → InitializeEndOfSpeech → InitializeBehavior → InitializationCompleted
// Each handler enqueues the next phase to bootstrapChannel, forming an ordered chain.
// =============================================================================

// InitializeAssistantPacket loads the assistant config, starts dispatchers, inits auth executor.
type InitializeAssistantPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeAssistantPacket) ContextId() string      { return f.ContextID }
func (f InitializeAssistantPacket) PacketName() PacketName { return PacketNameInitializeAssistant }

// InitializeConversationPacket creates or resumes the conversation.
type InitializeConversationPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeConversationPacket) ContextId() string { return f.ContextID }
func (f InitializeConversationPacket) PacketName() PacketName {
	return PacketNameInitializeConversation
}

// InitializeSessionRuntimePacket initializes collectors, recorder, normalizers, metrics.
type InitializeSessionRuntimePacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeSessionRuntimePacket) ContextId() string { return f.ContextID }
func (f InitializeSessionRuntimePacket) PacketName() PacketName {
	return PacketNameInitializeSessionRuntime
}

// InitializeConversationRecordingExecutorPacket initializes the conversation recording executor.
type InitializeConversationRecordingExecutorPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeConversationRecordingExecutorPacket) ContextId() string { return f.ContextID }
func (f InitializeConversationRecordingExecutorPacket) PacketName() PacketName {
	return PacketNameInitializeConversationRecordingExecutor
}

// InitializeArtifactPushExecutorPacket initializes external artifact push executors.
type InitializeArtifactPushExecutorPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeArtifactPushExecutorPacket) ContextId() string { return f.ContextID }
func (f InitializeArtifactPushExecutorPacket) PacketName() PacketName {
	return PacketNameInitializeArtifactPushExecutor
}

// InitializeAnalysisExecutorPacket initializes post-conversation analysis executors.
type InitializeAnalysisExecutorPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeAnalysisExecutorPacket) ContextId() string { return f.ContextID }
func (f InitializeAnalysisExecutorPacket) PacketName() PacketName {
	return PacketNameInitializeAnalysisExecutor
}

// InitializeAuthenticationPacket initializes the session authentication executor.
type InitializeAuthenticationPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeAuthenticationPacket) ContextId() string { return f.ContextID }
func (f InitializeAuthenticationPacket) PacketName() PacketName {
	return PacketNameInitializeAuthentication
}

// ExecuteAuthenticationPacket executes session authentication.
type ExecuteAuthenticationPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f ExecuteAuthenticationPacket) ContextId() string { return f.ContextID }
func (f ExecuteAuthenticationPacket) PacketName() PacketName {
	return PacketNameExecuteAuthentication
}

// SessionAuthenticationSucceededPacket carries successful auth output.
// Authenticated can be false when fail_behavior=allow is applied.
type SessionAuthenticationSucceededPacket struct {
	ContextID      string
	Authenticated  bool
	Arguments      map[string]interface{}
	Metadata       map[string]interface{}
	Options        map[string]interface{}
	Initialization *protos.ConversationInitialization
}

func (f SessionAuthenticationSucceededPacket) ContextId() string { return f.ContextID }
func (f SessionAuthenticationSucceededPacket) PacketName() PacketName {
	return PacketNameSessionAuthenticationSucceeded
}

// SessionAuthenticationFailedPacket signals auth stage failure.
type SessionAuthenticationFailedPacket struct {
	ContextID      string
	Error          error
	Initialization *protos.ConversationInitialization
}

func (f SessionAuthenticationFailedPacket) ContextId() string { return f.ContextID }
func (f SessionAuthenticationFailedPacket) PacketName() PacketName {
	return PacketNameSessionAuthenticationFailed
}
func (f SessionAuthenticationFailedPacket) IsRecoverable() bool { return false }
func (f SessionAuthenticationFailedPacket) Err() error          { return f.Error }
func (f SessionAuthenticationFailedPacket) ErrMessage() string {
	return fmt.Sprintf("session_authentication: %s", f.Error.Error())
}

// InitializeSpeechToTextPacket initializes speech-to-text.
type InitializeSpeechToTextPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeSpeechToTextPacket) ContextId() string { return f.ContextID }
func (f InitializeSpeechToTextPacket) PacketName() PacketName {
	return PacketNameInitializeSpeechToText
}

type InitializeAssistantExecutorPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeAssistantExecutorPacket) ContextId() string { return f.ContextID }
func (f InitializeAssistantExecutorPacket) PacketName() PacketName {
	return PacketNameInitializeAssistantExecutor
}

// InitializeTextToSpeechPacket initializes text-to-speech.
type InitializeTextToSpeechPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeTextToSpeechPacket) ContextId() string { return f.ContextID }
func (f InitializeTextToSpeechPacket) PacketName() PacketName {
	return PacketNameInitializeTextToSpeech
}

// InitializeVoiceActivityDetectionPacket initializes voice activity detection.
type InitializeVoiceActivityDetectionPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeVoiceActivityDetectionPacket) IsAsync() bool     { return true }
func (f InitializeVoiceActivityDetectionPacket) ContextId() string { return f.ContextID }
func (f InitializeVoiceActivityDetectionPacket) PacketName() PacketName {
	return PacketNameInitializeVoiceActivityDetection
}

// InitializeEndOfSpeechPacket initializes end-of-speech detection.
type InitializeEndOfSpeechPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeEndOfSpeechPacket) IsAsync() bool          { return true }
func (f InitializeEndOfSpeechPacket) ContextId() string      { return f.ContextID }
func (f InitializeEndOfSpeechPacket) PacketName() PacketName { return PacketNameInitializeEndOfSpeech }

// InitializeDenoisePacket initializes the denoiser for text->audio switch.
type InitializeDenoisePacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeDenoisePacket) IsAsync() bool          { return true }
func (f InitializeDenoisePacket) ContextId() string      { return f.ContextID }
func (f InitializeDenoisePacket) PacketName() PacketName { return PacketNameInitializeDenoise }

// InitializeBehaviorPacket sets up greeting, idle timeout, max session.
type InitializeBehaviorPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializeBehaviorPacket) ContextId() string      { return f.ContextID }
func (f InitializeBehaviorPacket) PacketName() PacketName { return PacketNameInitializeBehavior }

// InitializationCompletedPacket is emitted when the connect initialization chain succeeds.
type InitializationCompletedPacket struct {
	ContextID string
	Config    *protos.ConversationInitialization
}

func (f InitializationCompletedPacket) ContextId() string { return f.ContextID }
func (f InitializationCompletedPacket) PacketName() PacketName {
	return PacketNameInitializationCompleted
}

// AsyncPacket marks a packet whose handler runs in its own goroutine.
type AsyncPacket interface {
	Packet
	IsAsync() bool
}

// InitializeInboundDispatcherPacket starts the ingress dispatcher.
type InitializeInboundDispatcherPacket struct {
	ContextID string
}

func (p InitializeInboundDispatcherPacket) ContextId() string { return p.ContextID }
func (p InitializeInboundDispatcherPacket) PacketName() PacketName {
	return PacketNameInitializeInboundDispatcher
}

// FinalizeInboundDispatcherPacket stops accepting queued ingress packets before
// component finalizers close VAD, STT, EOS, and denoise resources.
type FinalizeInboundDispatcherPacket struct {
	ContextID string
}

func (p FinalizeInboundDispatcherPacket) ContextId() string { return p.ContextID }
func (p FinalizeInboundDispatcherPacket) PacketName() PacketName {
	return PacketNameFinalizeInboundDispatcher
}

// InitializationStage identifies which initialization phase failed.
type InitializationStage string

const (
	InitializationStageAssistant           InitializationStage = "assistant"
	InitializationStageConversation        InitializationStage = "conversation"
	InitializationStageService             InitializationStage = "service"
	InitializationStageAuthentication      InitializationStage = "authentication"
	InitializationStageSpeechToText        InitializationStage = "stt"
	InitializationStageTextToSpeech        InitializationStage = "tts"
	InitializationStageVoiceActivity       InitializationStage = "vad"
	InitializationStageDenoise             InitializationStage = "denoise"
	InitializationStageEndOfSpeech         InitializationStage = "eos"
	InitializationStageBehavior            InitializationStage = "behavior"
	InitializationStageAnalysis            InitializationStage = "analysis"
	InitializationStageRecording           InitializationStage = "recording"
	InitializationStageInputNormalizer     InitializationStage = "input_normalizer"
	InitializationStageOutputNormalizer    InitializationStage = "output_normalizer"
	InitializationStageInitializationFinal InitializationStage = "initialization_completed"
)

// InitializationFailedPacket signals that initialization failed.
// The handler notifies the client and fires conversation error handling.
type InitializationFailedPacket struct {
	ContextID string
	Stage     InitializationStage
	Error     error
}

func (f InitializationFailedPacket) ContextId() string      { return f.ContextID }
func (f InitializationFailedPacket) PacketName() PacketName { return PacketNameInitializationFailed }
func (f InitializationFailedPacket) IsRecoverable() bool    { return false }
func (f InitializationFailedPacket) Err() error             { return f.Error }
func (f InitializationFailedPacket) ErrMessage() string {
	if f.Stage != "" {
		return fmt.Sprintf("init[%s]: %s", string(f.Stage), f.Error.Error())
	}
	return fmt.Sprintf("init: %s", f.Error.Error())
}

// =============================================================================
// Runtime mode switch chain (text <-> audio).
// Uses dedicated mode-switch stage packets so switch flow remains independent
// from bootstrap/finalization chains.
// =============================================================================

type ModeSwitchRequestedPacket struct {
	ContextID   string
	StreamMode  protos.StreamMode
	RequestedAt time.Time
}

func (f ModeSwitchRequestedPacket) ContextId() string      { return f.ContextID }
func (f ModeSwitchRequestedPacket) PacketName() PacketName { return PacketNameModeSwitchRequested }

type ModeSwitchCompletedPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchCompletedPacket) ContextId() string      { return f.ContextID }
func (f ModeSwitchCompletedPacket) PacketName() PacketName { return PacketNameModeSwitchCompleted }

type ModeSwitchErrorType string

const (
	ModeSwitchErrorTypeUnknown                          ModeSwitchErrorType = "unknown"
	ModeSwitchErrorTypePreconditionNotReady             ModeSwitchErrorType = "precondition_not_ready"
	ModeSwitchErrorTypeUnsupportedMode                  ModeSwitchErrorType = "unsupported_mode"
	ModeSwitchErrorTypeInitializeSpeechToText           ModeSwitchErrorType = "initialize_stt"
	ModeSwitchErrorTypeInitializeTextToSpeech           ModeSwitchErrorType = "initialize_tts"
	ModeSwitchErrorTypeInitializeVoiceActivityDetection ModeSwitchErrorType = "initialize_vad"
	ModeSwitchErrorTypeInitializeEndOfSpeech            ModeSwitchErrorType = "initialize_eos"
	ModeSwitchErrorTypeFinalizeEndOfSpeech              ModeSwitchErrorType = "finalize_eos"
	ModeSwitchErrorTypeInitializeDenoise                ModeSwitchErrorType = "initialize_denoise"
	ModeSwitchErrorTypeFinalizeDenoise                  ModeSwitchErrorType = "finalize_denoise"
	ModeSwitchErrorTypeFinalizeVoiceActivityDetection   ModeSwitchErrorType = "finalize_vad"
	ModeSwitchErrorTypeFinalizeTextToSpeech             ModeSwitchErrorType = "finalize_tts"
	ModeSwitchErrorTypeFinalizeSpeechToText             ModeSwitchErrorType = "finalize_stt"
)

type ModeSwitchErrorPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
	Type       ModeSwitchErrorType
	Error      error
}

func (f ModeSwitchErrorPacket) ContextId() string      { return f.ContextID }
func (f ModeSwitchErrorPacket) PacketName() PacketName { return PacketNameModeSwitchError }
func (f ModeSwitchErrorPacket) IsRecoverable() bool {
	return f.Type == ModeSwitchErrorTypeUnknown ||
		f.Type == ModeSwitchErrorTypePreconditionNotReady ||
		f.Type == ModeSwitchErrorTypeUnsupportedMode
}
func (f ModeSwitchErrorPacket) Err() error { return f.Error }
func (f ModeSwitchErrorPacket) ErrMessage() string {
	msg := "unknown error"
	if f.Error != nil {
		msg = f.Error.Error()
	}
	return fmt.Sprintf("mode_switch[%s:%s]: %s", string(f.Type), f.StreamMode.String(), msg)
}

// ModeSwitchInitializeSpeechToTextPacket initializes STT for text->audio switch.
// Sync — runs serially on the bootstrap goroutine; on success emits the next
// packet in the chain, on failure emits ModeSwitchErrorPacket (non-recoverable).
type ModeSwitchInitializeSpeechToTextPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchInitializeSpeechToTextPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchInitializeSpeechToTextPacket) PacketName() PacketName {
	return PacketNameModeSwitchInitializeSpeechToText
}
func (f ModeSwitchInitializeSpeechToTextPacket) IsAsync() bool { return true }

// ModeSwitchInitializeTextToSpeechPacket initializes TTS for text->audio switch.
type ModeSwitchInitializeTextToSpeechPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchInitializeTextToSpeechPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchInitializeTextToSpeechPacket) PacketName() PacketName {
	return PacketNameModeSwitchInitializeTextToSpeech
}

// ModeSwitchInitializeVoiceActivityDetectionPacket initializes VAD for text->audio switch.
type ModeSwitchInitializeVoiceActivityDetectionPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchInitializeVoiceActivityDetectionPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchInitializeVoiceActivityDetectionPacket) PacketName() PacketName {
	return PacketNameModeSwitchInitializeVoiceActivityDetection
}
func (f ModeSwitchInitializeVoiceActivityDetectionPacket) IsAsync() bool { return true }

// ModeSwitchInitializeEndOfSpeechPacket initializes EOS for text->audio switch.
type ModeSwitchInitializeEndOfSpeechPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchInitializeEndOfSpeechPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchInitializeEndOfSpeechPacket) PacketName() PacketName {
	return PacketNameModeSwitchInitializeEndOfSpeech
}

// ModeSwitchInitializeDenoisePacket initializes the denoiser for text->audio switch.
type ModeSwitchInitializeDenoisePacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchInitializeDenoisePacket) ContextId() string { return f.ContextID }
func (f ModeSwitchInitializeDenoisePacket) PacketName() PacketName {
	return PacketNameModeSwitchInitializeDenoise
}
func (f ModeSwitchInitializeDenoisePacket) IsAsync() bool { return true }

// ModeSwitchFinalizeSpeechToTextPacket finalizes STT for audio->text switch.
// Async — runs in its own goroutine. Fire-and-forget; the client has already
// been confirmed in text mode by the time these handlers run. Errors are logged.
type ModeSwitchFinalizeSpeechToTextPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchFinalizeSpeechToTextPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchFinalizeSpeechToTextPacket) PacketName() PacketName {
	return PacketNameModeSwitchFinalizeSpeechToText
}
func (f ModeSwitchFinalizeSpeechToTextPacket) IsAsync() bool { return true }

// ModeSwitchFinalizeTextToSpeechPacket finalizes TTS for audio->text switch.
type ModeSwitchFinalizeTextToSpeechPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchFinalizeTextToSpeechPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchFinalizeTextToSpeechPacket) PacketName() PacketName {
	return PacketNameModeSwitchFinalizeTextToSpeech
}
func (f ModeSwitchFinalizeTextToSpeechPacket) IsAsync() bool { return true }

// ModeSwitchFinalizeVoiceActivityDetectionPacket finalizes VAD for audio->text switch.
type ModeSwitchFinalizeVoiceActivityDetectionPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchFinalizeVoiceActivityDetectionPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchFinalizeVoiceActivityDetectionPacket) PacketName() PacketName {
	return PacketNameModeSwitchFinalizeVoiceActivityDetection
}
func (f ModeSwitchFinalizeVoiceActivityDetectionPacket) IsAsync() bool { return true }

// ModeSwitchFinalizeEndOfSpeechPacket finalizes EOS for audio->text switch.
type ModeSwitchFinalizeEndOfSpeechPacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchFinalizeEndOfSpeechPacket) ContextId() string { return f.ContextID }
func (f ModeSwitchFinalizeEndOfSpeechPacket) PacketName() PacketName {
	return PacketNameModeSwitchFinalizeEndOfSpeech
}
func (f ModeSwitchFinalizeEndOfSpeechPacket) IsAsync() bool { return true }

// ModeSwitchFinalizeDenoisePacket finalizes the denoiser for audio->text switch.
type ModeSwitchFinalizeDenoisePacket struct {
	ContextID  string
	StreamMode protos.StreamMode
}

func (f ModeSwitchFinalizeDenoisePacket) ContextId() string { return f.ContextID }
func (f ModeSwitchFinalizeDenoisePacket) PacketName() PacketName {
	return PacketNameModeSwitchFinalizeDenoise
}
func (f ModeSwitchFinalizeDenoisePacket) IsAsync() bool { return true }

// =============================================================================
// Finalization chain:
// FinalizeInboundDispatcher → FinalizeBehavior → FinalizeEndOfSpeech
// → FinalizeVoiceActivityDetection → FinalizeTextToSpeech → FinalizeSpeechToText
// → FinalizeAuthentication → FinalizeConversationRecordingExecutor
// → FinalizeSessionRuntime → FinalizeArtifactPushExecutor → ExecuteAnalysis
// → FinalizeConversation → FinalizeAnalysisExecutor → FinalizeAssistant
// → FinalizationCompleted
// Each handler enqueues the next phase to backgroundCh, forming an ordered chain.
// =============================================================================

// FinalizeBehaviorPacket finalizes session behavior controls (timers).
type FinalizeBehaviorPacket struct {
	ContextID string
}

func (f FinalizeBehaviorPacket) ContextId() string      { return f.ContextID }
func (f FinalizeBehaviorPacket) PacketName() PacketName { return PacketNameFinalizeBehavior }

// FinalizeEndOfSpeechPacket finalizes end-of-speech processing.
type FinalizeEndOfSpeechPacket struct {
	ContextID string
}

func (f FinalizeEndOfSpeechPacket) ContextId() string      { return f.ContextID }
func (f FinalizeEndOfSpeechPacket) PacketName() PacketName { return PacketNameFinalizeEndOfSpeech }

// FinalizeVoiceActivityDetectionPacket finalizes VAD processing.
type FinalizeVoiceActivityDetectionPacket struct {
	ContextID string
}

func (f FinalizeVoiceActivityDetectionPacket) ContextId() string { return f.ContextID }
func (f FinalizeVoiceActivityDetectionPacket) PacketName() PacketName {
	return PacketNameFinalizeVoiceActivityDetection
}

// FinalizeTextToSpeechPacket finalizes text-to-speech resources.
type FinalizeTextToSpeechPacket struct {
	ContextID string
}

func (f FinalizeTextToSpeechPacket) ContextId() string      { return f.ContextID }
func (f FinalizeTextToSpeechPacket) PacketName() PacketName { return PacketNameFinalizeTextToSpeech }

// FinalizeSpeechToTextPacket finalizes speech-to-text resources.
type FinalizeSpeechToTextPacket struct {
	ContextID string
}

func (f FinalizeSpeechToTextPacket) ContextId() string      { return f.ContextID }
func (f FinalizeSpeechToTextPacket) PacketName() PacketName { return PacketNameFinalizeSpeechToText }

// FinalizeAuthenticationPacket finalizes session authentication stage.
type FinalizeAuthenticationPacket struct {
	ContextID string
}

func (f FinalizeAuthenticationPacket) ContextId() string { return f.ContextID }
func (f FinalizeAuthenticationPacket) PacketName() PacketName {
	return PacketNameFinalizeAuthentication
}

// FinalizeConversationRecordingExecutorPacket finalizes conversation recording.
type FinalizeConversationRecordingExecutorPacket struct {
	ContextID string
}

func (f FinalizeConversationRecordingExecutorPacket) ContextId() string { return f.ContextID }
func (f FinalizeConversationRecordingExecutorPacket) PacketName() PacketName {
	return PacketNameFinalizeConversationRecordingExecutor
}

// FinalizeSessionRuntimePacket finalizes runtime resources.
type FinalizeSessionRuntimePacket struct {
	ContextID string
}

func (f FinalizeSessionRuntimePacket) ContextId() string { return f.ContextID }
func (f FinalizeSessionRuntimePacket) PacketName() PacketName {
	return PacketNameFinalizeSessionRuntime
}

// FinalizeArtifactPushExecutorPacket finalizes external artifact push executors.
type FinalizeArtifactPushExecutorPacket struct {
	ContextID string
}

func (f FinalizeArtifactPushExecutorPacket) ContextId() string { return f.ContextID }
func (f FinalizeArtifactPushExecutorPacket) PacketName() PacketName {
	return PacketNameFinalizeArtifactPushExecutor
}

// ExecuteAnalysisPacket executes post-conversation analysis.
type ExecuteAnalysisPacket struct {
	ContextID string
}

func (f ExecuteAnalysisPacket) ContextId() string      { return f.ContextID }
func (f ExecuteAnalysisPacket) PacketName() PacketName { return PacketNameExecuteAnalysis }

// FinalizeConversationPacket finalizes conversation-level collectors/events.
type FinalizeConversationPacket struct {
	ContextID string
}

func (f FinalizeConversationPacket) ContextId() string      { return f.ContextID }
func (f FinalizeConversationPacket) PacketName() PacketName { return PacketNameFinalizeConversation }

// FinalizeAnalysisExecutorPacket finalizes post-conversation analysis executors.
type FinalizeAnalysisExecutorPacket struct {
	ContextID string
}

func (f FinalizeAnalysisExecutorPacket) ContextId() string { return f.ContextID }
func (f FinalizeAnalysisExecutorPacket) PacketName() PacketName {
	return PacketNameFinalizeAnalysisExecutor
}

// FinalizeAssistantPacket finalizes assistant runtime resources.
type FinalizeAssistantPacket struct {
	ContextID string
}

func (f FinalizeAssistantPacket) ContextId() string      { return f.ContextID }
func (f FinalizeAssistantPacket) PacketName() PacketName { return PacketNameFinalizeAssistant }

// FinalizationCompletedPacket marks terminal completion of disconnect flow.
type FinalizationCompletedPacket struct {
	ContextID string
}

func (f FinalizationCompletedPacket) ContextId() string      { return f.ContextID }
func (f FinalizationCompletedPacket) PacketName() PacketName { return PacketNameFinalizationCompleted }

// StartIdleTimeoutPacket explicitly (re)starts the idle timeout timer.
// Routed on outputCh so producers can order it relative to InjectMessagePacket
// and TTS output packets that share the same channel.
type StartIdleTimeoutPacket struct {
	ContextID string
}

func (f StartIdleTimeoutPacket) ContextId() string      { return f.ContextID }
func (f StartIdleTimeoutPacket) PacketName() PacketName { return PacketNameStartIdleTimeout }

// StopIdleTimeoutPacket explicitly stops the idle timeout timer.
// ResetCount = true also clears the consecutive idle backoff counter
// (used when the user actively engages, not for system-driven stops).
type StopIdleTimeoutPacket struct {
	ContextID  string
	ResetCount bool
}

func (f StopIdleTimeoutPacket) ContextId() string      { return f.ContextID }
func (f StopIdleTimeoutPacket) PacketName() PacketName { return PacketNameStopIdleTimeout }

// IdleTimeoutExpiredPacket signals that the idle timeout watchdog expired.
type IdleTimeoutExpiredPacket struct {
	ContextID string
	Count     uint64
}

func (f IdleTimeoutExpiredPacket) ContextId() string      { return f.ContextID }
func (f IdleTimeoutExpiredPacket) PacketName() PacketName { return PacketNameIdleTimeoutExpired }

// UnclearInputExpiredPacket signals that an interrupted turn did not produce accepted user input.
type UnclearInputExpiredPacket struct {
	ContextID string
}

func (f UnclearInputExpiredPacket) ContextId() string      { return f.ContextID }
func (f UnclearInputExpiredPacket) PacketName() PacketName { return PacketNameUnclearInputExpired }

// MaxSessionExpiredPacket signals that the max-session watchdog expired.
type MaxSessionExpiredPacket struct {
	ContextID string
}

func (f MaxSessionExpiredPacket) ContextId() string      { return f.ContextID }
func (f MaxSessionExpiredPacket) PacketName() PacketName { return PacketNameMaxSessionExpired }

// =============================================================================
// LLM Pipeline — execute -> delta -> done -> error -> tools
// =============================================================================

// LLMResponseDeltaPacket represents a streaming text delta from the LLM.
type LLMResponseDeltaPacket struct {
	ContextID string
	Text      string
}

func (f LLMResponseDeltaPacket) ContextId() string      { return f.ContextID }
func (f LLMResponseDeltaPacket) PacketName() PacketName { return PacketNameLLMResponseDelta }

// LLMResponseDonePacket signals the completion of an LLM response stream.
type LLMResponseDonePacket struct {
	ContextID string
	Text      string
}

func (f LLMResponseDonePacket) Content() string        { return f.Text }
func (f LLMResponseDonePacket) Role() string           { return "assistant" }
func (f LLMResponseDonePacket) ContextId() string      { return f.ContextID }
func (f LLMResponseDonePacket) PacketName() PacketName { return PacketNameLLMResponseDone }

// LLMErrorPacket signals that the LLM encountered an error during generation.

type LLMErrorType int

const (
	// UnknownError is the default zero-value fallback
	UnknownError LLMErrorType = iota

	// Recoverable errors (e.g., API rate limits, temporary network drops)
	LLMRateLimit
	LLMNetworkTimeout

	// Non-Recoverable LLMors (e.g., bad API keys, invalid prompt formats)
	LLMAuthentication
	LLMInvalidInput
	LLMSystemPanic
)

// When IsRecoverable is true, the conversation should be gracefully terminated.
type LLMErrorPacket struct {
	ContextID string
	Error     error
	Type      LLMErrorType
}

func (f LLMErrorPacket) ContextId() string      { return f.ContextID }
func (f LLMErrorPacket) PacketName() PacketName { return PacketNameLLMError }
func (f LLMErrorPacket) IsRecoverable() bool {
	return f.Type != LLMAuthentication && f.Type != LLMSystemPanic
}
func (f LLMErrorPacket) Err() error         { return f.Error }
func (f LLMErrorPacket) ErrMessage() string { return fmt.Sprintf("llm: %s", f.Error.Error()) }

// LLMToolCallPacket signals that a tool was invoked.
// Action determines whether the client/channel needs to act (e.g. end call, transfer).
type LLMToolCallPacket struct {
	ToolID    string
	Name      string
	ContextID string
	Action    protos.ToolCallAction
	Arguments map[string]string
}

func (f LLMToolCallPacket) ContextId() string      { return f.ContextID }
func (f LLMToolCallPacket) PacketName() PacketName { return PacketNameLLMToolCall }
func (f LLMToolCallPacket) ToolId() string         { return f.ToolID }

func (f LLMToolCallPacket) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"tool_id":    f.ToolID,
		"name":       f.Name,
		"context_id": f.ContextID,
		"action":     f.Action.String(),
		"arguments":  f.Arguments,
	})
}

func (f *LLMToolCallPacket) UnmarshalJSON(data []byte) error {
	var raw struct {
		ToolID    string            `json:"tool_id"`
		Name      string            `json:"name"`
		ContextID string            `json:"context_id"`
		Action    string            `json:"action"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.ToolID = raw.ToolID
	f.Name = raw.Name
	f.ContextID = raw.ContextID
	f.Action = protos.ToolCallAction(protos.ToolCallAction_value[raw.Action])
	f.Arguments = raw.Arguments
	return nil
}

// LLMToolResultPacket carries the result of a tool execution.
// Arrives from server-side tools (immediate) or from client/channel (directive).
type LLMToolResultPacket struct {
	ToolID    string
	Name      string
	ContextID string
	Action    protos.ToolCallAction
	Result    map[string]string
}

func (f LLMToolResultPacket) ToolId() string         { return f.ToolID }
func (f LLMToolResultPacket) ContextId() string      { return f.ContextID }
func (f LLMToolResultPacket) PacketName() PacketName { return PacketNameLLMToolResult }

func (f LLMToolResultPacket) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"tool_id":    f.ToolID,
		"name":       f.Name,
		"context_id": f.ContextID,
		"action":     f.Action.String(),
		"result":     f.Result,
	})
}

func (f *LLMToolResultPacket) UnmarshalJSON(data []byte) error {
	var raw struct {
		ToolID    string            `json:"tool_id"`
		Name      string            `json:"name"`
		ContextID string            `json:"context_id"`
		Action    string            `json:"action"`
		Result    map[string]string `json:"result"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.ToolID = raw.ToolID
	f.Name = raw.Name
	f.ContextID = raw.ContextID
	f.Action = protos.ToolCallAction(protos.ToolCallAction_value[raw.Action])
	f.Result = raw.Result
	return nil
}

// =============================================================================
// Output Pipeline — aggregate -> speak -> TTS audio -> TTS end
// =============================================================================

type TTSErrorType int

const (
	TTSUnknownError TTSErrorType = iota

	// Recoverable
	TTSRateLimit
	TTSNetworkTimeout

	// Non-Recoverable
	TTSAuthentication
	TTSInvalidInput
	TTSSystemPanic
)

type TextToSpeechErrorPacket struct {
	ContextID string
	Error     error
	Type      TTSErrorType
}

func (f TextToSpeechErrorPacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechErrorPacket) PacketName() PacketName { return PacketNameTextToSpeechError }
func (f TextToSpeechErrorPacket) IsRecoverable() bool {
	return f.Type != TTSAuthentication
}
func (f TextToSpeechErrorPacket) Err() error         { return f.Error }
func (f TextToSpeechErrorPacket) ErrMessage() string { return fmt.Sprintf("tts: %s", f.Error.Error()) }

// TextToSpeechTextPacket carries a sentence-ready text chunk for TTS synthesis.
type TextToSpeechTextPacket struct {
	ContextID string
	Text      string
}

func (f TextToSpeechTextPacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechTextPacket) PacketName() PacketName { return PacketNameTextToSpeechText }

// TextToSpeechDonePacket signals end of this turn's output. TTS flushes remaining audio.
type TextToSpeechDonePacket struct {
	ContextID string
	Text      string
}

func (f TextToSpeechDonePacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechDonePacket) PacketName() PacketName { return PacketNameTextToSpeechDone }

// TextToSpeechAudioPacket carries a TTS audio chunk produced by the TTS provider.
type TextToSpeechAudioPacket struct {
	ContextID  string
	AudioChunk []byte
}

func (f TextToSpeechAudioPacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechAudioPacket) PacketName() PacketName { return PacketNameTextToSpeechAudio }

// TextToSpeechEndPacket signals that TTS has finished producing audio.
type TextToSpeechEndPacket struct {
	ContextID string
}

func (f TextToSpeechEndPacket) ContextId() string      { return f.ContextID }
func (f TextToSpeechEndPacket) PacketName() PacketName { return PacketNameTextToSpeechEnd }

// =============================================================================
// Recording
// =============================================================================

// RecordUserAudioPacket carries a user audio chunk to be written to the recorder.
type RecordUserAudioPacket struct {
	ContextID string
	Audio     []byte
	Timestamp time.Time
}

func (f RecordUserAudioPacket) ContextId() string      { return f.ContextID }
func (f RecordUserAudioPacket) PacketName() PacketName { return PacketNameRecordUserAudio }

// RecordAssistantAudioPacket carries an assistant audio chunk to the recorder.
type RecordAssistantAudioPacket struct {
	ContextID string
	Audio     []byte
	Timestamp time.Time
}

func (f RecordAssistantAudioPacket) ContextId() string      { return f.ContextID }
func (f RecordAssistantAudioPacket) PacketName() PacketName { return PacketNameRecordAssistantAudio }

// ConversationRecordingCompletedPacket carries finalized recording audio for persistence.
type ConversationRecordingCompletedPacket struct {
	ContextID string
	Audio     ConversationRecordingAudio
}

func (f ConversationRecordingCompletedPacket) ContextId() string { return f.ContextID }
func (f ConversationRecordingCompletedPacket) PacketName() PacketName {
	return PacketNameConversationRecordingCompleted
}

// =============================================================================
// Persistence
// =============================================================================

// MessageCreatePacket persists a conversation message to the database and appends
// it to the in-memory history. It implements MessagePacket so it can be passed
// directly to onCreateMessage.
type MessageCreatePacket struct {
	ContextID   string
	MessageRole string
	Text        string
}

func (f MessageCreatePacket) ContextId() string      { return f.ContextID }
func (f MessageCreatePacket) PacketName() PacketName { return PacketNameMessageCreate }
func (f MessageCreatePacket) Role() string           { return f.MessageRole }
func (f MessageCreatePacket) Content() string        { return f.Text }

// ToolLogCreatePacket persists a tool call start to the database.
type ToolLogCreatePacket struct {
	ContextID string
	ToolID    string
	Name      string
	Request   []byte
}

func (f ToolLogCreatePacket) ContextId() string      { return f.ContextID }
func (f ToolLogCreatePacket) PacketName() PacketName { return PacketNameToolLogCreate }

// ToolLogUpdatePacket persists a tool call result to the database.
type ToolLogUpdatePacket struct {
	ContextID string
	ToolID    string
	Response  []byte
}

func (f ToolLogUpdatePacket) ContextId() string      { return f.ContextID }
func (f ToolLogUpdatePacket) PacketName() PacketName { return PacketNameToolLogUpdate }

// HTTPLogCreatePacket emits a generic request log for HTTP-backed execution.
type HTTPLogCreatePacket struct {
	ContextID       string
	Source          string
	SourceRefID     uint64
	SourceEvent     string
	HTTPURL         string
	HTTPMethod      string
	ResponseStatus  int64
	TimeTaken       int64
	RetryCount      uint32
	Status          type_enums.RecordState
	ErrorMessage    *string
	RequestPayload  []byte
	ResponsePayload []byte
}

func (f HTTPLogCreatePacket) ContextId() string      { return f.ContextID }
func (f HTTPLogCreatePacket) PacketName() PacketName { return PacketNameHTTPLogCreate }
func (f HTTPLogCreatePacket) IsAsync() bool          { return true }

// =============================================================================
// Observability
// =============================================================================

// ObservabilityRecordPacket is the base packet for emitting a typed observability record.
type ObservabilityRecordScope string

const (
	ObservabilityRecordScopeAssistant        ObservabilityRecordScope = "assistant"
	ObservabilityRecordScopeConversation     ObservabilityRecordScope = "conversation"
	ObservabilityRecordScopeUserMessage      ObservabilityRecordScope = "user-message"
	ObservabilityRecordScopeAssistantMessage ObservabilityRecordScope = "assistant-message"
)

func (t ObservabilityRecordScope) String() string {
	return string(t)
}

type ObservabilityRecordPacket interface {
	Packet
	GetScope() ObservabilityRecordScope
	GetRecord() observability.Record
}

// ObservabilityLogRecordPacket emits an observability.RecordLog.
type ObservabilityLogRecordPacket struct {
	ContextID string
	Scope     ObservabilityRecordScope
	Record    observability.RecordLog
}

func (p ObservabilityLogRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityLogRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityLogRecord
}
func (p ObservabilityLogRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}

func (p ObservabilityLogRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// ObservabilityEventRecordPacket emits an observability.RecordEvent.
type ObservabilityEventRecordPacket struct {
	ContextID string
	Scope     ObservabilityRecordScope
	Record    observability.RecordEvent
}

func (p ObservabilityEventRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityEventRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityEventRecord
}
func (p ObservabilityEventRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}

func (p ObservabilityEventRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// ObservabilityMetricRecordPacket emits an observability.RecordMetric.
type ObservabilityMetricRecordPacket struct {
	ContextID string
	Scope     ObservabilityRecordScope
	Record    observability.RecordMetric
}

func (p ObservabilityMetricRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityMetricRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityMetricRecord
}
func (p ObservabilityMetricRecordPacket) IsAsync() bool { return true }
func (p ObservabilityMetricRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}
func (p ObservabilityMetricRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// ObservabilityMetadataRecordPacket emits an observability.RecordMetadata.
type ObservabilityMetadataRecordPacket struct {
	ContextID string
	Scope     ObservabilityRecordScope
	Record    observability.RecordMetadata
}

func (p ObservabilityMetadataRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityMetadataRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityMetadataRecord
}
func (p ObservabilityMetadataRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}

func (p ObservabilityMetadataRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// ObservabilityUsageRecordPacket emits an observability.RecordUsage.
type ObservabilityUsageRecordPacket struct {
	ContextID   string
	Scope       ObservabilityRecordScope
	MessageRole observability.MessageRole
	Record      observability.RecordUsage
}

func (p ObservabilityUsageRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityUsageRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityUsageRecord
}
func (p ObservabilityUsageRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}
func (p ObservabilityUsageRecordPacket) GetMessageRole() observability.MessageRole {
	return p.MessageRole
}
func (p ObservabilityUsageRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// ObservabilityWebhookRecordPacket emits an observability.RecordWebhook.
type ObservabilityWebhookRecordPacket struct {
	ContextID   string
	Scope       ObservabilityRecordScope
	MessageRole observability.MessageRole
	Record      observability.RecordWebhook
}

func (p ObservabilityWebhookRecordPacket) ContextId() string { return p.ContextID }
func (p ObservabilityWebhookRecordPacket) PacketName() PacketName {
	return PacketNameObservabilityWebhookRecord
}
func (p ObservabilityWebhookRecordPacket) GetScope() ObservabilityRecordScope {
	return p.Scope
}
func (p ObservabilityWebhookRecordPacket) GetMessageRole() observability.MessageRole {
	return p.MessageRole
}
func (p ObservabilityWebhookRecordPacket) GetRecord() observability.Record {
	return p.Record
}

// =============================================================================
// Non-packet Support Types
// =============================================================================

// KnowledgeRetrieveOption contains options for knowledge retrieval operations.
type KnowledgeRetrieveOption struct {
	EmbeddingProviderCredential *protos.VaultCredential
	RetrievalMethod             string
	TopK                        uint32
	ScoreThreshold              float32
}

// KnowledgeContextResult holds a single knowledge retrieval result.
type KnowledgeContextResult struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
}
