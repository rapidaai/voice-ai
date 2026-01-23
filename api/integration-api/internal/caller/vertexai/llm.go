// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_callers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/genai"
)

type largeLanguageCaller struct {
	VertexAi
}

func NewLargeLanguageCaller(logger commons.Logger, credential *protos.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		VertexAi: vertexai(logger, credential),
	}
}

// StreamChatCompletion streams the chat response piece-by-piece from the AI model.
// Parameters:
// - ctx: Context for managing request lifecycle.
// - allMessages: List of all previous messages exchanged in the chat.
// - options: Configuration options for the chat completion request.
// - onStream: Callback function to handle streaming chunks of type `types.Message`.
// - onMetrics: Callback to collect metrics for processing completion.
// - onError: Callback to handle errors encountered during processing.
// Returns:
// - error: Any errors encountered during processing.

func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(types.Message) error,
	onMetrics func(*types.Message, types.Metrics) error,
	onError func(err error),
) error {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := llc.GetClient()
	if err != nil {
		options.AIOptions.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		onMetrics(nil, metrics.OnFailure().Build())
		onError(err)
		return err
	}

	// Setting up timeout for streaming
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	instruction, history, current := llc.buildHistory(allMessages)
	model, config := llc.getGenerationConfig(options)
	config.SystemInstruction = instruction
	chat, err := client.Chats.Create(ctx,
		model,
		config,
		history,
	)
	if err != nil {
		options.AIOptions.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		onMetrics(nil, metrics.OnFailure().Build())
		onError(err)
		return err
	}

	options.AIOptions.PreHook(llc.toSimplifiedJson(model, config, history, current))
	completeMsg := types.Message{
		Role: "model",
	}
	isToolCall := false
	accumulator := &GoogleChatCompletionAccumulator{}
	for resp, err := range chat.SendMessageStream(ctx, current) {
		if err != nil {
			options.AIOptions.PostHook(map[string]interface{}{
				"result": utils.ToJson(resp),
				"error":  err,
			}, metrics.OnFailure().Build())
			onMetrics(nil, metrics.OnFailure().Build())
			onError(err)
			return err
		}
		accumulator.AddChunk(resp)
		for _, choice := range resp.Candidates {
			if choice.Content != nil {
				for _, part := range choice.Content.Parts {
					if part.FunctionCall != nil {
						isToolCall = true
						if completeMsg.ToolCalls == nil {
							completeMsg.ToolCalls = make([]*types.ToolCall, 0)
						}
						for len(completeMsg.ToolCalls) <= int(choice.Index) {
							completeMsg.ToolCalls = append(completeMsg.ToolCalls, nil)
						}
						argsJSON, err := json.Marshal(part.FunctionCall.Args)
						if err != nil {
							llc.logger.Errorf("Error marshaling function args: %v", err)
							argsJSON = []byte("{}")
						}
						completeMsg.ToolCalls[int(choice.Index)] = &types.ToolCall{
							Id:   &part.FunctionCall.ID,
							Type: utils.Ptr("function"),
							Function: &types.FunctionCall{
								Name:      &part.FunctionCall.Name,
								Arguments: utils.Ptr(string(argsJSON)),
							},
						}
					}
					if part.Text != "" {
						if !isToolCall {
							onStream(types.Message{
								Contents: []*types.Content{
									{
										ContentType:   commons.TEXT_CONTENT.String(),
										ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
										Content:       []byte(part.Text),
									},
								},
								Role: choice.Content.Role,
							})
						}

						if completeMsg.Contents == nil {
							completeMsg.Contents = make([]*types.Content, 0)
						}
						for len(completeMsg.Contents) <= int(choice.Index) {
							completeMsg.Contents = append(completeMsg.Contents, &types.Content{
								ContentType:   commons.TEXT_CONTENT.String(),
								ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
								Content:       []byte{},
							})
						}
						completeMsg.Contents[int(choice.Index)].Content = append(completeMsg.Contents[int(choice.Index)].Content, []byte(part.Text)...)
					}
				}
			}
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{
		"result": accumulator,
	}, metrics.OnSuccess().Build())
	metrics.OnAddMetrics(llc.UsageMetrics(accumulator.UsageMetadata)...)
	onMetrics(&completeMsg, metrics.OnSuccess().Build())
	return nil
}

// GetChatCompletion performs a synchronous request for a single chat completion response.
// Parameters:
// - ctx: Context for managing request lifecycle.
// - allMessages: List of all previous messages exchanged in the chat.
// - options: Configuration options for the chat completion request.
// Returns:
// - *types.Message: Constructed message response.
// - types.Metrics: Metrics collected during execution.
// - error: Any errors encountered during processing.

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*types.Message, types.Metrics, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("getting error for chat completion %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	if len(allMessages) == 0 {
		err := errors.New("no messages in the input")
		llc.logger.Errorf("invalid input: %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	//
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	instruction, histories, current := llc.buildHistory(allMessages)
	model, config := llc.getGenerationConfig(options)
	config.SystemInstruction = instruction
	chat, err := client.Chats.Create(ctx,
		model,
		config,
		histories)

	if err != nil {
		llc.logger.Errorf("error creating chat: %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	options.AIOptions.PreHook(llc.toSimplifiedJson(model, config, histories, current))
	resp, err := chat.SendMessage(ctx, current)
	if err != nil {
		llc.logger.Errorf("getting error for chat completion %+v %+v", err, resp)
		metrics.OnFailure()
		options.AIOptions.PostHook(map[string]interface{}{"result": resp, "error": err}, metrics.Build())
		return nil, metrics.Build(), err
	}

	output := make([]*types.Content, len(resp.Candidates))
	metrics.OnSuccess()
	metrics.OnAddMetrics(llc.UsageMetrics(resp.UsageMetadata)...)
	for _, choice := range resp.Candidates {
		if choice.Content != nil {
			buf := strings.Builder{}
			if choice.Content != nil {
				for _, part := range choice.Content.Parts {
					_, _ = buf.WriteString(part.Text)
				}
			}
			output[choice.Index] = &types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(buf.String()),
			}
		}
	}
	options.AIOptions.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	return &types.Message{
		Role:     "model",
		Contents: output,
	}, metrics.Build(), nil
}

// buildHistory constructs the historical context for the chat session.
// Parameters:
// - allMessages: List of all previous messages exchanged in the chat.
// Returns:
// - *genai.Content: System instructions extracted.
// - []*genai.Content: Conversation history sent as input.
// - genai.Part: Last conversation part extracted.

func (llc *largeLanguageCaller) buildHistory(allMessages []*protos.Message) (*genai.Content, []*genai.Content, genai.Part) {
	history := make([]*genai.Content, 0)
	for _, msg := range allMessages {
		switch msg.GetRole() {
		case "user":
			content := &genai.Content{
				Role:  "user",
				Parts: make([]*genai.Part, 0, len(msg.GetContents())),
			}
			for _, ct := range msg.GetContents() {
				if ct.ContentType == commons.TEXT_CONTENT.String() {
					content.Parts = append(content.Parts, &genai.Part{Text: string(ct.GetContent())})
				} else {
					llc.logger.Warnf("Unknown content type: %s", ct.ContentType)
				}
			}
			history = append(history, content)
		case "model", "assistant":
			content := &genai.Content{
				Role:  "model",
				Parts: make([]*genai.Part, 0, len(msg.GetContents())),
			}
			for _, ct := range msg.GetContents() {
				if ct.ContentType == commons.TEXT_CONTENT.String() {
					content.Parts = append(content.Parts, &genai.Part{Text: string(ct.GetContent())})
				}
			}
			for _, tc := range msg.GetToolCalls() {
				var argumentMap map[string]any
				if err := json.Unmarshal([]byte(tc.GetFunction().GetArguments()), &argumentMap); err != nil {
					argumentMap = make(map[string]any)
				}
				content.Parts = append(content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   tc.GetId(),
						Args: argumentMap,
						Name: tc.GetFunction().GetName(),
					},
				})
			}
			history = append(history, content)
		case "system":
			sysContent := &genai.Content{
				Parts: make([]*genai.Part, 0, len(msg.GetContents())),
			}
			for _, ct := range msg.GetContents() {
				if ct.ContentType == commons.TEXT_CONTENT.String() {
					sysContent.Parts = append(sysContent.Parts, &genai.Part{Text: string(ct.GetContent())})
				}
			}
			history = append(history, sysContent)
		case "tool":
			content := &genai.Content{
				Role:  "user",
				Parts: make([]*genai.Part, 0, len(msg.GetContents())),
			}

			// get the last message and tool name and id

			for _, v := range msg.GetContents() {
				var responseMap map[string]any
				if err := json.Unmarshal([]byte(v.GetContent()), &responseMap); err != nil {
					responseMap = make(map[string]any)
				}
				content.Parts = append(content.Parts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						Name:     v.GetName(),
						ID:       v.GetContentType(),
						Response: responseMap,
					},
				})
			}
			history = append(history, content)
		default:
			llc.logger.Warnf("Unknown role: %s", msg.GetRole())
			continue
		}

	}

	var lastPart genai.Part
	if len(history) > 0 && len(history[len(history)-1].Parts) > 0 {
		lastPart = *history[len(history)-1].Parts[0]
	} else {
		lastPart = genai.Part{} // or some default value
	}

	if len(history) == 0 {
		return nil, history, lastPart
	}
	return history[0], history[1:], lastPart
}

// buildFunctionParameter converts internal function parameter definitions into Vertex AI schemas.
// Parameters:
// - fp: Internal representation of function parameters.
// Returns:
// - *genai.Schema: Converted schema in Vertex AI format.
func (llc *largeLanguageCaller) buildFunctionParameter(fp *internal_callers.FunctionParameter) *genai.Schema {
	schema := &genai.Schema{
		Type:       genai.Type(fp.Type),
		Properties: make(map[string]*genai.Schema),
	}
	if fp.Required != nil {
		schema.Required = fp.Required
	}
	for key, prop := range fp.Properties {
		schema.Properties[key] = llc.buildFunctionProperties(&prop)
	}
	return schema
}

// buildFunctionProperties maps function property details to Vertex AI schema properties.
// Parameters:
// - fpp: Function parameter property details.
// Returns:
// - *genai.Schema: Converted schema in Vertex AI format.
func (llc *largeLanguageCaller) buildFunctionProperties(fpp *internal_callers.FunctionParameterProperty) *genai.Schema {
	schema := &genai.Schema{
		Type:        genai.Type(fpp.Type),
		Description: fpp.Description,
	}
	if fpp.Enum != nil {
		schema.Enum = make([]string, len(fpp.Enum))
		for i, v := range fpp.Enum {
			if v != nil {
				schema.Enum[i] = *v
			}
		}
	}
	if fpp.Items != nil {
		schema.Items = &genai.Schema{
			Type: genai.Type(fpp.Items["type"].(string)),
		}
	}
	return schema
}

// getGenerationConfig converts user-provided options into Vertex AI configuration.
// Parameters:
// - opts: User-defined chat completion options.
// Returns:
// - mdl: Model name used for the request.
// - config: Generated configuration for the request.
func (llc *largeLanguageCaller) getGenerationConfig(
	opts *internal_callers.ChatCompletionOptions,
) (mdl string, config *genai.GenerateContentConfig) {
	config = &genai.GenerateContentConfig{}
	if len(opts.ToolDefinitions) > 0 {

		fd := make([]*genai.FunctionDeclaration, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			switch tl.Type {
			case "function":
				fn := tl.Function
				if fn != nil {
					funcDef := &genai.FunctionDeclaration{
						Name:        fn.Name,
						Description: fn.Description,
					}
					if fn.Parameters != nil {
						funcDef.Parameters = llc.buildFunctionParameter(fn.Parameters)
					}
					fd[idx] = funcDef
				}
			}
		}

		config.Tools = []*genai.Tool{{
			FunctionDeclarations: fd,
		}}
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				mdl = modelName
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat32(value); err == nil {
				config.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat32(value); err == nil {
				config.TopP = utils.Ptr(topP)
			}
		case "model.top_k":
			if topK, err := utils.AnyToFloat32(value); err == nil {
				config.TopK = utils.Ptr(topK)
			}
		case "model.max_completion_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				config.MaxOutputTokens = int32(maxTokens)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				config.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat32(value); err == nil {
				config.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat32(value); err == nil {
				config.PresencePenalty = utils.Ptr(pp)
			}
		case "model.seed":
			if seed, err := utils.AnyToInt32(value); err == nil {
				config.Seed = utils.Ptr(seed)
			}

		case "model.thinking":
			if format, err := utils.AnyToJSON(value); err == nil {
				config.ThinkingConfig = &genai.ThinkingConfig{}
				if enabled, ok := format["include_thoughts"].(bool); ok && enabled {
					config.ThinkingConfig.IncludeThoughts = enabled

					if budgetTokens, ok := format["thinking_budget"].(int32); ok {
						config.ThinkingConfig.ThinkingBudget = utils.Ptr(int32(budgetTokens))
					}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["response_mime_type"].(string) {
				case "text/x.enum":
					if schemaData, ok := format["response_schema"].(map[string]interface{}); ok {
						config.ResponseMIMEType = "text/x.enum"
						config.ResponseJsonSchema = schemaData
					}
				case "application/json":
					if schemaData, ok := format["response_schema"].(map[string]interface{}); ok {
						config.ResponseMIMEType = "application/json"
						config.ResponseJsonSchema = schemaData
					}
				}
			}
		}
	}
	return
}

// toSimplifiedJson simplifies the JSON representation of the final request payload.
// Parameters:
// - model: Model name used for the request.
// - cfg: Configuration settings used.
// - history: Historical context provided.
// - ct: Current conversation part.
// Returns:
// - map[string]interface{}: Simplified JSON payload.
func (llc *largeLanguageCaller) toSimplifiedJson(model string, cfg *genai.GenerateContentConfig, history []*genai.Content, ct genai.Part) map[string]interface{} {
	wt := struct {
		Config               *genai.GenerateContentConfig
		Current              genai.Part
		Model                string
		ComprehensiveHistory []*genai.Content
	}{
		Model:                model,
		Config:               cfg,
		Current:              ct,
		ComprehensiveHistory: history,
	}
	return utils.ToJson(wt)
}
