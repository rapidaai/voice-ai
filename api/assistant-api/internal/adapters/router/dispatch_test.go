package router

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type dispatchHandlerStub struct {
	calledUserText                  bool
	calledConversationRecordingDone bool
}

func (s *dispatchHandlerStub) HandleUserText(context.Context, internal_type.UserTextReceivedPacket) {
	s.calledUserText = true
}
func (s *dispatchHandlerStub) HandleUserAudio(context.Context, internal_type.UserAudioReceivedPacket) {
}
func (s *dispatchHandlerStub) HandleDenoise(context.Context, internal_type.DenoiseAudioPacket) {}
func (s *dispatchHandlerStub) HandleDenoisedAudio(context.Context, internal_type.DenoisedAudioPacket) {
}
func (s *dispatchHandlerStub) HandleVadAudio(context.Context, internal_type.VadAudioPacket) {}
func (s *dispatchHandlerStub) HandleVadSpeechActivity(context.Context, internal_type.VadSpeechActivityPacket) {
}
func (s *dispatchHandlerStub) HandleSpeechToText(context.Context, internal_type.SpeechToTextPacket) {}
func (s *dispatchHandlerStub) HandleInterimEndOfSpeech(context.Context, internal_type.InterimEndOfSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleEndOfSpeech(context.Context, internal_type.EndOfSpeechPacket) {}
func (s *dispatchHandlerStub) HandleUserInput(context.Context, internal_type.UserInputPacket)     {}
func (s *dispatchHandlerStub) HandleInterruptionDetected(context.Context, internal_type.InterruptionDetectedPacket) {
}
func (s *dispatchHandlerStub) HandleTextToSpeechInterrupt(context.Context, internal_type.TextToSpeechInterruptPacket) {
}
func (s *dispatchHandlerStub) HandleLLMInterrupt(context.Context, internal_type.LLMInterruptPacket) {}
func (s *dispatchHandlerStub) HandleDispatchPolicy(context.Context, internal_type.DispatchPolicyPacket) {
}
func (s *dispatchHandlerStub) HandleSpeechToTextEnd(context.Context, internal_type.SpeechToTextEndPacket) {
}
func (s *dispatchHandlerStub) HandleSpeechToTextStart(context.Context, internal_type.SpeechToTextStartPacket) {
}
func (s *dispatchHandlerStub) HandleTurnChange(context.Context, internal_type.TurnChangePacket) {}
func (s *dispatchHandlerStub) HandleLLMResponseDelta(context.Context, internal_type.LLMResponseDeltaPacket) {
}
func (s *dispatchHandlerStub) HandleLLMResponseDone(context.Context, internal_type.LLMResponseDonePacket) {
}
func (s *dispatchHandlerStub) HandleError(context.Context, internal_type.ErrorPacket) {}
func (s *dispatchHandlerStub) HandleInjectMessage(context.Context, internal_type.InjectMessagePacket) {
}
func (s *dispatchHandlerStub) HandleStartIdleTimeout(context.Context, internal_type.StartIdleTimeoutPacket) {
}
func (s *dispatchHandlerStub) HandleStopIdleTimeout(context.Context, internal_type.StopIdleTimeoutPacket) {
}
func (s *dispatchHandlerStub) HandleIdleTimeoutExpired(context.Context, internal_type.IdleTimeoutExpiredPacket) {
}
func (s *dispatchHandlerStub) HandleUnclearInputExpired(context.Context, internal_type.UnclearInputExpiredPacket) {
}
func (s *dispatchHandlerStub) HandleMaxSessionExpired(context.Context, internal_type.MaxSessionExpiredPacket) {
}
func (s *dispatchHandlerStub) HandleTextToSpeechText(context.Context, internal_type.TextToSpeechTextPacket) {
}
func (s *dispatchHandlerStub) HandleTextToSpeechDone(context.Context, internal_type.TextToSpeechDonePacket) {
}
func (s *dispatchHandlerStub) HandleTextToSpeechAudio(context.Context, internal_type.TextToSpeechAudioPacket) {
}
func (s *dispatchHandlerStub) HandleTextToSpeechEnd(context.Context, internal_type.TextToSpeechEndPacket) {
}
func (s *dispatchHandlerStub) HandleLLMToolCall(context.Context, internal_type.LLMToolCallPacket) {}
func (s *dispatchHandlerStub) HandleLLMToolResult(context.Context, internal_type.LLMToolResultPacket) {
}
func (s *dispatchHandlerStub) HandleRecordUserAudio(context.Context, internal_type.RecordUserAudioPacket) {
}
func (s *dispatchHandlerStub) HandleRecordAssistantAudio(context.Context, internal_type.RecordAssistantAudioPacket) {
}
func (s *dispatchHandlerStub) HandleConversationRecordingCompleted(context.Context, internal_type.ConversationRecordingCompletedPacket) {
	s.calledConversationRecordingDone = true
}
func (s *dispatchHandlerStub) HandleMessageCreate(context.Context, internal_type.MessageCreatePacket) {
}
func (s *dispatchHandlerStub) HandleToolLogCreate(context.Context, internal_type.ToolLogCreatePacket) {
}
func (s *dispatchHandlerStub) HandleToolLogUpdate(context.Context, internal_type.ToolLogUpdatePacket) {
}
func (s *dispatchHandlerStub) HandleHTTPLogCreate(context.Context, internal_type.HTTPLogCreatePacket) {
}
func (s *dispatchHandlerStub) HandleInitializeAssistant(context.Context, internal_type.InitializeAssistantPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeConversation(context.Context, internal_type.InitializeConversationPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeSessionRuntime(context.Context, internal_type.InitializeSessionRuntimePacket) {
}
func (s *dispatchHandlerStub) HandleInitializeConversationRecordingExecutor(context.Context, internal_type.InitializeConversationRecordingExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeArtifactPushExecutor(context.Context, internal_type.InitializeArtifactPushExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeAnalysisExecutor(context.Context, internal_type.InitializeAnalysisExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeAuthentication(context.Context, internal_type.InitializeAuthenticationPacket) {
}
func (s *dispatchHandlerStub) HandleExecuteAuthentication(context.Context, internal_type.ExecuteAuthenticationPacket) {
}
func (s *dispatchHandlerStub) HandleSessionAuthenticationSucceeded(context.Context, internal_type.SessionAuthenticationSucceededPacket) {
}
func (s *dispatchHandlerStub) HandleSessionAuthenticationFailed(context.Context, internal_type.SessionAuthenticationFailedPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeSpeechToText(context.Context, internal_type.InitializeSpeechToTextPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeTextToSpeech(context.Context, internal_type.InitializeTextToSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeVoiceActivityDetection(context.Context, internal_type.InitializeVoiceActivityDetectionPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeEndOfSpeech(context.Context, internal_type.InitializeEndOfSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeDenoise(context.Context, internal_type.InitializeDenoisePacket) {
}
func (s *dispatchHandlerStub) HandleInitializeAssistantExecutorPacket(context.Context, internal_type.InitializeAssistantExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeBehavior(context.Context, internal_type.InitializeBehaviorPacket) {
}
func (s *dispatchHandlerStub) HandleInitializationCompleted(context.Context, internal_type.InitializationCompletedPacket) {
}
func (s *dispatchHandlerStub) HandleInitializeInboundDispatcher(context.Context, internal_type.InitializeInboundDispatcherPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeInboundDispatcher(context.Context, internal_type.FinalizeInboundDispatcherPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchRequested(context.Context, internal_type.ModeSwitchRequestedPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchCompleted(context.Context, internal_type.ModeSwitchCompletedPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchInitializeSpeechToText(context.Context, internal_type.ModeSwitchInitializeSpeechToTextPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchInitializeTextToSpeech(context.Context, internal_type.ModeSwitchInitializeTextToSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchInitializeVoiceActivityDetection(context.Context, internal_type.ModeSwitchInitializeVoiceActivityDetectionPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchInitializeEndOfSpeech(context.Context, internal_type.ModeSwitchInitializeEndOfSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchInitializeDenoise(context.Context, internal_type.ModeSwitchInitializeDenoisePacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchFinalizeEndOfSpeech(context.Context, internal_type.ModeSwitchFinalizeEndOfSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchFinalizeDenoise(context.Context, internal_type.ModeSwitchFinalizeDenoisePacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchFinalizeVoiceActivityDetection(context.Context, internal_type.ModeSwitchFinalizeVoiceActivityDetectionPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchFinalizeTextToSpeech(context.Context, internal_type.ModeSwitchFinalizeTextToSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleModeSwitchFinalizeSpeechToText(context.Context, internal_type.ModeSwitchFinalizeSpeechToTextPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeBehavior(context.Context, internal_type.FinalizeBehaviorPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeEndOfSpeech(context.Context, internal_type.FinalizeEndOfSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeVoiceActivityDetection(context.Context, internal_type.FinalizeVoiceActivityDetectionPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeTextToSpeech(context.Context, internal_type.FinalizeTextToSpeechPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeSpeechToText(context.Context, internal_type.FinalizeSpeechToTextPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeAuthentication(context.Context, internal_type.FinalizeAuthenticationPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeConversationRecordingExecutor(context.Context, internal_type.FinalizeConversationRecordingExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeSessionRuntime(context.Context, internal_type.FinalizeSessionRuntimePacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeArtifactPushExecutor(context.Context, internal_type.FinalizeArtifactPushExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleExecuteAnalysis(context.Context, internal_type.ExecuteAnalysisPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeConversation(context.Context, internal_type.FinalizeConversationPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeAnalysisExecutor(context.Context, internal_type.FinalizeAnalysisExecutorPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizeAssistant(context.Context, internal_type.FinalizeAssistantPacket) {
}
func (s *dispatchHandlerStub) HandleFinalizationCompleted(context.Context, internal_type.FinalizationCompletedPacket) {
}
func (s *dispatchHandlerStub) HandleSpeechToTextAudio(context.Context, internal_type.SpeechToTextAudioPacket) {
}
func (s *dispatchHandlerStub) HandleEndOfSpeechInterruption(context.Context, internal_type.EndOfSpeechInterruptionPacket) {
}
func (s *dispatchHandlerStub) HandleEndOfSpeechAudio(context.Context, internal_type.EndOfSpeechAudioPacket) {
}
func (s *dispatchHandlerStub) HandleObservabilityRecordPacket(context.Context, internal_type.ObservabilityRecordPacket) {
}

func TestDispatchPacket_DispatchesKnownPacket(t *testing.T) {
	handler := &dispatchHandlerStub{}

	err := DispatchPacket(context.Background(), internal_type.UserTextReceivedPacket{ContextID: "c", Text: "hi"}, handler)
	if err != nil {
		t.Fatalf("expected known packet to not return an error, got: %v", err)
	}
	if !handler.calledUserText {
		t.Fatalf("expected HandleUserText to be called")
	}
}

func TestDispatchPacket_DispatchesConversationRecordingCompleted(t *testing.T) {
	handler := &dispatchHandlerStub{}

	err := DispatchPacket(context.Background(), internal_type.ConversationRecordingCompletedPacket{ContextID: "c"}, handler)
	if err != nil {
		t.Fatalf("expected known packet to not return an error, got: %v", err)
	}
	if !handler.calledConversationRecordingDone {
		t.Fatalf("expected HandleConversationRecordingCompleted to be called")
	}
}

func TestDispatchPacket_UnknownPacketReturnsFalse(t *testing.T) {
	var handler DispatchHandler
	err := DispatchPacket(context.Background(), unroutedPacket{contextID: "c"}, handler)
	if err == nil {
		t.Fatalf("expected unknown packet to return an error")
	}
}
