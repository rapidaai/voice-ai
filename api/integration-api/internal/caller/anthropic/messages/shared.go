// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_messages

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"google.golang.org/protobuf/types/known/anypb"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func buildHistory(
	logger commons.Logger,
	allMessages []*protos.Message,
) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	messages := make([]anthropic.MessageParam, 0)
	systemPrompt := make([]anthropic.TextBlockParam, 0)

	for _, msg := range allMessages {
		switch msg.GetRole() {
		case "assistant":
			mConnect := make([]anthropic.ContentBlockParamUnion, 0)
			if assistant := msg.GetAssistant(); assistant != nil {
				for _, c := range assistant.GetContents() {
					mConnect = append(mConnect, anthropic.ContentBlockParamUnion{
						OfText: &anthropic.TextBlockParam{Text: c},
					})
				}

				for _, tc := range assistant.GetToolCalls() {
					if tc.GetFunction() == nil {
						continue
					}
					var input map[string]interface{}
					if err := json.Unmarshal([]byte(tc.GetFunction().GetArguments()), &input); err != nil {
						logger.Warnf("Invalid JSON in tool call arguments: %v", err)
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
			}
			if len(mConnect) > 0 {
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: mConnect,
				})
			}
		case "user":
			if u := msg.GetUser(); u != nil && strings.TrimSpace(u.GetContent()) != "" {
				messages = append(messages, anthropic.MessageParam{
					Role: anthropic.MessageParamRoleUser,
					Content: []anthropic.ContentBlockParamUnion{{
						OfText: &anthropic.TextBlockParam{
							Text: u.GetContent(),
						},
					}},
				})
			}
		case "tool":
			tContent := make([]anthropic.ContentBlockParamUnion, 0)
			if tool := msg.GetTool(); tool != nil {
				for _, c := range tool.GetTools() {
					tContent = append(tContent, anthropic.ContentBlockParamUnion{
						OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: c.GetId(),
							Content: []anthropic.ToolResultBlockParamContentUnion{{
								OfText: &anthropic.TextBlockParam{Text: c.GetContent()},
							}},
						},
					})
				}
			}
			if len(tContent) > 0 {
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: tContent,
				})
			}
		case "system":
			if c := msg.GetSystem(); c != nil {
				systemPrompt = append(systemPrompt, anthropic.TextBlockParam{
					Text: c.GetContent(),
				})
			}
		}
	}
	return systemPrompt, messages
}

func buildMessageNewParams(opts *internal_callers.ChatCompletionOptions) anthropic.MessageNewParams {
	return buildMessageParams(opts.ModelParameter, opts.ToolDefinitions)
}

func buildStreamMessageNewParams(opts *internal_callers.ChatStreamCompletionOptions) anthropic.MessageNewParams {
	return buildMessageParams(opts.ModelParameter, opts.ToolDefinitions)
}

func buildMessageParams(
	modelParameter map[string]*anypb.Any,
	toolDefinitions []*internal_callers.ToolDefinition,
) anthropic.MessageNewParams {
	options := anthropic.MessageNewParams{}
	modelName := canonicalAnthropicModelName(anthropicModelName(modelParameter))
	allowSamplingControls := !anthropicModelDisallowsSamplingControls(modelName)

	if len(toolDefinitions) > 0 {
		fns := make([]anthropic.ToolUnionParam, len(toolDefinitions))
		for idx, tl := range toolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := &anthropic.ToolParam{Name: fn.Name}
			if fn.Description != "" {
				funcDef.Description = anthropic.String(fn.Description)
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
		options.Tools = fns
	}

	for key, value := range modelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = anthropic.Model(canonicalAnthropicModelName(mn))
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
			if !allowSamplingControls {
				continue
			}
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = anthropic.Float(temp)
			}
		case "model.top_k":
			if !allowSamplingControls {
				continue
			}
			if topk, err := utils.AnyToInt64(value); err == nil {
				options.TopK = anthropic.Int(topk)
			}
		case "model.top_p":
			if !allowSamplingControls {
				continue
			}
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.TopP = anthropic.Float(topP)
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				jsonSchemaParam := anthropic.OutputConfigParam{}
				jsonData, marshalErr := json.Marshal(format)
				if marshalErr == nil {
					_ = json.Unmarshal(jsonData, &jsonSchemaParam)
				}
				options.OutputConfig = jsonSchemaParam
			}
		}
	}
	return options
}

func anthropicModelName(modelParameter map[string]*anypb.Any) string {
	if value, ok := modelParameter["model.name"]; ok {
		if name, err := utils.AnyToString(value); err == nil {
			return name
		}
	}
	return ""
}

func canonicalAnthropicModelName(modelName string) string {
	switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(modelName)), "anthropic/") {
	case "claude-sonnet-4-20250514":
		return "claude-sonnet-4-6"
	case "claude-3-7-sonnet-20250219":
		return "claude-sonnet-4-6"
	case "claude-3-5-sonnet-20241022":
		return "claude-sonnet-4-6"
	case "claude-3-5-haiku-20241022":
		return "claude-haiku-5-5"
	case "claude-opus-4-20250514":
		return "claude-opus-4-8"
	default:
		return modelName
	}
}

func anthropicModelDisallowsSamplingControls(modelName string) bool {
	modelName = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(modelName)), "anthropic/")
	if modelName == "" {
		return false
	}

	parts := strings.Split(modelName, "-")
	major := -1
	minor := 0
	foundMajor := false

	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		if !foundMajor {
			major = n
			foundMajor = true
			continue
		}
		if len(part) > 2 {
			break
		}
		minor = n
		break
	}

	return major > 4 || major == 4 && minor >= 7
}

func convertAnthropicMessageToProto(message anthropic.Message) *protos.Message {
	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)

	for _, content := range message.Content {
		switch c := content.AsAny().(type) {
		case anthropic.TextBlock:
			contents = append(contents, c.Text)
		case anthropic.ToolUseBlock:
			toolCalls = append(toolCalls, &protos.ToolCall{
				Id:   c.ID,
				Type: "function",
				Function: &protos.FunctionCall{
					Name:      c.Name,
					Arguments: string(c.JSON.Input.Raw()),
				},
			})
		}
	}

	return &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents:  contents,
				ToolCalls: toolCalls,
			},
		},
	}
}
