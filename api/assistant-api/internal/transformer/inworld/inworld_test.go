// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger()
	return l
}

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewInworldOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewInworldOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "illegal vault config")
}

func TestNewInworldOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- Encoding Tests ---

func TestInworldGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, "pcm_16000", opt.GetEncoding())
}

// --- Voice / Model Defaults ---

func TestInworldVoiceAndModelDefaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, INWORLD_DEFAULT_VOICE_ID, opt.GetVoiceID())
	assert.Equal(t, INWORLD_DEFAULT_MODEL_ID, opt.GetModelID())
}

func TestInworldVoiceAndModelOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "Hades",
		"speak.model":    "inworld-tts-1.5-mini",
	}
	opt, _ := NewInworldOption(newTestLogger(), cred, opts)
	assert.Equal(t, "Hades", opt.GetVoiceID())
	assert.Equal(t, "inworld-tts-1.5-mini", opt.GetModelID())
}

// --- GetTextToSpeechConnectionString ---

func TestInworldGetTextToSpeechConnectionString(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()
	// Auth and configuration live in headers and frames — URL is static.
	assert.Equal(t, "wss://api.inworld.ai/tts/v1/voice:streamBidirectional", connStr)
}

// --- Name ---

func TestInworldTTSName(t *testing.T) {
	// Name is fixed — we test it on a zero-valued instance to avoid dialing.
	tts := &inworldTTS{}
	assert.Equal(t, "inworld-text-to-speech", tts.Name())
}
