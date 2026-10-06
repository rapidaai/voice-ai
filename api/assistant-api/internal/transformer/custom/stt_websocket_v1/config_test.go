// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

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
		optionKeyRequestRules:  `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]`,
		optionKeyResponseRules: `[{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
	}
}

func TestNewConfig_DefaultsAndOptionals(t *testing.T) {
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
		credentialKeyHeaders:      `{"Authorization":"Bearer token"}`,
	}), baseOptions())
	require.NoError(t, err)

	assert.Equal(t, "wss://example.com/stt", config.BaseURL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Equal(t, defaultEncoding, config.Encoding)
	assert.Equal(t, defaultSampleRate, config.SampleRate)
	assert.Empty(t, config.Model)
	assert.Empty(t, config.Language)
	require.Len(t, config.RequestRules, 1)
	assert.Equal(t, requestPacketAudio, config.RequestRules[0].When.Packet)
}

func TestNewConfig_WithOverrides(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyModel] = "model-a"
	opts[optionKeyLanguage] = "hi-IN"
	opts[optionKeyEncoding] = "MuLaw8"
	opts[optionKeySampleRate] = "8000"
	opts[optionKeyQueryParams] = `{"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}`
	opts[optionKeyRequestRules] = `[
		{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start","language":{"$path":"config.language"}}}},
		{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.base64"},"encoding":{"$path":"config.audio.encoding"}}}}
	]`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/stt",
	}), opts)
	require.NoError(t, err)

	assert.Equal(t, "model-a", config.Model)
	assert.Equal(t, "hi-IN", config.Language)
	assert.Equal(t, "MuLaw8", config.Encoding)
	assert.Equal(t, 8000, config.SampleRate)
	assert.NotNil(t, config.QueryParams)
	require.Len(t, config.RequestRules, 2)
	assert.Equal(t, requestPacketTurnChange, config.RequestRules[0].When.Packet)
	assert.Equal(t, requestPacketAudio, config.RequestRules[1].When.Packet)
}

func TestNewConfig_ChecksSampleRateConversion(t *testing.T) {
	opts := baseOptions()
	opts[optionKeySampleRate] = uint32(math.MaxUint32)

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/stt",
	}), opts)
	expected, conversionErr := utils.Uint32ToInt(math.MaxUint32)
	if conversionErr != nil {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	assert.Equal(t, expected, config.SampleRate)
}

func TestNewConfig_OptionalQueryParams(t *testing.T) {
	tests := []struct {
		name        string
		raw         any
		want        map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "empty string leaves query params empty",
			raw:  "",
			want: map[string]any{},
		},
		{
			name: "blank string leaves query params empty",
			raw:  " \t\n ",
			want: map[string]any{},
		},
		{
			name:        "invalid json errors with option key",
			raw:         `{"sample_rate":{"$cast":"number"`,
			wantErr:     true,
			errContains: optionKeyQueryParams,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := baseOptions()
			opts[optionKeyQueryParams] = test.raw

			config, err := NewConfig(testCredential(t, map[string]any{
				credentialKeyBaseURLCamel: "wss://example.com/stt",
			}), opts)
			if test.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, config.QueryParams)
		})
	}
}

func TestNewConfig_ValidateRequired(t *testing.T) {
	_, err := NewConfig(testCredential(t, map[string]any{}), baseOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base url")

	opts := baseOptions()
	delete(opts, optionKeyRequestRules)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyRequestRules)

	opts = baseOptions()
	delete(opts, optionKeyResponseRules)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseRules)
}

func TestNewConfig_InvalidJSON(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyRequestRules)

	opts = baseOptions()
	opts[optionKeyResponseRules] = `{"bad":"shape"}`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseRules)

	opts = baseOptions()
	opts[optionKeyResponseRules] = `[{"when":{"frame":"json"},"emit":{"unexpected":true}}]`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "emit.unexpected")
}

func TestNewConfig_TextResponseRules(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyResponseRules] = `[{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"interim":false}}]`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.NoError(t, err)
	require.Len(t, config.ResponseRules, 1)
	assert.Equal(t, frameTypeText, config.ResponseRules[0].When.Frame)
}

func TestNewConfig_TextResponseRulesRejectPath(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyResponseRules] = `[{"when":{"frame":"text","path":"type","equals":"partial"},"emit":{"script":{"$frame":"text"},"interim":true}}]`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "when.path cannot be used")
}

func TestNewConfig_QueryParamsRejectAudioVariable(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"chunk":{"$var":"audio"}}`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported variable "audio"`)
}

func TestNewConfig_RequestRulesRequireAudioPacket(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start"}}}]`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `when.packet "audio"`)
}
