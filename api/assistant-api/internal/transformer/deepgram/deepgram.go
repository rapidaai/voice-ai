// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_deepgram

import (
	"fmt"
	"strings"

	commons "github.com/rapidaai/pkg/commons"
	utils "github.com/rapidaai/pkg/utils"

	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	protos "github.com/rapidaai/protos"
)

func (dg *deepgramOption) GetEncoding() string {
	switch dg.audioConfig.GetAudioFormat() {
	case protos.AudioConfig_LINEAR16:
		return "linear16"
	case protos.AudioConfig_MuLaw8:
		return "mulaw"
	default:
		return "linear16"
	}

}

type deepgramOption struct {
	key         string
	logger      commons.Logger
	mdlOpts     utils.Option
	audioConfig *protos.AudioConfig
}

func NewDeepgramOption(
	logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	audioConfig *protos.AudioConfig,
	opts utils.Option) (*deepgramOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &deepgramOption{
		key:         cx.(string),
		logger:      logger,
		mdlOpts:     opts,
		audioConfig: audioConfig,
	}, nil
}

func (dgOpt *deepgramOption) GetKey() string {
	return dgOpt.key
}

func (dgOpt *deepgramOption) SpeechToTextOptions() *interfaces.LiveTranscriptionOptions {

	opts := &interfaces.LiveTranscriptionOptions{
		Model:          "nova",
		Language:       "en-US",
		Channels:       1,
		SmartFormat:    true,
		InterimResults: true,
		FillerWords:    true,
		VadEvents:      false,
		Endpointing:    "5",
		Punctuate:      true,
		NoDelay:        true,
		Encoding:       dgOpt.GetEncoding(),
		SampleRate:     int(dgOpt.audioConfig.GetSampleRate()),
		Diarize:        false,
		Multichannel:   false,
	}

	if language, err := dgOpt.mdlOpts.GetString("listen.language"); err == nil {
		opts.Language = language
	}

	if smartFormat, err := dgOpt.mdlOpts.GetBool("listen.smart_format"); err == nil {
		opts.SmartFormat = smartFormat
	}

	if fillerWords, err := dgOpt.mdlOpts.GetBool("listen.filler_words"); err == nil {
		opts.FillerWords = fillerWords
	}
	if vadEvents, err := dgOpt.mdlOpts.GetBool("listen.vad_events"); err == nil {
		opts.VadEvents = vadEvents
	}
	if endpointing, err := dgOpt.mdlOpts.GetString("listen.endpointing"); err == nil {
		opts.Endpointing = endpointing
	}
	if multichannel, err := dgOpt.mdlOpts.GetBool("listen.multichannel"); err == nil {
		opts.Multichannel = multichannel
	}
	if model, err := dgOpt.mdlOpts.GetString("listen.model"); err == nil {
		opts.Model = model
	}

	if keywordsRaw, exists := dgOpt.mdlOpts["listen.keyword"]; exists {
		var keywords []string
		switch v := keywordsRaw.(type) {
		case string:
			trimmed := strings.Trim(v, "[]")
			keywords = strings.Fields(trimmed)
		case []interface{}:
			keywords = make([]string, len(v))
			for i, keyword := range v {
				if str, ok := keyword.(string); ok {
					keywords[i] = strings.TrimSpace(str)
				}
			}
		default:
			dgOpt.logger.Warnf("Unexpected type for keywords: %T", keywordsRaw)
		}
		if len(keywords) > 0 {
			if opts.Model == "nova-2" {
				opts.Keywords = keywords
			}
			if opts.Model == "nova-3" {
				opts.Keyterm = keywords
			}

		}
	}
	return opts
}

func (dgOpt *deepgramOption) TextToSpeechOptions() *interfaces.WSSpeakOptions {
	opts := &interfaces.WSSpeakOptions{
		Model:      "aura-asteria-en",
		Encoding:   dgOpt.GetEncoding(),
		SampleRate: int(dgOpt.audioConfig.SampleRate),
	}

	if voiceIDValue, err := dgOpt.mdlOpts.GetString("speak.voice.id"); err == nil {
		opts.Model = voiceIDValue
	}

	return opts
}
