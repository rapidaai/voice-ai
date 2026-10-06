// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

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
		optionKeyRequestRules:  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.wav_base64"},"language":{"$path":"config.language"},"stream":false}}}]`,
		optionKeyResponseRules: `[{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
	}
}

func TestNewConfig_DefaultsAndOptionals(t *testing.T) {
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
		credentialKeyHeaders:      `{"Authorization":"Bearer token"}`,
	}), baseOptions())
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/predict", config.BaseURL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Equal(t, defaultEncoding, config.Encoding)
	assert.Equal(t, defaultSampleRate, config.SampleRate)
	assert.Empty(t, config.Model)
	assert.Empty(t, config.Language)
	assert.Len(t, config.RequestRules, 1)
	require.Len(t, config.ResponseRules, 1)
}

func TestNewConfig_WithOverrides(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyModel] = "stt-model"
	opts[optionKeyLanguage] = "hi"
	opts[optionKeySampleRate] = "8000"
	opts[optionKeyQueryParams] = `{"language":{"$var":"language"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "https://example.com/predict",
	}), opts)
	require.NoError(t, err)

	assert.Equal(t, "stt-model", config.Model)
	assert.Equal(t, "hi", config.Language)
	assert.Equal(t, 8000, config.SampleRate)
	assert.NotEmpty(t, config.QueryParams)
}

func TestNewConfig_ChecksSampleRateConversion(t *testing.T) {
	opts := baseOptions()
	opts[optionKeySampleRate] = uint32(math.MaxUint32)

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "https://example.com/predict",
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
			raw:         `{"language":{"$var":"language"`,
			wantErr:     true,
			errContains: optionKeyQueryParams,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := baseOptions()
			opts[optionKeyQueryParams] = test.raw

			config, err := NewConfig(testCredential(t, map[string]any{
				credentialKeyBaseURLCamel: "https://example.com/predict",
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
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyRequestRules)

	opts = baseOptions()
	delete(opts, optionKeyResponseRules)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseRules)
}

func TestNewConfig_RejectsInvalidRequestRulesAndAllowsEncodingOverride(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"state.audio"}}}}]`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "$path root")

	opts = baseOptions()
	opts[optionKeyEncoding] = "MuLaw8"
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.NoError(t, err)
	assert.Equal(t, "MuLaw8", config.Encoding)
}

func TestNewConfig_RequestRulesRequireAudioPacket(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyRequestRules] = `[{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start"}}}]`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `when.packet "audio"`)
}

func TestNewConfig_InvalidQueryParams(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"language":{"$var":"language"`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "https://example.com/predict",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyQueryParams)
}
