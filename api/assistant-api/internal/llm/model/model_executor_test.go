package internal_llm_model

import (
	"context"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func TestModel_Interruption_SupersedesPending(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"}
	e.history.AppendAssistant("ctx-1", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	require.Equal(t, "ctx-1", e.currentContextID())
	require.Empty(t, comm.pkts)

	ctx, followUp := e.history.FlushToolBlock()
	require.Equal(t, "ctx-1", ctx)
	require.False(t, followUp)
}

func TestModel_ValidateHistorySequence(t *testing.T) {
	e, _, _, _ := newModelTestEnv(t)

	valid := []*protos.Message{
		testToolAssistantMessage("t1"),
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "t1"}}}}},
	}
	require.NoError(t, e.validateHistorySequence(valid))

	missing := []*protos.Message{testToolAssistantMessage("t1")}
	require.ErrorContains(t, e.validateHistorySequence(missing), "not followed")

	orphan := []*protos.Message{{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "t1"}}}}}}
	require.ErrorContains(t, e.validateHistorySequence(orphan), "orphan")
}

func TestModel_CurrentContextAndStaleCheck(t *testing.T) {
	e, _, _, _ := newModelTestEnv(t)
	require.True(t, e.isStaleResponse("ctx-1"))
	require.Equal(t, "", e.currentContextID())

	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}
	require.False(t, e.isStaleResponse("ctx-1"))
	require.True(t, e.isStaleResponse("ctx-2"))
	require.Equal(t, "ctx-1", e.currentContextID())
}

func TestModel_ResponsePipeline_EmitsLocalTimingMetrics(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}
	e.requestStartedAt = time.Now().Add(-25 * time.Millisecond)
	e.waitingForFirstResponse = true

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"hello"}}},
		},
	})
	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"hello done"}}},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "9"}},
	})

	metrics := findPackets[internal_type.ObservabilityMetricRecordPacket](comm.pkts)
	require.Len(t, metrics, 2)
	require.Len(t, metrics[0].Record.Metrics, 1)
	require.Equal(t, observability.MetricAgentTTFTMs, metrics[0].Record.Metrics[0].Name)
	require.Len(t, metrics[1].Record.Metrics, 2)
	require.Equal(t, observability.MetricAgentResponseCharCount, metrics[1].Record.Metrics[0].Name)
	require.Equal(t, observability.MetricAgentTRTMs, metrics[1].Record.Metrics[1].Name)
}

func TestModel_Close_ThenLatePackets_NoCrash(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-close2"}

	require.NoError(t, e.Close(context.Background()))

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-close2",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"late"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
	})
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-close2", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))

	require.Empty(t, findPackets[internal_type.LLMResponseDonePacket](comm.pkts))
}
