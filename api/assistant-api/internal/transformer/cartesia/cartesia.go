// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_cartesia

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
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

const (
	URL                  = "wss://api.cartesia.ai/stt/websocket"
	CARTESIA_API_VERSION = "2024-06-10"
	RECONNECT_DELAY      = 5 * time.Second
	WRITE_TIME_OUT       = 10 * time.Second
)

func (co *cartesiaOption) GetEncoding() string {
	switch co.audioConfig.GetAudioFormat() {
	case protos.AudioConfig_LINEAR16:
		return "pcm_s16le"
	case protos.AudioConfig_MuLaw8:
		return "pcm_mulaw"
	default:
		fmt.Printf("Warning: Invalid encoding option '%s'. Using default (linear16).", co.audioConfig.GetAudioFormat())
		return "pcm_s16le"
	}
}

type cartesiaOption struct {
	key         string
	mdlOpts     utils.Option
	logger      commons.Logger
	audioConfig *protos.AudioConfig
}

func NewCartesiaOption(logger commons.Logger,
	vltC *protos.VaultCredential,
	audioConfig *protos.AudioConfig,
	opts utils.Option) (*cartesiaOption, error) {
	cx, ok := vltC.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("unable to get config parameters from vaults")
	}
	return &cartesiaOption{
		logger:      logger,
		mdlOpts:     opts,
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
			Encoding:   co.GetEncoding(),
			SampleRate: int(co.audioConfig.GetSampleRate()),
		},
		Transcript:    transcript,
		AddTimestamps: false,
	}

	if speed, err := co.mdlOpts.GetString("speak.__experimental_controls.speed"); err == nil {
		opts.ExperimentalControls.Speed = speed
	}

	if emotion, err := co.mdlOpts.GetString("speak.__experimental_controls.emotion"); err == nil {
		opts.ExperimentalControls.Emotion = strings.Split(emotion, commons.SEPARATOR)
	}

	if language, err := co.mdlOpts.GetString("speak.language"); err == nil {
		opts.Language = language
	}

	if model, err := co.mdlOpts.GetString("speak.model"); err == nil {
		opts.ModelID = model
	}
	if voice, err := co.mdlOpts.GetString("speak.voice.id"); err == nil {
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

	return opts
}

func (co *cartesiaOption) GetSpeechToTextConnectionString() string {
	params := url.Values{}
	params.Add("api_key", co.key)
	params.Add("cartesia_version", CARTESIA_API_VERSION)
	params.Add("encoding", co.GetEncoding())
	params.Add("sample_rate", fmt.Sprintf("%d", co.audioConfig.GetSampleRate()))
	// Check and add language
	if language, err := co.mdlOpts.GetString("listen.language"); err == nil {
		params.Add("language", language)
	}

	// Check and add model
	if model, err := co.mdlOpts.GetString("listen.model"); err == nil {
		params.Add("model", model)
	}
	// Construct the final URL
	return fmt.Sprintf("%s?%s", URL, params.Encode())
}

func (co *cartesiaOption) GetTextToSpeechConnectionString() string {
	baseURL := "wss://api.cartesia.ai/tts/websocket"
	params := url.Values{}
	params.Add("api_key", co.key)
	params.Add("cartesia_version", CARTESIA_API_VERSION)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}
