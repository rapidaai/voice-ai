package adapter_internal

import (
	"context"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
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

func drainControlPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, len(r.channels.ControlChannel()))
	for len(r.channels.ControlChannel()) > 0 {
		packets = append(packets, (<-r.channels.ControlChannel()).Pkt)
	}
	return packets
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
