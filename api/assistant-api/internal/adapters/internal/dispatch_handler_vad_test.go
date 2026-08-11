package adapter_internal

import (
	"context"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_vad "github.com/rapidaai/api/assistant-api/internal/vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingVADExecutor struct {
	enterCh   chan struct{}
	releaseCh chan struct{}
}

func (b *blockingVADExecutor) Name() string {
	return "blocking-vad"
}

func (b *blockingVADExecutor) Options() utils.Option {
	return nil
}

func (b *blockingVADExecutor) Arguments() (map[string]string, error) {
	return nil, nil
}

func (b *blockingVADExecutor) Execute(context.Context, internal_type.UserAudioReceivedPacket) error {
	close(b.enterCh)
	<-b.releaseCh
	return nil
}

func (b *blockingVADExecutor) Close(context.Context) error {
	return nil
}

func newDispatchHandlerVADTestRequestor(t *testing.T) *genericRequestor {
	t.Helper()

	logger, err := commons.NewApplicationLogger(
		commons.Name("dispatch-handler-vad-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)

	return &genericRequestor{
		logger:   logger,
		channels: adapter_channel.NewRequestorChannels(),
		source:   utils.Debugger,
		options:  map[string]interface{}{},
	}
}

func requireSingleModeSwitchErrorPacket(t *testing.T, r *genericRequestor) internal_type.ModeSwitchErrorPacket {
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

func TestHandleModeSwitchInitializeVoiceActivityDetection_ConfigError_EmitsModeSwitchError(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	h := requestorDispatchHandler{r: r}

	h.HandleModeSwitchInitializeVoiceActivityDetection(t.Context(), internal_type.ModeSwitchInitializeVoiceActivityDetectionPacket{
		ContextID:  "ctx-config-error",
		StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
	})

	errPkt := requireSingleModeSwitchErrorPacket(t, r)
	assert.Equal(t, "ctx-config-error", errPkt.ContextID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_AUDIO, errPkt.StreamMode)
	assert.Equal(t, internal_type.ModeSwitchErrorTypeInitializeVoiceActivityDetection, errPkt.Type)
	assert.Error(t, errPkt.Error)
	assert.Nil(t, r.vadExecutor)
}

func TestHandleModeSwitchInitializeVoiceActivityDetection_VADError_EmitsModeSwitchError(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
			InputAudio: &internal_assistant_entity.AssistantDeploymentAudio{},
		},
	}
	r.options = map[string]interface{}{
		internal_vad.OptionsKeyVadProvider:             internal_vad.SILERO_VAD,
		internal_options.MicrophoneVADOptionConfidence: 1.5, // invalid: detector requires (0,1)
	}
	h := requestorDispatchHandler{r: r}

	h.HandleModeSwitchInitializeVoiceActivityDetection(t.Context(), internal_type.ModeSwitchInitializeVoiceActivityDetectionPacket{
		ContextID:  "ctx-vad-init-error",
		StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
	})

	errPkt := requireSingleModeSwitchErrorPacket(t, r)
	assert.Equal(t, "ctx-vad-init-error", errPkt.ContextID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_AUDIO, errPkt.StreamMode)
	assert.Equal(t, internal_type.ModeSwitchErrorTypeInitializeVoiceActivityDetection, errPkt.Type)
	assert.Error(t, errPkt.Error)
	assert.Nil(t, r.vadExecutor)
}

func TestHandleVadAudio_ExecuteIsSynchronous(t *testing.T) {
	r := newDispatchHandlerVADTestRequestor(t)
	blocking := &blockingVADExecutor{
		enterCh:   make(chan struct{}),
		releaseCh: make(chan struct{}),
	}
	r.vadExecutor = blocking
	h := requestorDispatchHandler{r: r}

	done := make(chan struct{})
	go func() {
		h.HandleVadAudio(t.Context(), internal_type.VadAudioPacket{ContextID: "ctx-vad", Audio: []byte{0, 0}})
		close(done)
	}()

	select {
	case <-blocking.enterCh:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected VAD Execute to be invoked")
	}

	select {
	case <-done:
		t.Fatal("HandleVadAudio returned before Execute unblocked")
	case <-time.After(75 * time.Millisecond):
	}

	close(blocking.releaseCh)

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected HandleVadAudio to return after Execute unblocks")
	}
}
