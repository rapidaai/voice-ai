//go:build integration && cgo

package internal_pipecat

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"testing"
)

func TestPipecatQualityPredictions(t *testing.T) {
	outputPath := os.Getenv("PIPECAT_QUALITY_OUTPUT")
	if outputPath == "" {
		t.Skip("set PIPECAT_QUALITY_OUTPUT to export quality predictions")
	}
	detector, cases := loadPipecatNativeReference(t)
	reference, err := os.ReadFile(os.Getenv("PIPECAT_BENCHMARK_REFERENCE"))
	if err != nil {
		t.Fatal(err)
	}
	type predictionResult struct {
		Name            string  `json:"name"`
		Probability     float64 `json:"probability"`
		Prediction      int     `json:"prediction"`
		MaxFeatureError float64 `json:"max_feature_error"`
	}
	report := struct {
		ReferenceSHA256 string             `json:"reference_sha256"`
		GoVersion       string             `json:"go_version"`
		Cases           []predictionResult `json:"cases"`
	}{
		ReferenceSHA256: fmt.Sprintf("%x", sha256.Sum256(reference)),
		GoVersion:       runtime.Version(),
		Cases:           make([]predictionResult, 0, len(cases)),
	}
	for _, fixture := range cases {
		t.Run(fixture.Name, func(t *testing.T) {
			if fixture.AudioSHA256 == "" || fixture.FeaturesSHA256 == "" {
				t.Fatal("quality evaluation requires audio and feature SHA-256 values")
			}
			probability, err := detector.Predict(fixture.audio)
			if err != nil {
				t.Fatalf("predict reference audio: %v", err)
			}
			if math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
				t.Fatalf("invalid probability: %g", probability)
			}
			result := predictionResult{Name: fixture.Name, Probability: probability}
			if probability > 0.5 {
				result.Prediction = 1
			}
			for index, value := range detector.scratch.output {
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					t.Fatalf("feature %d is not finite", index)
				}
				result.MaxFeatureError = max(result.MaxFeatureError, math.Abs(float64(value)-float64(fixture.expectedFeatures[index])))
			}
			report.Cases = append(report.Cases, result)
			if result.MaxFeatureError > 2e-5 {
				t.Errorf("feature error = %g, want <= 2e-5", result.MaxFeatureError)
			}
			if difference := math.Abs(probability - *fixture.Probability); difference > 1e-4 {
				t.Errorf("probability error = %g, want <= 1e-4", difference)
			}
			if result.Prediction != *fixture.Prediction {
				t.Errorf("decision = %d, want %d", result.Prediction, *fixture.Prediction)
			}
		})
	}
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, append(output, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
