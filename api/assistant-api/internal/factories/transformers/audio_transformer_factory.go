package internal_transformer_factories

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	internal_transformer_assemblyai "github.com/rapidaai/api/assistant-api/internal/transformers/assembly-ai"
	internal_transformer_aws "github.com/rapidaai/api/assistant-api/internal/transformers/aws"
	internal_transformer_azure "github.com/rapidaai/api/assistant-api/internal/transformers/azure"
	internal_transformer_cartesia "github.com/rapidaai/api/assistant-api/internal/transformers/cartesia"
	internal_transformer_deepgram "github.com/rapidaai/api/assistant-api/internal/transformers/deepgram"
	internal_transformer_elevenlabs "github.com/rapidaai/api/assistant-api/internal/transformers/elevenlabs"
	internal_transformer_google "github.com/rapidaai/api/assistant-api/internal/transformers/google"
	internal_transformer_revai "github.com/rapidaai/api/assistant-api/internal/transformers/revai"
	internal_transformer_sarvam "github.com/rapidaai/api/assistant-api/internal/transformers/sarvam"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type AudioTransformer string

const (
	DEEPGRAM     AudioTransformer = "deepgram"
	AZURE        AudioTransformer = "azure"
	CARTESIA     AudioTransformer = "cartesia"
	GOOGLE       AudioTransformer = "google"
	GOOGLE_CLOUD AudioTransformer = "google-cloud"
	AWS          AudioTransformer = "aws"
	REVAI        AudioTransformer = "revai"
	SARVAM       AudioTransformer = "sarvam"
	ELEVENLABS   AudioTransformer = "elevenlabs"
	ASSEMBLYAI   AudioTransformer = "assemblyai"
)

func (at AudioTransformer) String() string {
	return string(at)
}
func GetTextToSpeechTransformer(
	at AudioTransformer,
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	opts *internal_transformers.TextToSpeechInitializeOptions) (internal_transformers.TextToSpeechTransformer, error) {

	switch at {
	case DEEPGRAM:
		return internal_transformer_deepgram.NewDeepgramTextToSpeech(ctx, logger, credential, opts)
	case AZURE:
		return internal_transformer_azure.NewAzureTextToSpeech(ctx, logger, credential, opts)
	case CARTESIA:
		return internal_transformer_cartesia.NewCartesiaTextToSpeech(ctx, logger, credential, opts)
	case GOOGLE, GOOGLE_CLOUD:
		return internal_transformer_google.NewGoogleTextToSpeech(ctx, logger, credential, opts)
	case AWS:
		return internal_transformer_aws.NewAWSTextToSpeech(ctx, logger, credential, opts)
	case REVAI:
		return internal_transformer_revai.NewRevaiTextToSpeech(ctx, logger, credential, opts)
	case SARVAM:
		return internal_transformer_sarvam.NewSarvamTextToSpeech(ctx, logger, credential, opts)
	case ELEVENLABS:
		return internal_transformer_elevenlabs.NewElevenlabsTextToSpeech(ctx, logger, credential, opts)
	default:
		return internal_transformer_deepgram.NewDeepgramTextToSpeech(ctx, logger, credential, opts)
	}
}

func GetSpeechToTextTransformer(at AudioTransformer,
	ctx context.Context,
	logger commons.Logger,
	communication internal_adapter_requests.Communication,
	opts *internal_transformers.SpeechToTextInitializeOptions) (internal_transformers.SpeechToTextTransformer, error) {

	credentialId, err := opts.ModelOptions.GetUint64("rapida.credential_id")
	if err != nil {
		logger.Errorf("unable to find credential from options %+v", err)
		return nil, err
	}
	credential, err := communication.
		VaultCaller().
		GetCredential(context.Background(), communication.Auth(), credentialId)
	if err != nil {
		logger.Errorf("Api call to find credential failed %+v", err)
		return nil, err
	}

	switch at {
	case DEEPGRAM:
		return internal_transformer_deepgram.NewDeepgramSpeechToText(ctx, logger, credential, opts)
	case AZURE:
		return internal_transformer_azure.NewAzureSpeechToText(ctx, logger, credential, opts)
	case GOOGLE, GOOGLE_CLOUD:
		return internal_transformer_google.NewGoogleSpeechToText(ctx, logger, credential, opts)
	case AWS:
		return internal_transformer_aws.NewAWSSpeechToText(ctx, logger, credential, opts)
	case ASSEMBLYAI:
		return internal_transformer_assemblyai.NewAssemblyaiSpeechToText(ctx, logger, credential, opts)
	case REVAI:
		return internal_transformer_revai.NewRevaiSpeechToText(ctx, logger, credential, opts)
	case SARVAM:
		return internal_transformer_sarvam.NewSarvamSpeechToText(ctx, logger, credential, opts)
	case CARTESIA:
		return internal_transformer_cartesia.NewCartesiaSpeechToText(ctx, logger, credential, opts)
	default:
		return internal_transformer_deepgram.NewDeepgramSpeechToText(ctx, logger, credential, opts)
	}
}
