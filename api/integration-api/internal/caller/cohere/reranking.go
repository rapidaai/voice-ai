package internal_cohere_callers

import (
	"context"
	"strings"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"

	lexatic_backend "github.com/rapidaai/protos"
)

type rerankingCaller struct {
	Cohere
}

func NewRerankingCaller(logger commons.Logger, credential *lexatic_backend.Credential) internal_callers.RerankingCaller {
	return &rerankingCaller{
		Cohere: NewCohere(logger, credential),
	}
}

func (rr *rerankingCaller) GetRerankRequest(opts *internal_callers.RerankerOptions) *cohere.RerankRequest {
	options := cohere.RerankRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = utils.Ptr(mn)
			}
		case "model.top_n":
			if topN, err := utils.AnyToInt(value); err == nil {
				options.TopN = utils.Ptr(topN)
			}
		case "model.max_chunks_per_doc":
			if mxChunk, err := utils.AnyToInt(value); err == nil {
				options.MaxChunksPerDoc = utils.Ptr(mxChunk)
			}
		case "model.rank_fields":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.RankFields = strings.Split(stopStr, ",")
			}
		}
	}
	return &options
}

func (rr *rerankingCaller) GetReranking(ctx context.Context,
	// providerModel string,
	query string,
	content map[int32]*lexatic_backend.Content,
	options *internal_callers.RerankerOptions,
) ([]*lexatic_backend.Reranking, types.Metrics, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := rr.GetClient()
	if err != nil {
		rr.logger.Errorf("chat reranker unable to get client for cohere %v", err)
		return nil, metrics.OnFailure().Build(), err
	}
	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// preseving the order

	input := make([]*cohere.RerankRequestDocumentsItem, len(content))
	for k, v := range content {
		input[k] = &cohere.RerankRequestDocumentsItem{
			String: string(v.Content),
			RerankDocument: map[string]string{
				"Description": string(v.Content),
			},
		}
	}

	rerankRequest := rr.GetRerankRequest(options)
	rerankRequest.Query = query
	rerankRequest.Documents = input

	options.AIOptions.PreHook(utils.ToJson(rerankRequest))
	resp, err := client.Rerank(
		ctx,
		rerankRequest,
	)

	if err != nil {
		options.AIOptions.PostHook(nil, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()
	output := make([]*lexatic_backend.Reranking, len(resp.Results))

	for _, rerankedData := range resp.Results {
		// preserve the index of the chunk
		output[rerankedData.Index] = &lexatic_backend.Reranking{
			Index:          int32(rerankedData.Index),
			Content:        content[int32(rerankedData.Index)],
			RelevanceScore: rerankedData.RelevanceScore,
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.Build())
	return output, metrics.Build(), nil
}
