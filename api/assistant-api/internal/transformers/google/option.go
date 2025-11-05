// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_google

import (
	"fmt"
	"strings"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/api/option"
)

type GoogleOption interface {
	SpeechToTextOptions() *speechpb.StreamingRecognitionConfig
	TextToSpeechOptions() *texttospeechpb.StreamingSynthesizeConfig
	GetClientOptions() []option.ClientOption
}

type googleOption struct {
	logger commons.Logger

	//
	clientOptons      []option.ClientOption
	audioConfig       *internal_voices.AudioConfig
	initializeOptions utils.Option
}

func GetSpeechToTextEncodingFromString(encoding string) speechpb.RecognitionConfig_AudioEncoding {
	switch strings.ToLower(encoding) {
	case "linear16", "Linear16":
		return speechpb.RecognitionConfig_LINEAR16
	case "flac":
		return speechpb.RecognitionConfig_FLAC
	case "mulaw", "MuLaw8":
		return speechpb.RecognitionConfig_MULAW
	case "amr":
		return speechpb.RecognitionConfig_AMR
	case "amr_wb":
		return speechpb.RecognitionConfig_AMR_WB
	case "ogg_opus":
		return speechpb.RecognitionConfig_OGG_OPUS
	case "speex_with_header_byte":
		return speechpb.RecognitionConfig_SPEEX_WITH_HEADER_BYTE
	case "mp3":
		return speechpb.RecognitionConfig_MP3
	case "webm_opus":
		return speechpb.RecognitionConfig_WEBM_OPUS
	default:
		return speechpb.RecognitionConfig_LINEAR16
	}
}

func NewGoogleOption(logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	opts utils.Option) (GoogleOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	co := make([]option.ClientOption, 0)
	if ok {
		co = append(co, option.WithAPIKey(cx.(string)))
	}

	prj, ok := vaultCredential.GetValue().AsMap()["project_id"]
	if ok {
		co = append(co, option.WithQuotaProject(prj.(string)))
	}

	serviceCrd, ok := vaultCredential.GetValue().AsMap()["service_account_key"]
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

// GetCredential returns the credential string if present in opts, otherwise returns an empty string.
func (gO *googleOption) GetClientOptions() []option.ClientOption {
	return gO.clientOptons
}

func (gog *googleOption) SpeechToTextOptions() *speechpb.StreamingRecognitionConfig {
	opts := &speechpb.RecognitionConfig{
		Encoding:                            GetSpeechToTextEncodingFromString(gog.audioConfig.GetFormat()),
		SampleRateHertz:                     int32(gog.audioConfig.GetSampleRate()),
		LanguageCode:                        "en-US",
		EnableAutomaticPunctuation:          true,
		EnableWordConfidence:                true,
		ProfanityFilter:                     true,
		AlternativeLanguageCodes:            []string{},
		AudioChannelCount:                   1,
		EnableSeparateRecognitionPerChannel: false,
	}
	if sampleRate, err := gog.initializeOptions.GetUint32("listen.output_format.sample_rate"); err == nil {
		opts.SampleRateHertz = int32(sampleRate)
	}

	if encoding, err := gog.initializeOptions.GetString("listen.output_format.encoding"); err == nil {
		opts.Encoding = GetSpeechToTextEncodingFromString(encoding)
	}

	if language, err := gog.initializeOptions.GetString("listen.language"); err == nil {
		opts.LanguageCode = language
	}

	if channels, err := gog.initializeOptions.GetUint32("listen.channels"); err == nil {
		opts.AudioChannelCount = int32(channels)
	}

	if model, err := gog.initializeOptions.GetString("listen.model"); err == nil {
		opts.Model = model
	}

	if langsRaw, exists := gog.initializeOptions["listen.other_languages"]; exists {
		var lgs []string
		switch v := langsRaw.(type) {
		case string:
			trimmed := strings.Trim(v, "[]")
			lgs = strings.Fields(trimmed)
		case []interface{}:
			lgs = make([]string, len(v))
			for i, keyword := range v {
				if str, ok := keyword.(string); ok {
					lgs[i] = strings.TrimSpace(str)
				}
			}
		default:
			gog.logger.Warnf("Unexpected type for keywords: %T", langsRaw)
		}
		if len(lgs) > 0 {
			opts.AlternativeLanguageCodes = lgs
		}
	}
	return &speechpb.StreamingRecognitionConfig{
		Config:         opts,
		InterimResults: true,
	}
}

func (goog *googleOption) TextToSpeechOptions() *texttospeechpb.StreamingSynthesizeConfig {
	options := &texttospeechpb.StreamingSynthesizeConfig{
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			Name:         "en-US-Chirp-HD-F",
		},
		StreamingAudioConfig: &texttospeechpb.StreamingAudioConfig{
			AudioEncoding:   GetTextToSpeechEncodingByName(goog.audioConfig.GetFormat()),
			SampleRateHertz: int32(goog.audioConfig.GetSampleRate()),
		},
	}
	// Default model

	goog.logger.Debugf("%+v", goog.initializeOptions)

	//
	languageCode, err := goog.initializeOptions.GetString("speak.language")
	if err != nil || languageCode == "" {
		languageCode = "en-US"
	}

	voice, err := goog.initializeOptions.GetString("speak.voice.id")
	if err != nil || voice == "" {
		voice = "achernar"
	}

	model, err := goog.initializeOptions.GetString("speak.model")
	if err != nil || model == "" {
		model = "Chirp-HD"
	}

	// Create the name from languageCode, model, and voice
	options.Voice.Name = fmt.Sprintf("%s-%s-%s", languageCode, model, voice)
	options.Voice.LanguageCode = languageCode

	if sampleRate, err := goog.initializeOptions.GetUint32("speak.output_format.sample_rate"); err == nil {
		options.StreamingAudioConfig.SampleRateHertz = int32(sampleRate)
	}

	if encoding, err := goog.initializeOptions.GetString("speak.output_format.encoding"); err == nil {
		options.StreamingAudioConfig.AudioEncoding = GetTextToSpeechEncodingByName(encoding)
	}

	return options

}

func GetTextToSpeechEncodingByName(name string) texttospeechpb.AudioEncoding {
	switch name {
	case "AUDIO_ENCODING_UNSPECIFIED":
		return texttospeechpb.AudioEncoding_AUDIO_ENCODING_UNSPECIFIED
	case "MP3":
		return texttospeechpb.AudioEncoding_MP3
	case "OGG_OPUS":
		return texttospeechpb.AudioEncoding_OGG_OPUS
	case "MULAW", "MuLaw8":
		return texttospeechpb.AudioEncoding_MULAW
	case "ALAW":
		return texttospeechpb.AudioEncoding_ALAW
	case "PCM", "Linear16":
		return texttospeechpb.AudioEncoding_PCM
	case "M4A":
		return texttospeechpb.AudioEncoding_M4A
	default:
		return texttospeechpb.AudioEncoding_AUDIO_ENCODING_UNSPECIFIED
	}
}
