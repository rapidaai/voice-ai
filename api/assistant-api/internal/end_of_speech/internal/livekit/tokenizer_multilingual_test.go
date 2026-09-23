package internal_livekit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTokenizerMultilingualPinnedNativeFixture(t *testing.T) {
	assetPath := filepath.Join("testdata", "tokenizer", "multilingual", "tokenizer.json")
	asset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := os.ReadFile(filepath.Join("testdata", "tokenizer", "multilingual", "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Provenance struct {
			AssetSHA256             string `json:"asset_sha256"`
			AssetRevision           string `json:"asset_revision"`
			AssetURL                string `json:"asset_url"`
			SubsetSHA256            string `json:"subset_sha256"`
			NativeHFExecuted        bool   `json:"native_hf_executed"`
			NativeSubsetVerified    bool   `json:"native_subset_verified"`
			NativeTokenizersVersion string `json:"native_tokenizers_version"`
			VersionSource           string `json:"version_source"`
			AddSpecialTokens        bool   `json:"add_special_tokens"`
			SourceVocabSize         int    `json:"source_vocab_size"`
			SourceMergeCount        int    `json:"source_merge_count"`
			SubsetVocabSize         int    `json:"subset_vocab_size"`
			SubsetMergeCount        int    `json:"subset_merge_count"`
		} `json:"provenance"`
		Cases []struct {
			Name string `json:"name"`
			Text string `json:"text"`
			IDs  []int  `json:"ids"`
		} `json:"cases"`
		PreTokenizerCases []struct {
			Name       string   `json:"name"`
			Text       string   `json:"text"`
			Boundaries []string `json:"boundaries"`
		} `json:"pretokenizer_cases"`
	}
	if err := json.Unmarshal(reference, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Provenance.AssetSHA256 != "9c5ae00e602b8860cbd784ba82a8aa14e8feecec692e7076590d014d7b7fdafa" ||
		fixture.Provenance.AssetRevision != "87e35fcb1e60a569bea70346191c4886ea92e281" ||
		fixture.Provenance.AssetURL != "https://huggingface.co/livekit/turn-detector/resolve/87e35fcb1e60a569bea70346191c4886ea92e281/tokenizer.json" {
		t.Fatal("fixture must retain the pinned multilingual tokenizer provenance")
	}
	if fixture.Provenance.SubsetSHA256 != "61c14efd3a19d994bb3a28be2f875c59eba06d15860e755f1b30607b666f5598" {
		t.Fatal("fixture must retain the pinned multilingual subset digest")
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(asset)); got != fixture.Provenance.SubsetSHA256 {
		t.Fatalf("subset digest = %s, want %s", got, fixture.Provenance.SubsetSHA256)
	}
	if !fixture.Provenance.NativeHFExecuted || !fixture.Provenance.NativeSubsetVerified ||
		fixture.Provenance.NativeTokenizersVersion != "0.22.2" || fixture.Provenance.VersionSource != "LiveKit agents uv.lock" ||
		fixture.Provenance.AddSpecialTokens || len(fixture.Cases) != 39 || len(fixture.PreTokenizerCases) != 39 {
		t.Fatal("expected 39 ID and boundary cases verified with native tokenizers 0.22.2 on the full asset and subset")
	}
	var subset struct {
		Model struct {
			Vocab  map[string]int `json:"vocab"`
			Merges [][]string     `json:"merges"`
		} `json:"model"`
	}
	if err := json.Unmarshal(asset, &subset); err != nil {
		t.Fatal(err)
	}
	if fixture.Provenance.SourceVocabSize != 151643 || fixture.Provenance.SourceMergeCount != 151387 ||
		fixture.Provenance.SubsetVocabSize != 825 || fixture.Provenance.SubsetMergeCount != 555 ||
		len(subset.Model.Vocab) != fixture.Provenance.SubsetVocabSize || len(subset.Model.Merges) != fixture.Provenance.SubsetMergeCount {
		t.Fatal("fixture must retain the bounded multilingual vocabulary and merge counts")
	}
	tok, err := newTokenizer(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		t.Run("ids/"+tc.Name, func(t *testing.T) {
			t.Parallel()
			for range 3 {
				if got := tok.Encode(tc.Text); !slices.Equal(got, tc.IDs) {
					t.Fatalf("Encode(%q) = %v, want %v", tc.Text, got, tc.IDs)
				}
			}
		})
	}
	for index, tc := range fixture.PreTokenizerCases {
		if tc.Name != fixture.Cases[index].Name || tc.Text != fixture.Cases[index].Text {
			t.Fatalf("boundary case %d must cover the corresponding token-ID input", index)
		}
		t.Run("boundaries/"+tc.Name, func(t *testing.T) {
			t.Parallel()
			if got := tok.preTokenize(tc.Text); !slices.Equal(got, tc.Boundaries) {
				t.Fatalf("preTokenize(%q) = %q, want %q", tc.Text, got, tc.Boundaries)
			}
		})
	}
}
