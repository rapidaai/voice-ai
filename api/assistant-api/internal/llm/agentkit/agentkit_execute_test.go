package internal_llm_agentkit

import (
	"context"
	"fmt"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_UserInputPacket(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello world",
	})

	require.NoError(t, err)

	evs := findPackets[internal_type.ObservabilityEventRecordPacket](collector.all())
	require.Len(t, evs, 1)
	assert.Equal(t, observability.AgentStarted, evs[0].Record.Event)
	assert.Equal(t, "11", evs[0].Record.Attributes["input_char_count"])

	talker.mu.Lock()
	defer talker.mu.Unlock()
	require.Len(t, talker.sendCalls, 1)
	msg := talker.sendCalls[0].GetUser()
	require.NotNil(t, msg)
	assert.Equal(t, "hello world", msg.GetText())
}

func TestExecute_InjectMessagePacket(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
		ContextID: "ctx-1",
		Text:      "static text",
	})

	require.NoError(t, err)
	assert.Empty(t, collector.all(), "InjectMessagePacket should emit no packets")

	talker.mu.Lock()
	defer talker.mu.Unlock()
	require.Len(t, talker.sendCalls, 1)
	msg := talker.sendCalls[0].GetAssistant()
	require.NotNil(t, msg)
	assert.Equal(t, "ctx-1", msg.GetId())
	assert.Equal(t, "static text", msg.GetText())
	assert.True(t, msg.GetCompleted())
	assert.NotNil(t, msg.GetTime())
}

func TestExecute_ToolPackets(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolCallPacket{
		ContextID: "ctx-1",
		ToolID:    "tool-1",
		Name:      "lookup",
		Arguments: map[string]string{"q": "test"},
	})
	require.NoError(t, err)

	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-1",
		ToolID:    "tool-1",
		Name:      "lookup",
		Action:    protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Result:    map[string]string{"ok": "true"},
	})
	require.NoError(t, err)
	assert.Empty(t, collector.all(), "tool packets should emit no local packets")

	talker.mu.Lock()
	defer talker.mu.Unlock()
	require.Len(t, talker.sendCalls, 2)

	toolCall := talker.sendCalls[0].GetToolCall()
	require.NotNil(t, toolCall)
	assert.Equal(t, "ctx-1", toolCall.GetId())
	assert.Equal(t, "tool-1", toolCall.GetToolId())
	assert.Equal(t, "lookup", toolCall.GetName())
	assert.Equal(t, map[string]string{"q": "test"}, toolCall.GetArgs())
	assert.NotNil(t, toolCall.GetTime())

	toolResult := talker.sendCalls[1].GetToolCallResult()
	require.NotNil(t, toolResult)
	assert.Equal(t, "ctx-1", toolResult.GetId())
	assert.Equal(t, "tool-1", toolResult.GetToolId())
	assert.Equal(t, "lookup", toolResult.GetName())
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, toolResult.GetAction())
	assert.Equal(t, map[string]string{"ok": "true"}, toolResult.GetResult())
	assert.NotNil(t, toolResult.GetTime())
}

func TestExecute_UnsupportedPacket(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.EndOfSpeechPacket{ContextID: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitExecuteUnsupportedPacket)
}

func TestExecute_UserInputPacket_SendError(t *testing.T) {
	talker := newMockTalker()
	talker.sendErr = fmt.Errorf("connection lost")
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestExecute_LLMInterruptPacket_ClearsCurrentContext(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	e.activeContextID = "ctx-1"
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	assert.Equal(t, "", e.activeContextID)

	talker.mu.Lock()
	defer talker.mu.Unlock()
	require.Len(t, talker.sendCalls, 1)
	interruption := talker.sendCalls[0].GetInterruption()
	require.NotNil(t, interruption)
	assert.Equal(t, "ctx-1", interruption.GetId())
	assert.Equal(t, protos.ConversationInterruption_INTERRUPTION_TYPE_WORD, interruption.GetType())
	assert.NotNil(t, interruption.GetTime())
}

func TestExecute_LLMInterruptPacket_SendError(t *testing.T) {
	talker := newMockTalker()
	talker.sendErr = fmt.Errorf("interrupt failed")
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitConnectionSend)
	assert.Contains(t, err.Error(), "interrupt failed")
}

func TestExecute_PropagatesSendError(t *testing.T) {
	talker := newMockTalker()
	talker.sendErr = fmt.Errorf("write failed")
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitConnectionSend)
	assert.Contains(t, err.Error(), "write failed")
}
