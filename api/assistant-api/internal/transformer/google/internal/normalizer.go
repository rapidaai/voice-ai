// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package google_internal

import (
	"fmt"
	"regexp"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Google Text Processor
// =============================================================================
// ProcessorConfig holds SSML conjunction break configuration for providers
// that support pauses (Google, Azure, AWS, ElevenLabs, Rime).
type ProcessorConfig struct {
	Conjunctions    []string
	PauseDurationMs uint64
}

func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		PauseDurationMs: 240,
	}
}

// googleProcessor handles Google Cloud TTS text preprocessing.
// Google supports standard W3C SSML with some Google-specific extensions.
type googleProcessor struct {
	logger             commons.Logger
	config             ProcessorConfig
	conjunctionPattern *regexp.Regexp
}

// NewGoogleProcessor creates a Google-specific text processor.
func NewGoogleProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	cfg := DefaultProcessorConfig()
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

	if conjunctionBreak, err := opts.GetUint64("speaker.conjunction.break"); err == nil {
		cfg.PauseDurationMs = conjunctionBreak
	}
	return &googleProcessor{
		logger:             logger,
		config:             cfg,
		conjunctionPattern: conjunctionPattern,
	}
}

// Process applies Google-specific text transformations.
// Markdown removal and whitespace processing are handled upstream.
func (n *googleProcessor) Process(text string) string {
	if text == "" {
		return text
	}

	// Escape XML special characters for SSML safety (Google uses SSML)
	text = n.escapeXML(text)

	// Insert breaks after conjunction boundaries
	if n.conjunctionPattern != nil && n.config.PauseDurationMs > 0 {
		text = n.insertConjunctionBreaks(text)
	}

	return text
}

// =============================================================================
// Private Helpers
// =============================================================================

// escapeXML escapes XML special characters for SSML (Google uses fewer escapes).
func (n *googleProcessor) escapeXML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}

func (n *googleProcessor) insertConjunctionBreaks(text string) string {
	breakTag := fmt.Sprintf(`<break time="%dms"/>`, n.config.PauseDurationMs)
	return n.conjunctionPattern.ReplaceAllStringFunc(text, func(match string) string {
		return match + breakTag
	})
}

// =============================================================================
// Google SSML Helpers
// =============================================================================

func (n *googleProcessor) WrapWithSSML(text string) string {
	return fmt.Sprintf(`<speak>%s</speak>`, text)
}

func (n *googleProcessor) AddBreak(durationMs int) string {
	return fmt.Sprintf(`<break time="%dms"/>`, durationMs)
}

func (n *googleProcessor) AddProsody(text string, rate, pitch, volume string) string {
	attrs := ""
	if rate != "" {
		attrs += fmt.Sprintf(` rate="%s"`, rate)
	}
	if pitch != "" {
		attrs += fmt.Sprintf(` pitch="%s"`, pitch)
	}
	if volume != "" {
		attrs += fmt.Sprintf(` volume="%s"`, volume)
	}
	if attrs == "" {
		return text
	}
	return fmt.Sprintf(`<prosody%s>%s</prosody>`, attrs, text)
}

func (n *googleProcessor) AddEmphasis(text, level string) string {
	return fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, level, text)
}

func (n *googleProcessor) SayAs(text, interpretAs, format string) string {
	if format != "" {
		return fmt.Sprintf(`<say-as interpret-as="%s" format="%s">%s</say-as>`, interpretAs, format, text)
	}
	return fmt.Sprintf(`<say-as interpret-as="%s">%s</say-as>`, interpretAs, text)
}

func (n *googleProcessor) AddAudio(src string, altText string) string {
	if altText != "" {
		return fmt.Sprintf(`<audio src="%s">%s</audio>`, src, altText)
	}
	return fmt.Sprintf(`<audio src="%s"/>`, src)
}
