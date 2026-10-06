// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package revai_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Rev AI Text Processor
// =============================================================================

// revaiProcessor handles Rev AI text preprocessing.
// Rev AI is primarily an STT service, but this processor handles any TTS needs.
// Rev AI does NOT support SSML - only plain text is accepted.
type revaiProcessor struct {
	logger   commons.Logger
	language string
}

// NewRevAIProcessor creates a Rev AI-specific text processor.
func NewRevAIProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &revaiProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Rev AI does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *revaiProcessor) Process(text string) string {
	return text
}
