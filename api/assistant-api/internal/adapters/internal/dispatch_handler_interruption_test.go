package adapter_internal

import (
	"context"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInterruptionTestRequestor(trigger string) *genericRequestor {
	options := map[string]interface{}{}
	if trigger != "" {
		options[internal_options.MicrophoneOptionBargeInTrigger] = trigger
	}

	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-active", type_enums.AudioMode)
	_ = lifecycle.AssistantGenerating("ctx-active")
	_ = lifecycle.AssistantSpeaking("ctx-active")

	return &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		options:          options,
		vadExecutor:      &blockingVADExecutor{},
	}
}

func newUnclearInputTestRequestor(trigger string, timeout float64, message string) *genericRequestor {
	r := newInterruptionTestRequestor(trigger)
	r.source = utils.PhoneCall
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				UnclearInputTimeout: &timeout,
				UnclearInputMessage: &message,
			},
		},
	}
	r.unclearInputWatchdog = watchdog.NewUnclearInputWatchdog(watchdog.WithOnPacket(r.OnPacket))
	drainBackgroundPackets(r)
	return r
}

func drainControlPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, len(r.channels.ControlChannel()))
	for len(r.channels.ControlChannel()) > 0 {
		packets = append(packets, (<-r.channels.ControlChannel()).Pkt)
	}
	return packets
}

func drainEgressPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, len(r.channels.EgressChannel()))
	for len(r.channels.EgressChannel()) > 0 {
		packets = append(packets, (<-r.channels.EgressChannel()).Pkt)
	}
	return packets
}

func drainIngressPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, len(r.channels.IngressChannel()))
	for len(r.channels.IngressChannel()) > 0 {
		packets = append(packets, (<-r.channels.IngressChannel()).Pkt)
	}
	return packets
}

func drainBackgroundPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, len(r.channels.BackgroundChannel()))
	for len(r.channels.BackgroundChannel()) > 0 {
		packets = append(packets, (<-r.channels.BackgroundChannel()).Pkt)
	}
	return packets
}

func waitForUnclearInputExpired(t *testing.T, r *genericRequestor) internal_type.UnclearInputExpiredPacket {
	t.Helper()

	var expired internal_type.UnclearInputExpiredPacket
	require.Eventually(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = typed
			}
		}
		return expired.ContextID != ""
	}, time.Second, 10*time.Millisecond)
	return expired
}

func TestHandleUserText_TextModeRotatesWordInterruptionBeforeEOS(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-active", type_enums.TextMode)
	require.NoError(t, lifecycle.AssistantGenerating("ctx-active"))
	require.NoError(t, lifecycle.AssistantSpeaking("ctx-active"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-active",
		Text:      "interrupt with text",
	})

	newContextID := r.GetID()
	require.NotEqual(t, "ctx-active", newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, newContextID, interim.ContextID)
	assert.Equal(t, "interrupt with text", interim.Speech)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, newContextID, eos.ContextID)
	assert.Equal(t, "interrupt with text", eos.Speech)
}

func TestHandleUserText_TextModeRotatesAfterPreviousUserFinished(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-first", type_enums.TextMode)
	require.NoError(t, lifecycle.UserFinished("ctx-first"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-first",
		Text:      "second text",
	})

	secondContextID := r.GetID()
	require.NotEqual(t, "ctx-first", secondContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, secondContextID, interim.ContextID)
	assert.Equal(t, "second text", interim.Speech)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, secondContextID, eos.ContextID)
	assert.Equal(t, "second text", eos.Speech)
}

func TestHandleUserText_TextModeDoesNotRotateWhileUserSpeaking(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-user-speaking", type_enums.TextMode)
	require.NoError(t, lifecycle.AssistantGenerating("ctx-user-speaking"))
	require.NoError(t, lifecycle.BeginInterrupt("ctx-user-speaking"))
	require.NoError(t, lifecycle.UserSpeaking("ctx-user-speaking"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-user-speaking",
		Text:      "same turn text",
	})

	assert.Equal(t, "ctx-user-speaking", r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, "ctx-user-speaking", interim.ContextID)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, "ctx-user-speaking", eos.ContextID)
}

func TestHandleUserText_TextModeDoesNotRotateWhenAlreadyInterrupted(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-interrupt", type_enums.TextMode)
	require.NoError(t, lifecycle.AssistantGenerating("ctx-interrupt"))
	require.NoError(t, lifecycle.BeginInterrupt("ctx-interrupt"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-interrupt",
		Text:      "same interrupt text",
	})

	assert.Equal(t, "ctx-interrupt", r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, "ctx-interrupt", interim.ContextID)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, "ctx-interrupt", eos.ContextID)
}

func TestHandleInterruptionDetected_TextModeIgnoresStaleWordInterruption(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-current", type_enums.TextMode)
	require.NoError(t, lifecycle.AssistantGenerating("ctx-current"))
	require.NoError(t, lifecycle.AssistantSpeaking("ctx-current"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-stale",
		Source:    internal_type.InterruptionSourceWord,
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleUserInput_DropsStaleContext(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-current", type_enums.TextMode)
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleUserInput(context.Background(), internal_type.UserInputPacket{
		ContextID: "ctx-stale",
		Text:      "old input",
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleSpeechToText_FinalRotatesAfterPreviousUserFinished(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-first", type_enums.AudioMode)
	require.NoError(t, lifecycle.UserFinished("ctx-first"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "new audio turn",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, "ctx-first", newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-first", eosInterrupt.ContextID)
	assert.Equal(t, "ctx-first", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-first", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-first", stopIdleTimeout.ContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 1)
	eos, ok := ingressPackets[0].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, newContextID, eos.ContextID)
	assert.Equal(t, "new audio turn", eos.Speech)

	h.HandleEndOfSpeech(context.Background(), eos)
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
}

func TestHandleSpeechToText_DoesNotRotateAgainForSameSpeechSegment(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-first", type_enums.AudioMode)
	require.NoError(t, lifecycle.UserFinished("ctx-first"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "Yes. I am.",
		Interim:   false,
	})

	turnContextID := r.GetID()
	require.NotEqual(t, "ctx-first", turnContextID)

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "Yeah.",
		Interim:   false,
	})

	assert.Equal(t, turnContextID, r.GetID())
	executed := executor.snapshotExecuted()
	require.Len(t, executed, 2)
	first, ok := executed[0].(internal_type.SpeechToTextPacket)
	require.True(t, ok)
	second, ok := executed[1].(internal_type.SpeechToTextPacket)
	require.True(t, ok)
	assert.Equal(t, turnContextID, first.ContextID)
	assert.Equal(t, turnContextID, second.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
}

func TestHandleSpeechToText_DropsStaleTranscriptAfterUserFinished(t *testing.T) {
	lifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-current", type_enums.AudioMode)
	require.NoError(t, lifecycle.UserFinished("ctx-current"))
	r := &genericRequestor{
		streamer:         &streamTestStreamer{},
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: lifecycle,
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-old",
		Script:    "late transcript",
		Interim:   false,
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleInterimEndOfSpeech_UsesPacketContextID(t *testing.T) {
	streamer := &streamTestStreamer{}
	r := &genericRequestor{
		streamer:         streamer,
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: adapter_lifecycle.NewMessageLifecycleWithContext("ctx-current", type_enums.AudioMode),
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleInterimEndOfSpeech(context.Background(), internal_type.InterimEndOfSpeechPacket{
		ContextID: "ctx-segment",
		Speech:    "partial text",
	})

	require.Len(t, streamer.sent, 1)
	userMessage, ok := streamer.sent[0].(*protos.ConversationUserMessage)
	require.True(t, ok, "expected ConversationUserMessage, got %T", streamer.sent[0])
	assert.Equal(t, "ctx-segment", userMessage.Id)
	assert.False(t, userMessage.Completed)
}

func TestHandleInterruptionDetected_VADTriggerUsesVADOnly(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.2, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	assert.Equal(t, "ctx-active", r.GetID())
	assert.Empty(t, r.channels.ControlChannel())
	assert.Empty(t, r.channels.EgressChannel())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	userTurnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", userTurnContextID)
	controlPackets := drainControlPackets(r)
	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	var sttStart internal_type.SpeechToTextStartPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		case internal_type.SpeechToTextStartPacket:
			sttStart = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, sttStart.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	var sttEnd internal_type.SpeechToTextEndPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.SpeechToTextEndPacket); ok {
			sttEnd = typed
		}
	}
	require.NotEmpty(t, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerUsesWordOnly(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.2, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	assert.Equal(t, "ctx-active", r.GetID())
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttStart, ok := controlPackets[0].(internal_type.SpeechToTextStartPacket)
	require.True(t, ok, "expected SpeechToTextStartPacket, got %T", controlPackets[0])
	assert.Equal(t, "ctx-active", sttStart.ContextID)
	assert.Empty(t, r.channels.EgressChannel())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	userTurnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", userTurnContextID)
	controlPackets = drainControlPackets(r)
	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	var sttEnd internal_type.SpeechToTextEndPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.SpeechToTextEndPacket); ok {
			sttEnd = typed
		}
	}
	require.NotEmpty(t, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
}

func TestHandleInterruptionDetected_VADTriggerStartsUnclearInputWatchdogAfterVADEnd(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.02, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	drainEgressPackets(r)
	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 40*time.Millisecond, 10*time.Millisecond)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerStartsUnclearInputWatchdogOnlyAfterWord(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.02, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	select {
	case packet := <-r.channels.EgressChannel():
		t.Fatalf("unclear input watchdog started before word interruption: %+v", packet.Pkt)
	case <-time.After(50 * time.Millisecond):
	}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerDuplicateWordExtendsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.06, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: contextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: contextID,
		Source:    internal_type.InterruptionSourceWord,
	})

	select {
	case packet := <-r.channels.EgressChannel():
		t.Fatalf("unclear input watchdog expired before duplicate word extended deadline: %+v", packet.Pkt)
	case <-time.After(35 * time.Millisecond):
	}

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerDuplicateAfterFinalDoesNotRestartUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	drainEgressPackets(r)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: newContextID,
		Source:    internal_type.InterruptionSourceWord,
	})

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleSpeechToText_InterimStopsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.06, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hel",
		Interim:   true,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleEndOfSpeech_StopsUnclearInputWatchdogForAcceptedSpeech(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)
	turnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", turnContextID)
	drainControlPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "hello",
	})

	assert.Equal(t, turnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	for _, packet := range drainControlPackets(r) {
		switch packet.(type) {
		case internal_type.TextToSpeechInterruptPacket, internal_type.LLMInterruptPacket:
			t.Fatalf("unexpected interrupt packet from EOS completion: %T", packet)
		}
	}

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleEndOfSpeech_DoesNotFinalizeWhileVADSpeaking(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	turnContextID := r.GetID()
	require.NotEqual(t, oldContextID, turnContextID)
	drainControlPackets(r)
	drainEgressPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "still speaking",
	})

	assert.Equal(t, turnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserSpeaking, r.messageLifecycle.State())
	assert.Empty(t, drainIngressPackets(r))

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainControlPackets(r)
	drainEgressPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "done speaking",
	})

	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 1)
	userInput, ok := ingressPackets[0].(internal_type.UserInputPacket)
	require.True(t, ok, "expected UserInputPacket, got %T", ingressPackets[0])
	assert.Equal(t, turnContextID, userInput.ContextID)
	assert.Equal(t, "done speaking", userInput.Text)
}

func TestHandleSpeechToText_FinalStopsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, oldContextID, ttsInterrupt.ContextID)
	assert.Equal(t, oldContextID, llmInterrupt.ContextID)

	var eos internal_type.EndOfSpeechPacket
	for _, packet := range drainIngressPackets(r) {
		if typed, ok := packet.(internal_type.EndOfSpeechPacket); ok {
			eos = typed
		}
	}
	require.Equal(t, "hello", eos.Speech)
	assert.Equal(t, newContextID, eos.ContextID)
	h.HandleEndOfSpeech(context.Background(), eos)

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleSpeechToText_InterimStartsTurnAndFinalKeepsContext(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.06, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hel",
		Interim:   true,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	assert.Equal(t, newContextID, r.GetID())

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleUnclearInputExpired_InjectsConfiguredMessage(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.BeginInterrupt(contextID))

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	newContextID := r.GetID()
	require.NotEqual(t, contextID, newContextID)

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.TextToSpeechInterruptPacket); ok {
			ttsInterrupt = typed
		}
	}

	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not restart idle timer before assistant completion: %+v", typed)
		}
	}

	assert.Equal(t, contextID, ttsInterrupt.ContextID)
	assert.Equal(t, newContextID, injectMessage.ContextID)
	assert.Equal(t, "Please say that again.", injectMessage.Text)
}

func TestHandleUnclearInputExpired_VADTriggerRotatesFromInterruptedContextAndInjectsNewPrompt(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "I didn't catch that.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	controlPackets := drainControlPackets(r)
	var startEOSInterrupt internal_type.EndOfSpeechInterruptionPacket
	var sttStart internal_type.SpeechToTextStartPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			startEOSInterrupt = typed
		case internal_type.SpeechToTextStartPacket:
			sttStart = typed
		}
	}
	userTurnContextID := r.GetID()
	require.NotEqual(t, oldContextID, userTurnContextID)
	assert.Equal(t, oldContextID, startEOSInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, startEOSInterrupt.Source)
	assert.Equal(t, userTurnContextID, sttStart.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserSpeaking, r.messageLifecycle.State())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	controlPackets = drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttEnd, ok := controlPackets[0].(internal_type.SpeechToTextEndPacket)
	require.True(t, ok)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	drainEgressPackets(r)
	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: userTurnContextID})

	promptContextID := r.GetID()
	require.NotEqual(t, userTurnContextID, promptContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
	assert.Equal(t, uint64(1), r.messageLifecycle.UserPromptCount())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, userTurnContextID, eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, eosInterrupt.Source)
	assert.Equal(t, userTurnContextID, ttsInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.StopIdleTimeoutPacket:
			stopIdleTimeout = typed
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not start idle before assistant completion: %+v", typed)
		}
	}
	assert.Equal(t, userTurnContextID, stopIdleTimeout.ContextID)
	assert.Equal(t, promptContextID, injectMessage.ContextID)
	assert.Equal(t, "I didn't catch that.", injectMessage.Text)
}

func TestHandleUnclearInputExpired_WordTriggerRotatesFromInterruptedContextAndInjectsNewPrompt(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.03, "I didn't catch that.")
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttStart, ok := controlPackets[0].(internal_type.SpeechToTextStartPacket)
	require.True(t, ok)
	assert.Equal(t, oldContextID, sttStart.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	userTurnContextID := r.GetID()
	require.NotEqual(t, oldContextID, userTurnContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
	controlPackets = drainControlPackets(r)
	var wordEOSInterrupt internal_type.EndOfSpeechInterruptionPacket
	for _, packet := range controlPackets {
		if typed, ok := packet.(internal_type.EndOfSpeechInterruptionPacket); ok {
			wordEOSInterrupt = typed
		}
	}
	assert.Equal(t, oldContextID, wordEOSInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, wordEOSInterrupt.Source)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	controlPackets = drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttEnd, ok := controlPackets[0].(internal_type.SpeechToTextEndPacket)
	require.True(t, ok)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)

	drainEgressPackets(r)
	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: userTurnContextID})

	promptContextID := r.GetID()
	require.NotEqual(t, userTurnContextID, promptContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
	assert.Equal(t, uint64(1), r.messageLifecycle.UserPromptCount())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, userTurnContextID, eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, userTurnContextID, ttsInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, llmInterrupt.ContextID)

	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not start idle before assistant completion: %+v", typed)
		}
	}
	assert.Equal(t, promptContextID, injectMessage.ContextID)
	assert.Equal(t, "I didn't catch that.", injectMessage.Text)
}

func TestHandleUnclearInputExpired_IgnoresWhenInterruptionIsNotPending(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.AssistantFinished(contextID))
	require.NoError(t, r.messageLifecycle.AssistantIdle(contextID))

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	assert.Equal(t, contextID, r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
}

func TestHandleIdleTimeoutExpired_InterruptsOldContextAndInjectsOnNewContext(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	idleTimeout := uint64(10)
	message := "Are you still there?"
	r.source = utils.PhoneCall
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				IdleTimeout:        &idleTimeout,
				IdleTimeoutMessage: &message,
			},
		},
	}
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()
	require.NoError(t, r.messageLifecycle.AssistantFinished(oldContextID))
	require.NoError(t, r.messageLifecycle.AssistantIdle(oldContextID))

	h.HandleIdleTimeoutExpired(context.Background(), internal_type.IdleTimeoutExpiredPacket{ContextID: oldContextID})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)
	assert.Equal(t, uint64(1), r.messageLifecycle.AssistantPromptCount())

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.TextToSpeechInterruptPacket); ok {
			ttsInterrupt = typed
		}
	}

	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("idle timeout prompt should not restart idle timer before assistant completion: %+v", typed)
		}
	}

	assert.Equal(t, oldContextID, ttsInterrupt.ContextID)
	assert.Equal(t, newContextID, injectMessage.ContextID)
	assert.Equal(t, message, injectMessage.Text)
}

func TestHandleIdleTimeoutExpired_InjectedPromptSpeaksBeforeIdleRestarts(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	idleTimeout := uint64(10)
	message := "Are you still there?"
	r.source = utils.PhoneCall
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				IdleTimeout:        &idleTimeout,
				IdleTimeoutMessage: &message,
			},
		},
	}
	r.messageLifecycle.SetMode(type_enums.AudioMode)
	r.textToSpeechTransformer = noopSpeechToTextTransformer{}
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()
	require.NoError(t, r.messageLifecycle.AssistantFinished(oldContextID))
	require.NoError(t, r.messageLifecycle.AssistantIdle(oldContextID))

	h.HandleIdleTimeoutExpired(context.Background(), internal_type.IdleTimeoutExpiredPacket{ContextID: oldContextID})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("idle prompt should not restart idle before assistant completion: %+v", typed)
		}
	}
	require.Equal(t, newContextID, injectMessage.ContextID)
	require.Equal(t, message, injectMessage.Text)

	h.HandleInjectMessage(context.Background(), injectMessage)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantGenerating, r.messageLifecycle.State())
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("injected idle prompt should not start idle while generating: %+v", typed)
		}
	}

	h.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: newContextID,
		Text:      message,
	})
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantGenerated, r.messageLifecycle.State())
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("idle prompt should not start idle at LLM done before TTS completion: %+v", typed)
		}
	}

	h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: newContextID,
		Text:      message,
	})
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("audio idle prompt should wait for TTS end before idle restart: %+v", typed)
		}
	}

	h.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: newContextID})

	var startIdleTimeout internal_type.StartIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			startIdleTimeout = typed
		}
	}
	assert.Equal(t, newContextID, startIdleTimeout.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
}

func TestHandleLLMResponseDone_DoesNotStartIdleTimeout(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()

	h.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("LLM done should not start idle timeout before assistant finishes: %+v", typed)
		}
	}
}

func TestHandleTextToSpeechDone_TextModeStartsIdleTimeout(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.AssistantFinished(contextID))
	require.NoError(t, r.messageLifecycle.AssistantIdle(contextID))
	require.NoError(t, r.messageLifecycle.AssistantGenerating(contextID))
	require.NoError(t, r.messageLifecycle.AssistantGenerated(contextID))

	h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	var startIdleTimeout internal_type.StartIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			startIdleTimeout = typed
		}
	}
	assert.Equal(t, contextID, startIdleTimeout.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
}

func TestHandleTextToSpeechDone_AudioModeWaitsForTextToSpeechEndBeforeIdleTimeout(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	r.messageLifecycle.SetMode(type_enums.AudioMode)
	r.textToSpeechTransformer = noopSpeechToTextTransformer{}
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.AssistantFinished(contextID))
	require.NoError(t, r.messageLifecycle.AssistantIdle(contextID))
	require.NoError(t, r.messageLifecycle.AssistantGenerating(contextID))
	require.NoError(t, r.messageLifecycle.AssistantGenerated(contextID))

	h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("audio TTS done should wait for TTS end before idle timeout: %+v", typed)
		}
	}
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())

	h.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})

	var startIdleTimeout internal_type.StartIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			startIdleTimeout = typed
		}
	}
	assert.Equal(t, contextID, startIdleTimeout.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
}

func TestHandleUnclearInputExpired_AudioModeBlocksInputUntilTextToSpeechEnd(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	r.messageLifecycle.SetMode(type_enums.AudioMode)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.BeginInterrupt(contextID))

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	newContextID := r.GetID()
	require.NotEqual(t, contextID, newContextID)
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 6)

	audioPolicy, ok := controlPackets[3].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, audioPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameUserAudioReceived, audioPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, audioPolicy.Policy.Action)

	textPolicy, ok := controlPackets[4].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, textPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameUserTextReceived, textPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, textPolicy.Policy.Action)

	interruptionPolicy, ok := controlPackets[5].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, interruptionPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameInterruptionDetected, interruptionPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, interruptionPolicy.Policy.Action)

	ttsInterrupt, ok := controlPackets[1].(internal_type.TextToSpeechInterruptPacket)
	require.True(t, ok)
	assert.Equal(t, contextID, ttsInterrupt.ContextID)
}
