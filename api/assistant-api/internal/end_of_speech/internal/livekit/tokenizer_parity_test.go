package internal_livekit

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTokenizerPinnedNativeFixture(t *testing.T) {
	assetPath := filepath.Join("testdata", "tokenizer", "tokenizer.json")
	asset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := os.ReadFile(filepath.Join("testdata", "tokenizer", "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Provenance struct {
			AssetSHA256             string `json:"asset_sha256"`
			AssetRevision           string `json:"asset_revision"`
			SubsetSHA256            string `json:"subset_sha256"`
			NativeHFExecuted        bool   `json:"native_hf_executed"`
			NativeSubsetVerified    bool   `json:"native_subset_verified"`
			NativeTokenizersVersion string `json:"native_tokenizers_version"`
		} `json:"provenance"`
		Cases []struct {
			Name string `json:"name"`
			Text string `json:"text"`
			IDs  []int  `json:"ids"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(reference, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Provenance.AssetSHA256 != "6f0dc4b1306b1b17da46b55b696c8f54ba7cb05a1b91dc526fd143f31e882fcb" ||
		fixture.Provenance.AssetRevision != "ebcab0c09c2b62d926e92180d364df3aaae68a09" {
		t.Fatal("fixture must retain the pinned production tokenizer provenance")
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(asset)); got != fixture.Provenance.SubsetSHA256 {
		t.Fatalf("subset digest = %s, want %s", got, fixture.Provenance.SubsetSHA256)
	}
	if !fixture.Provenance.NativeHFExecuted || !fixture.Provenance.NativeSubsetVerified ||
		fixture.Provenance.NativeTokenizersVersion != "0.22.2" || len(fixture.Cases) != 29 {
		t.Fatal("expected 29 cases verified with native tokenizers 0.22.2 on the full asset and subset")
	}
	tok, err := newTokenizer(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			for range 3 {
				if got := tok.Encode(tc.Text); !slices.Equal(got, tc.IDs) {
					t.Fatalf("Encode(%q) = %v, want %v", tc.Text, got, tc.IDs)
				}
			}
			t.Logf("Encode(%q) = %v", tc.Text, tc.IDs)
		})
	}
}

func TestTokenizerPreTokenizerSettings(t *testing.T) {
	asset, err := os.ReadFile(filepath.Join("testdata", "tokenizer", "tokenizer.json"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		config     string
		text       string
		boundaries []string
	}{
		{"absent", "", "foo  bar", []string{"foo  bar"}},
		{"null", "null", "foo  bar", []string{"foo  bar"}},
		{"byte_level", `{"type":"ByteLevel","add_prefix_space":false,"use_regex":true}`, "foo  bar", []string{"foo", " ", " bar"}},
		{"default_regex", `{"type":"ByteLevel","add_prefix_space":false}`, "foo  bar", []string{"foo", " ", " bar"}},
		{"regex_disabled", `{"type":"ByteLevel","add_prefix_space":false,"use_regex":false}`, "foo  bar", []string{"foo  bar"}},
		{"prefix_space", `{"type":"ByteLevel","add_prefix_space":true}`, "foo  bar", []string{" foo", " ", " bar"}},
		{"prefix_already_present", `{"type":"ByteLevel","add_prefix_space":true}`, " foo", []string{" foo"}},
		{"offsets_do_not_change_ids", `{"type":"ByteLevel","add_prefix_space":false,"trim_offsets":false}`, "foo  bar", []string{"foo", " ", " bar"}},
		{"digits_individual", `{"type":"Sequence","pretokenizers":[{"type":"Digits","individual_digits":true},{"type":"ByteLevel","add_prefix_space":false}]}`, "a 123b", []string{"a", " ", "1", "2", "3", "b"}},
		{"digits_contiguous", `{"type":"Sequence","pretokenizers":[{"type":"Digits","individual_digits":false},{"type":"ByteLevel","add_prefix_space":false}]}`, "a 123b", []string{"a", " ", "123", "b"}},
		{"prefix_each_digit_piece", `{"type":"Sequence","pretokenizers":[{"type":"Digits","individual_digits":true},{"type":"ByteLevel","add_prefix_space":true}]}`, "a12b", []string{" a", " 1", " 2", " b"}},
		{"digits_without_regex", `{"type":"Sequence","pretokenizers":[{"type":"Digits","individual_digits":true},{"type":"ByteLevel","add_prefix_space":false,"use_regex":false}]}`, "foo  bar12", []string{"foo  bar", "1", "2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(asset, &raw); err != nil {
				t.Fatal(err)
			}
			delete(raw, "pre_tokenizer")
			delete(raw, "normalizer")
			if tc.config != "" {
				raw["pre_tokenizer"] = json.RawMessage(tc.config)
			}
			data, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "tokenizer.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			tok, err := newTokenizer(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := tok.preTokenize(tc.text); !slices.Equal(got, tc.boundaries) {
				t.Fatalf("boundaries = %q, want %q", got, tc.boundaries)
			}
			var want []int
			for _, piece := range tc.boundaries {
				want = append(want, tok.bpeEncode(piece)...)
			}
			if got := tok.Encode(tc.text); !slices.Equal(got, want) {
				t.Fatalf("Encode(%q) = %v, want %v", tc.text, got, want)
			}
			if tc.name == "absent" || tc.name == "null" {
				if want := []int{15236, 256, 6299}; !slices.Equal(tok.Encode(tc.text), want) {
					t.Fatalf("absent pretokenizer must retain whole-segment encoding %v", want)
				}
			}
		})
	}
}

func TestTokenizerAbsentTextPreparationPreservesWhitespace(t *testing.T) {
	asset, err := os.ReadFile(filepath.Join("testdata", "tokenizer", "tokenizer.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"absent", "null"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(asset, &raw); err != nil {
				t.Fatal(err)
			}
			delete(raw, "normalizer")
			if name == "null" {
				raw["normalizer"] = json.RawMessage("null")
			}
			data, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "tokenizer.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			tok, err := newTokenizer(path)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := tok.Encode("foo  bar"), []int{15236, 216, 2753}; !slices.Equal(got, want) {
				t.Fatalf("Encode without preparation = %v, want %v", got, want)
			}
			if got, want := tok.Encode("<|im_start|> <|im_end|>"), []int{1, 216, 2}; !slices.Equal(got, want) {
				t.Fatalf("special-token whitespace without preparation = %v, want %v", got, want)
			}
		})
	}
}

func TestTokenizerRejectsUnsupportedTextPreparation(t *testing.T) {
	cases := []struct {
		name   string
		config string
	}{
		{"unknown_type", `{"type":"Lowercase"}`},
		{"missing_type", `{}`},
		{"wrong_shape", `[]`},
		{"empty_sequence", `{"type":"Sequence","normalizers":[]}`},
		{"wrong_order", `{"type":"Sequence","normalizers":[{"type":"Strip","strip_left":true,"strip_right":true},{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" "}]}`},
		{"wrong_pattern", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s"},"content":" "},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
		{"literal_pattern", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"String":" "},"content":" "},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
		{"wrong_replacement", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":"_"},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
		{"missing_replacement", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"}},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
		{"partial_strip", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" "},{"type":"Strip","strip_left":true,"strip_right":false}]}`},
		{"null_strip", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" "},{"type":"Strip","strip_left":null,"strip_right":true}]}`},
		{"missing_strip", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" "}]}`},
		{"unknown_setting", `{"type":"Sequence","enabled":true,"normalizers":[]}`},
		{"misplaced_setting", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" ","strip_left":true},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
		{"extra_stage", `{"type":"Sequence","normalizers":[{"type":"Replace","pattern":{"Regex":"\\s+"},"content":" "},{"type":"Strip","strip_left":true,"strip_right":true},{"type":"Strip","strip_left":true,"strip_right":true}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "tokenizer.json")
			data := []byte(`{"model":{"vocab":{"a":1},"merges":[]},"normalizer":` + tc.config + `}`)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			tok, err := newTokenizer(path)
			if tok != nil || !errors.Is(err, errTokenizerTextPreparation) {
				t.Fatalf("newTokenizer() = %v, %v; want unsupported text preparation error", tok, err)
			}
		})
	}
}

func TestTokenizerRejectsUnsupportedPreTokenizer(t *testing.T) {
	cases := []struct {
		name   string
		config string
	}{
		{"unknown_type", `{"type":"Whitespace"}`},
		{"missing_type", `{}`},
		{"wrong_shape", `[]`},
		{"empty_sequence", `{"type":"Sequence","pretokenizers":[]}`},
		{"wrong_order", `{"type":"Sequence","pretokenizers":[{"type":"ByteLevel","add_prefix_space":false},{"type":"Digits","individual_digits":true}]}`},
		{"nested_sequence", `{"type":"Sequence","pretokenizers":[{"type":"Sequence","pretokenizers":[]},{"type":"ByteLevel","add_prefix_space":false}]}`},
		{"missing_digit_setting", `{"type":"Sequence","pretokenizers":[{"type":"Digits"},{"type":"ByteLevel","add_prefix_space":false}]}`},
		{"digits_without_byte_level", `{"type":"Digits","individual_digits":true}`},
		{"missing_prefix_setting", `{"type":"ByteLevel","use_regex":true}`},
		{"invalid_prefix_type", `{"type":"ByteLevel","add_prefix_space":"false"}`},
		{"null_regex", `{"type":"ByteLevel","add_prefix_space":false,"use_regex":null}`},
		{"invalid_regex_type", `{"type":"ByteLevel","add_prefix_space":false,"use_regex":"true"}`},
		{"unknown_setting", `{"type":"ByteLevel","add_prefix_space":false,"pattern":"x"}`},
		{"misplaced_digit_setting", `{"type":"ByteLevel","add_prefix_space":false,"individual_digits":true}`},
		{"misplaced_sequence_setting", `{"type":"Sequence","add_prefix_space":false,"pretokenizers":[{"type":"Digits","individual_digits":true},{"type":"ByteLevel","add_prefix_space":false}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "tokenizer.json")
			data := []byte(`{"model":{"vocab":{"a":1},"merges":[]},"pre_tokenizer":` + tc.config + `}`)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			tok, err := newTokenizer(path)
			if tok != nil || !errors.Is(err, errTokenizerPreTokenizer) {
				t.Fatalf("newTokenizer() = %v, %v; want unsupported pretokenizer error", tok, err)
			}
		})
	}
}
