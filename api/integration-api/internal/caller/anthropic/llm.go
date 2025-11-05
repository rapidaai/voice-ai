package internal_anthropic_callers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	Anthropic
}

func NewLargeLanguageCaller(logger commons.Logger, credential *lexatic_backend.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Anthropic: anthropicAI(logger, credential),
	}
}

func (llc *largeLanguageCaller) BuildHistory(allMessages []*lexatic_backend.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	messages := make([]anthropic.MessageParam, 0)
	systemPrompt := make([]anthropic.TextBlockParam, 0)
	for _, msg := range allMessages {
		switch msg.GetRole() {
		case "assistant":
			// message :=
			mConnect := make([]anthropic.ContentBlockParamUnion, 0)
			for _, c := range msg.GetContents() {
				if c.GetContentType() == commons.TEXT_CONTENT.String() {
					mConnect = append(mConnect, anthropic.ContentBlockParamUnion{
						OfText: &anthropic.TextBlockParam{
							Text: string(c.GetContent()),
						},
					})
				}
			}
			for _, tc := range msg.GetToolCalls() {
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(tc.GetFunction().GetArguments()), &input); err != nil {
					llc.logger.Warnf("Invalid JSON in tool call arguments: %v", err)
					continue
				}
				mConnect = append(mConnect, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.GetId(),
						Name:  tc.GetFunction().GetName(),
						Input: input,
					},
				})
			}
			if len(mConnect) > 0 {
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: mConnect,
				})
			}
		case "user":
			uContent := make([]anthropic.ContentBlockParamUnion, 0)
			for _, c := range msg.GetContents() {
				if c.GetContentType() == commons.TEXT_CONTENT.String() {
					txtContent := string(c.GetContent())
					// ignore emty block
					if strings.TrimSpace(txtContent) != "" {
						uContent = append(uContent, anthropic.ContentBlockParamUnion{
							OfText: &anthropic.TextBlockParam{
								Text: string(c.GetContent()),
							},
						})
					}

				}
			}
			if len(uContent) > 0 {
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: uContent,
				})
			}
		case "tool":
			tContent := make([]anthropic.ContentBlockParamUnion, 0)
			for _, c := range msg.GetContents() {
				tContent = append(tContent, anthropic.ContentBlockParamUnion{
					OfToolResult: &anthropic.ToolResultBlockParam{
						ToolUseID: c.GetContentType(),
						Content: []anthropic.ToolResultBlockParamContentUnion{{
							OfText: &anthropic.TextBlockParam{
								Text: string(c.GetContent()),
							},
						}},
					},
				})
			}
			if len(tContent) > 0 {
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: tContent,
				})
			}
		case "system":
			for _, c := range msg.GetContents() {
				if c.GetContentType() == commons.TEXT_CONTENT.String() {
					systemPrompt = append(systemPrompt, anthropic.TextBlockParam{
						Text: string(c.GetContent()),
					})
				}
			}
		}

	}
	return systemPrompt, messages
}
func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*lexatic_backend.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(types.Message) error,
	onMetrics func(*types.Message, types.Metrics) error,
	onError func(err error),
) error {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	instruction, messages := llc.BuildHistory(allMessages)
	params := llc.GetMessageNewParams(options)
	params.Messages = messages
	params.System = instruction

	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for anthropic: %v", err)
		onError(err)
		onMetrics(nil, metrics.OnFailure().Build())
		options.AIOptions.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return err
	}
	options.AIOptions.PreHook(utils.ToJson(params))
	stream := client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}
	if stream.Err() != nil {
		llc.logger.Errorf("stream error: %v", stream.Err())
		options.AIOptions.PostHook(map[string]interface{}{
			"result": utils.ToJson(message),
			"error":  stream.Err(),
		}, metrics.Build())
		onMetrics(nil, metrics.OnFailure().Build())
		onError(stream.Err())
		return stream.Err()
	}

	completeMessage := types.Message{
		Role: "assistant",
	}
	var currentToolCall *types.ToolCall
	var currentContent *types.Content
	isToolCall := false
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			onError(err)
			continue
		}

		switch event := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			llc.logger.Debugf("ContentBlockStartEvent %+v", event.JSON)
			switch event.ContentBlock.Type {
			case "tool_use":
				isToolCall = true
				currentToolCall = &types.ToolCall{
					Id:   utils.Ptr(event.ContentBlock.ID),
					Type: utils.Ptr("function"),
					Function: &types.FunctionCall{
						Name:      utils.Ptr(event.ContentBlock.Name),
						Arguments: utils.Ptr(""),
					},
				}
			case "text":
				currentContent = &types.Content{
					ContentType:   commons.TEXT_CONTENT.String(),
					ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					Content:       []byte(""),
				}
			}

		case anthropic.ContentBlockDeltaEvent:
			switch event.Delta.Type {
			case "text_delta":
				content := event.Delta.Text
				if content != "" && currentContent != nil {
					currentContent.Content = append(currentContent.Content, []byte(content)...)
					if !isToolCall {
						if err := onStream(types.Message{
							Contents: []*types.Content{{
								ContentType:   commons.TEXT_CONTENT.String(),
								ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
								Content:       []byte(content),
							}},
							Role: "assistant",
						}); err != nil {
						}
					}
				}
			case "input_json_delta":
				if currentToolCall != nil {
					currentToolCall.Function.MergeArguments(utils.Ptr(event.Delta.PartialJSON))
				}
			}

		case anthropic.ContentBlockStopEvent:
			if currentToolCall != nil {
				completeMessage.ToolCalls = append(completeMessage.ToolCalls, currentToolCall)
			}
			if currentContent != nil {
				completeMessage.Contents = append(completeMessage.Contents, currentContent)
			}
			isToolCall = false

		case anthropic.MessageStopEvent:
			metrics.OnAddMetrics(llc.UsageMetrics(message.Usage)...)
			options.AIOptions.PostHook(map[string]interface{}{
				"result": utils.ToJson(message),
			}, metrics.OnSuccess().Build())
			onMetrics(&completeMessage, metrics.Build())
			return nil
		}

	}
	if stream.Err() != nil {
		llc.logger.Errorf("Stream error: %v", stream.Err())
		onError(stream.Err())
		return stream.Err()
	}
	return nil
}
func (llc *largeLanguageCaller) GetMessageNewParams(opts *internal_callers.ChatCompletionOptions) anthropic.MessageNewParams {
	options := anthropic.MessageNewParams{}
	if len(opts.ToolDefinitions) > 0 {
		fns := make([]anthropic.ToolUnionParam, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			switch tl.Type {
			case "tool":
			case "function":
				fn := tl.Function
				if fn != nil {
					funcDef := &anthropic.ToolParam{
						Name:        fn.Name,
						Description: anthropic.String(fn.Description),
					}
					if fn.Parameters != nil {
						funcDef.InputSchema = anthropic.ToolInputSchemaParam{
							Properties: fn.Parameters.Properties,
							Required:   fn.Parameters.Required,
						}
					}
					fns[idx] = anthropic.ToolUnionParam{
						OfTool: funcDef,
					}
				}
			}
		}
		options.Tools = fns
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = anthropic.Model(mn)
			}
		case "model.max_tokens":
			if mct, err := utils.AnyToInt64(value); err == nil {
				options.MaxTokens = mct
			}

		case "model.thinking":
			if format, err := utils.AnyToJSON(value); err == nil {
				if enabled, ok := format["enabled"].(bool); ok && enabled {
					if budgetTokens, ok := format["budget_tokens"].(float64); ok {
						options.Thinking = anthropic.ThinkingConfigParamOfEnabled(int64(budgetTokens))
					}
				}
			}

		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = anthropic.Float(temp)
			}
		case "model.top_k":
			if topk, err := utils.AnyToInt64(value); err == nil {
				options.TopK = anthropic.Int(topk)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.TopP = anthropic.Float(topP)
			}

		}

	}
	return options
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
		options.AIOptions.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return nil, metrics.OnFailure().Build(), err
	}

	instruction, messages := llc.BuildHistory(allMessages)
	params := llc.GetMessageNewParams(options)
	params.Messages = messages
	params.System = instruction

	options.AIOptions.PreHook(utils.ToJson(params))
	resp, err := client.Messages.New(ctx, params)
	if err != nil {
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": resp,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}

	internalMessage := llc.convertAnthropicMessageToInternal(*resp)
	metrics.OnAddMetrics(llc.UsageMetrics(resp.Usage)...)
	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
		"error":  err,
	}, metrics.OnSuccess().Build())
	return &internalMessage, metrics.Build(), nil
}

func (llc *largeLanguageCaller) convertAnthropicMessageToInternal(message anthropic.Message) types.Message {
	internalMessage := types.Message{
		Contents:  make([]*types.Content, 0),
		ToolCalls: make([]*types.ToolCall, 0),
	}
	for _, content := range message.Content {
		switch c := content.AsAny().(type) {
		case anthropic.TextBlock:
			internalMessage.Contents = append(internalMessage.Contents, &types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(c.Text),
			})
		case anthropic.ToolUseBlock:
			internalMessage.ToolCalls = append(internalMessage.ToolCalls, &types.ToolCall{
				Id:   utils.Ptr(c.ID),
				Type: utils.Ptr("function"),
				Function: &types.FunctionCall{
					Name:      utils.Ptr(c.Name),
					Arguments: utils.Ptr(string(c.JSON.Input.Raw())),
				},
			})
		}
	}

	return internalMessage
}
