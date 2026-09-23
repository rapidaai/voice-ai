// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_normalizers

import (
	"context"
	"regexp"
	"strings"

	internal_output_aggregator_normalizer "github.com/rapidaai/api/assistant-api/internal/normalizer/output/aggregator"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/variable"
	internal_namespace "github.com/rapidaai/api/assistant-api/internal/variable/namespace"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/parsers"
	"github.com/rapidaai/protos"
)

var (
	markdownBlock   = regexp.MustCompile("(?s)```[^`]*```")
	markdownInline  = regexp.MustCompile("`([^`]+)`")
	markdownHeading = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	markdownEmph    = regexp.MustCompile(`\*{1,2}([^*]+?)\*{1,2}|_{1,2}([^_]+?)_{1,2}`)
	markdownLink    = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	markdownImage   = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	markdownQuote   = regexp.MustCompile(`(?m)^>\s?`)
	markdownHr      = regexp.MustCompile(`(?m)^(-{3,}|\*{3,}|_{3,})$`)
	markdownStars   = regexp.MustCompile(`[*]+`)
	wordUnderscore  = regexp.MustCompile(`(\w)_(\w)`)
	emojiPattern    = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{200D}\x{20E3}\x{FE0F}]+`)
	whitespaceRun   = regexp.MustCompile(`\s+`)
)

type outputNormalizer struct {
	logger         commons.Logger
	aggregator     internal_type.LLMTextAggregator
	normalizers    []internal_type.TextNormalizer
	expandArgs     func() map[string]interface{}
	templateParser parsers.StringTemplateParser
	onPacket       func(context.Context, ...internal_type.Packet) error
}

// NewOutputNormalizer creates an output normalizer with an internal aggregator
// and text normalizer chain built from assistant options.
func NewOutputNormalizer(logger commons.Logger) internal_type.PacketNormalizer {
	return &outputNormalizer{
		logger:         logger,
		templateParser: parsers.NewPongo2StringTemplateParser(logger),
	}
}

func (n *outputNormalizer) Initialize(ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) error {
	aggregator, err := internal_output_aggregator_normalizer.NewLLMTextAggregator(ctx, n.logger, n.onAggregated)
	if err != nil {
		return err
	}
	n.aggregator = aggregator

	if dictionaries, err := communication.GetOptions().GetString("speaker.pronunciation.dictionaries"); err == nil && dictionaries != "" {
		n.normalizers = n.buildNormalizerPipeline(strings.Split(dictionaries, commons.SEPARATOR))
	}
	registry := internal_namespace.NewDefaultRegistry()
	n.expandArgs = func() map[string]interface{} {
		return registry.Expand(variable.NewCommunicationSource(communication), variable.ResolveContext{})
	}
	n.onPacket = func(ctx context.Context, pkts ...internal_type.Packet) error {
		return communication.OnPacket(ctx, pkts...)
	}
	return nil
}

func (n *outputNormalizer) Close(_ context.Context) error {
	if n.aggregator != nil {
		n.aggregator.Close()
	}
	n.onPacket = nil
	return nil
}

// onAggregated is the callback wired to the aggregator's output.
// When the aggregator flushes a sentence, it enters the Argumentation stage.
func (n *outputNormalizer) onAggregated(ctx context.Context, pkts ...internal_type.Packet) error {
	for _, pkt := range pkts {
		switch sp := pkt.(type) {
		case internal_type.TextToSpeechTextPacket:
			n.Run(ctx, ArgumentationPipeline{ContextID: sp.ContextID, Text: sp.Text})
		case internal_type.TextToSpeechDonePacket:
			n.Run(ctx, ArgumentationPipeline{ContextID: sp.ContextID, Text: sp.Text, IsFinal: true})
		}
	}
	return nil
}

func (n *outputNormalizer) Normalize(ctx context.Context, packets ...internal_type.Packet) error {
	for _, pkt := range packets {
		switch p := pkt.(type) {
		case internal_type.LLMResponseDeltaPacket:
			n.Run(ctx, AggregatePipeline{ContextID: p.ContextID, Text: p.Text})
		case internal_type.LLMResponseDonePacket:
			n.Run(ctx, AggregatePipeline{ContextID: p.ContextID, Text: p.Text, IsFinal: true})
		case internal_type.InjectMessagePacket:
			n.Run(ctx, ArgumentationPipeline{ContextID: p.ContextID, Text: p.Text})
			if !p.Interim {
				n.Run(ctx, ArgumentationPipeline{ContextID: p.ContextID, Text: p.Text, IsFinal: true})
			}
		case internal_type.InterruptionDetectedPacket:
			n.Run(ctx, InterruptPipeline{ContextID: p.ContextID})
		}
	}
	return nil
}

// =============================================================================
// Run — central pipeline dispatch
// =============================================================================

func (n *outputNormalizer) Run(ctx context.Context, p NormalizerPipeline) {
	switch v := p.(type) {
	case AggregatePipeline:
		n.handleAggregate(ctx, v)
	case ArgumentationPipeline:
		n.handleArgumentation(ctx, v)
	case CleanTextPipeline:
		n.handleCleanText(ctx, v)
	case OutputPipeline:
		n.handleOutput(ctx, v)
	case InterruptPipeline:
		n.handleInterrupt()
	}
}

// =============================================================================
// Pipeline handlers
// =============================================================================

func (n *outputNormalizer) handleAggregate(ctx context.Context, v AggregatePipeline) {
	var pkt internal_type.LLMPacket
	if v.IsFinal {
		pkt = internal_type.LLMResponseDonePacket{ContextID: v.ContextID, Text: v.Text}
	} else {
		pkt = internal_type.LLMResponseDeltaPacket{ContextID: v.ContextID, Text: v.Text}
	}
	if err := n.aggregator.Aggregate(ctx, pkt); err != nil {
		n.Run(ctx, ArgumentationPipeline{ContextID: v.ContextID, Text: v.Text, IsFinal: v.IsFinal})
	}
}

func (n *outputNormalizer) handleArgumentation(ctx context.Context, v ArgumentationPipeline) {
	text := v.Text
	if n.templateParser != nil && n.expandArgs != nil {
		if args := n.expandArgs(); len(args) > 0 {
			text = n.templateParser.Parse(text, args)
		}
	}
	n.Run(ctx, CleanTextPipeline{ContextID: v.ContextID, Text: text, IsFinal: v.IsFinal})
}

func (n *outputNormalizer) handleCleanText(ctx context.Context, v CleanTextPipeline) {
	text := n.removeMarkdown(v.Text)
	for _, norm := range n.normalizers {
		text = norm.Normalize(text)
	}
	if text == "" && !v.IsFinal {
		return
	}
	n.Run(ctx, OutputPipeline{ContextID: v.ContextID, Text: text, IsFinal: v.IsFinal})
}

func (n *outputNormalizer) handleOutput(ctx context.Context, v OutputPipeline) {
	if n.onPacket == nil {
		return
	}
	if v.IsFinal {
		n.onPacket(ctx, internal_type.TextToSpeechDonePacket{ContextID: v.ContextID, Text: v.Text})
	} else {
		n.onPacket(ctx, internal_type.TextToSpeechTextPacket{ContextID: v.ContextID, Text: v.Text})
	}
}

func (n *outputNormalizer) handleInterrupt() {
	if n.aggregator != nil {
		n.aggregator.Close()
	}
}

// =============================================================================
// Markdown removal
// =============================================================================

func (n *outputNormalizer) removeMarkdown(text string) string {
	text = markdownBlock.ReplaceAllString(text, "")
	text = markdownInline.ReplaceAllString(text, "$1")
	text = markdownHeading.ReplaceAllString(text, "")
	text = markdownEmph.ReplaceAllString(text, "$1$2")
	text = markdownImage.ReplaceAllString(text, "")
	text = markdownLink.ReplaceAllString(text, "$1")
	text = markdownQuote.ReplaceAllString(text, "")
	text = markdownHr.ReplaceAllString(text, "")
	text = markdownStars.ReplaceAllString(text, "")
	text = wordUnderscore.ReplaceAllString(text, "$1 $2")
	text = emojiPattern.ReplaceAllString(text, "")
	return text
}

// =============================================================================
// Normalizer pipeline builder
// =============================================================================

func (n *outputNormalizer) buildNormalizerPipeline(names []string) []internal_type.TextNormalizer {
	normalizers := make([]internal_type.TextNormalizer, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(strings.ToLower(name))
		var normalizer internal_type.TextNormalizer
		switch name {
		case "url":
			normalizer = NewUrlNormalizer(n.logger)
		case "currency":
			normalizer = NewCurrencyNormalizer(n.logger)
		case "date":
			normalizer = NewDateNormalizer(n.logger)
		case "time":
			normalizer = NewTimeNormalizer(n.logger)
		case "number", "number-to-word":
			normalizer = NewNumberToWordNormalizer(n.logger)
		case "symbol":
			normalizer = NewSymbolNormalizer(n.logger)
		case "general-abbreviation", "general":
			normalizer = NewGeneralAbbreviationNormalizer(n.logger)
		case "role-abbreviation", "role":
			normalizer = NewRoleAbbreviationNormalizer(n.logger)
		case "tech-abbreviation", "tech":
			normalizer = NewTechAbbreviationNormalizer(n.logger)
		case "address":
			normalizer = NewAddressNormalizer(n.logger)
		default:
			n.logger.Warnf("normalizer: unknown normalizer '%s', skipping", name)
			continue
		}
		normalizers = append(normalizers, normalizer)
	}
	return normalizers
}
