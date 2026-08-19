package internal_llm_model

import (
	"context"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

// Smoke tests for response handling: stale filtering, stream chunks, completion, and end-to-end LLM paths.

func TestModel_ResponsePipeline_DropsStaleResponse(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	require.Nil(t, e.currentPacket)

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Data:      &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"ignored"}}}},
	}})

	evt, ok := findPacket[internal_type.ObservabilityEventRecordPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, observability.AgentDiscarded, evt.Record.Event)
	require.False(t, evt.Record.OccurredAt.IsZero())
	require.Equal(t, "ignored", evt.Record.Attributes["script"])
	require.Empty(t, stream.sendCalls)
}

func TestModel_ResponsePipeline_Error_EmitsAgentErrorAndEvent(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Error:     &protos.Error{ErrorMessage: "provider down"},
	}})

	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", errPkt.ContextID)
	require.EqualError(t, errPkt.Error, "provider down")
	evt, ok := findPacket[internal_type.ObservabilityEventRecordPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, observability.AgentError, evt.Record.Event)
}

func TestModel_ResponsePipeline_Chunk_EmitsDeltaEvenWhenEmpty(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Data:      &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{""}}}},
	}})

	delta, ok := findPacket[internal_type.LLMResponseDeltaPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", delta.ContextID)
	require.Equal(t, "", delta.Text)
}

func TestModel_ResponsePipeline_DoneWithToolCalls_ExecutesToolsAndOpensBlock(t *testing.T) {
	e, comm, _, toolExec := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.StreamChatOutput{
		RequestId:    "ctx-1",
		FinishReason: "tool_calls",
		Metrics:      []*protos.Metric{{Name: "token_count", Value: "3"}},
		Data:         testToolAssistantMessage("t1", "t2"),
	}})

	require.Len(t, toolExec.calls, 1)
	require.Equal(t, "ctx-1", toolExec.calls[0].contextID)
	require.Len(t, toolExec.calls[0].tools, 2)
	require.Equal(t, "ctx-1", e.history.PendingContextID())

	done, ok := findPacket[internal_type.LLMResponseDonePacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", done.ContextID)
}

func TestModel_ResponsePipeline_DoneWithoutToolCalls_AppendsAssistant(t *testing.T) {
	e, comm, _, toolExec := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.StreamChatOutput{
		RequestId:    "ctx-1",
		FinishReason: "stop",
		Metrics:      []*protos.Metric{{Name: "token_count", Value: "3"}},
		Data:         &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"final"}}}},
	}})

	require.Empty(t, toolExec.calls)
	require.Empty(t, e.history.PendingContextID())
	snap := e.history.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, "final", snap[0].GetAssistant().GetContents()[0])
}

func TestModel_Flow_UserToLLM_Stream_Done(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-flow-1", Text: "hello"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-1",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Hi"}},
			},
		},
	})

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-flow-1",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Hi there"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](comm.pkts)
	require.Len(t, deltas, 1)
	require.Equal(t, "ctx-flow-1", deltas[0].ContextID)
	require.Equal(t, "Hi", deltas[0].Text)

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.Len(t, dones, 1)
	require.Equal(t, "ctx-flow-1", dones[0].ContextID)
	require.Equal(t, "Hi there", dones[0].Text)
}

func TestModel_Interrupt_LateResponseStillEmittedForPersistence(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-int"}

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-int"}))

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-int",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"late after interrupt"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.Len(t, dones, 1)
	require.Equal(t, "ctx-int", dones[0].ContextID)
	require.Equal(t, "late after interrupt", dones[0].Text)
}
