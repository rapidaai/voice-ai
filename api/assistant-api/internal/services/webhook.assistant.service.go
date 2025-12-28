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
)

type AssistantWebhookService interface {
	Get(ctx context.Context, auth types.SimplePrinciple, WebhookId uint64, assistantId uint64) (*internal_assistant_entity.AssistantWebhook, error)
	Delete(ctx context.Context, auth types.SimplePrinciple, WebhookId uint64, assistantId uint64) (*internal_assistant_entity.AssistantWebhook, error)
	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantEvents []string,
		timeoutSecond uint32,
		httpMethod string,
		httpUrl string,
		httpHeaders, httpBody map[string]string,
		retryStatusCodes []string,
		retryCount, executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantWebhook, error)
	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		webhookId uint64,
		assistantEvents []string,
		timeoutSecond uint32,
		httpMethod string,
		httpUrl string,
		httpHeaders, httpBody map[string]string,
		retryStatusCodes []string,
		maxRetryCount, executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantWebhook, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantWebhook, error)

	CreateLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		webhookId uint64,
		assistantId, conversationId uint64,
		httpUrl, httpMethod string,
		event string,
		responseStatus int64,
		timeTaken int64,
		retryCount uint32,
		status type_enums.RecordState,
		request, response []byte,
	) (*internal_assistant_entity.AssistantWebhookLog, error)

	GetAllLog(ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
		order *protos.Ordering,
	) (int64, []*internal_assistant_entity.AssistantWebhookLog, error)

	GetLog(ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		webhookLogId uint64) (*internal_assistant_entity.AssistantWebhookLog, error)
	GetLogObject(
		ctx context.Context,
		organizationId,
		projectId, webhookLogId uint64) (requestData []byte, responseData []byte, err error)
}
