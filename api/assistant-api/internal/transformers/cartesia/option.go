// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_cartesia

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type TextToSpeechVoice struct {
	Mode string `json:"mode"`
	ID   string `json:"id"`
}

type TextToSpeechExperimentalControls struct {
	Speed   string   `json:"speed"`
	Emotion []string `json:"emotion"`
}

type TextToSpeechOutputFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

type TextToSpeechInput struct {
	ModelID              string                           `json:"model_id"`
	ContextID            string                           `json:"context_id"`
	Transcript           string                           `json:"transcript"`
	Voice                TextToSpeechVoice                `json:"voice"`
	ExperimentalControls TextToSpeechExperimentalControls `json:"__experimental_controls"`
	OutputFormat         TextToSpeechOutputFormat         `json:"output_format"`
	Language             string                           `json:"language"`
	Continue             bool                             `json:"continue"`
	AddTimestamps        bool                             `json:"add_timestamps"`
}

type TextToSpeechOuput struct {
	Type       string `json:"type"`
	Data       string `json:"data"`
	Done       bool   `json:"done"`
	StatusCode int    `json:"status_code"`
	ContextID  string `json:"context_id"`
}

type TranscriptWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type SpeechToTextOutput struct {
	Type      string           `json:"type"`
	IsFinal   bool             `json:"is_final"`
	RequestID string           `json:"request_id"`
	Text      string           `json:"text"`
	Duration  float64          `json:"duration"`
	Language  string           `json:"language"`
	Words     []TranscriptWord `json:"words"`
}

type Encoding string

const (
	CARTESIA_API_VERSION = "2024-06-10"

	RECONNECT_DELAY = 5 * time.Second
	WRITE_TIME_OUT  = 10 * time.Second
	// PCM_F32LE represents 32-bit floating-point PCM encoding
	PCM_F32LE Encoding = "pcm_f32le"
	// PCM_S16LE represents 16-bit signed integer PCM encoding
	PCM_S16LE Encoding = "pcm_s16le"
	// PCM_MULAW represents mu-law PCM encoding
	PCM_MULAW Encoding = "pcm_mulaw"
	//
	PCM_ALAW Encoding = "pcm_alaw"
)

func (m Encoding) String() string {
	return string(m)
}

func EncodingFromString(encoding string) string {
	switch Encoding(encoding) {
	case PCM_S16LE, "Linear16":
		return PCM_S16LE.String()
	case PCM_MULAW, "MuLaw8":
		return PCM_MULAW.String()
	default:
		fmt.Printf("Warning: Invalid encoding option '%s'. Using default (linear16).", encoding)
		return string(PCM_S16LE)
	}
}

type CartesiaOption interface {
	GetTextToSpeechInput(
		transcript string,
		oOpts map[string]interface{},
	) TextToSpeechInput
	GetSpeechToTextConnectionString() string
	GetTextToSpeechConnectionString() string
}

type cartesiaOption struct {
	key         string
	options     utils.Option
	logger      commons.Logger
	audioConfig *internal_voices.AudioConfig
}

func NewCartesiaOption(logger commons.Logger,
	vltC *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig,
	opts utils.Option) (CartesiaOption, error) {
	cx, ok := vltC.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &cartesiaOption{
		logger:      logger,
		options:     opts,
		audioConfig: audioConfig,
		key:         cx.(string),
	}, nil
}

func (co *cartesiaOption) GetTextToSpeechInput(
	transcript string,
	overriddenOpts map[string]interface{},
) TextToSpeechInput {
	opts := TextToSpeechInput{
		ModelID: "sonic-2-2025-03-07",
		Voice: TextToSpeechVoice{
			Mode: "id",
			ID:   "c2ac25f9-ecc4-4f56-9095-651354df60c0",
		},
		OutputFormat: TextToSpeechOutputFormat{
			Container:  "raw",
			Encoding:   EncodingFromString(co.audioConfig.GetFormat()),
			SampleRate: co.audioConfig.SampleRate,
		},
		Transcript:    transcript,
		AddTimestamps: false,
	}

	if encoding, err := co.options.GetString("speak.output_format.encoding"); err == nil {
		opts.OutputFormat.Encoding = EncodingFromString(encoding)
	}

	if sampleRate, err := co.options.GetUint32("speak.output_format.sample_rate"); err == nil {
		opts.OutputFormat.SampleRate = int(sampleRate)
	}

	if speed, err := co.options.GetString("speak.__experimental_controls.speed"); err == nil {
		opts.ExperimentalControls.Speed = speed
	}

	if emotion, err := co.options.GetString("speak.__experimental_controls.emotion"); err == nil {
		opts.ExperimentalControls.Emotion = strings.Split(emotion, commons.SEPARATOR)
	}

	if language, err := co.options.GetString("speak.language"); err == nil {
		opts.Language = language
	}

	if model, err := co.options.GetString("speak.model"); err == nil {
		opts.ModelID = model
	}
	if voice, err := co.options.GetString("speak.voice.id"); err == nil {
		opts.Voice = TextToSpeechVoice{
			Mode: "id",
			ID:   voice,
		}

	}
	v, ok := overriddenOpts["continue"]
	if ok {
		opts.Continue = v.(bool)
	}
	ctxId, ok := overriddenOpts["context_id"]
	if ok {
		opts.ContextID = ctxId.(string)
	}

	co.logger.Debugf("cartesia options %+v", opts)
	return opts
}

func (co *cartesiaOption) GetSpeechToTextConnectionString() string {
	baseURL := "wss://api.cartesia.ai/stt/websocket"
	params := url.Values{}
	params.Add("api_key", co.key)
	params.Add("cartesia_version", CARTESIA_API_VERSION)
	params.Add("encoding", EncodingFromString(co.audioConfig.GetFormat()))
	params.Add("sample_rate", fmt.Sprintf("%d", co.audioConfig.GetSampleRate()))

	// Check and add language
	if language, err := co.options.GetString("listen.language"); err == nil {
		params.Add("language", language)
	}

	// Check and add model
	if model, err := co.options.GetString("listen.model"); err == nil {
		params.Add("model", model)
	}

	// Check and add encoding
	if encoding, err := co.options.GetString("listen.output_format.encoding"); err == nil {
		params.Add("encoding", EncodingFromString(encoding))
	}

	// Check and add sample rate
	if sampleRate, err := co.options.GetString("listen.output_format.sample_rate"); err == nil {
		params.Add("sample_rate", sampleRate)
	}

	// Construct the final URL
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (co *cartesiaOption) GetTextToSpeechConnectionString() string {
	baseURL := "wss://api.cartesia.ai/tts/websocket"
	params := url.Values{}
	params.Add("api_key", co.key)
	params.Add("cartesia_version", CARTESIA_API_VERSION)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}
