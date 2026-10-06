// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_aggregator_processors

import (
	"context"
	"fmt"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// BenchmarkNewLLMTextAggregator measures the creation time of an aggregator
func BenchmarkNewLLMTextAggregator(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Close()
	}
}

// BenchmarkSingleTextTokenization measures processing a single sentence
func BenchmarkSingleTextTokenization(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "speaker1",
			Text:      "Hello world.",
		})
		aggregator.Close()
	}
}

// BenchmarkMultipleTexts measures processing multiple sentences
func BenchmarkMultipleTexts(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	sentences := []internal_type.LLMResponseDeltaPacket{
		{ContextID: "speaker1", Text: "First sentence."},
		{ContextID: "speaker1", Text: " Second sentence."},
		{ContextID: "speaker1", Text: " Third sentence."},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for _, s := range sentences {
			aggregator.Aggregate(ctx, s)
		}
		aggregator.Close()
	}
}

// BenchmarkLargeTexts measures processing large sentences
func BenchmarkLargeTexts(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	// Create a large sentence
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "word "
	}
	largeText += "."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "speaker1",
			Text:      largeText,
		})
		aggregator.Close()
	}
}

// BenchmarkMultipleBoundaries measures processing with multiple static boundaries
func BenchmarkMultipleBoundaries(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	testTexts := []string{
		"What is this?",
		"I don't know!",
		"Let's try.",
		"Really; absolutely.",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for _, s := range testTexts {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      s,
			})
		}
		aggregator.Close()
	}
}

// BenchmarkContextSwitching measures context switching overhead
func BenchmarkContextSwitching(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for speaker := 0; speaker < 5; speaker++ {
			for j := 0; j < 3; j++ {
				aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
					ContextID: fmt.Sprintf("speaker%d", speaker),
					Text:      fmt.Sprintf("Text %d.", j),
				})
			}
		}
		aggregator.Close()
	}
}

// BenchmarkCallbackConsumption measures the overhead of processing results via callback
func BenchmarkCallbackConsumption(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)

		// Send sentences
		for j := 0; j < 10; j++ {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      fmt.Sprintf("Text %d.", j),
			})
		}

		aggregator.Close()
	}
}

// BenchmarkCompleteFlag measures processing with LLMResponseDonePacket
func BenchmarkCompleteFlag(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "speaker1",
			Text:      "This is a test",
		}, internal_type.LLMResponseDonePacket{
			ContextID: "speaker1",
		})

		aggregator.Close()
	}
}

// BenchmarkBufferingWithoutPunctuation measures buffering text without sentence boundaries
func BenchmarkBufferingWithoutPunctuation(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for j := 0; j < 5; j++ {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      "Text segment",
			})
		}
		aggregator.Close()
	}
}

// BenchmarkStreamingLargeText measures processing streaming text
func BenchmarkStreamingLargeText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	// Simulate streaming text chunks
	chunks := []string{
		"The quick brown fox ",
		"jumps over the ",
		"lazy dog.",
		" This is a test.",
		" Another sentence follows.",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for _, chunk := range chunks {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      chunk,
			})
		}
		aggregator.Close()
	}
}

// BenchmarkClosing measures the cost of closing an aggregator
func BenchmarkClosing(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Close()
	}
}

// BenchmarkEmptyAndCompleteFlush measures flushing empty buffers
func BenchmarkEmptyAndCompleteFlush(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		// Send empty with complete flag
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "speaker1",
			Text:      "",
		}, internal_type.LLMResponseDonePacket{
			ContextID: "speaker1",
		})
		aggregator.Close()
	}
}

// BenchmarkComplexScenario measures a realistic complex scenario
func BenchmarkComplexScenario(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)

		// Simulate a realistic conversation
		conversationTurns := []struct {
			speaker string
			text    string
		}{
			{"alice", "Hello, "},
			{"alice", "how are you today?"},
			{"bob", " I'm doing great!"},
			{"bob", " How about you."},
			{"alice", " Not bad; "},
			{"alice", "just working on code."},
		}

		for _, turn := range conversationTurns {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: turn.speaker,
				Text:      turn.text,
			})
		}

		// Flush remaining
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "alice",
			Text:      "",
		}, internal_type.LLMResponseDonePacket{
			ContextID: "alice",
		})

		aggregator.Close()
	}
}

// BenchmarkParallelProcessing measures parallel token processing
func BenchmarkParallelProcessing(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      "Hello world.",
			})
			aggregator.Close()
		}
	})
}

// BenchmarkWhitespaceProcessing measures text with various whitespace
func BenchmarkWhitespaceProcessing(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	textWithWhitespace := "  \n\tHello  \n  world.  \t\n  "

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "speaker1",
			Text:      textWithWhitespace,
		})
		aggregator.Close()
	}
}

// BenchmarkUnicodeBoundaries measures processing text with unicode boundaries
func BenchmarkUnicodeBoundaries(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	unicodeTexts := []string{
		"こんにちは。元気ですか。",
		"नमस्ते। कैसे हैं।",
		"Wait… Really…",
		"テスト．完了．",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		aggregator, _ := NewLLMTextAggregator(b.Context(), logger, nil)
		for _, text := range unicodeTexts {
			aggregator.Aggregate(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "speaker1",
				Text:      text,
			})
		}
		aggregator.Close()
	}
}
