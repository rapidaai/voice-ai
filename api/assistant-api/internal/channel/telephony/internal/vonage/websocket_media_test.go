package internal_vonage_telephony

import (
	"errors"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeVonageMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

func (engine *fakeVonageMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
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

func (engine *fakeVonageMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error {
	return nil
}

func (engine *fakeVonageMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeVonageMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeVonageMediaEngine) ClearOutputBuffer() {}

func (engine *fakeVonageMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeVonageMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func (engine *fakeVonageMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakeVonageMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

func TestNewVonageWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "vonage",
	}

	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(callContext),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	vonageStreamer, ok := streamer.(*vonageWebsocketStreamer)
	require.True(t, ok, "expected vonage websocket streamer")
	defer vonageStreamer.Cancel()

	require.NotNil(t, vonageStreamer.mediaSession)
}

func TestHandleMediaEvent_EmitsBridgeUserAudio(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "vonage",
	}
	mediaEngine := &fakeVonageMediaEngine{}
	vonageStreamer := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	vonageStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     vonageStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  vonageStreamer.Input,
	})

	providerAudio := []byte{9, 8, 7}
	err := vonageStreamer.handleMediaEvent(providerAudio)
	require.NoError(t, err)

	select {
	case stream := <-vonageStreamer.LowCh:
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
		Provider:       "vonage",
	}
	mediaEngine := &fakeVonageMediaEngine{processError: errors.New("media process failed")}
	vonageStreamer := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	vonageStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     vonageStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  vonageStreamer.Input,
	})

	err := vonageStreamer.handleMediaEvent([]byte{9, 8, 7})
	require.ErrorContains(t, err, "media process failed")
}
