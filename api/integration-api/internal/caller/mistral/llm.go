package internal_mistral_callers

import (
	"context"
	"encoding/json"
	"errors"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	Mistral
}

func NewLargeLanguageCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Mistral: mistral(logger, credential),
	}
}

// StreamChatCompletion implements internal_callers.LargeLanguageCaller.
func (*largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	// providerModel string,
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
	// providerModel string,
	allMessages []*lexatic_backend.Message,
	options *internal_callers.ChatCompletionOptions,
) (*types.Message, types.Metrics, error) {
	llc.logger.Debugf("getting chat completion from google llc")
	//
	// Working with chat complition with vision
	//
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	requestBody := map[string]interface{}{
		// "model": providerModel,
	}

	msg := make([]map[string]string, 0)
	var lastRole string

	for _, cntn := range allMessages {
		if len(cntn.GetContents()) == 0 {
			// there might be problem in initiator
			continue
		}

		currentRole := cntn.GetRole()
		if currentRole == "user" || currentRole == "system" {
			if lastRole == "user" {
				// Skip this message to ensure alternation
				continue
			}

			txt := cntn.Contents[0].GetContent()
			msg = append(msg, map[string]string{
				"role":    "user",
				"content": string(txt),
			})
			lastRole = "user"
		}

		if currentRole == "assistant" {
			if lastRole == "assistant" {
				// Skip this message to ensure alternation
				continue
			}
			txt := cntn.Contents[0].GetContent()
			msg = append(msg, map[string]string{
				"role":    "assistant",
				"content": string(txt),
			})
			lastRole = "assistant"
		}
	}

	requestBody["messages"] = msg
	headers := map[string]string{}
	options.AIOptions.PreHook(requestBody)
	res, err := llc.Call(ctx, "chat/completions", "POST", headers, requestBody)

	//
	if err != nil {
		llc.logger.Errorf("getting error for chat complition %v", err)

		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()
	var resp MistralMessageResponse
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		llc.logger.Errorf("error while parsing chat complition response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	//
	//
	output := make([]*types.Content, len(resp.Choices))
	metrics.OnAddMetrics(llc.UsageMetrics(resp.Usage)...)

	for idx, choice := range resp.Choices {
		output[idx] = &types.Content{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(choice.Message.Content),
		}

	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": res,
	}, metrics.Build())
	return &types.Message{
		Contents: output,
	}, metrics.Build(), nil
}

func (llc *largeLanguageCaller) GetCompletion(
	ctx context.Context,
	providerModel string,
	prompts []string,
	options *internal_callers.CompletionOptions,
) ([]*types.Content, types.Metrics, error) {

	//
	// Working with chat complition with vision
	//
	llc.logger.Debugf("getting for completion for anthropic")
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	return nil, metrics.OnFailure().Build(), errors.New("illegal implimentation")

}
