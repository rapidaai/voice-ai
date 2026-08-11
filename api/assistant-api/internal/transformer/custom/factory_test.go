// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func testVaultCredential(t *testing.T, values map[string]any) *protos.VaultCredential {
	t.Helper()
	value, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.VaultCredential{Value: value}
}

func TestResolveCompatibility_Default(t *testing.T) {
	compatibility, err := ResolveCompatibility(nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultCompatibility, compatibility)
}

func TestResolveCompatibility_SupportsCamelAndSnake(t *testing.T) {
	compatibility, err := ResolveCompatibility(testVaultCredential(t, map[string]any{
		CredentialKeyAPICompatibilityCamel: "websocket_v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityWebSocketV1, compatibility)

	compatibility, err = ResolveCompatibility(testVaultCredential(t, map[string]any{
		CredentialKeyAPICompatibilitySnake: "websocket_v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityWebSocketV1, compatibility)

	compatibility, err = ResolveCompatibility(testVaultCredential(t, map[string]any{
		CredentialKeyAPICompatibilityCamel: "http_v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityHTTPV1, compatibility)
}

func TestResolveCompatibility_ValidateType(t *testing.T) {
	_, err := ResolveCompatibility(testVaultCredential(t, map[string]any{
		CredentialKeyAPICompatibilityCamel: 123,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestCompatibility_UsesProviderSpecificErrorLabels(t *testing.T) {
	_, err := compatibility("custom-tts", map[string]any{
		CredentialKeyAPICompatibilityCamel: 123,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom-tts: api compatibility must be a string")

	_, err = compatibility("custom-stt", map[string]any{
		CredentialKeyAPICompatibilityCamel: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom-stt: api compatibility must not be empty")
}

func TestNewTextToSpeech_UnsupportedCompatibility(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	_, err := NewTextToSpeech(
		context.Background(),
		logger,
		testVaultCredential(t, map[string]any{
			CredentialKeyAPICompatibilityCamel: "unknown",
			"baseUrl":                          "wss://example.invalid/ws",
		}),
		func(pkt ...internal_type.Packet) error { return nil },
		utils.Option{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported api compatibility")
}

func TestNewSpeechToText_UnsupportedCompatibility(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	_, err := NewSpeechToText(
		WithContext(context.Background()),
		WithLogger(logger),
		WithCredential(testVaultCredential(t, map[string]any{
			CredentialKeyAPICompatibilityCamel: "unknown",
			"baseUrl":                          "wss://example.invalid/ws",
		})),
		WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
		WithOptions(utils.Option{}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported api compatibility")
}

func TestNewSpeechToText_HTTPCompatibility(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	transformer, err := NewSpeechToText(
		WithContext(context.Background()),
		WithLogger(logger),
		WithCredential(testVaultCredential(t, map[string]any{
			CredentialKeyAPICompatibilityCamel: "http_v1",
			"baseUrl":                          "https://example.invalid/predict",
		})),
		WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
		WithOptions(utils.Option{
			"listen.request_rules":  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.wav_base64"}}}}]`,
			"listen.response_rules": `[{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "custom-stt-http-v1", transformer.Name())
}
