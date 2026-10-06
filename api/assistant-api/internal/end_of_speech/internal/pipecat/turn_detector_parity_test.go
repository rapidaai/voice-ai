//go:build integration && cgo

package internal_pipecat

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type pipecatNativeCase struct {
	Name           string   `json:"name"`
	AudioFile      string   `json:"audio_file"`
	FeaturesFile   string   `json:"features_file"`
	AudioSHA256    string   `json:"audio_sha256"`
	FeaturesSHA256 string   `json:"features_sha256"`
	Probability    *float64 `json:"probability"`
	Prediction     *int     `json:"prediction"`

	audio            []float32
	expectedFeatures []float32
}

func loadPipecatNativeReference(tb testing.TB) (*PipecatDetector, []pipecatNativeCase) {
	tb.Helper()
	path := os.Getenv("PIPECAT_BENCHMARK_REFERENCE")
	if path == "" {
		tb.Skip("set PIPECAT_BENCHMARK_REFERENCE to the actual Pipecat runner's reference JSON")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	var reference struct {
		ModelSHA256 string              `json:"model_sha256"`
		Cases       []pipecatNativeCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		tb.Fatalf("read Pipecat reference %q: %v", path, err)
	}
	wantDigest, err := hex.DecodeString(reference.ModelSHA256)
	if err != nil || len(wantDigest) != sha256.Size {
		tb.Fatalf("reference model_sha256 must encode %d bytes: %q", sha256.Size, reference.ModelSHA256)
	}
	if len(reference.Cases) == 0 {
		tb.Fatal("Pipecat reference has no cases")
	}
	modelPath := resolvePctModelPath("")
	model, err := os.Open(modelPath)
	if err != nil {
		tb.Fatal(err)
	}
	digest := sha256.New()
	_, readErr := io.Copy(digest, model)
	closeErr := model.Close()
	if readErr != nil {
		tb.Fatalf("hash Pipecat model %q: %v", modelPath, readErr)
	}
	if closeErr != nil {
		tb.Fatal(closeErr)
	}
	if got := digest.Sum(nil); !bytes.Equal(got, wantDigest) {
		tb.Fatalf("Pipecat model %q SHA-256 = %x, want %s", modelPath, got, reference.ModelSHA256)
	}

	names := make(map[string]bool, len(reference.Cases))
	for index := range reference.Cases {
		fixture := &reference.Cases[index]
		if fixture.Name == "" || names[fixture.Name] {
			tb.Fatalf("reference case %d has an empty or duplicate name: %q", index, fixture.Name)
		}
		names[fixture.Name] = true
		if fixture.Probability == nil || fixture.Prediction == nil {
			tb.Fatalf("reference case %q requires probability and prediction", fixture.Name)
		}
		probability := *fixture.Probability
		if math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
			tb.Fatalf("reference case %q has invalid probability %g", fixture.Name, probability)
		}
		prediction := 0
		if probability > 0.5 {
			prediction = 1
		}
		if *fixture.Prediction != prediction {
			tb.Fatalf("reference case %q prediction = %d, want %d for probability %g",
				fixture.Name, *fixture.Prediction, prediction, probability)
		}
		fixture.audio = readPipecatNativeFloat32File(tb, fixture.AudioFile, fixture.AudioSHA256)
		fixture.expectedFeatures = readPipecatNativeFloat32File(tb, fixture.FeaturesFile, fixture.FeaturesSHA256)
		if len(fixture.expectedFeatures) != whisperNMels*whisperMaxFrames {
			tb.Fatalf("reference case %q feature count = %d, want %d", fixture.Name,
				len(fixture.expectedFeatures), whisperNMels*whisperMaxFrames)
		}
	}

	detector, err := NewPipecatDetector(PipecatDetectorConfig{ModelPath: modelPath})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(detector.Destroy)
	return detector, reference.Cases
}

func readPipecatNativeFloat32File(tb testing.TB, path, expectedSHA256 string) []float32 {
	tb.Helper()
	if !filepath.IsAbs(path) {
		tb.Fatalf("Pipecat fixture path must be absolute: %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	if expectedSHA256 != "" {
		expectedDigest, err := hex.DecodeString(expectedSHA256)
		if err != nil || len(expectedDigest) != sha256.Size {
			tb.Fatalf("fixture %q has invalid SHA-256: %q", path, expectedSHA256)
		}
		actualDigest := sha256.Sum256(data)
		if !bytes.Equal(actualDigest[:], expectedDigest) {
			tb.Fatalf("fixture %q checksum mismatch", path)
		}
	}
	if len(data) == 0 || len(data)%4 != 0 {
		tb.Fatalf("Pipecat fixture %q must contain nonempty little-endian float32 values, got %d bytes", path, len(data))
	}
	values := make([]float32, len(data)/4)
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, values); err != nil {
		tb.Fatalf("read Pipecat float32 fixture %q: %v", path, err)
	}
	for index, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			tb.Fatalf("Pipecat fixture %q value %d is not finite: %g", path, index, value)
		}
	}
	return values
}

func TestPipecatNativeParity(t *testing.T) {
	detector, cases := loadPipecatNativeReference(t)
	for _, fixture := range cases {
		t.Run(fixture.Name, func(t *testing.T) {
			features := detector.features.extractInto(fixture.audio, detector.scratch.output[:], detector.scratch)
			if len(features) != len(fixture.expectedFeatures) {
				t.Fatalf("feature count = %d, want %d", len(features), len(fixture.expectedFeatures))
			}
			var maxError float64
			var worstIndex int
			for index, value := range features {
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					t.Fatalf("production feature %d is not finite: %g", index, value)
				}
				difference := math.Abs(float64(value) - float64(fixture.expectedFeatures[index]))
				if difference > maxError {
					maxError, worstIndex = difference, index
				}
			}
			if maxError > 2e-5 {
				t.Errorf("feature max absolute error = %.9g at index %d, want <= 2e-5", maxError, worstIndex)
			}
			inferenceProbability, err := detector.infer(fixture.expectedFeatures)
			if err != nil {
				t.Fatalf("infer Python features: %v", err)
			}
			predictionProbability, err := detector.Predict(fixture.audio)
			if err != nil {
				t.Fatalf("predict reference audio: %v", err)
			}
			t.Logf("max_feature_error=%.9g mel=%d frame=%d python_probability=%.9g inference_probability=%.9g predict_probability=%.9g prediction=%d",
				maxError, worstIndex/whisperMaxFrames, worstIndex%whisperMaxFrames,
				*fixture.Probability, inferenceProbability, predictionProbability, *fixture.Prediction)
			for _, result := range []struct {
				name        string
				probability float64
				tolerance   float64
			}{
				{"inference", inferenceProbability, 1e-5},
				{"predict", predictionProbability, 1e-4},
			} {
				if math.IsNaN(result.probability) || math.IsInf(result.probability, 0) ||
					result.probability < 0 || result.probability > 1 {
					t.Errorf("%s probability is invalid: %g", result.name, result.probability)
					continue
				}
				if difference := math.Abs(result.probability - *fixture.Probability); difference > result.tolerance {
					t.Errorf("%s probability error = %.9g, want <= %g", result.name, difference, result.tolerance)
				}
				prediction := 0
				if result.probability > 0.5 {
					prediction = 1
				}
				if prediction != *fixture.Prediction {
					t.Errorf("%s decision = %d, want %d", result.name, prediction, *fixture.Prediction)
				}
			}
		})
	}
}
