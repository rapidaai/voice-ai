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
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInterruptionTestRequestor(trigger string) *genericRequestor {
	options := map[string]interface{}{}
	if trigger != "" {
		options[internal_options.MicrophoneVADOptionBargeInTrigger] = trigger
	}

	lifecycle := adapter_lifecycle.NewMessageLifecycle()
	lifecycle.SetContextID("ctx-active")

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

func TestHandleInterruptionDetected_VADTriggerUsesVADOnly(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
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

	require.NotEqual(t, "ctx-active", r.GetID())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var sttStart internal_type.SpeechToTextStartPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket

	require.Eventually(t, func() bool {
		for _, packet := range drainControlPackets(r) {
			switch typed := packet.(type) {
			case internal_type.EndOfSpeechInterruptionPacket:
				eosInterrupt = typed
			case internal_type.SpeechToTextStartPacket:
				sttStart = typed
			case internal_type.TextToSpeechInterruptPacket:
				ttsInterrupt = typed
			case internal_type.LLMInterruptPacket:
				llmInterrupt = typed
			}
		}
		return eosInterrupt.ContextID != "" &&
			sttStart.ContextID != "" &&
			ttsInterrupt.ContextID != "" &&
			llmInterrupt.ContextID != ""
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, internal_type.InterruptionSourceVad, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, r.GetID(), sttStart.ContextID)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

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
	assert.Equal(t, r.GetID(), sttEnd.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerUsesWordOnly(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerWord)
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

	require.NotEqual(t, "ctx-active", r.GetID())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket

	require.Eventually(t, func() bool {
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
		return eosInterrupt.ContextID != "" &&
			ttsInterrupt.ContextID != "" &&
			llmInterrupt.ContextID != ""
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

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
	assert.Equal(t, "ctx-active", sttEnd.ContextID)
}

func TestHandleInterruptionDetected_VADTriggerStartsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.02, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
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

func TestHandleSpeechToText_InterimExtendsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.06, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-active",
		Script:    "hel",
		Interim:   true,
	})

	select {
	case packet := <-r.channels.EgressChannel():
		t.Fatalf("unclear input watchdog expired before extended deadline: %+v", packet.Pkt)
	case <-time.After(35 * time.Millisecond):
	}

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleEndOfSpeech_StopsUnclearInputWatchdogForAcceptedSpeech(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	drainEgressPackets(r)
	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: r.GetID(),
		Speech:    "hello",
	})

	select {
	case packet := <-r.channels.EgressChannel():
		t.Fatalf("unclear input watchdog expired after accepted speech: %+v", packet.Pkt)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestHandleSpeechToText_FinalStopsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	drainEgressPackets(r)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-active",
		Script:    "hello",
		Interim:   false,
	})

	select {
	case packet := <-r.channels.EgressChannel():
		t.Fatalf("unclear input watchdog expired after final speech: %+v", packet.Pkt)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestHandleUnclearInputExpired_InjectsConfiguredMessage(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.TextToSpeechInterruptPacket); ok {
			ttsInterrupt = typed
		}
	}

	var injectMessage internal_type.InjectMessagePacket
	var startIdleTimeout internal_type.StartIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			startIdleTimeout = typed
		}
	}

	assert.Equal(t, contextID, ttsInterrupt.ContextID)
	assert.Equal(t, contextID, injectMessage.ContextID)
	assert.Equal(t, "Please say that again.", injectMessage.Text)
	assert.Equal(t, contextID, startIdleTimeout.ContextID)
}
