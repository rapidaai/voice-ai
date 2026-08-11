// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silero_vad

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
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

func newTestOptions(tb testing.TB, threshold float64) utils.Option {
	opts := map[string]interface{}{}
	if threshold >= 0 {
		opts[internal_options.MicrophoneVADOptionConfidence] = threshold
	}
	return opts
}

func getModelPath() string {
	envModelPath := os.Getenv("SILERO_MODEL_PATH")
	if envModelPath == "" {
		_, path, _, _ := runtime.Caller(0)
		envModelPath = filepath.Join(filepath.Dir(path), "models/silero_vad_20251001.onnx")
	}
	return envModelPath
}

func newSileroOrSkip(t *testing.T, threshold float64, cb func(ctx context.Context, pkt ...internal_type.Packet) error) *SileroVAD {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(t, threshold)
	vad, err := newSileroVADForTest(t.Context(), logger, cb, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("silero model missing at %s", getModelPath())
		}
		require.NoError(t, err)
	}
	silero := vad.(*SileroVAD)
	t.Cleanup(func() { _ = silero.Close(context.Background()) })
	return silero
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

func TestNew_DefaultThreshold(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, -1, callback)

	assert.NotNil(t, vad.detector)
}

func TestSileroVAD_Name(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

	assert.Equal(t, "silero_vad", vad.Name())
}

func TestNew_EmitsInitializationObservability(t *testing.T) {
	var packets []internal_type.Packet
	callback := func(_ context.Context, pkt ...internal_type.Packet) error {
		packets = append(packets, pkt...)
		return nil
	}

	_ = newSileroOrSkip(t, 0.5, callback)

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
				typed.Record.Message == "silero_vad: initialization completed" &&
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

func TestSileroVAD_Process_Silence_NoCallback(t *testing.T) {
	detectionFired := false
	callback := func(_ context.Context, pkts ...internal_type.Packet) error {
		for _, p := range pkts {
			if _, ok := p.(internal_type.InterruptionDetectedPacket); ok {
				detectionFired = true
			}
		}
		return nil
	}

	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(16000))
	require.NoError(t, err)
	assert.False(t, detectionFired, "silence should not trigger a speech detection event")
}

func TestSileroVAD_Process_Speech_AllowsCallback(t *testing.T) {
	var result internal_type.InterruptionDetectedPacket
	callback := func(ctx context.Context, pkt ...internal_type.Packet) error {
		if len(pkt) > 0 {
			if interruption, ok := pkt[0].(internal_type.InterruptionDetectedPacket); ok {
				result = interruption
			}
		}
		return nil
	}

	vad := newSileroOrSkip(t, 0.2, callback)

	err := vad.Execute(context.Background(), generateSineWave(16000, 440, 0.9))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.EndAt, result.StartAt)
}

func TestSileroVAD_Process_CorruptedData(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

	corrupted := make([]byte, 999) // Odd length
	err := vad.Execute(context.Background(), internal_type.UserAudioReceivedPacket{Audio: corrupted})
	_ = err // Accept error or nil; should not panic
}

func TestSileroVAD_Process_VerySmallChunks(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

	sizes := []int{1, 2, 5, 10, 20}
	for _, size := range sizes {
		size := size
		t.Run(fmt.Sprintf("%d_samples", size), func(t *testing.T) {
			err := vad.Execute(context.Background(), generateSilence(size))
			_ = err
		})
	}
}

func TestSileroVAD_Process_Concurrent(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

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

func TestSileroVAD_Close_Idempotent(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(t, 0.5)

	vad, err := newSileroVADForTest(t.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("silero model missing at %s", getModelPath())
		}
		require.NoError(t, err)
	}

	require.NoError(t, vad.Close(context.Background()))
	err = vad.Close(context.Background())
	_ = err
}

func TestSileroVAD_Close_EmitsDurationUsageAndClosedEvent(t *testing.T) {
	var packets []internal_type.Packet
	logger, _ := commons.NewApplicationLogger()
	callback := func(_ context.Context, pkt ...internal_type.Packet) error {
		packets = append(packets, pkt...)
		return nil
	}
	opts := newTestOptions(t, 0.5)

	vad, err := newSileroVADForTest(t.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("silero model missing at %s", getModelPath())
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

func TestSileroVAD_ModelPath_Environment(t *testing.T) {
	modelPath := getModelPath()
	if _, err := os.Stat(modelPath); err != nil {
		t.Skipf("silero model missing at %s", modelPath)
	}

	original := os.Getenv("SILERO_MODEL_PATH")
	os.Setenv("SILERO_MODEL_PATH", modelPath)
	t.Cleanup(func() {
		if original != "" {
			os.Setenv("SILERO_MODEL_PATH", original)
		} else {
			os.Unsetenv("SILERO_MODEL_PATH")
		}
	})

	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	_ = newSileroOrSkip(t, 0.5, callback)
}

func TestSileroVAD_Process_NoisePatterns(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateNoise(16000))
	require.NoError(t, err)
}

func TestSileroVAD_Process_MaxAmplitude(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

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

func TestSileroVAD_Process_RepeatedCalls(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	vad := newSileroOrSkip(t, 0.5, callback)

	chunk := generateSilence(1600)
	for i := 0; i < 50; i++ {
		err := vad.Execute(context.Background(), chunk)
		require.NoError(t, err)
	}
}

func TestSileroVAD_StatefulProcessing(t *testing.T) {
	var calls int
	callback := func(context.Context, ...internal_type.Packet) error {
		calls++
		return nil
	}

	vad := newSileroOrSkip(t, 0.3, callback)

	for i := 0; i < 10; i++ {
		err := vad.Execute(context.Background(), generateSineWave(1600, 440, 0.8))
		require.NoError(t, err)
	}

	assert.GreaterOrEqual(t, calls, 0)
}

func TestSileroVAD_Process_ExactWindow_512Samples(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(512))
	require.NoError(t, err)
	assert.Equal(t, 512, vad.detector.currSample, "exact 512-sample chunk should process one full window")
}

func TestSileroVAD_Process_ExactMultiple_1024Samples(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(1024))
	require.NoError(t, err)
	assert.Equal(t, 1024, vad.detector.currSample, "exact 1024-sample chunk should process two full windows")
}

func TestSileroVAD_Process_CrossCallWindowCarry(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(320))
	require.NoError(t, err)
	assert.Equal(t, 0, vad.detector.currSample, "320 samples alone should not process a full 512-sample window")

	err = vad.Execute(context.Background(), generateSilence(192))
	require.NoError(t, err)
	assert.Equal(t, 512, vad.detector.currSample, "320+192 samples across calls should process one full window")
}

func TestSileroVAD_Process_RemainderCarry(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	vad := newSileroOrSkip(t, 0.5, callback)

	err := vad.Execute(context.Background(), generateSilence(1600))
	require.NoError(t, err)
	assert.Equal(t, 1536, vad.detector.currSample, "1600 samples should process three windows and carry 64")

	err = vad.Execute(context.Background(), generateSilence(448))
	require.NoError(t, err)
	assert.Equal(t, 2048, vad.detector.currSample, "64 carry + 448 samples should process one more window")
}
