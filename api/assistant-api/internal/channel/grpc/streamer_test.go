// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_grpc

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type recordingAssistantTalkServer struct {
	grpc.ServerStream
	mu   sync.Mutex
	sent []*protos.AssistantTalkResponse
}

func (s *recordingAssistantTalkServer) Recv() (*protos.AssistantTalkRequest, error) {
	return nil, io.EOF
}

func (s *recordingAssistantTalkServer) Send(response *protos.AssistantTalkResponse) error {
	s.mu.Lock()
	s.sent = append(s.sent, response)
	s.mu.Unlock()
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
	require.NoError(t, streamer.Send(internal_type.PauseOutput{}))
	require.NoError(t, streamer.Send(oldMessage))
	assert.Empty(t, server.sentResponses())
	require.NoError(t, streamer.Send(internal_type.ContinueOutput{}))
	responses := server.sentResponses()
	require.Len(t, responses, 1)
	assert.Same(t, oldMessage, responses[0].GetAssistant())

	require.NoError(t, streamer.Send(internal_type.PauseOutput{}))
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(internal_type.FlushOutput{}))
	require.NoError(t, streamer.Send(oldMessage))
	require.NoError(t, streamer.Send(internal_type.ContinueOutput{}))
	require.Len(t, server.sentResponses(), 1)

	newMessage := &protos.ConversationAssistantMessage{Id: "response-2", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{2}}}
	require.NoError(t, streamer.Send(newMessage))
	responses = server.sentResponses()
	require.Len(t, responses, 2)
	assert.Same(t, newMessage, responses[1].GetAssistant())

	interruption := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_VAD,
	}
	require.NoError(t, streamer.Send(interruption))
	responses = server.sentResponses()
	require.Len(t, responses, 3)
	assert.Same(t, interruption, responses[2].GetInterruption())
}

func TestSend_OutputControlsAreConcurrentSafe(t *testing.T) {
	t.Parallel()
	server := &recordingAssistantTalkServer{}
	streamer := &unidirectionalStreamer{server: server}
	controls := []internal_type.Stream{
		internal_type.PauseOutput{},
		internal_type.ContinueOutput{},
		internal_type.FlushOutput{},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(controls)*32)
	for range 32 {
		for _, control := range controls {
			wg.Add(1)
			go func(control internal_type.Stream) {
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
	assert.Empty(t, server.sentResponses())
}
