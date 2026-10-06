// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSLEngine_BuildConnectionURLAndEvaluateRequestRules(t *testing.T) {
	config := &Config{
		BaseURL:    "wss://example.com/stt?tenant=test",
		Model:      "model-a",
		Language:   "en-US",
		Encoding:   defaultEncoding,
		SampleRate: 16000,
		QueryParams: map[string]any{
			"model": map[string]any{"$var": "model"},
			"sample_rate": map[string]any{
				"$cast": "number",
				"value": map[string]any{"$var": "sample_rate"},
			},
		},
		RequestRules: []RequestRule{
			{
				When: RequestWhen{Packet: requestPacketTurnChange},
				Send: RequestSend{
					Frame: frameTypeJSON,
					Body: map[string]any{
						"type":     "start",
						"language": map[string]any{"$path": "config.language"},
					},
				},
			},
			{
				When: RequestWhen{Packet: requestPacketAudio},
				Send: RequestSend{
					Frame: frameTypeJSON,
					Body: map[string]any{
						"audio":    map[string]any{"$path": "packet.audio.wav_base64"},
						"encoding": map[string]any{"$path": "config.audio.encoding"},
					},
				},
			},
			{
				When: RequestWhen{Packet: requestPacketAudio},
				Send: RequestSend{
					Frame: frameTypeBinary,
					Body:  map[string]any{"$path": "packet.audio.bytes"},
				},
			},
		},
	}
	engine := config.newEngine()
	queryScope := config.newQueryScope()

	url, err := engine.BuildConnectionURL(queryScope)
	require.NoError(t, err)
	assert.Contains(t, url, "tenant=test")
	assert.Contains(t, url, "model=model-a")
	assert.Contains(t, url, "sample_rate=16000")

	scope, err := config.newRequestScope(requestPacketTurnChange, "ctx_1", nil)
	require.NoError(t, err)
	turnChangeRequests, err := engine.EvaluateRequestRules(requestPacketTurnChange, scope)
	require.NoError(t, err)
	require.Len(t, turnChangeRequests, 1)
	assert.Equal(t, frameTypeJSON, turnChangeRequests[0].Frame)
	assert.Equal(t, map[string]any{
		"type":     "start",
		"language": "en-US",
	}, turnChangeRequests[0].Body)

	scope, err = config.newRequestScope(requestPacketAudio, "ctx_1", []byte{0x00, 0x01})
	require.NoError(t, err)
	audioRequests, err := engine.EvaluateRequestRules(requestPacketAudio, scope)
	require.NoError(t, err)
	require.Len(t, audioRequests, 2)
	assert.Equal(t, frameTypeJSON, audioRequests[0].Frame)
	assert.Equal(t, "UklGRiYAAABXQVZFZm10IBAAAAABAAEAgD4AAAB9AAACABAAZGF0YQIAAAAAAQ==", audioRequests[0].Body.(map[string]any)["audio"])
	assert.Equal(t, "LINEAR16", audioRequests[0].Body.(map[string]any)["encoding"])
	assert.Equal(t, frameTypeBinary, audioRequests[1].Frame)
	assert.Equal(t, []byte{0x00, 0x01}, audioRequests[1].Body)
}

func TestPCM16MonoWAVSampleRate(t *testing.T) {
	for _, sampleRate := range []int{-1, 0, math.MaxUint32/2 + 1} {
		config := &Config{SampleRate: sampleRate}
		scope, err := config.newRequestScope(requestPacketAudio, "ctx-1", []byte{0, 1})
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

func TestDSLEngine_ParseAndEvaluateResponse(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "partial"},
				Emit: map[string]any{
					"script":     map[string]any{"$path": "text"},
					"confidence": map[string]any{"$cast": "number", "value": map[string]any{"$path": "confidence"}},
					"language":   map[string]any{"$path": "language"},
					"interim":    true,
				},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "final"},
				Emit: map[string]any{
					"script":  map[string]any{"$path": "text"},
					"interim": false,
				},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "error"},
				Emit: map[string]any{
					"error": map[string]any{"$path": "error.message"},
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`{"type":"partial","text":"hello","confidence":"0.7","language":"en-US"}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "hello", outcome.Script)
	assert.InDelta(t, 0.7, outcome.Confidence, 0.0001)
	assert.Equal(t, "en-US", outcome.Language)
	assert.True(t, outcome.Interim)

	frame, err = engine.ParseFrame(1, []byte(`{"type":"final","text":"hello world"}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "hello world", outcome.Script)
	assert.False(t, outcome.Interim)

	frame, err = engine.ParseFrame(1, []byte(`{"type":"error","error":{"message":"bad request"}}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "bad request", outcome.ErrorText)
}

func TestDSLEngine_ParseAndEvaluateTextResponse(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeText},
				Emit: map[string]any{
					"script":   map[string]any{"$frame": frameTypeText},
					"language": "hi",
					"interim":  false,
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte("namaste"))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "namaste", outcome.Script)
	assert.Equal(t, "hi", outcome.Language)
	assert.False(t, outcome.Interim)
}

func TestDSLEngine_ParseAndEvaluateQuotedJSONTextResponse(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeText},
				Emit: map[string]any{
					"script":   map[string]any{"$frame": frameTypeText},
					"language": "hi",
					"interim":  false,
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`" hello"`))
	require.NoError(t, err)
	assert.Equal(t, frameTypeText, frame.Kind)
	assert.Equal(t, " hello", frame.Text)

	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, " hello", outcome.Script)
	assert.Equal(t, "hi", outcome.Language)
	assert.False(t, outcome.Interim)
}

func TestDSLEngine_ParseAndEvaluateJSONObjectResponse(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON},
				Emit: map[string]any{
					"script":   map[string]any{"$path": "text"},
					"language": map[string]any{"$path": "language"},
					"interim":  map[string]any{"$path": "interim"},
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`{"text":"namaste duniya","language":"hi","interim":false}`))
	require.NoError(t, err)
	assert.Equal(t, frameTypeJSON, frame.Kind)

	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "namaste duniya", outcome.Script)
	assert.Equal(t, "hi", outcome.Language)
	assert.False(t, outcome.Interim)
}

func TestDSLEngine_InvalidVariable(t *testing.T) {
	config := &Config{
		BaseURL: "wss://example.com/stt",
		QueryParams: map[string]any{
			"bad": map[string]any{"$var": "unknown"},
		},
	}
	engine := config.newEngine()
	_, err := engine.BuildConnectionURL(config.newQueryScope())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown variable")
}

func TestDSLEngine_NoMatch(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "final"},
				Emit: map[string]any{"script": map[string]any{"$path": "text"}, "interim": false},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`{"type":"partial","text":"hello"}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.False(t, outcome.Matched)
}
