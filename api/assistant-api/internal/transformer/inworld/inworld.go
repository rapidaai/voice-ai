// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"fmt"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	// INWORLD_WSS_URL is the bidirectional streaming TTS endpoint.
	INWORLD_WSS_URL = "wss://api.inworld.ai/tts/v1/voice:streamBidirectional"

	// Defaults chosen to match Rapida's pcm_16000 encoding used by elevenlabs
	// and cartesia. Inworld's LINEAR16 @ 16000 Hz is byte-for-byte equivalent.
	INWORLD_AUDIO_ENCODING = "LINEAR16"
	INWORLD_SAMPLE_RATE    = 16000

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
// LINEAR16 @ 16000 Hz is equivalent to pcm_16000.
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

// GetTextToSpeechConnectionString returns the WebSocket URL. Inworld carries
// auth and configuration in headers and frames respectively, so no query
// parameters are needed here.
func (co *inworldOption) GetTextToSpeechConnectionString() string {
	return INWORLD_WSS_URL
}
