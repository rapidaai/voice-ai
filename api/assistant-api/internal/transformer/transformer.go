// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer

import (
	"context"
	"fmt"

	internal_transformer_assemblyai "github.com/rapidaai/api/assistant-api/internal/transformer/assembly-ai"
	internal_transformer_aws "github.com/rapidaai/api/assistant-api/internal/transformer/aws"
	internal_transformer_azure "github.com/rapidaai/api/assistant-api/internal/transformer/azure"
	internal_transformer_cartesia "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia"
	internal_transformer_deepgram "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram"
	internal_transformer_elevenlabs "github.com/rapidaai/api/assistant-api/internal/transformer/elevenlabs"
	internal_transformer_google "github.com/rapidaai/api/assistant-api/internal/transformer/google"
	internal_transformer_groq "github.com/rapidaai/api/assistant-api/internal/transformer/groq"
	internal_transformer_inworld "github.com/rapidaai/api/assistant-api/internal/transformer/inworld"
	internal_transformer_minimax "github.com/rapidaai/api/assistant-api/internal/transformer/minimax"
	internal_transformer_neuphonic "github.com/rapidaai/api/assistant-api/internal/transformer/neuphonic"
	internal_transformer_nvidia "github.com/rapidaai/api/assistant-api/internal/transformer/nvidia"
	internal_transformer_resembleai "github.com/rapidaai/api/assistant-api/internal/transformer/resembleai"
	internal_transformer_revai "github.com/rapidaai/api/assistant-api/internal/transformer/revai"
	internal_transformer_rime "github.com/rapidaai/api/assistant-api/internal/transformer/rime"
	internal_transformer_sarvam "github.com/rapidaai/api/assistant-api/internal/transformer/sarvam"
	internal_transformer_speechmatics "github.com/rapidaai/api/assistant-api/internal/transformer/speechmatics"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type AudioTransformer string

const (
	DEEPGRAM              AudioTransformer = "deepgram"
	GOOGLE_SPEECH_SERVICE AudioTransformer = "google-speech-service"
	AZURE_SPEECH_SERVICE  AudioTransformer = "azure-speech-service"
	CARTESIA              AudioTransformer = "cartesia"
	REVAI                 AudioTransformer = "revai"
	SARVAM                AudioTransformer = "sarvamai"
	ELEVENLABS            AudioTransformer = "elevenlabs"
	INWORLD               AudioTransformer = "inworld"
	RIME                  AudioTransformer = "rime"
	ASSEMBLYAI            AudioTransformer = "assemblyai"
	SPEECHMATICS          AudioTransformer = "speechmatics"
	RESEMBLEAI            AudioTransformer = "resembleai"
	NEUPHONIC             AudioTransformer = "neuphonic"
	MINIMAX               AudioTransformer = "minimax"
	NVIDIA                AudioTransformer = "nvidia"
	GROQ                  AudioTransformer = "groq"
	AWS                   AudioTransformer = "aws"
)

func (at AudioTransformer) String() string {
	return string(at)
}

func GetTextToSpeechTransformer(ctx context.Context,
	logger commons.Logger,
	provider string,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	switch AudioTransformer(provider) {
	case DEEPGRAM:
		return internal_transformer_deepgram.NewDeepgramTextToSpeech(ctx, logger, credential, onPacket, opts)
	case AZURE_SPEECH_SERVICE:
		return internal_transformer_azure.NewAzureTextToSpeech(ctx, logger, credential, onPacket, opts)
	case CARTESIA:
		return internal_transformer_cartesia.NewCartesiaTextToSpeech(ctx, logger, credential, onPacket, opts)
	case GOOGLE_SPEECH_SERVICE:
		return internal_transformer_google.NewGoogleTextToSpeech(ctx, logger, credential, onPacket, opts)
	case REVAI:
		return internal_transformer_revai.NewRevaiTextToSpeech(ctx, logger, credential, onPacket, opts)
	case SARVAM:
		return internal_transformer_sarvam.NewSarvamTextToSpeech(ctx, logger, credential, onPacket, opts)
	case ELEVENLABS:
		return internal_transformer_elevenlabs.NewElevenlabsTextToSpeech(ctx, logger, credential, onPacket, opts)
	case INWORLD:
		return internal_transformer_inworld.NewInworldTextToSpeech(ctx, logger, credential, onPacket, opts)
	case RIME:
		return internal_transformer_rime.NewRimeTextToSpeech(ctx, logger, credential, onPacket, opts)
	case RESEMBLEAI:
		return internal_transformer_resembleai.NewResembleAITextToSpeech(ctx, logger, credential, onPacket, opts)
	case NEUPHONIC:
		return internal_transformer_neuphonic.NewNeuPhonicTextToSpeech(ctx, logger, credential, onPacket, opts)
	case MINIMAX:
		return internal_transformer_minimax.NewMiniMaxTextToSpeech(ctx, logger, credential, onPacket, opts)
	case GROQ:
		return internal_transformer_groq.NewGroqTextToSpeech(ctx, logger, credential, onPacket, opts)
	case SPEECHMATICS:
		return internal_transformer_speechmatics.NewSpeechmaticsTextToSpeech(ctx, logger, credential, onPacket, opts)
	case NVIDIA:
		return internal_transformer_nvidia.NewNvidiaTextToSpeech(ctx, logger, credential, onPacket, opts)
	case AWS:
		return internal_transformer_aws.NewAWSTextToSpeech(ctx, logger, credential, onPacket, opts)
	default:
		return nil, fmt.Errorf("illegal text to speech idenitfier")
	}
}

func GetSpeechToTextTransformer(ctx context.Context,
	logger commons.Logger,
	provider string,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	switch AudioTransformer(provider) {
	case DEEPGRAM:
		return internal_transformer_deepgram.NewDeepgramSpeechToText(ctx, logger, credential, onPacket, opts)
	case AZURE_SPEECH_SERVICE:
		return internal_transformer_azure.NewAzureSpeechToText(ctx, logger, credential, onPacket, opts)
	case GOOGLE_SPEECH_SERVICE:
		return internal_transformer_google.NewGoogleSpeechToText(ctx, logger, credential, onPacket, opts)
	case ASSEMBLYAI:
		return internal_transformer_assemblyai.NewAssemblyaiSpeechToText(ctx, logger, credential, onPacket, opts)
	case REVAI:
		return internal_transformer_revai.NewRevaiSpeechToText(ctx, logger, credential, onPacket, opts)
	case SARVAM:
		return internal_transformer_sarvam.NewSarvamSpeechToText(ctx, logger, credential, onPacket, opts)
	case CARTESIA:
		return internal_transformer_cartesia.NewCartesiaSpeechToText(ctx, logger, credential, onPacket, opts)
	case SPEECHMATICS:
		return internal_transformer_speechmatics.NewSpeechmaticsSpeechToText(ctx, logger, credential, onPacket, opts)
	case GROQ:
		return internal_transformer_groq.NewGroqSpeechToText(ctx, logger, credential, onPacket, opts)
	case NVIDIA:
		return internal_transformer_nvidia.NewNvidiaSpeechToText(ctx, logger, credential, onPacket, opts)
	case AWS:
		return internal_transformer_aws.NewAWSSpeechToText(ctx, logger, credential, onPacket, opts)
	default:
		return nil, fmt.Errorf("illegal speech to text idenitfier")
	}
}
