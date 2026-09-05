// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestWebhookCollectorWithoutDependenciesIsNoop(t *testing.T) {
	collector := webhook.New(context.Background(), webhook.Config{})
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected no-op collector, got %T", collector)
	}
}

func newTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("webrtc-test"), commons.EnableFile(false))
	require.NoError(t, err)
	return l
}

func newTestObserver(t *testing.T) observability.Recorder {
	t.Helper()
	observer := observability.New(observability.WithGlobalScope(observability.GlobalScope{
		OrganizationID: 1,
		ProjectID:      1,
	}))
	t.Cleanup(func() { _ = observer.Close(context.Background()) })
	return observer
}

type testObservabilityCollector struct {
	mu       sync.Mutex
	logs     []observability.RecordLog
	events   []observability.RecordEvent
	metrics  []observability.RecordMetric
	metadata []observability.RecordMetadata
	webhooks []observability.RecordWebhook
}

func (c *testObservabilityCollector) Key() string {
	return "test"
}

func (c *testObservabilityCollector) Collect(_ context.Context, _ observability.Scope, _ observability.Context, record observability.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch typed := record.(type) {
	case observability.RecordLog:
		c.logs = append(c.logs, typed)
	case observability.RecordEvent:
		c.events = append(c.events, typed)
	case observability.RecordMetric:
		c.metrics = append(c.metrics, typed)
	case observability.RecordMetadata:
		c.metadata = append(c.metadata, typed)
	case observability.RecordWebhook:
		c.webhooks = append(c.webhooks, typed)
	}
	return nil
}

func (c *testObservabilityCollector) Close(context.Context) error {
	return nil
}

// newTestStreamer creates a WebRTC streamer with test-owned dependencies.
func newTestStreamer(t *testing.T) *webrtcStreamer {
	t.Helper()
	logger := newTestLogger(t)
	opusCodec, err := webrtc_internal.NewOpusCodec()
	require.NoError(t, err)
	resampler, err := internal_audio_resampler.GetResampler(logger)
	require.NoError(t, err)

	return &webrtcStreamer{
		BaseStreamer:     channel_base.NewBaseStreamerWithChannelCapacity(logger, 16, 16),
		peerConfig:       webrtc_internal.DefaultConfig(),
		sessionID:        "test-session",
		resampler:        resampler,
		opusCodec:        opusCodec,
		currentMode:      protos.StreamMode_STREAM_MODE_TEXT,
		sessionState:     webrtc_internal.SessionState{Scope: observability.ProjectScope{}},
		audioBufferState: newWebRTCAudioBufferState(),
		flushAudioCh:     make(chan struct{}, 1),
		observer:         newTestObserver(t),
	}
}

type fakeAmbientMixer struct {
	cfg        internal_ambient.Config
	ambientOut []byte
}

type failingGRPCStream struct {
	sendErr error
	recvMsg []*protos.WebTalkRequest
	recvErr error
}

func (f *failingGRPCStream) Recv() (*protos.WebTalkRequest, error) {
	if len(f.recvMsg) > 0 {
		msg := f.recvMsg[0]
		f.recvMsg = f.recvMsg[1:]
		return msg, nil
	}
	if f.recvErr != nil {
		return nil, f.recvErr
	}
	return nil, io.EOF
}

func (f *failingGRPCStream) Send(*protos.WebTalkResponse) error {
	return f.sendErr
}

func (f *failingGRPCStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *failingGRPCStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *failingGRPCStream) SetTrailer(metadata.MD) {}

func (f *failingGRPCStream) Context() context.Context {
	return context.Background()
}

func (f *failingGRPCStream) SendMsg(any) error {
	return nil
}

func (f *failingGRPCStream) RecvMsg(any) error {
	return io.EOF
}

func (f *fakeAmbientMixer) Configure(cfg internal_ambient.Config) error {
	f.cfg = cfg
	return nil
}

func (f *fakeAmbientMixer) Mix(primary []byte) ([]byte, error) {
	if primary == nil {
		return append([]byte(nil), f.ambientOut...), nil
	}
	return append([]byte(nil), primary...), nil
}

func (f *fakeAmbientMixer) Reset() {}

func (f *fakeAmbientMixer) CurrentConfig() internal_ambient.Config { return f.cfg }

func requireObservabilityEvent(t *testing.T, collector *testObservabilityCollector, eventType string) observability.RecordEvent {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			collector.mu.Lock()
			for _, event := range collector.events {
				if event.Attributes[webrtc_internal.DataType] == eventType {
					collector.mu.Unlock()
					return event
				}
			}
			collector.mu.Unlock()
		case <-deadline:
			t.Fatalf("timed out waiting for WebRTC observability event %q", eventType)
		}
	}
}

func requireObservabilityLog(t *testing.T, collector *testObservabilityCollector, eventType string) observability.RecordLog {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			collector.mu.Lock()
			for _, log := range collector.logs {
				if log.Attributes[webrtc_internal.DataType] == eventType {
					collector.mu.Unlock()
					return log
				}
			}
			collector.mu.Unlock()
		case <-deadline:
			t.Fatalf("timed out waiting for WebRTC observability log %q", eventType)
		}
	}
}

func requireObservabilityWebhook(t *testing.T, collector *testObservabilityCollector, event observability.EventName) observability.RecordWebhook {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			collector.mu.Lock()
			for _, webhook := range collector.webhooks {
				if webhook.Event == event {
					collector.mu.Unlock()
					return webhook
				}
			}
			collector.mu.Unlock()
		case <-deadline:
			t.Fatalf("timed out waiting for WebRTC webhook %q", event)
		}
	}
}

func requireObservabilityMetric(t *testing.T, collector *testObservabilityCollector, name string) *protos.Metric {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			collector.mu.Lock()
			for _, record := range collector.metrics {
				for _, metric := range record.Metrics {
					if metric.Name == name {
						collector.mu.Unlock()
						return metric
					}
				}
			}
			collector.mu.Unlock()
		case <-deadline:
			t.Fatalf("timed out waiting for WebRTC metric %q", name)
		}
	}
}

func TestBuildGRPCResponse_Disconnection(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationDisconnection{}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetDisconnection())
}

func TestBuildGRPCResponse_AssistantText(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello world"},
	}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetAssistant())
}

func TestBuildGRPCResponse_ToolCall(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetToolCall())
}

func TestBuildGRPCResponse_Event(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationEvent{Name: "test", Data: map[string]string{"key": "val"}}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetEvent())
}

func TestNew_UsesConfiguredICEServers(t *testing.T) {
	t.Setenv("TURN_USERNAME", "turn-user")
	t.Setenv("TURN_CREDENTIAL", "turn-secret")
	streamer, err := New(
		WithContext(context.Background()),
		WithLogger(newTestLogger(t)),
		WithServer(&failingGRPCStream{sendErr: io.EOF}),
		WithObserver(newTestObserver(t)),
		WithServerConfig(&assistant_config.WebRTCConfig{
			ICEServers: []assistant_config.WebRTCICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs: []string{
						"turn:turn.rapida.ai:3478?transport=udp",
						"turn:turn.rapida.ai:3478?transport=tcp",
						"turns:turn.rapida.ai:443?transport=tcp",
					},
					Username:   "${TURN_USERNAME}",
					Credential: "${TURN_CREDENTIAL}",
				},
			},
			ICETransportPolicy: webrtc_internal.ICETransportPolicyRelay,
		}),
	)
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	require.Len(t, s.peerConfig.ICEServers, 2)
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, s.peerConfig.ICEServers[0].URLs)
	assert.Equal(t, []string{
		"turn:turn.rapida.ai:3478?transport=udp",
		"turn:turn.rapida.ai:3478?transport=tcp",
		"turns:turn.rapida.ai:443?transport=tcp",
	}, s.peerConfig.ICEServers[1].URLs)
	assert.Equal(t, "turn-user", s.peerConfig.ICEServers[1].Username)
	assert.Equal(t, "turn-secret", s.peerConfig.ICEServers[1].Credential)
	assert.Equal(t, webrtc_internal.ICETransportPolicyRelay, s.peerConfig.ICETransportPolicy)
}

func TestNew_DefaultsToGoogleSTUN(t *testing.T) {
	t.Parallel()
	streamer, err := New(
		WithContext(context.Background()),
		WithLogger(newTestLogger(t)),
		WithServer(&failingGRPCStream{sendErr: io.EOF}),
		WithObserver(newTestObserver(t)),
		WithServerConfig(&assistant_config.WebRTCConfig{}),
	)
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	require.Len(t, s.peerConfig.ICEServers, 2)
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, s.peerConfig.ICEServers[0].URLs)
	assert.Equal(t, []string{"stun:stun1.l.google.com:19302"}, s.peerConfig.ICEServers[1].URLs)
	assert.Equal(t, webrtc_internal.ICETransportPolicyAll, s.peerConfig.ICETransportPolicy)
}

func TestNew_InvalidICETransportPolicyFallsBackToAll(t *testing.T) {
	t.Parallel()
	streamer, err := New(
		WithContext(context.Background()),
		WithLogger(newTestLogger(t)),
		WithServer(&failingGRPCStream{sendErr: io.EOF}),
		WithObserver(newTestObserver(t)),
		WithServerConfig(&assistant_config.WebRTCConfig{
			ICETransportPolicy: "invalid",
		}),
	)
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	assert.Equal(t, webrtc_internal.ICETransportPolicyAll, s.peerConfig.ICETransportPolicy)
}

func TestNew_IgnoresEmptyICEServerEntries(t *testing.T) {
	t.Setenv("WEBRTC_TURN_URL", "turn:turn.rapida.ai:3478?transport=tcp")
	streamer, err := New(
		WithContext(context.Background()),
		WithLogger(newTestLogger(t)),
		WithServer(&failingGRPCStream{sendErr: io.EOF}),
		WithObserver(newTestObserver(t)),
		WithServerConfig(&assistant_config.WebRTCConfig{
			ICEServers: []assistant_config.WebRTCICEServer{
				{URLs: []string{"", "   "}},
				{URLs: nil},
				{
					URLs:       []string{" ${WEBRTC_TURN_URL} "},
					Username:   "turn-user",
					Credential: "turn-secret",
				},
			},
		}),
	)
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	require.Len(t, s.peerConfig.ICEServers, 1)
	assert.Equal(t, []string{"turn:turn.rapida.ai:3478?transport=tcp"}, s.peerConfig.ICEServers[0].URLs)
	assert.Equal(t, "turn-user", s.peerConfig.ICEServers[0].Username)
	assert.Equal(t, "turn-secret", s.peerConfig.ICEServers[0].Credential)
}

func TestDispatchOutput_SendFailureClosesStreamer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{sendErr: errors.New("client closed")}

	ok := s.dispatchOutput(&protos.WebTalkResponse{})

	assert.False(t, ok)
	assert.True(t, s.sessionState.CloseStarted())
	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR, disc.GetType())
	default:
		t.Fatal("expected disconnection on gRPC send failure")
	}
}

func TestDispatchOutput_NormalStreamCloseUsesUserDisconnect(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{sendErr: io.EOF}

	ok := s.dispatchOutput(&protos.WebTalkResponse{})

	assert.False(t, ok)
	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER, disc.GetType())
	default:
		t.Fatal("expected disconnection on gRPC stream close")
	}
}

func TestRunGrpcReader_ReceiveFailureUsesErrorDisconnect(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{recvErr: errors.New("recv failed")}

	s.runGrpcReader()

	assert.True(t, s.sessionState.CloseStarted())
	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR, disc.GetType())
	default:
		t.Fatal("expected disconnection on gRPC receive failure")
	}
}

func TestRunGrpcReader_NormalStreamCloseUsesUserDisconnect(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{recvErr: io.EOF}

	s.runGrpcReader()

	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER, disc.GetType())
	default:
		t.Fatal("expected disconnection on gRPC stream close")
	}
}

func TestRunGrpcReader_ClientDisconnectionClosesStreamer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{
		recvMsg: []*protos.WebTalkRequest{{
			Request: &protos.WebTalkRequest_Disconnection{
				Disconnection: &protos.ConversationDisconnection{
					Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
				},
			},
		}},
	}

	s.runGrpcReader()

	assert.True(t, s.sessionState.CloseStarted())
	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER, disc.GetType())
	default:
		t.Fatal("expected disconnection on client disconnection request")
	}
}

func TestServerSignaling_UsesActiveSignalingSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"

	s.signalConfig()

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.Equal(t, "media-signaling-session", signaling.GetSessionId())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server signaling")
	}
}

func TestServerSignaling_FallsBackToStreamerSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.signalClear()

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.Equal(t, s.sessionID, signaling.GetSessionId())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server signaling")
	}
}

func TestServerSignaling_ConfigIncludesICEServersAndAudioDefaults(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	s.peerConfig.ICEServers = []webrtc_internal.ICEServer{
		{
			URLs:       []string{"turn:turn.rapida.ai:3478?transport=tcp"},
			Username:   "turn-user",
			Credential: "turn-secret",
		},
	}

	s.signalConfig()

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.Equal(t, "media-signaling-session", signaling.GetSessionId())
		require.NotNil(t, signaling.GetConfig())
		require.Len(t, signaling.GetConfig().GetIceServers(), 1)
		assert.Equal(t, []string{"turn:turn.rapida.ai:3478?transport=tcp"}, signaling.GetConfig().GetIceServers()[0].GetUrls())
		assert.Equal(t, "turn-user", signaling.GetConfig().GetIceServers()[0].GetUsername())
		assert.Equal(t, "turn-secret", signaling.GetConfig().GetIceServers()[0].GetCredential())
		assert.Equal(t, "opus", signaling.GetConfig().GetAudioCodec())
		assert.Equal(t, int32(webrtc_internal.OpusSampleRate), signaling.GetConfig().GetSampleRate())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC config")
	}
}

func TestServerTrickleICECandidate_UsesActiveSignalingSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	s.signalOfferSent = true
	sdpMid := "audio"
	sdpMLineIndex := uint16(0)
	usernameFragment := "ufrag"

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:           &sdpMid,
		SDPMLineIndex:    &sdpMLineIndex,
		UsernameFragment: &usernameFragment,
	}, mediaSessionID)

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		require.NotNil(t, signaling.GetIceCandidate())
		assert.Equal(t, "media-signaling-session", signaling.GetSessionId())
		assert.NotEmpty(t, signaling.GetIceCandidate().GetCandidate())
		assert.Equal(t, sdpMid, signaling.GetIceCandidate().GetSdpMid())
		assert.Equal(t, int32(sdpMLineIndex), signaling.GetIceCandidate().GetSdpMLineIndex())
		assert.Equal(t, usernameFragment, signaling.GetIceCandidate().GetUsernameFragment())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server ICE candidate")
	}
}

func TestServerTrickleICECandidate_CachesUntilOfferSignaled(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	sdpMid := "audio"
	sdpMLineIndex := uint16(0)

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:     "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:        &sdpMid,
		SDPMLineIndex: &sdpMLineIndex,
	}, mediaSessionID)

	s.Mu.Lock()
	pendingCandidateCount := len(s.signalPendingLocalICECandidates)
	signalOfferSent := s.signalOfferSent
	s.Mu.Unlock()

	assert.Equal(t, 1, pendingCandidateCount)
	assert.False(t, signalOfferSent)
	select {
	case msg := <-s.OutputCh:
		t.Fatalf("ICE candidate should not be sent before offer is signaled: %T", msg)
	default:
	}
}

func TestQueueLocalICECandidate_EnqueuesWebRTCOperation(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 1)
	mediaSessionID := s.sessionState.StartMediaSession()
	sdpMid := "audio"
	sdpMLineIndex := uint16(0)

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:     "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:        &sdpMid,
		SDPMLineIndex: &sdpMLineIndex,
	}, mediaSessionID)

	select {
	case operation := <-s.webrtcOperationCh:
		assert.Equal(t, webrtc_internal.WebRTCOperationSendLocalICECandidate, operation.Kind)
		assert.Equal(t, mediaSessionID, operation.MediaSessionID)
		assert.Equal(t, "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host", operation.LocalICECandidate.Candidate)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for local ICE WebRTC operation")
	}
}

func TestQueueLocalICECandidate_IgnoresStaleMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 1)
	activeMediaSessionID := s.sessionState.StartMediaSession()

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate: "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
	}, activeMediaSessionID+1)

	select {
	case operation := <-s.webrtcOperationCh:
		t.Fatalf("stale local ICE candidate should not enqueue operation: %+v", operation)
	default:
	}
}

func TestInitiateWebRTCHandshake_SendsOfferBeforeTrickleCandidates(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()

	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    mediaSessionID,
		SignalMediaConfig: true,
	})

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.NotNil(t, signaling.GetConfig())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC config")
	}

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		require.NotNil(t, signaling.GetSdp())
		assert.Equal(t, protos.WebRTCSDP_OFFER, signaling.GetSdp().GetType())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC offer")
	}
}

func TestHandleConfigurationMessage_SameModeNoop(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)

	s.Mu.Lock()
	peerConnection := s.peerConnection
	s.Mu.Unlock()
	assert.Nil(t, peerConnection, "peer connection should not be created for same mode")
}

func TestWebRTCRemoteAudioTrack_SelectedCodecUsesTrackCodecWhenAvailable(t *testing.T) {
	t.Parallel()
	trackCodec := pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus},
		PayloadType:        webrtc_internal.OpusPayloadType,
	}
	receiverCodecs := []pionwebrtc.RTPCodecParameters{
		{RTPCodecCapability: pionwebrtc.RTPCodecCapability{MimeType: "audio/PCMU"}},
	}

	selectedCodec, ok := webrtc_internal.WebRTCRemoteAudioTrack{
		TrackCodec:     trackCodec,
		ReceiverCodecs: receiverCodecs,
	}.SelectedCodec()

	require.True(t, ok)
	assert.Equal(t, pionwebrtc.MimeTypeOpus, selectedCodec.MimeType)
	assert.Equal(t, pionwebrtc.PayloadType(webrtc_internal.OpusPayloadType), selectedCodec.PayloadType)
}

func TestWebRTCRemoteAudioTrack_SelectedCodecFallsBackToNegotiatedReceiverCodec(t *testing.T) {
	t.Parallel()
	receiverCodecs := []pionwebrtc.RTPCodecParameters{
		{RTPCodecCapability: pionwebrtc.RTPCodecCapability{MimeType: "audio/PCMU"}},
		{
			RTPCodecCapability: pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus},
			PayloadType:        webrtc_internal.OpusPayloadType,
		},
	}

	selectedCodec, ok := webrtc_internal.WebRTCRemoteAudioTrack{
		ReceiverCodecs: receiverCodecs,
	}.SelectedCodec()

	require.True(t, ok)
	assert.Equal(t, pionwebrtc.MimeTypeOpus, selectedCodec.MimeType)
	assert.Equal(t, pionwebrtc.PayloadType(webrtc_internal.OpusPayloadType), selectedCodec.PayloadType)
}

func TestWebRTCRemoteAudioTrack_SelectedCodecReturnsFalseWithoutNegotiatedCodec(t *testing.T) {
	t.Parallel()

	_, ok := webrtc_internal.WebRTCRemoteAudioTrack{}.SelectedCodec()

	assert.False(t, ok)
}

func TestTryStartRemoteAudioReader_AllowsOneReaderPerMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	peerConnection := &pionwebrtc.PeerConnection{}
	mediaSessionID := s.sessionState.StartMediaSession()
	s.peerConnection = peerConnection

	started := s.tryStartRemoteAudioReader(peerConnection, mediaSessionID)
	duplicateStarted := s.tryStartRemoteAudioReader(peerConnection, mediaSessionID)
	s.mediaWorkers.Done()

	assert.True(t, started)
	assert.False(t, duplicateStarted)
	assert.Equal(t, mediaSessionID, s.sessionState.RemoteAudioReaderMediaSessionID())
}

func TestTryStartRemoteAudioReader_RejectsStalePeerConnection(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	activePeerConnection := &pionwebrtc.PeerConnection{}
	stalePeerConnection := &pionwebrtc.PeerConnection{}
	mediaSessionID := s.sessionState.StartMediaSession()
	s.peerConnection = activePeerConnection

	started := s.tryStartRemoteAudioReader(stalePeerConnection, mediaSessionID)

	assert.False(t, started)
	assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
}

func TestShouldRestartConnectedNoRemoteAudioTrack(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	watchdogCheckedAt := time.Now()
	mediaHealthState := webrtc_internal.MediaHealthState{
		PeerConnectedAt: watchdogCheckedAt.Add(-webrtc_internal.ConnectedNoUserAudioThreshold),
	}

	missingRemoteAudioTrackMediaSessionID, shouldRestartMissingRemoteAudioTrack := s.shouldRestartConnectedNoRemoteAudioTrack(mediaHealthState, watchdogCheckedAt)

	assert.True(t, shouldRestartMissingRemoteAudioTrack)
	assert.Equal(t, mediaSessionID, missingRemoteAudioTrackMediaSessionID)
}

func TestShouldRestartConnectedNoRemoteAudioTrack_SkipsWhenTrackReceived(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	assert.True(t, s.sessionState.TryStartRemoteAudioReader(mediaSessionID))
	watchdogCheckedAt := time.Now()
	mediaHealthState := webrtc_internal.MediaHealthState{
		PeerConnectedAt: watchdogCheckedAt.Add(-webrtc_internal.ConnectedNoUserAudioThreshold),
	}

	missingRemoteAudioTrackMediaSessionID, shouldRestartMissingRemoteAudioTrack := s.shouldRestartConnectedNoRemoteAudioTrack(mediaHealthState, watchdogCheckedAt)

	assert.False(t, shouldRestartMissingRemoteAudioTrack)
	assert.Zero(t, missingRemoteAudioTrackMediaSessionID)
}

func TestShouldRestartConnectedNoRemoteAudioTrack_SkipsBeforeThreshold(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.sessionState.StartMediaSession()
	watchdogCheckedAt := time.Now()
	mediaHealthState := webrtc_internal.MediaHealthState{
		PeerConnectedAt: watchdogCheckedAt.Add(-(webrtc_internal.ConnectedNoUserAudioThreshold - time.Millisecond)),
	}

	missingRemoteAudioTrackMediaSessionID, shouldRestartMissingRemoteAudioTrack := s.shouldRestartConnectedNoRemoteAudioTrack(mediaHealthState, watchdogCheckedAt)

	assert.False(t, shouldRestartMissingRemoteAudioTrack)
	assert.Zero(t, missingRemoteAudioTrackMediaSessionID)
}

func TestHandleConfigurationMessage_AudioNegotiatingNoop(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT
	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
	s.signalingSessionID = "active-signaling"

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)

	assert.Equal(t, mediaSessionID, s.sessionState.ActiveMediaSessionID())
	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.MediaStateAudioNegotiating, s.sessionState.MediaState())
	assert.Equal(t, "active-signaling", s.signalingSessionID)
	assert.Nil(t, s.peerConnection)
	s.Mu.Unlock()
	select {
	case msg := <-s.OutputCh:
		t.Fatalf("duplicate audio mode should not signal or restart media: %T", msg)
	default:
	}
}

func TestHandleConfigurationMessage_TextToAudioStartsNegotiation(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)
	t.Cleanup(func() { s.stopMediaSession() })

	s.Mu.Lock()
	mode := s.currentMode
	mediaState := s.sessionState.MediaState()
	peerConnection := s.peerConnection
	s.Mu.Unlock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, mode, "current mode changes after peer connects")
	assert.Equal(t, webrtc_internal.MediaStateAudioNegotiating, mediaState)
	assert.NotNil(t, peerConnection)
}

func TestHandleConfigurationMessage_TextStopsAudioNegotiationBeforeAudioConnected(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.BaseStreamer = channel_base.NewBaseStreamerWithChannelCapacity(s.Logger, 64, 64)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)
	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)

	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	assert.False(t, s.sessionState.PeerConnected())
	assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
	s.Mu.Lock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Empty(t, s.signalingSessionID)
	assert.Nil(t, s.peerConnection)
	assert.Nil(t, s.assistantAudioTrack)
	assert.Nil(t, s.assistantRTPSender)
	s.Mu.Unlock()
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	mediaSessionID := s.sessionState.StartMediaSession()

	err := s.Close()
	assert.NoError(t, err)

	err = s.Close()
	assert.NoError(t, err)

	assert.True(t, s.sessionState.CloseStarted())
	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCDisconnected)
	payload, ok := webhook.Payload.(observability.WebRTCDisconnectedWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, "closed", payload.Reason)
	assert.Len(t, collector.webhooks, 1)
}

func TestClose_ConcurrentSafe(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	var wg sync.WaitGroup
	closeCount := 20

	for i := 0; i < closeCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Close()
		}()
	}

	wg.Wait()
	assert.True(t, s.sessionState.CloseStarted())
}

func TestResetAudioSession_ClearsState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	previousMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	now := time.Now()
	s.mediaHealthState = webrtc_internal.MediaHealthState{
		ICEStartedAt:                           now,
		PeerConnectedAt:                        now,
		FirstUserAudioReceivedAt:               now,
		LastUserAudioReceivedAt:                now,
		UserAudioReadErrors:                    2,
		UserAudioConsecutiveReadErrors:         1,
		UserAudioEmptyRTPPayloads:              3,
		UserAudioRTPUnmarshalFailures:          4,
		UserAudioOpusDecodeFailures:            5,
		UserAudioResampleFailures:              6,
		FirstAssistantAudioQueuedAt:            now,
		LastAssistantAudioQueuedAt:             now,
		LastAssistantFrameSentAt:               now,
		AssistantFrameWriteFailures:            7,
		ConsecutiveAssistantFrameWriteFailures: 8,
		LastAssistantFrameWriteFailureAt:       now,
		ReceiverReports:                        9,
		LastReceiverReportAt:                   now,
		LastReceiverReportFractionLost:         10,
		LastReceiverReportPacketLossPercent:    9.5,
		LastReceiverReportTotalLost:            11,
		LastReceiverReportJitterMs:             11.5,
		LastReceiverReportRoundTripTimeMs:      12,
		LastReceiverReportRoundTripTimeUsable:  true,
	}

	s.stopMediaSessionAndFallbackToText()

	assert.False(t, s.sessionState.PeerConnected(), "peerConnected should be false after reset")
	assert.Greater(t, s.sessionState.ActiveMediaSessionID(), previousMediaSessionID)
	s.Mu.Lock()
	assert.Nil(t, s.peerConnection, "peer connection should be nil after reset")
	assert.Nil(t, s.assistantAudioTrack, "assistant audio track should be nil after reset")
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	assert.Equal(t, webrtc_internal.MediaHealthState{}, s.mediaHealthState)
	s.Mu.Unlock()
}

func TestHandleClientSignaling_IgnoresStaleSignalingSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"

	s.queueClientSignal(&protos.ClientSignaling{
		SessionId: "stale-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{Candidate: "candidate:1 1 udp 1 127.0.0.1 9 typ host"},
		},
	})

	assert.False(t, s.sessionState.CloseStarted())
}

func TestHandleClientSignal_QueuesRemoteICEUntilAnswer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "current-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{
				Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
				SdpMid:           "audio",
				SdpMLineIndex:    0,
				UsernameFragment: "remote",
			},
		},
	})

	s.Mu.Lock()
	defer s.Mu.Unlock()
	require.Len(t, s.signalPendingRemoteICECandidates, 1)
	assert.Equal(t, "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host", s.signalPendingRemoteICECandidates[0].Candidate)
}

func TestHandleClientSignal_CapsPendingRemoteICEBeforeAnswer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	for i := 0; i < webrtc_internal.PendingRemoteICECandidateLimit+1; i++ {
		s.handleClientSignal(&protos.ClientSignaling{
			SessionId: "current-signaling-session",
			Message: &protos.ClientSignaling_IceCandidate{
				IceCandidate: &protos.ICECandidate{
					Candidate:        fmt.Sprintf("candidate:%d 1 udp 2130706431 127.0.0.1 9 typ host", i+1),
					SdpMid:           "audio",
					SdpMLineIndex:    0,
					UsernameFragment: "remote",
				},
			},
		})
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()
	assert.Len(t, s.signalPendingRemoteICECandidates, webrtc_internal.PendingRemoteICECandidateLimit)
}

func TestStopMediaSession_InvalidatesCurrentMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	previousMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)

	s.stopMediaSession()

	assert.False(t, s.sessionState.PeerConnected())
	assert.Greater(t, s.sessionState.ActiveMediaSessionID(), previousMediaSessionID)
	s.Mu.Lock()
	assert.Nil(t, s.peerConnection)
	assert.Nil(t, s.assistantAudioTrack)
	assert.Nil(t, s.assistantRTPSender)
	assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
	assert.Nil(t, s.mediaCtx)
	assert.Nil(t, s.cancelMedia)
	s.Mu.Unlock()
}

func TestHandlePeerState_ClosedStopsMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioConnected)

	s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateClosed, time.Now())

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	s.Mu.Unlock()
}

func TestHandlePeerState_ConnectedMarksAudioConnected(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
	connectedAt := time.Now()
	s.mediaHealthState.StartICE(connectedAt.Add(-25 * time.Millisecond))

	s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateConnected, connectedAt)

	assert.True(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_AUDIO, s.currentMode)
	assert.Equal(t, webrtc_internal.MediaStateAudioConnected, s.sessionState.MediaState())
	assert.Equal(t, connectedAt, s.mediaHealthState.PeerConnectedAt)
	s.Mu.Unlock()
	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCConnected)
	payload, ok := webhook.Payload.(observability.WebRTCConnectedWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, int64(25), payload.ICELatencyMs)
	metric := requireObservabilityMetric(t, collector, observability.MetricICELatencyMs)
	assert.Equal(t, "25", metric.Value)
}

func TestQueueMediaSessionRestart_QueuesLifecycleEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.mediaLifecycleCh = make(chan webrtc_internal.MediaLifecycleEvent, 1)
	requestedAt := time.Now()

	s.queueMediaSessionRestart(42, webrtc_internal.ReasonPeerFailed, requestedAt)

	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, webrtc_internal.MediaLifecycleEventRestart, event.Kind)
		assert.Equal(t, uint64(42), event.MediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonPeerFailed, event.Reason)
		assert.Equal(t, requestedAt, event.RequestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for media lifecycle event")
	}
}

func TestQueueMediaSessionRecovery_QueuesLifecycleEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.mediaLifecycleCh = make(chan webrtc_internal.MediaLifecycleEvent, 1)
	requestedAt := time.Now()

	s.queueMediaSessionRecovery(42, webrtc_internal.ReasonICEFailed, requestedAt)

	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, webrtc_internal.MediaLifecycleEventRecover, event.Kind)
		assert.Equal(t, uint64(42), event.MediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonICEFailed, event.Reason)
		assert.Equal(t, requestedAt, event.RequestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for media lifecycle event")
	}
}

func TestWebRTCOperationKind_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "send_offer", webrtc_internal.WebRTCOperationSendOffer.String())
	assert.Equal(t, "apply_remote_answer", webrtc_internal.WebRTCOperationApplyRemoteAnswer.String())
	assert.Equal(t, "add_remote_ice_candidate", webrtc_internal.WebRTCOperationAddRemoteICECandidate.String())
	assert.Equal(t, "send_local_ice_candidate", webrtc_internal.WebRTCOperationSendLocalICECandidate.String())
	assert.Equal(t, "restart_ice", webrtc_internal.WebRTCOperationRestartICE.String())
	assert.Equal(t, "ice_gathering_complete", webrtc_internal.WebRTCOperationICEGatheringComplete.String())
	assert.Equal(t, "unknown", webrtc_internal.WebRTCOperationKind(999).String())
}

func TestWebRTCOperationLoop_SendsInitialOfferBeforeTrickleCandidates(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, webrtc_internal.WebRTCOperationChannelSize)
	go s.runWebRTCOperationLoop()
	t.Cleanup(s.Cancel)

	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	sdpMid := "audio"
	sdpMLineIndex := uint16(0)
	usernameFragment := "ufrag"
	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:           &sdpMid,
		SDPMLineIndex:    &sdpMLineIndex,
		UsernameFragment: &usernameFragment,
	}, mediaSessionID)

	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    mediaSessionID,
		SignalMediaConfig: true,
	})

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.NotNil(t, signaling.GetConfig())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC config")
	}
	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.NotNil(t, signaling.GetSdp())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC offer")
	}
	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.NotNil(t, signaling.GetIceCandidate())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC ICE candidate")
	}
}

func TestWebRTCOperationLoop_HandlesICEGatheringComplete(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetICEGatheringActive(true)

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationICEGatheringComplete,
		MediaSessionID: mediaSessionID,
	})

	assert.False(t, s.sessionState.ICEGatheringActive())
	assert.False(t, s.sessionState.DeferredICERestartPending(mediaSessionID))
}

func TestWebRTCOperation_DefersICERestartDuringGathering(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetICEGatheringActive(true)

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationRestartICE,
		MediaSessionID: mediaSessionID,
		Reason:         webrtc_internal.ReasonICEFailed,
		RequestedAt:    time.Now(),
		OfferOptions:   &pionwebrtc.OfferOptions{ICERestart: true},
	})

	assert.True(t, s.sessionState.ICEGatheringActive())
	assert.True(t, s.sessionState.DeferredICERestartPending(mediaSessionID))
	event := requireObservabilityEvent(t, collector, webrtc_internal.EventICERestartDeferred)
	assert.Equal(t, webrtc_internal.WebRTCOperationRestartICE.String(), event.Attributes[webrtc_internal.DataOperation])
	assert.Equal(t, "true", event.Attributes[webrtc_internal.DataICERestart])

	select {
	case msg := <-s.OutputCh:
		t.Fatalf("deferred ICE restart should not emit offer immediately: %T", msg)
	default:
	}
}

func TestWebRTCOperation_RunsDeferredICERestartAfterGatheringComplete(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })
	_, _ = s.sessionState.BeginNegotiation(false)
	s.sessionState.SetICEGatheringActive(true)
	s.sessionState.DeferICERestart(webrtc_internal.WebRTCDeferredICERestart{
		MediaSessionID: mediaSessionID,
		Reason:         webrtc_internal.ReasonICEFailed,
		RequestedAt:    time.Now(),
	})

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationICEGatheringComplete,
		MediaSessionID: mediaSessionID,
	})

	assert.False(t, s.sessionState.DeferredICERestartPending(mediaSessionID))
	assert.Equal(t, webrtc_internal.NegotiationStateRetryPending, s.sessionState.NegotiationState())
	assert.True(t, s.sessionState.NegotiationRetryICE())
	event := requireObservabilityEvent(t, collector, webrtc_internal.EventNegotiationRetryQueued)
	assert.Equal(t, webrtc_internal.WebRTCOperationRestartICE.String(), event.Attributes[webrtc_internal.DataOperation])
	assert.Equal(t, "true", event.Attributes[webrtc_internal.DataICERestart])
}

func TestWebRTCOperation_ICEGatheringCompleteWithoutDeferredRestartClearsGatheringState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetICEGatheringActive(true)

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationICEGatheringComplete,
		MediaSessionID: mediaSessionID,
	})

	assert.False(t, s.sessionState.ICEGatheringActive())
	assert.False(t, s.sessionState.DeferredICERestartPending(mediaSessionID))
	select {
	case msg := <-s.OutputCh:
		t.Fatalf("ICE gathering completion without deferred restart should not signal: %T", msg)
	default:
	}
}

func TestWebRTCOperationLoop_CachesRemoteICEBeforeAnswer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, webrtc_internal.WebRTCOperationChannelSize)
	go s.runWebRTCOperationLoop()
	t.Cleanup(s.Cancel)

	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	sdpMid := "audio"
	sdpMLineIndex := uint16(0)
	usernameFragment := "remote"
	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationAddRemoteICECandidate,
		MediaSessionID: mediaSessionID,
		RemoteICECandidate: pionwebrtc.ICECandidateInit{
			Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
			SDPMid:           &sdpMid,
			SDPMLineIndex:    &sdpMLineIndex,
			UsernameFragment: &usernameFragment,
		},
	})

	require.Eventually(t, func() bool {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		return len(s.signalPendingRemoteICECandidates) == 1
	}, time.Second, 10*time.Millisecond)
}

func TestEnqueueWebRTCOperation_WaitsForCriticalOperationCapacity(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 1)
	s.webrtcOperationCh <- webrtc_internal.WebRTCOperation{
		Kind: webrtc_internal.WebRTCOperationAddRemoteICECandidate,
	}

	started := make(chan struct{})
	operationQueued := make(chan struct{})
	go func() {
		close(started)
		s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
			Kind: webrtc_internal.WebRTCOperationApplyRemoteAnswer,
		})
		close(operationQueued)
	}()

	<-started
	select {
	case <-operationQueued:
		t.Fatal("critical WebRTC operation should wait when operation queue is full")
	case <-time.After(20 * time.Millisecond):
	}

	<-s.webrtcOperationCh
	select {
	case <-operationQueued:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for critical WebRTC operation to queue")
	}
}

func TestWebRTCOperation_AppliesAnswerThenDrainsRemoteICE(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    mediaSessionID,
		SignalMediaConfig: true,
	})

	<-s.OutputCh
	offerMessage, ok := (<-s.OutputCh).(*protos.ServerSignaling)
	require.True(t, ok)
	require.NotNil(t, offerMessage.GetSdp())

	remoteMediaEngine := &pionwebrtc.MediaEngine{}
	require.NoError(t, remoteMediaEngine.RegisterDefaultCodecs())
	remoteAPI := pionwebrtc.NewAPI(pionwebrtc.WithMediaEngine(remoteMediaEngine))
	remotePeerConnection, err := remoteAPI.NewPeerConnection(pionwebrtc.Configuration{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, remotePeerConnection.Close()) })

	require.NoError(t, remotePeerConnection.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  offerMessage.GetSdp().GetSdp(),
	}))
	answer, err := remotePeerConnection.CreateAnswer(nil)
	require.NoError(t, err)
	require.NoError(t, remotePeerConnection.SetLocalDescription(answer))

	sdpMid := "audio"
	sdpMLineIndex := uint16(0)
	usernameFragment := "remote"
	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationAddRemoteICECandidate,
		MediaSessionID: mediaSessionID,
		RemoteICECandidate: pionwebrtc.ICECandidateInit{
			Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
			SDPMid:           &sdpMid,
			SDPMLineIndex:    &sdpMLineIndex,
			UsernameFragment: &usernameFragment,
		},
	})

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:            webrtc_internal.WebRTCOperationApplyRemoteAnswer,
		MediaSessionID:  mediaSessionID,
		RemoteAnswerSDP: answer.SDP,
	})

	s.Mu.Lock()
	assert.Empty(t, s.signalPendingRemoteICECandidates)
	assert.False(t, s.mediaHealthState.RemoteDescriptionSetAt.IsZero())
	s.Mu.Unlock()
}

func TestWebRTCOperation_EmitsNegotiationLifecycleEvents(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    mediaSessionID,
		SignalMediaConfig: true,
	})

	<-s.OutputCh
	offerMessage, ok := (<-s.OutputCh).(*protos.ServerSignaling)
	require.True(t, ok)
	require.NotNil(t, offerMessage.GetSdp())
	offerSentEvent := requireObservabilityEvent(t, collector, webrtc_internal.EventNegotiationOfferSent)
	assert.Equal(t, webrtc_internal.WebRTCOperationSendOffer.String(), offerSentEvent.Attributes[webrtc_internal.DataOperation])
	assert.Equal(t, "false", offerSentEvent.Attributes[webrtc_internal.DataICERestart])

	remoteMediaEngine := &pionwebrtc.MediaEngine{}
	require.NoError(t, remoteMediaEngine.RegisterDefaultCodecs())
	remoteAPI := pionwebrtc.NewAPI(pionwebrtc.WithMediaEngine(remoteMediaEngine))
	remotePeerConnection, err := remoteAPI.NewPeerConnection(pionwebrtc.Configuration{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, remotePeerConnection.Close()) })

	require.NoError(t, remotePeerConnection.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  offerMessage.GetSdp().GetSdp(),
	}))
	answer, err := remotePeerConnection.CreateAnswer(nil)
	require.NoError(t, err)
	require.NoError(t, remotePeerConnection.SetLocalDescription(answer))

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:            webrtc_internal.WebRTCOperationApplyRemoteAnswer,
		MediaSessionID:  mediaSessionID,
		RemoteAnswerSDP: answer.SDP,
	})

	answerEvent := requireObservabilityEvent(t, collector, webrtc_internal.EventNegotiationAnswerReceived)
	assert.Equal(t, webrtc_internal.WebRTCOperationApplyRemoteAnswer.String(), answerEvent.Attributes[webrtc_internal.DataOperation])
	assert.Equal(t, "false", answerEvent.Attributes[webrtc_internal.DataRetryPending])
}

func TestWebRTCOperation_IgnoresStaleMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	activeMediaSessionID := s.sessionState.StartMediaSession()

	s.handleWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    activeMediaSessionID + 1,
		SignalMediaConfig: true,
	})

	select {
	case msg := <-s.OutputCh:
		t.Fatalf("stale WebRTC operation should not emit output: %T", msg)
	default:
	}
}

func TestWebRTCOperation_ICEFailedDuringPendingOfferQueuesRetry(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })
	_, _ = s.sessionState.BeginNegotiation(false)

	s.restartICEOrMediaSessionFallback(mediaSessionID, webrtc_internal.ReasonICEFailed, time.Now())

	assert.Equal(t, webrtc_internal.NegotiationStateRetryPending, s.sessionState.NegotiationState())
	assert.True(t, s.sessionState.NegotiationRetryICE())
	event := requireObservabilityEvent(t, collector, webrtc_internal.EventNegotiationRetryQueued)
	assert.Equal(t, webrtc_internal.WebRTCOperationRestartICE.String(), event.Attributes[webrtc_internal.DataOperation])
	assert.Equal(t, "true", event.Attributes[webrtc_internal.DataICERestart])
}

func TestWebRTCOperation_QueuesICERestartRetryWhenOfferPending(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.peerConnection = &pionwebrtc.PeerConnection{}
	_, _ = s.sessionState.BeginNegotiation(false)

	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationRestartICE,
		MediaSessionID: mediaSessionID,
		OfferOptions:   &pionwebrtc.OfferOptions{ICERestart: true},
	})

	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.NegotiationStateRetryPending, s.sessionState.NegotiationState())
	assert.True(t, s.sessionState.NegotiationRetryICE())
	s.Mu.Unlock()
	select {
	case msg := <-s.OutputCh:
		t.Fatalf("queued ICE restart should not emit an offer immediately: %T", msg)
	default:
	}
}

func TestWebRTCRepeatedAudioModeToggles_NoDuplicateReadersNoDeadlock(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.BaseStreamer = channel_base.NewBaseStreamerWithChannelCapacity(s.Logger, 128, 128)
	t.Cleanup(s.Cancel)

	for i := 0; i < 5; i++ {
		s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)
		s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)
	}

	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	assert.False(t, s.sessionState.PeerConnected())
	assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
}

func TestWebRTCSequentialTextAudioModeSwitchSoak_LeavesTextStable(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.BaseStreamer = channel_base.NewBaseStreamerWithChannelCapacity(s.Logger, 512, 512)
	t.Cleanup(s.Cancel)

	const switchCount = 25
	for i := 0; i < switchCount; i++ {
		s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)
		assert.Equal(t, webrtc_internal.MediaStateAudioNegotiating, s.sessionState.MediaState())
		assert.False(t, s.sessionState.PeerConnected())
		s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)
		assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
		assert.False(t, s.sessionState.PeerConnected())
		assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
	}

	s.Mu.Lock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Empty(t, s.signalingSessionID)
	assert.Nil(t, s.peerConnection)
	assert.Nil(t, s.assistantAudioTrack)
	assert.Nil(t, s.assistantRTPSender)
	assert.Nil(t, s.mediaCtx)
	assert.Nil(t, s.cancelMedia)
	s.Mu.Unlock()
}

func TestWebRTCConcurrentTextAudioModeSwitchRace_LeavesValidState(t *testing.T) {
	s := newTestStreamer(t)
	s.BaseStreamer = channel_base.NewBaseStreamerWithChannelCapacity(s.Logger, 1024, 1024)
	t.Cleanup(s.Cancel)

	const workerCount = 2
	const switchesPerWorker = 4
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		workerIndex := workerIndex
		go func() {
			defer workers.Done()
			for switchIndex := 0; switchIndex < switchesPerWorker; switchIndex++ {
				if (workerIndex+switchIndex)%2 == 0 {
					s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)
					continue
				}
				s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)
			}
		}()
	}
	workers.Wait()

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)

	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	assert.False(t, s.sessionState.PeerConnected())
	assert.Zero(t, s.sessionState.RemoteAudioReaderMediaSessionID())
	s.Mu.Lock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Empty(t, s.signalingSessionID)
	assert.Nil(t, s.peerConnection)
	assert.Nil(t, s.assistantAudioTrack)
	assert.Nil(t, s.assistantRTPSender)
	s.Mu.Unlock()
}

func TestWebRTCTrackBeforeRTP_StartsRemoteAudioReaderOnce(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	peerConnection := &pionwebrtc.PeerConnection{}
	s.peerConnection = peerConnection

	assert.True(t, s.tryStartRemoteAudioReader(peerConnection, mediaSessionID))
	assert.False(t, s.tryStartRemoteAudioReader(peerConnection, mediaSessionID))
	assert.Equal(t, mediaSessionID, s.sessionState.RemoteAudioReaderMediaSessionID())
	s.mediaWorkers.Done()
}

func TestHandlePeerState_DisconnectedQueuesRecovery(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	s.mediaLifecycleCh = make(chan webrtc_internal.MediaLifecycleEvent, 1)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioConnected)
	disconnectedAt := time.Now()

	s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateDisconnected, disconnectedAt)

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.MediaStateAudioNegotiating, s.sessionState.MediaState())
	s.Mu.Unlock()
	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, webrtc_internal.MediaLifecycleEventRecover, event.Kind)
		assert.Equal(t, mediaSessionID, event.MediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonPeerDisconnected, event.Reason)
		assert.Equal(t, disconnectedAt, event.RequestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for recovery event")
	}
	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCDisconnected)
	payload, ok := webhook.Payload.(observability.WebRTCDisconnectedWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, "peer_disconnected", payload.Type)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, webrtc_internal.ReasonPeerDisconnected, payload.Reason)
}

func TestHandlePeerState_FailedRecordsWebhook(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	s.mediaLifecycleCh = make(chan webrtc_internal.MediaLifecycleEvent, 1)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	failedAt := time.Now()

	s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateFailed, failedAt)

	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, webrtc_internal.MediaLifecycleEventRecover, event.Kind)
		assert.Equal(t, mediaSessionID, event.MediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonPeerFailed, event.Reason)
		assert.Equal(t, failedAt, event.RequestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for recovery event")
	}
	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCFailed)
	payload, ok := webhook.Payload.(observability.WebRTCFailedWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, "peer_failed", payload.Type)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, webrtc_internal.ReasonPeerFailed, payload.Reason)
}

func TestRestartICERecordsReconnectWebhook(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 1)
	mediaSessionID := s.sessionState.StartMediaSession()

	s.restartICEOrMediaSessionFallback(mediaSessionID, webrtc_internal.ReasonICEFailed, time.Now())

	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCReconnecting)
	payload, ok := webhook.Payload.(observability.WebRTCReconnectingWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, webrtc_internal.EventICERestarting, payload.Type)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, webrtc_internal.ReasonICEFailed, payload.Reason)
	assert.Equal(t, uint64(1), payload.RestartAttempt)
	assert.Equal(t, uint64(webrtc_internal.ICERestartAttemptLimit), payload.RestartLimit)
}

func TestHandlePeerICEConnectionState_RecordsSeparateICEState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	mediaSessionID := s.sessionState.StartMediaSession()
	changedAt := time.Now()

	s.handlePeerICEConnectionState(mediaSessionID, pionwebrtc.ICEConnectionStateChecking, changedAt)

	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.ICEStateChecking, s.mediaHealthState.ICEConnectionState)
	assert.Equal(t, changedAt, s.mediaHealthState.ICEConnectionStateChangedAt)
	assert.Equal(t, changedAt, s.mediaHealthState.ICECheckingStartedAt)
	s.Mu.Unlock()

	event := requireObservabilityEvent(t, collector, webrtc_internal.EventICEConnectionState)
	assert.Equal(t, webrtc_internal.ICEStateChecking, event.Attributes[webrtc_internal.DataICEConnectionState])
}

func TestHandlePeerICEConnectionState_FailedQueuesRecovery(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	s.mediaLifecycleCh = make(chan webrtc_internal.MediaLifecycleEvent, 1)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	failedAt := time.Now()

	s.handlePeerICEConnectionState(mediaSessionID, pionwebrtc.ICEConnectionStateFailed, failedAt)

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.ICEStateFailed, s.mediaHealthState.ICEConnectionState)
	assert.Equal(t, failedAt, s.mediaHealthState.ICEFailedAt)
	s.Mu.Unlock()

	event := requireObservabilityEvent(t, collector, "ice_failed")
	assert.Equal(t, webrtc_internal.ICEStateFailed, event.Attributes[webrtc_internal.DataICEConnectionState])

	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, webrtc_internal.MediaLifecycleEventRecover, event.Kind)
		assert.Equal(t, mediaSessionID, event.MediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonICEFailed, event.Reason)
		assert.Equal(t, failedAt, event.RequestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ICE recovery event")
	}
}

func TestSessionState_TryBeginICERestartHonorsLimit(t *testing.T) {
	t.Parallel()
	var state webrtc_internal.SessionState

	attempt, ok := state.TryBeginICERestart(webrtc_internal.ICERestartAttemptLimit)
	require.True(t, ok)
	assert.Equal(t, uint64(1), attempt)

	attempt, ok = state.TryBeginICERestart(webrtc_internal.ICERestartAttemptLimit)
	assert.False(t, ok)
	assert.Equal(t, uint64(1), attempt)

	state.ResetICERestartAttempts()
	attempt, ok = state.TryBeginICERestart(webrtc_internal.ICERestartAttemptLimit)
	require.True(t, ok)
	assert.Equal(t, uint64(1), attempt)
}

func TestRestartMediaSession_LimitFallsBackToText(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { require.NoError(t, s.observer.Close(context.Background())) })
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioConnected)
	_, ok := s.sessionState.TryBeginMediaRestart(webrtc_internal.MediaRestartAttemptLimit)
	require.True(t, ok)

	s.restartMediaSessionOrFallbackToText(mediaSessionID, webrtc_internal.ReasonPeerFailed, time.Now())

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Equal(t, webrtc_internal.MediaStateText, s.sessionState.MediaState())
	s.Mu.Unlock()
	webhook := requireObservabilityWebhook(t, collector, observability.WebRTCFailed)
	payload, ok := webhook.Payload.(observability.WebRTCFailedWebhookPayload)
	require.True(t, ok)
	assert.Equal(t, "media_restart_limit_reached", payload.Type)
	assert.Equal(t, s.sessionID, payload.SessionID)
	assert.Equal(t, mediaSessionID, payload.MediaSessionID)
	assert.Equal(t, webrtc_internal.ReasonPeerFailed, payload.Reason)
	assert.Equal(t, "text", payload.Fallback)
}

func TestMediaHealthState_RecordsInputMediaHealth(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.StartICE(now)
	state.RecordPeerConnected(now.Add(time.Millisecond))
	state.RecordUserAudioReadError(1)
	state.RecordUserAudioReadError(2)
	state.RecordUserAudioRTPUnmarshalFailure()
	state.RecordUserAudioEmptyRTPPayload()
	state.RecordUserAudioOpusDecodeFailure()
	state.RecordUserAudioResampleFailure()
	state.RecordUserAudioReceived(now.Add(2 * time.Millisecond))

	assert.Equal(t, now, state.ICEStartedAt)
	assert.Equal(t, now.Add(time.Millisecond), state.PeerConnectedAt)
	assert.Equal(t, uint64(2), state.UserAudioReadErrors)
	assert.Equal(t, 0, state.UserAudioConsecutiveReadErrors)
	assert.Equal(t, uint64(1), state.UserAudioRTPUnmarshalFailures)
	assert.Equal(t, uint64(1), state.UserAudioEmptyRTPPayloads)
	assert.Equal(t, uint64(1), state.UserAudioOpusDecodeFailures)
	assert.Equal(t, uint64(1), state.UserAudioResampleFailures)
	assert.Equal(t, now.Add(2*time.Millisecond), state.FirstUserAudioReceivedAt)
	assert.Equal(t, now.Add(2*time.Millisecond), state.LastUserAudioReceivedAt)
}

func TestMediaHealthState_HandshakeDeadlineExceeded(t *testing.T) {
	t.Parallel()
	now := time.Now()

	tests := []struct {
		name  string
		state webrtc_internal.MediaHealthState
		at    time.Time
		want  string
		ok    bool
	}{
		{
			name: "remote_answer_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt: now,
			},
			at:   now.Add(webrtc_internal.SignalingAnswerDeadline),
			want: webrtc_internal.ReasonRemoteAnswerDeadline,
			ok:   true,
		},
		{
			name: "ice_connected_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
			},
			at:   now.Add(webrtc_internal.ICEConnectedDeadline),
			want: webrtc_internal.ReasonICEConnectedDeadline,
			ok:   true,
		},
		{
			name: "peer_connected_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
				ICEConnectedAt:         now.Add(2 * time.Millisecond),
			},
			at:   now.Add(webrtc_internal.PeerConnectedDeadline),
			want: webrtc_internal.ReasonPeerConnectedDeadline,
			ok:   true,
		},
		{
			name: "connected",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
				ICEConnectedAt:         now.Add(2 * time.Millisecond),
				PeerConnectedAt:        now.Add(3 * time.Millisecond),
			},
			at: now.Add(webrtc_internal.PeerConnectedDeadline),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reason, _, _, ok := tt.state.HandshakeDeadlineExceeded(tt.at)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, reason)
		})
	}
}

func TestMediaHealthState_RecordsReceiverReportQuality(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.RecordReceiverReport(
		now,
		64,
		3,
		webrtc_internal.OpusFrameSamples,
		0,
		0,
	)

	assert.Equal(t, uint64(1), state.ReceiverReports)
	assert.Equal(t, now, state.LastReceiverReportAt)
	assert.Equal(t, uint8(64), state.LastReceiverReportFractionLost)
	assert.Equal(t, 25.0, state.LastReceiverReportPacketLossPercent)
	assert.Equal(t, uint32(3), state.LastReceiverReportTotalLost)
	assert.Equal(t, 20.0, state.LastReceiverReportJitterMs)
	assert.False(t, state.LastReceiverReportRoundTripTimeUsable)
	assert.Zero(t, state.LastReceiverReportRoundTripTimeMs)
}

func TestMediaHealthState_RecordsReceiverReportRoundTripTime(t *testing.T) {
	t.Parallel()
	now := time.Now()
	delayMs := int64(20)
	roundTripMs := int64(80)
	state := webrtc_internal.MediaHealthState{}
	lastSenderReport := webrtc_internal.CompactNTP(now.Add(-time.Duration(delayMs+roundTripMs) * time.Millisecond))
	delayUnits := uint32(delayMs * webrtc_internal.RTCPCompactNTPUnitsPerSec / webrtc_internal.MillisecondsPerSecond)

	state.RecordReceiverReport(
		now,
		0,
		0,
		0,
		lastSenderReport,
		delayUnits,
	)

	assert.True(t, state.LastReceiverReportRoundTripTimeUsable)
	assert.InDelta(t, roundTripMs, state.LastReceiverReportRoundTripTimeMs, 1)
}

func TestMediaHealthState_QualityState(t *testing.T) {
	t.Parallel()
	now := time.Now()

	tests := []struct {
		name  string
		state webrtc_internal.MediaHealthState
		want  string
	}{
		{
			name: "excellent",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportPacketLossPercent: 1.0,
				LastReceiverReportJitterMs:          10.0,
			},
			want: webrtc_internal.QualityStateExcellent,
		},
		{
			name: "good",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportPacketLossPercent: webrtc_internal.QualityGoodPacketLossPercent,
			},
			want: webrtc_internal.QualityStateGood,
		},
		{
			name: "poor",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportJitterMs: webrtc_internal.QualityPoorJitterMs,
			},
			want: webrtc_internal.QualityStatePoor,
		},
		{
			name: "lost_on_write_failures",
			state: webrtc_internal.MediaHealthState{
				ConsecutiveAssistantFrameWriteFailures: webrtc_internal.RepeatedWriteFailuresThreshold,
			},
			want: webrtc_internal.QualityStateLost,
		},
		{
			name: "lost_on_missing_feedback",
			state: webrtc_internal.MediaHealthState{
				LastAssistantFrameSentAt: now.Add(-webrtc_internal.RTCPFeedbackMissingThreshold),
			},
			want: webrtc_internal.QualityStateLost,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.QualityState(now))
		})
	}
}

func TestMediaHealthState_RecordsAssistantWriteFailures(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.RecordAssistantFrameWriteFailure(now)
	state.RecordAssistantFrameWriteFailure(now.Add(time.Millisecond))

	assert.Equal(t, uint64(2), state.AssistantFrameWriteFailures)
	assert.Equal(t, uint64(2), state.ConsecutiveAssistantFrameWriteFailures)
	assert.Equal(t, now.Add(time.Millisecond), state.LastAssistantFrameWriteFailureAt)

	state.RecordAssistantFrameSent(now.Add(2 * time.Millisecond))

	assert.Equal(t, uint64(2), state.AssistantFrameWriteFailures)
	assert.Zero(t, state.ConsecutiveAssistantFrameWriteFailures)
	assert.Equal(t, now.Add(2*time.Millisecond), state.LastAssistantFrameSentAt)
}

func TestMediaHealthState_RecordsSelectedICECandidatePair(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}
	pair := webrtc_internal.SelectedICECandidatePair{
		ID:                          "pair-1",
		LocalCandidateType:          "host",
		LocalProtocol:               "udp",
		RemoteCandidateType:         "srflx",
		RemoteProtocol:              "udp",
		CurrentRoundTripTimeMs:      42,
		AvailableOutgoingBitrateBps: 64000,
	}

	assert.True(t, state.RecordSelectedICECandidatePair(pair, now))
	assert.Equal(t, "pair-1", state.SelectedICECandidatePairID)
	assert.Equal(t, "host", state.SelectedICELocalCandidateType)
	assert.Equal(t, "udp", state.SelectedICELocalProtocol)
	assert.Equal(t, "srflx", state.SelectedICERemoteCandidateType)
	assert.Equal(t, int64(42), state.SelectedICECandidatePairRTTMs)
	assert.Equal(t, now, state.SelectedICECandidatePairChangedAt)

	assert.False(t, state.RecordSelectedICECandidatePair(pair, now.Add(time.Second)))
	assert.Equal(t, now, state.SelectedICECandidatePairChangedAt)
}

func TestSelectedICECandidatePairFromStats(t *testing.T) {
	t.Parallel()
	report := pionwebrtc.StatsReport{
		"local": pionwebrtc.ICECandidateStats{
			ID:            "local",
			Protocol:      "udp",
			CandidateType: pionwebrtc.ICECandidateTypeHost,
		},
		"remote": pionwebrtc.ICECandidateStats{
			ID:            "remote",
			Protocol:      "tcp",
			CandidateType: pionwebrtc.ICECandidateTypeRelay,
		},
		"pair": pionwebrtc.ICECandidatePairStats{
			ID:                       "pair",
			LocalCandidateID:         "local",
			RemoteCandidateID:        "remote",
			State:                    pionwebrtc.StatsICECandidatePairStateSucceeded,
			Nominated:                true,
			CurrentRoundTripTime:     0.125,
			AvailableOutgoingBitrate: 48000,
		},
	}

	pair, ok := selectedICECandidatePairFromStats(report)

	require.True(t, ok)
	assert.Equal(t, "pair", pair.ID)
	assert.Equal(t, "host", pair.LocalCandidateType)
	assert.Equal(t, "udp", pair.LocalProtocol)
	assert.Equal(t, "relay", pair.RemoteCandidateType)
	assert.Equal(t, "tcp", pair.RemoteProtocol)
	assert.Equal(t, int64(125), pair.CurrentRoundTripTimeMs)
	assert.Equal(t, int64(48000), pair.AvailableOutgoingBitrateBps)
}

func TestResetAudioSession_FlushesPendingOutput(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.withOutputAudioBuffer(func(buf *bytes.Buffer) {
		buf.Write([]byte{0x01, 0x02, 0x03, 0x04})
	})
	s.Output(&protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0xAA, 0xBB}},
	})

	s.stopMediaSessionAndFallbackToText()

	s.withOutputAudioBuffer(func(buf *bytes.Buffer) {
		assert.Equal(t, 0, buf.Len(), "outputAudioBuffer accumulation buffer should be cleared")
	})

	select {
	case <-s.OutputCh:
		t.Fatal("outputAudioBuffer channel should be drained after reset")
	default:
	}
}

func TestAudioBuffer_InputEmitsBridgeAudioAndFramedUserAudio(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	inputAudioReceivedAt := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	audio := bytes.Repeat([]byte{0x11}, webrtc_internal.InputBufferThreshold)

	s.bufferAndSendInput(audio, inputAudioReceivedAt)

	bridgeAudio, ok := (<-s.InputCh).(*protos.ConversationBridgeUserAudio)
	require.True(t, ok)
	assert.Equal(t, audio, bridgeAudio.GetAudio())
	assert.Equal(t, inputAudioReceivedAt, bridgeAudio.GetTime().AsTime())

	userAudio, ok := (<-s.InputCh).(*protos.ConversationUserMessage)
	require.True(t, ok)
	assert.Equal(t, audio, userAudio.GetAudio())
	assert.Equal(t, inputAudioReceivedAt, userAudio.GetTime().AsTime())
}

func TestAudioBuffer_OutputFramesAssistantAudio(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	audio := bytes.Repeat([]byte{0x22}, webrtc_internal.WebRTCOutputPCM16kFrameBytes*2+1)

	s.bufferAndSendOutput(audio)

	for i := 0; i < 2; i++ {
		msg, ok := (<-s.OutputCh).(*protos.ConversationAssistantMessage)
		require.True(t, ok)
		assert.Len(t, msg.GetAudio(), webrtc_internal.WebRTCOutputPCM16kFrameBytes)
		assert.NotNil(t, msg.GetTime())
	}
	s.withOutputAudioBuffer(func(buf *bytes.Buffer) {
		assert.Equal(t, 1, buf.Len())
	})
}

func TestSend_TextMessage(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello"},
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_AudioBuffersWebRTCOutputPCM16kFrame(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	audio := bytes.Repeat([]byte{0x22}, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{Audio: audio},
	}

	err := s.Send(msg)
	require.NoError(t, err)

	select {
	case out := <-s.OutputCh:
		assistant, ok := out.(*protos.ConversationAssistantMessage)
		require.True(t, ok, "expected ConversationAssistantMessage, got %T", out)
		got := assistant.GetAudio()
		assert.Len(t, got, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
		assert.Equal(t, audio, got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for assistant audio")
	}
}

func TestSend_Interruption(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	s.enqueueOutputAudio([]byte{0x10})
	s.enqueueOutputAudio([]byte{0x20})

	msg := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
	}
	err := s.Send(msg)
	assert.NoError(t, err)

	s.outputAudioQueueMu.Lock()
	assert.Empty(t, s.outputAudioQueue)
	s.outputAudioQueueMu.Unlock()

	log := requireObservabilityLog(t, collector, webrtc_internal.EventOutputQueueCleared)
	require.NoError(t, s.observer.Close(context.Background()))
	assert.Equal(t, observability.LevelInfo, log.Level)
	assert.Contains(t, log.Message, "user interruption")
	assert.Equal(t, webrtc_internal.EventOutputQueueCleared, log.Attributes[webrtc_internal.DataType])
	assert.Equal(t, webrtc_internal.OutputQueueClearReasonInterruption, log.Attributes[webrtc_internal.DataReason])
	assert.Equal(t, "2", log.Attributes[webrtc_internal.DataClearedFrames])
	assert.Equal(t, fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueEmptySize), log.Attributes[webrtc_internal.DataRemainingQueueFrames])
	assert.Empty(t, collector.metrics)
}

func TestSend_EndConversation(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_TransferConversation_PushesFailedResult(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{
		Id:     "tc-transfer",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err := s.Send(msg)
	require.NoError(t, err)

	select {
	case incoming := <-s.CriticalCh:
		result, ok := incoming.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", incoming)
		assert.Equal(t, "tc-transfer", result.GetId())
		assert.Equal(t, "tool-transfer", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported for WebRTC")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}

func TestHandleClientSignal_IgnoresStaleSignalingSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 2)
	s.signalingSessionID = "active-signaling-session"
	s.sessionState.StartMediaSession()

	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "stale-signaling-session",
		Message: &protos.ClientSignaling_Sdp{
			Sdp: &protos.WebRTCSDP{
				Type: protos.WebRTCSDP_ANSWER,
				Sdp:  "v=0\r\n",
			},
		},
	})
	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "stale-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{
				Candidate: "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
				SdpMid:    "audio",
			},
		},
	})

	select {
	case operation := <-s.webrtcOperationCh:
		t.Fatalf("stale client signaling should not enqueue WebRTC operation: %+v", operation)
	default:
	}
}

func TestHandleClientSignal_EnqueuesActiveAnswerAndICECandidate(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.webrtcOperationCh = make(chan webrtc_internal.WebRTCOperation, 2)
	s.signalingSessionID = "active-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()

	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "active-signaling-session",
		Message: &protos.ClientSignaling_Sdp{
			Sdp: &protos.WebRTCSDP{
				Type: protos.WebRTCSDP_ANSWER,
				Sdp:  "v=0\r\n",
			},
		},
	})
	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "active-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{
				Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
				SdpMid:           "audio",
				SdpMLineIndex:    0,
				UsernameFragment: "remote-ufrag",
			},
		},
	})

	var answerOperation webrtc_internal.WebRTCOperation
	select {
	case answerOperation = <-s.webrtcOperationCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for remote answer operation")
	}
	assert.Equal(t, webrtc_internal.WebRTCOperationApplyRemoteAnswer, answerOperation.Kind)
	assert.Equal(t, mediaSessionID, answerOperation.MediaSessionID)
	assert.Equal(t, "v=0\r\n", answerOperation.RemoteAnswerSDP)

	var iceOperation webrtc_internal.WebRTCOperation
	select {
	case iceOperation = <-s.webrtcOperationCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for remote ICE operation")
	}
	assert.Equal(t, webrtc_internal.WebRTCOperationAddRemoteICECandidate, iceOperation.Kind)
	assert.Equal(t, mediaSessionID, iceOperation.MediaSessionID)
	assert.Equal(t, "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host", iceOperation.RemoteICECandidate.Candidate)
	require.NotNil(t, iceOperation.RemoteICECandidate.SDPMid)
	require.NotNil(t, iceOperation.RemoteICECandidate.SDPMLineIndex)
	require.NotNil(t, iceOperation.RemoteICECandidate.UsernameFragment)
	assert.Equal(t, "audio", *iceOperation.RemoteICECandidate.SDPMid)
	assert.Equal(t, uint16(0), *iceOperation.RemoteICECandidate.SDPMLineIndex)
	assert.Equal(t, "remote-ufrag", *iceOperation.RemoteICECandidate.UsernameFragment)
}

func TestQueueClientSignal_QueuesPeerEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.peerEventCh = make(chan webrtc_internal.PeerEvent, 1)
	signaling := &protos.ClientSignaling{
		SessionId: "signaling-session",
		Message: &protos.ClientSignaling_Disconnect{
			Disconnect: true,
		},
	}

	s.queueClientSignal(signaling)

	select {
	case event := <-s.peerEventCh:
		assert.Equal(t, webrtc_internal.SignalEventClientMessage, event.Kind)
		assert.Same(t, signaling, event.SignalClientMessage)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for peer event")
	}
}

func TestHandleClientSignal_DisconnectClosesStreamer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.handleClientSignal(&protos.ClientSignaling{
		Message: &protos.ClientSignaling_Disconnect{
			Disconnect: true,
		},
	})

	assert.True(t, s.sessionState.CloseStarted())
	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER, disc.GetType())
	default:
		t.Fatal("expected disconnection on client signaling disconnect")
	}
}

func TestEnqueuePeerEvent_PreservesPeerConnectionStateTransitions(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.peerEventCh = make(chan webrtc_internal.PeerEvent, 2)

	s.enqueuePeerEvent(webrtc_internal.PeerEvent{
		Kind:               webrtc_internal.PeerEventStateChanged,
		MediaSessionID:     10,
		PeerState:          pionwebrtc.PeerConnectionStateDisconnected,
		PeerStateChangedAt: time.Now(),
	})
	s.enqueuePeerEvent(webrtc_internal.PeerEvent{
		Kind:               webrtc_internal.PeerEventStateChanged,
		MediaSessionID:     10,
		PeerState:          pionwebrtc.PeerConnectionStateConnected,
		PeerStateChangedAt: time.Now().Add(time.Millisecond),
	})

	first := <-s.peerEventCh
	second := <-s.peerEventCh
	assert.Equal(t, pionwebrtc.PeerConnectionStateDisconnected, first.PeerState)
	assert.Equal(t, pionwebrtc.PeerConnectionStateConnected, second.PeerState)
}

func TestApplyAmbientConfig_ReadsTypedConfig(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{}
	s.ambientMixer = fake

	s.applyAmbientConfig(internal_ambient.NewConfig("cafe", 37), "test")

	assert.Equal(t, "cafe", fake.cfg.Profile)
	assert.Equal(t, 37, fake.cfg.Volume)
	assert.True(t, fake.cfg.Enabled)
}

func TestApplyAmbientConfig_InvalidAmbientFallsBackToNone(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{}
	s.ambientMixer = fake

	s.applyAmbientConfig(internal_ambient.NewConfig("foobar", 24), "test")

	assert.Equal(t, "none", fake.cfg.Profile)
	assert.Equal(t, 24, fake.cfg.Volume)
	assert.False(t, fake.cfg.Enabled)
}

func TestApplyAmbientToFrame_AmbientOnlyOnSilenceTicks(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{
		ambientOut: make([]byte, webrtc_internal.WebRTCOutputPCM16kFrameBytes),
	}
	for i := range fake.ambientOut {
		fake.ambientOut[i] = 0x11
	}
	s.ambientMixer = fake

	out := s.applyAmbientToFrame(nil)
	require.NotNil(t, out)
	assert.Len(t, out, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
	assert.NotEqual(t, make([]byte, len(out)), out)
}

func TestApplyAmbientToFrame_NoneLeavesPrimaryUntouched(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.ambientMixer = nil

	pcm16k := make([]byte, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
	for i := range pcm16k {
		pcm16k[i] = byte(i % 251)
	}
	out := s.applyAmbientToFrame(pcm16k)
	assert.Equal(t, pcm16k, out)
}

func TestEnqueueOutputAudio_BoundedDropOldest_EmitsOverflowEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	collector := &testObservabilityCollector{}
	s.observer = observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)

	limit := webrtc_internal.OutputAudioQueueMaxFrames
	for i := 0; i < limit+1; i++ {
		s.enqueueOutputAudio([]byte{byte(i % 251)})
	}

	s.outputAudioQueueMu.Lock()
	require.Len(t, s.outputAudioQueue, limit)
	// Oldest frame should be dropped on overflow; new head is former index 1.
	require.Len(t, s.outputAudioQueue[0].Audio, 1)
	assert.Equal(t, byte(1), s.outputAudioQueue[0].Audio[0])
	assert.False(t, s.outputAudioQueue[0].QueuedAt.IsZero())
	s.outputAudioQueueMu.Unlock()

	log := requireObservabilityLog(t, collector, webrtc_internal.EventOutputQueueOverflow)
	require.NoError(t, s.observer.Close(context.Background()))
	assert.Equal(t, observability.LevelInfo, log.Level)
	assert.Contains(t, log.Message, "queue overflow")
	assert.Equal(t, webrtc_internal.EventOutputQueueOverflow, log.Attributes[webrtc_internal.DataType])
	assert.Equal(t, webrtc_internal.OutputQueuePolicyDropOldest, log.Attributes[webrtc_internal.DataPolicy])
	assert.Equal(t, "1", log.Attributes[webrtc_internal.DataDroppedFrames])
	assert.Equal(t, fmt.Sprintf("%d", limit), log.Attributes[webrtc_internal.DataLimitFrames])
	assert.Equal(t, fmt.Sprintf("%d", limit), log.Attributes[webrtc_internal.DataQueueDepthFrames])
	metric := requireObservabilityMetric(t, collector, observability.MetricWebRTCOutputQueueDrops)
	assert.Equal(t, "1", metric.Value)
}

func TestClearOutputAudio_ReturnsClearedFrameCount(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.enqueueOutputAudio([]byte{0x01})
	s.enqueueOutputAudio([]byte{0x02})

	clearedFrames := s.clearOutputAudio()

	assert.Equal(t, 2, clearedFrames)
	s.outputAudioQueueMu.Lock()
	assert.Empty(t, s.outputAudioQueue)
	s.outputAudioQueueMu.Unlock()
}

func TestEnqueueOutputAudio_StoresQueuedAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	audio := []byte{0x11, 0x22}

	queuedBefore := time.Now()
	s.enqueueOutputAudio(audio)
	queuedAfter := time.Now()

	s.outputAudioQueueMu.Lock()
	require.Len(t, s.outputAudioQueue, 1)
	outputFrame := s.outputAudioQueue[0]
	s.outputAudioQueueMu.Unlock()

	assert.Equal(t, audio, outputFrame.Audio)
	assert.False(t, outputFrame.QueuedAt.IsZero())
	assert.False(t, outputFrame.QueuedAt.Before(queuedBefore))
	assert.False(t, outputFrame.QueuedAt.After(queuedAfter))
}

func TestEnqueueOutputAudio_TracksFirstAssistantAudioQueuedAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.enqueueOutputAudio([]byte{0x01})

	s.Mu.Lock()
	firstQueuedAt := s.mediaHealthState.FirstAssistantAudioQueuedAt
	lastQueuedAt := s.mediaHealthState.LastAssistantAudioQueuedAt
	s.Mu.Unlock()

	require.False(t, firstQueuedAt.IsZero())
	assert.Equal(t, firstQueuedAt, lastQueuedAt)

	time.Sleep(time.Millisecond)
	s.enqueueOutputAudio([]byte{0x02})

	s.Mu.Lock()
	assert.Equal(t, firstQueuedAt, s.mediaHealthState.FirstAssistantAudioQueuedAt)
	assert.True(t, s.mediaHealthState.LastAssistantAudioQueuedAt.After(firstQueuedAt))
	s.Mu.Unlock()
}

func TestNextFrame_StampsActiveMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.enqueueOutputAudio([]byte{0x01, 0x02})

	frame := s.NextFrame()

	assert.Equal(t, []byte{0x01, 0x02}, frame)
	assert.Equal(t, mediaSessionID, s.sessionState.PacedAssistantFrameMediaSessionID())
}

func TestConsumeFrame_TracksWriteFailureWithoutRecordingAssistantAudio(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	assistantPCM16k := bytes.Repeat([]byte{0x33}, webrtc_internal.WebRTCOutputPCM16kFrameBytes)

	err := s.ConsumeFrame(assistantPCM16k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assistant audio track is not ready")
	assert.True(t, s.mediaHealthState.LastAssistantFrameSentAt.IsZero())
	assert.Equal(t, uint64(1), s.mediaHealthState.AssistantFrameWriteFailures)
	assert.Equal(t, uint64(1), s.mediaHealthState.ConsecutiveAssistantFrameWriteFailures)
	assert.False(t, s.mediaHealthState.LastAssistantFrameWriteFailureAt.IsZero())

	select {
	case msg := <-s.InputCh:
		t.Fatalf("failed assistant frame should not be recorded, got %T", msg)
	default:
	}
}

func TestConsumeFrame_DropsStalePacedMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	staleMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.StampPacedAssistantFrame(staleMediaSessionID)
	s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	assistantPCM16k := bytes.Repeat([]byte{0x55}, webrtc_internal.WebRTCOutputPCM16kFrameBytes)

	err := s.ConsumeFrame(assistantPCM16k)

	require.NoError(t, err)
	assert.True(t, s.mediaHealthState.LastAssistantFrameSentAt.IsZero())
	assert.Zero(t, s.mediaHealthState.AssistantFrameWriteFailures)

	select {
	case msg := <-s.InputCh:
		t.Fatalf("stale assistant frame should not be recorded, got %T", msg)
	default:
	}
}

func TestConsumeFrame_TracksLastAssistantFrameSentAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	assistantAudioTrack, err := pionwebrtc.NewTrackLocalStaticSample(
		pionwebrtc.RTPCodecCapability{
			MimeType:  pionwebrtc.MimeTypeOpus,
			ClockRate: webrtc_internal.OpusSampleRate,
			Channels:  webrtc_internal.OpusChannels,
		},
		"audio",
		"rapida-audio",
	)
	require.NoError(t, err)
	s.assistantAudioTrack = assistantAudioTrack

	assistantPCM16k := bytes.Repeat([]byte{0x44}, webrtc_internal.WebRTCOutputPCM16kFrameBytes)

	err = s.ConsumeFrame(assistantPCM16k)
	require.NoError(t, err)

	s.Mu.Lock()
	lastSentAt := s.mediaHealthState.LastAssistantFrameSentAt
	s.Mu.Unlock()

	assert.False(t, lastSentAt.IsZero())

	select {
	case msg := <-s.InputCh:
		bridge, ok := msg.(*protos.ConversationBridgeOperatorAudio)
		require.True(t, ok, "expected ConversationBridgeOperatorAudio, got %T", msg)
		assert.Equal(t, assistantPCM16k, bridge.GetAudio())
		assert.NotNil(t, bridge.GetTime())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for assistant bridge audio")
	}
}

func TestWriteAudioFrame_ReturnsErrorWhenAssistantTrackMissing(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	err := s.writeAudioFrame([]byte{0x01, 0x02})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "assistant audio track is not ready")
}
