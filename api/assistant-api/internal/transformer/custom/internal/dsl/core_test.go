// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_dsl

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreParseFrame_SupportsBinaryJSONAndText(t *testing.T) {
	core := NewCore("test")

	frame, err := core.ParseFrame(2, []byte{0x01, 0x02}, func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameBinary, frame.Kind)
	assert.Equal(t, []byte{0x01, 0x02}, frame.Binary)

	frame, err = core.ParseFrame(1, []byte(`{"type":"partial","text":"hello"}`), func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameJSON, frame.Kind)
	assert.Equal(t, `{"type":"partial","text":"hello"}`, frame.Text)

	frame, err = core.ParseFrame(1, []byte("namaste"), func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameText, frame.Kind)
	assert.Equal(t, "namaste", frame.Text)
}

func TestCoreMatchWhen_TextFrames(t *testing.T) {
	core := NewCore("test")
	frame := Frame{Kind: FrameText, Text: "namaste"}

	matched, err := core.MatchWhen(When{Frame: FrameText}, frame)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = core.MatchWhen(When{Frame: FrameText, Equals: "namaste"}, frame)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = core.MatchWhen(When{Frame: FrameText, Equals: "hello"}, frame)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestCoreValidateResponseRules_TextContracts(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedResponseFrames: []string{FrameJSON, FrameText},
		SupportedEmitKeys:       []string{"script", "interim", "error"},
		AllowedFrameSelectors:   []string{FrameText},
	}

	err := core.ValidateResponseRules([]ResponseRule{
		{
			When: When{Frame: FrameText},
			Emit: map[string]any{
				"script":  map[string]any{"$frame": FrameText},
				"interim": true,
			},
		},
	}, contract, "listen.response_rules")
	require.NoError(t, err)

	err = core.ValidateResponseRules([]ResponseRule{
		{
			When: When{Frame: FrameText, Path: "type"},
			Emit: map[string]any{
				"script": map[string]any{"$frame": FrameText},
			},
		},
	}, contract, "listen.response_rules")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "when.path cannot be used")
}

func TestCoreEvalResponseExpr_SupportsTextFrameSelectors(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	frame := Frame{Kind: FrameText, Text: "bonjour"}
	contract := Contract{
		AllowedFrameSelectors: []string{FrameText},
	}

	value, err := core.EvalResponseExpr(map[string]any{
		"script": map[string]any{"$frame": FrameText},
	}, frame, contract)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"script": "bonjour"}, value)
}

func TestCoreValidateRequestObject_RejectsUnknownVariable(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedVariables: []string{"audio", "encoding"},
	}

	err := core.ValidateRequestObject(map[string]any{
		"audio": map[string]any{"$var": "chunk"},
	}, contract, "speak.query_params")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported variable "chunk"`)
}

func TestCoreValidateRequestRules_SupportsPacketScopedPaths(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedRequestPackets: []string{"turn_change", "audio", "interrupt"},
		SupportedRequestFrames:  []string{FrameBinary, FrameJSON, FrameText},
		SupportedPathRoots:      []string{"config", "packet"},
		RequestValidationScopes: map[string]any{
			"audio": map[string]any{
				"config": map[string]any{
					"audio": map[string]any{
						"encoding":    "LINEAR16",
						"sample_rate": 16000,
					},
				},
				"packet": map[string]any{
					"kind":       "audio",
					"context_id": "ctx_123",
					"audio": map[string]any{
						"bytes":  []byte{0x00, 0x01},
						"base64": "AAE=",
					},
				},
			},
		},
	}

	err := core.ValidateRequestRules([]RequestRule{
		{
			When: RequestWhen{Packet: "audio"},
			Send: Send{
				Frame: FrameBinary,
				Body:  map[string]any{"$path": "packet.audio.bytes"},
			},
		},
		{
			When: RequestWhen{Packet: "audio"},
			Send: Send{
				Frame: FrameJSON,
				Body: map[string]any{
					"audio":    map[string]any{"$path": "packet.audio.base64"},
					"encoding": map[string]any{"$path": "config.audio.encoding"},
				},
			},
		},
	}, contract, "listen.request_rules")
	require.NoError(t, err)
}

func TestCoreValidateRequestRules_RejectsUnknownPathRoot(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedRequestPackets: []string{"audio"},
		SupportedRequestFrames:  []string{FrameBinary},
		SupportedPathRoots:      []string{"config", "packet"},
	}

	err := core.ValidateRequestRules([]RequestRule{
		{
			When: RequestWhen{Packet: "audio"},
			Send: Send{
				Frame: FrameBinary,
				Body:  map[string]any{"$path": "state.audio.bytes"},
			},
		},
	}, contract, "listen.request_rules")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `$path root must be "config" or "packet"`)
}

func TestCoreValidateQueryParams_RejectsNestedValues(t *testing.T) {
	core := NewCore("custom websocket_v1")
	contract := Contract{
		SupportedVariables: []string{"language", "sample_rate"},
	}

	err := core.ValidateQueryParams(map[string]any{
		"language": map[string]any{"$var": "language"},
		"rate":     map[string]any{"$cast": "number", "value": map[string]any{"$var": "sample_rate"}},
	}, contract, "ws.query_params")
	require.NoError(t, err)

	err = core.ValidateQueryParams(map[string]any{
		"metadata": map[string]any{"lang": "hi"},
	}, contract, "ws.query_params")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must resolve to primitive value")

	err = core.ValidateQueryParams(map[string]any{
		"rates": []any{16000},
	}, contract, "ws.query_params")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must resolve to primitive value")
}

func TestCoreBuildConnectionURL_FormatsPrefixedParseErrors(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")

	_, err := core.BuildConnectionURL("://invalid", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom-stt websocket_v1: invalid base url")
	assert.NotContains(t, err.Error(), "%!w")
}

func TestCoreToNumber_RejectsUnsignedOverflow(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")

	_, err := core.ToNumber(uint64(math.MaxUint64))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum signed integer")
}

func TestCoreValuesEqual_PreservesLargeUnsignedValues(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")

	assert.True(t, core.ValuesEqual(uint64(math.MaxUint64), uint64(math.MaxUint64)))
	assert.False(t, core.ValuesEqual(uint64(math.MaxUint64), int64(-1)))
}
