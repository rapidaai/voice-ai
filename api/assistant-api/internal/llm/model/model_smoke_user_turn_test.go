package internal_llm_model

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

// Smoke tests for user-turn entry points, history preparation, and turn-over-turn behavior.

func TestModel_ExecuteUserTurn_SendsChatAndAppendsUser(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	msgs := stream.sendCalls[0].GetChat().GetConversations()
	require.NotEmpty(t, msgs)
	require.Equal(t, "user", msgs[len(msgs)-1].GetRole())
	require.Equal(t, "hello", msgs[len(msgs)-1].GetUser().GetContent())
	require.Len(t, e.history.Snapshot(), 1)

	evt, ok := findPacket[internal_type.ObservabilityEventRecordPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, observability.AgentStarted, evt.Record.Event)
	require.Equal(t, "5", evt.Record.Attributes["input_char_count"])
}

func TestModel_ExecuteUserTurn_BlocksInvalidHistory(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.messages = append(e.history.messages, &protos.Message{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{}}})

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 0)

	pkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", pkt.ContextID)
	require.ErrorContains(t, pkt.Error, "history integrity")
}

func TestModel_InjectMessage_AppendsToHistory(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{ContextID: "ctx-1", Text: "hello from inject"})
	require.NoError(t, err)
	require.Empty(t, comm.pkts)

	snap := e.history.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, "assistant", snap[0].GetRole())
	require.Equal(t, "hello from inject", snap[0].GetAssistant().GetContents()[0])
}

func TestModel_InjectThenUser_RequestContainsInjectedHistory(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{ContextID: "ctx-1", Text: "hello inject"}))
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-2", Text: "user text"}))

	require.Len(t, stream.sendCalls, 1)
	convs := stream.sendCalls[0].GetChat().GetConversations()
	require.GreaterOrEqual(t, len(convs), 2)
	require.Equal(t, "assistant", convs[len(convs)-2].GetRole())
	require.Equal(t, "hello inject", convs[len(convs)-2].GetAssistant().GetContents()[0])
	require.Equal(t, "user", convs[len(convs)-1].GetRole())
	require.Equal(t, "user text", convs[len(convs)-1].GetUser().GetContent())
}

func TestModel_UserUser_LateFirstResponseDropped(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)

	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "first"}))
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-2", Text: "second"}))
	require.Len(t, stream.sendCalls, 2)

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-1",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"late first"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	e.handleResponse(context.Background(), comm, &protos.StreamChatOutput{
		RequestId: "ctx-2",
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"second done"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "2"}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](comm.pkts)
	require.Len(t, dones, 1)
	require.Equal(t, "ctx-2", dones[0].ContextID)
	require.Equal(t, "second done", dones[0].Text)
}

func TestModel_ExecuteUserTurn_SupersedesOpenToolBlock(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-old", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-new", Text: "new user msg"})
	require.NoError(t, err)

	evt, ok := findPacket[internal_type.ObservabilityLogRecordPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "supersede_tool_block", evt.Record.Attributes["operation"])
	require.Equal(t, "user_interrupted", evt.Record.Attributes["reason"])
}

func TestModel_ExecuteUserTurn_FiltersConnectionAndCredentialFromModelParameters(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	comm.assistant.AssistantProviderModel.AssistantModelOptions = []*internal_assistant_entity.AssistantProviderModelOption{
		{Metadata: gorm_models.Metadata{Key: "model.temperature", Value: "0.6"}},
		{Metadata: gorm_models.Metadata{Key: "connection.transport", Value: "websocket"}},
		{Metadata: gorm_models.Metadata{Key: "rapida.credential_id", Value: "9"}},
	}
	comm.options = utils.Option{
		"model.top_p":                                   "0.8",
		internal_options.ModelOptionCredentialID:        "12",
		internal_options.ModelOptionConnectionTransport: "chat_complete",
	}
	e.providerOptions = withModelOverrides(comm.assistant.AssistantProviderModel.GetOptions(), comm.GetOptions())

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello",
	})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)

	modelParameters := stream.sendCalls[0].GetChat().GetModelParameters()
	require.Contains(t, modelParameters, "model.temperature")
	require.Contains(t, modelParameters, "model.top_p")
	_, hasTransport := modelParameters["connection.transport"]
	require.False(t, hasTransport)
	_, hasCredential := modelParameters["rapida.credential_id"]
	require.False(t, hasCredential)
	_, hasOverrideTransport := modelParameters[internal_options.ModelOptionConnectionTransport]
	require.False(t, hasOverrideTransport)
	_, hasOverrideCredential := modelParameters[internal_options.ModelOptionCredentialID]
	require.False(t, hasOverrideCredential)
}
