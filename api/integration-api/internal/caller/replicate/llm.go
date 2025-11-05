package internal_replicate_callers

import (
	"context"
	"strings"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
	replicate_go "github.com/replicate/replicate-go"
)

type largeLanguageCaller struct {
	Replicate
}

func NewLargeLanguageCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Replicate: replicate(logger, credential),
	}
}

// StreamChatCompletion implements internal_callers.LargeLanguageCaller.
func (*largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*lexatic_backend.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(types.Message) error,
	onMetrics func(*types.Message, types.Metrics) error,
	onError func(err error),
) error {
	panic("unimplemented")
}

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*lexatic_backend.Message,
	options *internal_callers.ChatCompletionOptions,
) (*types.Message, types.Metrics, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("complition unable to get client for cohere %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	input := replicate_go.PredictionInput{}

	options.AIOptions.PreHook(utils.ToJson(input))
	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	prediction, err := client.CreatePrediction(ctx,
		// *options.Version,
		"",
		input,
		nil,
		false)
	if err != nil {
		metrics.OnFailure()
		llc.logger.Errorf("unable to create replicate prediction %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": prediction,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}
	err = client.Wait(ctx, prediction) // Wait for the prediction to finish
	if err != nil {
		metrics.OnFailure()
		llc.logger.Errorf("after waiting prediction failed to response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": prediction,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	// all the usages into the metrics
	metrics.OnAddMetrics(llc.UsageMetrics(prediction.Metrics)...)
	v, ok := prediction.Output.([]string)
	if !ok {
		metrics.OnFailure()
		llc.logger.Errorf("response is not string %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": prediction,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()

	// options.AIOptions.PreHook(llc.toString(response))
	options.AIOptions.PostHook(map[string]interface{}{
		"result": prediction,
	}, metrics.Build())

	return &types.Message{
		Role: "chatbot",
		Contents: []*types.Content{{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(strings.Join(v, "")),
		}},
	}, metrics.Build(), nil
}
