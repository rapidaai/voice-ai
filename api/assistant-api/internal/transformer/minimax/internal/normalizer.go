// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package minimax_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// minimaxProcessor handles MiniMax TTS text preprocessing.
// MiniMax does NOT support SSML - only plain text is accepted.
type minimaxProcessor struct {
	logger commons.Logger
}

// NewMiniMaxProcessor creates a MiniMax-specific text processor.
func NewMiniMaxProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	return &minimaxProcessor{
		logger: logger,
	}
}

// Process returns text unchanged. MiniMax does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *minimaxProcessor) Process(text string) string {
	return text
}
