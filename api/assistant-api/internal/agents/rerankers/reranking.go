package internal_agent_rerankers

import (
	"context"

	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

// Reranking is a generic interface that defines the contract for reranking operations.
// It is parameterized with type O, allowing for flexibility in the type of objects being reranked.
//
// The Rerank method is responsible for reordering a set of objects based on various criteria.
// It takes into account the following parameters:
// - ctx: The context for the reranking operation, which can be used for cancellation or timeout.
// - auth: A SimplePrinciple object representing the authentication credentials.
// - knowledgeCollection: A pointer to AssistantKnowledgeConfiguration, providing necessary knowledge for reranking.
// - s: A string parameter (purpose may vary depending on implementation).
// - in: An object of type O, representing the input to be reranked.
// - query: A string representing the query against which the reranking is performed.
//
// The method returns a slice of reranked objects of type O and an error if any occurs during the process.

type RerankingOption struct {
	ProviderCredential lexatic_backend.VaultCredential
	ModelProviderName  string
	ModelProviderId    uint64
	Options            map[string]interface{}
}
type Reranking[O any] interface {
	Rerank(ctx context.Context,
		auth types.SimplePrinciple,
		config *RerankingOption,
		in []O, query string, additionalData map[string]string) (map[int32]O, error)
}

type TextReranking interface {
	Reranking[string]
}
