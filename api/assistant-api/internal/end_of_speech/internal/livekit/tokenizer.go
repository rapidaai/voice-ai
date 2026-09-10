// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// tokenizer encodes LiveKit text with byte-level BPE and supported tokenizer.json stages.
// Added tokens are extracted before text preparation and pretokenization.
type tokenizer struct {
	vocab     map[string]int
	merges    map[mergePair]int
	special   map[string]int
	byteToStr [256]string

	individualDigits   *bool
	addPrefixSpace     bool
	byteLevelPattern   *regexp2.Regexp
	hasTextPreparation bool
}

type mergePair struct {
	a, b string
}

// tokenizerJSON matches the HuggingFace tokenizer.json schema.
type tokenizerJSON struct {
	TextPreparation json.RawMessage `json:"normalizer"`
	PreTokenizer    json.RawMessage `json:"pre_tokenizer"`
	Model           struct {
		Vocab  map[string]int `json:"vocab"`
		Merges [][2]string    `json:"merges"`
	} `json:"model"`
	AddedTokens []struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
		Special bool   `json:"special"`
	} `json:"added_tokens"`
}

type textPreparationJSON struct {
	Type    string                `json:"type"`
	Steps   []textPreparationJSON `json:"normalizers"`
	Pattern *struct {
		Regex string `json:"Regex"`
	} `json:"pattern"`
	Content    *string `json:"content"`
	StripLeft  *bool   `json:"strip_left"`
	StripRight *bool   `json:"strip_right"`
}

type preTokenizerJSON struct {
	Type             string             `json:"type"`
	PreTokenizers    []preTokenizerJSON `json:"pretokenizers"`
	IndividualDigits *bool              `json:"individual_digits"`
	AddPrefixSpace   *bool              `json:"add_prefix_space"`
	TrimOffsets      *bool              `json:"trim_offsets"`
	UseRegex         json.RawMessage    `json:"use_regex"`
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
	if err := t.configureTextPreparation(raw.TextPreparation); err != nil {
		return nil, err
	}
	if err := t.configurePreTokenizer(raw.PreTokenizer); err != nil {
		return nil, err
	}
	t.merges = make(map[mergePair]int, len(raw.Model.Merges))
	for rank, merge := range raw.Model.Merges {
		t.merges[mergePair{a: merge[0], b: merge[1]}] = rank
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

func (t *tokenizer) configureTextPreparation(data json.RawMessage) error {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var config textPreparationJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("%w: %w", errTokenizerTextPreparation, err)
	}
	if config.Type != "Sequence" || len(config.Steps) != 2 || config.Pattern != nil ||
		config.Content != nil || config.StripLeft != nil || config.StripRight != nil {
		return fmt.Errorf("%w: expected whitespace Replace then Strip", errTokenizerTextPreparation)
	}
	replace, strip := config.Steps[0], config.Steps[1]
	if replace.Type != "Replace" || replace.Pattern == nil || replace.Pattern.Regex != `\s+` ||
		replace.Content == nil || *replace.Content != " " || len(replace.Steps) != 0 ||
		replace.StripLeft != nil || replace.StripRight != nil {
		return fmt.Errorf("%w: expected whitespace replacement with one space", errTokenizerTextPreparation)
	}
	if strip.Type != "Strip" || strip.StripLeft == nil || !*strip.StripLeft ||
		strip.StripRight == nil || !*strip.StripRight || len(strip.Steps) != 0 ||
		strip.Pattern != nil || strip.Content != nil {
		return fmt.Errorf("%w: expected Strip on both sides", errTokenizerTextPreparation)
	}
	t.hasTextPreparation = true
	return nil
}

func (t *tokenizer) configurePreTokenizer(data json.RawMessage) error {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var config preTokenizerJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("%w: %w", errTokenizerPreTokenizer, err)
	}
	if config.Type == "Sequence" {
		if len(config.PreTokenizers) != 2 || config.IndividualDigits != nil ||
			config.AddPrefixSpace != nil || config.TrimOffsets != nil || config.UseRegex != nil {
			return fmt.Errorf("%w: expected Digits then ByteLevel", errTokenizerPreTokenizer)
		}
		digits := config.PreTokenizers[0]
		if digits.Type != "Digits" || digits.IndividualDigits == nil ||
			len(digits.PreTokenizers) != 0 || digits.AddPrefixSpace != nil ||
			digits.TrimOffsets != nil || digits.UseRegex != nil {
			return fmt.Errorf("%w: expected Digits with individual_digits", errTokenizerPreTokenizer)
		}
		t.individualDigits = digits.IndividualDigits
		config = config.PreTokenizers[1]
	}
	if config.Type != "ByteLevel" || config.AddPrefixSpace == nil ||
		len(config.PreTokenizers) != 0 || config.IndividualDigits != nil {
		return fmt.Errorf("%w: expected ByteLevel with add_prefix_space", errTokenizerPreTokenizer)
	}
	t.addPrefixSpace = *config.AddPrefixSpace
	// Hugging Face defaults an omitted use_regex to true; trim_offsets affects offsets only.
	useRegex := true
	if config.UseRegex != nil {
		var value *bool
		if err := json.Unmarshal(config.UseRegex, &value); err != nil {
			return fmt.Errorf("%w: use_regex: %w", errTokenizerPreTokenizer, err)
		}
		if value == nil {
			return fmt.Errorf("%w: use_regex must be a boolean", errTokenizerPreTokenizer)
		}
		useRegex = *value
	}
	if useRegex {
		t.byteLevelPattern = regexp2.MustCompile(tokenizerByteLevelPattern, regexp2.None)
	}
	return nil
}

func (t *tokenizer) preTokenize(text string) []string {
	segments := []string{text}
	if t.individualDigits != nil {
		segments = nil
		start, inNumber := 0, false
		for offset, r := range text {
			isNumber := unicode.IsNumber(r)
			if offset > start && (isNumber != inNumber || isNumber && *t.individualDigits) {
				segments = append(segments, text[start:offset])
				start = offset
			}
			inNumber = isNumber
		}
		if start < len(text) {
			segments = append(segments, text[start:])
		}
	}
	var pieces []string
	for _, segment := range segments {
		if t.addPrefixSpace && !strings.HasPrefix(segment, " ") {
			segment = " " + segment
		}
		if t.byteLevelPattern == nil {
			pieces = append(pieces, segment)
			continue
		}
		match, err := t.byteLevelPattern.FindStringMatch(segment)
		for match != nil && err == nil {
			pieces = append(pieces, match.String())
			match, err = t.byteLevelPattern.FindNextMatch(match)
		}
		if err != nil {
			return nil
		}
	}
	return pieces
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
		if t.hasTextPreparation {
			// Strip applies to each non-special segment, not across added-token boundaries.
			seg = strings.Join(strings.Fields(seg), " ")
			if seg == "" {
				continue
			}
		}
		pieces := t.preTokenize(seg)
		if len(pieces) == 0 {
			return nil
		}
		for _, piece := range pieces {
			ids = append(ids, t.bpeEncode(piece)...)
		}
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
	for len(symbols) > 1 {
		mergeIndex, mergeRank := -1, 0
		// Rank only adjacent candidates; scanning the entire vocabulary dominates long histories.
		for index := 0; index < len(symbols)-1; index++ {
			rank, exists := t.merges[mergePair{a: symbols[index], b: symbols[index+1]}]
			if exists && (mergeIndex < 0 || rank < mergeRank) {
				mergeIndex, mergeRank = index, rank
			}
		}
		if mergeIndex < 0 {
			break
		}
		symbols[mergeIndex] += symbols[mergeIndex+1]
		copy(symbols[mergeIndex+1:], symbols[mergeIndex+2:])
		symbols[len(symbols)-1] = ""
		symbols = symbols[:len(symbols)-1]
	}
	return symbols
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
