package internal_llm_agentflow

import (
	"context"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/node"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/prompt"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

type fakeAgentHandler struct {
	result prompt.Result
}

func (f fakeAgentHandler) Type() string {
	return NodeTypeAgent
}

func (f fakeAgentHandler) Close(context.Context) error {
	return nil
}

func (f fakeAgentHandler) Execute(ctx context.Context, request node.Request) (node.Result, error) {
	if f.result.Text != "" && f.result.TransitionID == "" && f.result.TransitionName == "" {
		_ = request.Communication.OnPacket(ctx,
			internal_type.LLMResponseDeltaPacket{ContextID: request.ContextID, Text: f.result.Text},
			internal_type.LLMResponseDonePacket{ContextID: request.ContextID, Text: f.result.Text},
		)
		request.RuntimeState.AppendUserTurn(request.InputText)
		request.RuntimeState.AppendAssistantTurn(f.result.Text)
		request.RuntimeState.RecordNodeOutputValue(request.Node.ID, "response", f.result.Text)
		return node.Result{WaitForNextInput: true}, nil
	}

	transitions := request.Node.AgentTransitions()
	transitionID := f.result.TransitionID
	if transitionID == "" {
		for _, transition := range transitions {
			if transition.Name == f.result.TransitionName {
				transitionID = transition.ID
				break
			}
		}
	}
	request.RuntimeState.AppendUserTurn(request.InputText)
	request.RuntimeState.RecordTransitionOutput(request.Node.ID, transitionID, f.result.TransitionName, f.result.Arguments)
	_ = request.Communication.OnPacket(ctx, internal_type.ObservabilityEventRecordPacket{
		ContextID: request.ContextID,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.RecordEvent{
			Component: observability.ComponentAgent,
			Event:     observability.AgentTransitionTriggered,
			Attributes: observability.Attributes{
				"context_id":      request.ContextID,
				"from_node_id":    request.Node.ID,
				"from_node_label": request.Node.Label,
				"transition_id":   transitionID,
				"transition_name": f.result.TransitionName,
				"arguments":       observability.AttributeValue(f.result.Arguments),
			},
			OccurredAt: time.Now(),
		},
	})
	return node.Result{RouteHandles: []string{transitionID, f.result.TransitionName}}, nil
}

func withFakeAgentResult(result prompt.Result) Option {
	return withHandlerFactories(map[string]node.HandlerFactory{
		NodeTypeAgent: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (node.Handler, error) {
			return fakeAgentHandler{result: result}, nil
		},
	})
}

type fakeCommunication struct {
	assistant *internal_assistant_entity.Assistant
	packets   []internal_type.Packet
}

func (f *fakeCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	f.packets = append(f.packets, pkts...)
	return nil
}

func (f *fakeCommunication) IntegrationCaller() integration_client.IntegrationServiceClient {
	return nil
}

func (f *fakeCommunication) VaultCaller() web_client.VaultClient { return nil }

func (f *fakeCommunication) DeploymentCaller() endpoint_client.DeploymentServiceClient {
	return nil
}

func (f *fakeCommunication) Auth() *types.Authentication { return &types.Authentication{} }

func (f *fakeCommunication) GetSource() utils.RapidaSource { return utils.PhoneCall }

func (f *fakeCommunication) Assistant() (*internal_assistant_entity.Assistant, error) {
	return f.assistant, nil
}

func (f *fakeCommunication) GetMode() type_enums.MessageMode { return "" }

func (f *fakeCommunication) Conversation() (*internal_conversation_entity.AssistantConversation, error) {
	return &internal_conversation_entity.AssistantConversation{
		Audited:     gorm_model.Audited{Id: 10},
		AssistantId: 20,
		Identifier:  "caller",
	}, nil
}

func (f *fakeCommunication) GetHistories() []internal_type.MessagePacket { return nil }

func (f *fakeCommunication) Metadata() map[string]interface{} { return nil }

func (f *fakeCommunication) GetArgs() map[string]interface{} { return nil }

func (f *fakeCommunication) GetOptions() utils.Option { return utils.Option{} }

func (f *fakeCommunication) GetKnowledge(context.Context, uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return nil, nil
}

func (f *fakeCommunication) RetrieveToolKnowledge(
	context.Context,
	*internal_knowledge_gorm.Knowledge,
	string,
	string,
	map[string]interface{},
	*internal_type.KnowledgeRetrieveOption,
) ([]internal_type.KnowledgeContextResult, error) {
	return nil, nil
}

func (f *fakeCommunication) IsConditionAllowed(utils.Option, string) bool { return true }

func TestExecutorStaticMessageThenEnd(t *testing.T) {
	comm := &fakeCommunication{assistant: testAssistant(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "message-1", "type": NodeTypeMessage, "label": "Message", "config": map[string]interface{}{"message": "Hello"}},
			{"id": "end-1", "type": NodeTypeEnd, "label": "End"},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "message-1"},
			{"id": "edge-2", "source": "message-1", "sourceHandle": "response", "target": "end-1"},
		},
	})}
	executor, err := New(WithCommunication(comm))
	require.NoError(t, err)

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hi"})
	require.NoError(t, err)

	require.Len(t, comm.packets, 4)
	require.IsType(t, internal_type.LLMResponseDeltaPacket{}, comm.packets[0])
	require.IsType(t, internal_type.LLMResponseDonePacket{}, comm.packets[1])
	matchedEvent, ok := comm.packets[2].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionMatched, matchedEvent.Record.Event)
	require.Equal(t, "message-1", matchedEvent.Record.Attributes["from_node_id"])
	require.Equal(t, "end-1", matchedEvent.Record.Attributes["to_node_id"])
	endCall, ok := comm.packets[3].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	require.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, endCall.Action)
}

func TestExecutorAgentTransitionRoutesToTransfer(t *testing.T) {
	comm := &fakeCommunication{assistant: testAssistant(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "agent-1", "type": NodeTypeAgent, "label": "Agent", "config": map[string]interface{}{
				"transitions": []map[string]interface{}{
					{"id": "transition-transfer", "name": "transfer_to_human", "description": "Transfer caller"},
				},
			}},
			{"id": "transfer-1", "type": NodeTypeTransfer, "label": "Transfer", "config": map[string]interface{}{"transfer_to": "+15551234567"}},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "agent-1"},
			{"id": "edge-2", "source": "agent-1", "sourceHandle": "transition-transfer", "target": "transfer-1"},
		},
	})}
	executor, err := New(
		WithCommunication(comm),
		withFakeAgentResult(prompt.Result{TransitionName: "transfer_to_human", Arguments: utils.Option{"reason": "asked for human"}}),
	)
	require.NoError(t, err)

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "human please"})
	require.NoError(t, err)

	require.Len(t, comm.packets, 3)
	triggeredEvent, ok := comm.packets[0].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionTriggered, triggeredEvent.Record.Event)
	require.Equal(t, "agent-1", triggeredEvent.Record.Attributes["from_node_id"])
	require.Equal(t, "transition-transfer", triggeredEvent.Record.Attributes["transition_id"])
	require.Equal(t, "transfer_to_human", triggeredEvent.Record.Attributes["transition_name"])
	matchedEvent, ok := comm.packets[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionMatched, matchedEvent.Record.Event)
	require.Equal(t, "transfer-1", matchedEvent.Record.Attributes["to_node_id"])
	transferCall, ok := comm.packets[2].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	require.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, transferCall.Action)
	require.Equal(t, "+15551234567", transferCall.Arguments["transfer_to"])
}

func TestExecutorAgentResponseWaitsOnSameNode(t *testing.T) {
	comm := &fakeCommunication{assistant: testAssistant(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "agent-1", "type": NodeTypeAgent, "label": "Agent"},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "agent-1"},
		},
	})}
	executor, err := New(
		WithCommunication(comm),
		withFakeAgentResult(prompt.Result{Text: "I can help with that."}),
	)
	require.NoError(t, err)

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.NoError(t, err)

	require.Len(t, comm.packets, 2)
	require.IsType(t, internal_type.LLMResponseDeltaPacket{}, comm.packets[0])
	require.IsType(t, internal_type.LLMResponseDonePacket{}, comm.packets[1])

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-2", Text: "next"})
	require.NoError(t, err)
	require.Len(t, comm.packets, 4)
	require.IsType(t, internal_type.LLMResponseDeltaPacket{}, comm.packets[2])
	require.IsType(t, internal_type.LLMResponseDonePacket{}, comm.packets[3])
}

func TestExecutorAgentTransitionMissingEdgeEmitsEvent(t *testing.T) {
	comm := &fakeCommunication{assistant: testAssistant(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "agent-1", "type": NodeTypeAgent, "label": "Agent", "config": map[string]interface{}{
				"transitions": []map[string]interface{}{
					{"id": "transition-transfer", "name": "transfer_to_human", "description": "Transfer caller"},
				},
			}},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "agent-1"},
		},
	})}
	executor, err := New(
		WithCommunication(comm),
		withFakeAgentResult(prompt.Result{TransitionName: "transfer_to_human", Arguments: utils.Option{"reason": "asked for human"}}),
	)
	require.NoError(t, err)

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "human please"})
	require.NoError(t, err)

	require.Len(t, comm.packets, 2)
	triggeredEvent, ok := comm.packets[0].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionTriggered, triggeredEvent.Record.Event)
	missingEdgeEvent, ok := comm.packets[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionMissingEdge, missingEdgeEvent.Record.Event)
	require.Equal(t, "agent-1", missingEdgeEvent.Record.Attributes["from_node_id"])
	require.Equal(t, "transition-transfer,transfer_to_human", missingEdgeEvent.Record.Attributes["route_handles"])
}

func TestExecutorConditionRoutesByAgentTransitionParameter(t *testing.T) {
	comm := &fakeCommunication{assistant: testAssistant(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "agent-1", "type": NodeTypeAgent, "label": "Agent", "config": map[string]interface{}{
				"transitions": []map[string]interface{}{
					{
						"id":          "transition-order-status",
						"name":        "check_order_status",
						"description": "Check order status",
						"parameters": []map[string]interface{}{
							{"id": "parameter-order-id", "name": "order_id", "type": "string", "description": "Order id"},
						},
					},
				},
			}},
			{"id": "condition-1", "type": NodeTypeCondition, "label": "If / Else", "config": map[string]interface{}{
				"conditions": []map[string]interface{}{
					{
						"id":           "condition-has-order-id",
						"sourceNodeId": "agent-1",
						"sourceHandle": "transition-order-status",
						"field":        "check_order_status.order_id",
						"operator":     "exists",
					},
				},
			}},
			{"id": "end-matched", "type": NodeTypeEnd, "label": "Matched", "config": map[string]interface{}{"reason": "matched"}},
			{"id": "end-else", "type": NodeTypeEnd, "label": "Else", "config": map[string]interface{}{"reason": "else"}},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "agent-1"},
			{"id": "edge-2", "source": "agent-1", "sourceHandle": "transition-order-status", "target": "condition-1"},
			{"id": "edge-3", "source": "condition-1", "sourceHandle": "condition-has-order-id", "target": "end-matched"},
			{"id": "edge-4", "source": "condition-1", "sourceHandle": "else", "target": "end-else"},
		},
	})}
	executor, err := New(
		WithCommunication(comm),
		withFakeAgentResult(prompt.Result{
			TransitionName: "check_order_status",
			Arguments:      utils.Option{"order_id": "A-100"},
		}),
	)
	require.NoError(t, err)

	err = executor.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "order status"})
	require.NoError(t, err)

	require.Len(t, comm.packets, 4)
	triggeredEvent, ok := comm.packets[0].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionTriggered, triggeredEvent.Record.Event)
	require.Equal(t, "check_order_status", triggeredEvent.Record.Attributes["transition_name"])
	agentMatchedEvent, ok := comm.packets[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionMatched, agentMatchedEvent.Record.Event)
	require.Equal(t, "condition-1", agentMatchedEvent.Record.Attributes["to_node_id"])
	conditionMatchedEvent, ok := comm.packets[2].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	require.Equal(t, observability.AgentTransitionMatched, conditionMatchedEvent.Record.Event)
	require.Equal(t, "condition-1", conditionMatchedEvent.Record.Attributes["from_node_id"])
	require.Equal(t, "end-matched", conditionMatchedEvent.Record.Attributes["to_node_id"])
	endCall, ok := comm.packets[3].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	require.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, endCall.Action)
	require.Equal(t, "matched", endCall.Arguments["reason"])
}

func testAssistant(definition map[string]interface{}) *internal_assistant_entity.Assistant {
	return &internal_assistant_entity.Assistant{
		AssistantProvider: type_enums.AGENTFLOW,
		AssistantProviderAgentflow: &internal_assistant_entity.AssistantProviderAgentflow{
			SchemaVersion: "2026-07-06",
			Definition:    gorm_types.PromptMap(definition),
		},
	}
}
