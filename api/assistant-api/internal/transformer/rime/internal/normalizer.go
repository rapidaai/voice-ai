// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package rime_internal

import (
	"fmt"
	"regexp"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// rimeProcessor handles Rime TTS text preprocessing.
// Rime does NOT support SSML. Custom pauses use <ms> syntax (e.g., <500> for 500ms pause).
type rimeProcessor struct {
	logger   commons.Logger
	config   internal_type.ProcessorConfig
	language string

	// conjunctionPattern is instance-level: compiled from user-configured boundaries.
	conjunctionPattern *regexp.Regexp
}

// NewRimeProcessor creates a Rime-specific text processor.
func NewRimeProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	cfg := internal_type.DefaultProcessorConfig()

	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "eng"
	}

	var conjunctionPattern *regexp.Regexp
	if conjunctionBoundaries, err := opts.GetString("speaker.conjunction.boundaries"); err == nil && conjunctionBoundaries != "" {
		cfg.Conjunctions = strings.Split(conjunctionBoundaries, commons.SEPARATOR)
		escaped := make([]string, len(cfg.Conjunctions))
		for i, c := range cfg.Conjunctions {
			escaped[i] = regexp.QuoteMeta(strings.TrimSpace(c))
		}
		conjunctionPattern = regexp.MustCompile(`(` + strings.Join(escaped, "|") + `)`)
	}

	if conjunctionBreak, err := opts.GetUint64("speaker.conjunction.break"); err == nil {
		cfg.PauseDurationMs = conjunctionBreak
	}

	return &rimeProcessor{
		logger:             logger,
		config:             cfg,
		language:           language,
		conjunctionPattern: conjunctionPattern,
	}
}

// Process applies Rime-specific text transformations.
// Rime uses <ms> syntax for pauses instead of SSML.
// Markdown removal and whitespace processing are handled upstream.
func (n *rimeProcessor) Process(text string) string {
	if text == "" {
		return text
	}

	if n.conjunctionPattern != nil && n.config.PauseDurationMs > 0 {
		text = n.insertConjunctionBreaks(text)
	}

	return text
}

// =============================================================================
// Private Helpers
// =============================================================================

// insertConjunctionBreaks adds pauses after conjunctions using Rime's <ms> syntax.
func (n *rimeProcessor) insertConjunctionBreaks(text string) string {
	pauseTag := fmt.Sprintf(" <%d> ", n.config.PauseDurationMs)
	return n.conjunctionPattern.ReplaceAllStringFunc(text, func(match string) string {
		return match + pauseTag
	})
}
