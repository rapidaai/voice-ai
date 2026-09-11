package internal_telnyx_telephony

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type fakeTelnyxMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
	clearCount    atomic.Int32
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

func (engine *fakeTelnyxMediaEngine) OutputDrained() bool { return true }

func (engine *fakeTelnyxMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeTelnyxMediaEngine) ClearOutputBuffer() {
	engine.clearCount.Add(1)
}

func (engine *fakeTelnyxMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeTelnyxMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

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

func TestSend_ConsumesOutputControls(t *testing.T) {
	outputControlTestCases := []struct {
		name               string
		control            proto.Message
		resumeProbe        bool
		wantLocalClears    int32
		wantProviderClears int32
	}{
		{name: "pause", control: &protos.ConversationPlaybackPause{}},
		{name: "continue", control: &protos.ConversationPlaybackContinue{}, resumeProbe: true},
		{name: "flush", control: &protos.ConversationPlaybackFlush{}, wantLocalClears: 1, wantProviderClears: 1},
	}

	for _, testCase := range outputControlTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mediaEngine := &fakeTelnyxMediaEngine{}
			var providerClearCount atomic.Int32
			streamer := &telnyxWebsocketStreamer{
				BaseTelephonyStreamer: internal_telephony_base.New(nil, &callcontext.CallContext{}, nil, nil),
			}
			streamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
				Context:     context.Background(),
				MediaEngine: mediaEngine,
				SendProviderClear: func() error {
					providerClearCount.Add(1)
					return nil
				},
			})
			if testCase.resumeProbe {
				_, outputControlError := streamer.mediaSession.HandleOutputControl(&protos.ConversationPlaybackPause{})
				require.NoError(t, outputControlError)
				providerClearCount.Store(0)
			}

			require.NoError(t, streamer.Send(testCase.control))
			if testCase.resumeProbe {
				require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
			}
			assert.Equal(t, testCase.wantLocalClears, mediaEngine.clearCount.Load())
			assert.Equal(t, testCase.wantProviderClears, providerClearCount.Load())
			select {
			case <-streamer.OutputCh.Ready():
				output, err := streamer.OutputCh.TryReceive()
				require.NoError(t, err)
				t.Fatalf("output control was forwarded: %T", output)
			default:
			}
		})
	}
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
	case <-telnyxStreamer.LowCh.Ready():
		stream, err := telnyxStreamer.LowCh.TryReceive()
		require.NoError(t, err)
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
