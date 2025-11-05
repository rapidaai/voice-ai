package internal_services

import (
	"context"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type AssistantAnalysisService interface {
	Get(ctx context.Context, auth types.SimplePrinciple, analysisId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantAnalysis, error)
	Delete(ctx context.Context, auth types.SimplePrinciple, analysisId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantAnalysis, error)
	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name string,
		endpointId uint64,
		endpointVersion string,
		endpointParameters map[string]string,
		executionPriority uint32,
		description *string,
	) (*internal_assistant_gorm.AssistantAnalysis, error)
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
	) (*internal_assistant_gorm.AssistantAnalysis, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*lexatic_backend.Criteria,
		paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantAnalysis, error)
}
