package internal_mistral_callers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type embeddingCaller struct {
	Mistral
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		Mistral: mistral(logger, credential),
	}
}

// GetText2Speech implements internal_callers.Text2SpeechCaller.
func (ec *embeddingCaller) GetEmbedding(ctx context.Context,
	// providerModel string,
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
			"error":  err,
			"result": res,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()

	var resp MistralEmbeddingResponse
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		ec.logger.Errorf("error while parsing embedding response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": res,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	output := make([]*integration_api.Embedding, len(resp.Data[len(resp.Data)-1]))

	for _, embeddingData := range resp.Data[len(resp.Data)-1] {
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

// Convert a slice of float32 to a byte array
func float32SliceToByteArray(data []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
