// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_messages

import (
	"testing"

	caller_testutil "github.com/rapidaai/api/integration-api/internal/caller/internal/testutil"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildHistory_UserMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello"},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	assert.Empty(t, system, "no system prompt expected")
	require.Len(t, history, 1)
	assert.Equal(t, "user", string(history[0].Role))
	require.Len(t, history[0].Content, 1)
	assert.Equal(t, "Hello", history[0].Content[0].OfText.Text)
}

func TestBuildHistory_SystemMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "system",
			Message: &protos.Message_System{
				System: &protos.SystemMessage{Content: "You are helpful"},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	require.Len(t, system, 1)
	assert.Equal(t, "You are helpful", system[0].Text)
	assert.Empty(t, history, "system messages should not be in history")
}

func TestBuildHistory_AssistantWithContent(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Hello!", "How can I help?"},
				},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	assert.Empty(t, system)
	require.Len(t, history, 1)
	assert.Equal(t, "assistant", string(history[0].Role))
	require.Len(t, history[0].Content, 2)
	assert.Equal(t, "Hello!", history[0].Content[0].OfText.Text)
	assert.Equal(t, "How can I help?", history[0].Content[1].OfText.Text)
}

func TestBuildHistory_AssistantWithToolCall(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Let me check"},
					ToolCalls: []*protos.ToolCall{
						{
							Id:   "call_1",
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"city":"NYC"}`,
							},
						},
					},
				},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	assert.Empty(t, system)
	require.Len(t, history, 1)
	require.Len(t, history[0].Content, 2)
	assert.Equal(t, "Let me check", history[0].Content[0].OfText.Text)
	assert.Equal(t, "call_1", history[0].Content[1].OfToolUse.ID)
	assert.Equal(t, "get_weather", history[0].Content[1].OfToolUse.Name)
}

func TestBuildHistory_ToolMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{
							Id:      "call_1",
							Name:    "get_weather",
							Content: `{"temp": 72}`,
						},
					},
				},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	assert.Empty(t, system)
	require.Len(t, history, 1)
	assert.Equal(t, "user", string(history[0].Role))
	require.Len(t, history[0].Content, 1)
	assert.Equal(t, "call_1", history[0].Content[0].OfToolResult.ToolUseID)
}

func TestBuildHistory_MixedMessages(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role:    "system",
			Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be brief"}},
		},
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hi"}},
		},
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Hello!"}},
			},
		},
	}

	system, history := buildHistory(newTestLogger(), msgs)
	require.Len(t, system, 1)
	assert.Equal(t, "Be brief", system[0].Text)
	require.Len(t, history, 2)
	assert.Equal(t, "user", string(history[0].Role))
	assert.Equal(t, "assistant", string(history[1].Role))
}

func TestBuildHistory_EmptyMessages(t *testing.T) {
	system, history := buildHistory(newTestLogger(), []*protos.Message{})
	assert.Empty(t, system)
	assert.Empty(t, history)
}

func TestBuildHistory_EmptyUserContent(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "  "}},
		},
	}

	_, history := buildHistory(newTestLogger(), msgs)
	assert.Empty(t, history, "whitespace-only user messages should be skipped")
}

func TestBuildHistory_InvalidToolCallJSON(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"text"},
					ToolCalls: []*protos.ToolCall{
						{
							Id:   "call_bad",
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      "fn",
								Arguments: "invalid json {{{",
							},
						},
					},
				},
			},
		},
	}

	_, history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 1)
	require.Len(t, history[0].Content, 1)
	assert.Equal(t, "text", history[0].Content[0].OfText.Text)
}

func TestBuildMessageParamsDropsSamplingForClaudeFive(t *testing.T) {
	modelParams := caller_testutil.BuildModelParameters(map[string]interface{}{
		"model.name":        "claude-sonnet-5-5",
		"model.max_tokens":  int64(1028),
		"model.temperature": 0.3,
		"model.top_k":       int64(5),
		"model.top_p":       0.8,
	})

	params := buildMessageParams(modelParams, nil)

	assert.Equal(t, "claude-sonnet-5-5", string(params.Model))
	assert.Equal(t, int64(1028), params.MaxTokens)
	assert.False(t, params.Temperature.Valid())
	assert.False(t, params.TopK.Valid())
	assert.False(t, params.TopP.Valid())
}

func TestBuildMessageParamsKeepsSamplingForOlderClaude(t *testing.T) {
	modelParams := caller_testutil.BuildModelParameters(map[string]interface{}{
		"model.name":        "claude-3-5-haiku-20241022",
		"model.temperature": 0.3,
		"model.top_k":       int64(5),
		"model.top_p":       0.8,
	})

	params := buildMessageParams(modelParams, nil)

	assert.Equal(t, "claude-3-5-haiku-20241022", string(params.Model))
	require.True(t, params.Temperature.Valid())
	require.True(t, params.TopK.Valid())
	require.True(t, params.TopP.Valid())
	assert.Equal(t, 0.3, params.Temperature.Value)
	assert.Equal(t, int64(5), params.TopK.Value)
	assert.Equal(t, 0.8, params.TopP.Value)
}

func TestBuildMessageParamsMapsRetiredClaudeSonnetFour(t *testing.T) {
	modelParams := caller_testutil.BuildModelParameters(map[string]interface{}{
		"model.name":        "claude-sonnet-4-20250514",
		"model.temperature": 0.3,
		"model.top_k":       int64(5),
		"model.top_p":       0.8,
	})

	params := buildMessageParams(modelParams, nil)

	assert.Equal(t, "claude-sonnet-4-6", string(params.Model))
	require.True(t, params.Temperature.Valid())
	require.True(t, params.TopK.Valid())
	require.True(t, params.TopP.Valid())
}

func TestBuildMessageParamsMapsRetiredClaudeSonnetThreeSeven(t *testing.T) {
	modelParams := caller_testutil.BuildModelParameters(map[string]interface{}{
		"model.name":        "claude-3-7-sonnet-20250219",
		"model.temperature": 0.3,
		"model.top_k":       int64(5),
		"model.top_p":       0.8,
	})

	params := buildMessageParams(modelParams, nil)

	assert.Equal(t, "claude-sonnet-4-6", string(params.Model))
	require.True(t, params.Temperature.Valid())
	require.True(t, params.TopK.Valid())
	require.True(t, params.TopP.Valid())
}

func TestBuildMessageParamsMapsRetiredClaudeOpusFour(t *testing.T) {
	modelParams := caller_testutil.BuildModelParameters(map[string]interface{}{
		"model.name":        "claude-opus-4-20250514",
		"model.temperature": 0.3,
		"model.top_k":       int64(5),
		"model.top_p":       0.8,
	})

	params := buildMessageParams(modelParams, nil)

	assert.Equal(t, "claude-opus-4-8", string(params.Model))
	assert.False(t, params.Temperature.Valid())
	assert.False(t, params.TopK.Valid())
	assert.False(t, params.TopP.Valid())
}
