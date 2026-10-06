// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package groq_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// groqProcessor handles Groq TTS text preprocessing.
// Groq does NOT support SSML - only plain text is accepted.
type groqProcessor struct {
	logger commons.Logger
}

// NewGroqProcessor creates a Groq-specific text processor.
func NewGroqProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	return &groqProcessor{
		logger: logger,
	}
}

// Process returns text unchanged. Groq does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *groqProcessor) Process(text string) string {
	return text
}
