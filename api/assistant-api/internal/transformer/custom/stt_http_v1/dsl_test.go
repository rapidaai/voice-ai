// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

import (
	"encoding/binary"
	"math"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSLEngine_BuildRequestURLAndEvaluateRequestRules(t *testing.T) {
	config := &Config{
		BaseURL:    "https://example.com/predict?existing=1",
		Model:      "stt-model",
		Language:   "hi",
		Encoding:   defaultEncoding,
		SampleRate: 16000,
		QueryParams: map[string]any{
			"language":    map[string]any{"$var": "language"},
			"sample_rate": map[string]any{"$cast": "number", "value": map[string]any{"$var": "sample_rate"}},
		},
		RequestRules: []RequestRule{
			{
				When: RequestWhen{Packet: requestPacketAudio},
				Send: RequestSend{
					Frame: frameTypeJSON,
					Body: map[string]any{
						"audio":            map[string]any{"$path": "packet.audio.wav_base64"},
						"language":         map[string]any{"$path": "config.language"},
						"speech_enhance":   true,
						"max_tokens":       1024,
						"sample_rate_copy": map[string]any{"$cast": "number", "value": map[string]any{"$path": "config.audio.sample_rate"}},
					},
				},
			},
		},
	}

	engine := config.newEngine()
	requestURL, err := engine.BuildRequestURL(config.newQueryScope())
	require.NoError(t, err)

	parsedURL, err := url.Parse(requestURL)
	require.NoError(t, err)
	assert.Equal(t, "1", parsedURL.Query().Get("existing"))
	assert.Equal(t, "hi", parsedURL.Query().Get("language"))
	assert.Equal(t, "16000", parsedURL.Query().Get("sample_rate"))

	scope, err := config.newRequestScope("ctx-1", []byte{0x00, 0x01})
	require.NoError(t, err)
	requests, err := engine.EvaluateRequestRules(requestPacketAudio, scope)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, frameTypeJSON, requests[0].Frame)
	assert.Equal(t, map[string]any{
		"audio":            "UklGRiYAAABXQVZFZm10IBAAAAABAAEAgD4AAAB9AAACABAAZGF0YQIAAAAAAQ==",
		"language":         "hi",
		"speech_enhance":   true,
		"max_tokens":       1024,
		"sample_rate_copy": int64(16000),
	}, requests[0].Body)
}

func TestPCM16MonoWAVSampleRate(t *testing.T) {
	for _, sampleRate := range []int{-1, 0, math.MaxUint32/2 + 1} {
		config := &Config{SampleRate: sampleRate}
		scope, err := config.newRequestScope("ctx-1", []byte{0, 1})
		require.ErrorContains(t, err, "sample rate")
		require.Nil(t, scope)
	}
	wavAudio, err := makePCM16MonoWAV([]byte{0, 1}, math.MaxUint32/2)
	require.NoError(t, err)
	assert.Equal(t, uint32(math.MaxUint32/2), binary.LittleEndian.Uint32(wavAudio[24:28]))
	assert.Equal(t, uint32(math.MaxUint32-1), binary.LittleEndian.Uint32(wavAudio[28:32]))
	assert.Equal(t, uint32(38), binary.LittleEndian.Uint32(wavAudio[4:8]))
	assert.Equal(t, uint32(2), binary.LittleEndian.Uint32(wavAudio[40:44]))
	assert.Equal(t, []byte{0, 1}, wavAudio[44:])
}

func TestDSLEngine_ParseHTTPResponse(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON},
				Emit: map[string]any{
					"script":     map[string]any{"$path": "text"},
					"confidence": map[string]any{"$cast": "number", "value": map[string]any{"$path": "confidence"}},
					"language":   map[string]any{"$path": "language"},
					"interim":    false,
				},
			},
		},
	}

	engine := config.newEngine()
	frame, err := engine.ParseHTTPResponse([]byte(`{"text":"namaste","confidence":"0.9","language":"hi"}`))
	require.NoError(t, err)

	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "namaste", outcome.Script)
	assert.Equal(t, 0.9, outcome.Confidence)
	assert.Equal(t, "hi", outcome.Language)
	assert.False(t, outcome.Interim)
}
