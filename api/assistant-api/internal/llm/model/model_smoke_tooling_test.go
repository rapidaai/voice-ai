package internal_llm_model

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

// Smoke tests for tool-call lifecycle: pending blocks, tool results, follow-up dispatch, and error handling.

func TestModel_ToolResultIgnored_EmitsEvent(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{ContextID: "ctx-1", ToolID: "t1", Name: "weather", Result: map[string]string{"ok": "1"}})
	require.NoError(t, err)

	evt, ok := findPacket[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, observability.ComponentTool.String(), evt.Record.Attributes["component"])
	require.Equal(t, "ignore_tool_result", evt.Record.Attributes["operation"])
	require.Equal(t, "no_pending_block", evt.Record.Attributes["reason"])
}

func TestModel_ToolResultResolved_TriggersFollowUp(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-tool", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{ContextID: "ctx-tool", ToolID: "t1", Name: "weather", Result: map[string]string{"ok": "1"}})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	require.Equal(t, "ctx-tool", stream.sendCalls[0].GetChat().GetRequestId())

	snap := e.history.Snapshot()
	require.Len(t, snap, 2)
	require.Equal(t, "assistant", snap[0].GetRole())
	require.Equal(t, "tool", snap[1].GetRole())
}

func TestModel_ToolFollowUp_ValidationFailureBlocksSend(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}
	e.history.messages = append(e.history.messages,
		&protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}}}}},
	)

	e.Run(context.Background(), comm, ToolFollowUpPipeline{ContextID: "ctx-1"})
	require.Empty(t, stream.sendCalls)
	require.Equal(t, "", e.currentContextID())
	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", errPkt.ContextID)
}

func TestModel_ToolFollowUpSendFailurePreventsLateDone(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-tool"}
	e.history.AppendAssistant("ctx-tool", testToolAssistantMessage("t1"))
	stream.sendErr = errors.New("send failed")

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-tool",
		ToolID:    "t1",
		Name:      "weather",
		Result:    map[string]string{"ok": "1"},
	}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-tool",
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"late success"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
	})

	require.Empty(t, findPackets[internal_type.LLMResponseDonePacket](comm.pkts))
	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-tool", errPkt.ContextID)
}

func TestModel_Flow_UserToLLM_Stream_Tool_Done(t *testing.T) {
	e, comm, stream, toolExec := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-flow-2", Text: "what is weather"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-2",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Let me check."}},
			},
		},
	})

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId:    "ctx-flow-2",
		FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Let me check."},
					ToolCalls: []*protos.ToolCall{
						{Id: "tool-1", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: `{"city":"sf"}`}},
					},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "5"}},
	})

	require.Len(t, toolExec.calls, 1)
	require.Equal(t, "ctx-flow-2", toolExec.calls[0].contextID)
	require.Len(t, toolExec.calls[0].tools, 1)

	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-flow-2",
		ToolID:    "tool-1",
		Name:      "get_weather",
		Result:    map[string]string{"temp": "72F"},
	})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 2)
	require.Equal(t, "ctx-flow-2", stream.sendCalls[1].GetChat().GetRequestId())

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-2",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"It is 72F in SF."}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "7"}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.Len(t, dones, 1)
	require.Equal(t, "It is 72F in SF.", dones[0].Text)

	snap := e.history.Snapshot()
	require.Len(t, snap, 4)
	require.Equal(t, "user", snap[0].GetRole())
	require.Equal(t, "assistant", snap[1].GetRole())
	require.Equal(t, "tool", snap[2].GetRole())
	require.Equal(t, "assistant", snap[3].GetRole())
}

func TestModel_Flow_ToolResultToLLM_Stream_Done(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendUser("plan my day")
	e.history.AppendAssistant("ctx-flow-3", testToolAssistantMessage("tool-1"))
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-flow-3"}

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-flow-3",
		ToolID:    "tool-1",
		Name:      "calendar_lookup",
		Result:    map[string]string{"next": "meeting at 10"},
	})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	require.Equal(t, "ctx-flow-3", stream.sendCalls[0].GetChat().GetRequestId())

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-3",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"I found your next event."}},
			},
		},
	})

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-3",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Your next meeting is at 10 AM."}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "4"}},
	})

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](comm.pkts)
	require.Len(t, deltas, 1)
	require.Equal(t, "I found your next event.", deltas[0].Text)

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.NotEmpty(t, dones)
	require.Equal(t, "Your next meeting is at 10 AM.", dones[len(dones)-1].Text)
}

func TestModel_MultiTool_OutOfOrderResults_FollowUpOnce(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-tools", testToolAssistantMessage("t1", "t2", "t3"))

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-tools", ToolID: "t2", Name: "fn2", Result: map[string]string{"ok": "1"},
	}))
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-tools", ToolID: "t1", Name: "fn1", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 0)

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-tools", ToolID: "t3", Name: "fn3", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 1)
	require.Equal(t, "ctx-tools", stream.sendCalls[0].GetChat().GetRequestId())
}

func TestModel_ToolResult_DuplicateID_SecondIgnored(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-dup", testToolAssistantMessage("t1"))

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-dup", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 1)

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-dup", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))
	evts := findPackets[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.NotEmpty(t, evts)
	last := evts[len(evts)-1]
	require.Equal(t, "ignore_tool_result", last.Record.Attributes["operation"])
	require.Equal(t, "no_pending_block", last.Record.Attributes["reason"])
}

func TestModel_ToolResult_WrongContext_Ignored(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-a", testToolAssistantMessage("t1"))

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-b", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 0)
	evts := findPackets[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.NotEmpty(t, evts)
	last := evts[len(evts)-1]
	require.Equal(t, "ignore_tool_result", last.Record.Attributes["operation"])
	require.Equal(t, "context_or_id_mismatch", last.Record.Attributes["reason"])
	require.Equal(t, "ctx-a", last.Record.Attributes["pending_context"])
}

func TestModel_ToolResult_UnknownID_Ignored(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-u", testToolAssistantMessage("t1"))

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-u", ToolID: "bad-id", Name: "fn", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 0)
	evts := findPackets[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.NotEmpty(t, evts)
	last := evts[len(evts)-1]
	require.Equal(t, "ignore_tool_result", last.Record.Attributes["operation"])
	require.Equal(t, "context_or_id_mismatch", last.Record.Attributes["reason"])
}

func TestModel_InterruptThenLateToolResult_NoFollowUp(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-int-tool"}
	e.history.AppendAssistant("ctx-int-tool", testToolAssistantMessage("t1"))

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-int-tool"}))
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-int-tool", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 0)
}

func TestModel_ToolFollowUp_SendError_NoPanic(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-senderr"}
	stream.sendErr = errors.New("send failed")
	e.history.AppendAssistant("ctx-senderr", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-senderr", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	require.Equal(t, "", e.currentContextID())
	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-senderr", errPkt.ContextID)
}

func TestModel_ContextSwitch_OldToolResultIgnored_NewUserContinues(t *testing.T) {
	// MT1: tool_call(ctx-1) -> user(ctx-2) -> late tool_result(ctx-1) ignored, ctx-2 continues.
	e, comm, stream, _ := newModelTestEnv(t)

	// Turn 1: user request goes out.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1", Text: "first request",
	}))
	require.Len(t, stream.sendCalls, 1)

	// LLM responds with tool calls for ctx-1 (opens pending tool block).
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId:    "ctx-1",
		FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"checking..."},
					ToolCalls: []*protos.ToolCall{
						{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "lookup", Arguments: `{"q":"x"}`}},
					},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "3"}},
	})
	require.Equal(t, "ctx-1", e.history.PendingContextID())

	// Turn 2: new user context supersedes old pending tool block and must proceed.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-2", Text: "second request",
	}))
	require.Len(t, stream.sendCalls, 2)
	require.Equal(t, "ctx-2", stream.sendCalls[1].GetChat().GetRequestId())

	// Late tool result for ctx-1 should be ignored for execution.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-1", ToolID: "t1", Name: "lookup", Result: map[string]string{"ok": "1"},
	}))
	require.Len(t, stream.sendCalls, 2, "old tool result must not trigger follow-up send")

	events := findPackets[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.NotEmpty(t, events)
	lastEvent := events[len(events)-1]
	require.Equal(t, observability.ComponentTool.String(), lastEvent.Record.Attributes["component"])
	require.Equal(t, "ignore_tool_result", lastEvent.Record.Attributes["operation"])
	require.Equal(t, "context_or_id_mismatch", lastEvent.Record.Attributes["reason"])
	require.Equal(t, "ctx-1", lastEvent.Record.Attributes["pending_context"])

	// Confirm ctx-2 can still complete normally (not stuck).
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-2",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"second answer"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "4"}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.NotEmpty(t, dones)
	require.Equal(t, "ctx-2", dones[len(dones)-1].ContextID)
	require.Equal(t, "second answer", dones[len(dones)-1].Text)
}
