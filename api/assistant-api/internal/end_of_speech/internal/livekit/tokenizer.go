// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// tokenizer implements a BPE tokenizer compatible with HuggingFace tokenizer.json
// format (GPT-2 style). It loads vocabulary, merges, and special tokens from
// the JSON config and encodes text into token IDs for ONNX model input.
type tokenizer struct {
	vocab     map[string]int
	merges    []mergePair
	special   map[string]int
	byteToStr [256]string
}

type mergePair struct {
	a, b string
}

// tokenizerJSON matches the HuggingFace tokenizer.json schema.
type tokenizerJSON struct {
	Model struct {
		Vocab  map[string]int `json:"vocab"`
		Merges [][2]string    `json:"merges"`
	} `json:"model"`
	AddedTokens []struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
		Special bool   `json:"special"`
	} `json:"added_tokens"`
}

// newTokenizer loads a HuggingFace tokenizer.json and returns a ready tokenizer.
func newTokenizer(path string) (*tokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errTokenizerReadFile, err)
	}

	var raw tokenizerJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %w", errTokenizerUnmarshal, err)
	}

	t := &tokenizer{
		vocab:   raw.Model.Vocab,
		special: make(map[string]int),
	}

	// Parse merge rules (each merge is a [2]string pair: [left, right])
	t.merges = make([]mergePair, 0, len(raw.Model.Merges))
	for _, m := range raw.Model.Merges {
		t.merges = append(t.merges, mergePair{a: m[0], b: m[1]})
	}

	// Register special tokens (including added_tokens that are marked special)
	for _, at := range raw.AddedTokens {
		if at.Special {
			t.special[at.Content] = at.ID
		}
		// Ensure all added tokens are in vocab
		if _, ok := t.vocab[at.Content]; !ok {
			t.vocab[at.Content] = at.ID
		}
	}

	// Build byte-to-unicode mapping (GPT-2 byte-level BPE encoding)
	t.buildByteEncoder()

	return t, nil
}

// Encode tokenizes text into a sequence of token IDs.
// Special tokens in the text are recognized and mapped directly.
func (t *tokenizer) Encode(text string) []int {
	if text == "" {
		return nil
	}

	// Extract and handle special tokens
	segments := t.splitOnSpecialTokens(text)

	var ids []int
	for _, seg := range segments {
		if id, ok := t.special[seg]; ok {
			ids = append(ids, id)
			continue
		}
		// BPE encode normal text
		ids = append(ids, t.bpeEncode(seg)...)
	}
	return ids
}

// splitOnSpecialTokens splits text around special token boundaries.
// Returns segments where special tokens appear as standalone entries.
func (t *tokenizer) splitOnSpecialTokens(text string) []string {
	if len(t.special) == 0 {
		return []string{text}
	}

	// Sort special tokens by length (longest first) for greedy matching
	specials := make([]string, 0, len(t.special))
	for tok := range t.special {
		specials = append(specials, tok)
	}
	sort.Slice(specials, func(i, j int) bool {
		return len(specials[i]) > len(specials[j])
	})

	var segments []string
	remaining := text
	for len(remaining) > 0 {
		earliest := -1
		var matched string
		for _, sp := range specials {
			idx := strings.Index(remaining, sp)
			if idx >= 0 && (earliest < 0 || idx < earliest) {
				earliest = idx
				matched = sp
			}
		}
		if earliest < 0 {
			segments = append(segments, remaining)
			break
		}
		if earliest > 0 {
			segments = append(segments, remaining[:earliest])
		}
		segments = append(segments, matched)
		remaining = remaining[earliest+len(matched):]
	}
	return segments
}

// bpeEncode applies byte-level BPE to a text segment (no special tokens).
func (t *tokenizer) bpeEncode(text string) []int {
	if text == "" {
		return nil
	}

	// Convert text bytes to unicode representation (GPT-2 style)
	symbols := t.bytesToUnicodeSymbols(text)
	if len(symbols) == 0 {
		return nil
	}

	// Apply BPE merges iteratively
	symbols = t.applyMerges(symbols)

	// Convert merged symbols to IDs
	var ids []int
	for _, sym := range symbols {
		if id, ok := t.vocab[sym]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// bytesToUnicodeSymbols converts each byte to its GPT-2 unicode representation.
func (t *tokenizer) bytesToUnicodeSymbols(text string) []string {
	raw := []byte(text)
	symbols := make([]string, len(raw))
	for i, b := range raw {
		symbols[i] = t.byteToStr[b]
	}
	return symbols
}

// applyMerges iteratively applies BPE merges in priority order.
func (t *tokenizer) applyMerges(symbols []string) []string {
	for _, merge := range t.merges {
		symbols = t.applyMerge(symbols, merge)
		if len(symbols) <= 1 {
			break
		}
	}
	return symbols
}

// applyMerge applies a single BPE merge rule across the symbol sequence.
func (t *tokenizer) applyMerge(symbols []string, merge mergePair) []string {
	if len(symbols) < 2 {
		return symbols
	}

	result := make([]string, 0, len(symbols))
	i := 0
	for i < len(symbols) {
		if i < len(symbols)-1 && symbols[i] == merge.a && symbols[i+1] == merge.b {
			result = append(result, merge.a+merge.b)
			i += 2
		} else {
			result = append(result, symbols[i])
			i++
		}
	}
	return result
}

// buildByteEncoder constructs the GPT-2 byte-to-unicode mapping.
// This maps each byte value to a unicode character used in the BPE vocabulary.
func (t *tokenizer) buildByteEncoder() {
	n := 0
	for i := range 256 {
		b := byte(i)
		if isGPT2PrintableByte(b) {
			t.byteToStr[i] = string(rune(b))
		} else {
			// Map non-printable bytes to unicode range starting at U+0100
			t.byteToStr[i] = string(rune(256 + n))
			n++
		}
	}
}

// isGPT2PrintableByte returns true if the byte is directly representable
// in GPT-2's byte-to-unicode mapping (printable ASCII + Latin-1 supplement).
func isGPT2PrintableByte(b byte) bool {
	// '!' (33) through '~' (126)
	if b >= '!' && b <= '~' {
		return true
	}
	// '\u00a1' (161) through '\u00ac' (172)
	if b >= 0xa1 && b <= 0xac {
		return true
	}
	// '\u00ae' (174) through '\u00ff' (255)
	if b >= 0xae && b <= 0xff {
		return true
	}
	return false
}

// SpecialTokenID returns the token ID for a special token string,
// or -1 if the token is not registered.
func (t *tokenizer) SpecialTokenID(token string) int {
	if id, ok := t.special[token]; ok {
		return id
	}
	if id, ok := t.vocab[token]; ok {
		return id
	}
	return -1
}

// VocabSize returns the total vocabulary size.
func (t *tokenizer) VocabSize() int {
	return len(t.vocab)
}

// DecodeToken returns the string representation of a single token ID.
// Used primarily for debugging.
func (t *tokenizer) DecodeToken(id int) string {
	for tok, tid := range t.vocab {
		if tid == id {
			return tok
		}
	}
	return ""
}

// bytesToUTF8 is a helper that decodes GPT-2 encoded tokens back to UTF-8 text.
// This is the inverse of the byte-to-unicode encoding used in GPT-2 BPE.
func bytesToUTF8(tokens []string) string {
	// Build reverse mapping
	var reverseMap [65536]byte
	var hasMapping [65536]bool
	n := 0
	for i := range 256 {
		b := byte(i)
		if isGPT2PrintableByte(b) {
			reverseMap[rune(b)] = b
			hasMapping[rune(b)] = true
		} else {
			r := rune(256 + n)
			reverseMap[r] = b
			hasMapping[r] = true
			n++
		}
	}

	var buf []byte
	for _, tok := range tokens {
		for _, r := range tok {
			if int(r) < len(hasMapping) && hasMapping[r] {
				buf = append(buf, reverseMap[r])
			} else {
				// Fallback: encode rune as UTF-8
				var tmp [4]byte
				n := utf8.EncodeRune(tmp[:], r)
				buf = append(buf, tmp[:n]...)
			}
		}
	}
	return string(buf)
}
