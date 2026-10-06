// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resembleai_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// resembleaiProcessor handles ResembleAI TTS text preprocessing.
// ResembleAI does NOT support SSML - only plain text is accepted.
type resembleaiProcessor struct {
	logger commons.Logger
}

// NewResembleAIProcessor creates a ResembleAI-specific text processor.
func NewResembleAIProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	return &resembleaiProcessor{
		logger: logger,
	}
}

// Process returns text unchanged. ResembleAI does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *resembleaiProcessor) Process(text string) string {
	return text
}
