// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

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
	if _, err := os.Stat(resolvePctModelPath("")); err != nil {
		t.Skipf("pipecat model asset unavailable: %v", err)
	}

	for _, testCase := range []struct {
		name            string
		options         utils.Option
		threshold       float64
		extendedTimeout time.Duration
		fallbackTimeout time.Duration
	}{
		{name: "defaults", threshold: 0.85, extendedTimeout: 4 * time.Second, fallbackTimeout: time.Second},
		{name: "nil values", options: utils.Option{optPctThreshold: nil, optPctExtendedTimeout: nil, optPctFallbackTimeout: nil},
			threshold: 0.85, extendedTimeout: 4 * time.Second, fallbackTimeout: time.Second},
		{name: "numeric strings", options: utils.Option{optPctThreshold: "0.6", optPctExtendedTimeout: "2500", optPctFallbackTimeout: json.Number("1500")},
			threshold: 0.6, extendedTimeout: 2500 * time.Millisecond, fallbackTimeout: 1500 * time.Millisecond},
		{name: "unsigned timeouts", options: utils.Option{optPctThreshold: 1.0, optPctExtendedTimeout: uint(2000), optPctFallbackTimeout: uint64(500)},
			threshold: 1, extendedTimeout: 2 * time.Second, fallbackTimeout: 500 * time.Millisecond},
		{name: "zero waits", options: utils.Option{optPctThreshold: 0, optPctExtendedTimeout: 0, optPctFallbackTimeout: 0}},
		{name: "duration bound", options: utils.Option{optPctExtendedTimeout: math.MaxInt64 / int64(time.Millisecond)},
			threshold: 0.85, extendedTimeout: time.Duration(math.MaxInt64/int64(time.Millisecond)) * time.Millisecond, fallbackTimeout: time.Second},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			executor, err := New(nil, WithContext(nil), WithOptions(testCase.options),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
			endOfSpeech := executor.(*pipecatEndOfSpeech)
			assert.Equal(t, testCase.threshold, endOfSpeech.threshold)
			assert.Equal(t, testCase.extendedTimeout, endOfSpeech.extendedTimeout)
			assert.Equal(t, testCase.fallbackTimeout, endOfSpeech.fallbackTimeout)
			assert.Equal(t, 5*time.Second, endOfSpeech.turnStopTimeout)
		})
	}
}

func TestNew_InvalidParametersBeforeModelLoad(t *testing.T) {
	for _, optionKey := range []string{optPctThreshold, optPctExtendedTimeout, optPctFallbackTimeout} {
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
			{name: "unsupported type", value: struct{}{}},
			{name: "duration overflow", value: float64(math.MaxInt64/int64(time.Millisecond)) + 1},
		} {
			t.Run(optionKey+"/"+testCase.name, func(t *testing.T) {
				var callbacks atomic.Int32
				executor, err := New(WithOptions(utils.Option{
					optionKey: testCase.value, optPctModelPath: "/nonexistent/model.onnx",
				}), WithOnPacket(func(context.Context, ...internal_type.Packet) error {
					callbacks.Add(1)
					return nil
				}))
				require.ErrorIs(t, err, errPipecatInvalidOption)
				assert.NotErrorIs(t, err, errPipecatInitDetector)
				assert.ErrorContains(t, err, optionKey)
				assert.Nil(t, executor)
				assert.Zero(t, callbacks.Load())
				if optionKey == optPctThreshold && testCase.name == "invalid string" {
					var parseError *strconv.NumError
					assert.ErrorAs(t, err, &parseError)
				}
			})
		}
	}
	t.Run("threshold above one", func(t *testing.T) {
		_, err := New(WithOptions(utils.Option{optPctThreshold: 1.01}),
			WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }))
		require.ErrorIs(t, err, errPipecatInvalidOption)
	})
	for _, optionKey := range []string{optPctExtendedTimeout, optPctFallbackTimeout} {
		for _, value := range []any{0.5, uint64(math.MaxUint64)} {
			t.Run(optionKey+"/invalid timeout", func(t *testing.T) {
				_, err := New(WithOptions(utils.Option{optionKey: value}),
					WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }))
				require.ErrorIs(t, err, errPipecatInvalidOption)
				assert.ErrorContains(t, err, optionKey)
			})
		}
	}
}

func TestNew_ConstructionErrorsAreReturnedOnly(t *testing.T) {
	executor, err := New()
	require.ErrorIs(t, err, errPipecatOnPacketRequired)
	assert.Nil(t, executor)
	var callbacks atomic.Int32
	executor, err = New(WithOptions(utils.Option{optPctModelPath: "/nonexistent/model.onnx"}),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error {
			callbacks.Add(1)
			return nil
		}))
	require.ErrorIs(t, err, errPipecatInitDetector)
	assert.Nil(t, executor)
	assert.Zero(t, callbacks.Load())
}
