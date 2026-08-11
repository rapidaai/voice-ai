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

	internal_transformer_assemblyai "github.com/rapidaai/api/assistant-api/internal/transformer/assembly-ai"
	internal_transformer_aws "github.com/rapidaai/api/assistant-api/internal/transformer/aws"
	internal_transformer_azure "github.com/rapidaai/api/assistant-api/internal/transformer/azure"
	internal_transformer_cartesia "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia"
	internal_transformer_custom "github.com/rapidaai/api/assistant-api/internal/transformer/custom"
	internal_transformer_deepgram "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram"
	internal_transformer_google "github.com/rapidaai/api/assistant-api/internal/transformer/google"
	internal_transformer_groq "github.com/rapidaai/api/assistant-api/internal/transformer/groq"
	internal_transformer_nvidia "github.com/rapidaai/api/assistant-api/internal/transformer/nvidia"
	internal_transformer_revai "github.com/rapidaai/api/assistant-api/internal/transformer/revai"
	internal_transformer_sarvam "github.com/rapidaai/api/assistant-api/internal/transformer/sarvam"
	internal_transformer_smallest "github.com/rapidaai/api/assistant-api/internal/transformer/smallest"
	internal_transformer_speechmatics "github.com/rapidaai/api/assistant-api/internal/transformer/speechmatics"
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

	assistantID    uint64
	conversationID uint64
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

func WithAssistantID(assistantID uint64) Option {
	return func(options *options) {
		options.assistantID = assistantID
	}
}

func WithConversationID(conversationID uint64) Option {
	return func(options *options) {
		options.conversationID = conversationID
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
			internal_transformer_deepgram.WithAssistantID(options.assistantID),
			internal_transformer_deepgram.WithConversationID(options.conversationID),
		)
	case AZURE_SPEECH_SERVICE:
		return internal_transformer_azure.NewSpeechToText(
			internal_transformer_azure.WithContext(options.ctx),
			internal_transformer_azure.WithLogger(options.logger),
			internal_transformer_azure.WithCredential(options.credential),
			internal_transformer_azure.WithOnPacket(options.onPacket),
			internal_transformer_azure.WithOptions(options.sttOptions),
			internal_transformer_azure.WithAssistantID(options.assistantID),
			internal_transformer_azure.WithConversationID(options.conversationID),
		)
	case GOOGLE_SPEECH_SERVICE:
		return internal_transformer_google.NewSpeechToText(
			internal_transformer_google.WithContext(options.ctx),
			internal_transformer_google.WithLogger(options.logger),
			internal_transformer_google.WithCredential(options.credential),
			internal_transformer_google.WithOnPacket(options.onPacket),
			internal_transformer_google.WithOptions(options.sttOptions),
			internal_transformer_google.WithAssistantID(options.assistantID),
			internal_transformer_google.WithConversationID(options.conversationID),
		)
	case ASSEMBLYAI:
		return internal_transformer_assemblyai.NewSpeechToText(
			internal_transformer_assemblyai.WithContext(options.ctx),
			internal_transformer_assemblyai.WithLogger(options.logger),
			internal_transformer_assemblyai.WithCredential(options.credential),
			internal_transformer_assemblyai.WithOnPacket(options.onPacket),
			internal_transformer_assemblyai.WithOptions(options.sttOptions),
			internal_transformer_assemblyai.WithAssistantID(options.assistantID),
			internal_transformer_assemblyai.WithConversationID(options.conversationID),
		)
	case REVAI:
		return internal_transformer_revai.NewSpeechToText(
			internal_transformer_revai.WithContext(options.ctx),
			internal_transformer_revai.WithLogger(options.logger),
			internal_transformer_revai.WithCredential(options.credential),
			internal_transformer_revai.WithOnPacket(options.onPacket),
			internal_transformer_revai.WithOptions(options.sttOptions),
			internal_transformer_revai.WithAssistantID(options.assistantID),
			internal_transformer_revai.WithConversationID(options.conversationID),
		)
	case SARVAM:
		return internal_transformer_sarvam.NewSpeechToText(
			internal_transformer_sarvam.WithContext(options.ctx),
			internal_transformer_sarvam.WithLogger(options.logger),
			internal_transformer_sarvam.WithCredential(options.credential),
			internal_transformer_sarvam.WithOnPacket(options.onPacket),
			internal_transformer_sarvam.WithOptions(options.sttOptions),
			internal_transformer_sarvam.WithAssistantID(options.assistantID),
			internal_transformer_sarvam.WithConversationID(options.conversationID),
		)
	case CARTESIA:
		return internal_transformer_cartesia.NewSpeechToText(
			internal_transformer_cartesia.WithContext(options.ctx),
			internal_transformer_cartesia.WithLogger(options.logger),
			internal_transformer_cartesia.WithCredential(options.credential),
			internal_transformer_cartesia.WithOnPacket(options.onPacket),
			internal_transformer_cartesia.WithOptions(options.sttOptions),
			internal_transformer_cartesia.WithAssistantID(options.assistantID),
			internal_transformer_cartesia.WithConversationID(options.conversationID),
		)
	case SPEECHMATICS:
		return internal_transformer_speechmatics.NewSpeechToText(
			internal_transformer_speechmatics.WithContext(options.ctx),
			internal_transformer_speechmatics.WithLogger(options.logger),
			internal_transformer_speechmatics.WithCredential(options.credential),
			internal_transformer_speechmatics.WithOnPacket(options.onPacket),
			internal_transformer_speechmatics.WithOptions(options.sttOptions),
			internal_transformer_speechmatics.WithAssistantID(options.assistantID),
			internal_transformer_speechmatics.WithConversationID(options.conversationID),
		)
	case GROQ:
		return internal_transformer_groq.NewSpeechToText(
			internal_transformer_groq.WithContext(options.ctx),
			internal_transformer_groq.WithLogger(options.logger),
			internal_transformer_groq.WithCredential(options.credential),
			internal_transformer_groq.WithOnPacket(options.onPacket),
			internal_transformer_groq.WithOptions(options.sttOptions),
			internal_transformer_groq.WithAssistantID(options.assistantID),
			internal_transformer_groq.WithConversationID(options.conversationID),
		)
	case NVIDIA:
		return internal_transformer_nvidia.NewSpeechToText(
			internal_transformer_nvidia.WithContext(options.ctx),
			internal_transformer_nvidia.WithLogger(options.logger),
			internal_transformer_nvidia.WithCredential(options.credential),
			internal_transformer_nvidia.WithOnPacket(options.onPacket),
			internal_transformer_nvidia.WithOptions(options.sttOptions),
			internal_transformer_nvidia.WithAssistantID(options.assistantID),
			internal_transformer_nvidia.WithConversationID(options.conversationID),
		)
	case AWS:
		return internal_transformer_aws.NewSpeechToText(
			internal_transformer_aws.WithContext(options.ctx),
			internal_transformer_aws.WithLogger(options.logger),
			internal_transformer_aws.WithCredential(options.credential),
			internal_transformer_aws.WithOnPacket(options.onPacket),
			internal_transformer_aws.WithOptions(options.sttOptions),
			internal_transformer_aws.WithAssistantID(options.assistantID),
			internal_transformer_aws.WithConversationID(options.conversationID),
		)
	case CUSTOM_STT:
		return internal_transformer_custom.NewSpeechToText(
			internal_transformer_custom.WithContext(options.ctx),
			internal_transformer_custom.WithLogger(options.logger),
			internal_transformer_custom.WithCredential(options.credential),
			internal_transformer_custom.WithOnPacket(options.onPacket),
			internal_transformer_custom.WithOptions(options.sttOptions),
			internal_transformer_custom.WithAssistantID(options.assistantID),
			internal_transformer_custom.WithConversationID(options.conversationID),
		)
	case SMALLEST:
		return internal_transformer_smallest.NewSpeechToText(
			internal_transformer_smallest.WithContext(options.ctx),
			internal_transformer_smallest.WithLogger(options.logger),
			internal_transformer_smallest.WithCredential(options.credential),
			internal_transformer_smallest.WithOnPacket(options.onPacket),
			internal_transformer_smallest.WithOptions(options.sttOptions),
			internal_transformer_smallest.WithAssistantID(options.assistantID),
			internal_transformer_smallest.WithConversationID(options.conversationID),
		)
	default:
		return nil, fmt.Errorf("stt: provider %q is not implemented", options.provider)
	}
}
