// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"math"
	"testing"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func testCredential(t *testing.T, values map[string]any) *protos.VaultCredential {
	t.Helper()
	value, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.VaultCredential{Value: value}
}

func baseOptions() utils.Option {
	return utils.Option{
		optionKeyRequestRules:  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}}]`,
		optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
	}
}

func TestNewConfig_DefaultsAndOptionals(t *testing.T) {
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
		credentialKeyHeaders:      `{"Authorization":"Bearer token"}`,
	}), baseOptions())
	require.NoError(t, err)

	assert.Equal(t, "wss://example.com/ws", config.BaseURL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Empty(t, config.VoiceID)
	assert.Equal(t, defaultEncoding, config.Encoding)
	assert.Equal(t, defaultSampleRate, config.SampleRate)
	assert.Empty(t, config.Model)
	assert.Empty(t, config.Language)
	require.Len(t, config.RequestRules, 1)
	assert.Equal(t, requestPacketText, config.RequestRules[0].When.Packet)
}

func TestNewConfig_WithOverrides(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyVoiceID] = "voice-1"
	opts[optionKeyEncoding] = "MuLaw8"
	opts[optionKeySampleRate] = "48000"
	opts[optionKeyModel] = "my-model"
	opts[optionKeyLanguage] = "hi-IN"
	opts[optionKeyQueryParams] = `{"lang":{"$var":"language"}}`
	opts[optionKeyRequestRules] = `[
		{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"message_id":{"$path":"packet.message_id"}}}},
		{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done","message_id":{"$path":"packet.message_id"}}}}
	]`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/ws",
	}), opts)
	require.NoError(t, err)

	assert.Equal(t, "MuLaw8", config.Encoding)
	assert.Equal(t, 48000, config.SampleRate)
	assert.Equal(t, "voice-1", config.VoiceID)
	assert.Equal(t, "my-model", config.Model)
	assert.Equal(t, "hi-IN", config.Language)
	assert.NotNil(t, config.QueryParams)
	require.Len(t, config.RequestRules, 2)
	assert.Equal(t, requestPacketText, config.RequestRules[0].When.Packet)
	assert.Equal(t, requestPacketDone, config.RequestRules[1].When.Packet)
}

func TestNewConfig_ChecksSampleRateConversion(t *testing.T) {
	opts := baseOptions()
	opts[optionKeySampleRate] = uint32(math.MaxUint32)

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/ws",
	}), opts)
	expected, conversionErr := utils.Uint32ToInt(math.MaxUint32)
	if conversionErr != nil {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	assert.Equal(t, expected, config.SampleRate)
}

func TestNewConfig_ValidateRequired(t *testing.T) {
	_, err := NewConfig(testCredential(t, map[string]any{}), baseOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base url")

	opts := baseOptions()
	delete(opts, optionKeyRequestRules)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyRequestRules)

	opts = baseOptions()
	delete(opts, optionKeyResponseRules)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseRules)
}

func TestNewConfig_InvalidJSON(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyRequestRules)

	opts = baseOptions()
	opts[optionKeyResponseRules] = `{"bad":"shape"}`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseRules)

	opts = baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}] trailing`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing content")
}

func TestNewConfig_OptionalQueryParamsValidateOnlyNonBlankStrings(t *testing.T) {
	tests := []struct {
		name string
		raw  any
	}{
		{name: "empty string", raw: ""},
		{name: "blank string", raw: "   "},
		{name: "non string", raw: map[string]any{"language": map[string]any{"$var": "language"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := baseOptions()
			opts[optionKeyQueryParams] = tt.raw

			config, err := NewConfig(testCredential(t, map[string]any{
				credentialKeyBaseURLCamel: "wss://example.com/ws",
			}), opts)
			require.NoError(t, err)
			assert.Empty(t, config.QueryParams)
		})
	}

	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"language":{"$var":"language"`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyQueryParams)
}

func TestNewConfig_QueryParamsMustResolveToPrimitive(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"metadata":{"lang":"hi"}}`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must resolve to primitive value")
}

func TestNewConfig_QueryParamsRejectTextVariable(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"q":{"$var":"text"}}`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported variable "text"`)
}

func TestNewConfig_QueryParamsAllowSupportedVariables(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyModel] = "model-a"
	opts[optionKeyLanguage] = "en-US"
	opts[optionKeyQueryParams] = `{
		"message_id":{"$var":"message_id"},
		"voice_id":{"$var":"voice_id"},
		"model":{"$var":"model"},
		"language":{"$var":"language"},
		"encoding":{"$var":"encoding"},
		"sample_rate":{"$var":"sample_rate"}
	}`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.NoError(t, err)
	assert.NotNil(t, config.QueryParams)
}

func TestNewConfig_RequestRulesRequireTextPacket(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done"}}}]`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `when.packet "text"`)
}
