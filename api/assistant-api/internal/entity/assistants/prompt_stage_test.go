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
		"prompt": [
			{"role": "system", "content": "You are a helpful assistant in stage {{stage}}"}
		],
		"promptVariables": [
			{"name": "stage", "type": "string", "defaultValue": "default"}
		]
	}`

	err := stage.SetPrompt(promptJSON)
	assert.NoError(t, err, "SetPrompt should not return error for valid JSON")

	template := stage.GetTemplate()
	assert.NotNil(t, template, "Template should not be nil")
	assert.Contains(t, template, "prompt", "Template should contain prompt")
}

func TestAssistantPromptStage_SetPrompt_InvalidJSON(t *testing.T) {
	stage := &AssistantPromptStage{}

	err := stage.SetPrompt("not valid json")
	assert.Error(t, err, "SetPrompt should return error for invalid JSON")

	template := stage.GetTemplate()
	assert.Nil(t, template, "Template should be nil for invalid JSON")
}
