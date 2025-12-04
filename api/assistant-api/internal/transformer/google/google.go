// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_google

import (
	"cloud.google.com/go/speech/apiv1/speechpb"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
	"google.golang.org/api/option"
)

// Introduced constants for default values
const (
	DefaultLanguageCode = "en-US"            // Default language code for Speech-to-Text
	DefaultModel        = "default"          // Default model used for Speech recognition
	DefaultVoice        = "en-US-Chirp-HD-F" // Default voice for Text-to-Speech
)

// googleOption is the primary configuration structure for Google services
type googleOption struct {
	logger            commons.Logger
	clientOptons      []option.ClientOption
	audioConfig       *internal_audio.AudioConfig
	initializeOptions utils.Option
}

// NewGoogleOption initializes googleOption with provided credentials, audio configurations, and options.
// Improves error handling and logging for better debugging and robustness.
func NewGoogleOption(logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	audioConfig *internal_audio.AudioConfig,
	opts utils.Option) (*googleOption, error) {
	co := make([]option.ClientOption, 0)
	credentialsMap := vaultCredential.GetValue().AsMap()
	cx, ok := credentialsMap["key"]
	if ok {
		co = append(co, option.WithAPIKey(cx.(string)))
	}
	prj, ok := credentialsMap["project_id"]
	if ok {
		co = append(co, option.WithQuotaProject(prj.(string)))
	}
	serviceCrd, ok := credentialsMap["service_account_key"]
	if ok {
		serviceCrdJSON := []byte(serviceCrd.(string)) // Convert string to []byte
		co = append(co, option.WithCredentialsJSON(serviceCrdJSON))
	}

	return &googleOption{
		logger:            logger,
		initializeOptions: opts,
		clientOptons:      co,
		audioConfig:       audioConfig,
	}, nil
}

// GetClientOptions returns all configured Google API client options.
func (gO *googleOption) GetClientOptions() []option.ClientOption {
	return gO.clientOptons
}

// SpeechToTextOptions generates a configuration for Google Speech-to-Text streaming recognition.
// Default language and model are used unless overridden via initializeOptions.
func (gog *googleOption) SpeechToTextOptions() *speechpb.StreamingRecognitionConfig {
	audioEncoding, _ := gog.GetAudioEncoding(gog.audioConfig.Format)

	opts := &speechpb.StreamingRecognitionConfig{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   audioEncoding,
			SampleRateHertz:            int32(gog.audioConfig.GetSampleRate()),
			EnableAutomaticPunctuation: true,
			EnableWordConfidence:       true,
			ProfanityFilter:            true,
			LanguageCode:               DefaultLanguageCode,
			Model:                      DefaultModel,
		},
		InterimResults: true,
	}

	// Override language code if specified in options
	if language, err := gog.initializeOptions.GetString("listen.language"); err == nil {
		opts.Config.LanguageCode = language
	} else {
		gog.logger.Warn("Language not specified, defaulting to " + DefaultLanguageCode)
	}

	// Override model if specified in options
	if model, err := gog.initializeOptions.GetString("listen.model"); err == nil {
		opts.Config.Model = model
	} else {
		gog.logger.Warn("Model not specified, defaulting to " + DefaultModel)
	}

	return opts
}

// TextToSpeechOptions generates a configuration for Google Text-to-Speech streaming synthesis.
func (goog *googleOption) TextToSpeechOptions() *texttospeechpb.StreamingSynthesizeConfig {
	_, audioEncoding := goog.GetAudioEncoding(goog.audioConfig.Format)

	options := &texttospeechpb.StreamingSynthesizeConfig{
		Voice: &texttospeechpb.VoiceSelectionParams{
			Name: DefaultVoice,
		},
		StreamingAudioConfig: &texttospeechpb.StreamingAudioConfig{
			AudioEncoding:   audioEncoding,
			SampleRateHertz: int32(goog.audioConfig.GetSampleRate()),
		},
	}

	// Override voice configuration if specified in options
	if voice, err := goog.initializeOptions.GetString("speak.voice.id"); err == nil {
		options.Voice.Name = voice
	} else {
		goog.logger.Warn("Voice not specified, defaulting to " + DefaultVoice)
	}

	return options
}

// GetSpeechToTextEncodingFromString maps internal_audio.AudioFormat to Google's Speech-to-Text encoding.
func (gog *googleOption) GetSpeechToTextEncodingFromString(encoding internal_audio.AudioFormat) speechpb.RecognitionConfig_AudioEncoding {
	switch encoding {
	case internal_audio.Linear16:
		return speechpb.RecognitionConfig_LINEAR16
	case internal_audio.MuLaw8:
		return speechpb.RecognitionConfig_MULAW
	default:
		return speechpb.RecognitionConfig_LINEAR16
	}
}

// GetTextToSpeechEncodingByName maps internal_audio.AudioFormat to Google's Text-to-Speech encoding.
func (gog *googleOption) GetTextToSpeechEncodingByName(encoding internal_audio.AudioFormat) texttospeechpb.AudioEncoding {
	switch encoding {
	case internal_audio.Linear16:
		return texttospeechpb.AudioEncoding_PCM
	case internal_audio.MuLaw8:
		return texttospeechpb.AudioEncoding_MULAW
	default:
		return texttospeechpb.AudioEncoding_PCM
	}
}

// GetAudioEncoding returns audio encoding for both SpeechToText and TextToSpeech based on internal_audio.AudioFormat.
// Reduces repetitive logic in audio encoding handling.
func (gog *googleOption) GetAudioEncoding(audioFormat internal_audio.AudioFormat) (speechpb.RecognitionConfig_AudioEncoding, texttospeechpb.AudioEncoding) {
	switch audioFormat {
	case internal_audio.Linear16:
		return speechpb.RecognitionConfig_LINEAR16, texttospeechpb.AudioEncoding_PCM
	case internal_audio.MuLaw8:
		return speechpb.RecognitionConfig_MULAW, texttospeechpb.AudioEncoding_MULAW
	default:
		return speechpb.RecognitionConfig_LINEAR16, texttospeechpb.AudioEncoding_PCM
	}
}
