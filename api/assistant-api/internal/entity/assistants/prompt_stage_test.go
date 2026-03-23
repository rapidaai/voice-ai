// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssistantPromptStage_SetAndGetPrompt(t *testing.T) {
	stage := &AssistantPromptStage{}

	promptJSON := `{
		"text_chat_complete": {
			"prompt": [
				{"role": "system", "content": "You are a helpful assistant in stage {{stage}}"}
			],
			"variables": [
				{"name": "stage", "type": "string", "default": "default"}
			]
		}
	}`

	stage.SetPrompt(promptJSON)

	template := stage.GetTemplate()
	assert.NotNil(t, template, "Template should not be nil")
	assert.Contains(t, template, "text_chat_complete", "Template should contain text_chat_complete")
}

func TestAssistantPromptStage_SetPrompt_InvalidJSON(t *testing.T) {
	stage := &AssistantPromptStage{}

	stage.SetPrompt("not valid json")

	template := stage.GetTemplate()
	assert.Nil(t, template, "Template should be nil for invalid JSON")
}
