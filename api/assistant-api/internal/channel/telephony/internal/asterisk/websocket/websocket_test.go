// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_websocket

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAsteriskMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

func (engine *fakeAsteriskMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
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

func (engine *fakeAsteriskMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error {
	return nil
}

func (engine *fakeAsteriskMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeAsteriskMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakeAsteriskMediaEngine) ClearOutputBuffer() {}

func (engine *fakeAsteriskMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error {
	return nil
}

func (engine *fakeAsteriskMediaEngine) OutputFrameDuration() time.Duration {
	return 20 * time.Millisecond
}

func (engine *fakeAsteriskMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakeAsteriskMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

// newTestStreamer creates a minimal asteriskWebsocketStreamer for unit testing.
// It has no real WebSocket connection and no AudioProcessor, so transport-level
// side effects (sendCommand, audio processing) are safely no-ops.
func newTestStreamer(t *testing.T) *asteriskWebsocketStreamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return &asteriskWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.BaseTelephonyStreamer{
			BaseStreamer: channel_base.New(channel_base.WithLogger(logger)),
		},
		// connection is nil, so sendCommand returns nil and Cancel skips close.
		// audioProcessor is nil, so stopAudioProcessing is a no-op.
	}
}

func TestNewAsteriskWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "asterisk_ws",
	}

	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(callContext),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	asteriskStreamer, ok := streamer.(*asteriskWebsocketStreamer)
	require.True(t, ok, "expected asterisk websocket streamer")
	defer asteriskStreamer.Cancel()

	require.NotNil(t, asteriskStreamer.mediaSession)
}

func TestHandleAudioData_EmitsBridgeUserAudio(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "asterisk_ws",
	}
	mediaEngine := &fakeAsteriskMediaEngine{}
	asteriskStreamer := &asteriskWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	asteriskStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     asteriskStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  asteriskStreamer.Input,
	})

	providerAudio := []byte{9, 8, 7}
	err = asteriskStreamer.handleAudioData(providerAudio)
	require.NoError(t, err)

	select {
	case stream := <-asteriskStreamer.LowCh:
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

func TestHandleAudioData_ReturnsMediaProcessingError(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	callContext := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       "asterisk_ws",
	}
	mediaEngine := &fakeAsteriskMediaEngine{processError: errors.New("media process failed")}
	asteriskStreamer := &asteriskWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, callContext, nil, nil),
	}
	asteriskStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     asteriskStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  asteriskStreamer.Input,
	})

	err = asteriskStreamer.handleAudioData([]byte{9, 8, 7})
	require.ErrorContains(t, err, "media process failed")
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-123",
		ToolId: "tool-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	// The ToolCallResult should be routed to CriticalCh by Input().
	select {
	case msg := <-aws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-123", result.GetId())
		assert.Equal(t, "tool-456", result.GetToolId())
		assert.Equal(t, "end_conversation", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_EndConversation_DoesNotCancelStreamer(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-789",
		ToolId: "tool-001",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	// Drain the tool call result.
	select {
	case <-aws.CriticalCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-aws.Ctx.Done():
		t.Fatal("streamer context should remain open after end-conversation")
	default:
	}
}

func TestSend_TransferConversation_MissingTarget(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-1",
		ToolId: "tool-transfer",
		Name:   "transfer_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""}, // empty target
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-aws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-1", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or channel name")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_TextAssistantMessage_NoError(t *testing.T) {
	aws := newTestStreamer(t)

	// A text assistant message is a no-op in the switch (only Audio is handled).
	// Verify it returns nil without panicking.
	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello"},
	}

	err := aws.Send(msg)
	assert.NoError(t, err)
}

func TestSend_UnhandledType_NoError(t *testing.T) {
	aws := newTestStreamer(t)

	// An unrecognised message type falls through the switch and returns nil.
	msg := &protos.ConversationUserMessage{}

	err := aws.Send(msg)
	assert.NoError(t, err)
}

func TestDisconnectTypeFromReadError(t *testing.T) {
	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED,
		disconnectTypeFromReadError(nil),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
		disconnectTypeFromReadError(io.EOF),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
		disconnectTypeFromReadError(&websocket.CloseError{Code: websocket.CloseNormalClosure}),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED,
		disconnectTypeFromReadError(errors.New("read failed")),
	)
}
