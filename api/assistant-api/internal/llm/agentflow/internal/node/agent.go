// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/prompt"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	internal_llm_model "github.com/rapidaai/api/assistant-api/internal/llm/model"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type AgentHandler struct {
	inputBuilder integration_client_builders.InputChatBuilder
	connection   *internal_llm_model.ModelConnection
	connectionMu sync.Mutex
	providerName string
}

func NewAgentHandler(
	ctx context.Context,
	logger commons.Logger,
	communication internal_type.Communication,
	node schema.Node,
) (*AgentHandler, error) {
	providerName := strings.ToLower(strings.TrimSpace(node.StringConfig("model_provider")))
	if providerName == "" {
		providerName = "openai"
	}

	modelParameters := node.ModelParameters()
	credentialID, err := modelParameters.GetUint64("rapida.credential_id")
	if err != nil {
		return nil, fmt.Errorf("agentflow: rapida.credential_id is required: %w", err)
	}

	credential, err := communication.VaultCaller().GetCredential(ctx, communication.Auth(), credentialID)
	if err != nil {
		return nil, fmt.Errorf("agentflow: unable to get model credential: %w", err)
	}

	connection := internal_llm_model.NewModelConnection(providerName)
	if err := connection.OpenStream(ctx, communication); err != nil {
		return nil, err
	}
	if err := connection.Send(&protos.StreamChatRequest{
		Request: &protos.StreamChatRequest_Configuration{
			Configuration: &protos.StreamChatConfiguration{
				Credential:   &protos.Credential{Id: credential.GetId(), Value: credential.GetValue()},
				ProviderName: providerName,
				ConnectionOptions: func() map[string]string {
					connectionOptions := utils.Option{}
					for key, value := range utils.MergeMaps(modelParameters, communication.GetOptions()) {
						if strings.HasPrefix(key, "connection.") && value != nil {
							connectionOptions[key] = value
						}
					}
					return connectionOptions.ToStringMap()
				}(),
			},
		},
	}); err != nil {
		_ = connection.Close("agentflow configuration failed")
		return nil, err
	}

	return &AgentHandler{
		inputBuilder: integration_client_builders.NewChatInputBuilder(logger),
		connection:   connection,
		providerName: providerName,
	}, nil
}

func (handler *AgentHandler) Type() string {
	return schema.NodeTypeAgent
}

func (handler *AgentHandler) Close(ctx context.Context) error {
	if handler.connection == nil {
		return nil
	}
	return handler.connection.Close("agentflow session ended")
}

func (handler *AgentHandler) Execute(ctx context.Context, request Request) (Result, error) {
	request.RuntimeState.SetCurrentNodeID(request.Node.ID)

	transitions := request.Node.AgentTransitions()
	promptResult, err := handler.runPrompt(ctx, request.Communication, prompt.Request{
		ContextID:        request.ContextID,
		Node:             request.Node,
		InputText:        request.InputText,
		ContinuationText: request.ContinuationText,
		History:          request.RuntimeState.ConversationTurnsSnapshot(),
		Transitions:      transitions,
		Variables:        request.RuntimeState.VariablesSnapshot(),
	})
	if err != nil {
		request.Communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: request.ContextID, Error: err})
		return Result{}, err
	}

	request.RuntimeState.AppendUserTurn(request.InputText)
	request.RuntimeState.AppendAssistantTurn(promptResult.Text)

	if promptResult.TransitionName == "" && promptResult.TransitionID == "" {
		if promptResult.Text != "" {
			request.RuntimeState.RecordNodeOutputValue(request.Node.ID, "response", promptResult.Text)
		}
		return Result{WaitForNextInput: true}, nil
	}

	transitionID := promptResult.TransitionID
	if transitionID == "" {
		for _, transition := range transitions {
			if transition.Name == promptResult.TransitionName {
				transitionID = transition.ID
				break
			}
		}
	}
	request.RuntimeState.RecordTransitionOutput(request.Node.ID, transitionID, promptResult.TransitionName, promptResult.Arguments)
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
				"transition_name": promptResult.TransitionName,
				"arguments":       observability.AttributeValue(promptResult.Arguments),
			},
			OccurredAt: time.Now(),
		},
	})
	return Result{
		RouteHandles: []string{transitionID, promptResult.TransitionName},
	}, nil
}

func (handler *AgentHandler) runPrompt(ctx context.Context, communication internal_type.Communication, request prompt.Request) (prompt.Result, error) {
	modelParameters := request.Node.ModelParameters()
	if handler.connection == nil {
		return prompt.Result{}, fmt.Errorf("agentflow: model connection is not initialized")
	}

	handler.connectionMu.Lock()
	defer handler.connectionMu.Unlock()

	chatInput := handler.streamChatInput(communication, request, modelParameters)
	if err := handler.connection.Send(&protos.StreamChatRequest{Request: &protos.StreamChatRequest_Chat{Chat: chatInput}}); err != nil {
		_ = handler.connection.Close("agentflow prompt send failed")
		return prompt.Result{}, err
	}

	for {
		response, err := handler.connection.Recv()
		if err != nil {
			_ = handler.connection.Close("agentflow prompt receive failed")
			return prompt.Result{}, err
		}
		switch typedResponse := response.GetResponse().(type) {
		case *protos.StreamChatResponse_Chat:
			result, complete, err := prompt.StreamOutputToPromptResult(typedResponse.Chat, request.Transitions)
			if err != nil {
				return prompt.Result{}, err
			}
			if !complete {
				_ = communication.OnPacket(ctx, internal_type.LLMResponseDeltaPacket{ContextID: request.ContextID, Text: result.Text})
				continue
			}

			if result.TransitionID != "" || result.TransitionName != "" {
				return result, nil
			}
			_ = communication.OnPacket(ctx, internal_type.LLMResponseDonePacket{ContextID: request.ContextID, Text: result.Text})
			return result, nil
		case *protos.StreamChatResponse_Close:
			_ = handler.connection.Close("")
			return prompt.Result{}, fmt.Errorf("agentflow: model stream closed: %s", typedResponse.Close.GetReason())
		case *protos.StreamChatResponse_Configuration:
		}
	}
}

func (handler *AgentHandler) streamChatInput(
	communication internal_type.Communication,
	request prompt.Request,
	modelParameters utils.Option,
) *protos.StreamChatInput {
	modelOptions := make(utils.Option, len(modelParameters))
	for key, value := range utils.MergeMaps(modelParameters, communication.GetOptions()) {
		if key == "rapida.credential_id" || strings.HasPrefix(key, "connection.") {
			continue
		}
		modelOptions[key] = value
	}

	return &protos.StreamChatInput{
		RequestId:       request.ContextID,
		ProviderName:    handler.providerName,
		Conversations:   prompt.BuildPromptMessages(request),
		AdditionalData:  additionalData(communication, request),
		ModelParameters: handler.inputBuilder.Options(modelOptions, nil),
		ToolDefinitions: prompt.TransitionToolDefinitions(request.Transitions),
	}
}

func additionalData(communication internal_type.Communication, request prompt.Request) map[string]string {
	data := map[string]string{"agentflow_node_id": request.Node.ID}
	if conversation, err := communication.Conversation(); err == nil {
		data["assistant_id"] = fmt.Sprintf("%d", conversation.AssistantId)
		data["conversation_id"] = fmt.Sprintf("%d", conversation.Id)
		data["user_identifier"] = conversation.Identifier
	}
	data["message_id"] = request.ContextID
	return data
}
