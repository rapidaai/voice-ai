// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package cartesia_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Cartesia Text Processor
// =============================================================================

// cartesiaProcessor handles Cartesia TTS text preprocessing.
// Cartesia does NOT support SSML - only plain text is accepted.
type cartesiaProcessor struct {
	logger   commons.Logger
	language string
}

// NewCartesiaProcessor creates a Cartesia-specific text processor.
func NewCartesiaProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &cartesiaProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Cartesia does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *cartesiaProcessor) Process(text string) string {
	return text
}
