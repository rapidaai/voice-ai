package internal_assistant_executors

import (
	"context"
	"errors"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_executors "github.com/rapidaai/api/assistant-api/internal/executors"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type assistantExecutor struct {
	logger   commons.Logger
	executor internal_executors.AssistantExecutor
}

// Init implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Init(ctx context.Context, communication internal_adapter_requests.Communication) error {
	switch communication.Assistant().AssistantProvider {
	case type_enums.AGENTKIT:
		a.executor = NewAgentKitAssistantExecutor(a.logger)
	case type_enums.WEBSOCKET:
		a.executor = NewWebsocketAssistantExecutor(a.logger)
	case type_enums.MODEL:
		a.executor = NewModelAssistantExecutor(a.logger)
	default:
		return errors.New("illegal assistant executor")
	}
	return a.executor.Init(ctx, communication)
}

// Name implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Name() string {
	return a.executor.Name()
}

// Talk implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Talk(ctx context.Context, messageid string, msg *types.Message, communcation internal_adapter_requests.Communication) error {
	return a.executor.Talk(ctx, messageid, msg, communcation)
}

func (a *assistantExecutor) Connect(
	ctx context.Context,
	assistantId uint64,
	assistantConversationId uint64,
) error {
	return a.executor.Connect(ctx, assistantId, assistantConversationId)
}

func (a *assistantExecutor) Disconnect(
	ctx context.Context,
	assistantId uint64,
	assistantConversationId uint64,
) error {
	return a.executor.Disconnect(ctx, assistantId, assistantConversationId)
}

func NewAssistantExecutor(
	logger commons.Logger,
) internal_executors.AssistantExecutor {
	return &assistantExecutor{
		logger: logger,
	}
}
