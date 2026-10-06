package internal_llm_model

import (
	"context"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

// Smoke tests for deep multi-turn conversational sequences (4-5 turns) that
// mix direct completions and tool-call/follow-up paths.

func TestModel_MultiTurn_4Turn_MixedDoneAndToolFlow(t *testing.T) {
	// MT5: 4-turn mixed sequence (done -> tool-flow -> done -> tool-flow) stays consistent end-to-end.
	e, comm, stream, _ := newModelTestEnv(t)

	// Turn 1: user -> llm done
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "t1", Text: "hello"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t1", Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"t1 done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "3"}},
	})

	// Turn 2: user -> llm tool call -> tool result -> llm done
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "t2", Text: "check weather"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t2", FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"checking"},
					ToolCalls: []*protos.ToolCall{
						{Id: "t2-tool", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: `{"city":"sf"}`}},
					},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "5"}},
	})
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "t2", ToolID: "t2-tool", Name: "get_weather", Result: map[string]string{"temp": "72F"},
	}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t2", Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"t2 done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "4"}},
	})

	// Turn 3: user -> llm done
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "t3", Text: "thanks"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t3", Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"t3 done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	// Turn 4: user -> llm tool call -> tool result -> llm done
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "t4", Text: "book a cab"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t4", FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"booking"},
					ToolCalls: []*protos.ToolCall{
						{Id: "t4-tool", Type: "function", Function: &protos.FunctionCall{Name: "book_cab", Arguments: `{"to":"airport"}`}},
					},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "5"}},
	})
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "t4", ToolID: "t4-tool", Name: "book_cab", Result: map[string]string{"status": "confirmed"},
	}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "t4", Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"t4 done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "4"}},
	})

	// Assertions: 4 user sends + 2 tool follow-up sends.
	require.Len(t, stream.sendCalls, 6)
	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.Len(t, dones, 4)
	require.Equal(t, "t4", dones[len(dones)-1].ContextID)
	require.Equal(t, "t4 done", dones[len(dones)-1].Text)
}

func TestModel_MultiTurn_5Turn_WithContextSwitchDuringPendingTool(t *testing.T) {
	// MT6: 5-turn sequence with pending-tool supersession ignores late old-tool result and continues.
	e, comm, stream, _ := newModelTestEnv(t)

	// Turn 1 done.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "s1", Text: "hi"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s1", Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"s1 done"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
	})

	// Turn 2 opens tool block.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "s2", Text: "find places"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s2", FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents: []string{"searching"},
				ToolCalls: []*protos.ToolCall{
					{Id: "s2-tool", Type: "function", Function: &protos.FunctionCall{Name: "search_places", Arguments: `{"q":"coffee"}`}},
				},
			}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "3"}},
	})

	// Turn 3 arrives before s2 tool result => supersede pending s2 block.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "s3", Text: "new question"}))

	// Late tool result for s2 should be ignored and not create extra send.
	beforeLate := len(stream.sendCalls)
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "s2", ToolID: "s2-tool", Name: "search_places", Result: map[string]string{"n": "3"},
	}))
	require.Equal(t, beforeLate, len(stream.sendCalls))

	// Turn 3 done.
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s3", Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"s3 done"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	// Turn 4 done.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "s4", Text: "another"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s4", Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"s4 done"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	// Turn 5 tool flow completes.
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "s5", Text: "final tool request"}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s5", FinishReason: "tool_calls",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents: []string{"working s5"},
				ToolCalls: []*protos.ToolCall{
					{Id: "s5-tool", Type: "function", Function: &protos.FunctionCall{Name: "do_work", Arguments: `{"x":1}`}},
				},
			}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "3"}},
	})
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "s5", ToolID: "s5-tool", Name: "do_work", Result: map[string]string{"ok": "1"},
	}))
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "s5", Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"s5 done"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	events := findPackets[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.NotEmpty(t, events)
	foundIgnoredOldTool := false
	for _, ev := range events {
		if ev.Record.Attributes["component"] == observability.ComponentTool.String() && ev.Record.Attributes["operation"] == "ignore_tool_result" && ev.Record.Attributes["reason"] == "context_or_id_mismatch" {
			foundIgnoredOldTool = true
			break
		}
	}
	require.True(t, foundIgnoredOldTool, "late s2 tool result should be ignored")

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.NotEmpty(t, dones)
	require.Equal(t, "s5", dones[len(dones)-1].ContextID)
	require.Equal(t, "s5 done", dones[len(dones)-1].Text)
}
