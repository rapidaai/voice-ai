package internal_vobiz_telephony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeVobizMediaEngine struct {
	clearCount atomic.Int32
}

func (*fakeVobizMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	return internal_telephony_media.InputAudioFrame{ReceivedAt: frame.ReceivedAt}, nil
}

func (*fakeVobizMediaEngine) ProcessAssistantAudio([]byte, bool) error {
	return nil
}

func (*fakeVobizMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (*fakeVobizMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeVobizMediaEngine) ClearOutputBuffer() {
	engine.clearCount.Add(1)
}

func (*fakeVobizMediaEngine) ConfigureAmbient(internal_ambient.Config) error {
	return nil
}

func (*fakeVobizMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func TestSend_ConsumesOutputControls(t *testing.T) {
	outputControlTestCases := []struct {
		name               string
		control            internal_type.Stream
		resumeProbe        bool
		wantLocalClears    int32
		wantProviderClears int32
	}{
		{name: "pause", control: internal_type.PauseOutput{}},
		{name: "continue", control: internal_type.ContinueOutput{}, resumeProbe: true},
		{name: "flush", control: internal_type.FlushOutput{}, wantLocalClears: 1, wantProviderClears: 1},
	}

	for _, testCase := range outputControlTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mediaEngine := &fakeVobizMediaEngine{}
			var providerClearCount atomic.Int32
			streamer := &vobizWebsocketStreamer{
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
				_, outputControlError := streamer.mediaSession.HandleOutputControl(internal_type.PauseOutput{})
				require.NoError(t, outputControlError)
				providerClearCount.Store(0)
			}

			require.NoError(t, streamer.Send(testCase.control))
			if testCase.resumeProbe {
				require.NoError(t, streamer.Send(internal_type.PauseOutput{}))
			}
			assert.Equal(t, testCase.wantLocalClears, mediaEngine.clearCount.Load())
			assert.Equal(t, testCase.wantProviderClears, providerClearCount.Load())
			select {
			case output := <-streamer.OutputCh:
				t.Fatalf("output control was forwarded: %T", output)
			default:
			}
		})
	}
}
