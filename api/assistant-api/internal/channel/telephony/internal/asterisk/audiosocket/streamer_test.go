// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_audiosocket

import (
	"bufio"
	"context"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"net"
	"testing"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_asterisk "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk/internal"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStreamer builds a minimal Streamer suitable for Send() tests.
// The returned net.Conn is the "remote" side of the pipe that receives
// frames written by the streamer.
func newTestStreamer(t *testing.T) (*Streamer, net.Conn) {
	t.Helper()

	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	cc := &callcontext.CallContext{
		ContextID:   "test-uuid",
		AssistantID: 1,
	}

	local, remote := net.Pipe()

	reader := bufio.NewReader(local)
	writer := bufio.NewWriter(local)

	audioProcessor, err := internal_asterisk.NewAudioProcessor(logger, internal_asterisk.AudioProcessorConfig{
		AsteriskConfig:   internal_audio.NewLinear8khzMonoAudioConfig(),
		DownstreamConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		SilenceByte:      0x00,
		FrameSize:        320,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	as := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger, cc, nil, nil,
		),
		conn:           local,
		reader:         reader,
		writer:         writer,
		audioProcessor: audioProcessor,
		ctx:            ctx,
		cancel:         cancel,
		initialUUID:    "test-uuid",
	}
	as.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     ctx,
		Logger:      logger,
		MediaEngine: audioProcessor,
		OutputSink:  as.sendOutputFrame,
	})

	t.Cleanup(func() {
		cancel()
		local.Close()
		remote.Close()
	})

	return as, remote
}

func drainRemoteConn(remote net.Conn) {
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	as, remote := newTestStreamer(t)
	drainRemoteConn(remote)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-123",
		ToolId: "tool-456",
		Name:   "end_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// 1. Verify the ConversationToolCallResult was pushed to CriticalCh.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-123", result.GetId())
		assert.Equal(t, "tool-456", result.GetToolId())
		assert.Equal(t, "end_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}

	select {
	case <-as.Context().Done():
		t.Fatal("streamer context should remain open; teardown is owned by Talk loop")
	default:
	}
}

func TestSend_EndConversation_SecondCall_StillPushesToolResult(t *testing.T) {
	as, remote := newTestStreamer(t)
	drainRemoteConn(remote)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-789",
		ToolId: "tool-012",
		Name:   "end_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	// First call emits tool result only.
	err := as.Send(toolCall)
	require.NoError(t, err)

	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult")
		assert.Equal(t, "call-789", result.GetId())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Second call should still return nil and push another tool result.
	err = as.Send(toolCall)
	require.NoError(t, err)
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-789", result.GetId())
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second ConversationToolCallResult")
	}

	// No extra messages should be present.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		t.Fatalf("unexpected extra message after second end_conversation: %T", msg)
	case <-time.After(200 * time.Millisecond):
		// expected: no extra messages
	}
}

func TestSend_ConversationDisconnection_ClosesStreamer(t *testing.T) {
	as, remote := newTestStreamer(t)
	drainRemoteConn(remote)

	err := as.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
	})
	require.NoError(t, err)

	// Server-initiated Send no longer requeues the disconnect onto CriticalCh —
	// the server callsite already knows the reason. The talker exits via the
	// Recv-err path once Cancel cancels s.Ctx.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		t.Fatalf("server-initiated Send must not push to CriticalCh; got %T", msg)
	default:
	}

	// Streamer should be closed: s.Ctx cancelled so Talker.Recv returns EOF.
	select {
	case <-as.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("expected streamer context to be cancelled after disconnect")
	}
}

func TestSend_TransferConversation_Unsupported(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote so nothing blocks.
	drainRemoteConn(remote)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-abc",
		ToolId: "tool-def",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// Should push a failed result because transfer is not supported.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-abc", result.GetId())
		assert.Equal(t, "tool-def", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	select {
	case <-as.Context().Done():
		t.Fatal("streamer should remain open after failed transfer")
	default:
	}
}

func TestSend_TransferConversation_EmptyToolId_StillPushesFailedResult(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	drainRemoteConn(remote)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-xyz",
		ToolId: "", // empty ToolId should still return a failed result
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// Transfer failure should still emit a failed result with empty ToolId.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-xyz", result.GetId())
		assert.Equal(t, "", result.GetToolId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}

func TestSend_UnknownToolCallAction_NoOp(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	drainRemoteConn(remote)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-unk",
		ToolId: "tool-unk",
		Name:   "unknown_action",
		Action: protos.ToolCallAction(999),
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// No result or frame for an unrecognized action.
	select {
	case <-as.CriticalCh.Ready():
		msg, err := as.CriticalCh.TryReceive()
		require.NoError(t, err)
		t.Fatalf("unexpected message on CriticalCh: %T", msg)
	case <-time.After(200 * time.Millisecond):
		// Expected: no message.
	}

	select {
	case <-as.Context().Done():
		t.Fatal("streamer should remain open")
	default:
	}
}

func TestSend_InterruptionIsNotificationOnly(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	drainRemoteConn(remote)

	interruption := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
	}

	err := as.Send(interruption)
	require.NoError(t, err)
}

func TestSend_OutputControlsRouteBeforeAssistantAudio(t *testing.T) {
	as, remote := newTestStreamer(t)
	defer remote.Close()
	audio := make([]byte, 6400)
	for i := range audio {
		audio[i] = byte(i)
	}

	require.NoError(t, as.Send(&protos.ConversationAssistantMessage{
		Id:        "response-1",
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: audio},
		Completed: true,
	}))
	require.NoError(t, as.Send(&protos.ConversationPlaybackPause{}))
	assert.Nil(t, as.mediaSession.NextFrame())
	require.NoError(t, as.Send(&protos.ConversationPlaybackContinue{}))
	assert.NotEmpty(t, as.mediaSession.NextFrame())

	require.NoError(t, as.Send(&protos.ConversationAssistantMessage{
		Id:        "response-2",
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: audio},
		Completed: true,
	}))
	require.NoError(t, as.Send(&protos.ConversationPlaybackFlush{}))
	require.NoError(t, as.Send(&protos.ConversationAssistantMessage{
		Id:        "response-2",
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: audio},
		Completed: true,
	}))
	assert.Nil(t, as.mediaSession.NextFrame())
	require.NoError(t, as.Send(&protos.ConversationAssistantMessage{
		Id:        "response-3",
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: audio},
		Completed: true,
	}))
	assert.NotEmpty(t, as.mediaSession.NextFrame())
}

// Compile-time check that Streamer implements internal_type.Streamer.
var _ internal_type.Streamer = (*Streamer)(nil)
