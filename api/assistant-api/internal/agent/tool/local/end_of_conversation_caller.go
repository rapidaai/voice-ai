// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_local_tool

import (
	"context"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type endOfConversationCaller struct {
	toolCaller
}

func (afkTool *endOfConversationCaller) Call(
	ctx context.Context,
	messageId string,
	args string,
	communication internal_adapter_requests.Communication,
) (map[string]interface{}, []*types.Metric) {
	start := time.Now()
	metrics := make([]*types.Metric, 0)
	err := communication.
		Notify(
			ctx,
			&protos.AssistantMessagingResponse_Action{
				Action: &protos.AssistantConversationAction{
					Name:   afkTool.Name(),
					Action: protos.AssistantConversationAction_END_CONVERSATION,
				},
			},
		)
	metrics = append(metrics, types.NewTimeTakenMetric(time.Since(start)))
	if err != nil {
		return afkTool.Result("Unable to disconnect. Please try again later.", false), metrics
	}
	return afkTool.Result("Disconnected successfully.", true), metrics
}

func NewEndOfConversationCaller(
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communcation internal_adapter_requests.Communication,
) (ToolCaller, error) {
	return &endOfConversationCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
	}, nil
}
