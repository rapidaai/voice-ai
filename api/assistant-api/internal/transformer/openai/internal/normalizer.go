// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_openai

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// OpenAI Text Processor
// =============================================================================

// openaiProcessor handles OpenAI TTS text preprocessing.
// OpenAI TTS does NOT support SSML - only plain text is accepted.
type openaiProcessor struct {
	logger   commons.Logger
	language string
}

// NewOpenAIProcessor creates an OpenAI-specific text processor.
func NewOpenAIProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &openaiProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. OpenAI TTS does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *openaiProcessor) Process(text string) string {
	return text
}
