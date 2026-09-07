// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_linear

import (
	"math"
	"sync"
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/protos"
)

// Baseline single-op benchmarks
func BenchmarkResample(b *testing.B) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resampler.Resample(data, source, target)
	}
}

func BenchmarkConvertToFloat32Samples(b *testing.B) {
	resampler := newTestResampler(b)
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resampler.ConvertToFloat32Samples(data, config)
	}
}

func BenchmarkConvertToByteSamples(b *testing.B) {
	resampler := newTestResampler(b)
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	samples := make([]float32, 100000)
	for i := range samples {
		samples[i] = float32(math.Sin(float64(i) * 2 * math.Pi / 1000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resampler.ConvertToByteSamples(samples, config)
	}
}

func BenchmarkGetAudioInfo(b *testing.B) {
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = internal_audio.GetAudioInfo(data, config)
	}
}

// Concurrent/parallel scaling benchmarks
func BenchmarkResampleSequential(b *testing.B) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resampler.Resample(data, source, target)
	}
}

func benchParallelResample(b *testing.B, goroutines int) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkResampleParallel2Cores(b *testing.B)  { benchParallelResample(b, 2) }
func BenchmarkResampleParallel4Cores(b *testing.B)  { benchParallelResample(b, 4) }
func BenchmarkResampleParallel8Cores(b *testing.B)  { benchParallelResample(b, 8) }
func BenchmarkResampleParallel16Cores(b *testing.B) { benchParallelResample(b, 16) }

func benchDataSizeParallel(b *testing.B, samples int, goroutines int) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(samples)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkSmallDataParallel(b *testing.B)  { benchDataSizeParallel(b, 10000, 8) }
func BenchmarkMediumDataParallel(b *testing.B) { benchDataSizeParallel(b, 500000, 8) }
func BenchmarkLargeDataParallel(b *testing.B)  { benchDataSizeParallel(b, 1000000, 8) }

func benchConvertFloat32Parallel(b *testing.B, goroutines int) {
	resampler := newTestResampler(b)
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	data := generateLinear16Data(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.ConvertToFloat32Samples(data, config)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConvertFloat32Parallel(b *testing.B) { benchConvertFloat32Parallel(b, 8) }

func benchConvertByteParallel(b *testing.B, goroutines int) {
	resampler := newTestResampler(b)
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	samples := make([]float32, 100000)
	for i := range samples {
		samples[i] = float32(math.Sin(float64(i) * 2 * math.Pi / 1000))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.ConvertToByteSamples(samples, config)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConvertByteParallel(b *testing.B) { benchConvertByteParallel(b, 8) }

func BenchmarkMultiFormatParallel(b *testing.B) {
	resampler := newTestResampler(b)
	formats := [][2]*protos.AudioConfig{
		{internal_audio.NewLinear16khzMonoAudioConfig(), internal_audio.NewLinear24khzMonoAudioConfig()},
		{internal_audio.NewLinear16khzMonoAudioConfig(), internal_audio.NewMulaw8khzMonoAudioConfig()},
		{internal_audio.NewMulaw8khzMonoAudioConfig(), internal_audio.NewLinear16khzMonoAudioConfig()},
	}
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, fmt := range formats {
			wg.Add(1)
			go func(source, target *protos.AudioConfig) {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}(fmt[0], fmt[1])
		}
		wg.Wait()
	}
}

func BenchmarkHighConcurrencyResampling(b *testing.B) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkMixedOperationsParallel(b *testing.B) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		for j := 0; j < 4; j++ { // resample ops
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		for j := 0; j < 2; j++ { // float32 conversions
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.ConvertToFloat32Samples(data, source)
			}()
		}
		for j := 0; j < 2; j++ { // audio info
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = internal_audio.GetAudioInfo(data, source)
			}()
		}

		wg.Wait()
	}
}

func BenchmarkResampleWithChannelConversion(b *testing.B) {
	resampler := newTestResampler(b)
	source := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
	target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 2}
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkComplexTransformationParallel(b *testing.B) {
	resampler := newTestResampler(b)
	source := &protos.AudioConfig{SampleRate: 8000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
	target := &protos.AudioConfig{SampleRate: 48000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 2}
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkStressTest(b *testing.B) {
	resampler := newTestResampler(b)
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 256; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = resampler.Resample(data, source, target)
			}()
		}
		wg.Wait()
	}
}
