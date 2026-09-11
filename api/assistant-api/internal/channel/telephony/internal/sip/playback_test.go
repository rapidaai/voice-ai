package internal_sip_telephony

import (
	"bytes"
	"sync"
	"testing"
	"time"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMediaPortPlaybackTransferDiscardsFetchedAndQueuedResponses(t *testing.T) {
	var completions []*protos.ConversationPlaybackComplete
	port, _, audioOut := newMediaPortForTest(t, func(message proto.Message) {
		if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
			completions = append(completions, completion)
		}
	})
	defer func() { require.NoError(t, port.Close()) }()
	for _, id := range []string{"first", "second"} {
		accepted, err := port.HandleAssistantAudio(id, make([]byte, BridgeOutputFrameSize), true)
		require.NoError(t, err)
		require.True(t, accepted)
	}
	frame := port.mediaSession.NextFrame()
	require.NotEmpty(t, frame)
	require.True(t, port.EnterTransferMode(DefaultRingtone))
	require.NoError(t, port.mediaSession.ConsumeFrame(frame))
	require.Empty(t, audioOut)
	accepted, err := port.HandleAssistantAudio("during-transfer", make([]byte, BridgeOutputFrameSize), true)
	require.NoError(t, err)
	require.False(t, accepted)
	require.True(t, port.ResumeAssistant())
	require.Empty(t, port.mediaSession.NextFrame())
	require.Empty(t, completions)
	accepted, err = port.HandleAssistantAudio("first", make([]byte, BridgeOutputFrameSize), true)
	require.NoError(t, err)
	require.True(t, accepted)
	for tick := 0; tick < 4; tick++ {
		if frame := port.mediaSession.NextFrame(); len(frame) > 0 {
			require.NoError(t, port.mediaSession.ConsumeFrame(frame))
		}
	}
	require.NotEmpty(t, audioOut)
	require.Empty(t, completions)
}

type replayMediaEngine struct {
	*AudioProcessor
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (engine *replayMediaEngine) ProcessAssistantAudio(audio []byte, completed bool) error {
	if len(audio) > 0 {
		engine.once.Do(func() {
			close(engine.entered)
			<-engine.release
		})
	}
	return engine.AudioProcessor.ProcessAssistantAudio(audio, completed)
}

func TestStreamerPlaybackPreanswerReplayPrecedesLiveAudio(t *testing.T) {
	s := newTestSIPStreamer(t)
	completions := make(chan *protos.ConversationPlaybackComplete, 4)
	sink := func(message proto.Message) {
		if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
			completions <- completion
		}
	}
	port, _, audioOut := newMediaPortForTest(t, sink)
	s.mediaPort = port
	defer func() { require.NoError(t, s.Close()) }()
	engine := &replayMediaEngine{AudioProcessor: port.audioProcessor, entered: make(chan struct{}), release: make(chan struct{})}
	port.mediaSession.Shutdown()
	port.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context: port.ctx, MediaEngine: engine, StreamSink: sink, OutputSink: port.deliverAssistantFrame,
	})
	require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
		Id:      "preanswer",
		Message: &protos.ConversationAssistantMessage_Audio{Audio: make([]byte, BridgeOutputFrameSize)},
	}))
	require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
		Id: "preanswer", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
	}))
	require.Empty(t, audioOut)
	require.Empty(t, completions)
	started := make(chan struct{})
	go func() { defer close(started); s.StartAssistantOutput() }()
	select {
	case <-engine.entered:
	case <-time.After(time.Second):
		t.Fatal("preanswer replay did not start")
	}
	liveSent := make(chan error, 1)
	go func() {
		liveSent <- s.Send(&protos.ConversationAssistantMessage{
			Id: "live", Completed: true,
			Message: &protos.ConversationAssistantMessage_Audio{Audio: make([]byte, BridgeOutputFrameSize)},
		})
	}()
	select {
	case err := <-liveSent:
		t.Fatalf("live output overtook replay: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(engine.release)
	<-started
	require.NoError(t, <-liveSent)
	for index, wantID := range []string{"preanswer", "live"} {
		select {
		case got := <-completions:
			require.Equal(t, wantID, got.GetId(), "completion %d id", index)
			require.NotNil(t, got.GetTime(), "completion %d time", index)
		case <-time.After(time.Second):
			t.Fatalf("missing completion for %s", wantID)
		}
	}
}

func TestSIPAudioProcessorClearDiscardsResamplerTail(t *testing.T) {
	processor := NewAudioProcessor(AudioProcessorConfig{RTPHandler: newTestRTPHandler(nil)})
	defer processor.Close()
	require.NoError(t, processor.ProcessAssistantAudio(bytes.Repeat([]byte{1, 2}, 401), false))
	processor.ClearOutputBuffer()
	require.NoError(t, processor.ProcessAssistantAudio(nil, true))
	_, ok := processor.NextOutputFrame()
	require.False(t, ok, "flushed filter history returned as response audio")
	require.NoError(t, processor.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), true))
	frame, ok := processor.NextOutputFrame()
	require.True(t, ok)
	require.Equal(t, bytes.Repeat([]byte{MulawSilenceByte}, MulawFrameSize), frame.ProviderAudio)
}

func TestStreamerTransferDropsPendingAudioAndAcceptsSameContextResume(t *testing.T) {
	s := newTestSIPStreamer(t)
	port, _, _ := newMediaPortForTest(t, nil)
	s.mediaPort = port
	defer func() { require.NoError(t, s.Close()) }()
	require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
		Id: "same-context", Completed: true,
		Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1, 2}},
	}))
	require.Len(t, s.pendingAssistantAudioFrames, 1)
	s.EnterTransferMode(nil, "", DefaultRingtone)
	require.Empty(t, s.pendingAssistantAudioFrames)
	s.ResumeAssistant()
	require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
		Id: "same-context", Completed: true,
		Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{3, 4}},
	}))
	require.Len(t, s.pendingAssistantAudioFrames, 1)
	require.Equal(t, []byte{3, 4}, s.pendingAssistantAudioFrames[0].audio)
}
