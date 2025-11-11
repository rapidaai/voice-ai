package internal_adapter_requests

import (
	"context"

	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type KnowledgeRetriveOption struct {
	EmbeddingProviderCredential *lexatic_backend.VaultCredential
	RetrievalMethod             string
	TopK                        uint32
	ScoreThreshold              float32
}

type KnowledgeContextResult struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
}

type Talking interface {
	Communication
	Connect(
		ctx context.Context,
		auth types.SimplePrinciple,
		identifier string,
		connectionConfig *lexatic_backend.AssistantConversationConfiguration) error
	Talk(
		ctx context.Context,
		auth types.SimplePrinciple,
		identifier string,
	) error
}
