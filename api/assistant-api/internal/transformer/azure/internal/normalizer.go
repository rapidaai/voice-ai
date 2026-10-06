// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package azure_internal

import (
	"fmt"
	"regexp"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Azure Text Processor
// =============================================================================

// azureProcessor handles Azure Cognitive Services TTS text preprocessing.
// Azure supports full SSML with mstts extensions for expressive speech.
type azureProcessor struct {
	logger    commons.Logger
	config    internal_type.ProcessorConfig
	voiceName string
	language  string

	// conjunction handling
	conjunctionPattern *regexp.Regexp
}

// NewAzureProcessor creates an Azure-specific text processor.
func NewAzureProcessor(logger commons.Logger, opts utils.Option) internal_type.TextProcessor {
	cfg := internal_type.DefaultProcessorConfig()

	// Get voice name and language
	voiceName, _ := opts.GetString("speaker.voice.name")
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en-US"
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

	return &azureProcessor{
		logger:             logger,
		config:             cfg,
		voiceName:          voiceName,
		language:           language,
		conjunctionPattern: conjunctionPattern,
	}
}

// Process applies Azure-specific text transformations.
// Markdown removal and whitespace processing are handled upstream.
func (n *azureProcessor) Process(text string) string {
	if text == "" {
		return text
	}

	// Escape XML special characters for SSML safety (Azure uses SSML)
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

func (n *azureProcessor) escapeXML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(text)
}

func (n *azureProcessor) insertConjunctionBreaks(text string) string {
	breakTag := fmt.Sprintf(`<break time="%dms"/>`, n.config.PauseDurationMs)
	return n.conjunctionPattern.ReplaceAllStringFunc(text, func(match string) string {
		return match + breakTag
	})
}

// =============================================================================
// Azure SSML Helpers
// =============================================================================

func (n *azureProcessor) WrapWithSSML(text string) string {
	return fmt.Sprintf(
		`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="%s"><voice name="%s">%s</voice></speak>`,
		n.language, n.voiceName, text,
	)
}

func (n *azureProcessor) AddBreak(durationMs int) string {
	return fmt.Sprintf(`<break time="%dms"/>`, durationMs)
}

func (n *azureProcessor) AddProsody(text string, rate, pitch, volume string) string {
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

func (n *azureProcessor) AddEmphasis(text, level string) string {
	return fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, level, text)
}

func (n *azureProcessor) AddExpressAs(text, style string) string {
	return fmt.Sprintf(`<mstts:express-as style="%s">%s</mstts:express-as>`, style, text)
}

func (n *azureProcessor) SayAs(text, interpretAs, format string) string {
	if format != "" {
		return fmt.Sprintf(`<say-as interpret-as="%s" format="%s">%s</say-as>`, interpretAs, format, text)
	}
	return fmt.Sprintf(`<say-as interpret-as="%s">%s</say-as>`, interpretAs, text)
}
