package adapter_internal

import (
	"context"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type streamTestStreamer struct {
	ctx      context.Context
	recv     []internal_type.Stream
	recvErr  error
	recvIdx  int
	recvCall int

	mu    sync.Mutex
	sent  []internal_type.Stream
	modes []protos.StreamMode
}

func (s *streamTestStreamer) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *streamTestStreamer) Recv() (internal_type.Stream, error) {
	s.recvCall++
	if s.recvIdx < len(s.recv) {
		msg := s.recv[s.recvIdx]
		s.recvIdx++
		return msg, nil
	}
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	return nil, io.EOF
}

func (s *streamTestStreamer) Send(in internal_type.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, in)
	return nil
}

func (s *streamTestStreamer) NotifyMode(mode protos.StreamMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modes = append(s.modes, mode)
}

func TestTalk_RecvErrorBeforeInitialization_ReturnsNil(t *testing.T) {
	streamer := &streamTestStreamer{recvErr: io.EOF}
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	r := &genericRequestor{
		streamer:         streamer,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycle(),
		sessionCtx:       sessionCtx,
		cancelSession:    cancelSession,
		channels:         adapter_channel.NewRequestorChannels(),
	}

	err := r.Talk(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, streamer.recvCall)
}

func TestOnCallCompletion_EmitsConversationDurationInMilliseconds(t *testing.T) {
	r := &genericRequestor{
		assistantConversation: &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:      adapter_lifecycle.NewMessageLifecycle(),
		channels:              adapter_channel.NewRequestorChannels(),
	}
	r.assistantConversation.Id = 42

	r.OnCallCompletion(time.Now().Add(-2 * time.Second))

	envelope := <-r.channels.BackgroundChannel()
	packet, ok := envelope.Pkt.(internal_type.ObservabilityMetricRecordPacket)
	require.True(t, ok)

	metrics := make(map[string]*protos.Metric, len(packet.Record.Metrics))
	for _, metric := range packet.Record.Metrics {
		metrics[metric.GetName()] = metric
	}

	durationMetric := metrics[observability.MetricConversationDuration]
	require.NotNil(t, durationMetric)
	assert.NotContains(t, metrics, "duration")

	durationMs, err := strconv.ParseInt(durationMetric.GetValue(), 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, durationMs, int64(1900))
	assert.Less(t, durationMs, int64(3000))
}

func TestTalk_BuffersPacketsBeforeInitialization(t *testing.T) {
	streamer := &streamTestStreamer{
		recv: []internal_type.Stream{
			&protos.ConversationUserMessage{
				Message: &protos.ConversationUserMessage_Text{Text: "hello"},
			},
			&protos.ConversationMetadata{
				AssistantConversationId: 42,
				Metadata: []*protos.Metadata{{
					Key:   "k",
					Value: "v",
				}},
			},
			&protos.ConversationMetric{
				AssistantConversationId: 42,
				Metrics: []*protos.Metric{{
					Name:  "status",
					Value: "in_progress",
				}},
			},
			&protos.ConversationEvent{
				Name: "session",
				Data: map[string]string{"kind": "noop"},
				Time: timestamppb.Now(),
			},
			&protos.ConversationDisconnection{
				Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
			},
		},
		recvErr: io.EOF,
	}

	sessionCtx, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	r := &genericRequestor{
		streamer:         streamer,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycle(),
		sessionCtx:       sessionCtx,
		cancelSession:    cancelSession,
		// Before initialization completes, packets should be buffered in channels.
		channels: func() *adapter_channel.RequestorChannels {
			ch := adapter_channel.NewRequestorChannels()
			return ch
		}(),
	}

	err := r.Talk(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, len(r.channels.ControlChannel()))
	assert.Equal(t, 1, len(r.channels.BootstrapChannel()))
	assert.Equal(t, 1, len(r.channels.IngressChannel()))
	assert.Equal(t, 0, len(r.channels.EgressChannel()))
	assert.Equal(t, 0, len(r.channels.DataChannel()))
	assert.Equal(t, 3, len(r.channels.BackgroundChannel()))
	assert.Equal(t, 0, len(streamer.modes))
}

func TestNotify_ForwardsAllActionData(t *testing.T) {
	streamer := &streamTestStreamer{}
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	r := &genericRequestor{
		streamer:         streamer,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycle(),
		sessionCtx:       sessionCtx,
		cancelSession:    cancelSession,
		channels:         adapter_channel.NewRequestorChannels(),
	}

	a := &protos.ConversationEvent{Name: "alpha"}
	b := &protos.ConversationMetric{
		AssistantConversationId: 77,
		Metrics:                 []*protos.Metric{{Name: "m1", Value: "v1"}},
	}

	err := r.Notify(context.Background(), a, b)
	require.NoError(t, err)
	require.Len(t, streamer.sent, 2)
	assert.Same(t, a, streamer.sent[0])
	assert.Same(t, b, streamer.sent[1])
}

func TestOnNotifyAssistantConfiguration_ForwardsInitializationConfig(t *testing.T) {
	streamer := &streamTestStreamer{}
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	conversationInitialization := &protos.ConversationInitialization{
		StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
	}
	r := &genericRequestor{
		streamer:         streamer,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycle(),
		sessionCtx:       sessionCtx,
		cancelSession:    cancelSession,
		channels:         adapter_channel.NewRequestorChannels(),
	}
	conversation := &internal_conversation_entity.AssistantConversation{}
	conversation.Id = 33

	r.OnNotifyAssistantConfiguration(context.Background(), conversationInitialization, conversation)

	require.Eventually(t, func() bool {
		streamer.mu.Lock()
		defer streamer.mu.Unlock()
		return len(streamer.sent) == 1
	}, time.Second, 10*time.Millisecond)
	require.Len(t, streamer.sent, 1)
	init, ok := streamer.sent[0].(*protos.ConversationInitialization)
	require.True(t, ok)
	assert.Same(t, conversationInitialization, init)
}
