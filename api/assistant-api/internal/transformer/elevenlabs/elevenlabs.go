// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_elevenlabs

import (
	"fmt"
	"net/url"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	ELEVENLABS_VOICE_ID = "TWUKKXAylkYxxlPe4gx0"
)

func (elabs *elevenLabsOption) GetEncoding() string {
	switch elabs.audioConfig.GetAudioFormat() {
	case protos.AudioConfig_LINEAR16:
		return "pcm_16000"
	case protos.AudioConfig_MuLaw8:
		return "ulaw_8000"
	default:
		elabs.logger.Debugf("Warning: Invalid encoding mdlOpts. Using default (linear16).")
		return "pcm_16000"
	}
}

type elevenLabsOption struct {
	key         string
	logger      commons.Logger
	mdlOpts     utils.Option
	audioConfig *protos.AudioConfig
}

func NewElevenLabsOption(logger commons.Logger, vaultCredential *protos.VaultCredential,
	audioConfig *protos.AudioConfig,
	opts utils.Option) (*elevenLabsOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("elevenLabs: illegal vault config")
	}
	return &elevenLabsOption{
		key:         cx.(string),
		audioConfig: audioConfig,
		mdlOpts:     opts,
		logger:      logger,
	}, nil
}

func (co *elevenLabsOption) GetKey() string {
	return co.key
}

func (co *elevenLabsOption) GetTextToSpeechConnectionString() string {
	params := url.Values{}
	params.Add("output_format", co.GetEncoding())
	params.Add("enable_ssml_parsing", "true")

	if language, err := co.mdlOpts.GetString("speak.language"); err == nil {
		params.Add("language", language)
	}

	if model, err := co.mdlOpts.GetString("speak.model"); err == nil {
		params.Add("model_id", model)
	}

	voiceId := ELEVENLABS_VOICE_ID
	if voiceIDValue, err := co.mdlOpts.GetString("speak.voice.id"); err == nil {
		voiceId = voiceIDValue
	}

	return fmt.Sprintf("wss://api.elevenlabs.io/v1/text-to-speech/%s/multi-stream-input?%s", voiceId, params.Encode())
}
