// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package speechmatics_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Speechmatics Text Processor
// =============================================================================

// speechmaticsProcessor handles Speechmatics text preprocessing.
// Speechmatics is primarily an STT service, but this processor handles any TTS needs.
// Speechmatics does NOT support SSML - only plain text is accepted.
type speechmaticsProcessor struct {
	logger   commons.Logger
	language string
}

// NewSpeechmaticsProcessor creates a Speechmatics-specific text processor.
func NewSpeechmaticsProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &speechmaticsProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Speechmatics does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *speechmaticsProcessor) Process(text string) string {
	return text
}
