package adapter_internal

import (
	"context"
	"sync"
	"testing"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingEOSExecutor struct {
	mu           sync.Mutex
	executed     []internal_type.Packet
	closeCalls   int
	lastCloseCtx context.Context
	executeErr   error
	closeErr     error
}

func (e *recordingEOSExecutor) Name() string {
	return "recording-eos"
}

func (e *recordingEOSExecutor) Options() utils.Option {
	return nil
}

func (e *recordingEOSExecutor) Arguments() (map[string]string, error) {
	return nil, nil
}

func (e *recordingEOSExecutor) Execute(_ context.Context, packet internal_type.Packet) error {
	e.mu.Lock()
	e.executed = append(e.executed, packet)
	e.mu.Unlock()
	return e.executeErr
}

func (e *recordingEOSExecutor) Close(ctx context.Context) error {
	e.mu.Lock()
	e.closeCalls++
	e.lastCloseCtx = ctx
	e.mu.Unlock()
	return e.closeErr
}

func (e *recordingEOSExecutor) snapshotExecuted() []internal_type.Packet {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]internal_type.Packet, len(e.executed))
	copy(cp, e.executed)
	return cp
}

func requireSingleInitializationFailedPacket(t *testing.T, r *genericRequestor) internal_type.InitializationFailedPacket {
	t.Helper()

	select {
	case env := <-r.channels.BootstrapChannel():
		pkt, ok := env.Pkt.(internal_type.InitializationFailedPacket)
		require.True(t, ok, "expected InitializationFailedPacket, got %T", env.Pkt)
		return pkt
	default:
		t.Fatal("expected InitializationFailedPacket in bootstrap channel")
		return internal_type.InitializationFailedPacket{}
	}
}

func requireSingleModeSwitchErrorFromEgress(t *testing.T, r *genericRequestor) internal_type.ModeSwitchErrorPacket {
	t.Helper()

	select {
	case env := <-r.channels.EgressChannel():
		pkt, ok := env.Pkt.(internal_type.ModeSwitchErrorPacket)
		require.True(t, ok, "expected ModeSwitchErrorPacket, got %T", env.Pkt)
		return pkt
	default:
		t.Fatal("expected ModeSwitchErrorPacket in egress channel")
		return internal_type.ModeSwitchErrorPacket{}
	}
}

func TestHandleInitializeEndOfSpeech_ConfigError_EmitsInitializationFailed(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	h := requestorDispatchHandler{r: r}

	h.HandleInitializeEndOfSpeech(t.Context(), internal_type.InitializeEndOfSpeechPacket{
		ContextID: "ctx-eos-init-config-error",
	})

	errPkt := requireSingleInitializationFailedPacket(t, r)
	assert.Equal(t, "ctx-eos-init-config-error", errPkt.ContextID)
	assert.Equal(t, internal_type.InitializationStageEndOfSpeech, errPkt.Stage)
	assert.Error(t, errPkt.Error)
	assert.Nil(t, r.endOfSpeechExecutor)
}

func TestHandleModeSwitchInitializeEndOfSpeech_GetEOSError_EmitsModeSwitchError(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
			InputAudio: &internal_assistant_entity.AssistantDeploymentAudio{},
		},
	}
	r.options = map[string]interface{}{
		"microphone.eos.pipecat.model_path": "/tmp/missing-smart-turn-model.onnx",
	}
	h := requestorDispatchHandler{r: r}

	h.HandleModeSwitchInitializeEndOfSpeech(t.Context(), internal_type.ModeSwitchInitializeEndOfSpeechPacket{
		ContextID:  "ctx-mode-switch-eos-error",
		StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
	})

	errPkt := requireSingleModeSwitchErrorFromEgress(t, r)
	assert.Equal(t, "ctx-mode-switch-eos-error", errPkt.ContextID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_AUDIO, errPkt.StreamMode)
	assert.Equal(t, internal_type.ModeSwitchErrorTypeInitializeEndOfSpeech, errPkt.Type)
	assert.Error(t, errPkt.Error)
	assert.Nil(t, r.endOfSpeechExecutor)
}

func TestHandleEndOfSpeechAudio_ExecutesEOS(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleEndOfSpeechAudio(t.Context(), internal_type.EndOfSpeechAudioPacket{
		ContextID: "ctx-eos-audio",
		Audio:     []byte{1, 2, 3},
	})

	executed := executor.snapshotExecuted()
	require.Len(t, executed, 1)
	audioPkt, ok := executed[0].(internal_type.EndOfSpeechAudioPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-eos-audio", audioPkt.ContextID)
}

func TestHandleSpeechToText_WithEOSExecutor_ExecutesAndSkipsFallback(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	r.messageLifecycle = adapter_lifecycle.NewMessageLifecycle()
	r.messageLifecycle.SetContextID("ctx-eos-stt")
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-stt-packet",
		Script:    "hello world",
		Interim:   false,
	})

	executed := executor.snapshotExecuted()
	require.Len(t, executed, 1)
	sttPkt, ok := executed[0].(internal_type.SpeechToTextPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-stt-packet", sttPkt.ContextID)

	select {
	case env := <-r.channels.IngressChannel():
		t.Fatalf("unexpected fallback packet when EOS executor exists: %T", env.Pkt)
	default:
	}
}

func TestHandleSpeechToText_WithoutEOSExecutor_EmitsFallbackOnlyForFinal(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	r.messageLifecycle = adapter_lifecycle.NewMessageLifecycle()
	r.messageLifecycle.SetContextID("ctx-eos-fallback")
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{
		Script:  "interim text",
		Interim: true,
	})

	select {
	case env := <-r.channels.IngressChannel():
		t.Fatalf("unexpected packet for interim STT fallback: %T", env.Pkt)
	default:
	}

	h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-final-packet",
		Script:    "final text",
		Interim:   false,
	})

	select {
	case env := <-r.channels.IngressChannel():
		eosPkt, ok := env.Pkt.(internal_type.EndOfSpeechPacket)
		require.True(t, ok, "expected EndOfSpeechPacket, got %T", env.Pkt)
		assert.Equal(t, "ctx-final-packet", eosPkt.ContextID)
		assert.Equal(t, "final text", eosPkt.Speech)
		require.Len(t, eosPkt.Speechs, 1)
		assert.False(t, eosPkt.Speechs[0].Interim)
	default:
		t.Fatal("expected EndOfSpeech fallback packet for final STT")
	}
}

func TestHandleEndOfSpeechInterruption_ExecutesEOS(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleEndOfSpeechInterruption(t.Context(), internal_type.EndOfSpeechInterruptionPacket{
		ContextID: "ctx-eos-interruption",
		Source:    internal_type.InterruptionSourceVad,
	})

	executed := executor.snapshotExecuted()
	require.Len(t, executed, 1)
	pkt, ok := executed[0].(internal_type.EndOfSpeechInterruptionPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-eos-interruption", pkt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, pkt.Source)
}

func TestHandleFinalizeEndOfSpeech_ClosesExecutorAndEnqueuesNextFinalize(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleFinalizeEndOfSpeech(t.Context(), internal_type.FinalizeEndOfSpeechPacket{
		ContextID: "ctx-eos-finalize",
	})

	assert.Nil(t, r.endOfSpeechExecutor)
	assert.Equal(t, 1, executor.closeCalls)
	assert.NotNil(t, executor.lastCloseCtx)

	select {
	case env := <-r.channels.DataChannel():
		pkt, ok := env.Pkt.(internal_type.FinalizeVoiceActivityDetectionPacket)
		require.True(t, ok, "expected FinalizeVoiceActivityDetectionPacket, got %T", env.Pkt)
		assert.Equal(t, "ctx-eos-finalize", pkt.ContextID)
	default:
		t.Fatal("expected FinalizeVoiceActivityDetectionPacket in data channel")
	}
}
