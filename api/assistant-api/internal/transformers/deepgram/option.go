// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_deepgram

import (
	"fmt"
	"strings"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	commons "github.com/rapidaai/pkg/commons"
	utils "github.com/rapidaai/pkg/utils"

	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	lexatic_backend "github.com/rapidaai/protos"
)

type Encoding string

const (
	//
	EncodingLinear16 Encoding = "linear16"
	EncodingMulaw    Encoding = "mulaw"
	EncodingAlaw     Encoding = "alaw"
)

// Name implements internal_transformers.SpeechToTextTransformer.
func (*deepgramTTS) Name() string {
	return "deepgram-speech-to-text"
}

func (d Encoding) String() string {
	return string(d)
}

func EncodingFromString(encoding string) string {
	switch Encoding(encoding) {
	case EncodingLinear16, "Linear16":
		return EncodingLinear16.String()
	case EncodingMulaw, "MuLaw8":
		return EncodingMulaw.String()
	default:
		fmt.Printf("Warning: Invalid encoding option '%s'. Using default (linear16).", encoding)
		return string(EncodingLinear16)
	}
}

type DeepgramOption interface {
	SpeechToTextOptions() *interfaces.LiveTranscriptionOptions
	TextToSpeechOptions() *interfaces.WSSpeakOptions
	GetKey() string
}

type deepgramOption struct {
	key         string
	logger      commons.Logger
	options     utils.Option
	audioConfig *internal_voices.AudioConfig
}

func NewDeepgramOption(
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	opts utils.Option) (DeepgramOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &deepgramOption{
		key:         cx.(string),
		logger:      logger,
		options:     opts,
		audioConfig: audioConfig,
	}, nil
}

func (dgOpt *deepgramOption) GetKey() string {
	return dgOpt.key
}

func (dgOpt *deepgramOption) SpeechToTextOptions() *interfaces.LiveTranscriptionOptions {
	opts := &interfaces.LiveTranscriptionOptions{
		Model:          "nova-2",
		Language:       "en-US",
		Channels:       1,
		SmartFormat:    true,
		InterimResults: true,
		FillerWords:    true,
		VadEvents:      false,
		Endpointing:    "5",
		Punctuate:      true,
		NoDelay:        true,
		Encoding:       EncodingFromString(dgOpt.audioConfig.GetFormat()),
		SampleRate:     dgOpt.audioConfig.SampleRate,
		Diarize:        false,
		Multichannel:   false,
	}

	if sampleRate, err := dgOpt.options.GetUint32("listen.output_format.sample_rate"); err == nil {
		opts.SampleRate = int(sampleRate)
	}

	if encoding, err := dgOpt.options.GetString("listen.output_format.encoding"); err == nil {
		opts.Encoding = EncodingFromString(encoding)
	}

	if language, err := dgOpt.options.GetString("listen.language"); err == nil {
		opts.Language = language
	}
	if channels, err := dgOpt.options.GetUint32("listen.channel"); err == nil {
		opts.Channels = int(channels)
	}
	if smartFormat, err := dgOpt.options.GetBool("listen.smart_format"); err == nil {
		opts.SmartFormat = smartFormat
	}

	if fillerWords, err := dgOpt.options.GetBool("listen.filler_words"); err == nil {
		opts.FillerWords = fillerWords
	}
	if vadEvents, err := dgOpt.options.GetBool("listen.vad_events"); err == nil {
		opts.VadEvents = vadEvents
	}
	if endpointing, err := dgOpt.options.GetString("listen.endpointing"); err == nil {
		opts.Endpointing = endpointing
	}
	if multichannel, err := dgOpt.options.GetBool("listen.multichannel"); err == nil {
		opts.Multichannel = multichannel
	}
	if model, err := dgOpt.options.GetString("listen.model"); err == nil {
		opts.Model = model
	}
	if utteranceEndMs, err := dgOpt.options.GetString("listen.utterance_end"); err == nil {
		opts.UtteranceEndMs = utteranceEndMs
	}

	if keywordsRaw, exists := dgOpt.options["listen.keyword"]; exists {
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
	dgOpt.logger.Debugf("deepgram options %+v", opts)
	return opts
}

func (dgOpt *deepgramOption) TextToSpeechOptions() *interfaces.WSSpeakOptions {
	opts := &interfaces.WSSpeakOptions{
		Model:      "aura-asteria-en",
		Encoding:   EncodingFromString(dgOpt.audioConfig.GetFormat()),
		SampleRate: dgOpt.audioConfig.SampleRate,
	}

	if sampleRate, err := dgOpt.options.GetUint32("speak.output_format.sample_rate"); err == nil {
		opts.SampleRate = int(sampleRate)
	}

	if encoding, err := dgOpt.options.GetString("speak.output_format.encoding"); err == nil {
		opts.Encoding = EncodingFromString(encoding)
	}

	var model, voiceID, language string
	if modelValue, err := dgOpt.options.GetString("speak.model"); err == nil {
		model = modelValue
	}

	if voiceIDValue, err := dgOpt.options.GetString("speak.voice.id"); err == nil {
		voiceID = voiceIDValue
	}

	if languageValue, err := dgOpt.options.GetString("speak.language"); err == nil {
		language = languageValue
	}

	if model == "" && voiceID == "" && language == "" {
		opts.Model = "aura-asteria-en"
	} else {
		opts.Model = fmt.Sprintf("%s-%s-%s", model, voiceID, language)
	}

	return opts
}
