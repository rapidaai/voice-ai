// Rapida -- Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_minimax_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestMessageBuilding_UserAndAssistant(t *testing.T) {
	msgs := []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hello"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Hi there"}}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "How are you?"}}},
	}

	result := buildMessages(msgs)
	require.Len(t, result, 3)
	assert.Equal(t, "user", result[0]["role"])
	assert.Equal(t, "Hello", result[0]["content"])
	assert.Equal(t, "assistant", result[1]["role"])
	assert.Equal(t, "Hi there", result[1]["content"])
	assert.Equal(t, "user", result[2]["role"])
	assert.Equal(t, "How are you?", result[2]["content"])
}

func TestMessageBuilding_SystemMessage(t *testing.T) {
	msgs := []*protos.Message{
		{Role: "system", Message: &protos.Message_System{System: &protos.SystemMessage{Content: "You are helpful"}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hi"}}},
	}

	result := buildMessages(msgs)
	require.Len(t, result, 2)
	assert.Equal(t, "system", result[0]["role"])
	assert.Equal(t, "You are helpful", result[0]["content"])
	assert.Equal(t, "user", result[1]["role"])
}

func TestMessageBuilding_ToolMessages(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{""},
					ToolCalls: []*protos.ToolCall{
						{
							Id:   "call_123",
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"NYC"}`,
							},
						},
					},
				},
			},
		},
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolContent{
						{Id: "call_123", Content: `{"temp": 72}`},
					},
				},
			},
		},
	}

	result := buildMessages(msgs)
	require.Len(t, result, 2)

	// Check assistant message with tool calls
	assert.Equal(t, "assistant", result[0]["role"])
	toolCalls, ok := result[0]["tool_calls"]
	require.True(t, ok, "assistant message should have tool_calls")
	tcSlice, ok := toolCalls.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, tcSlice, 1)
	assert.Equal(t, "call_123", tcSlice[0]["id"])

	// Check tool message
	assert.Equal(t, "tool", result[1]["role"])
	assert.Equal(t, `{"temp": 72}`, result[1]["content"])
	assert.Equal(t, "call_123", result[1]["tool_call_id"])
}

func TestMessageBuilding_Empty(t *testing.T) {
	result := buildMessages([]*protos.Message{})
	assert.Empty(t, result)
}

func TestStripThinkingTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no thinking tags",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "simple thinking tag",
			input:    "<think>reasoning here</think>The answer is 42.",
			expected: "The answer is 42.",
		},
		{
			name:     "multiple thinking tags",
			input:    "<think>first</think>Hello <think>second</think>world",
			expected: "Hello world",
		},
		{
			name:     "incomplete thinking tag",
			input:    "<think>never closed",
			expected: "",
		},
		{
			name:     "empty content after stripping",
			input:    "<think>only thinking</think>",
			expected: "",
		},
		{
			name:     "thinking with whitespace",
			input:    "  <think>reasoning</think>  Result  ",
			expected: "Result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripThinkingTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampTemperature(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"zero", 0, 0.01},
		{"negative", -0.5, 0.01},
		{"normal", 0.7, 0.7},
		{"max", 1.0, 1.0},
		{"over max", 1.5, 1.0},
		{"min valid", 0.01, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampTemperature(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestUsageMetrics(t *testing.T) {
	mm := &MiniMax{logger: newTestLogger()}

	t.Run("nil usage", func(t *testing.T) {
		metrics := mm.UsageMetrics(nil)
		assert.Empty(t, metrics)
	})

	t.Run("valid usage", func(t *testing.T) {
		usage := &MiniMaxUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		}
		metrics := mm.UsageMetrics(usage)
		require.Len(t, metrics, 3)

		metricMap := map[string]string{}
		for _, m := range metrics {
			metricMap[m.Name] = m.Value
		}
		assert.Equal(t, "50", metricMap["OUTPUT_TOKEN"])
		assert.Equal(t, "100", metricMap["INPUT_TOKEN"])
		assert.Equal(t, "150", metricMap["TOTAL_TOKEN"])
	})
}

func TestEndpoint(t *testing.T) {
	mm := &MiniMax{logger: newTestLogger()}

	result := mm.Endpoint("chat/completions")
	assert.Equal(t, "https://api.minimax.io/v1/chat/completions", result)
}

func TestGetCompletion_ReturnsError(t *testing.T) {
	caller := &largeLanguageCaller{
		MiniMax: MiniMax{logger: newTestLogger()},
	}

	_, metrics, err := caller.GetCompletion(nil, "model", []string{"prompt"}, &internal_callers.CompletionOptions{})
	assert.Error(t, err, "GetCompletion should return error for unsupported operation")
	assert.Contains(t, err.Error(), "not supported")
	assert.NotNil(t, metrics)
}

func TestMiniMaxErrorString(t *testing.T) {
	t.Run("with error detail", func(t *testing.T) {
		mmErr := MiniMaxError{
			Error: &struct {
				Message string `json:"message,omitempty"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: "invalid api key",
				Type:    "authentication_error",
				Code:    "invalid_api_key",
			},
			StatusCode: 401,
		}
		assert.Contains(t, mmErr.ErrorString(), "invalid api key")
		assert.Contains(t, mmErr.ErrorString(), "authentication_error")
	})

	t.Run("without error detail", func(t *testing.T) {
		mmErr := MiniMaxError{StatusCode: 500}
		assert.Contains(t, mmErr.ErrorString(), "500")
	})
}
