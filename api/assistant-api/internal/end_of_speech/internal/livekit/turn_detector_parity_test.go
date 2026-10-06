//go:build integration && cgo

package internal_livekit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type liveKitNativeCase struct {
	Name        string        `json:"name"`
	Language    string        `json:"language"`
	History     []chatMessage `json:"history"`
	Text        string        `json:"text"`
	Complete    *bool         `json:"complete"`
	Prompt      string        `json:"prompt"`
	InputIDs    []int64       `json:"input_ids"`
	Probability *float64      `json:"probability"`
	Decision    *bool         `json:"decision"`
}

type liveKitNativeReference struct {
	Schema           int                 `json:"schema"`
	ModelType        string              `json:"model_type"`
	Threshold        float64             `json:"threshold"`
	MaxHistoryTurns  int                 `json:"max_history_turns"`
	MaxHistoryTokens int                 `json:"max_history_tokens"`
	ModelPath        string              `json:"model_path"`
	ModelSHA256      string              `json:"model_sha256"`
	TokenizerPath    string              `json:"tokenizer_path"`
	TokenizerSHA256  string              `json:"tokenizer_sha256"`
	ORTLibraryPath   string              `json:"ort_library_path"`
	ORTLibrarySHA256 string              `json:"ort_library_sha256"`
	Cases            []liveKitNativeCase `json:"cases"`
}

func (reference liveKitNativeReference) validate() error {
	if reference.Schema != 1 || (reference.ModelType != "en" && reference.ModelType != "multilingual") {
		return fmt.Errorf("invalid schema or model type")
	}
	if math.IsNaN(reference.Threshold) || math.IsInf(reference.Threshold, 0) || reference.Threshold < 0 || reference.Threshold > 1 {
		return fmt.Errorf("invalid threshold")
	}
	if reference.MaxHistoryTurns != int(defaultMaxHistory) || reference.MaxHistoryTokens != maxHistoryTokens {
		return fmt.Errorf("reference history limits differ from the current defaults")
	}
	for _, asset := range []struct{ path, checksum string }{
		{reference.ModelPath, reference.ModelSHA256},
		{reference.TokenizerPath, reference.TokenizerSHA256},
		{reference.ORTLibraryPath, reference.ORTLibrarySHA256},
	} {
		checksum, err := hex.DecodeString(asset.checksum)
		if !filepath.IsAbs(asset.path) || err != nil || len(checksum) != sha256.Size {
			return fmt.Errorf("asset requires an absolute path and SHA-256: %q", asset.path)
		}
	}
	if len(reference.Cases) == 0 {
		return fmt.Errorf("reference has no cases")
	}
	seen := make(map[string]bool, len(reference.Cases))
	for _, fixture := range reference.Cases {
		if fixture.Name == "" || seen[fixture.Name] || fixture.Language == "" || fixture.Prompt == "" ||
			len(fixture.InputIDs) == 0 || len(fixture.InputIDs) > maxHistoryTokens ||
			fixture.Probability == nil || fixture.Decision == nil {
			return fmt.Errorf("invalid or duplicate case %q", fixture.Name)
		}
		seen[fixture.Name] = true
		for _, id := range fixture.InputIDs {
			if id < 0 {
				return fmt.Errorf("negative token ID in %q", fixture.Name)
			}
		}
		p := *fixture.Probability
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 || *fixture.Decision != (p >= reference.Threshold) {
			return fmt.Errorf("invalid probability or decision in %q", fixture.Name)
		}
	}
	return nil
}

func liveKitBenchmarkDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func loadLiveKitNativeReference(tb testing.TB) (*TurnDetector, liveKitNativeReference) {
	tb.Helper()
	path := os.Getenv("LIVEKIT_BENCHMARK_REFERENCE")
	if path == "" {
		tb.Skip("set LIVEKIT_BENCHMARK_REFERENCE to opt into local native model inference")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	var reference liveKitNativeReference
	if err := json.Unmarshal(content, &reference); err != nil {
		tb.Fatal(err)
	}
	if err := reference.validate(); err != nil {
		tb.Fatal(err)
	}
	for _, asset := range []struct{ path, checksum string }{
		{reference.ModelPath, reference.ModelSHA256},
		{reference.TokenizerPath, reference.TokenizerSHA256},
		{reference.ORTLibraryPath, reference.ORTLibrarySHA256},
	} {
		actual, err := liveKitBenchmarkDigest(asset.path)
		if err != nil || actual != asset.checksum {
			tb.Fatalf("asset checksum mismatch: %s: got %s, want %s, error %v", asset.path, actual, asset.checksum, err)
		}
	}
	// Check the loaded library, not merely the build flags or a claimed path.
	var libraries []byte
	switch runtime.GOOS {
	case "darwin":
		libraries, err = exec.Command("/usr/sbin/lsof", "-p", strconv.Itoa(os.Getpid()), "-Fn").Output()
	case "linux":
		libraries, err = os.ReadFile("/proc/self/maps")
	default:
		tb.Fatalf("runtime linkage verification supports macOS and Linux, got %s", runtime.GOOS)
	}
	if err != nil {
		tb.Fatalf("inspect loaded runtime: %v", err)
	}
	wanted, err := filepath.EvalSymlinks(reference.ORTLibraryPath)
	if err != nil {
		tb.Fatal(err)
	}
	loaded := false
	for _, line := range strings.Split(string(libraries), "\n") {
		var candidate string
		if runtime.GOOS == "darwin" {
			candidate = strings.TrimPrefix(line, "n")
		} else if index := strings.IndexByte(line, '/'); index >= 0 {
			candidate = line[index:]
		}
		if !strings.Contains(filepath.Base(candidate), "libonnxruntime") {
			continue
		}
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil && resolved == wanted {
			loaded = true
		}
	}
	if !loaded {
		tb.Fatalf("Go has not loaded the recorded ORT library %s; fix native linker/search paths", wanted)
	}
	detector, err := NewTurnDetector(TurnDetectorConfig{
		ModelPath: reference.ModelPath, TokenizerPath: reference.TokenizerPath, ModelType: reference.ModelType,
	})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(detector.Destroy)
	tb.Logf("model=%s model_sha256=%s tokenizer_sha256=%s ort_sha256=%s threshold=%g",
		reference.ModelType, reference.ModelSHA256, reference.TokenizerSHA256, reference.ORTLibrarySHA256, reference.Threshold)
	return detector, reference
}

func liveKitBenchmarkInfer(detector *TurnDetector, ids []int64) (float64, error) {
	if !detector.multilingual {
		return detector.infer(ids, nil)
	}
	probabilities, err := detector.inferMulti(ids, nil)
	if err != nil {
		return 0, err
	}
	if len(probabilities) == 0 {
		return 0, fmt.Errorf("multilingual inference returned no probabilities")
	}
	return probabilities[len(probabilities)-1], nil
}

func liveKitBenchmarkTokenIDs(detector *TurnDetector, prompt string) []int64 {
	tokens := detector.tok.Encode(prompt)
	if len(tokens) > maxHistoryTokens {
		tokens = tokens[len(tokens)-maxHistoryTokens:]
	}
	ids := make([]int64, len(tokens))
	for i, id := range tokens {
		ids[i] = int64(id)
	}
	return ids
}

func TestLiveKitNativeParity(t *testing.T) {
	detector, reference := loadLiveKitNativeReference(t)
	rows := make([]map[string]any, 0, len(reference.Cases))
	for _, fixture := range reference.Cases {
		t.Run(fixture.Name, func(t *testing.T) {
			prompt := formatChatTemplateFromHistory(fixture.History, fixture.Text, reference.MaxHistoryTurns, reference.ModelType)
			ids := liveKitBenchmarkTokenIDs(detector, fixture.Prompt)
			promptMatch, tokenMatch := prompt == fixture.Prompt, slices.Equal(ids, fixture.InputIDs)
			row := map[string]any{"name": fixture.Name, "prompt": prompt, "input_ids": ids,
				"pipeline_input_ids": liveKitBenchmarkTokenIDs(detector, prompt),
				"prompt_match":       promptMatch, "token_match": tokenMatch, "results": map[string]any{}}
			rows = append(rows, row)
			if !promptMatch {
				t.Errorf("prompt mismatch: Go=%q Python=%q", prompt, fixture.Prompt)
			}
			if !tokenMatch {
				t.Errorf("token mismatch on identical reference prompt: Go=%v Python=%v", ids, fixture.InputIDs)
			}
			for _, stage := range []struct {
				name string
				run  func() (float64, error)
			}{
				{"inference", func() (float64, error) { return liveKitBenchmarkInfer(detector, fixture.InputIDs) }},
				{"predict", func() (float64, error) { return detector.Predict(fixture.Prompt) }},
				{"pipeline", func() (float64, error) { return detector.Predict(prompt) }},
			} {
				p, err := stage.run()
				if err != nil || math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
					row["results"].(map[string]any)[stage.name] = map[string]any{"error": fmt.Sprintf("p=%g error=%v", p, err)}
					t.Errorf("%s invalid result: p=%g error=%v", stage.name, p, err)
					continue
				}
				decision := p >= reference.Threshold
				difference := math.Abs(p - *fixture.Probability)
				row["results"].(map[string]any)[stage.name] = map[string]any{"probability": p, "decision": decision, "absolute_error": difference}
				if difference > 1e-5 || decision != *fixture.Decision {
					t.Errorf("%s: probability=%g reference=%g error=%g decision=%t reference_decision=%t",
						stage.name, p, *fixture.Probability, difference, decision, *fixture.Decision)
				}
			}
		})
	}
	if output := os.Getenv("LIVEKIT_BENCHMARK_OUTPUT"); output != "" {
		sourceHashes := map[string]string{}
		_, source, _, _ := runtime.Caller(0)
		entries, err := os.ReadDir(filepath.Dir(source))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !slices.Contains([]string{".go", ".h", ".c"}, filepath.Ext(entry.Name())) {
				continue
			}
			digest, err := liveKitBenchmarkDigest(filepath.Join(filepath.Dir(source), entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			sourceHashes[entry.Name()] = digest
		}
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		binaryDigest, err := liveKitBenchmarkDigest(executable)
		if err != nil {
			t.Fatal(err)
		}
		referenceDigest, err := liveKitBenchmarkDigest(os.Getenv("LIVEKIT_BENCHMARK_REFERENCE"))
		if err != nil {
			t.Fatal(err)
		}
		content, err := json.MarshalIndent(map[string]any{"schema": 1, "reference_sha256": referenceDigest,
			"model_type": reference.ModelType, "threshold": reference.Threshold, "cases": rows,
			"source_sha256": sourceHashes, "binary_sha256": binaryDigest, "go_version": runtime.Version(),
			"goos": runtime.GOOS, "goarch": runtime.GOARCH, "gomaxprocs": runtime.GOMAXPROCS(0)}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, append(content, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLiveKitBenchmarkReferenceValidation(t *testing.T) {
	p, decision := 0.5, true
	valid := liveKitNativeReference{Schema: 1, ModelType: "en", Threshold: 0.5,
		MaxHistoryTurns: int(defaultMaxHistory), MaxHistoryTokens: maxHistoryTokens,
		ModelPath: "/model.onnx", TokenizerPath: "/tokenizer.json", ORTLibraryPath: "/libonnxruntime.so",
		ModelSHA256: strings.Repeat("a", 64), TokenizerSHA256: strings.Repeat("b", 64), ORTLibrarySHA256: strings.Repeat("c", 64),
		Cases: []liveKitNativeCase{{Name: "one", Language: "en", Prompt: "text", InputIDs: []int64{1}, Probability: &p, Decision: &decision}},
	}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*liveKitNativeReference)
	}{
		{"unknown_model", func(r *liveKitNativeReference) { r.ModelType = "auto" }},
		{"relative_asset", func(r *liveKitNativeReference) { r.ModelPath = "model.onnx" }},
		{"missing_digest", func(r *liveKitNativeReference) { r.ModelSHA256 = "" }},
		{"bad_threshold", func(r *liveKitNativeReference) { r.Threshold = math.NaN() }},
		{"missing_cases", func(r *liveKitNativeReference) { r.Cases = nil }},
		{"duplicate_cases", func(r *liveKitNativeReference) { r.Cases = append(r.Cases, r.Cases[0]) }},
		{"missing_probability", func(r *liveKitNativeReference) { r.Cases[0].Probability = nil }},
		{"bad_decision", func(r *liveKitNativeReference) { r.Cases[0].Decision = new(bool) }},
		{"missing_ids", func(r *liveKitNativeReference) { r.Cases[0].InputIDs = nil }},
		{"negative_id", func(r *liveKitNativeReference) { r.Cases[0].InputIDs = []int64{-1} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := valid
			reference.Cases = slices.Clone(valid.Cases)
			test.edit(&reference)
			if err := reference.validate(); err == nil {
				t.Fatal("expected invalid reference to fail")
			}
		})
	}
}
