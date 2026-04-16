// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	// INWORLD_STREAM_URL is Inworld's HTTP streaming TTS endpoint. Each POST
	// returns an NDJSON stream of audio chunks and closes when synthesis
	// finishes. Rapida's aggregator already splits LLM deltas at sentence
	// boundaries, so we issue one request per sentence.
	INWORLD_STREAM_URL = "https://api.inworld.ai/tts/v1/voice:stream"

	// Defaults chosen to match Rapida's pcm_16000 encoding used by elevenlabs
	// and cartesia. Inworld's PCM encoding is raw little-endian 16-bit
	// samples with no container — byte-for-byte what the Rapida pipeline
	// already consumes. (LINEAR16 is the same samples but wraps every
	// streaming NDJSON chunk in a RIFF/WAVE header, which would have to
	// be stripped to avoid clicks; PCM avoids the wrapper entirely.)
	INWORLD_AUDIO_ENCODING = "PCM"
	INWORLD_SAMPLE_RATE    = 16000

	// INWORLD_USER_AGENT identifies traffic originating from the Rapida
	// integration so Inworld can bucket usage by SDK. Sent as X-User-Agent
	// on every synth request.
	INWORLD_USER_AGENT = "rapida-sdk"

	// Defaults for voice and model. Both can be overridden via utils.Option
	// keys "speak.voice.id" and "speak.model".
	INWORLD_DEFAULT_VOICE_ID = "Ashley"
	INWORLD_DEFAULT_MODEL_ID = "inworld-tts-1.5-max"
)

// inworldOption holds the resolved credential and per-conversation options
// for the Inworld TTS transformer.
type inworldOption struct {
	key     string
	logger  commons.Logger
	mdlOpts utils.Option
}

// NewInworldOption validates the vault credential and returns a ready-to-use
// option struct. The key is expected to already be a pre-encoded basic-auth
// token (Inworld issues keys as base64 "client:secret" pairs).
func NewInworldOption(logger commons.Logger, vaultCredential *protos.VaultCredential,
	opts utils.Option) (*inworldOption, error) {
	if vaultCredential == nil {
		return nil, fmt.Errorf("inworld: nil vault credential")
	}
	val := vaultCredential.GetValue()
	if val == nil {
		return nil, fmt.Errorf("inworld: nil vault value")
	}
	cx, ok := val.AsMap()["key"]
	if !ok {
		return nil, fmt.Errorf("inworld: illegal vault config")
	}
	key, ok := cx.(string)
	if !ok {
		return nil, fmt.Errorf("inworld: vault key is not a string")
	}
	if key == "" {
		return nil, fmt.Errorf("inworld: empty vault key")
	}
	return &inworldOption{
		key:     key,
		logger:  logger,
		mdlOpts: opts,
	}, nil
}

// GetKey returns the raw basic-auth token to use in the Authorization header.
func (co *inworldOption) GetKey() string {
	return co.key
}

// GetEncoding returns Rapida's canonical encoding identifier. Inworld's
// PCM @ 16000 Hz is raw little-endian 16-bit samples — identical to
// pcm_16000.
func (co *inworldOption) GetEncoding() string {
	return "pcm_16000"
}

// GetVoiceID returns the configured voice, or the default if unset.
func (co *inworldOption) GetVoiceID() string {
	if v, err := co.mdlOpts.GetString("speak.voice.id"); err == nil && v != "" {
		return v
	}
	return INWORLD_DEFAULT_VOICE_ID
}

// GetModelID returns the configured TTS model, or the default if unset.
func (co *inworldOption) GetModelID() string {
	if m, err := co.mdlOpts.GetString("speak.model"); err == nil && m != "" {
		return m
	}
	return INWORLD_DEFAULT_MODEL_ID
}

// GetTextToSpeechConnectionString returns the HTTP streaming URL. Auth and
// configuration live in the Authorization header and request body; there
// are no query parameters.
func (co *inworldOption) GetTextToSpeechConnectionString() string {
	return INWORLD_STREAM_URL
}

// newInworldHTTPClient returns an *http.Client tuned for the Inworld TTS
// streaming endpoint. We keep a small idle-conn pool so successive
// sentences reuse the same TCP+TLS connection (and an HTTP/2 multiplex slot
// if Inworld supports it), which is what makes HTTP streaming latency
// competitive with the WebSocket approach after the first request.
//
// No top-level timeout is set — per-synth deadlines are carried through the
// request context so a canceled turn unblocks in-flight reads immediately.
func newInworldHTTPClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &http.Client{Transport: tr}
}
