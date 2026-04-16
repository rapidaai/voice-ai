// Rapida -- Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_minimax_callers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	MiniMax
}

func NewLargeLanguageCaller(logger commons.Logger, credential *protos.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		MiniMax: minimax(logger, credential),
	}
}

// clampTemperature ensures the temperature is within MiniMax's valid range (0, 1].
// MiniMax does not accept temperature=0; the minimum is just above 0.
func clampTemperature(temp float64) float64 {
	if temp <= 0 {
		return 0.01
	}
	if temp > 1 {
		return 1.0
	}
	return temp
}

func (llc *largeLanguageCaller) buildRequestBody(
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) map[string]interface{} {
	requestBody := map[string]interface{}{}

	for key, value := range options.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				requestBody["model"] = modelName
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				requestBody["temperature"] = clampTemperature(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				requestBody["top_p"] = topP
			}
		case "model.max_completion_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				requestBody["max_tokens"] = maxTokens
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				stops := []string{}
				for _, s := range strings.Split(stopStr, ",") {
					if trimmed := strings.TrimSpace(s); trimmed != "" {
						stops = append(stops, trimmed)
					}
				}
				if len(stops) > 0 {
					requestBody["stop"] = stops
				}
			}
		case "model.tool_choice":
			if choice, err := utils.AnyToString(value); err == nil {
				requestBody["tool_choice"] = choice
			}
		}
	}

	// Build tool definitions
	if len(options.ToolDefinitions) > 0 {
		tools := make([]map[string]interface{}, 0, len(options.ToolDefinitions))
		for _, tl := range options.ToolDefinitions {
			if tl.Type != "function" && tl.Type != "tool" {
				continue
			}
			if tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := map[string]interface{}{
				"name": fn.Name,
			}
			if fn.Description != "" {
				funcDef["description"] = fn.Description
			}
			if fn.Parameters != nil {
				funcDef["parameters"] = fn.Parameters.ToMap()
			} else {
				funcDef["parameters"] = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			tools = append(tools, map[string]interface{}{
				"type":     "function",
				"function": funcDef,
			})
		}
		if len(tools) > 0 {
			requestBody["tools"] = tools
		}
	}

	// Build messages
	requestBody["messages"] = buildMessages(allMessages)

	return requestBody
}

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	llc.logger.Debugf("getting chat completion from minimax")
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	requestBody := llc.buildRequestBody(allMessages, options)
	options.PreHook(requestBody)

	res, err := llc.CallJSON(ctx, "chat/completions", "POST", map[string]string{}, requestBody)
	if err != nil {
		llc.logger.Errorf("getting error for minimax chat completion %v", err)
		options.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()

	var resp MiniMaxMessageResponse
	if err := json.Unmarshal([]byte(*res), &resp); err != nil {
		llc.logger.Errorf("error while parsing minimax chat completion response %v", err)
		options.PostHook(map[string]interface{}{
			"result": res,
			"error":  err,
		}, metrics.Build())
		return nil, metrics.Build(), err
	}

	metrics.OnAddMetrics(llc.UsageMetrics(resp.Usage)...)

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}

	for _, choice := range resp.Choices {
		switch choice.FinishReason {
		case "stop":
			content := stripThinkingTags(choice.Message.Content)
			assistantMsg.Contents = append(assistantMsg.Contents, content)
		case "tool_calls":
			for _, tc := range choice.Message.ToolCalls {
				if tc.Type == "function" {
					assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
						Id:   tc.ID,
						Type: tc.Type,
						Function: &protos.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
			}
		}
	}

	message := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}
	options.PostHook(map[string]interface{}{
		"result": res,
	}, metrics.Build())
	return message, metrics.Build(), nil
}

func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(string, *protos.Message) error,
	onMetrics func(string, *protos.Message, []*protos.Metric) error,
	onError func(string, error),
) error {
	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	requestBody := llc.buildRequestBody(allMessages, options)
	requestBody["stream"] = true
	options.PreHook(requestBody)

	resp, err := llc.Call(ctx, "chat/completions", "POST", map[string]string{}, requestBody)
	if err != nil {
		llc.logger.Errorf("minimax stream chat completion request failed: %v", err)
		onError(options.Request.GetRequestId(), err)
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Errorf("minimax stream returned status %d", resp.StatusCode)
		onError(options.Request.GetRequestId(), errMsg)
		options.PostHook(map[string]interface{}{
			"error": errMsg,
		}, metrics.OnFailure().Build())
		return errMsg
	}

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	contentBuffer := make([]string, 0)
	hasToolCalls := false

	// Track tool call accumulation by index
	toolCallMap := map[int]*protos.ToolCall{}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk MiniMaxStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			llc.logger.Warnf("failed to parse minimax stream chunk: %v", err)
			continue
		}

		// Capture final usage from the last chunk
		if chunk.Usage != nil {
			metrics.OnAddMetrics(llc.UsageMetrics(chunk.Usage)...)
		}

		for i, choice := range chunk.Choices {
			// Accumulate tool calls
			for _, tc := range choice.Delta.ToolCalls {
				hasToolCalls = true
				existing, ok := toolCallMap[tc.Index]
				if !ok {
					existing = &protos.ToolCall{
						Id:   tc.ID,
						Type: tc.Type,
						Function: &protos.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
					toolCallMap[tc.Index] = existing
				} else {
					if tc.ID != "" {
						existing.Id = tc.ID
					}
					if tc.Function.Name != "" {
						existing.Function.Name += tc.Function.Name
					}
					existing.Function.Arguments += tc.Function.Arguments
				}
			}

			content := choice.Delta.Content
			if content != "" {
				if len(contentBuffer) <= i {
					contentBuffer = append(contentBuffer, content)
				} else {
					contentBuffer[i] += content
				}

				if !hasToolCalls {
					if firstTokenTime == nil {
						now := time.Now()
						firstTokenTime = &now
					}
					tokenMsg := &protos.Message{
						Role: "assistant",
						Message: &protos.Message_Assistant{
							Assistant: &protos.AssistantMessage{
								Contents: []string{content},
							},
						},
					}
					if err := onStream(options.Request.GetRequestId(), tokenMsg); err != nil {
						llc.logger.Warnf("error streaming token: %v", err)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		llc.logger.Errorf("error reading minimax stream: %v", err)
		onError(options.Request.GetRequestId(), err)
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return err
	}

	// Strip thinking tags from accumulated content
	for i, c := range contentBuffer {
		contentBuffer[i] = stripThinkingTags(c)
	}
	assistantMsg.Contents = contentBuffer

	// Collect accumulated tool calls
	for _, tc := range toolCallMap {
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
	}

	protoMsg := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        "FIRST_TOKEN_RECIEVED_TIME",
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	metrics.OnSuccess()

	onMetrics(options.Request.GetRequestId(), protoMsg, metrics.Build())
	options.PostHook(map[string]interface{}{
		"result": contentBuffer,
	}, metrics.Build())

	return nil
}

func (llc *largeLanguageCaller) GetCompletion(
	ctx context.Context,
	providerModel string,
	prompts []string,
	options *internal_callers.CompletionOptions,
) ([]string, []*protos.Metric, error) {
	llc.logger.Debugf("getting completion for minimax")
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	return nil, metrics.OnFailure().Build(), errors.New("text completion not supported by MiniMax; use chat completion instead")
}

// buildMessages converts proto messages to the OpenAI-compatible format used by MiniMax.
func buildMessages(allMessages []*protos.Message) []map[string]interface{} {
	msg := make([]map[string]interface{}, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case "user":
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, map[string]interface{}{
					"role":    "user",
					"content": user.GetContent(),
				})
			}
		case "assistant":
			if assistant := cntn.GetAssistant(); assistant != nil {
				entry := map[string]interface{}{
					"role":    "assistant",
					"content": strings.Join(assistant.GetContents(), ""),
				}
				if toolCalls := assistant.GetToolCalls(); len(toolCalls) > 0 {
					tcs := make([]map[string]interface{}, 0, len(toolCalls))
					for _, tc := range toolCalls {
						tcs = append(tcs, map[string]interface{}{
							"id":   tc.GetId(),
							"type": "function",
							"function": map[string]string{
								"name":      tc.GetFunction().GetName(),
								"arguments": tc.GetFunction().GetArguments(),
							},
						})
					}
					entry["tool_calls"] = tcs
				}
				msg = append(msg, entry)
			}
		case "system":
			if system := cntn.GetSystem(); system != nil {
				msg = append(msg, map[string]interface{}{
					"role":    "system",
					"content": system.GetContent(),
				})
			}
		case "tool":
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msg = append(msg, map[string]interface{}{
						"role":            "tool",
						"content":         t.GetContent(),
						"tool_call_id":    t.GetId(),
					})
				}
			}
		}
	}
	return msg
}

// stripThinkingTags removes <think>...</think> blocks from MiniMax model responses.
func stripThinkingTags(content string) string {
	for {
		start := strings.Index(content, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(content, "</think>")
		if end == -1 {
			// Incomplete think tag — remove from <think> to end
			content = content[:start]
			break
		}
		content = content[:start] + content[end+len("</think>"):]
	}
	return strings.TrimSpace(content)
}
