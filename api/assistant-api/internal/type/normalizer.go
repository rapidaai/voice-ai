// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import (
	"context"

	"github.com/rapidaai/protos"
)

// TextProcessor defines the contract for provider-specific TTS text preprocessing.
type TextProcessor interface {
	Process(text string) string
}

// PacketProcessor defines the contract for packet-level preprocessors (input/output).
type PacketProcessor interface {
	Initialize(ctx context.Context, communication Communication, cfg *protos.ConversationInitialization) error
	Process(ctx context.Context, in ...Packet) error
	Close(ctx context.Context) error
}

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
