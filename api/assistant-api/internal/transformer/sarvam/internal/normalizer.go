// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sarvam_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Sarvam Text Processor
// =============================================================================

// sarvamProcessor handles Sarvam AI TTS text preprocessing.
// Sarvam does NOT support SSML - only plain text is accepted.
// Sarvam specializes in Indian languages (Hindi, Tamil, Telugu, etc.).
type sarvamProcessor struct {
	logger   commons.Logger
	language string
}

// NewSarvamProcessor creates a Sarvam-specific text processor.
func NewSarvamProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "hi-IN" // Default to Hindi
	}

	return &sarvamProcessor{
		logger:   logger,
		language: language,
	}
}

// Process returns text unchanged. Sarvam does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *sarvamProcessor) Process(text string) string {
	return text
}
