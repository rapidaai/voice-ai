// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package nvidia_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// nvidiaProcessor handles Nvidia TTS text preprocessing.
// Nvidia does NOT support SSML - only plain text is accepted.
type nvidiaProcessor struct {
	logger commons.Logger
}

// NewNvidiaProcessor creates an Nvidia-specific text processor.
func NewNvidiaProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	return &nvidiaProcessor{
		logger: logger,
	}
}

// Process returns text unchanged. Nvidia does NOT support SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *nvidiaProcessor) Process(text string) string {
	return text
}
