// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio_telephony

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_twilio "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTwilioMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

type recordingObserver struct {
	records []observability.Record
}

func (r *recordingObserver) Record(_ context.Context, _ observability.Scope, records ...observability.Record) error {
	r.records = append(r.records, records...)
	return nil
}

func (r *recordingObserver) AddCollectors(...observability.Collector) error {
	return nil
}

func (r *recordingObserver) Close(context.Context) error {
	return nil
}

func (engine *fakeTwilioMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
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

func (engine *fakeTwilioMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error {
	return nil
}

func (engine *fakeTwilioMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeTwilioMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeTwilioMediaEngine) ClearOutputBuffer() {}

func (engine *fakeTwilioMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeTwilioMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func (engine *fakeTwilioMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakeTwilioMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

// testWSPair creates a connected WebSocket client/server pair for unit tests.
// The server side is discarded; only the client *websocket.Conn is returned.
// The caller should call cleanup() when done.
func testWSPair(t *testing.T) (clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Drain messages so write side does not block.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	return conn, func() {
		conn.Close()
		server.Close()
	}
}

// newTestTwilioStreamer creates a twilioWebsocketStreamer with a real WebSocket
// connection (required because Send returns early if connection is nil) but
// without starting the background reader goroutine.
//
// ChannelUUID is intentionally left empty so the Twilio API call block inside
// END_CONVERSATION is skipped, isolating the ToolCallResult + Cancel logic.
func newTestTwilioStreamer(t *testing.T) (*twilioWebsocketStreamer, func()) {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		ChannelUUID:    "", // empty so the Twilio API block is skipped
	}

	conn, cleanup := testWSPair(t)

	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger, cc, nil, nil,
		),
		streamID:   "test-stream",
		connection: conn,
	}
	// Do not start runWebSocketReader because these tests exercise Send only.
	return tws, cleanup
}

func TestNewTwilioWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "twilio",
	}
	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(callContext),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	tws, ok := streamer.(*twilioWebsocketStreamer)
	require.True(t, ok, "expected twilio websocket streamer")
	defer tws.Cancel()

	require.NotNil(t, tws.mediaSession)
}

func TestHandleMediaEvent_EmitsBridgeUserAudio(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "twilio",
	}
	mediaEngine := &fakeTwilioMediaEngine{}
	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	tws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     tws.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  tws.Input,
	})

	providerAudio := make([]byte, internal_twilio.OutputChunkSize)
	for i := range providerAudio {
		providerAudio[i] = internal_twilio.MulawSilence
	}
	mediaEvent := internal_twilio.TwilioMediaEvent{
		Media: &internal_twilio.TwilioMedia{
			Payload: tws.Encoder().EncodeToString(providerAudio),
		},
	}
	err := tws.handleMediaEvent(mediaEvent)
	require.NoError(t, err)

	select {
	case stream := <-tws.LowCh:
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
		Provider:       "twilio",
	}
	mediaEngine := &fakeTwilioMediaEngine{processError: errors.New("media process failed")}
	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	tws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     tws.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  tws.Input,
	})

	providerAudio := make([]byte, internal_twilio.OutputChunkSize)
	for i := range providerAudio {
		providerAudio[i] = internal_twilio.MulawSilence
	}
	mediaEvent := internal_twilio.TwilioMediaEvent{
		Media: &internal_twilio.TwilioMedia{
			Payload: tws.Encoder().EncodeToString(providerAudio),
		},
	}
	err := tws.handleMediaEvent(mediaEvent)
	require.ErrorContains(t, err, "media process failed")
}

func TestHandleMediaEvent_MissingMediaPayloadDoesNotPanic(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "twilio",
	}
	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}

	err := tws.handleMediaEvent(internal_twilio.TwilioMediaEvent{})
	require.NoError(t, err)
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tool-call-id-123",
		ToolId: "tool-id-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tool-call-id-123", result.GetId())
		assert.Equal(t, "tool-id-456", result.GetToolId())
		assert.Equal(t, "end_conversation", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}
}

func TestSend_EndConversation_NilConnectionStillPushesToolCallResult(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger,
			&callcontext.CallContext{
				AssistantID:    1,
				ConversationID: 2,
			},
			nil,
			nil,
		),
		connection: nil,
	}

	toolCall := &protos.ConversationToolCall{
		Id:     "tool-call-id-123",
		ToolId: "tool-id-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "completed", result.GetResult()["status"])
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}
}

func TestSend_EndConversation_DoesNotCancelStreamer(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-1",
		ToolId: "t-1",
		Name:   "hangup",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	_ = tws.Send(toolCall)

	// Drain the tool call result.
	select {
	case <-tws.CriticalCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Context should remain open; disconnect is owned by handleToolResult.
	select {
	case <-tws.Ctx.Done():
		t.Fatal("streamer context should remain open")
	default:
	}
	assert.False(t, tws.closed.Load(), "streamer should remain open")
}

func TestSend_Disconnection_LogsTwilioClientError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	observer := &recordingObserver{}

	conn, cleanup := testWSPair(t)
	defer cleanup()

	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger,
			&callcontext.CallContext{
				AssistantID:    1,
				ConversationID: 2,
				ChannelUUID:    "CA123",
			},
			nil,
			observer,
		),
		connection: conn,
	}

	err = tws.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
	})
	require.NoError(t, err)

	require.Len(t, observer.records, 2)
	logRecord, ok := observer.records[0].(observability.RecordLog)
	require.True(t, ok)
	assert.Equal(t, "Failed to create Twilio client for server-side disconnect", logRecord.Message)
	assert.Equal(t, "CA123", logRecord.Attributes["conversation_uuid"])
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER.String(), logRecord.Attributes["disconnection_type"])

	metricRecord, ok := observer.records[1].(observability.RecordMetric)
	require.True(t, ok)
	require.Len(t, metricRecord.Metrics, 1)
	assert.Equal(t, observability.MetricCallStatus, metricRecord.Metrics[0].Name)
	assert.Equal(t, "FAILED", metricRecord.Metrics[0].Value)
}

func TestSend_TransferConversation_MissingTarget(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-missing",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""},
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-missing", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or call ID")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_TransferConversation_NoCallUUID(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	// ChannelUUID is already empty from newTestTwilioStreamer
	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-no-uuid",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-no-uuid", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or call ID")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}
