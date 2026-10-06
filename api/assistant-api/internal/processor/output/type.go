// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_processors

type ProcessorPipeline interface {
	processorPipeline()
}

// AggregatePipeline buffers streaming deltas and flushes at sentence boundaries.
type AggregatePipeline struct {
	ContextID string
	Text      string
	IsFinal   bool
}

// ArgumentationPipeline substitutes template variables ({{customer_name}}, etc.).
type ArgumentationPipeline struct {
	ContextID string
	Text      string
	IsFinal   bool
}

// CleanTextPipeline strips markdown, special chars, and processs text for TTS.
type CleanTextPipeline struct {
	ContextID string
	Text      string
	IsFinal   bool
}

// OutputPipeline emits the final TextToSpeechTextPacket to TTS.
type OutputPipeline struct {
	ContextID string
	Text      string
	IsFinal   bool
}

// InterruptPipeline clears aggregation buffers on user interruption.
type InterruptPipeline struct {
	ContextID string
}

func (AggregatePipeline) processorPipeline()     {}
func (ArgumentationPipeline) processorPipeline() {}
func (CleanTextPipeline) processorPipeline()     {}
func (OutputPipeline) processorPipeline()        {}
func (InterruptPipeline) processorPipeline()     {}
