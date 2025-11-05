package internal_callers

import (
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type RerankerOptions struct {
	AIOptions
}

func NewRerankerOptions(
	requestId uint64,
	irRequest *lexatic_backend.RerankingRequest,
	preHook func(rst map[string]interface{}),
	postHook func(rst map[string]interface{}, metrics types.Metrics),
) *RerankerOptions {
	cc := &RerankerOptions{
		AIOptions: AIOptions{
			RequestId:      requestId,
			PreHook:        preHook,
			PostHook:       postHook,
			ModelParameter: irRequest.GetModelParameters(),
		},
	}
	return cc
}
