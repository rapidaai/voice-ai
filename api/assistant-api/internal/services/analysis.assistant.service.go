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
	protos "github.com/rapidaai/protos"
)

type AssistantAnalysisService interface {
	Get(ctx context.Context, auth types.SimplePrinciple, analysisId uint64, assistantId uint64) (*internal_assistant_entity.AssistantAnalysis, error)
	Delete(ctx context.Context, auth types.SimplePrinciple, analysisId uint64, assistantId uint64) (*internal_assistant_entity.AssistantAnalysis, error)
	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name string,
		endpointId uint64,
		endpointVersion string,
		endpointParameters map[string]string,
		executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantAnalysis, error)
	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		analysisId uint64,
		name string,
		endpointId uint64,
		endpointVersion string,
		endpointParameters map[string]string,
		executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantAnalysis, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantAnalysis, error)
}
