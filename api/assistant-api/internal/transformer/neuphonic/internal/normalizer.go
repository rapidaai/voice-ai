// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package neuphonic_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// neuphonicProcessor handles NeuPhonic TTS text preprocessing.
// NeuPhonic does NOT support SSML - only plain text is accepted.
type neuphonicProcessor struct {
	logger commons.Logger
}

// NewNeuPhonicProcessor creates a NeuPhonic-specific text processor.
func NewNeuPhonicProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	return &neuphonicProcessor{
		logger: logger,
	}
}

// Process returns text unchanged. NeuPhonic does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *neuphonicProcessor) Process(text string) string {
	return text
}
