package internal_agent_embeddings

import (
	"context"

	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type TextEmbeddingOption struct {
	ProviderCredential *lexatic_backend.VaultCredential
	ModelProviderName  string
	ModelProviderId    uint64
	Options            map[string]interface{}
	AdditionalData     map[string]string
}

type QueryEmbedding interface {
	TextQueryEmbedding(
		ctx context.Context,
		auth types.SimplePrinciple,
		query string,
		opts *TextEmbeddingOption) (*lexatic_backend.EmbeddingResponse, error)
}
