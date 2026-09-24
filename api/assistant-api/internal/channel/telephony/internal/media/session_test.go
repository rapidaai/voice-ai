package internal_telephony_media

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeMediaEngine struct {
	inputFrame        InputAudioFrame
	inputErr          error
	providerAudio     []byte
	providerReceived  time.Time
	assistantAudio    []byte
	assistantComplete bool
	assistantErr      error
	outputFrames      chan AssistantOutputFrame
	idleFrame         AssistantOutputFrame
	frameDuration     time.Duration
	clearCount        atomic.Int32
	configureCount    atomic.Int32
}

func (mediaEngine *fakeMediaEngine) ProcessProviderAudioFrame(frame ProviderAudioFrame) (InputAudioFrame, error) {
	mediaEngine.providerAudio = append([]byte(nil), frame.Audio...)
	mediaEngine.providerReceived = frame.ReceivedAt
	if mediaEngine.inputFrame.ReceivedAt.IsZero() {
		mediaEngine.inputFrame.ReceivedAt = frame.ReceivedAt
	}
	return mediaEngine.inputFrame, mediaEngine.inputErr
}

func (mediaEngine *fakeMediaEngine) ProcessAssistantAudio(audio []byte, completed bool) error {
	mediaEngine.assistantAudio = append([]byte(nil), audio...)
	mediaEngine.assistantComplete = completed
	return mediaEngine.assistantErr
}

func (mediaEngine *fakeMediaEngine) NextOutputFrame() (AssistantOutputFrame, bool) {
	select {
	case outputFrame := <-mediaEngine.outputFrames:
		return outputFrame, true
	default:
		return AssistantOutputFrame{}, false
	}
}

func (mediaEngine *fakeMediaEngine) OutputDrained() bool {
	return len(mediaEngine.outputFrames) == 0
}

func (mediaEngine *fakeMediaEngine) IdleOutputFrame() (AssistantOutputFrame, bool) {
	if len(mediaEngine.idleFrame.ProviderAudio) == 0 {
		return AssistantOutputFrame{}, false
	}
	return mediaEngine.idleFrame, true
}

func (mediaEngine *fakeMediaEngine) ClearOutputBuffer() {
	mediaEngine.clearCount.Add(1)
}

func (mediaEngine *fakeMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	mediaEngine.configureCount.Add(1)
	return nil
}

func (mediaEngine *fakeMediaEngine) OutputFrameDuration() time.Duration {
	if mediaEngine.frameDuration <= 0 {
		return 5 * time.Millisecond
	}
	return mediaEngine.frameDuration
}

func mustAnyValue(t *testing.T, value *structpb.Value) *anypb.Any {
	t.Helper()
	anyValue, err := anypb.New(value)
	if err != nil {
		t.Fatalf("any value conversion failed: %v", err)
	}
	return anyValue
}

func TestMediaSession_StartAndShutdown_Idempotent(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink:  func(frame AssistantOutputFrame) error { return nil },
	})

	mediaSession.Start()
	mediaSession.Start()
	mediaSession.Shutdown()
	mediaSession.Shutdown()

	if !mediaSession.started.Load() {
		t.Fatal("media session did not start")
	}
	if !mediaSession.closed.Load() {
		t.Fatal("media session did not close")
	}
}

func TestMediaSession_HandleInitialization_ParsesAmbient(t *testing.T) {
	mediaEngine := &fakeMediaEngine{}
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
	})

	mediaSession.HandleInitialization(&protos.ConversationInitialization{Options: map[string]*anypb.Any{
		"speaker.ambient":        mustAnyValue(t, structpb.NewStringValue("cafe")),
		"speaker.ambient_volume": mustAnyValue(t, structpb.NewNumberValue(40)),
	}})

	if mediaEngine.configureCount.Load() != 1 {
		t.Fatalf("configureCount=%d want=1", mediaEngine.configureCount.Load())
	}
}

func TestMediaSession_HandleProviderAudioFrame_EmitsBridgeAndPipelineAudio(t *testing.T) {
	receivedAt := time.Now().Add(-time.Second)
	mediaEngine := &fakeMediaEngine{
		inputFrame: InputAudioFrame{
			BridgeAudio:   []byte{1, 2},
			PipelineAudio: []byte{3, 4},
		},
		outputFrames: make(chan AssistantOutputFrame, 1),
	}
	streams := make(chan proto.Message, 2)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		StreamSink:  func(stream proto.Message) { streams <- stream },
	})

	if err := mediaSession.HandleProviderAudioFrame(ProviderAudioFrame{
		Audio:      []byte{9},
		ReceivedAt: receivedAt,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bridgeAudio, ok := (<-streams).(*protos.ConversationBridgeUserAudio)
	if !ok {
		t.Fatalf("expected bridge user audio")
	}
	if len(bridgeAudio.Audio) != 2 || bridgeAudio.Audio[0] != 1 {
		t.Fatalf("unexpected bridge audio: %v", bridgeAudio.Audio)
	}
	if !bridgeAudio.Time.AsTime().Equal(receivedAt) {
		t.Fatalf("bridge time=%s want=%s", bridgeAudio.Time.AsTime(), receivedAt)
	}

	userAudio, ok := (<-streams).(*protos.ConversationUserMessage)
	if !ok {
		t.Fatalf("expected user audio")
	}
	if len(userAudio.GetAudio()) != 2 || userAudio.GetAudio()[0] != 3 {
		t.Fatalf("unexpected user audio: %v", userAudio.GetAudio())
	}
	if !userAudio.Time.AsTime().Equal(receivedAt) {
		t.Fatalf("user time=%s want=%s", userAudio.Time.AsTime(), receivedAt)
	}
	select {
	case stream := <-streams:
		t.Fatalf("unexpected extra stream %T", stream)
	default:
	}
}

func TestMediaSession_HandleProviderAudioFrame_UsesServerTimeWhenMissing(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		inputFrame:   InputAudioFrame{BridgeAudio: []byte{1}},
		outputFrames: make(chan AssistantOutputFrame, 1),
	}
	streams := make(chan proto.Message, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		StreamSink:  func(stream proto.Message) { streams <- stream },
	})

	before := time.Now()
	if err := mediaSession.HandleProviderAudioFrame(ProviderAudioFrame{Audio: []byte{9}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	bridgeAudio, ok := (<-streams).(*protos.ConversationBridgeUserAudio)
	if !ok {
		t.Fatalf("expected bridge user audio")
	}
	bridgeTime := bridgeAudio.Time.AsTime()
	if bridgeTime.Before(before) || bridgeTime.After(after) {
		t.Fatalf("bridge time=%s outside receive window %s..%s", bridgeTime, before, after)
	}
	if mediaEngine.providerReceived.IsZero() {
		t.Fatal("provider received time was not set")
	}
}

func TestMediaSession_HandleProviderAudioFrame_DropsPipelineAudioWhenStreamSinkMissing(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		inputFrame:   InputAudioFrame{PipelineAudio: []byte{7, 8}},
		outputFrames: make(chan AssistantOutputFrame, 1),
	}
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
	})

	if err := mediaSession.HandleProviderAudioFrame(ProviderAudioFrame{Audio: []byte{1}, ReceivedAt: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mediaEngine.providerAudio) != 1 || mediaEngine.providerAudio[0] != 1 {
		t.Fatalf("provider audio not processed: %v", mediaEngine.providerAudio)
	}
}

func TestMediaSession_HandleAssistantAudio_UsesMediaEngine(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
	})

	if assistantAudioAccepted, assistantAudioError := mediaSession.HandleAssistantAudio("response-1", []byte{1, 2, 3}, true); assistantAudioError != nil || !assistantAudioAccepted {
		t.Fatalf("unexpected error: %v", assistantAudioError)
	}
	if len(mediaEngine.assistantAudio) != 3 || mediaEngine.assistantAudio[0] != 1 {
		t.Fatalf("assistant audio not passed to media engine: %v", mediaEngine.assistantAudio)
	}
	if !mediaEngine.assistantComplete {
		t.Fatal("assistant completion was not passed to media engine")
	}
}

func TestMediaSession_HandleAssistantAudio_PropagatesError(t *testing.T) {
	mediaEngine := &fakeMediaEngine{assistantErr: errors.New("process failed")}
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
	})

	if _, assistantAudioError := mediaSession.HandleAssistantAudio("response-1", []byte{1}, false); assistantAudioError == nil {
		t.Fatal("expected assistant audio error")
	}
}

func TestMediaSession_OutputControlsWithoutMediaEngine(t *testing.T) {
	mediaSession := NewMediaSession(MediaSessionConfig{Context: context.Background()})

	for _, outputControl := range []proto.Message{
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE},
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE},
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH},
	} {
		outputControlHandled, outputControlError := mediaSession.HandleOutputControl(outputControl)
		if outputControlError != nil || !outputControlHandled {
			t.Fatalf("%T handled=%t err=%v", outputControl, outputControlHandled, outputControlError)
		}
	}
}

func TestMediaSession_OutputControlPauseContinueRetainsFetchedFrame(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	mediaEngine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1, 2}, BridgeAudio: []byte{3, 4}}
	var providerClearCount atomic.Int32
	written := make(chan AssistantOutputFrame, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context: context.Background(), MediaEngine: mediaEngine,
		SendProviderClear: func() error { providerClearCount.Add(1); return nil },
		OutputSink:        func(frame AssistantOutputFrame) error { written <- frame; return nil },
	})

	providerAudio := mediaSession.NextFrame()
	if !bytes.Equal(providerAudio, []byte{1, 2}) {
		t.Fatalf("provider audio=%v want=[1 2]", providerAudio)
	}
	outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE})
	if outputControlError != nil || !outputControlHandled {
		t.Fatalf("pause handled=%t err=%v", outputControlHandled, outputControlError)
	}
	if frame := mediaSession.NextFrame(); frame != nil {
		t.Fatalf("paused next frame=%v want=nil", frame)
	}
	if err := mediaSession.ConsumeFrame(providerAudio); err != nil {
		t.Fatalf("consume paused frame: %v", err)
	}
	select {
	case frame := <-written:
		t.Fatalf("paused frame was written: %+v", frame)
	default:
	}
	if providerClearCount.Load() != 0 {
		t.Fatalf("provider clears=%d want=0", providerClearCount.Load())
	}

	outputControlHandled, outputControlError = mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE})
	if outputControlError != nil || !outputControlHandled {
		t.Fatalf("continue handled=%t err=%v", outputControlHandled, outputControlError)
	}
	providerAudio = mediaSession.NextFrame()
	if !bytes.Equal(providerAudio, []byte{1, 2}) {
		t.Fatalf("continued provider audio=%v want=[1 2]", providerAudio)
	}
	if err := mediaSession.ConsumeFrame(providerAudio); err != nil {
		t.Fatalf("consume continued frame: %v", err)
	}
	select {
	case frame := <-written:
		if !bytes.Equal(frame.BridgeAudio, []byte{3, 4}) {
			t.Fatalf("bridge audio=%v want=[3 4]", frame.BridgeAudio)
		}
	default:
		t.Fatal("continued frame was not written")
	}
}

func TestMediaSession_OutputControlFlushRejectsOldResponse(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	mediaEngine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1, 2}}
	var providerClearCount atomic.Int32
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context: context.Background(), MediaEngine: mediaEngine,
		SendProviderClear: func() error { providerClearCount.Add(1); return nil },
	})

	if assistantAudioAccepted, assistantAudioError := mediaSession.HandleAssistantAudio("response-1", []byte{1}, false); assistantAudioError != nil || !assistantAudioAccepted {
		t.Fatalf("initial assistant audio: %v", assistantAudioError)
	}
	if frame := mediaSession.NextFrame(); frame == nil {
		t.Fatal("expected fetched output frame")
	}
	if outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE}); outputControlError != nil || !outputControlHandled {
		t.Fatalf("pause handled=%t err=%v", outputControlHandled, outputControlError)
	}
	outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
	if outputControlError != nil || !outputControlHandled {
		t.Fatalf("flush handled=%t err=%v", outputControlHandled, outputControlError)
	}
	if frame := mediaSession.NextFrame(); frame != nil {
		t.Fatalf("frame survived flush: %v", frame)
	}
	if mediaEngine.clearCount.Load() != 1 || providerClearCount.Load() != 1 {
		t.Fatalf("engine clears=%d provider clears=%d want=1,1", mediaEngine.clearCount.Load(), providerClearCount.Load())
	}
	if assistantAudioAccepted, assistantAudioError := mediaSession.HandleAssistantAudio("response-1", []byte{2}, true); assistantAudioError != nil || assistantAudioAccepted {
		t.Fatalf("stale assistant audio: %v", assistantAudioError)
	}
	if assistantAudioAccepted, assistantAudioError := mediaSession.HandleAssistantAudio("response-2", []byte{3}, false); assistantAudioError != nil || !assistantAudioAccepted {
		t.Fatalf("next assistant audio: %v", assistantAudioError)
	}
}

func TestMediaSession_OutputControlFlushBeforeFirstAudioBlocksID(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	var providerClearCount atomic.Int32
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context: context.Background(), MediaEngine: mediaEngine,
		SendProviderClear: func() error { providerClearCount.Add(1); return nil },
	})

	handled, err := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response-preaudio", Kind: protos.ConversationPlaybackControl_FLUSH})
	if err != nil || !handled {
		t.Fatalf("flush handled=%t err=%v", handled, err)
	}
	if mediaEngine.clearCount.Load() != 1 || providerClearCount.Load() != 1 {
		t.Fatalf("engine clears=%d provider clears=%d want=1,1", mediaEngine.clearCount.Load(), providerClearCount.Load())
	}

	accepted, err := mediaSession.HandleAssistantAudio("response-preaudio", []byte{1}, true)
	if err != nil || accepted {
		t.Fatalf("flushed preaudio response accepted=%t err=%v", accepted, err)
	}
	accepted, err = mediaSession.HandleAssistantAudio("response-next", []byte{2}, true)
	if err != nil || !accepted {
		t.Fatalf("next response accepted=%t err=%v", accepted, err)
	}
}

func TestMediaSession_OutputControlRepeatedFlushReleasesPause(t *testing.T) {
	for _, testCase := range []struct {
		name               string
		providerClearError error
	}{
		{name: "provider clear succeeds"},
		{name: "provider clear fails", providerClearError: errors.New("clear failed")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
			var providerClearCount atomic.Int32
			written := make(chan AssistantOutputFrame, 1)
			mediaSession := NewMediaSession(MediaSessionConfig{
				Context:     context.Background(),
				MediaEngine: mediaEngine,
				SendProviderClear: func() error {
					providerClearCount.Add(1)
					return testCase.providerClearError
				},
				OutputSink: func(frame AssistantOutputFrame) error {
					written <- frame
					return nil
				},
			})
			defer mediaSession.Shutdown()

			accepted, err := mediaSession.HandleAssistantAudio("response-A", []byte{1}, false)
			if err != nil || !accepted {
				t.Fatalf("response A accepted=%t err=%v", accepted, err)
			}
			handled, err := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response-A", Kind: protos.ConversationPlaybackControl_FLUSH})
			if !handled || !errors.Is(err, testCase.providerClearError) {
				t.Fatalf("flush A handled=%t err=%v want=%v", handled, err, testCase.providerClearError)
			}

			handled, err = mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response-B", Kind: protos.ConversationPlaybackControl_PAUSE})
			if err != nil || !handled {
				t.Fatalf("pause B handled=%t err=%v", handled, err)
			}
			handled, err = mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response-B", Kind: protos.ConversationPlaybackControl_FLUSH})
			if err != nil || !handled {
				t.Fatalf("flush B handled=%t err=%v", handled, err)
			}
			if mediaEngine.clearCount.Load() != 1 || providerClearCount.Load() != 1 {
				t.Fatalf("engine clears=%d provider clears=%d want=1,1", mediaEngine.clearCount.Load(), providerClearCount.Load())
			}
			accepted, err = mediaSession.HandleAssistantAudio("response-B", []byte{2}, true)
			if err != nil || accepted {
				t.Fatalf("response B accepted=%t err=%v", accepted, err)
			}

			accepted, err = mediaSession.HandleAssistantAudio("response-C", []byte{3, 4}, true)
			if err != nil || !accepted {
				t.Fatalf("response C accepted=%t err=%v", accepted, err)
			}
			mediaEngine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{3, 4}, BridgeAudio: []byte{5, 6}}
			providerAudio := mediaSession.NextFrame()
			if !bytes.Equal(providerAudio, []byte{3, 4}) {
				t.Fatalf("response C provider audio=%v want=[3 4]", providerAudio)
			}
			if err := mediaSession.ConsumeFrame(providerAudio); err != nil {
				t.Fatalf("consume response C: %v", err)
			}
			select {
			case frame := <-written:
				if !bytes.Equal(frame.ProviderAudio, []byte{3, 4}) || !bytes.Equal(frame.BridgeAudio, []byte{5, 6}) {
					t.Fatalf("response C frame=%+v want provider=[3 4] bridge=[5 6]", frame)
				}
			default:
				t.Fatal("response C was not written after repeated flush")
			}
		})
	}
}

func TestMediaSession_OutputControlReturnsProviderClearError(t *testing.T) {
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context: context.Background(), MediaEngine: &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)},
		SendProviderClear: func() error { return errors.New("clear failed") },
	})

	outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
	if !outputControlHandled || outputControlError == nil {
		t.Fatalf("flush handled=%t err=%v", outputControlHandled, outputControlError)
	}
}

func TestMediaSession_OutputPacer_EmitsOperatorBridgeAfterProviderSend(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		outputFrames:  make(chan AssistantOutputFrame, 1),
		frameDuration: 2 * time.Millisecond,
	}
	outputFrames := make(chan AssistantOutputFrame, 1)
	streams := make(chan proto.Message, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			outputFrames <- frame
			return nil
		},
		StreamSink: func(stream proto.Message) { streams <- stream },
	})

	mediaEngine.outputFrames <- AssistantOutputFrame{
		ProviderAudio: []byte{1, 2},
		BridgeAudio:   []byte{3, 4},
	}
	mediaSession.Start()
	defer mediaSession.Shutdown()

	select {
	case outputFrame := <-outputFrames:
		if len(outputFrame.ProviderAudio) != 2 || outputFrame.ProviderAudio[0] != 1 {
			t.Fatalf("unexpected provider frame: %v", outputFrame.ProviderAudio)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting provider output frame")
	}

	select {
	case stream := <-streams:
		bridgeAudio, ok := stream.(*protos.ConversationBridgeOperatorAudio)
		if !ok {
			t.Fatalf("expected operator bridge audio, got %T", stream)
		}
		if len(bridgeAudio.Audio) != 2 || bridgeAudio.Audio[0] != 3 {
			t.Fatalf("unexpected bridge audio: %v", bridgeAudio.Audio)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting operator bridge audio")
	}
}

func TestMediaSession_OutputPacer_DoesNotBridgeIdleFrame(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		outputFrames: make(chan AssistantOutputFrame, 1),
		idleFrame: AssistantOutputFrame{
			ProviderAudio: []byte{0xff},
			BridgeAudio:   []byte{9},
		},
		frameDuration: 2 * time.Millisecond,
	}
	outputFrames := make(chan AssistantOutputFrame, 1)
	streams := make(chan proto.Message, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			outputFrames <- frame
			return nil
		},
		StreamSink: func(stream proto.Message) { streams <- stream },
	})

	mediaSession.Start()
	defer mediaSession.Shutdown()

	select {
	case outputFrame := <-outputFrames:
		if !outputFrame.Idle {
			t.Fatal("expected idle output frame")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting idle output frame")
	}

	select {
	case stream := <-streams:
		t.Fatalf("idle frame emitted bridge stream: %T", stream)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestMediaSession_OutputPacer_EmitsEventOnOutputSendError(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		outputFrames:  make(chan AssistantOutputFrame, 1),
		frameDuration: 2 * time.Millisecond,
	}
	records := make(chan observability.Record, 2)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			return errors.New("send failed")
		},
		Record: func(record ...observability.Record) error {
			for _, item := range record {
				records <- item
			}
			return nil
		},
	})

	mediaEngine.outputFrames <- AssistantOutputFrame{
		ProviderAudio: []byte{1},
		BridgeAudio:   []byte{2},
	}
	mediaSession.Start()
	defer mediaSession.Shutdown()

	select {
	case record := <-records:
		log, ok := record.(observability.RecordLog)
		if !ok {
			t.Fatalf("record type=%T want RecordLog", record)
		}
		if log.Message != "Telephony output send failed" {
			t.Fatalf("message=%q want Telephony output send failed", log.Message)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting send error record")
	}
}

func TestMediaSession_OutputPacer_DoesNotRecordBridgeAudioOnOutputSendError(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		outputFrames:  make(chan AssistantOutputFrame, 1),
		frameDuration: 2 * time.Millisecond,
	}
	records := make(chan observability.Record, 2)
	streams := make(chan proto.Message, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			return errors.New("rtp queue full")
		},
		StreamSink: func(stream proto.Message) { streams <- stream },
		Record: func(record ...observability.Record) error {
			for _, item := range record {
				records <- item
			}
			return nil
		},
	})

	mediaEngine.outputFrames <- AssistantOutputFrame{
		ProviderAudio: []byte{1},
		BridgeAudio:   []byte{2},
	}
	mediaSession.Start()
	defer mediaSession.Shutdown()

	select {
	case record := <-records:
		log, ok := record.(observability.RecordLog)
		if !ok {
			t.Fatalf("record type=%T want RecordLog", record)
		}
		if log.Message != "Telephony output send failed" {
			t.Fatalf("message=%q want Telephony output send failed", log.Message)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting send error record")
	}
	select {
	case stream := <-streams:
		t.Fatalf("send failure emitted bridge recording stream: %T", stream)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestMediaSession_HandleInterrupt_ClearsAndSendsProviderClear(t *testing.T) {
	mediaEngine := &fakeMediaEngine{}
	var clearCount atomic.Int32
	records := make(chan observability.Record, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		SendProviderClear: func() error {
			clearCount.Add(1)
			return nil
		},
		Record: func(record ...observability.Record) error {
			for _, item := range record {
				records <- item
			}
			return nil
		},
	})

	outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
	if outputControlError != nil || !outputControlHandled {
		t.Fatalf("flush handled=%t err=%v", outputControlHandled, outputControlError)
	}

	if mediaEngine.clearCount.Load() != 1 {
		t.Fatalf("clearCount=%d want=1", mediaEngine.clearCount.Load())
	}
	if clearCount.Load() != 1 {
		t.Fatalf("sendProviderClear=%d want=1", clearCount.Load())
	}
	select {
	case record := <-records:
		event, ok := record.(observability.RecordEvent)
		if !ok {
			t.Fatalf("record type=%T want RecordEvent", record)
		}
		if event.Attributes["status"] != "output_queue_cleared" {
			t.Fatalf("status=%q want output_queue_cleared", event.Attributes["status"])
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting clear record")
	}
}

func TestMediaSession_FlushDropsFetchedOutputFrame(t *testing.T) {
	mediaEngine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	mediaEngine.outputFrames <- AssistantOutputFrame{
		ProviderAudio: []byte{1, 2},
		BridgeAudio:   []byte{3, 4},
	}
	written := make(chan AssistantOutputFrame, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			written <- frame
			return nil
		},
	})

	providerAudio := mediaSession.NextFrame()
	if !bytes.Equal(providerAudio, []byte{1, 2}) {
		t.Fatalf("provider audio=%v want=[1 2]", providerAudio)
	}

	outputControlHandled, outputControlError := mediaSession.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
	if outputControlError != nil || !outputControlHandled {
		t.Fatalf("flush handled=%t err=%v", outputControlHandled, outputControlError)
	}
	if err := mediaSession.ConsumeFrame(providerAudio); err != nil {
		t.Fatalf("consume interrupted frame: %v", err)
	}

	select {
	case frame := <-written:
		t.Fatalf("interrupted frame was written: %+v", frame)
	default:
	}
}
