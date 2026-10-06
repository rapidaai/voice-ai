// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_end_of_speech

import (
	"context"
	"fmt"

	internal_livekit "github.com/rapidaai/api/assistant-api/internal/end_of_speech/internal/livekit"
	internal_pipecat "github.com/rapidaai/api/assistant-api/internal/end_of_speech/internal/pipecat"
	internal_silence_based "github.com/rapidaai/api/assistant-api/internal/end_of_speech/internal/silence_based"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type EndOfSpeechIdentifier string

type options struct {
	ctx      context.Context
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	options  utils.Option
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(opts utils.Option) Option {
	return func(options *options) {
		options.options = opts
	}
}

func New(opts ...Option) (internal_type.EndOfSpeechExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}

	provider, _ := options.options.GetString(EndOfSpeechOptionsKeyProvider)
	switch EndOfSpeechIdentifier(provider) {
	case SilenceBasedEndOfSpeech:
		return internal_silence_based.New(
			internal_silence_based.WithContext(options.ctx),
			internal_silence_based.WithLogger(options.logger),
			internal_silence_based.WithOnPacket(options.onPacket),
			internal_silence_based.WithOptions(options.options),
		)
	case LiveKitEndOfSpeech:
		return internal_livekit.New(
			internal_livekit.WithContext(options.ctx),
			internal_livekit.WithLogger(options.logger),
			internal_livekit.WithOnPacket(options.onPacket),
			internal_livekit.WithOptions(options.options),
		)
	case PipecatSmartTurnEndOfSpeech:
		return internal_pipecat.New(
			internal_pipecat.WithContext(options.ctx),
			internal_pipecat.WithLogger(options.logger),
			internal_pipecat.WithOnPacket(options.onPacket),
			internal_pipecat.WithOptions(options.options),
		)
	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedEndOfSpeechProvider, provider)
	}
}
