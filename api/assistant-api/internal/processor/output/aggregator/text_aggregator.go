// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package internal_output_processors provides the default sentence-boundary
// text aggregator for streaming LLM responses.
//
// The aggregator accumulates incoming text deltas, splits them at sentence
// boundaries, and pushes complete sentences directly to the onPacket callback.
// It supports multilingual punctuation (Latin, CJK, Devanagari, Arabic)
// and handles context switching between concurrent speakers/contexts.
//
// # Usage
//
//	agg, err := NewDefaultLLMTextAggregator(ctx, logger, func(ctx context.Context, pkts ...internal_type.Packet) error {
//	    for _, pkt := range pkts {
//	        process(pkt)
//	    }
//	    return nil
//	})
//	if err != nil { ... }
//	defer agg.Close()
//
//	agg.Aggregate(ctx, deltaPacket1, deltaPacket2, donePacket)
package internal_output_aggregator_processors

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// ============================================================================
// Constants
// ============================================================================

// sentenceBoundaries defines punctuation marks that delimit sentence endings
// across multiple writing systems: Latin, CJK, Devanagari, and Arabic.
var sentenceBoundaries = []string{
	".", "!", "?", "|", ";", ":", "…", // Latin / general
	"。", "．", // CJK full stop / fullwidth full stop
	"।", // Devanagari danda
	"۔", // Arabic full stop
}

const (
	// emitBufferPrealloc is the initial capacity for the per-call emit buffer,
	// sized to avoid reallocation in the common case of a few sentences.
	emitBufferPrealloc = 8
)

// ============================================================================
// textAggregator — sentence-level LLM text aggregator
// ============================================================================

// textAggregator implements internal_type.LLMTextAggregator using regex-based
// sentence boundary detection. It accumulates streamed text deltas, extracts
// complete sentences at punctuation boundaries, and pushes them directly to
// the onPacket callback as TextToSpeechTextPacket values.
//
// Thread safety: all mutable state is guarded by mu. The onPacket callback is
// invoked outside the lock to prevent deadlocks with slow consumers.
type textAggregator struct {
	logger commons.Logger

	onPacket func(context.Context, ...internal_type.Packet) error

	closed bool

	// mu guards buffer, currentContext, closed, and toEmitBuffer.
	mu sync.Mutex

	// Buffering state: accumulates partial text until a sentence boundary is found.
	buffer         strings.Builder
	currentContext string

	// boundaryRegex is the pre-compiled pattern matching any sentence boundary
	// followed by optional trailing whitespace.
	boundaryRegex *regexp.Regexp

	// toEmitBuffer is a reusable slice that collects packets to emit during
	// a single Aggregate call, reducing per-call heap allocations.
	toEmitBuffer []internal_type.Packet
}

// NewDefaultLLMTextAggregator creates a sentence-boundary text aggregator.
//
// Sentence boundaries are statically defined to support multiple languages and
// punctuation styles (Latin, CJK, Devanagari, Arabic). The boundary regex is
// compiled once during construction.
//
// Returns an error if the boundary regex compilation fails.
func NewLLMTextAggregator(_ context.Context, logger commons.Logger, onPacket func(context.Context, ...internal_type.Packet) error) (internal_type.LLMTextAggregator, error) {
	regex, err := compileBoundaryRegex()
	if err != nil {
		return nil, err
	}
	return &textAggregator{
		logger:        logger,
		onPacket:      onPacket,
		toEmitBuffer:  make([]internal_type.Packet, 0, emitBufferPrealloc),
		boundaryRegex: regex,
	}, nil
}

// compileBoundaryRegex builds a regex that matches any sentence boundary
// character. Whitespace after the boundary is intentionally NOT consumed
// so that it is preserved as a leading space in the next emitted chunk,
// preventing TTS engines from merging words across sentence boundaries.
func compileBoundaryRegex() (*regexp.Regexp, error) {
	parts := make([]string, len(sentenceBoundaries))
	for i, b := range sentenceBoundaries {
		parts[i] = regexp.QuoteMeta(b)
	}

	pattern := fmt.Sprintf(`(%s)`, strings.Join(parts, "|"))
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile sentence boundary regex: %w", err)
	}
	return regex, nil
}

// ============================================================================
// LLMTextAggregator interface implementation
// ============================================================================

// Aggregate processes one or more LLM packets and pushes completed sentences
// to the onPacket callback as TextToSpeechTextPacket values.
//
// Behaviour per packet type:
//   - LLMResponseDeltaPacket: text is appended to the buffer. If a context
//     switch is detected (different ContextID), the buffer is reset first.
//     Complete sentences are extracted at boundary positions and pushed.
//   - LLMResponseDonePacket: any remaining buffered text for the active
//     context is flushed as TextToSpeechTextPacket{IsFinal: false}, then a final
//     TextToSpeechTextPacket{IsFinal: true} is pushed to signal end of generation.
//
// Returns an error if the aggregator has been closed.
func (st *textAggregator) Aggregate(ctx context.Context, pkts ...internal_type.LLMPacket) error {
	toEmit, err := st.processPackets(pkts)
	if err != nil {
		return err
	}

	if len(toEmit) == 0 || st.onPacket == nil {
		return nil
	}

	return st.onPacket(ctx, toEmit...)
}

// Close gracefully shuts down the aggregator.
//
// After Close is called, subsequent Aggregate calls return an error.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (st *textAggregator) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed {
		return nil
	}

	st.buffer.Reset()
	st.currentContext = ""
	st.closed = true

	return nil
}

// ============================================================================
// Internal: packet processing (lock-guarded)
// ============================================================================

// processPackets processes all packets under a single lock acquisition.
// Returns a snapshot of packets to emit, allowing the caller to invoke
// the callback without holding the lock.
func (st *textAggregator) processPackets(pkts []internal_type.LLMPacket) ([]internal_type.Packet, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed {
		return nil, errors.New("text aggregator is closed")
	}

	// Reset the reusable emit buffer for this call.
	st.toEmitBuffer = st.toEmitBuffer[:0]

	for _, pkt := range pkts {
		st.dispatchPacketLocked(pkt)
	}

	// Snapshot the emit buffer so the caller can invoke the callback outside the lock.
	snapshot := make([]internal_type.Packet, len(st.toEmitBuffer))
	copy(snapshot, st.toEmitBuffer)
	return snapshot, nil
}

// dispatchPacketLocked routes a single LLM packet to the appropriate handler.
// MUST be called with mu held.
func (st *textAggregator) dispatchPacketLocked(pkt internal_type.LLMPacket) {
	switch input := pkt.(type) {
	case internal_type.LLMResponseDeltaPacket:
		st.handleDeltaLocked(input)
	case internal_type.LLMResponseDonePacket:
		st.handleDoneLocked(input)
	default:
		st.logger.Warnf("unsupported LLM packet type: %T", pkt)
	}
}

// handleDeltaLocked appends delta text to the buffer and extracts any
// complete sentences at boundary positions.
// MUST be called with mu held.
func (st *textAggregator) handleDeltaLocked(delta internal_type.LLMResponseDeltaPacket) {
	// Context switch: discard the previous context's partial buffer.
	if delta.ContextID != st.currentContext && st.currentContext != "" {
		st.buffer.Reset()
	}
	st.currentContext = delta.ContextID

	st.buffer.WriteString(delta.Text)
	st.extractSentencesAtBoundaryLocked(delta.ContextID)
}

// handleDoneLocked flushes any remaining buffered text for the active context
// as a non-final TextToSpeechTextPacket, then emits a final TextToSpeechTextPacket to signal
// end of generation.
// MUST be called with mu held.
func (st *textAggregator) handleDoneLocked(done internal_type.LLMResponseDonePacket) {
	if done.ContextID == st.currentContext {
		st.flushBufferLocked(done.ContextID)
		st.currentContext = ""
	}
	st.toEmitBuffer = append(st.toEmitBuffer, internal_type.TextToSpeechDonePacket{
		ContextID: done.ContextID,
		Text:      done.Text,
	})
}

// ============================================================================
// Internal: sentence extraction and buffer management
// ============================================================================

// extractSentencesAtBoundaryLocked scans the buffer for sentence boundaries,
// emits all complete text up to the last boundary as a single TextToSpeechTextPacket,
// and retains any trailing partial sentence in the buffer.
// MUST be called with mu held.
func (st *textAggregator) extractSentencesAtBoundaryLocked(contextID string) {
	text := st.buffer.String()

	matches := st.boundaryRegex.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return
	}

	// The last match end position is the split point between complete and
	// incomplete text.
	lastBoundaryEnd := matches[len(matches)-1][1]
	if lastBoundaryEnd == 0 {
		return
	}

	if complete := text[:lastBoundaryEnd]; complete != "" {
		st.toEmitBuffer = append(st.toEmitBuffer, internal_type.TextToSpeechTextPacket{
			ContextID: contextID,
			Text:      complete,
		})
	}

	// Retain any trailing partial sentence after the last boundary.
	st.buffer.Reset()
	if lastBoundaryEnd < len(text) {
		st.buffer.WriteString(text[lastBoundaryEnd:])
	}
}

// flushBufferLocked emits any non-empty buffered text as a non-final
// TextToSpeechTextPacket and resets the buffer.
// MUST be called with mu held.
func (st *textAggregator) flushBufferLocked(contextID string) {
	if remaining := st.buffer.String(); remaining != "" {
		st.toEmitBuffer = append(st.toEmitBuffer, internal_type.TextToSpeechTextPacket{
			ContextID: contextID,
			Text:      remaining,
		})
	}
	st.buffer.Reset()
}
