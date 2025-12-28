// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"
	"errors"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type assistantExecutor struct {
	logger   commons.Logger
	executor AssistantExecutor
}

// Init implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Initialize(ctx context.Context, communication internal_adapter_requests.Communication) error {
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
	return a.executor.Initialize(ctx, communication)
}

// Name implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Name() string {
	return a.executor.Name()
}

// Talk implements internal_executors.AssistantExecutor.
func (a *assistantExecutor) Talk(ctx context.Context, messageid string, msg *types.Message, communcation internal_adapter_requests.Communication) error {
	return a.executor.Talk(ctx, messageid, msg, communcation)
}

func (a *assistantExecutor) Close(
	ctx context.Context,
	communcation internal_adapter_requests.Communication,
) error {
	return a.executor.Close(ctx, communcation)
}

func NewAssistantExecutor(
	logger commons.Logger,
) AssistantExecutor {
	return &assistantExecutor{
		logger: logger,
	}
}
