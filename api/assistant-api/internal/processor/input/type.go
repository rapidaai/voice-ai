// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_input_processors

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
)

const (
	LanguageAttributeISO6391 = "language.iso639_1"
	LanguageAttributeISO6392 = "language.iso639_2"
	LanguageAttributeName    = "language.name"
	LanguageAttributeSource  = "language.source"
)

type ProcessorPipeline interface {
	processorPipeline()
}

type InputPipeline struct {
	ContextID string
	Speech    string
	Speechs   []internal_type.SpeechToTextPacket
}

type DetectLanguagePipeline struct {
	ContextID string
	Speech    string
	Speechs   []internal_type.SpeechToTextPacket
}

type OutputPipeline struct {
	ContextID string
	Speech    string
	Language  types.Language
}

func (InputPipeline) processorPipeline()          {}
func (DetectLanguagePipeline) processorPipeline() {}
func (OutputPipeline) processorPipeline()         {}
