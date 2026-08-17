// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type testPredictor struct {
	predict func(string) (float64, error)
}

func (predictor testPredictor) Predict(text string) (float64, error) {
	return predictor.predict(text)
}

func (predictor testPredictor) Destroy() {}

// --- tokenizer tests ---

func TestBuildByteEncoder(t *testing.T) {
	tok := &tokenizer{}
	tok.buildByteEncoder()

	// Printable ASCII maps to itself
	assert.Equal(t, "A", tok.byteToStr['A'])
	assert.Equal(t, "z", tok.byteToStr['z'])
	assert.Equal(t, "!", tok.byteToStr['!'])

	// Space (0x20) is not in printable range, should map to extended unicode
	assert.NotEqual(t, " ", tok.byteToStr[' '])
	assert.True(t, len(tok.byteToStr[' ']) > 0)
}

func TestIsGPT2PrintableByte(t *testing.T) {
	assert.True(t, isGPT2PrintableByte('A'))
	assert.True(t, isGPT2PrintableByte('~'))
	assert.True(t, isGPT2PrintableByte('!'))
	assert.True(t, isGPT2PrintableByte(0xa1))
	assert.True(t, isGPT2PrintableByte(0xae))
	assert.True(t, isGPT2PrintableByte(0xff))
	assert.False(t, isGPT2PrintableByte(' '))
	assert.False(t, isGPT2PrintableByte('\t'))
	assert.False(t, isGPT2PrintableByte('\n'))
	assert.False(t, isGPT2PrintableByte(0x00))
	assert.False(t, isGPT2PrintableByte(0xad)) // between 0xac and 0xae
}

func TestApplyMerge(t *testing.T) {
	tok := &tokenizer{}

	symbols := []string{"a", "b", "c", "d"}
	result := tok.applyMerge(symbols, mergePair{a: "b", b: "c"})
	assert.Equal(t, []string{"a", "bc", "d"}, result)

	// No match
	result = tok.applyMerge(symbols, mergePair{a: "x", b: "y"})
	assert.Equal(t, []string{"a", "b", "c", "d"}, result)

	// Multiple matches
	symbols = []string{"a", "b", "a", "b"}
	result = tok.applyMerge(symbols, mergePair{a: "a", b: "b"})
	assert.Equal(t, []string{"ab", "ab"}, result)
}

func TestSplitOnSpecialTokens(t *testing.T) {
	tok := &tokenizer{
		special: map[string]int{
			"<|im_start|>": 49153,
			"<|im_end|>":   49154,
		},
	}

	segments := tok.splitOnSpecialTokens("<|im_start|>user\nhello<|im_end|>")
	assert.Equal(t, []string{"<|im_start|>", "user\nhello", "<|im_end|>"}, segments)

	// No special tokens
	segments = tok.splitOnSpecialTokens("plain text")
	assert.Equal(t, []string{"plain text"}, segments)

	// Only special tokens
	segments = tok.splitOnSpecialTokens("<|im_start|><|im_end|>")
	assert.Equal(t, []string{"<|im_start|>", "<|im_end|>"}, segments)
}

// --- chat_template tests ---

func TestFormatChatTemplateFromHistory_Empty(t *testing.T) {
	result := formatChatTemplateFromHistory(nil, "", 5)
	assert.Equal(t, "", result)
}

func TestFormatChatTemplateFromHistory_CurrentOnly(t *testing.T) {
	result := formatChatTemplateFromHistory(nil, "hello", 5)
	assert.Equal(t, "<|im_start|>user\nhello", result)
}

func TestFormatChatTemplateFromHistory_WithHistory(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello there"},
	}
	result := formatChatTemplateFromHistory(history, "how are you", 5)
	expected := "<|im_start|>user\nhi<|im_end|>\n<|im_start|>assistant\nhello there<|im_end|>\n<|im_start|>user\nhow are you"
	assert.Equal(t, expected, result)
}

func TestFormatChatTemplateFromHistory_MaxTurns(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "old message"},
		{Role: "assistant", Content: "old reply"},
		{Role: "user", Content: "recent message"},
		{Role: "assistant", Content: "recent reply"},
	}

	// maxTurns=2 should only include the last 2 history entries
	result := formatChatTemplateFromHistory(history, "new text", 2)
	assert.NotContains(t, result, "old message")
	assert.Contains(t, result, "recent message")
	assert.Contains(t, result, "recent reply")
	assert.Contains(t, result, "new text")
}

func TestFormatChatTemplateFromHistory_LastMessageOpen(t *testing.T) {
	result := formatChatTemplateFromHistory(nil, "yes", 5)
	// The last message should NOT end with <|im_end|>
	assert.True(t, len(result) > 0)
	assert.False(t, result[len(result)-1] == '>')
	assert.NotContains(t, result, "<|im_end|>")
}

func TestFormatChatTemplateFromHistory_SkipsEmptyMessages(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "hi"},
		{Role: "", Content: "skip me"},
		{Role: "assistant", Content: ""},
		{Role: "assistant", Content: "real reply"},
	}
	result := formatChatTemplateFromHistory(history, "test", 10)
	assert.NotContains(t, result, "skip me")
	assert.Contains(t, result, "real reply")
}

func TestLivekitEndOfSpeech_AssistantHistoryFromLLMResponseDonePacket(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	err := endOfSpeech.Execute(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-history",
		Text:      "hi there",
	})
	require.NoError(t, err)

	assert.Len(t, endOfSpeech.history, 1)
	assert.Equal(t, "assistant", endOfSpeech.history[0].Role)
	assert.Equal(t, "hi there", endOfSpeech.history[0].Content)
}

func TestLivekitEndOfSpeech_EnqueueAfterClose_DoesNotEnqueueCommand(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	close(endOfSpeech.stopCh)

	endOfSpeech.enqueueCommand(workerCommand{fireImmediately: true})

	assert.Equal(t, 0, len(endOfSpeech.commandCh))
}

func TestLivekitEndOfSpeech_UserInputImmediateTriggerUsesQueuedSegmentSnapshot(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					called <- endOfSpeechPacket
				}
			}
			return nil
		},
		commandCh: make(chan workerCommand, 4),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-first",
		Text:      "first",
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-second",
		Text:      "second",
	}))

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-first", packet.ContextID)
		assert.Equal(t, "first", packet.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for first immediate callback")
	}

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-second", packet.ContextID)
		assert.Equal(t, "second", packet.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for second immediate callback")
	}
}

func TestLivekitEndOfSpeech_EnqueueCommandBlocksUntilChannelHasSpace(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.commandCh <- workerCommand{segment: speechSegment{Text: "first", FinalText: "first"}}

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		endOfSpeech.enqueueCommand(workerCommand{segment: speechSegment{Text: "second", FinalText: "second"}})
		close(done)
	}()

	<-started
	select {
	case <-done:
		t.Fatal("enqueueCommand should wait while channel is full")
	case <-time.After(50 * time.Millisecond):
	}

	first := <-endOfSpeech.commandCh
	assert.Equal(t, "first", first.segment.Text)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("enqueueCommand should resume after channel space is available")
	}

	second := <-endOfSpeech.commandCh
	assert.Equal(t, "second", second.segment.Text)
}

func TestLivekitEndOfSpeech_FinalSTTInferenceFailure_UsesFallbackTimeout(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 60 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}

	err := endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-fallback",
		Script:    "fallback path",
	})
	require.NoError(t, err)

	select {
	case command := <-endOfSpeech.commandCh:
		assert.Equal(t, 60*time.Millisecond, command.timeout)
		assert.Equal(t, 0.0, command.confidence)
		assert.Equal(t, "fallback path", command.segment.Text)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for fallback command")
	}
}

func TestLivekitEndOfSpeech_VADEndFlushesCurrentSegment(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state: &endOfSpeechState{
			segment:    speechSegment{ContextID: "ctx-vad", Text: "hello", FinalText: "hello"},
			confidence: 0.73,
		},
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 250 * time.Millisecond,
	}

	err := endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	})
	require.NoError(t, err)

	select {
	case command := <-endOfSpeech.commandCh:
		t.Fatalf("unexpected command on VAD start: %+v", command)
	default:
	}

	err = endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	})
	require.NoError(t, err)

	select {
	case command := <-endOfSpeech.commandCh:
		assert.False(t, command.fireImmediately)
		assert.Equal(t, 20*time.Millisecond, command.timeout)
		assert.Equal(t, "hello", command.segment.Text)
		assert.InDelta(t, 0.73, command.confidence, 0.0001)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for VAD end command")
	}
}

func TestLivekitEndOfSpeech_FaultyVADRecoversPendingFinal(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 60 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-faulty-vad",
		Script:    "hello",
		Interim:   false,
	}))

	select {
	case packet := <-called:
		t.Fatalf("callback fired before stuck VAD recovery elapsed: %+v", packet)
	case <-time.After(90 * time.Millisecond):
	}

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-faulty-vad", packet.ContextID)
		assert.Equal(t, "hello", packet.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback after stuck VAD recovery")
	}

	select {
	case packet := <-called:
		t.Fatalf("unexpected duplicate callback after stuck VAD recovery: %+v", packet)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestLivekitEndOfSpeech_VADEndFlushesPendingFinal(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 100 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-vad-success",
		Script:    "done",
		Interim:   false,
	}))

	deadline := time.After(300 * time.Millisecond)
	for {
		endOfSpeech.mu.RLock()
		pending := endOfSpeech.state.pending != nil
		endOfSpeech.mu.RUnlock()
		if pending {
			break
		}
		select {
		case packet := <-called:
			t.Fatalf("callback fired before VAD end: %+v", packet)
		case <-deadline:
			t.Fatal("timeout waiting for pending final")
		case <-time.After(10 * time.Millisecond):
		}
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-vad-success", packet.ContextID)
		assert.Equal(t, "done", packet.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD end")
	}

	select {
	case packet := <-called:
		t.Fatalf("unexpected duplicate callback after VAD end: %+v", packet)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestLivekitEndOfSpeech_VADEndFlushWindowUsesLateFinalSTT(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    40 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 200 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "Rightly?",
		Interim:   false,
	}))

	select {
	case packet := <-called:
		assert.Equal(t, "me an idea about the how how do I do things? Rightly?", packet.Speech)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for late final callback")
	}
}

func TestLivekitEndOfSpeech_VADEndCompleteClearsPendingInterimWhenFinalIsShorter(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 70 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	ctx := context.Background()
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things? Rightly?",
		Interim:   true,
	}))
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}))
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case packet := <-called:
		assert.Equal(t, "me an idea about the how how do I do things?", packet.Speech)
		require.Len(t, packet.Speechs, 2)
		assert.True(t, packet.Speechs[0].Interim)
		assert.False(t, packet.Speechs[1].Interim)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for final transcript")
	}
}

func TestLivekitEndOfSpeech_StaleTimerCompletionDoesNotShrinkLatestSegment(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					called <- endOfSpeechPacket
				}
			}
			return nil
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 100 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	endOfSpeech.mu.Lock()
	endOfSpeech.state.segment = speechSegment{
		Revision:  1,
		ContextID: "ctx-stale",
		Text:      "me an idea about the how how do I do things?",
		FinalText: "me an idea about the how how do I do things?",
		Timestamp: time.Now(),
	}
	oldSegment := endOfSpeech.state.segment
	endOfSpeech.mu.Unlock()

	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: oldSegment,
		timeout: 30 * time.Millisecond,
	})

	time.Sleep(10 * time.Millisecond)
	latestSegment := speechSegment{
		Revision:  2,
		ContextID: "ctx-stale",
		Text:      "me an idea about the how how do I do things? Rightly?",
		FinalText: "me an idea about the how how do I do things? Rightly?",
		Timestamp: time.Now(),
	}
	endOfSpeech.mu.Lock()
	endOfSpeech.state.segment = latestSegment
	endOfSpeech.mu.Unlock()

	select {
	case packet := <-called:
		t.Fatalf("stale completion fired: %q", packet.Speech)
	case <-time.After(80 * time.Millisecond):
	}

	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: latestSegment,
		timeout: 10 * time.Millisecond,
	})

	select {
	case packet := <-called:
		assert.Equal(t, latestSegment.Text, packet.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for latest completion")
	}
}

func TestLivekitEndOfSpeech_OnlyInterimWithVADDoesNotComplete(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 60 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-interim-only",
		Script:    "interim only",
		Interim:   true,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case packet := <-called:
		t.Fatalf("callback should not fire for interim-only VAD input: %+v", packet)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestLivekitEndOfSpeech_PredictorSerializedUnderConcurrentExecute(t *testing.T) {
	var inFlight int32
	var maxInFlight int32

	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				current := atomic.AddInt32(&inFlight, 1)
				for {
					maximum := atomic.LoadInt32(&maxInFlight)
					if current <= maximum || atomic.CompareAndSwapInt32(&maxInFlight, maximum, current) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&inFlight, -1)
				return 0.0, nil
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  500 * time.Millisecond,
		fallbackTimeout: 50 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 16),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
				ContextID: "ctx-serialized",
				Script:    "hello",
			})
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&maxInFlight))
}

func TestLivekitEndOfSpeech_DetectedEventIncludesModelConfidence(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
				if !ok || event.Record.Event != observability.EOSCompleted {
					continue
				}

				select {
				case events <- event:
				default:
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0.42, nil
			},
		},
		threshold:       0.2,
		quickTimeout:    10 * time.Millisecond,
		silenceTimeout:  100 * time.Millisecond,
		fallbackTimeout: 50 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 4),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	err := endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-confidence",
		Script:    "hello world",
	})
	require.NoError(t, err)

	select {
	case event := <-events:
		assert.Equal(t, "ctx-confidence", event.ContextID)
		assert.Equal(t, "0.4200", event.Record.Attributes["confidence"])
		assert.Equal(t, "hello world", event.Record.Attributes["speech"])
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for detected event")
	}
}

func TestLivekitEndOfSpeech_ObservabilityEventShape(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	events := make(chan internal_type.ObservabilityEventRecordPacket, 4)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 2)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok {
				select {
				case events <- event:
				default:
				}
			}
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
		}
		return nil
	}

	executor, err := newLivekitEndOfSpeechForTest(context.Background(), logger, callback, utils.Option{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer func() { _ = executor.Close(context.Background()) }()

	if err := executor.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-events",
		Text:      "hello world",
	}); err != nil {
		t.Fatalf("execute: %v", err)
	}

	timeout := time.After(500 * time.Millisecond)
	var sawDetected, sawMetric bool
	for !sawDetected || !sawMetric {
		select {
		case event := <-events:
			if event.Record.Event != observability.EOSCompleted {
				continue
			}
			assert.Equal(t, "ctx-events", event.ContextID)
			assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, event.Scope)
			assert.Equal(t, observability.ComponentEOS, event.Record.Component)
			assert.Equal(t, eosName, event.Record.Attributes["provider"])
			assert.Equal(t, "ctx-events", event.Record.Attributes["context_id"])
			assert.Equal(t, "hello world", event.Record.Attributes["speech"])
			assert.Equal(t, "0.0000", event.Record.Attributes["confidence"])
			_, parseErr := strconv.Atoi(event.Record.Attributes["text_to_trigger_ms"])
			assert.NoError(t, parseErr)
			_, parseErr = strconv.Atoi(event.Record.Attributes["wait_to_trigger_ms"])
			assert.NoError(t, parseErr)
			assert.False(t, event.Record.OccurredAt.IsZero())
			sawDetected = true
		case metric := <-metrics:
			if len(metric.Record.Metrics) == 0 {
				continue
			}
			if metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
				continue
			}
			assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, metric.Scope)
			assert.Equal(t, eosName, metric.Record.Attributes["provider"])
			_, parseErr := strconv.Atoi(metric.Record.Metrics[0].Value)
			assert.NoError(t, parseErr)
			sawMetric = true
		case <-timeout:
			t.Fatal("timeout waiting for eos conversation events")
		}
	}
}

func TestLivekitEndOfSpeech_ObservabilityLifecycleEvents(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	events := make(chan internal_type.ObservabilityEventRecordPacket, 8)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 8)
	logs := make(chan internal_type.ObservabilityLogRecordPacket, 8)
	usages := make(chan internal_type.ObservabilityUsageRecordPacket, 2)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok {
				select {
				case events <- event:
				default:
				}
			}
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
			if log, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok {
				select {
				case logs <- log:
				default:
				}
			}
			if usage, ok := packet.(internal_type.ObservabilityUsageRecordPacket); ok {
				select {
				case usages <- usage:
				default:
				}
			}
		}
		return nil
	}

	executor, err := newLivekitEndOfSpeechForTest(context.Background(), logger, callback, utils.Option{})
	require.NoError(t, err)

	require.NoError(t, executor.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-debug",
		Text:      "hello",
	}))

	sawInitMetric := false
	sawInitLog := false
	sawStarted := false
	sawDetected := false
	timeout := time.After(500 * time.Millisecond)
	for !sawInitMetric || !sawInitLog || !sawStarted || !sawDetected {
		select {
		case event := <-events:
			if event.Record.Event == observability.EOSStarted {
				sawStarted = true
				continue
			}
			if event.Record.Event == observability.EOSCompleted {
				sawDetected = true
				continue
			}
			t.Fatalf("unexpected eos event: %+v", event)
		case metric := <-metrics:
			if len(metric.Record.Metrics) > 0 && metric.Record.Metrics[0].Name == observability.MetricEOSInitLatencyMs {
				sawInitMetric = true
			}
		case log := <-logs:
			if log.Record.Level == observability.LevelInfo &&
				log.Record.Attributes["provider"] == eosName &&
				log.Record.Attributes["options"] != "" {
				sawInitLog = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for eos lifecycle events")
		}
	}

	require.NoError(t, executor.Close(context.Background()))

	timeout = time.After(500 * time.Millisecond)
	sawClosed := false
	sawUsage := false
	for !sawClosed || !sawUsage {
		select {
		case event := <-events:
			if event.Record.Event == observability.EOSClosed {
				sawClosed = true
			}
		case usage := <-usages:
			if usage.Record.Component == observability.ComponentName(observability.UsageConversationEOSDuration) {
				assert.Equal(t, eosName, usage.Record.Attributes["provider"])
				sawUsage = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for closed eos event")
		}
	}

	require.NoError(t, executor.Close(context.Background()))
}

func TestLivekitEndOfSpeech_ObservabilityStartedForSpeechToText(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
				if !ok || event.Record.Event != observability.EOSStarted {
					continue
				}
				select {
				case events <- event:
				default:
				}
			}
			return nil
		},
		stopCh: make(chan struct{}),
		state:  &endOfSpeechState{segment: speechSegment{}},
	}
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	ctx := context.Background()
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello",
		Interim:   true,
	}))

	select {
	case event := <-events:
		assert.Equal(t, "ctx-started", event.ContextID)
		assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, event.Scope)
		assert.Equal(t, observability.ComponentEOS, event.Record.Component)
		assert.Equal(t, eosName, event.Record.Attributes["provider"])
		assert.Equal(t, "ctx-started", event.Record.Attributes["context_id"])
		assert.Equal(t, "hello", event.Record.Attributes["speech"])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for eos started event")
	}

	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello again",
		Interim:   true,
	}))

	select {
	case event := <-events:
		t.Fatalf("unexpected duplicate started event: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestLivekitEndOfSpeech_KeepsMetrics(t *testing.T) {
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 2)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				switch typed := packet.(type) {
				case internal_type.ObservabilityMetricRecordPacket:
					select {
					case metrics <- typed:
					default:
					}
				}
			}
			return nil
		},
		threshold:       defaultThreshold,
		quickTimeout:    10 * time.Millisecond,
		silenceTimeout:  100 * time.Millisecond,
		fallbackTimeout: 50 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 4),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-off",
		Text:      "hello",
	}))

	select {
	case metric := <-metrics:
		require.NotEmpty(t, metric.Record.Metrics)
		assert.Equal(t, observability.MetricEOSLatencyMs, metric.Record.Metrics[0].Name)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for eos metric")
	}
}

func TestLivekitEndOfSpeech_MetricUsesLastTimerArm(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				switch typed := packet.(type) {
				case internal_type.ObservabilityEventRecordPacket:
					if typed.Record.Event != observability.EOSCompleted {
						continue
					}
					select {
					case events <- typed:
					default:
					}
				case internal_type.ObservabilityMetricRecordPacket:
					select {
					case metrics <- typed:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 120 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	ctx := context.Background()
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-reset",
		Script:    "hello",
		Interim:   false,
	}))
	time.Sleep(80 * time.Millisecond)
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-reset",
		Script:    "...",
		Interim:   true,
	}))

	timeout := time.After(800 * time.Millisecond)
	var detected internal_type.ObservabilityEventRecordPacket
	var metric internal_type.ObservabilityMetricRecordPacket
	for detected.Record.Event == "" || len(metric.Record.Metrics) == 0 {
		select {
		case detected = <-events:
		case metric = <-metrics:
		case <-timeout:
			t.Fatal("timeout waiting for detected eos packets")
		}
	}

	textMs, err := strconv.Atoi(detected.Record.Attributes["text_to_trigger_ms"])
	require.NoError(t, err)
	waitMs, err := strconv.Atoi(detected.Record.Attributes["wait_to_trigger_ms"])
	require.NoError(t, err)
	require.NotEmpty(t, metric.Record.Metrics)
	assert.Equal(t, observability.MetricEOSLatencyMs, metric.Record.Metrics[0].Name)
	metricMs, err := strconv.Atoi(metric.Record.Metrics[0].Value)
	require.NoError(t, err)

	assert.InDelta(t, waitMs, metricMs, 30)
	assert.Greater(t, textMs, waitMs+40)
}

func TestLivekitEndOfSpeech_RespectsExplicitEmptyConcat(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if endOfSpeechPacket, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					select {
					case called <- endOfSpeechPacket:
					default:
					}
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:       defaultThreshold,
		quickTimeout:    20 * time.Millisecond,
		silenceTimeout:  900 * time.Millisecond,
		fallbackTimeout: 80 * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 8),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	empty := ""
	packets := []internal_type.SpeechToTextPacket{
		{ContextID: "ctx-concat", Script: "I", Interim: false},
		{ContextID: "ctx-concat", Script: "'m", Concat: &empty, Interim: false},
		{ContextID: "ctx-concat", Script: "thinking", Interim: false},
		{ContextID: "ctx-concat", Script: ".", Concat: &empty, Interim: false},
	}
	for _, packet := range packets {
		require.NoError(t, endOfSpeech.Execute(context.Background(), packet))
	}

	select {
	case result := <-called:
		assert.Equal(t, "I'm thinking.", result.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for end of speech")
	}
}
