// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"fmt"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type azureOption struct {
	logger          commons.Logger
	options         utils.Option
	audioConfig     *internal_audio.AudioConfig
	endpoint        string
	subscriptionKey string
}

func NewAzureOption(
	logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	audioConfig *internal_audio.AudioConfig,
	options utils.Option,
) (*azureOption, error) {
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

	if err != nil {
		az.logger.Errorf("azure: error while building text to speech options")
		return nil, err
	}
	cfg.SetSpeechSynthesisOutputFormat(
		az.GetSpeechSynthesisOutputFormat(),
	)
	if voiceIDValue, ok := az.options.GetString("speak.voice.id"); ok == nil {
		az.logger.Debugf("azure options %v", voiceIDValue)
		cfg.SetSpeechSynthesisVoiceName(voiceIDValue)
	}
	if language, ok := az.options.GetString("speak.language"); ok == nil {
		cfg.SetSpeechSynthesisLanguage(language)
		az.logger.Debugf("azure options %v", language)
	}

	return cfg, err
}

func (az *azureOption) GetSpeechSynthesisOutputFormat() common.SpeechSynthesisOutputFormat {
	switch az.audioConfig.Format {
	case internal_audio.MuLaw8:
		return common.Raw8Khz8BitMonoMULaw
	case internal_audio.Linear16:
		return common.Raw16Khz16BitMonoPcm
	default:
		return common.Raw16Khz16BitMonoPcm
	}
}

func (az *azureOption) GetAudioStreamFormat() *audio.AudioStreamFormat {
	switch az.audioConfig.Format {
	case internal_audio.MuLaw8:
		v, _ := audio.GetWaveFormat(uint32(az.audioConfig.SampleRate), uint8(8), uint8(1), audio.WaveMULAW)
		return v
	case internal_audio.Linear16:
		v, _ := audio.GetWaveFormat(uint32(az.audioConfig.SampleRate), uint8(16), 1, audio.WavePCM)
		return v
	default:
		v, _ := audio.GetWaveFormat(uint32(az.audioConfig.SampleRate), uint8(16), 1, audio.WavePCM)
		return v
	}
}
