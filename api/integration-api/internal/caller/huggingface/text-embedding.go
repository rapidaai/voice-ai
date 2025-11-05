package internal_huggingface_callers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type embeddingCaller struct {
	Huggingface
}

func NewEmbeddingCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.EmbeddingCaller {
	return &embeddingCaller{
		Huggingface: huggingface(logger,
			DEFUALT_URL,
			credential),
	}
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
	// options.AIOptions.PreHook(ec.toString(request))
	res, err := ec.Call(ctx, fmt.Sprintf("pipeline/%s/%s",
		"embedding",
	// providerModel
	), "POST", headers, request)

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

	var resp [][]float32
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		ec.logger.Errorf("error while parsing embedding response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	output := make([]*integration_api.Embedding, len(resp))

	for ix, embeddingData := range resp {
		// preserve the index of the chunk
		output[ix] = &integration_api.Embedding{
			Index:     int32(ix),
			Embedding: utils.EmbeddingToFloat64(embeddingData),
			Base64:    utils.EmbeddingToBase64(utils.EmbeddingToFloat64(embeddingData)),
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

func embeddingToBase64(embedding []float32) string {
	byteArray, err := float32SliceToByteArray(embedding)
	if err != nil {
		return ""
	}
	base64Str := base64.StdEncoding.EncodeToString(byteArray)
	return base64Str
}
