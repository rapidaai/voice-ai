package internal_openai_callers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	OpenAI
}

func NewLargeLanguageCaller(logger commons.Logger, credential *lexatic_backend.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		OpenAI: openAI(logger, credential),
	}
}

func (llc *largeLanguageCaller) ChatCompletionOptions(
	opts *internal_callers.ChatCompletionOptions,
) openai.ChatCompletionNewParams {
	options := openai.ChatCompletionNewParams{}
	if len(opts.ToolDefinitions) > 0 {
		fns := make([]openai.ChatCompletionToolParam, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			switch tl.Type {
			case "tool":
			case "function":
				fn := tl.Function
				if fn != nil {
					funcDef := openai.FunctionDefinitionParam{
						Name:        fn.Name,
						Description: openai.String(fn.Description),
					}
					if fn.Parameters != nil {
						funcDef.Parameters = fn.Parameters.ToMap()
					}
					fns[idx] = openai.ChatCompletionToolParam{
						Function: funcDef,
					}
				}
			}
		}
		options.Tools = fns
	}

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
		case "model.reasoning_effort":
			if re, err := utils.AnyToString(value); err == nil {
				options.ReasoningEffort = shared.ReasoningEffort(re)
			}
		case "model.seed":
			if seed, err := utils.AnyToInt64(value); err == nil {
				options.Seed = openai.Int(seed)
			}
		case "model.service_tier":
			if st, err := utils.AnyToString(value); err == nil {
				options.ServiceTier = openai.ChatCompletionNewParamsServiceTier(st)
			}
		case "model.top_logprobs":
			if tl, err := utils.AnyToInt64(value); err == nil {
				options.TopLogprobs = openai.Int(tl)
			}
		case "model.metadata":
			format, _ := utils.AnyToString(value)
			var mtd map[string]string
			if err := json.Unmarshal([]byte(format), &mtd); err == nil {
				options.Metadata = shared.Metadata(mtd)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = openai.Float(fp)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = openai.Float(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.TopP = openai.Float(topP)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = openai.Float(pp)
			}
		case "model.max_completion_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				options.MaxTokens = openai.Int(maxTokens)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				for _, stopper := range strings.Split(stopStr, ",") {
					if strings.TrimSpace(stopper) != "" {
						options.Stop.OfStringArray = append(options.Stop.OfStringArray, stopper)
					}
				}
			}
		case "model.tool_choice":
			if choice, err := utils.AnyToString(value); err == nil {
				switch choice {
				case "auto":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("auto"),
					}
				case "required":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("required"),
					}
				case "none":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("none"),
					}
				default:
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("none"),
					}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "json_object":
					options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
						OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
					}
				case "text":
					options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{}
				case "json_schema":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						jsonSchemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{}
						jsonData, err := json.Marshal(schemaData)
						if err == nil {
							json.Unmarshal(jsonData, &jsonSchemaParam)
						}
						options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
							OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
								JSONSchema: jsonSchemaParam,
							},
						}
					}
				}
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
		llc.logger.Errorf("chat complition unable to get client for openai %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	// message and options
	llmRequest := llc.ChatCompletionOptions(options)
	llmRequest.Messages = llc.BuildHistory(allMessages)

	// prehook
	options.AIOptions.PreHook(utils.ToJson(llmRequest))

	//chat complitions
	resp, err := client.Chat.Completions.New(ctx, llmRequest)
	if err != nil {
		llc.logger.Errorf("chat complition failed to get response from openai %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": resp,
		}, metrics.OnFailure().Build())
		return nil, metrics.OnFailure().Build(), err
	}

	message := types.Message{
		Contents: make([]*types.Content, 0),
	}
	metrics.OnSuccess()
	metrics.OnAddMetrics(llc.GetComplitionUsages(resp.Usage)...)
	// all the usages into the metrics

	for _, choice := range resp.Choices {
		message.Role = string(choice.Message.Role)
		switch choice.FinishReason {
		case "length", "content_filter":
		case "stop":
			message.Contents = append(message.Contents, &types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(choice.Message.Content),
			})
		case "function_call", "tool_calls":
			if choice.Message.ToolCalls != nil {
				for _, tool := range choice.Message.ToolCalls {
					if tool.Type == "function" {
						newToolCall := &types.ToolCall{
							Id:   &tool.ID,
							Type: utils.Ptr(string(tool.Type)),
							Function: &types.FunctionCall{
								Name:      utils.Ptr(tool.Function.Name),
								Arguments: utils.Ptr(tool.Function.Arguments),
							},
						}
						if message.ToolCalls == nil {
							message.ToolCalls = make([]*types.ToolCall, 0)
						}
						message.ToolCalls = append(message.ToolCalls, newToolCall)
					}
				}
			}
		}
	}

	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.OnSuccess().Build())
	return &message, metrics.Build(), nil
}

func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*lexatic_backend.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(types.Message) error,
	onMetrics func(*types.Message, types.Metrics) error,
	onError func(err error),
) error {
	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for openai: %v", err)
		onError(err)
		onMetrics(nil, metrics.OnFailure().Build())
		return err
	}

	completionsOptions := llc.ChatCompletionOptions(options)
	completionsOptions.Messages = llc.BuildHistory(allMessages)
	options.AIOptions.PreHook(utils.ToJson(completionsOptions))
	llc.logger.Benchmark("Openai.llm.GetChatCompletion.llmRequestPrepare", time.Since(start))

	// Get streaming response
	resp := client.Chat.Completions.NewStreaming(ctx, completionsOptions)
	if resp.Err() != nil {
		llc.logger.Errorf("Failed to get chat completions stream: %v", resp.Err())
		options.AIOptions.PostHook(map[string]interface{}{
			"result": utils.ToJson(resp),
			"error":  resp.Err(),
		}, metrics.Build())
		onMetrics(nil, metrics.OnFailure().Build())
		onError(resp.Err())
		return resp.Err()
	}
	defer resp.Close()
	completeMsg := types.Message{
		Role:      "assistant",
		Contents:  make([]*types.Content, 0),
		ToolCalls: make([]*types.ToolCall, 0),
	}

	accumulate := openai.ChatCompletionAccumulator{}
	for resp.Next() {
		chatCompletions := resp.Current()
		accumulate.AddChunk(chatCompletions)

		if _, ok := accumulate.JustFinishedContent(); ok {
			metrics.OnAddMetrics(llc.GetComplitionUsages(accumulate.Usage)...)
			metrics.OnSuccess()
			options.AIOptions.PostHook(map[string]interface{}{
				"result": utils.ToJson(accumulate),
			}, metrics.Build())
			onMetrics(&completeMsg, metrics.Build())
			return nil
		}

		if tool, ok := accumulate.JustFinishedToolCall(); ok {
			completeMsg.ToolCalls = append(completeMsg.ToolCalls, &types.ToolCall{
				Id: utils.Ptr(tool.ID),
				Function: &types.FunctionCall{
					Name:      utils.Ptr(tool.Name),
					Arguments: utils.Ptr(tool.Arguments),
				},
			})

			// Stream the complete message with tool calls
			if err := onStream(completeMsg); err != nil {
				llc.logger.Errorf("Error sending tool call data: %v", err)
				return err
			}

			metrics.OnAddMetrics(llc.GetComplitionUsages(accumulate.Usage)...)
			options.AIOptions.PostHook(map[string]interface{}{
				"result": utils.ToJson(accumulate),
			}, metrics.Build())
			onMetrics(&completeMsg, metrics.Build())
			return nil
		}

		deltaMsg := types.Message{
			Role:     "assistant",
			Contents: make([]*types.Content, 0),
		}

		for i, choice := range chatCompletions.Choices {
			content := choice.Delta.Content
			if content != "" {
				// Update complete message
				if len(completeMsg.Contents) <= i {
					completeMsg.Contents = append(completeMsg.Contents, &types.Content{
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
						Content:       []byte(content),
					})
				} else {
					completeMsg.Contents[i].Content = append(completeMsg.Contents[i].Content, []byte(content)...)
				}
				// Update delta message
				deltaMsg.Contents = append(deltaMsg.Contents, &types.Content{
					ContentType:   commons.TEXT_CONTENT.String(),
					ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					Content:       []byte(content),
				})
			}
		}

		// Stream content if there are changes and no tool calls
		if len(deltaMsg.Contents) > 0 {
			if err := onStream(deltaMsg); err != nil {
				llc.logger.Errorf("Error sending stream data: %v", err)
				return err
			}
		}
	}

	return nil
}

func (llc *largeLanguageCaller) BuildHistory(allMessages []*lexatic_backend.Message) []openai.ChatCompletionMessageParamUnion {
	msg := make([]openai.ChatCompletionMessageParamUnion, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case ChatRoleUser:
			var messageContent []openai.ChatCompletionContentPartUnionParam
			for _, ct := range cntn.GetContents() {
				switch ct.ContentType {
				case commons.TEXT_CONTENT.String():
					messageContent = append(messageContent, openai.ChatCompletionContentPartUnionParam{
						OfText: &openai.ChatCompletionContentPartTextParam{
							Text: string(ct.GetContent()),
						},
					})
				case commons.IMAGE_CONTENT.String():
					if ct.GetContentFormat() == commons.IMAGE_CONTENT_FORMAT_URL.String() {
						messageContent = append(messageContent, openai.ChatCompletionContentPartUnionParam{
							OfImageURL: &openai.ChatCompletionContentPartImageParam{
								ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
									URL: string(ct.GetContent()),
								},
							},
						})

					}
				default:
					llc.logger.Warnf("Unknown content type: %s", ct.ContentType)
				}
			}
			msg = append(msg, openai.UserMessage(messageContent))
		case ChatRoleAssistant:
			txtContent := types.OnlyStringProtoContent(cntn.GetContents())
			toolCalls := cntn.GetToolCalls()
			assistantMessage := openai.ChatCompletionAssistantMessageParam{}
			if len(txtContent) > 0 || len(toolCalls) > 0 {
				if len(txtContent) > 0 {
					assistantMessage.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(txtContent),
					}
				}
				if len(toolCalls) > 0 {
					fctCall := make([]openai.ChatCompletionMessageToolCallParam, 0)
					for _, ttc := range toolCalls {
						fctCall = append(fctCall, openai.ChatCompletionMessageToolCallParam{
							ID: ttc.GetId(),
							Function: openai.ChatCompletionMessageToolCallFunctionParam{
								Name:      ttc.GetFunction().GetName(),
								Arguments: ttc.GetFunction().GetArguments(),
							},
						})
					}
					assistantMessage.ToolCalls = fctCall
				}
				msg = append(msg, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMessage,
				})

			}

		case ChatRoleSystem:
			txtContent := types.OnlyStringProtoContent(cntn.GetContents())
			if len(txtContent) > 0 {
				msg = append(msg, openai.SystemMessage(txtContent))
			}

		case ChatRoleTool:
			for _, tcl := range cntn.GetContents() {
				toolId := tcl.GetContentType()
				msg = append(msg, openai.ToolMessage(string(tcl.GetContent()), toolId))
			}
		}

	}
	return msg
}
