// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_smallest

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	smallest_internal "github.com/rapidaai/api/assistant-api/internal/transformer/smallest/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	// SPEECH_TO_TEXT_URL is Smallest's Pulse realtime WebSocket endpoint.
	// Only the "pulse" model is served here — Pulse Pro is pre-recorded/HTTP
	// only and returns 400 on this endpoint, so listen.model is intentionally
	// not forwarded to the connection string below.
	SPEECH_TO_TEXT_URL = "wss://api.smallest.ai/waves/v1/pulse/get_text"

	// TEXT_TO_SPEECH_URL is Smallest's Lightning streaming WebSocket endpoint.
	// Both lightning_v3.1 and lightning_v3.1_pro are selected per-message via
	// the "model" field in the JSON body, not via the connection string.
	TEXT_TO_SPEECH_URL = "wss://api.smallest.ai/waves/v1/tts/live"

	// DefaultSampleRate matches the raw PCM 16kHz mono contract used
	// throughout the rest of the audio pipeline (denoise/VAD/EOS).
	DefaultSampleRate = 16000

	DefaultTextToSpeechModel = "lightning_v3.1"
	DefaultVoiceID           = "magnus"

	// SourceHeader/SourceName identify this integration to Smallest so usage
	// can be attributed to Rapida, mirroring the X-Source header other SDK
	// integrations send on connect.
	SourceHeader = "X-Source"
	SourceName   = "rapida"
)

// setSourceHeaders stamps the outbound WebSocket upgrade request with this
// integration's identity so Smallest can attribute traffic to Rapida.
func setSourceHeaders(header http.Header) {
	header.Set(SourceHeader, SourceName)
}

type smallestOption struct {
	key     string
	mdlOpts utils.Option
	logger  commons.Logger
}

func NewSmallestOption(logger commons.Logger,
	vltC *protos.VaultCredential,
	opts utils.Option) (*smallestOption, error) {
	cx, ok := vltC.GetValue().AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("smallest: missing 'key' in vault credential")
	}
	key, ok := cx.(string)
	if !ok {
		return nil, fmt.Errorf("smallest: vault 'key' must be a string, got %T", cx)
	}
	return &smallestOption{
		logger:  logger,
		mdlOpts: opts,
		key:     key,
	}, nil
}

// GetKey returns the API key used both as the STT/TTS WebSocket
// Authorization header value.
func (so *smallestOption) GetKey() string {
	return so.key
}

// GetEncoding returns the PCM encoding literal Smallest expects on the Pulse
// realtime query string ("linear16" == 16-bit signed little-endian PCM).
func (so *smallestOption) GetEncoding() string {
	return "linear16"
}

func (so *smallestOption) GetSpeechToTextConnectionString() string {
	params := url.Values{}
	params.Add("encoding", so.GetEncoding())
	params.Add("sample_rate", fmt.Sprintf("%d", DefaultSampleRate))

	if language, err := so.mdlOpts.GetString(internal_options.ListenOptionLanguage); err == nil && language != "" {
		params.Add("language", language)
	}
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionWordTimestamps); err == nil {
		params.Add("word_timestamps", strconv.FormatBool(v))
	}
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionSentenceTimestamps); err == nil {
		params.Add("sentence_timestamps", strconv.FormatBool(v))
	}
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionDiarize); err == nil {
		params.Add("diarize", strconv.FormatBool(v))
	}
	// PII/PCI redaction: Names, addresses, phone numbers (PII) and card
	// numbers, CVVs, ZIP codes, account numbers (PCI) are replaced with
	// [ENTITYTYPE_N] placeholder tokens in the finalized transcript.
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionRedactPII); err == nil {
		params.Add("redact_pii", strconv.FormatBool(v))
	}
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionRedactPCI); err == nil {
		params.Add("redact_pci", strconv.FormatBool(v))
	}
	if numerals, err := so.mdlOpts.GetString(internal_options.ListenOptionNumerals); err == nil && numerals != "" {
		params.Add("numerals", numerals)
	}
	// format controls punctuation/capitalization in the transcript (Smallest
	// defaults this to true server-side; disabling returns raw lowercase text).
	// Reuses the shared smart-format option key rather than a Smallest-only one.
	if v, err := so.mdlOpts.GetBool(internal_options.ListenOptionSmartFormat); err == nil {
		params.Add("format", strconv.FormatBool(v))
	}
	return fmt.Sprintf("%s?%s", SPEECH_TO_TEXT_URL, params.Encode())
}

func (so *smallestOption) GetTextToSpeechConnectionString() string {
	return TEXT_TO_SPEECH_URL
}

// GetTextToSpeechInput builds the per-message JSON body sent to Smallest's
// Lightning streaming endpoint. overriddenOpts carries per-call session
// state ("continue", "flush", "context_id") set by the caller in tts.go.
func (so *smallestOption) GetTextToSpeechInput(
	text string,
	overriddenOpts map[string]interface{},
) smallest_internal.TextToSpeechInput {
	opts := smallest_internal.TextToSpeechInput{
		Model:      DefaultTextToSpeechModel,
		VoiceID:    DefaultVoiceID,
		SampleRate: DefaultSampleRate,
		Text:       text,
	}

	if language, err := so.mdlOpts.GetString(internal_options.SpeakOptionLanguage); err == nil && language != "" {
		opts.Language = language
	}
	if model, err := so.mdlOpts.GetString(internal_options.SpeakOptionModel); err == nil && model != "" {
		opts.Model = model
	}
	if voice, err := so.mdlOpts.GetString(internal_options.SpeakOptionVoiceID); err == nil && voice != "" {
		opts.VoiceID = voice
	}
	if speed, err := so.mdlOpts.GetFloat64(internal_options.SpeakOptionSpeed); err == nil && speed > 0 {
		opts.Speed = speed
	}

	if v, ok := overriddenOpts["continue"]; ok {
		opts.Continue = v.(bool)
	}
	if v, ok := overriddenOpts["flush"]; ok {
		opts.Flush = v.(bool)
	}
	if contextID, ok := overriddenOpts["context_id"]; ok {
		opts.ContextID = contextID.(string)
	}

	return opts
}
