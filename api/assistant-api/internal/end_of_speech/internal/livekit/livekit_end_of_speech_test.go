// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"errors"
	"os"
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
	predict        func(string) (float64, error)
	predictContext func(context.Context, string) (float64, error)
}

func (predictor testPredictor) Predict(text string) (float64, error) {
	return predictor.predict(text)
}

func (predictor testPredictor) PredictContext(ctx context.Context, text string) (float64, error) {
	if predictor.predictContext != nil {
		return predictor.predictContext(ctx, text)
	}
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

func TestApplyMerges(t *testing.T) {
	for _, test := range []struct {
		name    string
		merges  map[mergePair]int
		symbols []string
		want    []string
	}{
		{name: "no match", merges: map[mergePair]int{{a: "x", b: "y"}: 0}, symbols: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "single pair", merges: map[mergePair]int{{a: "b", b: "c"}: 0}, symbols: []string{"a", "b", "c", "d"}, want: []string{"a", "bc", "d"}},
		{name: "repeated pairs", merges: map[mergePair]int{{a: "a", b: "b"}: 0}, symbols: []string{"a", "b", "a", "b"}, want: []string{"ab", "ab"}},
		{name: "rank before position", merges: map[mergePair]int{{a: "a", b: "b"}: 1, {a: "b", b: "c"}: 0}, symbols: []string{"a", "b", "c"}, want: []string{"a", "bc"}},
		{name: "new adjacent candidate", merges: map[mergePair]int{{a: "a", b: "b"}: 5, {a: "ab", b: "c"}: 0}, symbols: []string{"a", "b", "c"}, want: []string{"abc"}},
		{name: "leftmost equal rank", merges: map[mergePair]int{{a: "a", b: "a"}: 0}, symbols: []string{"a", "a", "a"}, want: []string{"aa", "a"}},
		{name: "single symbol", symbols: []string{"a"}, want: []string{"a"}},
		{name: "empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tokenizer := &tokenizer{merges: test.merges}
			require.Equal(t, test.want, tokenizer.applyMerges(test.symbols))
		})
	}
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
	result := formatChatTemplateFromHistory(nil, "", 5, defaultModelType)
	assert.Equal(t, "", result)
}

func TestFormatChatTemplateFromHistory_CurrentOnly(t *testing.T) {
	result := formatChatTemplateFromHistory(nil, "hello", 5, defaultModelType)
	assert.Equal(t, "<|im_start|><|user|>hello", result)
}

func TestFormatChatTemplateFromHistory_WithHistory(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello there"},
	}
	result := formatChatTemplateFromHistory(history, "how are you", 5, defaultModelType)
	expected := "<|im_start|><|user|>hi<|im_end|><|im_start|><|assistant|>hello there<|im_end|><|im_start|><|user|>how are you"
	assert.Equal(t, expected, result)
}

func TestFormatChatTemplateFromHistory_MaxTurns(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "old message"},
		{Role: "assistant", Content: "old reply"},
		{Role: "user", Content: "recent message"},
		{Role: "assistant", Content: "recent reply"},
	}

	result := formatChatTemplateFromHistory(history, "new text", 2, defaultModelType)
	assert.NotContains(t, result, "old message")
	assert.NotContains(t, result, "recent message")
	assert.Contains(t, result, "recent reply")
	assert.Contains(t, result, "new text")
}

func TestFormatChatTemplateFromHistory_EnglishPreservesCaseAndPunctuation(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "Hello, THERE!!!"},
		{Role: "user", Content: "I'm still-talking."},
		{Role: "assistant", Content: "OK..."},
	}

	result := formatChatTemplateFromHistory(history, "What now?", 10, defaultModelType)

	expected := "<|im_start|><|user|>Hello, THERE!!! I'm still-talking.<|im_end|><|im_start|><|assistant|>OK...<|im_end|><|im_start|><|user|>What now?"
	assert.Equal(t, expected, result)
}

func TestFormatChatTemplateFromHistory_MultilingualCleansAndMergesAdjacentTurns(t *testing.T) {
	history := []chatMessage{
		{Role: "user", Content: "Hello, THERE!!!"},
		{Role: "user", Content: "I'm still-talking."},
		{Role: "assistant", Content: "OK..."},
	}

	result := formatChatTemplateFromHistory(history, "What now?", 10, multilingualModelType)

	expected := "<|im_start|>user\nhello there i'm still-talking<|im_end|>\n<|im_start|>assistant\nok<|im_end|>\n<|im_start|>user\nwhat now"
	assert.Equal(t, expected, result)
}

func TestFormatChatTemplateFromHistory_LastMessageOpen(t *testing.T) {
	result := formatChatTemplateFromHistory(nil, "yes", 5, defaultModelType)
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
	result := formatChatTemplateFromHistory(history, "test", 10, defaultModelType)
	assert.NotContains(t, result, "skip me")
	assert.Contains(t, result, "real reply")
}

func TestLivekitEndOfSpeech_AssistantHistoryFromLLMResponseDonePacket(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh: make(chan struct{}, 1),
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
		commandCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	close(endOfSpeech.stopCh)

	endOfSpeech.enqueueCommand(workerCommand{fireImmediately: true})

	assert.Equal(t, 0, len(endOfSpeech.commandCh))
}

func TestLivekitEndOfSpeech_IgnoresInterruptionForDifferentContext(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		silenceTimeout: 30 * time.Millisecond,
		state: &endOfSpeechState{segment: speechSegment{
			Revision:  1,
			ContextID: "ctx-new",
			Text:      "new turn",
			FinalText: "new turn",
			Timestamp: time.Now(),
		}},
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.EndOfSpeechInterruptionPacket{
		ContextID: "ctx-old",
		Source:    internal_type.InterruptionSourceVad,
	}))

	select {
	case command := <-endOfSpeech.commandCh:
		t.Fatalf("unexpected command for old context: %+v", command)
	default:
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.EndOfSpeechInterruptionPacket{
		ContextID: "ctx-new",
		Source:    internal_type.InterruptionSourceVad,
	}))

	select {
	case <-endOfSpeech.commandCh:
		require.Len(t, endOfSpeech.commands, 1)
		assert.Equal(t, "ctx-new", endOfSpeech.commands[0].segment.ContextID)
	default:
		t.Fatal("expected command for active context")
	}
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
		commandCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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

func TestLivekitEndOfSpeech_CallbackCanSubmitBurst(t *testing.T) {
	completed := make(chan string, 65)
	var endOfSpeech *livekitEndOfSpeech
	endOfSpeech = &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet.Speech
					if packet.Speech == "start" {
						for index := range 64 {
							if err := endOfSpeech.Execute(ctx, internal_type.UserTextReceivedPacket{Text: strconv.Itoa(index)}); err != nil {
								return err
							}
						}
					}
				}
			}
			return nil
		},
		commandCh: make(chan struct{}, 1), stopCh: make(chan struct{}),
		workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.UserTextReceivedPacket{Text: "start"}))
	for index := -1; index < 64; index++ {
		select {
		case speech := <-completed:
			if index == -1 {
				require.Equal(t, "start", speech)
			} else {
				require.Equal(t, strconv.Itoa(index), speech)
			}
		case <-time.After(time.Second):
			t.Fatal("reentrant burst blocked its own delivery worker")
		}
	}
}

func TestLivekitEndOfSpeech_FinalSTTInferenceFailure_UsesQuickTimeout(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0, errors.New("predict failed")
			},
		},
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())

	started := time.Now()
	err := endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-fallback",
		Script:    "fallback path",
	})
	require.NoError(t, err)

	select {
	case packet := <-completed:
		assert.GreaterOrEqual(t, time.Since(started), endOfSpeech.quickTimeout)
		assert.Equal(t, "fallback path", packet.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("inference failure did not use the minimum delay")
	}
}

func TestLivekitEndOfSpeech_VADEndFlushesCurrentSegment(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				return 0.8, nil
			},
		},
		threshold: 0.5,
		commandCh: make(chan struct{}, 1),
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
	case <-endOfSpeech.commandCh:
		require.Len(t, endOfSpeech.commands, 1)
		command := endOfSpeech.commands[0]
		assert.False(t, command.fireImmediately)
		require.False(t, endOfSpeech.state.lastSpeechEnd.IsZero())
		assert.Equal(t, endOfSpeech.state.lastSpeechEnd, command.deadline)
		assert.Equal(t, "hello", command.segment.Text)
		assert.True(t, command.predict)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for VAD end command")
	}
}

func TestLivekitEndOfSpeech_VADStartCancelsPendingFinalUntilVADEnd(t *testing.T) {
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
				return 0.9, nil
			},
		},
		threshold:      0.5,
		quickTimeout:   30 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-vad-restart",
		Script:    "hello",
		Interim:   false,
	}))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))

	select {
	case packet := <-called:
		t.Fatalf("callback fired after speech restarted: %+v", packet)
	case <-time.After(120 * time.Millisecond):
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-vad-restart", packet.ContextID)
		assert.Equal(t, "hello", packet.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD end")
	}

	select {
	case packet := <-called:
		t.Fatalf("unexpected duplicate callback after VAD end: %+v", packet)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestLivekitEndOfSpeech_FinalSTTWhileVADSpeakingWaitsForVADEnd(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	var predictorCalls atomic.Int32
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
				predictorCalls.Add(1)
				return 0.9, nil
			},
		},
		threshold:      0.5,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-missing-vad-end",
		Script:    "hello",
		Interim:   false,
	}))

	assert.Zero(t, predictorCalls.Load())
	select {
	case packet := <-called:
		t.Fatalf("callback fired while VAD remained speaking: %+v", packet)
	case <-time.After(100 * time.Millisecond):
	}
	assert.Zero(t, predictorCalls.Load())

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))
	require.Eventually(t, func() bool { return predictorCalls.Load() == 1 }, time.Second, time.Millisecond)

	select {
	case packet := <-called:
		assert.Equal(t, "ctx-missing-vad-end", packet.ContextID)
		assert.Equal(t, "hello", packet.Speech)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD end")
	}
}

func TestLivekitEndOfSpeech_FinalSTTAfterVADEndUsesModelPrediction(t *testing.T) {
	var predictorCalls int32
	completed := make(chan internal_type.EndOfSpeechPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet
				}
			}
			return nil
		},
		predictor: testPredictor{
			predict: func(string) (float64, error) {
				atomic.AddInt32(&predictorCalls, 1)
				return 0.1, nil
			},
		},
		threshold:      0.5,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 250 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))
	endOfSpeech.mu.RLock()
	lastSpeechEnd := endOfSpeech.state.lastSpeechEnd
	endOfSpeech.mu.RUnlock()
	require.False(t, lastSpeechEnd.IsZero())
	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-final-after-vad",
		Script:    "not done yet",
		Interim:   false,
	}))

	select {
	case packet := <-completed:
		assert.GreaterOrEqual(t, time.Since(lastSpeechEnd), endOfSpeech.silenceTimeout)
		assert.Equal(t, "not done yet", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for model-backed completion")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&predictorCalls))
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
				return 0.9, nil
			},
		},
		threshold:      0.5,
		quickTimeout:   40 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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

	select {
	case packet := <-called:
		t.Fatalf("callback fired before VAD end: %+v", packet)
	case <-time.After(60 * time.Millisecond):
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
		threshold:      defaultThreshold,
		quickTimeout:   40 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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

	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: oldSegment,
		timeout: 30 * time.Millisecond,
	})
	endOfSpeech.mu.Unlock()

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

	endOfSpeech.mu.Lock()
	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: latestSegment,
		timeout: 10 * time.Millisecond,
	})
	endOfSpeech.mu.Unlock()

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
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 500 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		workerDone:     make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())

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

	require.Eventually(t, func() bool { return atomic.LoadInt32(&maxInFlight) > 0 }, time.Second, time.Millisecond)
	require.NoError(t, endOfSpeech.Close(context.Background()))
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
		threshold:      0.2,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: 100 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
	if _, err := os.Stat(resolveModelPath("", false)); err != nil {
		t.Skipf("livekit model asset unavailable: %v", err)
	}
	if _, err := os.Stat(resolveTokenizerPath("")); err != nil {
		t.Skipf("livekit tokenizer asset unavailable: %v", err)
	}

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

	executor, err := New(
		WithContext(context.Background()),
		WithLogger(logger),
		WithOnPacket(callback),
		WithOptions(utils.Option{}),
	)
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
	if _, err := os.Stat(resolveModelPath("", false)); err != nil {
		t.Skipf("livekit model asset unavailable: %v", err)
	}
	if _, err := os.Stat(resolveTokenizerPath("")); err != nil {
		t.Skipf("livekit tokenizer asset unavailable: %v", err)
	}

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

	executor, err := New(
		WithContext(context.Background()),
		WithLogger(logger),
		WithOnPacket(callback),
		WithOptions(utils.Option{}),
	)
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
		stopCh:     make(chan struct{}),
		commandCh:  make(chan struct{}, 1),
		workerDone: make(chan struct{}),
		state:      &endOfSpeechState{segment: speechSegment{}},
	}
	go endOfSpeech.worker()
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
		threshold:      defaultThreshold,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: 100 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
	completed := make(chan internal_type.EndOfSpeechPacket, 1)
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				switch typed := packet.(type) {
				case internal_type.EndOfSpeechPacket:
					select {
					case completed <- typed:
					default:
					}
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
		threshold:      defaultThreshold,
		quickTimeout:   120 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
	assert.InDelta(t, textMs, waitMs, 30, "interim must not rearm the final transcript timer")
	assert.GreaterOrEqual(t, waitMs, 90)
	assert.Equal(t, "hello", detected.Record.Attributes["speech"])
	select {
	case packet := <-completed:
		assert.Equal(t, "hello", packet.Speech)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for final-only completion")
	}
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
		threshold:      defaultThreshold,
		quickTimeout:   20 * time.Millisecond,
		silenceTimeout: 900 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}
	endOfSpeech.workerDone = make(chan struct{})
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
