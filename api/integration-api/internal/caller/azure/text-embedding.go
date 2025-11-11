package internal_azure_callers

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type embeddingCaller struct {
	AzureAi
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		AzureAi: azure(logger, credential),
	}
}

func (ec *embeddingCaller) GetEmbeddingNewParams(opts *internal_callers.EmbeddingOptions) openai.EmbeddingNewParams {
	options := openai.EmbeddingNewParams{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = modelName
			}
		case "model.user":
			if user, err := utils.AnyToString(value); err == nil {
				options.User = openai.String(user)
			}
		case "model.encoding_format":
			if re, err := utils.AnyToString(value); err == nil {
				options.EncodingFormat = openai.EmbeddingNewParamsEncodingFormat(re)
			}
		case "model.dimensions":
			if dimensions, err := utils.AnyToInt64(value); err == nil {
				options.Dimensions = openai.Int(dimensions)
			}

		}
	}
	return options

}

// GetText2Speech implements internal_callers.Text2SpeechCaller.
func (ec *embeddingCaller) GetEmbedding(ctx context.Context,
	// providerModel string,
	content map[int32]string,
	options *internal_callers.EmbeddingOptions) ([]*integration_api.Embedding, types.Metrics, error) {

	mertics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	mertics.OnStart()

	client, err := ec.GetClient()
	if err != nil {
		return nil, mertics.OnFailure().Build(), err
	}

	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// preseving the order

	input := make([]string, len(content))
	for k, v := range content {
		input[k] = v
	}

	opts := ec.GetEmbeddingNewParams(options)
	opts.Input = openai.EmbeddingNewParamsInputUnion{
		OfArrayOfStrings: input,
	}

	options.AIOptions.PreHook(map[string]interface{}{"input": opts})
	resp, err := client.Embeddings.New(
		ctx,
		opts,
	)

	if err != nil {
		options.AIOptions.PostHook(map[string]interface{}{
			"result": resp,
			"error":  err,
		}, mertics.OnFailure().Build())
		return nil, mertics.Build(), err
	}
	mertics.OnSuccess()
	output := make([]*integration_api.Embedding, len(resp.Data))

	// all the usages into the metrics
	mertics.OnAddMetrics(&types.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", resp.Usage.PromptTokens),
		Description: "Input token",
	}, &types.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", resp.Usage.TotalTokens),
		Description: "Total Token",
	})

	for _, embeddingData := range resp.Data {
		// preserve the index of the chunk
		output[embeddingData.Index] = &integration_api.Embedding{
			Index:     int32(embeddingData.Index),
			Embedding: embeddingData.Embedding,
			// Base64:    embeddingData.EmbeddingBase64,
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
	}, mertics.Build())
	return output, mertics.Build(), nil
}
