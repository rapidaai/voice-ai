// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_cartesia

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	cartesia_internal "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	URL                      = "wss://api.cartesia.ai/stt/websocket"
	CARTESIA_API_VERSION     = "2024-06-10"
	CARTESIA_STT_API_VERSION = "2026-03-01"
	RECONNECT_DELAY          = 5 * time.Second
	WRITE_TIME_OUT           = 10 * time.Second
)

func (co *cartesiaOption) GetEncoding() string {
	return "pcm_s16le"
}

type cartesiaOption struct {
	key     string
	mdlOpts utils.Option
	logger  commons.Logger
}

func NewCartesiaOption(logger commons.Logger,
	vltC *protos.VaultCredential,
	opts utils.Option) (*cartesiaOption, error) {
	cx, ok := vltC.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("unable to get config parameters from vaults")
	}
	return &cartesiaOption{
		logger:  logger,
		mdlOpts: opts,
		key:     cx.(string),
	}, nil
}

func (co *cartesiaOption) GetTextToSpeechInput(
	transcript string,
	overriddenOpts map[string]interface{},
) cartesia_internal.TextToSpeechInput {
	opts := cartesia_internal.TextToSpeechInput{
		ModelID: "sonic-2-2025-03-07",
		Voice: cartesia_internal.TextToSpeechVoice{
			Mode: "id",
			ID:   "c2ac25f9-ecc4-4f56-9095-651354df60c0",
		},
		OutputFormat: cartesia_internal.TextToSpeechOutputFormat{
			Container:  "raw",
			Encoding:   co.GetEncoding(),
			SampleRate: 16000,
		},
		Transcript:    transcript,
		AddTimestamps: false,
	}

	if speed, err := co.mdlOpts.GetString(internal_options.SpeakOptionExperimentalControlsSpeed); err == nil {
		opts.ExperimentalControls.Speed = speed
	}

	if emotion, err := co.mdlOpts.GetString(internal_options.SpeakOptionExperimentalControlsEmotion); err == nil {
		opts.ExperimentalControls.Emotion = strings.Split(emotion, commons.SEPARATOR)
	}

	if language, err := co.mdlOpts.GetString(internal_options.SpeakOptionLanguage); err == nil {
		opts.Language = language
	}

	if model, err := co.mdlOpts.GetString(internal_options.SpeakOptionModel); err == nil {
		opts.ModelID = model
	}
	if voice, err := co.mdlOpts.GetString(internal_options.SpeakOptionVoiceID); err == nil {
		opts.Voice = cartesia_internal.TextToSpeechVoice{
			Mode: "id",
			ID:   voice,
		}

	}
	v, ok := overriddenOpts["continue"]
	if ok {
		opts.Continue = v.(bool)
	}
	contextID, ok := overriddenOpts["context_id"]
	if ok {
		opts.ContextID = contextID.(string)
	}

	return opts
}

func (co *cartesiaOption) GetSpeechToTextConnectionString() string {
	params := url.Values{}
	params.Add("encoding", co.GetEncoding())
	params.Add("sample_rate", "16000")

	model := "ink-2"
	if configuredModel, err := co.mdlOpts.GetString(internal_options.ListenOptionModel); err == nil {
		model = configuredModel
	}
	params.Add("model", model)

	// Check and add language
	if language, err := co.mdlOpts.GetString(internal_options.ListenOptionLanguage); err == nil {
		params.Add("language", language)
	}

	// Construct the final URL
	return fmt.Sprintf("%s?%s", URL, params.Encode())
}

func (co *cartesiaOption) GetSpeechToTextHeader() http.Header {
	header := http.Header{}
	header.Set("X-API-Key", co.key)
	header.Set("Cartesia-Version", CARTESIA_STT_API_VERSION)
	return header
}

func (co *cartesiaOption) GetTextToSpeechConnectionString() string {
	baseURL := "wss://api.cartesia.ai/tts/websocket"
	params := url.Values{}
	params.Add("api_key", co.key)
	params.Add("cartesia_version", CARTESIA_API_VERSION)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}
