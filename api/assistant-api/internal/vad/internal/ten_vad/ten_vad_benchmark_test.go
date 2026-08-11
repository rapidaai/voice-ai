// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_ten_vad

import (
	"context"
	"encoding/binary"
	"math"
	"sync"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// Benchmark helpers

func newBenchmarkTenVAD(b *testing.B, confidence float64) *TenVAD {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(b, confidence)

	vad, err := newTenVADForTest(b.Context(), logger, callback, opts)
	if err != nil {
		b.Skipf("ten_vad library not available: %v", err)
	}
	b.Cleanup(func() { vad.Close(context.Background()) })
	return vad.(*TenVAD)
}

func generateBenchmarkSilence(samples int) internal_type.UserAudioReceivedPacket {
	return internal_type.UserAudioReceivedPacket{Audio: make([]byte, samples*2)}
}

func generateBenchmarkSineWave(samples int, frequency, amplitude float64) internal_type.UserAudioReceivedPacket {
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(amplitude * 32767 * math.Sin(2*math.Pi*float64(i)*frequency/16000))
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return internal_type.UserAudioReceivedPacket{Audio: data}
}

// Single operation benchmarks

func BenchmarkTenVAD_Process_Silence_80ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(1280) // 80ms at 16kHz — production chunk

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Silence_100ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(1600) // 100ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Silence_500ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(8000) // 500ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Silence_1s(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(16000) // 1s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Speech_80ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSineWave(1280, 440, 0.8) // 80ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Speech_100ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSineWave(1600, 440, 0.8) // 100ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Speech_500ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSineWave(8000, 440, 0.8) // 500ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Speech_1s(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSineWave(16000, 440, 0.8) // 1s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

// Different chunk sizes

func BenchmarkTenVAD_Process_ChunkSize_16ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(256) // 16ms at 16kHz — single TEN VAD frame

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_ChunkSize_50ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(800) // 50ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_ChunkSize_200ms(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(3200) // 200ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_ChunkSize_2s(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(32000) // 2s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

// Different confidence values

func BenchmarkTenVAD_Process_Confidence_0_1(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.1)
	data := generateBenchmarkSineWave(8000, 440, 0.8)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Confidence_0_5(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSineWave(8000, 440, 0.8)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkTenVAD_Process_Confidence_0_9(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.9)
	data := generateBenchmarkSineWave(8000, 440, 0.8)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

// Parallel processing benchmarks

func BenchmarkTenVAD_Process_Parallel_2Streams(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(b, 0.5)

	vads := make([]*TenVAD, 2)
	for i := 0; i < 2; i++ {
		callback := func(context.Context, ...internal_type.Packet) error { return nil }
		vad, err := newTenVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			b.Skipf("ten_vad library not available: %v", err)
		}
		vads[i] = vad.(*TenVAD)
		b.Cleanup(func() { vad.Close(context.Background()) })
	}

	data := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, vad := range vads {
			wg.Add(1)
			go func(v *TenVAD) {
				defer wg.Done()
				_ = v.Execute(context.Background(), data)
			}(vad)
		}
		wg.Wait()
	}
}

func BenchmarkTenVAD_Process_Parallel_8Streams(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(b, 0.5)

	vads := make([]*TenVAD, 8)
	for i := 0; i < 8; i++ {
		callback := func(context.Context, ...internal_type.Packet) error { return nil }
		vad, err := newTenVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			b.Skipf("ten_vad library not available: %v", err)
		}
		vads[i] = vad.(*TenVAD)
		b.Cleanup(func() { vad.Close(context.Background()) })
	}

	data := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, vad := range vads {
			wg.Add(1)
			go func(v *TenVAD) {
				defer wg.Done()
				_ = v.Execute(context.Background(), data)
			}(vad)
		}
		wg.Wait()
	}
}

// Sequential stream processing

func BenchmarkTenVAD_Process_SequentialStream_10Chunks(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(1280) // 80ms production chunks

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = vad.Execute(context.Background(), data)
		}
	}
}

func BenchmarkTenVAD_Process_SequentialStream_50Chunks(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(1280)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 50; j++ {
			_ = vad.Execute(context.Background(), data)
		}
	}
}

func BenchmarkTenVAD_Process_SequentialStream_100Chunks(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(1280)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = vad.Execute(context.Background(), data)
		}
	}
}

// Mixed content benchmarks

func BenchmarkTenVAD_Process_MixedContent_SpeechSilence(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	speech := generateBenchmarkSineWave(8000, 440, 0.8)
	silence := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), speech)
		_ = vad.Execute(context.Background(), silence)
	}
}

func BenchmarkTenVAD_Process_MixedContent_Alternating(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	chunks := []internal_type.UserAudioReceivedPacket{
		generateBenchmarkSineWave(1280, 440, 0.8),
		generateBenchmarkSilence(1280),
		generateBenchmarkSineWave(1280, 880, 0.7),
		generateBenchmarkSilence(1280),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, chunk := range chunks {
			_ = vad.Execute(context.Background(), chunk)
		}
	}
}

// Initialization benchmark

func BenchmarkTenVAD_Initialization(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(b, 0.5)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vad, err := newTenVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			b.Skipf("ten_vad library not available: %v", err)
		}
		_ = vad.Close(context.Background())
	}
}

// Callback overhead benchmark

func BenchmarkTenVAD_Process_WithCallback(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()

	callbackCount := 0
	callback := func(context.Context, ...internal_type.Packet) error {
		callbackCount++
		return nil
	}
	opts := newTestOptions(b, 0.3)

	vad, err := newTenVADForTest(b.Context(), logger, callback, opts)
	if err != nil {
		b.Skipf("ten_vad library not available: %v", err)
	}
	b.Cleanup(func() { vad.Close(context.Background()) })

	speech := generateBenchmarkSineWave(8000, 440, 0.8)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), speech)
	}
	b.ReportMetric(float64(callbackCount)/float64(b.N), "callbacks/op")
}

// Throughput benchmark

func BenchmarkTenVAD_Throughput_RealTime(b *testing.B) {
	vad := newBenchmarkTenVAD(b, 0.5)
	data := generateBenchmarkSilence(16000) // 1 second of audio

	b.ResetTimer()
	b.ReportAllocs()

	var totalSamples int64
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
		totalSamples += 16000
	}

	samplesPerSec := float64(totalSamples) / b.Elapsed().Seconds()
	b.ReportMetric(samplesPerSec, "samples/sec")
	b.ReportMetric(samplesPerSec/16000, "x_realtime")
}
