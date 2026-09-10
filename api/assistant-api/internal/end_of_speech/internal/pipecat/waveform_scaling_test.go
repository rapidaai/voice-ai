package internal_pipecat

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestWhisperWaveformScalingReference(t *testing.T) {
	payload, err := os.ReadFile("testdata/waveform_scaling/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		Seed  uint32 `json:"seed"`
		Cases []struct {
			Pattern      string `json:"pattern"`
			Samples      int    `json:"samples"`
			InputSHA256  string `json:"input_sha256"`
			OutputSHA256 string `json:"output_sha256"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(payload, &reference); err != nil {
		t.Fatal(err)
	}
	if len(reference.Cases) == 0 {
		t.Fatal("waveform reference has no cases")
	}
	for _, fixture := range reference.Cases {
		t.Run(fmt.Sprintf("%s_%d", fixture.Pattern, fixture.Samples), func(t *testing.T) {
			samples := make([]float32, fixture.Samples)
			state := reference.Seed
			for index := range samples {
				state = 1664525*state + 1013904223
				value := int32(state>>16) - 32768
				switch fixture.Pattern {
				case "pcm":
					samples[index] = float32(value) / 32768
				case "quiet_dc":
					samples[index] = float32(float32(value)/(1<<30)) + float32(0.125)
				case "cancellation":
					samples[index] = [...]float32{1, 1e-7, -1, 1e-7}[index%4]
				case "constant":
					samples[index] = 0.25
				case "negative_zero":
					samples[index] = math.Float32frombits(1 << 31)
				default:
					t.Fatalf("unknown waveform pattern: %q", fixture.Pattern)
				}
			}
			encoded := make([]byte, len(samples)*4)
			for index, sample := range samples {
				binary.LittleEndian.PutUint32(encoded[index*4:], math.Float32bits(sample))
			}
			if digest := fmt.Sprintf("%x", sha256.Sum256(encoded)); digest != fixture.InputSHA256 {
				t.Fatalf("input digest = %s, want %s", digest, fixture.InputSHA256)
			}
			normalize(samples)
			for index, sample := range samples {
				binary.LittleEndian.PutUint32(encoded[index*4:], math.Float32bits(sample))
			}
			if digest := fmt.Sprintf("%x", sha256.Sum256(encoded)); digest != fixture.OutputSHA256 {
				t.Fatalf("scaled waveform digest = %s, want %s", digest, fixture.OutputSHA256)
			}
		})
	}
}

func TestWhisperWaveformScalingUnevenChunks(t *testing.T) {
	samples := make([]float32, whisperReductionChunkSamples)
	for length := 1; length <= len(samples); length++ {
		for index := range samples[:length] {
			samples[index] = 0.25
		}
		normalize(samples[:length])
		for index, value := range samples[:length] {
			if value != 0 {
				t.Fatalf("constant waveform length %d sample %d = %g, want 0", length, index, value)
			}
		}
	}
}
