// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ProviderParameters(t *testing.T) {
	for _, path := range []string{resolveModelPath("", false), resolveTokenizerPath("")} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("turn detector asset unavailable: %v", err)
		}
	}
	for _, testCase := range []struct {
		name            string
		options         utils.Option
		threshold       float64
		quickTimeout    time.Duration
		extendedTimeout time.Duration
		maxHistory      int
	}{
		{name: "defaults", threshold: 0.0289, quickTimeout: 250 * time.Millisecond, extendedTimeout: 3 * time.Second, maxHistory: 6},
		{name: "nil values", options: utils.Option{optKeyThreshold: nil, optKeyQuickTimeout: nil, optKeyExtendedTimeout: nil, optKeyMaxHistory: nil},
			threshold: 0.0289, quickTimeout: 250 * time.Millisecond, extendedTimeout: 3 * time.Second, maxHistory: 6},
		{name: "numeric strings", options: utils.Option{optKeyThreshold: "0.5", optKeyQuickTimeout: "100", optKeyExtendedTimeout: json.Number("1500"), optKeyMaxHistory: "3"},
			threshold: 0.5, quickTimeout: 100 * time.Millisecond, extendedTimeout: 1500 * time.Millisecond, maxHistory: 3},
		{name: "unsigned counts", options: utils.Option{optKeyThreshold: 1.0, optKeyQuickTimeout: uint(200), optKeyExtendedTimeout: uint64(500), optKeyMaxHistory: uint32(4)},
			threshold: 1, quickTimeout: 200 * time.Millisecond, extendedTimeout: 500 * time.Millisecond, maxHistory: 4},
		{name: "zero waits and unsliced history", options: utils.Option{optKeyThreshold: 0, optKeyQuickTimeout: 0, optKeyExtendedTimeout: 0, optKeyMaxHistory: 0}},
		{name: "duration bound", options: utils.Option{optKeyExtendedTimeout: math.MaxInt64 / int64(time.Millisecond)},
			threshold: 0.0289, quickTimeout: 250 * time.Millisecond, extendedTimeout: time.Duration(math.MaxInt64/int64(time.Millisecond)) * time.Millisecond, maxHistory: 6},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			executor, err := New(nil, WithContext(nil), WithOptions(testCase.options),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
			endOfSpeech := executor.(*livekitEndOfSpeech)
			assert.Equal(t, testCase.threshold, endOfSpeech.threshold)
			assert.Equal(t, testCase.quickTimeout, endOfSpeech.quickTimeout)
			assert.Equal(t, testCase.extendedTimeout, endOfSpeech.silenceTimeout)
			assert.Equal(t, testCase.maxHistory, endOfSpeech.maxHistory)
			assert.Equal(t, defaultModelType, endOfSpeech.modelType)
		})
	}
}

func TestNew_InvalidParametersBeforeModelLoad(t *testing.T) {
	for _, optionKey := range []string{optKeyThreshold, optKeyQuickTimeout, optKeyExtendedTimeout, optKeyMaxHistory} {
		for _, testCase := range []struct {
			name  string
			value any
		}{
			{name: "negative", value: -1},
			{name: "not a number", value: math.NaN()},
			{name: "positive infinity", value: math.Inf(1)},
			{name: "negative infinity", value: math.Inf(-1)},
			{name: "invalid string", value: "invalid"},
			{name: "empty string", value: ""},
			{name: "boolean", value: true},
		} {
			t.Run(optionKey+"/"+testCase.name, func(t *testing.T) {
				var callbacks atomic.Int32
				executor, err := New(WithOptions(utils.Option{
					optionKey: testCase.value, optKeyModelPath: "/nonexistent/model.onnx",
				}), WithOnPacket(func(context.Context, ...internal_type.Packet) error {
					callbacks.Add(1)
					return nil
				}))
				require.ErrorIs(t, err, errLivekitInvalidOption)
				assert.NotErrorIs(t, err, errLivekitInitTurnDetector)
				assert.ErrorContains(t, err, optionKey)
				assert.Nil(t, executor)
				assert.Zero(t, callbacks.Load())
				if optionKey == optKeyThreshold && testCase.name == "invalid string" {
					var parseError *strconv.NumError
					assert.ErrorAs(t, err, &parseError)
				}
			})
		}
	}
	for _, testCase := range []struct {
		name      string
		optionKey string
		value     any
	}{
		{name: "threshold above one", optionKey: optKeyThreshold, value: 1.01},
		{name: "fractional quick delay", optionKey: optKeyQuickTimeout, value: 0.5},
		{name: "fractional extended delay", optionKey: optKeyExtendedTimeout, value: "0.5"},
		{name: "fractional history", optionKey: optKeyMaxHistory, value: 1.5},
		{name: "quick delay overflow", optionKey: optKeyQuickTimeout, value: uint64(math.MaxInt64/int64(time.Millisecond)) + 1},
		{name: "extended delay overflow", optionKey: optKeyExtendedTimeout, value: uint64(math.MaxUint64)},
		{name: "history overflow", optionKey: optKeyMaxHistory, value: uint64(math.MaxInt) + 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := New(WithOptions(utils.Option{testCase.optionKey: testCase.value}),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }))
			require.ErrorIs(t, err, errLivekitInvalidOption)
			assert.ErrorContains(t, err, testCase.optionKey)
		})
	}
}

func TestNew_ConstructionErrorsAreReturnedOnly(t *testing.T) {
	executor, err := New()
	require.ErrorIs(t, err, errLivekitOnPacketRequired)
	assert.Nil(t, executor)
	var callbacks atomic.Int32
	executor, err = New(WithOptions(utils.Option{optKeyTokenizerPath: "/nonexistent/tokenizer.json"}),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error {
			callbacks.Add(1)
			return nil
		}))
	require.ErrorIs(t, err, errLivekitInitTurnDetector)
	assert.Nil(t, executor)
	assert.Zero(t, callbacks.Load())
}
