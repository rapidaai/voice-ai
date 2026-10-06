// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package deepgram_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Deepgram Text Processor
// =============================================================================

// deepgramProcessor handles Deepgram TTS text preprocessing.
// Deepgram does NOT support SSML - only plain text is accepted.
type deepgramProcessor struct {
	logger   commons.Logger
	language string
}

// NewDeepgramProcessor creates a Deepgram-specific text processor.
func NewDeepgramProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &deepgramProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Deepgram does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *deepgramProcessor) Process(text string) string {
	return text
}
