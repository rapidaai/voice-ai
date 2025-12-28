// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
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
