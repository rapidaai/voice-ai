package internal_telnyx_telephony

import (
	"errors"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTelnyxMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

func (engine *fakeTelnyxMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	engine.providerFrame = frame
	if engine.processError != nil {
		return internal_telephony_media.InputAudioFrame{}, engine.processError
	}
	return internal_telephony_media.InputAudioFrame{
		BridgeAudio:   []byte{1},
		PipelineAudio: []byte{2},
		ReceivedAt:    frame.ReceivedAt,
	}, nil
}

func (engine *fakeTelnyxMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error {
	return nil
}

func (engine *fakeTelnyxMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeTelnyxMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeTelnyxMediaEngine) ClearOutputBuffer() {}

func (engine *fakeTelnyxMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeTelnyxMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func (engine *fakeTelnyxMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakeTelnyxMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

func TestNewTelnyxWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "telnyx",
	}

	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(callContext),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	telnyxStreamer, ok := streamer.(*telnyxWebsocketStreamer)
	require.True(t, ok, "expected telnyx websocket streamer")
	defer telnyxStreamer.Cancel()

	require.NotNil(t, telnyxStreamer.mediaSession)
}

func TestHandleMediaEvent_EmitsBridgeUserAudio(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "telnyx",
	}
	mediaEngine := &fakeTelnyxMediaEngine{}
	telnyxStreamer := &telnyxWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	telnyxStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     telnyxStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  telnyxStreamer.Input,
	})

	providerAudio := []byte{9, 8, 7}
	mediaEvent := internal_telnyx.TelnyxWebSocketEvent{
		Media: &internal_telnyx.TelnyxMediaEvent{
			Payload: telnyxStreamer.Encoder().EncodeToString(providerAudio),
		},
	}
	err := telnyxStreamer.handleMediaEvent(mediaEvent)
	require.NoError(t, err)

	select {
	case stream := <-telnyxStreamer.LowCh:
		bridgeAudio, ok := stream.(*protos.ConversationBridgeUserAudio)
		require.True(t, ok, "expected bridge user audio, got %T", stream)
		assert.NotEmpty(t, bridgeAudio.GetAudio())
		assert.NotNil(t, bridgeAudio.GetTime())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridge user audio")
	}
	assert.Equal(t, providerAudio, mediaEngine.providerFrame.Audio)
	assert.False(t, mediaEngine.providerFrame.ReceivedAt.IsZero())
}

func TestHandleMediaEvent_ReturnsMediaProcessingError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "telnyx",
	}
	mediaEngine := &fakeTelnyxMediaEngine{processError: errors.New("media process failed")}
	telnyxStreamer := &telnyxWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	telnyxStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     telnyxStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  telnyxStreamer.Input,
	})

	mediaEvent := internal_telnyx.TelnyxWebSocketEvent{
		Media: &internal_telnyx.TelnyxMediaEvent{
			Payload: telnyxStreamer.Encoder().EncodeToString([]byte{9, 8, 7}),
		},
	}
	err := telnyxStreamer.handleMediaEvent(mediaEvent)
	require.ErrorContains(t, err, "media process failed")
}

func TestHandleMediaEvent_MissingMediaPayloadDoesNotPanic(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "telnyx",
	}
	telnyxStreamer := &telnyxWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}

	err := telnyxStreamer.handleMediaEvent(internal_telnyx.TelnyxWebSocketEvent{})
	require.NoError(t, err)
}
