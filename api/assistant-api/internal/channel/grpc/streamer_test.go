// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_grpc

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type recordingAssistantTalkServer struct {
	grpc.ServerStream
	mu       sync.Mutex
	sent     []*protos.AssistantTalkResponse
	requests []*protos.AssistantTalkRequest
	recvErr  error
	sendErr  error
	onSend   func(*protos.AssistantTalkResponse) error
}

func (s *recordingAssistantTalkServer) Recv() (*protos.AssistantTalkRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	if len(s.requests) == 0 {
		return nil, io.EOF
	}
	request := s.requests[0]
	s.requests = s.requests[1:]
	wire, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	decoded := &protos.AssistantTalkRequest{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func (s *recordingAssistantTalkServer) Send(response *protos.AssistantTalkResponse) error {
	if s.onSend != nil {
		if err := s.onSend(response); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sendErr != nil {
		return s.sendErr
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	decoded := &protos.AssistantTalkResponse{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		return err
	}
	s.sent = append(s.sent, decoded)
	return nil
}

func (s *recordingAssistantTalkServer) sentResponses() []*protos.AssistantTalkResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*protos.AssistantTalkResponse(nil), s.sent...)
}

func TestWebhookCollectorWithoutDependenciesIsNoop(t *testing.T) {
	collector := webhook.New(context.Background(), webhook.Config{})
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected no-op collector, got %T", collector)
	}
}

func TestSend_OutputControlsPauseContinueAndFlushAssistantAudio(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}

	oldMessage := &protos.ConversationAssistantMessage{Id: "response-1", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}}}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	require.NoError(t, streamer.Send(oldMessage))
	require.Len(t, server.sentResponses(), 1)
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	responses := server.sentResponses()
	require.Len(t, responses, 3)
	assert.NotNil(t, responses[0].GetPlaybackPause())
	assert.NotNil(t, responses[1].GetPlaybackContinue())
	assert.True(t, proto.Equal(oldMessage, responses[2].GetAssistant()))

	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackFlush{}))
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	responses = server.sentResponses()
	require.Len(t, responses, 6)
	assert.NotNil(t, responses[3].GetPlaybackPause())
	assert.NotNil(t, responses[4].GetPlaybackFlush())
	assert.NotNil(t, responses[5].GetPlaybackContinue())

	newMessage := &protos.ConversationAssistantMessage{Id: "response-2", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}}}
	require.NoError(t, streamer.Send(newMessage))
	responses = server.sentResponses()
	require.Len(t, responses, 7)
	assert.True(t, proto.Equal(newMessage, responses[6].GetAssistant()))

	interruption := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_VAD,
	}
	require.NoError(t, streamer.Send(interruption))
	responses = server.sentResponses()
	require.Len(t, responses, 8)
	assert.True(t, proto.Equal(interruption, responses[7].GetInterruption()))
}

func TestSend_FlushBeforeFirstAudioBlocksID(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}

	require.NoError(t, streamer.Send(&protos.ConversationPlaybackFlush{Id: "response-preaudio"}))
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response-preaudio", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
	}))
	nextMessage := &protos.ConversationAssistantMessage{
		Id: "response-next", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}},
	}
	require.NoError(t, streamer.Send(nextMessage))

	responses := server.sentResponses()
	require.Len(t, responses, 2)
	assert.NotNil(t, responses[0].GetPlaybackFlush())
	assert.True(t, proto.Equal(nextMessage, responses[1].GetAssistant()))
}

func TestSend_RepeatedFlushBlocksNewID(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	oldMessage := &protos.ConversationAssistantMessage{
		Id: "response-A", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
	}

	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackFlush{Id: "response-A"}))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{Id: "response-B"}))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackFlush{Id: "response-B"}))
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response-B", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}},
	}))
	nextMessage := &protos.ConversationAssistantMessage{
		Id: "response-C", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{3}},
	}
	require.NoError(t, streamer.Send(nextMessage))

	responses := server.sentResponses()
	require.Len(t, responses, 5)
	assert.True(t, proto.Equal(oldMessage, responses[0].GetAssistant()))
	assert.NotNil(t, responses[1].GetPlaybackFlush())
	assert.NotNil(t, responses[2].GetPlaybackPause())
	assert.NotNil(t, responses[3].GetPlaybackFlush())
	assert.True(t, proto.Equal(nextMessage, responses[4].GetAssistant()))
}

func TestSend_OutputControlsAreConcurrentSafe(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	controls := []proto.Message{
		&protos.ConversationPlaybackPause{},
		&protos.ConversationPlaybackContinue{},
		&protos.ConversationPlaybackFlush{},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(controls)*32)
	for range 32 {
		for _, control := range controls {
			wg.Add(1)
			go func(control proto.Message) {
				defer wg.Done()
				errCh <- streamer.Send(control)
			}(control)
		}
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	assert.Len(t, server.sentResponses(), len(controls)*32)
}

func TestSend_GeneratedWireVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message proto.Message
		field   protoreflect.Name
		number  protoreflect.FieldNumber
	}{
		{"pause", &protos.ConversationPlaybackPause{Id: "response-1"}, "playbackPause", 21},
		{"continue", &protos.ConversationPlaybackContinue{Id: "response-1"}, "playbackContinue", 22},
		{"flush", &protos.ConversationPlaybackFlush{Id: "response-1"}, "playbackFlush", 23},
		{"initialization", &protos.ConversationInitialization{}, "initialization", 0},
		{"configuration", &protos.ConversationConfiguration{}, "configuration", 0},
		{"interruption", &protos.ConversationInterruption{}, "interruption", 0},
		{"user", &protos.ConversationUserMessage{}, "user", 0},
		{"assistant", &protos.ConversationAssistantMessage{Id: "response"}, "assistant", 0},
		{"tool call", &protos.ConversationToolCall{}, "toolCall", 0},
		{"tool result", &protos.ConversationToolCallResult{}, "toolCallResult", 0},
		{"metadata", &protos.ConversationMetadata{}, "metadata", 0},
		{"metric", &protos.ConversationMetric{}, "metric", 0},
		{"error", &protos.ConversationError{}, "error", 0},
		{"event", &protos.ConversationEvent{}, "event", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := &recordingAssistantTalkServer{}
			streamer := &unidirectionalStreamer{server: server}
			require.NoError(t, streamer.Send(tt.message))
			responses := server.sentResponses()
			require.Len(t, responses, 1)
			response := responses[0]
			reflected := response.ProtoReflect()
			field := reflected.WhichOneof(reflected.Descriptor().Oneofs().ByName("data"))
			require.NotNil(t, field)
			assert.Equal(t, tt.field, field.Name())
			if tt.number != 0 {
				assert.Equal(t, tt.number, field.Number())
			}
			assert.True(t, proto.Equal(tt.message, reflected.Get(field).Message().Interface()))
			if tt.name == "error" {
				assert.EqualValues(t, 500, response.GetCode())
				assert.False(t, response.GetSuccess())
			} else {
				assert.EqualValues(t, 200, response.GetCode())
				assert.True(t, response.GetSuccess())
			}

			sendErr := errors.New("send failed")
			server.sendErr = sendErr
			assert.ErrorIs(t, streamer.Send(tt.message), sendErr)
		})
	}
}

func TestSend_PausedQueuePreservesAudioOrderAndAllowsText(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	for _, frame := range []byte{1, 2, 3} {
		require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
			Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{frame}},
		}))
	}
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "text"},
	}))
	responses := server.sentResponses()
	require.Len(t, responses, 2)
	assert.Equal(t, "text", responses[1].GetAssistant().GetText())
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	responses = server.sentResponses()
	require.Len(t, responses, 6)
	assert.NotNil(t, responses[2].GetPlaybackContinue())
	for i, frame := range []byte{1, 2, 3} {
		assert.Equal(t, []byte{frame}, responses[i+3].GetAssistant().GetAudio())
	}
}

func TestSend_ExplicitAudioTerminalRespectsPlaybackControls(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		withAudio bool
		flush     bool
	}{
		{name: "continue after audio", withAudio: true},
		{name: "continue terminal only"},
		{name: "flush after audio", withAudio: true, flush: true},
		{name: "flush terminal only", flush: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := &recordingAssistantTalkServer{}
			streamer := &unidirectionalStreamer{server: server}
			terminal := &protos.ConversationAssistantMessage{
				Id: "old", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
			}
			queued := []*protos.ConversationAssistantMessage{}
			if tt.withAudio {
				queued = append(queued, &protos.ConversationAssistantMessage{
					Id: "old", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
				})
			}
			queued = append(queued, terminal)
			require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
			for _, message := range queued {
				require.NoError(t, streamer.Send(message))
			}
			require.Len(t, server.sentResponses(), 1, "terminal must wait with paused audio")
			if tt.flush {
				require.NoError(t, streamer.Send(&protos.ConversationPlaybackFlush{}))
				for _, message := range queued {
					require.NoError(t, streamer.Send(message))
				}
				require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
				responses := server.sentResponses()
				require.Len(t, responses, 3, "flush must discard queued and late old terminals")
				assert.NotNil(t, responses[1].GetPlaybackFlush())
				assert.NotNil(t, responses[2].GetPlaybackContinue())
				newTerminal := &protos.ConversationAssistantMessage{
					Id: "new", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}
				require.NoError(t, streamer.Send(newTerminal))
				responses = server.sentResponses()
				require.Len(t, responses, 4)
				assert.True(t, proto.Equal(newTerminal, responses[3].GetAssistant()))
				return
			}
			require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
			responses := server.sentResponses()
			require.Len(t, responses, len(queued)+2)
			assert.NotNil(t, responses[1].GetPlaybackContinue())
			for i, message := range queued {
				assert.True(t, proto.Equal(message, responses[i+2].GetAssistant()))
			}
		})
	}
}

func TestSend_NilPayloadCompletionDoesNotUsePlaybackQueue(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	terminal := &protos.ConversationAssistantMessage{Id: "response", Completed: true}

	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	require.NoError(t, streamer.Send(terminal))
	responses := server.sentResponses()
	require.Len(t, responses, 2)
	assert.NotNil(t, responses[0].GetPlaybackPause())
	assert.True(t, proto.Equal(terminal, responses[1].GetAssistant()))
}

func TestSend_FlushPrecedesNewResponseWhileWireSendIsBlocked(t *testing.T) {
	t.Parallel()
	flushStarted := make(chan struct{})
	releaseFlush := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(releaseFlush) }) })
	server := &recordingAssistantTalkServer{onSend: func(response *protos.AssistantTalkResponse) error {
		if response.GetPlaybackFlush() != nil {
			close(flushStarted)
			<-releaseFlush
		}
		return nil
	}}
	streamer := &unidirectionalStreamer{server: server}
	oldMessage := &protos.ConversationAssistantMessage{
		Id: "old", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
	}
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	require.NoError(t, streamer.Send(oldMessage))
	flushDone := make(chan error, 1)
	go func() { flushDone <- streamer.Send(&protos.ConversationPlaybackFlush{}) }()
	select {
	case <-flushStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("flush did not reach server.Send")
	}
	outputStarted := make(chan struct{})
	outputDone := make(chan error, 1)
	go func() {
		close(outputStarted)
		if err := streamer.Send(oldMessage); err != nil {
			outputDone <- err
			return
		}
		outputDone <- streamer.Send(&protos.ConversationAssistantMessage{
			Id: "new", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}},
		})
	}()
	<-outputStarted
	releaseOnce.Do(func() { close(releaseFlush) })
	select {
	case err := <-flushDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("flush did not complete")
	}
	select {
	case err := <-outputDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("new output did not complete")
	}
	responses := server.sentResponses()
	require.Len(t, responses, 4)
	assert.Equal(t, "old", responses[0].GetAssistant().GetId())
	assert.NotNil(t, responses[1].GetPlaybackPause())
	assert.NotNil(t, responses[2].GetPlaybackFlush())
	assert.Equal(t, "new", responses[3].GetAssistant().GetId())
}

func TestSend_ConcurrentPauseAndContinueKeepAudioBehindControls(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{onSend: func(*protos.AssistantTalkResponse) error {
		runtime.Gosched()
		return nil
	}}
	streamer := &unidirectionalStreamer{server: server}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	for range 32 {
		require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
			Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
		}))
	}
	messages := []proto.Message{
		&protos.ConversationPlaybackPause{}, &protos.ConversationPlaybackContinue{},
		&protos.ConversationAssistantMessage{
			Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}},
		},
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(messages)*32)
	for range 32 {
		for _, message := range messages {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- streamer.Send(message)
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	paused := false
	var frames int
	for _, response := range server.sentResponses() {
		switch response.GetData().(type) {
		case *protos.AssistantTalkResponse_PlaybackPause:
			paused = true
		case *protos.AssistantTalkResponse_PlaybackContinue:
			paused = false
		case *protos.AssistantTalkResponse_Assistant:
			assert.False(t, paused, "audio must not reach the wire during a remote pause")
			frames++
		}
	}
	assert.Equal(t, 64, frames, "all buffered and concurrent frames must be sent exactly once")
}

func TestSend_ControlFailuresPreserveLocalSafety(t *testing.T) {
	t.Parallel()
	sendErr := errors.New("send failed")
	server := &recordingAssistantTalkServer{sendErr: sendErr}
	streamer := &unidirectionalStreamer{server: server}
	oldMessage := &protos.ConversationAssistantMessage{
		Id: "old", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
	}
	assert.ErrorIs(t, streamer.Send(&protos.ConversationPlaybackPause{}), sendErr)
	require.NoError(t, streamer.Send(oldMessage))
	assert.ErrorIs(t, streamer.Send(&protos.ConversationPlaybackContinue{}), sendErr)
	require.NoError(t, streamer.Send(oldMessage))
	assert.Empty(t, server.sentResponses())
	server.sendErr = nil
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	responses := server.sentResponses()
	require.Len(t, responses, 3)
	assert.NotNil(t, responses[0].GetPlaybackContinue())
	assert.True(t, proto.Equal(oldMessage, responses[1].GetAssistant()))
	assert.True(t, proto.Equal(oldMessage, responses[2].GetAssistant()))

	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	require.NoError(t, streamer.Send(oldMessage))
	server.sendErr = sendErr
	assert.ErrorIs(t, streamer.Send(&protos.ConversationPlaybackFlush{}), sendErr)
	server.sendErr = nil
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	assert.Len(t, server.sentResponses(), 5)
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "new", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}},
	}))
	assert.Len(t, server.sentResponses(), 6)
}

func TestSend_ReplayFailureReturnsErrorAndRetainsRemainingFrames(t *testing.T) {
	t.Parallel()
	sendErr := errors.New("audio send failed")
	failed := false
	server := &recordingAssistantTalkServer{onSend: func(response *protos.AssistantTalkResponse) error {
		if response.GetAssistant() != nil && !failed {
			failed = true
			return sendErr
		}
		return nil
	}}
	streamer := &unidirectionalStreamer{server: server}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{}))
	for _, frame := range []byte{1, 2} {
		require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
			Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{frame}},
		}))
	}
	assert.ErrorIs(t, streamer.Send(&protos.ConversationPlaybackContinue{}), sendErr)
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{}))
	responses := server.sentResponses()
	require.Len(t, responses, 4)
	assert.Equal(t, []byte{2}, responses[3].GetAssistant().GetAudio())
}

func TestSend_SerializesAllEnvelopes(t *testing.T) {
	t.Parallel()
	var active atomic.Int32
	var overlapped atomic.Bool
	server := &recordingAssistantTalkServer{onSend: func(*protos.AssistantTalkResponse) error {
		if active.Add(1) != 1 {
			overlapped.Store(true)
		}
		runtime.Gosched()
		active.Add(-1)
		return nil
	}}
	streamer := &unidirectionalStreamer{server: server}
	messages := []proto.Message{
		&protos.ConversationPlaybackPause{}, &protos.ConversationPlaybackContinue{},
		&protos.ConversationPlaybackFlush{}, &protos.ConversationInitialization{},
		&protos.ConversationConfiguration{}, &protos.ConversationInterruption{},
		&protos.ConversationUserMessage{}, &protos.ConversationAssistantMessage{},
		&protos.ConversationToolCall{}, &protos.ConversationToolCallResult{},
		&protos.ConversationMetadata{}, &protos.ConversationMetric{},
		&protos.ConversationError{}, &protos.ConversationEvent{},
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(messages)*16)
	for range 16 {
		for _, message := range messages {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- streamer.Send(message)
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.False(t, overlapped.Load(), "server.Send calls must not overlap")
	assert.Len(t, server.sentResponses(), len(messages)*16)
}

func TestRecv_GeneratedCompletionAndLegacyRequests(t *testing.T) {
	t.Parallel()
	initialization := &protos.ConversationInitialization{Assistant: &protos.AssistantDefinition{AssistantId: 1}}
	configuration := &protos.ConversationConfiguration{}
	message := &protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1, 2}}}
	metadata := &protos.ConversationMetadata{}
	metric := &protos.ConversationMetric{}
	complete := &protos.ConversationPlaybackComplete{
		Id: "response", Time: timestamppb.New(time.Unix(123, 456)),
	}
	tests := []struct {
		name    string
		request *protos.AssistantTalkRequest
		want    proto.Message
		wantErr error
	}{
		{"initialization", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Initialization{Initialization: initialization}}, initialization, nil},
		{"configuration", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Configuration{Configuration: configuration}}, configuration, nil},
		{"message", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Message{Message: message}}, message, nil},
		{"metadata", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Metadata{Metadata: metadata}}, metadata, nil},
		{"metric", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Metric{Metric: metric}}, metric, nil},
		{"completion", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_PlaybackComplete{PlaybackComplete: complete}}, complete, nil},
		{"empty completion", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_PlaybackComplete{PlaybackComplete: &protos.ConversationPlaybackComplete{}}}, &protos.ConversationPlaybackComplete{}, nil},
		{"empty request", &protos.AssistantTalkRequest{}, nil, nil},
		{"disconnection remains ignored", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Disconnection{}}, nil, nil},
		{"tool result remains ignored", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_ToolCallResult{}}, nil, nil},
		{"operator audio remains ignored", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_BridgeOperatorAudio{}}, nil, nil},
		{"user bridge audio remains ignored", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_BridgeUserAudio{}}, nil, nil},
		{"invalid initialization", &protos.AssistantTalkRequest{Request: &protos.AssistantTalkRequest_Initialization{Initialization: &protos.ConversationInitialization{}}}, nil, errInvalidInitialization},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := &recordingAssistantTalkServer{requests: []*protos.AssistantTalkRequest{tt.request}}
			streamer := &unidirectionalStreamer{server: server}
			got, err := streamer.Recv()
			require.ErrorIs(t, err, tt.wantErr)
			assert.True(t, proto.Equal(tt.want, got), "received %T, wanted %T", got, tt.want)
			if tt.name == "completion" {
				field := tt.request.ProtoReflect().Descriptor().Fields().ByName("playbackComplete")
				require.NotNil(t, field)
				assert.Equal(t, protoreflect.FieldNumber(10), field.Number())
			}
		})
	}
}

func TestExplicitAudioTerminalSurvivesPauseAndContinue(t *testing.T) {
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackPause{Id: "response"}))
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1, 2}},
	}))
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
	}))
	require.Len(t, server.sentResponses(), 1)
	require.NoError(t, streamer.Send(&protos.ConversationPlaybackContinue{Id: "response"}))
	var messages []*protos.ConversationAssistantMessage
	for _, response := range server.sentResponses() {
		if message := response.GetAssistant(); message != nil {
			messages = append(messages, message)
		}
	}
	require.Len(t, messages, 2)
	assert.Equal(t, []byte{1, 2}, messages[0].GetAudio())
	assert.True(t, messages[1].GetCompleted())
	assert.NotNil(t, messages[1].GetMessage().(*protos.ConversationAssistantMessage_Audio))
	for range 3 {
		server.requests = append(server.requests, &protos.AssistantTalkRequest{
			Request: &protos.AssistantTalkRequest_PlaybackComplete{PlaybackComplete: &protos.ConversationPlaybackComplete{
				Id: "response",
			}},
		})
	}
	for range 3 {
		message, err := streamer.Recv()
		require.NoError(t, err)
		completion, ok := message.(*protos.ConversationPlaybackComplete)
		require.True(t, ok)
		assert.Equal(t, "response", completion.GetId())
	}
}

func TestRecv_PropagatesErrorsAndDoesNotInventPlaybackCompletion(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	require.NoError(t, streamer.Send(&protos.ConversationAssistantMessage{
		Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1}},
	}))
	message, err := streamer.Recv()
	assert.Nil(t, message)
	assert.ErrorIs(t, err, io.EOF)
	recvErr := errors.New("receive failed")
	server.recvErr = recvErr
	message, err = streamer.Recv()
	assert.Nil(t, message)
	assert.ErrorIs(t, err, recvErr)
}
