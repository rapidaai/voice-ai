// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package smallest_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Smallest AI Text Processor
// =============================================================================

// smallestProcessor handles Lightning TTS text preprocessing.
// Lightning does NOT support SSML - only plain text is accepted.
type smallestProcessor struct {
	logger   commons.Logger
	language string
}

// NewSmallestProcessor creates a Smallest-specific text processor.
func NewSmallestProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &smallestProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Lightning does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *smallestProcessor) Process(text string) string {
	return text
}
