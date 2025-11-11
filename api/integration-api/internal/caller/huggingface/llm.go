package internal_huggingface_callers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	Huggingface
}

func NewLargeLanguageCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Huggingface: huggingface(logger,
			DEFUALT_URL, credential),
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
		// "model":      providerModel,
		"max_tokens": 1024,
	}

	msg := make([]map[string]interface{}, 0)
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
			contents := make([]map[string]interface{}, 0)
			for _, c := range cntn.Contents {
				if c.GetContentType() == commons.TEXT_CONTENT.String() {
					contents = append(contents, map[string]interface{}{
						"type": "text",
						"text": string(c.GetContent()),
					})
				}
			}
			// txt := cntn.Contents[0].GetContent()
			msg = append(msg, map[string]interface{}{
				"role":    "user",
				"content": contents,
			})
			lastRole = "user"
		}

		if currentRole == "assistant" {
			if lastRole == "assistant" {
				// Skip this message to ensure alternation
				continue
			}
			txt := cntn.Contents[0].GetContent()
			msg = append(msg, map[string]interface{}{
				"role":    "assistant",
				"content": string(txt),
			})
			lastRole = "assistant"
		}
	}

	requestBody["messages"] = msg
	headers := map[string]string{}
	options.AIOptions.PreHook(requestBody)
	res, err := llc.Call(ctx, fmt.Sprintf("models/%s"), // providerModel,
		"POST", headers, requestBody)

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
	var resp HuggingfaceInferenceResponse
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
	output := make([]*types.Content, 0)
	// metrics.OnAddMetrics(llc.UsageMetrics(resp.Usage)...)

	output = append(output, &types.Content{
		ContentType:   commons.TEXT_CONTENT.String(),
		ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
		Content:       []byte(resp.Text),
	})
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

	var buffer bytes.Buffer
	buffer.WriteString("\n\nHuman: %s\n\nAssistant:")
	requestBody := map[string]interface{}{
		"prompt":               fmt.Sprintf(buffer.String(), prompts[0]),
		"model":                providerModel,
		"max_tokens_to_sample": 100,
	}

	headers := map[string]string{}
	options.AIOptions.PreHook(requestBody)
	res, err := llc.Call(ctx, fmt.Sprintf("models/%s", providerModel), "POST", headers, requestBody)
	//
	//
	if err != nil {
		llc.logger.Errorf("getting error for completion %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()
	var resp HuggingfaceInferenceResponse
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		llc.logger.Errorf("error while parsing complition response %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	options.AIOptions.PostHook(map[string]interface{}{
		"result": res,
	}, metrics.Build())
	return []*types.Content{{
		ContentType:   commons.TEXT_CONTENT.String(),
		ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
		Content:       []byte(resp.Text),
	}}, metrics.Build(), nil
}
