// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	protos "github.com/rapidaai/protos"
	workflow_api "github.com/rapidaai/protos"
)

type AssistantToolService interface {
	Get(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantToolId uint64,
		assistantId uint64) (*internal_assistant_entity.AssistantTool, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantTool, error)

	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name string,
		description string,
		fields map[string]interface{},
		executionMethod string,
		executionOptions []*protos.Metadata,
	) (*internal_assistant_entity.AssistantTool, error)

	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantToolId uint64,
		assistantId uint64,
		name string,
		description string,
		fields map[string]interface{},
		executionMethod string,
		executionOptions []*protos.Metadata,
	) (*internal_assistant_entity.AssistantTool, error)

	Delete(ctx context.Context,
		auth types.SimplePrinciple,
		toolId uint64,
		assistantId uint64) (*internal_assistant_entity.AssistantTool, error)

	CreateLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId, conversationId uint64,
		toolId uint64,
		messageId string,
		toolName string,
		timeTaken int64,
		executionMethod string,
		status type_enums.RecordState,
		request, response []byte,
	) (*internal_assistant_entity.AssistantToolLog, error)

	GetLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		toolLogId uint64) (*internal_assistant_entity.AssistantToolLog, error)

	GetAllLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
		order *protos.Ordering) (int64, []*internal_assistant_entity.AssistantToolLog, error)

	GetLogObject(
		ctx context.Context,
		organizationId,
		projectId, toolLogId uint64) (requestData []byte, responseData []byte, err error)
}
