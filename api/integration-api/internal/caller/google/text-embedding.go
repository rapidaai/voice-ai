package internal_google_callers

import (
	"context"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/genai"
)

type embeddingCaller struct {
	Google
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		Google: google(logger, credential),
	}
}

func (ec *embeddingCaller) GetEmbedContentConfig(opts *internal_callers.EmbeddingOptions) (model string, cfg *genai.EmbedContentConfig) {
	cfg = &genai.EmbedContentConfig{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				model = modelName
			}
		case "model.output_dimensionality":
			if dimensions, err := utils.AnyToInt32(value); err == nil {
				cfg.OutputDimensionality = utils.Ptr(dimensions)
			}

		}
	}
	return
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
		ec.logger.Errorf("getting error for chat completion %v", err)
		metrics.OnFailure()
		options.AIOptions.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		return nil, metrics.Build(), err
	}

	options.AIOptions.PreHook(map[string]interface{}{
		"request": content,
	})
	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	output := make([]*integration_api.Embedding, len(content))
	mdl, cfg := ec.GetEmbedContentConfig(options)
	contents := make([]*genai.Content, len(content))
	for idx, st := range content {
		contents[idx] = genai.NewContentFromText(st, "user")
	}

	resp, err := client.Models.EmbedContent(ctx, mdl, contents, cfg)
	if err != nil {
		ec.logger.Errorf("failed to unmarshal", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": resp,
			"error":  err,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}

	for ix, v := range resp.Embeddings {
		output[ix] = &integration_api.Embedding{
			Index:     int32(ix),
			Embedding: utils.EmbeddingToFloat64(v.Values),
			Base64:    utils.EmbeddingToBase64(utils.EmbeddingToFloat64(v.Values)),
		}
	}

	metrics.OnSuccess()
	options.AIOptions.PostHook(map[string]interface{}{"result": output}, metrics.Build())
	return output, metrics.Build(), nil
}
