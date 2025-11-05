// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_assemblyai

import (
	"fmt"
	"net/url"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type TranscriptMessage struct {
	TurnOrder           int     `json:"turn_order"`
	TurnIsFormatted     bool    `json:"turn_is_formatted"`
	EndOfTurn           bool    `json:"end_of_turn"`
	Transcript          string  `json:"transcript"`
	EndOfTurnConfidence float64 `json:"end_of_turn_confidence"`
	Words               []Word  `json:"words"`
	Type                string  `json:"type"`
}

type Word struct {
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Text        string  `json:"text"`
	Confidence  float64 `json:"confidence"`
	WordIsFinal bool    `json:"word_is_final"`
}
type Encoding string

const (
	PCM_S16LE Encoding = "pcm_s16le"
	PCM_MULAW Encoding = "pcm_mulaw"
)

func EncodingFromString(encoding string) string {
	switch Encoding(encoding) {
	case "Linear16", PCM_S16LE:
		return string(PCM_S16LE)
	case "MuLaw8", PCM_MULAW:
		return string(PCM_MULAW)
	default:
		fmt.Printf("Warning: Invalid encoding option '%s'. Using default (linear16).", encoding)
		return string(PCM_S16LE)
	}
}

type AssemblyaiOption interface {
	GetSpeechToTextConnectionString() string
	GetKey() string
}

type assemblyaiOption struct {
	logger      commons.Logger
	key         string
	options     utils.Option
	audioConfig *internal_voices.AudioConfig
}

func NewAssemblyaiOption(
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	options utils.Option) (AssemblyaiOption, error) {
	cx, ok := vaultCredential.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &assemblyaiOption{
		logger:      logger,
		options:     options,
		audioConfig: audioConfig,
		key:         cx.(string),
	}, nil
}

func (co *assemblyaiOption) GetKey() string {
	return co.key
}

func (co *assemblyaiOption) GetSpeechToTextConnectionString() string {
	baseURL := "wss://streaming.assemblyai.com/v3/ws"
	params := url.Values{}
	params.Add("sample_rate", fmt.Sprintf("%d", co.audioConfig.SampleRate))
	params.Add("encoding", EncodingFromString(co.audioConfig.GetFormat()))

	// Check and add language
	if language, err := co.options.
		GetString("listen.language"); err == nil {
		params.Add("language", language)
	}

	// Check and add model
	if model, err := co.options.
		GetString("listen.model"); err == nil {
		params.Add("model", model)
	}

	// Check and add encoding
	if encoding, err := co.options.
		GetString("listen.output_format.encoding"); err == nil {
		params.Add("encoding", EncodingFromString(encoding))
	}

	// Check and add sample rate
	if sampleRate, err := co.options.
		GetString("listen.output_format.sample_rate"); err == nil {
		params.Add("sample_rate", sampleRate)
	}

	// Construct the final URL
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}
