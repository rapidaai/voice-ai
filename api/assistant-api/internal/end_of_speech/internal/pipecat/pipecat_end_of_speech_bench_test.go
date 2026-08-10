// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync/atomic"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// ============================================================================
// MEL SPECTROGRAM BENCHMARKS
// ============================================================================

// BenchmarkMelFeatures_Init measures one-time cost of creating the feature extractor.
// Includes Hann window computation and mel filterbank matrix construction.
func BenchmarkMelFeatures_Init(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = newWhisperFeatures()
	}
}

// BenchmarkMelFeatures_Extract_1s measures mel extraction for 1 second of audio.
func BenchmarkMelFeatures_Extract_1s(b *testing.B) {
	wf := newWhisperFeatures()
	audio := generateSineWave(16000, 440)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// BenchmarkMelFeatures_Extract_4s measures mel extraction for 4 seconds of audio.
func BenchmarkMelFeatures_Extract_4s(b *testing.B) {
	wf := newWhisperFeatures()
	audio := generateSineWave(64000, 440)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// BenchmarkMelFeatures_Extract_8s measures mel extraction for a full 8 seconds.
// This is the worst-case as the model always processes 8s of audio.
func BenchmarkMelFeatures_Extract_8s(b *testing.B) {
	wf := newWhisperFeatures()
	audio := generateSineWave(128000, 440)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// BenchmarkMelFeatures_Extract_Silence measures mel extraction for silence.
// Silence exercises the zero-handling and log floor paths.
func BenchmarkMelFeatures_Extract_Silence(b *testing.B) {
	wf := newWhisperFeatures()
	audio := make([]float32, 128000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// BenchmarkMelFeatures_Extract_WhiteNoise measures mel extraction for random-like audio.
func BenchmarkMelFeatures_Extract_WhiteNoise(b *testing.B) {
	wf := newWhisperFeatures()
	audio := make([]float32, 128000)
	// Pseudo-random using LCG (deterministic, no import needed)
	state := uint32(42)
	for i := range audio {
		state = state*1103515245 + 12345
		audio[i] = float32(int32(state)>>16) / 32768.0
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// ============================================================================
// FFT BENCHMARKS
// ============================================================================

// BenchmarkFFT_512 measures a single 512-point FFT (the size used per STFT frame).
func BenchmarkFFT_512(b *testing.B) {
	x := make([]complex128, 512)
	for i := range x {
		x[i] = complex(math.Sin(2.0*math.Pi*float64(i)/512.0), 0)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Reset data
		for j := range x {
			x[j] = complex(math.Sin(2.0*math.Pi*float64(j)/512.0), 0)
		}
		fft(x)
	}
}

// BenchmarkFFT_1024 measures a 1024-point FFT for comparison.
func BenchmarkFFT_1024(b *testing.B) {
	n := 1024
	x := make([]complex128, n)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range x {
			x[j] = complex(math.Sin(2.0*math.Pi*float64(j)/float64(n)), 0)
		}
		fft(x)
	}
}

// ============================================================================
// AUDIO BUFFER BENCHMARKS
// ============================================================================

// BenchmarkAppendAudio_SmallChunk measures appending a typical audio chunk (20ms at 16kHz).
func BenchmarkAppendAudio_SmallChunk(b *testing.B) {
	eos := &pipecatEndOfSpeech{
		audioBuffer: make([]float32, 0, maxAudioSamples),
	}
	pcm := make([]byte, 320*2) // 20ms at 16kHz = 320 samples

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eos.appendAudio(pcm)
		// Periodically reset to avoid growing beyond cap
		if len(eos.audioBuffer) > maxAudioSamples-1000 {
			eos.audioBuffer = eos.audioBuffer[:0]
		}
	}
}

// BenchmarkAppendAudio_WithEviction measures appending when the buffer is full.
func BenchmarkAppendAudio_WithEviction(b *testing.B) {
	eos := &pipecatEndOfSpeech{
		audioBuffer: make([]float32, maxAudioSamples, maxAudioSamples),
	}
	pcm := make([]byte, 320*2)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eos.appendAudio(pcm)
	}
}

// ============================================================================
// NORMALIZE BENCHMARKS
// ============================================================================

// BenchmarkNormalize_128k measures normalization of a full 8-second buffer.
func BenchmarkNormalize_128k(b *testing.B) {
	samples := make([]float32, 128000)
	for i := range samples {
		samples[i] = float32(math.Sin(2.0 * math.Pi * 440.0 * float64(i) / 16000.0))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Copy to avoid mutation across iterations
		buf := make([]float32, len(samples))
		copy(buf, samples)
		normalize(buf)
	}
}

// ============================================================================
// PREPARE AUDIO BENCHMARKS
// ============================================================================

// BenchmarkPrepareAudio_Truncate measures truncation of longer-than-8s audio.
func BenchmarkPrepareAudio_Truncate(b *testing.B) {
	audio := make([]float32, 160000) // 10s

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = prepareAudio(audio)
	}
}

// BenchmarkPrepareAudio_Pad measures padding of short audio.
func BenchmarkPrepareAudio_Pad(b *testing.B) {
	audio := make([]float32, 16000) // 1s

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = prepareAudio(audio)
	}
}

// ============================================================================
// EOS INPUT BENCHMARKS (without ONNX model — fallback path)
// ============================================================================

// BenchmarkExecute_UserInput measures the fast path (immediate fire).
func BenchmarkExecute_UserInput(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = eos.Execute(ctx, userInput("hello"))
	}
}

// BenchmarkExecute_STTInput measures STT processing with timer scheduling.
func BenchmarkExecute_STTInput(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = eos.Execute(ctx, sttInput("transcription", i%5 == 0))
	}
}

// BenchmarkExecute_AudioInput measures the audio accumulation path.
func BenchmarkExecute_AudioInput(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	pkt := audioInput(320) // 20ms chunk
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = eos.Execute(ctx, pkt)
	}
}

// BenchmarkExecute_EmptyInput measures the fast rejection path.
func BenchmarkExecute_EmptyInput(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = eos.Execute(ctx, userInput(""))
	}
}

// ============================================================================
// CONCURRENT BENCHMARKS
// ============================================================================

// BenchmarkExecute_Concurrent measures thread-safe access under load.
func BenchmarkExecute_Concurrent(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = eos.Execute(ctx, userInput("bench"))
		}
	})
}

// BenchmarkExecute_ConcurrentMixed measures concurrent performance with all packet types.
func BenchmarkExecute_ConcurrentMixed(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := int64(0)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := atomic.AddInt64(&counter, 1)
			switch c % 4 {
			case 0:
				_ = eos.Execute(ctx, userInput("user"))
			case 1:
				_ = eos.Execute(ctx, interruptInput())
			case 2:
				_ = eos.Execute(ctx, sttInput("stt", c%5 == 0))
			case 3:
				_ = eos.Execute(ctx, audioInput(160))
			}
		}
	})
}

// BenchmarkExecute_ConcurrentAudioOnly measures high-frequency audio packet ingestion.
func BenchmarkExecute_ConcurrentAudioOnly(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pkt := audioInput(320)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = eos.Execute(ctx, pkt)
		}
	})
}

// ============================================================================
// MEMORY ALLOCATION BENCHMARKS
// ============================================================================

// BenchmarkMemory_MelExtraction measures allocations in the mel extraction hot path.
func BenchmarkMemory_MelExtraction(b *testing.B) {
	wf := newWhisperFeatures()
	audio := generateSineWave(128000, 440)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wf.Extract(audio)
	}
}

// BenchmarkMemory_AudioAppend measures allocations in the audio accumulation path.
func BenchmarkMemory_AudioAppend(b *testing.B) {
	eos := &pipecatEndOfSpeech{
		audioBuffer: make([]float32, 0, maxAudioSamples),
	}
	pcm := make([]byte, 640) // 320 samples

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eos.appendAudio(pcm)
		if len(eos.audioBuffer) > maxAudioSamples-1000 {
			eos.audioBuffer = eos.audioBuffer[:0]
		}
	}
}

// BenchmarkMemory_ReflectPad measures allocations for STFT padding.
func BenchmarkMemory_ReflectPad(b *testing.B) {
	signal := make([]float32, 128000)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = reflectPad(signal, whisperNFFT/2)
	}
}

// ============================================================================
// SCALE BENCHMARKS
// ============================================================================

// BenchmarkExecute_HighThroughput measures sustained mixed-input throughput.
func BenchmarkExecute_HighThroughput(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			_ = eos.Execute(ctx, userInput(fmt.Sprintf("msg %d", i)))
		case 1:
			_ = eos.Execute(ctx, interruptInput())
		case 2:
			_ = eos.Execute(ctx, sttInput(fmt.Sprintf("stt %d", i), i%7 == 0))
		case 3:
			_ = eos.Execute(ctx, audioInput(320))
		}
	}
}

// BenchmarkExecute_RapidFireInputs measures stale timer invalidation efficiency.
func BenchmarkExecute_RapidFireInputs(b *testing.B) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = eos.Execute(ctx, userInput(fmt.Sprintf("input %d", i)))
	}
}

// ============================================================================
// RACE DETECTION BENCHMARKS
// ============================================================================

// BenchmarkExecute_RaceDetection should pass with `go test -bench=. -race`.
func BenchmarkExecute_RaceDetection(b *testing.B) {
	var callCount int64
	callback := func(context.Context, ...internal_type.Packet) error {
		atomic.AddInt64(&callCount, 1)
		return nil
	}
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 50.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			switch idx % 4 {
			case 0:
				_ = eos.Execute(ctx, userInput(fmt.Sprintf("race %d", idx)))
			case 1:
				_ = eos.Execute(ctx, interruptInput())
			case 2:
				_ = eos.Execute(ctx, sttInput(fmt.Sprintf("race %d", idx), idx%5 == 0))
			case 3:
				_ = eos.Execute(ctx, audioInput(160))
			}
			idx++
		}
	})
	b.Logf("Callbacks: %d", atomic.LoadInt64(&callCount))
}

// ============================================================================
// Helpers
// ============================================================================

func generateSineWave(nSamples int, freqHz float64) []float32 {
	audio := make([]float32, nSamples)
	for i := range audio {
		audio[i] = float32(math.Sin(2.0 * math.Pi * freqHz * float64(i) / 16000.0))
	}
	return audio
}

// generatePCM16 creates PCM16 LE bytes for a sine wave.
func generatePCM16(nSamples int, freqHz float64) []byte {
	pcm := make([]byte, nSamples*2)
	for i := 0; i < nSamples; i++ {
		v := int16(16000.0 * math.Sin(2.0*math.Pi*freqHz*float64(i)/16000.0))
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(v))
	}
	return pcm
}
