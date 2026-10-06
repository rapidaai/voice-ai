// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_input_processors

import (
	"context"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	rapida_language "github.com/rapidaai/pkg/language"
	rapida_types "github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type inputProcessor struct {
	logger   commons.Logger
	onPacket func(ctx context.Context, packets ...internal_type.Packet) error
	parser   rapida_language.Parser
}

func NewInputProcessor(logger commons.Logger) internal_type.PacketProcessor {
	return &inputProcessor{
		logger: logger,
		parser: rapida_language.NewLinguaParser(logger),
	}
}

func (n *inputProcessor) Initialize(ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) error {
	n.onPacket = communication.OnPacket
	return nil
}

func (n *inputProcessor) Close(_ context.Context) error {
	n.onPacket = nil
	return nil
}

func (n *inputProcessor) Process(ctx context.Context, packets ...internal_type.Packet) error {
	for _, pkt := range packets {
		switch p := pkt.(type) {
		case internal_type.EndOfSpeechPacket:
			n.Run(ctx, InputPipeline{
				ContextID: p.ContextID,
				Speech:    p.Speech,
				Speechs:   p.Speechs,
			})
		case internal_type.UserTextReceivedPacket:
			input := InputPipeline{
				ContextID: p.ContextID,
				Speech:    p.Text,
			}
			if p.Language != "" {
				input.Speechs = []internal_type.SpeechToTextPacket{{
					ContextID: p.ContextID,
					Script:    p.Text,
					Language:  p.Language,
				}}
			}
			n.Run(ctx, input)
		}
	}
	return nil
}

func (n *inputProcessor) Run(ctx context.Context, p ProcessorPipeline) {
	switch v := p.(type) {
	case InputPipeline:
		language := n.detectLanguage(v.Speech, v.Speechs)
		n.Run(ctx, OutputPipeline{
			ContextID: v.ContextID,
			Speech:    v.Speech,
			Language:  language,
		})

	case DetectLanguagePipeline:
		language := n.detectLanguage(v.Speech, v.Speechs)
		n.Run(ctx, OutputPipeline{
			ContextID: v.ContextID,
			Speech:    v.Speech,
			Language:  language,
		})

	case OutputPipeline:
		if n.onPacket != nil {
			n.onPacket(ctx, internal_type.UserInputPacket{
				ContextID: v.ContextID,
				Text:      v.Speech,
				Language:  v.Language,
			})
		}

	default:
		n.logger.Errorf("unknown processor pipeline type: %T", p)
	}
}

func (n *inputProcessor) detectLanguage(speech string, speechs []internal_type.SpeechToTextPacket) rapida_types.Language {
	if code := n.consensusLanguageCode(speechs); code != "" {
		return rapida_types.LookupLanguage(code)
	}
	if strings.TrimSpace(speech) == "" {
		return rapida_types.UNKNOWN_LANGUAGE
	}
	parsed, _ := n.parser.Parse(speech)
	return parsed
}

func (n *inputProcessor) consensusLanguageCode(speeches []internal_type.SpeechToTextPacket) string {
	if len(speeches) == 0 {
		return ""
	}
	counts := make(map[string]int)
	bestCode := ""
	bestCount := 0
	for _, s := range speeches {
		code := processLanguageCode(s.Language)
		if code == "" {
			continue
		}
		counts[code]++
		if counts[code] > bestCount {
			bestCount = counts[code]
			bestCode = code
		}
	}
	return bestCode
}

func processLanguageCode(v string) string {
	clean := strings.TrimSpace(strings.ToLower(v))
	if clean == "" {
		return ""
	}
	if idx := strings.Index(clean, "-"); idx > 0 {
		clean = clean[:idx]
	}
	if len(clean) != 2 {
		return ""
	}
	return rapida_types.LookupLanguage(clean).ISO639_1
}
