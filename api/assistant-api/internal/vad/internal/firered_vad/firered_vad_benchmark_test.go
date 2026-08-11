// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_firered_vad

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

func newBenchmarkFireRedVAD(b *testing.B, confidence float64) *FireRedVAD {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(b, confidence)

	vad, err := newFireRedVADForTest(b.Context(), logger, callback, opts)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			b.Skipf("firered model not available: %v", err)
		}
		b.Fatal(err)
	}
	b.Cleanup(func() { vad.Close(context.Background()) })
	return vad.(*FireRedVAD)
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

func BenchmarkFireRedVAD_Process_Silence_80ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(1280) // 80ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Silence_100ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(1600) // 100ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Silence_500ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(8000) // 500ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Silence_1s(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(16000) // 1s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Speech_80ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSineWave(1280, 440, 0.8) // 80ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Speech_100ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSineWave(1600, 440, 0.8) // 100ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Speech_500ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSineWave(8000, 440, 0.8) // 500ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_Speech_1s(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSineWave(16000, 440, 0.8) // 1s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

// Different chunk sizes

func BenchmarkFireRedVAD_Process_ChunkSize_10ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(160) // 10ms = 1 frame shift

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_ChunkSize_50ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(800) // 50ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_ChunkSize_200ms(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(3200) // 200ms at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

func BenchmarkFireRedVAD_Process_ChunkSize_2s(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(32000) // 2s at 16kHz

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), data)
	}
}

// Parallel processing benchmarks

func BenchmarkFireRedVAD_Process_Parallel_2Streams(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(b, 0.5)

	vads := make([]*FireRedVAD, 2)
	for i := 0; i < 2; i++ {
		callback := func(context.Context, ...internal_type.Packet) error { return nil }
		vad, err := newFireRedVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
				b.Skipf("firered model not available: %v", err)
			}
			b.Fatal(err)
		}
		vads[i] = vad.(*FireRedVAD)
		b.Cleanup(func() { vad.Close(context.Background()) })
	}

	data := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, vad := range vads {
			wg.Add(1)
			go func(v *FireRedVAD) {
				defer wg.Done()
				_ = v.Execute(context.Background(), data)
			}(vad)
		}
		wg.Wait()
	}
}

func BenchmarkFireRedVAD_Process_Parallel_8Streams(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOptions(b, 0.5)

	vads := make([]*FireRedVAD, 8)
	for i := 0; i < 8; i++ {
		callback := func(context.Context, ...internal_type.Packet) error { return nil }
		vad, err := newFireRedVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
				b.Skipf("firered model not available: %v", err)
			}
			b.Fatal(err)
		}
		vads[i] = vad.(*FireRedVAD)
		b.Cleanup(func() { vad.Close(context.Background()) })
	}

	data := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, vad := range vads {
			wg.Add(1)
			go func(v *FireRedVAD) {
				defer wg.Done()
				_ = v.Execute(context.Background(), data)
			}(vad)
		}
		wg.Wait()
	}
}

// Sequential stream processing

func BenchmarkFireRedVAD_Process_SequentialStream_10Chunks(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(1280) // 80ms production chunks

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = vad.Execute(context.Background(), data)
		}
	}
}

func BenchmarkFireRedVAD_Process_SequentialStream_50Chunks(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	data := generateBenchmarkSilence(1280)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 50; j++ {
			_ = vad.Execute(context.Background(), data)
		}
	}
}

func BenchmarkFireRedVAD_Process_SequentialStream_100Chunks(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
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

func BenchmarkFireRedVAD_Process_MixedContent_SpeechSilence(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
	speech := generateBenchmarkSineWave(8000, 440, 0.8)
	silence := generateBenchmarkSilence(8000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = vad.Execute(context.Background(), speech)
		_ = vad.Execute(context.Background(), silence)
	}
}

// Initialization benchmark

func BenchmarkFireRedVAD_Initialization(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	opts := newTestOptions(b, 0.5)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vad, err := newFireRedVADForTest(b.Context(), logger, callback, opts)
		if err != nil {
			if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
				b.Skipf("firered model not available: %v", err)
			}
			b.Fatal(err)
		}
		_ = vad.Close(context.Background())
	}
}

// Throughput benchmark

func BenchmarkFireRedVAD_Throughput_RealTime(b *testing.B) {
	vad := newBenchmarkFireRedVAD(b, 0.5)
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
