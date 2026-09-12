// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWhisperFeaturesPythonParity(t *testing.T) {
	metadata, err := os.ReadFile("testdata/features/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		Commit string `json:"reference_commit"`
		Cases  []struct {
			Name         string `json:"name"`
			File         string `json:"file"`
			AudioSamples int    `json:"audio_samples"`
			SHA256       string `json:"sha256"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(metadata, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Commit == "" || len(reference.Cases) == 0 {
		t.Fatal("missing Python reference provenance or cases")
	}

	// Covers float32 reduction order and float64 DFT/rFFT rounding, not bit identity.
	const tolerance = 2e-5
	wf := newWhisperFeatures()
	scratch := newWhisperFeatureScratch()
	output := make([]float32, whisperNMels*whisperMaxFrames)
	for _, fixture := range reference.Cases {
		t.Run(fixture.Name, func(t *testing.T) {
			compressed, err := os.ReadFile(filepath.Join("testdata/features", fixture.File))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(compressed)); got != fixture.SHA256 {
				t.Fatalf("fixture SHA-256 = %s, want %s", got, fixture.SHA256)
			}
			audio, expected := readWhisperFeatureFixture(t, compressed)
			if len(audio) != fixture.AudioSamples {
				t.Fatalf("audio samples = %d, want %d", len(audio), fixture.AudioSamples)
			}
			original := slices.Clone(audio)
			got := wf.extractInto(audio, output, scratch)
			if len(got) != len(expected) {
				t.Fatalf("feature count = %d, want %d", len(got), len(expected))
			}
			var maxError float64
			var worstIndex int
			for i, value := range got {
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					t.Fatalf("feature %d is not finite: %g", i, value)
				}
				difference := math.Abs(float64(value) - float64(expected[i]))
				if difference > maxError {
					maxError, worstIndex = difference, i
				}
			}
			t.Logf("max absolute error %.9g at mel %d, frame %d", maxError,
				worstIndex/whisperMaxFrames, worstIndex%whisperMaxFrames)
			if maxError > tolerance {
				t.Errorf("max absolute error %.9g exceeds %g: Go %g, Python %g",
					maxError, tolerance, got[worstIndex], expected[worstIndex])
			}
			if !slices.Equal(audio, original) {
				t.Error("feature extraction changed caller audio")
			}
			if allocating := wf.Extract(audio); !slices.Equal(allocating, got) {
				t.Error("fresh extraction differs from reused scratch")
			}
		})
	}
}

func TestWhisperFeaturesConcurrentScratch(t *testing.T) {
	features := newWhisperFeatures()
	for _, name := range []string{"short", "long", "quiet_dc", "silence"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			compressed, err := os.ReadFile(filepath.Join("testdata/features", name+".f32.gz"))
			if err != nil {
				t.Fatal(err)
			}
			audio, _ := readWhisperFeatureFixture(t, compressed)
			expected := features.Extract(audio)
			scratch := newWhisperFeatureScratch()
			for range 3 {
				features.extractInto(nil, scratch.output[:], scratch)
				if got := features.extractInto(audio, scratch.output[:], scratch); !slices.Equal(got, expected) {
					t.Fatal("independent scratch changed feature output after silence")
				}
			}
		})
	}
}

func readWhisperFeatureFixture(t testing.TB, compressed []byte) ([]float32, []float32) {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var counts [2]uint32
	if err := binary.Read(reader, binary.LittleEndian, &counts); err != nil {
		t.Fatal(err)
	}
	if counts[0] > 10*whisperSampleRate+1000 || counts[1] != whisperNMels*whisperMaxFrames {
		t.Fatalf("unexpected fixture shape: %v", counts)
	}
	audio, expected := make([]float32, counts[0]), make([]float32, counts[1])
	for _, values := range [][]float32{audio, expected} {
		if err := binary.Read(reader, binary.LittleEndian, values); err != nil {
			t.Fatal(err)
		}
	}
	var trailing [1]byte
	if n, err := reader.Read(trailing[:]); n != 0 || err != io.EOF {
		t.Fatalf("fixture trailing bytes or checksum failure: bytes=%d, error=%v", n, err)
	}
	return audio, expected
}

func BenchmarkWhisperFeaturesPythonFixture(b *testing.B) {
	for _, name := range []string{"short", "long", "quiet_dc"} {
		b.Run(name, func(b *testing.B) {
			compressed, err := os.ReadFile(filepath.Join("testdata/features", name+".f32.gz"))
			if err != nil {
				b.Fatal(err)
			}
			audio, _ := readWhisperFeatureFixture(b, compressed)
			wf := newWhisperFeatures()
			scratch := newWhisperFeatureScratch()
			output := make([]float32, whisperNMels*whisperMaxFrames)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				wf.extractInto(audio, output, scratch)
			}
		})
	}
}
