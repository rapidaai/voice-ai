package internal_callers

import (
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type EmbeddingOptions struct {
	AIOptions
}

func NewEmbeddingOptions(
	requestId uint64,
	irRequest *lexatic_backend.EmbeddingRequest,
	preHook func(rst map[string]interface{}),
	postHook func(rst map[string]interface{}, metrics types.Metrics),
) *EmbeddingOptions {
	cc := &EmbeddingOptions{
		AIOptions: AIOptions{
			RequestId:      requestId,
			PreHook:        preHook,
			PostHook:       postHook,
			ModelParameter: irRequest.GetModelParameters(),
		},
	}
	return cc
}
