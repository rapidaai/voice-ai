// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package elevenlabs_internal

import (
	"fmt"
	"regexp"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// ElevenLabs Text Processor
// =============================================================================

// elevenlabsProcessor handles ElevenLabs TTS text preprocessing.
// ElevenLabs supports LIMITED SSML: only <break> and <phoneme> tags.
// Break time is specified in SECONDS (e.g., time="0.5s" not "500ms").
type elevenlabsProcessor struct {
	logger   commons.Logger
	config   internal_type.ProcessorConfig
	language string

	// conjunction handling
	conjunctionPattern *regexp.Regexp
}

// NewElevenLabsProcessor creates an ElevenLabs-specific text processor.
func NewElevenLabsProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	cfg := internal_type.DefaultProcessorConfig()

	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	// Parse conjunction boundaries from options
	var conjunctionPattern *regexp.Regexp
	if conjunctionBoundaries, err := opts.GetString("speaker.conjunction.boundaries"); err == nil && conjunctionBoundaries != "" {
		cfg.Conjunctions = strings.Split(conjunctionBoundaries, commons.SEPARATOR)

		escaped := make([]string, len(cfg.Conjunctions))
		for i, c := range cfg.Conjunctions {
			escaped[i] = regexp.QuoteMeta(strings.TrimSpace(c))
		}
		pattern := `(` + strings.Join(escaped, "|") + `)`
		conjunctionPattern = regexp.MustCompile(pattern)
	}

	// Parse conjunction break duration
	if conjunctionBreak, err := opts.GetUint64("speaker.conjunction.break"); err == nil {
		cfg.PauseDurationMs = conjunctionBreak
	}

	return &elevenlabsProcessor{
		logger:             logger,
		config:             cfg,
		language:           language,
		conjunctionPattern: conjunctionPattern,
	}
}

// Process applies ElevenLabs-specific text transformations.
// ElevenLabs supports only <break> and <phoneme> SSML tags.
// Markdown removal and whitespace processing are handled upstream.
func (n *elevenlabsProcessor) Process(text string) string {
	if text == "" {
		return text
	}

	// ElevenLabs supports limited SSML, so we escape XML characters
	// except where we insert our own SSML tags
	text = n.escapeXML(text)

	// Insert breaks after conjunction boundaries (ElevenLabs uses seconds)
	if n.conjunctionPattern != nil && n.config.PauseDurationMs > 0 {
		text = n.insertConjunctionBreaks(text)
	}

	return text
}

// =============================================================================
// Private Helpers
// =============================================================================

// escapeXML escapes XML special characters for limited SSML safety.
func (n *elevenlabsProcessor) escapeXML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}

// insertConjunctionBreaks adds breaks after conjunctions.
// ElevenLabs uses seconds format (e.g., "0.5s" instead of "500ms").
func (n *elevenlabsProcessor) insertConjunctionBreaks(text string) string {
	// Convert milliseconds to seconds for ElevenLabs
	seconds := float64(n.config.PauseDurationMs) / 1000.0
	breakTag := fmt.Sprintf(`<break time="%.2fs"/>`, seconds)

	return n.conjunctionPattern.ReplaceAllStringFunc(text, func(match string) string {
		return match + breakTag
	})
}

// =============================================================================
// ElevenLabs SSML Helpers (Limited Support)
// =============================================================================

// AddBreak adds a pause. ElevenLabs uses seconds format.
func (n *elevenlabsProcessor) AddBreak(durationMs int) string {
	seconds := float64(durationMs) / 1000.0
	return fmt.Sprintf(`<break time="%.2fs"/>`, seconds)
}

// AddPhoneme wraps text with phoneme pronunciation.
// ElevenLabs supports IPA phoneme alphabet.
func (n *elevenlabsProcessor) AddPhoneme(text, ipa string) string {
	return fmt.Sprintf(`<phoneme alphabet="ipa" ph="%s">%s</phoneme>`, ipa, text)
}
