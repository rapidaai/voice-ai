package internal_pipecat

import "testing"

func BenchmarkWhisperWaveformScaling(b *testing.B) {
	input := make([]float32, whisperMaxSamples)
	samples := make([]float32, len(input))
	for index := range input {
		input[index] = float32(index%32768)/32768 - 0.5
	}
	b.ReportAllocs()
	for b.Loop() {
		copy(samples, input)
		normalize(samples)
	}
}
