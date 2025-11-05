// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_azure

import (
	"fmt"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

// this is work around for mac as azure have little faster impliemntation for sdk using c++

// export LD_LIBRARY_PATH="/Users/prashant.srivastav/Documents/envs/speechsdk/lib/arm64:$LD_LIBRARY_PATH"
// export CGO_CFLAGS="-I/Users/prashant.srivastav/Documents/envs/speechsdk/include/c_api"
// export CGO_LDFLAGS="-L/Users/prashant.srivastav/Documents/envs/speechsdk/lib/arm64 -lMicrosoft.CognitiveServices.Speech.core"

// export CGO_CFLAGS="-I/Users/prashant.srivastav/Documents/envs/speechsdk/MicrosoftCognitiveServicesSpeech.xcframework/macos-arm64_x86_64/MicrosoftCognitiveServicesSpeech.framework/Headers"
// export CGO_LDFLAGS="-Wl,-rpath,/Users/prashant.srivastav/Documents/envs/speechsdk/MicrosoftCognitiveServicesSpeech.xcframework/macos-arm64_x86_64 -F/Users/prashant.srivastav/Documents/envs/speechsdk/MicrosoftCognitiveServicesSpeech.xcframework/macos-arm64_x86_64 -framework MicrosoftCognitiveServicesSpeech"

type AzureOption interface {
	SpeechToTextOption() (*speech.SpeechConfig, error)
	TextToSpeechOption() (*speech.SpeechConfig, error)
}

type azureOption struct {
	logger          commons.Logger
	options         utils.Option
	audioConfig     *internal_voices.AudioConfig
	endpoint        string
	subscriptionKey string
}

func NewAzureOption(
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	options utils.Option,
) (AzureOption, error) {
	subscriptionKey, ok := vaultCredential.GetValue().AsMap()["subscription_key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config key subscription_key not found")
	}
	endpoint, ok := vaultCredential.GetValue().AsMap()["endpoint"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config key endpoint not found")
	}

	return &azureOption{
		logger:          logger,
		options:         options,
		audioConfig:     audioConfig,
		endpoint:        endpoint.(string),
		subscriptionKey: subscriptionKey.(string),
	}, nil
}

func (az *azureOption) SpeechToTextOption() (*speech.SpeechConfig, error) {
	cfg, err := speech.NewSpeechConfigFromEndpointWithSubscription(az.endpoint, az.subscriptionKey)
	if language, ok := az.options.GetString("listen.language"); ok == nil {
		cfg.SetSpeechRecognitionLanguage(language)
	}
	return cfg, err
}

func (az *azureOption) TextToSpeechOption() (*speech.SpeechConfig, error) {
	cfg, err := speech.
		NewSpeechConfigFromEndpointWithSubscription(
			az.endpoint,
			az.subscriptionKey)

	cfg.SetSpeechSynthesisOutputFormat(
		GetSpeechSynthesisOutputFormat(az.audioConfig.GetFormat()),
	)
	if voiceIDValue, ok := az.options.GetString("speak.voice.id"); ok == nil {
		cfg.SetSpeechSynthesisVoiceName(voiceIDValue)
	}
	if language, ok := az.options.GetString("speak.language"); ok == nil {
		cfg.SetSpeechSynthesisLanguage(language)
	}
	if format, ok := az.options.GetString("speak.output_format.encoding"); ok == nil {
		cfg.SetSpeechSynthesisOutputFormat(GetSpeechSynthesisOutputFormat(format))
	}
	return cfg, err
}

func GetSpeechSynthesisOutputFormat(format string) common.SpeechSynthesisOutputFormat {
	switch format {
	case "Raw8Khz8BitMonoMULaw", "MuLaw8":
		return common.Raw8Khz8BitMonoMULaw
	case "Riff16Khz16KbpsMonoSiren":
		return common.Riff16Khz16KbpsMonoSiren
	case "Audio16Khz16KbpsMonoSiren":
		return common.Audio16Khz16KbpsMonoSiren
	case "Audio16Khz32KBitRateMonoMp3":
		return common.Audio16Khz32KBitRateMonoMp3
	case "Audio16Khz128KBitRateMonoMp3":
		return common.Audio16Khz128KBitRateMonoMp3
	case "Audio16Khz64KBitRateMonoMp3":
		return common.Audio16Khz64KBitRateMonoMp3
	case "Audio24Khz48KBitRateMonoMp3":
		return common.Audio24Khz48KBitRateMonoMp3
	case "Audio24Khz96KBitRateMonoMp3":
		return common.Audio24Khz96KBitRateMonoMp3
	case "Audio24Khz160KBitRateMonoMp3":
		return common.Audio24Khz160KBitRateMonoMp3
	case "Raw16Khz16BitMonoTrueSilk":
		return common.Raw16Khz16BitMonoTrueSilk
	case "Riff16Khz16BitMonoPcm":
		return common.Riff16Khz16BitMonoPcm
	case "Riff8Khz16BitMonoPcm":
		return common.Riff8Khz16BitMonoPcm
	case "Riff24Khz16BitMonoPcm":
		return common.Riff24Khz16BitMonoPcm
	case "Riff8Khz8BitMonoMULaw":
		return common.Riff8Khz8BitMonoMULaw
	case "Raw16Khz16BitMonoPcm":
		return common.Raw16Khz16BitMonoPcm
	case "Raw24Khz16BitMonoPcm", "Linear16":
		return common.Raw24Khz16BitMonoPcm
	default:
		return common.Raw24Khz16BitMonoPcm
	}
}
