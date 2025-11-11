package internal_cohere_callers

import (
	"context"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type embeddingCaller struct {
	Cohere
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		Cohere: NewCohere(logger, credential),
	}
}
func (ec *embeddingCaller) GetEmbedRequest(opts *internal_callers.EmbeddingOptions) *cohere.V2EmbedRequest {
	options := &cohere.V2EmbedRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = modelName
			}
		case "model.input_type":
			if inputType, err := utils.AnyToString(value); err == nil {
				options.InputType = cohere.EmbedInputType(inputType)
			}
		case "model.dimensions":
			if dimensions, err := utils.AnyToInt(value); err == nil {
				options.OutputDimension = cohere.Int(dimensions)
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
	//
	// Working with chat complition with vision
	//
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := ec.GetClient()
	if err != nil {
		return nil, metrics.OnFailure().Build(), err
	}

	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// preseving the order

	input := make([]string, len(content))
	for k, v := range content {
		input[k] = v
	}

	// &cohere.V2EmbedRequest{
	// 		Texts: input,
	// 		// Model: &providerModel,
	// 	},
	request := ec.GetEmbedRequest(options)
	request.Texts = input
	resp, err := client.V2.Embed(
		ctx,
		request,
	)

	if err != nil {
		options.AIOptions.PostHook(
			map[string]interface{}{
				"result": resp,
				"error":  err,
			},
			metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	output := make([]*integration_api.Embedding, len(input))
	// all the usages into the metrics

	for idx, embeddingData := range resp.GetEmbeddings().GetFloat() {
		// preserve the index of the chunk
		output[idx] = &integration_api.Embedding{
			Index:     int32(idx),
			Embedding: utils.EmbeddingToFloat64(embeddingData),
			Base64:    utils.EmbeddingToBase64(embeddingData),
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.Build())
	return output, metrics.Build(), nil
}
