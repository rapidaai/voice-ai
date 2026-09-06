// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_exotel_telephony

import (
	"errors"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_exotel "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel/internal"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExotelMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

func (engine *fakeExotelMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
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

func (engine *fakeExotelMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error {
	return nil
}

func (engine *fakeExotelMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeExotelMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeExotelMediaEngine) ClearOutputBuffer() {}

func (engine *fakeExotelMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeExotelMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func (engine *fakeExotelMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakeExotelMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

// newTestExotelStreamer creates an exotelWebsocketStreamer without starting
// the background WebSocket reader goroutine. The connection is nil so Cancel()
// is a no-op on the transport side.
func newTestExotelStreamer(t *testing.T) *exotelWebsocketStreamer {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		ChannelUUID:    "test-channel-uuid",
	}
	return &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger, cc, nil, nil,
		),
		streamID:   "test-stream",
		connection: nil, // nil so Cancel() skips conn.Close()
	}
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tool-call-id-123",
		ToolId: "tool-id-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// The ToolCallResult should have been pushed to CriticalCh (since Input
	// routes ConversationToolCallResult to CriticalCh).
	select {
	case msg := <-exotel.CriticalCh:
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

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-exotel.Ctx.Done():
		t.Fatal("streamer context should remain open")
	default:
	}
}

func TestSend_EndConversation_DoesNotCancelStreamerImmediately(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-1",
		ToolId: "t-1",
		Name:   "hangup",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	_ = exotel.Send(toolCall)

	assert.False(t, exotel.closed.Load(), "streamer should remain open")
}

func TestSend_TransferConversation_PushesFailedResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer",
		ToolId: "t-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// Transfer is not supported for Exotel, so it should push a failed result.
	select {
	case msg := <-exotel.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer", result.GetId())
		assert.Equal(t, "t-transfer", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}

	// Streamer should NOT be cancelled for transfer failure.
	select {
	case <-exotel.Ctx.Done():
		t.Fatal("Streamer context should NOT be cancelled on transfer failure")
	default:
		// expected - context is still alive
	}
}

func TestNewExotelWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "exotel",
	}

	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(callContext),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	exotel, ok := streamer.(*exotelWebsocketStreamer)
	require.True(t, ok, "expected exotel websocket streamer")
	defer exotel.Cancel()

	require.NotNil(t, exotel.mediaSession)
}

func TestHandleMediaEvent_EmitsBridgeUserAudio(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "exotel",
	}
	mediaEngine := &fakeExotelMediaEngine{}
	exotel := &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	exotel.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     exotel.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  exotel.Input,
	})

	providerAudio := []byte{9, 8, 7}
	mediaEvent := internal_exotel.ExotelMediaEvent{
		Media: &internal_exotel.ExotelMedia{
			Payload: exotel.Encoder().EncodeToString(providerAudio),
		},
	}
	err := exotel.handleMediaEvent(mediaEvent)
	require.NoError(t, err)

	select {
	case stream := <-exotel.LowCh:
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
		Provider:       "exotel",
	}
	mediaEngine := &fakeExotelMediaEngine{processError: errors.New("media process failed")}
	exotel := &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	exotel.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     exotel.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  exotel.Input,
	})

	mediaEvent := internal_exotel.ExotelMediaEvent{
		Media: &internal_exotel.ExotelMedia{
			Payload: exotel.Encoder().EncodeToString([]byte{9, 8, 7}),
		},
	}
	err := exotel.handleMediaEvent(mediaEvent)
	require.ErrorContains(t, err, "media process failed")
}

func TestHandleMediaEvent_MissingMediaPayloadDoesNotPanic(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	err := exotel.handleMediaEvent(internal_exotel.ExotelMediaEvent{})
	require.NoError(t, err)
}

func TestSend_TransferConversation_NoToolId_StillPushesFailedResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-no-tool",
		ToolId: "", // empty ToolId
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// Transfer failure should still emit a failed result even when ToolId is empty.
	select {
	case msg := <-exotel.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-no-tool", result.GetId())
		assert.Equal(t, "", result.GetToolId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}
}
