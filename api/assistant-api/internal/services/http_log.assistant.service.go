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
	"github.com/rapidaai/protos"
)

// AssistantHTTPLogService persists generic HTTP logs.
type AssistantHTTPLogService interface {
	CreateLog(
		ctx context.Context,
		auth *types.Authentication,
		source string,
		sourceRefId uint64,
		sourceEvent string,
		contextID string,
		assistantId uint64,
		assistantConversationId *uint64,
		httpUrl string,
		httpMethod string,
		responseStatus int64,
		timeTaken int64,
		retryCount uint32,
		status type_enums.RecordState,
		errorMessage *string,
		request []byte,
		response []byte,
	) (*internal_assistant_entity.AssistantHTTPLog, error)

	GetLog(
		ctx context.Context,
		auth *types.Authentication,
		projectId uint64,
		httpLogId uint64,
	) (*internal_assistant_entity.AssistantHTTPLog, error)

	GetAllLog(
		ctx context.Context,
		auth *types.Authentication,
		projectId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
		order *protos.Ordering,
	) (int64, []*internal_assistant_entity.AssistantHTTPLog, error)

	GetLogObject(
		ctx context.Context,
		auth *types.Authentication,
		httpLogId uint64,
	) (requestData []byte, responseData []byte, err error)

	RetryLog(
		ctx context.Context,
		auth *types.Authentication,
		projectId uint64,
		httpLogId uint64,
	) (*internal_assistant_entity.AssistantHTTPLog, error)
}
