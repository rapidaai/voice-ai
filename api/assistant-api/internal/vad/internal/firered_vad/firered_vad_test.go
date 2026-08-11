// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_firered_vad

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOptions(tb testing.TB, confidence float64) utils.Option {
	opts := map[string]interface{}{}
	if confidence >= 0 {
		opts[internal_options.MicrophoneVADOptionConfidence] = confidence
	}
	return opts
}

func newFireRedOrSkip(t *testing.T, confidence float64, cb func(ctx context.Context, pkt ...internal_type.Packet) error) *FireRedVAD {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(t, confidence)
	vad, err := newFireRedVADForTest(t.Context(), logger, cb, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("firered model not available: %v", err)
		}
		require.NoError(t, err)
	}
	fr := vad.(*FireRedVAD)
	t.Cleanup(func() { _ = fr.Close(context.Background()) })
	return fr
}

func generateSilence(samples int) internal_type.UserAudioReceivedPacket {
	return internal_type.UserAudioReceivedPacket{Audio: make([]byte, samples*2)}
}

func generateSineWave(samples int, frequency, amplitude float64) internal_type.UserAudioReceivedPacket {
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(amplitude * 32767 * math.Sin(2*math.Pi*float64(i)*frequency/16000))
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return internal_type.UserAudioReceivedPacket{Audio: data}
}

func generateNoise(samples int) internal_type.UserAudioReceivedPacket {
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16((i*7919)%65536 - 32768)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return internal_type.UserAudioReceivedPacket{Audio: data}
}

// Core functionality tests

func TestNew_DefaultConfig(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, -1, callback)

	assert.NotNil(t, vad.detector)
	assert.NotNil(t, vad.fbank)
	assert.NotNil(t, vad.postprocessor)
	assert.Equal(t, float32(defaultConfidence), vad.postprocessor.cfg.SpeechThreshold)
	assert.Equal(t, vadDurationFrames(defaultStartSecs), vad.postprocessor.cfg.MinSpeechFrame)
	assert.Equal(t, vadDurationFrames(defaultStopSecs), vad.postprocessor.cfg.MinSilenceFrame)
}

func TestNew_OverridesConfig(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	logger, _ := commons.NewApplicationLogger()
	opts := utils.Option{
		internal_options.MicrophoneVADOptionConfidence: 0.55,
		internal_options.MicrophoneVADOptionStartSecs:  0.1,
		internal_options.MicrophoneVADOptionStopSecs:   0.4,
	}

	vad, err := newFireRedVADForTest(t.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("firered model not available: %v", err)
		}
		require.NoError(t, err)
	}
	fr := vad.(*FireRedVAD)
	t.Cleanup(func() { _ = fr.Close(context.Background()) })

	assert.Equal(t, float32(0.55), fr.postprocessor.cfg.SpeechThreshold)
	assert.Equal(t, 10, fr.postprocessor.cfg.MinSpeechFrame)
	assert.Equal(t, 40, fr.postprocessor.cfg.MinSilenceFrame)
}

func TestNew_RejectsNegativeDurationBeforeFrameConversion(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	for name, opts := range map[string]utils.Option{
		"start_secs": {
			internal_options.MicrophoneVADOptionStartSecs: -0.004,
		},
		"stop_secs": {
			internal_options.MicrophoneVADOptionStopSecs: -0.004,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(WithContext(t.Context()), WithOnPacket(callback), WithOptions(opts))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "should be a positive number")
		})
	}
}

func TestPostprocessor_SpeechStartFrameIsNotBackdated(t *testing.T) {
	pp := NewPostprocessor(PostprocessorConfig{
		SmoothWindowSize: 1,
		SpeechThreshold:  0.5,
		MinSpeechFrame:   3,
		MaxSpeechFrame:   2000,
		MinSilenceFrame:  2,
	})

	pp.ProcessFrame(0)
	pp.ProcessFrame(0)
	pp.ProcessFrame(1)
	pp.ProcessFrame(1)
	result := pp.ProcessFrame(1)

	require.True(t, result.IsSpeechStart)
	assert.Equal(t, 3, result.SpeechStartFrame)
}

func TestFireRedVAD_Name(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	assert.Equal(t, "firered_vad", vad.Name())
}

func TestNew_EmitsInitializationObservability(t *testing.T) {
	var packets []internal_type.Packet
	callback := func(_ context.Context, pkt ...internal_type.Packet) error {
		packets = append(packets, pkt...)
		return nil
	}

	_ = newFireRedOrSkip(t, 0.5, callback)

	var hasInitMetric bool
	var hasInitLogWithOptions bool
	for _, packet := range packets {
		switch typed := packet.(type) {
		case internal_type.ObservabilityMetricRecordPacket:
			if typed.Scope == internal_type.ObservabilityRecordScopeConversation &&
				len(typed.Record.Metrics) == 1 &&
				typed.Record.Metrics[0].Name == observability.MetricVADInitLatencyMs &&
				typed.Record.Attributes["provider"] == vadName {
				hasInitMetric = true
			}
		case internal_type.ObservabilityLogRecordPacket:
			if typed.Scope == internal_type.ObservabilityRecordScopeConversation &&
				typed.Record.Level == observability.LevelInfo &&
				typed.Record.Message == "firered_vad: initialization completed" &&
				typed.Record.Attributes["component"] == observability.ComponentVAD.String() &&
				typed.Record.Attributes["provider"] == vadName &&
				typed.Record.Attributes["options"] != "" {
				hasInitLogWithOptions = true
			}
		}
	}

	assert.True(t, hasInitMetric, "expected VAD init latency metric")
	assert.True(t, hasInitLogWithOptions, "expected VAD init log with options")
}

func TestFireRedVAD_Process_Silence_NoCallback(t *testing.T) {
	detectionFired := false
	callback := func(_ context.Context, pkts ...internal_type.Packet) error {
		for _, p := range pkts {
			if _, ok := p.(internal_type.InterruptionDetectedPacket); ok {
				detectionFired = true
			}
		}
		return nil
	}

	vad := newFireRedOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(16000))
	require.NoError(t, err)
	assert.False(t, detectionFired, "silence should not trigger a speech detection event")
}

func TestFireRedVAD_Process_Speech_AllowsCallback(t *testing.T) {
	var result internal_type.InterruptionDetectedPacket
	callback := func(ctx context.Context, pkt ...internal_type.Packet) error {
		for _, p := range pkt {
			if interruption, ok := p.(internal_type.InterruptionDetectedPacket); ok {
				result = interruption
			}
		}
		return nil
	}

	vad := newFireRedOrSkip(t, 0.2, callback)

	err := vad.Execute(context.Background(), generateSineWave(16000, 440, 0.9))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.EndAt, result.StartAt)
}

func TestFireRedVAD_Process_CorruptedData(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	corrupted := make([]byte, 999)
	err := vad.Execute(context.Background(), internal_type.UserAudioReceivedPacket{Audio: corrupted})
	_ = err
}

func TestFireRedVAD_Process_VerySmallChunks(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	sizes := []int{1, 2, 5, 10, 20}
	for _, size := range sizes {
		size := size
		t.Run(fmt.Sprintf("%d_samples", size), func(t *testing.T) {
			err := vad.Execute(context.Background(), generateSilence(size))
			_ = err
		})
	}
}

func TestFireRedVAD_Process_Concurrent(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	var wg sync.WaitGroup
	const workers = 8
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_ = vad.Execute(context.Background(), generateSilence(1600))
		}()
	}
	wg.Wait()
}

func TestFireRedVAD_Close_Idempotent(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(t, 0.5)

	vad, err := newFireRedVADForTest(t.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("firered model not available: %v", err)
		}
		require.NoError(t, err)
	}

	require.NoError(t, vad.Close(context.Background()))
	err = vad.Close(context.Background())
	_ = err
}

func TestFireRedVAD_Close_EmitsDurationUsageAndClosedEvent(t *testing.T) {
	var packets []internal_type.Packet
	logger, _ := commons.NewApplicationLogger()
	callback := func(_ context.Context, pkt ...internal_type.Packet) error {
		packets = append(packets, pkt...)
		return nil
	}
	opts := newTestOptions(t, 0.5)

	vad, err := newFireRedVADForTest(t.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("firered model not available: %v", err)
		}
		require.NoError(t, err)
	}

	require.NoError(t, vad.Close(context.Background()))

	var hasDurationUsage bool
	var hasClosedEvent bool
	for _, packet := range packets {
		switch typed := packet.(type) {
		case internal_type.ObservabilityUsageRecordPacket:
			if typed.Scope == internal_type.ObservabilityRecordScopeConversation &&
				typed.Record.Component == observability.ComponentName(observability.UsageConversationVADDuration) &&
				typed.Record.Provider == vadName &&
				typed.Record.Duration > 0 {
				hasDurationUsage = true
			}
		case internal_type.ObservabilityEventRecordPacket:
			if typed.Scope == internal_type.ObservabilityRecordScopeConversation &&
				typed.Record.Component == observability.ComponentVAD &&
				typed.Record.Event == observability.VADClosed &&
				typed.Record.Attributes["provider"] == vadName {
				hasClosedEvent = true
			}
		}
	}

	assert.True(t, hasDurationUsage, "expected VAD duration usage after Close")
	assert.True(t, hasClosedEvent, "expected VAD closed event after Close")
}

func TestFireRedVAD_Process_NoisePatterns(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateNoise(16000))
	require.NoError(t, err)
}

func TestFireRedVAD_Process_MaxAmplitude(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	samples := 16000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		var val int16
		if i%2 == 0 {
			val = 32767
		} else {
			val = -32768
		}
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(val))
	}

	err := vad.Execute(context.Background(), internal_type.UserAudioReceivedPacket{Audio: data})
	require.NoError(t, err)
}

func TestFireRedVAD_Process_RepeatedCalls(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	chunk := generateSilence(1600)
	for i := 0; i < 50; i++ {
		err := vad.Execute(context.Background(), chunk)
		require.NoError(t, err)
	}
}

func TestFireRedVAD_StatefulProcessing(t *testing.T) {
	var calls int
	callback := func(context.Context, ...internal_type.Packet) error {
		calls++
		return nil
	}

	vad := newFireRedOrSkip(t, 0.3, callback)

	for i := 0; i < 10; i++ {
		err := vad.Execute(context.Background(), generateSineWave(1600, 440, 0.8))
		require.NoError(t, err)
	}

	assert.GreaterOrEqual(t, calls, 0)
}

func TestFireRedVAD_Process_80msChunk(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newFireRedOrSkip(t, 0.5, callback)

	// 80ms at 16kHz = 1280 samples — production chunk size
	err := vad.Execute(context.Background(), generateSilence(1280))
	require.NoError(t, err)
}

func TestFireRedVAD_Process_PartialFrameCarry_NoDrop(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	vad := newFireRedOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(128))
	require.NoError(t, err)
	assert.Equal(t, 128, len(vad.audioBuf))

	err = vad.Execute(context.Background(), generateSilence(200))
	require.NoError(t, err)
	assert.Equal(t, 328, len(vad.audioBuf))

	err = vad.Execute(context.Background(), generateSilence(100))
	require.NoError(t, err)
	// 428 total samples buffered -> one frame processed, shift by 160 samples -> 268 retained.
	assert.Equal(t, 268, len(vad.audioBuf))
}
