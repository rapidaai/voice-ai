package internal_voyageai_callers

import (
	"context"
	"encoding/json"
	"fmt"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type embeddingCaller struct {
	Voyageai
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		Voyageai: voyageai(logger, credential),
	}
}

func (ec *embeddingCaller) GetEmbedRequest(opts *internal_callers.EmbeddingOptions) map[string]interface{} {
	options := map[string]interface{}{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options["model"] = modelName
			}
		case "model.encoding_format":
			if encodingFormat, err := utils.AnyToString(value); err == nil {
				options["encoding_format"] = encodingFormat
			}
		case "model.output_dimension":
			if dimensions, err := utils.AnyToInt(value); err == nil {
				options["output_dimension"] = dimensions
			}
		case "input_type":
			if inputType, err := utils.AnyToString(value); err == nil {
				options["input_type"] = inputType
			}
		case "truncation":
			if truncation, err := utils.AnyToBool(value); err == nil {
				options["truncation"] = truncation
			}
		case "output_dimension":
			if outputDimension, err := utils.AnyToInt(value); err == nil {
				options["output_dimension"] = outputDimension
			}
		case "output_dtype":
			if outputDtype, err := utils.AnyToString(value); err == nil {
				options["output_dtype"] = outputDtype
			}
		case "encoding_format":
			if encodingFormat, err := utils.AnyToString(value); err == nil {
				options["encoding_format"] = encodingFormat
			}
		}
	}
	return options
}

// GetText2Speech implements internal_callers.Text2SpeechCaller.
func (ec *embeddingCaller) GetEmbedding(ctx context.Context,
	content map[int32]string,
	options *internal_callers.EmbeddingOptions) ([]*integration_api.Embedding, types.Metrics, error) {

	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	// preseving the order

	input := make([]string, len(content))
	for k, v := range content {
		input[k] = v
	}

	request := map[string]interface{}{
		"input": input,
		// "model": providerModel,
	}

	headers := map[string]string{}
	options.AIOptions.PreHook(request)
	res, err := ec.Call(ctx, "embeddings", "POST", headers, request)

	//
	if err != nil {
		ec.logger.Errorf("getting error for chat complition %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()

	var resp VoyageaiEmbeddingResponse
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		ec.logger.Errorf("error while parsing embedding response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	if resp.Usage != nil {
		metrics.OnAddMetrics(&types.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", resp.Usage.TotalTokens),
			Description: "Total Token",
		})

	}
	output := make([]*integration_api.Embedding, len(resp.Data))
	for _, embeddingData := range resp.Data {
		// preserve the index of the chunk
		output[embeddingData.Index] = &integration_api.Embedding{
			Index:     int32(embeddingData.Index),
			Embedding: utils.EmbeddingToFloat64(embeddingData.Embedding),
			Base64:    utils.EmbeddingToBase64(utils.EmbeddingToFloat64(embeddingData.Embedding)),
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": res,
	}, metrics.Build())
	return output, metrics.Build(), nil
}
