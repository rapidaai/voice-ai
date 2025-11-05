package internal_cohere_callers

import (
	"context"
	"strings"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	Cohere
}

func NewLargeLanguageCaller(logger commons.Logger, credential *lexatic_backend.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Cohere: NewCohere(logger, credential),
	}
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
		llc.logger.Errorf("chat completion unable to get client for cohere: %v", err)
		onError(err)
		onMetrics(nil, metrics.OnFailure().Build())
		return err
	}

	chatRequest := llc.GetChatStreamRequest(options)
	chatRequest.Messages = llc.BuildHistory(allMessages)

	options.AIOptions.PreHook(utils.ToJson(chatRequest))
	llc.logger.Benchmark("Cohere.llm.GetChatCompletion.llmRequestPrepare", time.Since(start))

	resp, err := client.V2.ChatStream(ctx, chatRequest)
	if err != nil {
		llc.logger.Errorf("Failed to get chat completions stream: %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"result": utils.ToJson(resp),
			"error":  err,
		}, metrics.Build())
		onMetrics(nil, metrics.OnFailure().Build())
		onError(err)
		return err
	}

	defer resp.Close()
	msg := types.Message{
		Contents: make([]*types.Content, 0),
		Role:     "assistant",
	}
	var currentToolCall *types.ToolCall
	var currentContent *types.Content
	isToolCall := false
	for {
		select {
		case <-ctx.Done():
			llc.logger.Infof("Context canceled during stream processing")
			return ctx.Err()
		default:
			rep, _ := resp.Recv()
			switch {
			case rep.MessageStart != nil:
				continue
			case rep.ContentStart != nil:
				if rep.ContentStart.Delta != nil && rep.ContentStart.Delta.Message != nil && rep.ContentStart.Delta.Message.Content != nil {
					if text := rep.ContentStart.Delta.Message.Content.GetText(); text != nil {
						currentContent = &types.Content{
							ContentType:   commons.TEXT_CONTENT.String(),
							ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
							Content:       []byte(*text),
						}
					}

				}
			case rep.ContentDelta != nil:
				if rep.ContentDelta.Delta != nil && rep.ContentDelta.Delta.Message != nil && rep.ContentDelta.Delta.Message.Content != nil {
					if text := rep.ContentDelta.Delta.Message.Content.GetText(); text != nil {
						if !isToolCall {
							if err := onStream(types.Message{
								Contents: []*types.Content{{
									ContentType:   commons.TEXT_CONTENT.String(),
									ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
									Content:       []byte(*text),
								}},
								Role: "assistant",
							}); err != nil {
								return err
							}
						}
						currentContent.Content = append([]byte(currentContent.Content), []byte(*text)...)
					}
				}
			case rep.ContentEnd != nil:
				msg.Contents = append(msg.Contents, currentContent)
				currentContent = nil
			case rep.ToolCallStart != nil:
				isToolCall = true
				if rep.ToolCallStart.Delta != nil && rep.ToolCallStart.Delta.Message != nil && rep.ToolCallStart.Delta.Message.ToolCalls != nil {
					currentToolCall = &types.ToolCall{
						Id:   utils.Ptr(rep.ToolCallStart.Delta.Message.ToolCalls.Id),
						Type: utils.Ptr(rep.ToolCallStart.Delta.Message.ToolCalls.Type()),
						Function: &types.FunctionCall{
							Name:      rep.ToolCallStart.Delta.Message.ToolCalls.Function.Name,
							Arguments: rep.ToolCallStart.Delta.Message.ToolCalls.Function.Arguments,
						},
					}
				}
			case rep.ToolCallDelta != nil:
				if currentToolCall != nil && rep.ToolCallDelta.Delta != nil && rep.ToolCallDelta.Delta.Message != nil && rep.ToolCallDelta.Delta.Message.ToolCalls != nil {
					currentToolCall.Function.MergeArguments(rep.ToolCallDelta.Delta.Message.ToolCalls.Function.Arguments)
				}
			case rep.ToolCallEnd != nil:
				if msg.ToolCalls == nil {
					msg.ToolCalls = make([]*types.ToolCall, 0)
				}
				msg.ToolCalls = append(msg.ToolCalls, currentToolCall)
				currentToolCall = nil

			case rep.MessageEnd != nil:
				metrics.OnAddMetrics(llc.UsageMetrics(rep.MessageEnd.Delta.Usage)...)
				metrics.OnSuccess()
				options.AIOptions.PostHook(map[string]interface{}{
					"result": msg,
				}, metrics.Build())
				onMetrics(&msg, metrics.Build())
				return nil
			}
		}
	}
}

func (llc *largeLanguageCaller) BuildHistory(allMessages []*lexatic_backend.Message) cohere.ChatMessages {
	msgHistory := make([]*cohere.ChatMessageV2, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case "chatbot":
		case "assistant":
			_msg := &cohere.ChatMessageV2{
				Role:      "assistant",
				Assistant: &cohere.AssistantMessage{},
			}

			if len(cntn.GetContents()) > 0 {
				_msg.Assistant.Content = &cohere.AssistantMessageV2Content{
					String: types.OnlyStringProtoContent(cntn.GetContents()),
				}
			}
			if len(cntn.GetToolCalls()) > 0 {
				fctCall := make([]*cohere.ToolCallV2, 0)
				err := utils.Cast(cntn.ToolCalls, &fctCall)
				if err != nil {
					llc.logger.Errorf("unable to cast the to function tool call %v", err)
				}
				_msg.Assistant.ToolCalls = fctCall
			}
			msgHistory = append(msgHistory, _msg)
		case "system":
			msgHistory = append(msgHistory, &cohere.ChatMessageV2{
				Role: "system",
				System: &cohere.SystemMessageV2{
					Content: &cohere.SystemMessageV2Content{
						String: types.OnlyStringProtoContent(cntn.GetContents()),
					},
				},
			})
		case "user":
			msgHistory = append(msgHistory, &cohere.ChatMessageV2{
				Role: "user",
				User: &cohere.UserMessageV2{Content: &cohere.UserMessageV2Content{
					String: types.OnlyStringProtoContent(cntn.GetContents()),
				}},
			})
		case "tool":
			for _, tcl := range cntn.GetContents() {
				msgHistory = append(msgHistory,
					&cohere.ChatMessageV2{
						Role: "tool",
						Tool: &cohere.ToolMessageV2{
							ToolCallId: tcl.GetContentType(),
							Content: &cohere.ToolMessageV2Content{
								String: string(tcl.GetContent()),
							}},
					})
			}
		default:
			llc.logger.Warnf("Unknown role: %s and everytihgn", cntn.String())
			continue
		}

	}
	return msgHistory
}
func (llc *largeLanguageCaller) GetChatRequest(opts *internal_callers.ChatCompletionOptions) *cohere.V2ChatRequest {
	options := &cohere.V2ChatRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		}
	}
	return options
}
func (llc *largeLanguageCaller) GetChatStreamRequest(opts *internal_callers.ChatCompletionOptions) *cohere.V2ChatStreamRequest {
	options := &cohere.V2ChatStreamRequest{}
	if len(opts.ToolDefinitions) > 0 {
		options.Tools = make([]*cohere.ToolV2, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			options.Tools[idx] = &cohere.ToolV2{
				Function: &cohere.ToolV2Function{
					Name:        tl.Function.Name,
					Description: &tl.Function.Description,
					Parameters:  tl.Function.Parameters.ToMap(),
				},
			}
		}
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "text":
					options.ResponseFormat = &cohere.ResponseFormatV2{
						Type: "text",
					}
				case "json_object":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						options.ResponseFormat = &cohere.ResponseFormatV2{
							Type: "json_object",
							JsonObject: &cohere.JsonResponseFormatV2{
								JsonSchema: schemaData,
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
	llc.logger.Debugf("chat complition for cohere")
	//
	// Working with chat complition with vision
	//
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat complition unable to get client for cohere %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	metrics.OnStart()
	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	chatRequest := llc.GetChatRequest(options)
	chatRequest.Messages = llc.BuildHistory(allMessages)

	options.AIOptions.PreHook(utils.ToJson(*chatRequest))
	resp, err := client.V2.Chat(
		ctx,
		chatRequest,
	)
	if err != nil {
		llc.logger.Errorf("chat complition unable to get client for cohere %v", err)
		options.AIOptions.PostHook(map[string]interface{}{
			"error":  err,
			"result": resp,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()
	metrics.OnAddMetrics(llc.UsageMetrics(resp.GetUsage())...)

	// // call when you are done
	options.AIOptions.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.Build())

	message := &types.Message{
		Role:      resp.GetMessage().Role(),
		Contents:  make([]*types.Content, 0),
		ToolCalls: make([]*types.ToolCall, 0),
	}

	for _, msg := range resp.GetMessage().GetContent() {
		message.Contents = append(message.Contents, &types.Content{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(msg.Text.Text),
		})
	}

	for _, tl := range resp.GetMessage().GetToolCalls() {
		message.ToolCalls = append(message.ToolCalls, &types.ToolCall{
			Id:   utils.Ptr(tl.GetId()),
			Type: utils.Ptr(tl.Type()),
			Function: &types.FunctionCall{
				Name:      tl.GetFunction().GetName(),
				Arguments: tl.GetFunction().GetArguments(),
			},
		})
	}
	return message, metrics.Build(), nil
}
