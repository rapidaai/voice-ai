package internal_services

import (
	"context"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	workflow_api "github.com/rapidaai/protos"
)

type AssistantKnowledgeService interface {
	Get(ctx context.Context, auth types.SimplePrinciple,
		assistantKnowledgeId, assistantId uint64) (*internal_assistant_gorm.AssistantKnowledge, error)
	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_gorm.AssistantKnowledge, error)

	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		knowledgeId uint64,
		retrievalMethod gorm_types.RetrievalMethod,
		rerankEnabled bool,
		scoreThreshold float32,
		topK uint32,
		rerankerProviderModelId *uint64,
		rerankerProviderModelName *string,
		rerankerProviderModelOptions []*workflow_api.Metadata,
	) (*internal_assistant_gorm.AssistantKnowledge, error)

	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantKnowledgeId uint64,
		assistantId uint64,
		knowledgeId uint64,
		retrievalMethod gorm_types.RetrievalMethod,
		rerankEnabled bool,
		scoreThreshold float32,
		topK uint32,
		rerankerProviderModelId *uint64,
		rerankerProviderModelName *string,
		rerankerProviderModelOptions []*workflow_api.Metadata) (*internal_assistant_gorm.AssistantKnowledge, error)

	Delete(ctx context.Context,
		auth types.SimplePrinciple,
		assistantKnowledgeId, assistantId uint64) (*internal_assistant_gorm.AssistantKnowledge, error)
}
