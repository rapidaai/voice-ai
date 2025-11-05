// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_elevenlabs

import (
	"fmt"
	"net/url"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type Encoding string

const (
	ELEVENLABS_VOICE_ID             = "TWUKKXAylkYxxlPe4gx0"
	EncodingMp3_22050_32   Encoding = "mp3_22050_32"
	EncodingMp3_44100_32   Encoding = "mp3_44100_32"
	EncodingMp3_44100_64   Encoding = "mp3_44100_64"
	EncodingMp3_44100_96   Encoding = "mp3_44100_96"
	EncodingMp3_44100_128  Encoding = "mp3_44100_128"
	EncodingMp3_44100_192  Encoding = "mp3_44100_192"
	EncodingPcm_8000       Encoding = "pcm_8000"
	EncodingPcm_16000      Encoding = "pcm_16000"
	EncodingPcm_22050      Encoding = "pcm_22050"
	EncodingPcm_24000      Encoding = "pcm_24000"
	EncodingPcm_44100      Encoding = "pcm_44100"
	EncodingUlaw_8000      Encoding = "ulaw_8000"
	EncodingAlaw_8000      Encoding = "alaw_8000"
	EncodingOpus_48000_32  Encoding = "opus_48000_32"
	EncodingOpus_48000_64  Encoding = "opus_48000_64"
	EncodingOpus_48000_96  Encoding = "opus_48000_96"
	EncodingOpus_48000_128 Encoding = "opus_48000_128"
	EncodingOpus_48000_192 Encoding = "opus_48000_192"
)

func (d Encoding) String() string {
	return string(d)
}
func EncodingFromString(encoding string) string {
	switch Encoding(encoding) {
	case "Linear16":
		return EncodingPcm_24000.String()
	case "MuLaw8":
		return EncodingUlaw_8000.String()
	case EncodingMp3_22050_32, EncodingMp3_44100_32, EncodingMp3_44100_64,
		EncodingMp3_44100_96, EncodingMp3_44100_128, EncodingMp3_44100_192,
		EncodingPcm_8000, EncodingPcm_22050,
		EncodingPcm_24000, EncodingPcm_44100,
		EncodingUlaw_8000, EncodingAlaw_8000,
		EncodingOpus_48000_32, EncodingOpus_48000_64, EncodingOpus_48000_96,
		EncodingOpus_48000_128, EncodingOpus_48000_192:
		return encoding
	default:
		fmt.Printf("Warning: Invalid encoding option '%s'. Using default (linear16).", encoding)
		return string(EncodingPcm_24000)
	}
}

type ElevenLabsOption interface {
	GetTextToSpeechConnectionString() string
	GetKey() string
}

type elevenLabsOption struct {
	key         string
	logger      commons.Logger
	option      utils.Option
	audioConfig *internal_voices.AudioConfig
}

func NewElevenLabsOption(logger commons.Logger, vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	opts utils.Option) (ElevenLabsOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &elevenLabsOption{
		key:         cx.(string),
		audioConfig: audioConfig,
		option:      opts,
		logger:      logger,
	}, nil
}

func (co *elevenLabsOption) GetKey() string {
	return co.key
}

func (co *elevenLabsOption) GetTextToSpeechConnectionString() string {
	params := url.Values{}
	params.Add("output_format", EncodingFromString(co.audioConfig.GetFormat()))
	params.Add("enable_ssml_parsing", "true")
	// Check and add language
	if language, err := co.option.GetString("speak.language"); err == nil {
		params.Add("language", language)
	}

	// Check and add model
	if model, err := co.option.GetString("speak.model"); err == nil {
		params.Add("model_id", model)
	}

	// Check and add encoding
	if encoding, err := co.option.GetString("speak.output_format.encoding"); err == nil {
		params.Add("output_format", EncodingFromString(encoding))
	}

	voiceId := ELEVENLABS_VOICE_ID
	if voiceIDValue, err := co.option.GetString("speak.voice.id"); err == nil {
		voiceId = voiceIDValue
	}

	return fmt.Sprintf("wss://api.elevenlabs.io/v1/text-to-speech/%s/multi-stream-input?%s", voiceId, params.Encode())
}
