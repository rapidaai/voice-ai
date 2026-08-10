// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer

import (
	"context"
	"errors"
	"fmt"

	internal_transformer_deepgram "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type options struct {
	ctx        context.Context
	logger     commons.Logger
	provider   string
	credential *protos.VaultCredential
	onPacket   func(pkt ...internal_type.Packet) error
	sttOptions utils.Option
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

func WithProvider(provider string) Option {
	return func(options *options) {
		options.provider = provider
	}
}

func WithCredential(credential *protos.VaultCredential) Option {
	return func(options *options) {
		options.credential = credential
	}
}

func WithOnPacket(onPacket func(pkt ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(sttOptions utils.Option) Option {
	return func(options *options) {
		options.sttOptions = sttOptions
	}
}

// NewSpeechToText creates and initializes the STT transformer implementation
// matching the configured provider.
func NewSpeechToText(opts ...Option) (internal_type.SpeechToTextTransformer, error) {
	options := &options{ctx: context.Background(), sttOptions: utils.Option{}}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.provider == "" {
		return nil, errors.New("stt: provider is required")
	}
	if options.credential == nil {
		return nil, errors.New("stt: credential is required")
	}
	if options.onPacket == nil {
		return nil, errors.New("stt: on packet handler is required")
	}

	switch AudioTransformer(options.provider) {
	case DEEPGRAM:
		return internal_transformer_deepgram.NewSpeechToText(
			internal_transformer_deepgram.WithContext(options.ctx),
			internal_transformer_deepgram.WithLogger(options.logger),
			internal_transformer_deepgram.WithCredential(options.credential),
			internal_transformer_deepgram.WithOnPacket(options.onPacket),
			internal_transformer_deepgram.WithOptions(options.sttOptions),
		)
	default:
		return nil, fmt.Errorf("stt: provider %q is not implemented", options.provider)
	}
}
