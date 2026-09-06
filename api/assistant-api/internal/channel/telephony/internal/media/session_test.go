package internal_telephony_media

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
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

func (mediaEngine *fakeMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (mediaEngine *fakeMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

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
	streams := make(chan internal_type.Stream, 2)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		StreamSink:  func(stream internal_type.Stream) { streams <- stream },
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
	streams := make(chan internal_type.Stream, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		StreamSink:  func(stream internal_type.Stream) { streams <- stream },
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

	if err := mediaSession.HandleAssistantAudio([]byte{1, 2, 3}, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	if err := mediaSession.HandleAssistantAudio([]byte{1}, false); err == nil {
		t.Fatal("expected assistant audio error")
	}
}

func TestMediaSession_OutputPacer_EmitsOperatorBridgeAfterProviderSend(t *testing.T) {
	mediaEngine := &fakeMediaEngine{
		outputFrames:  make(chan AssistantOutputFrame, 1),
		frameDuration: 2 * time.Millisecond,
	}
	outputFrames := make(chan AssistantOutputFrame, 1)
	streams := make(chan internal_type.Stream, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			outputFrames <- frame
			return nil
		},
		StreamSink: func(stream internal_type.Stream) { streams <- stream },
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
	streams := make(chan internal_type.Stream, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			outputFrames <- frame
			return nil
		},
		StreamSink: func(stream internal_type.Stream) { streams <- stream },
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
	streams := make(chan internal_type.Stream, 1)
	mediaSession := NewMediaSession(MediaSessionConfig{
		Context:     context.Background(),
		MediaEngine: mediaEngine,
		OutputSink: func(frame AssistantOutputFrame) error {
			return errors.New("rtp queue full")
		},
		StreamSink: func(stream internal_type.Stream) { streams <- stream },
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

	mediaSession.HandleInterrupt()

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
