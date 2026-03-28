// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_callers

import (
	"fmt"

	internal_anthropic_callers "github.com/rapidaai/api/integration-api/internal/caller/anthropic"
	internal_azure_callers "github.com/rapidaai/api/integration-api/internal/caller/azure"
	internal_cohere_callers "github.com/rapidaai/api/integration-api/internal/caller/cohere"
	internal_gemini_callers "github.com/rapidaai/api/integration-api/internal/caller/gemini"
	internal_huggingface_callers "github.com/rapidaai/api/integration-api/internal/caller/huggingface"
	internal_minimax_callers "github.com/rapidaai/api/integration-api/internal/caller/minimax"
	internal_mistral_callers "github.com/rapidaai/api/integration-api/internal/caller/mistral"
	internal_openai_callers "github.com/rapidaai/api/integration-api/internal/caller/openai"
	internal_replicate_callers "github.com/rapidaai/api/integration-api/internal/caller/replicate"
	internal_vertexai_callers "github.com/rapidaai/api/integration-api/internal/caller/vertexai"
	internal_voyageai_callers "github.com/rapidaai/api/integration-api/internal/caller/voyageai"
	internal_types "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type IntegrationProvider string

const (
	OPENAI      IntegrationProvider = "openai"
	ANTHROPIC   IntegrationProvider = "anthropic"
	GEMINI      IntegrationProvider = "gemini"
	VERTEXAI    IntegrationProvider = "vertexai"
	AZURE       IntegrationProvider = "azure-foundry"
	COHERE      IntegrationProvider = "cohere"
	MISTRAL     IntegrationProvider = "mistral"
	MINIMAX     IntegrationProvider = "minimax"
	REPLICATE   IntegrationProvider = "replicate"
	HUGGINGFACE IntegrationProvider = "huggingface"
	VOYAGEAI    IntegrationProvider = "voyageai"
)

func GetLargeLanguageCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.LargeLanguageCaller, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_callers.NewLargeLanguageCaller(logger, credential), nil
	case ANTHROPIC:
		return internal_anthropic_callers.NewLargeLanguageCaller(logger, credential), nil
	case GEMINI:
		return internal_gemini_callers.NewLargeLanguageCaller(logger, credential), nil
	case VERTEXAI:
		return internal_vertexai_callers.NewLargeLanguageCaller(logger, credential), nil
	case AZURE:
		return internal_azure_callers.NewLargeLanguageCaller(logger, credential), nil
	case COHERE:
		return internal_cohere_callers.NewLargeLanguageCaller(logger, credential), nil
	case MISTRAL:
		return internal_mistral_callers.NewLargeLanguageCaller(logger, credential), nil
	case MINIMAX:
		return internal_minimax_callers.NewLargeLanguageCaller(logger, credential), nil
	case REPLICATE:
		return internal_replicate_callers.NewLargeLanguageCaller(logger, credential), nil
	case HUGGINGFACE:
		return internal_huggingface_callers.NewLargeLanguageCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

func GetEmbeddingCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.EmbeddingCaller, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_callers.NewEmbeddingCaller(logger, credential), nil
	case GEMINI:
		return internal_gemini_callers.NewEmbeddingCaller(logger, credential), nil
	case VERTEXAI:
		return internal_vertexai_callers.NewEmbeddingCaller(logger, credential), nil
	case AZURE:
		return internal_azure_callers.NewEmbeddingCaller(logger, credential), nil
	case COHERE:
		return internal_cohere_callers.NewEmbeddingCaller(logger, credential), nil
	case MISTRAL:
		return internal_mistral_callers.NewEmbeddingCaller(logger, credential), nil
	case HUGGINGFACE:
		return internal_huggingface_callers.NewEmbeddingCaller(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewEmbeddingCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}

func GetRerankingCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.RerankingCaller, error) {
	switch IntegrationProvider(provider) {
	case COHERE:
		return internal_cohere_callers.NewRerankingCaller(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewRerankingCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported reranking provider: %s", provider)
	}
}

func GetVerifier(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.Verifier, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_callers.NewVerifyCredentialCaller(logger, credential), nil
	case ANTHROPIC:
		return internal_anthropic_callers.NewVerifyCredentialCaller(logger, credential), nil
	case GEMINI:
		return internal_gemini_callers.NewVerifyCredentialCaller(logger, credential), nil
	case VERTEXAI:
		return internal_vertexai_callers.NewVerifyCredentialCaller(logger, credential), nil
	case AZURE:
		return internal_azure_callers.NewVerifyCredentialCaller(logger, credential), nil
	case COHERE:
		return internal_cohere_callers.NewVerifyCredentialCaller(logger, credential), nil
	case MISTRAL:
		return internal_mistral_callers.NewVerifyCredentialCaller(logger, credential), nil
	case MINIMAX:
		return internal_minimax_callers.NewVerifyCredentialCaller(logger, credential), nil
	case REPLICATE:
		return internal_replicate_callers.NewVerifyCredentialCaller(logger, credential), nil
	case HUGGINGFACE:
		return internal_huggingface_callers.NewVerifyCredentialCaller(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewVerifyCredentialCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported provider for credential verification: %s", provider)
	}
}
