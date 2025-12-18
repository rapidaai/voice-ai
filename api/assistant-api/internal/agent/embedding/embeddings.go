package internal_agent_embedding

import (
	"context"

	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type TextEmbeddingOption struct {
	ProviderCredential *protos.VaultCredential
	ModelProviderName  string
	Options            map[string]interface{}
	AdditionalData     map[string]string
}

type QueryEmbedding interface {
	TextQueryEmbedding(
		ctx context.Context,
		auth types.SimplePrinciple,
		query string,
		opts *TextEmbeddingOption) (*protos.EmbeddingResponse, error)
}
