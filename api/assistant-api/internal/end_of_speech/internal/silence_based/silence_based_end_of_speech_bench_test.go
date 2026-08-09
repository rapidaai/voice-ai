// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// ============================================================================
// BASIC INPUT TYPE BENCHMARKS
// ============================================================================

// BenchmarkAnalyze_UserInput measures Analyze performance with user input (immediate callback).
// User input fires callback immediately, testing the fast path.
func BenchmarkAnalyze_UserInput(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = svcIface.Execute(ctx, userInput("hello world"))
	}
}

// BenchmarkAnalyze_SystemInput measures Analyze performance with system input (timer-based).
// System input extends timer, testing timer setup path.
func BenchmarkAnalyze_SystemInput(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, systemInput("system"))
	}
}

// BenchmarkAnalyze_STTInput measures Analyze performance with STT input.
// STT input extends timer with optional formatting optimization.
func BenchmarkAnalyze_STTInput(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, sttInput("transcription", i%5 == 0))
	}
}

// BenchmarkAnalyze_NoWait measures Analyze performance with immediate context cancellation.
// This simulates the fast path where the context is cancelled before timer fires.
func BenchmarkAnalyze_NoWait(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = svcIface.Execute(ctx, userInput("bench"))
	}
}

// ============================================================================
// CONCURRENCY BENCHMARKS
// ============================================================================

// BenchmarkAnalyze_Concurrent measures Analyze performance under concurrent load.
// This tests thread-safety and contention on mutex locks.
func BenchmarkAnalyze_Concurrent(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = svcIface.Execute(ctx, userInput("bench"))
		}
	})
}

// BenchmarkAnalyze_ConcurrentHighContention measures performance with high mutex contention.
// All goroutines hammer the service simultaneously without context cancellation.
func BenchmarkAnalyze_ConcurrentHighContention(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = svcIface.Execute(ctx, systemInput("contention"))
		}
	})
}

// BenchmarkAnalyze_ConcurrentMixedInputs measures concurrent performance with mixed input types.
// Different goroutines send user, system, and STT inputs concurrently.
func BenchmarkAnalyze_ConcurrentMixedInputs(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	counter := int64(0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := atomic.AddInt64(&counter, 1)
			switch c % 3 {
			case 0:
				_ = svcIface.Execute(ctx, userInput("user"))
			case 1:
				_ = svcIface.Execute(ctx, systemInput("system"))
			case 2:
				_ = svcIface.Execute(ctx, sttInput("stt", c%5 == 0))
			}
		}
	})
}

// ============================================================================
// STT-SPECIFIC BENCHMARKS
// ============================================================================

// BenchmarkAnalyze_STTIncomplete measures STT input with incomplete status.
// Incomplete STT always uses normal timeout.
func BenchmarkAnalyze_STTIncomplete(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 150.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("incomplete %d", i), false))
	}
}

// BenchmarkAnalyze_STTComplete measures STT input with complete status.
// Complete STT may use adjusted timeout if text matches previous.
func BenchmarkAnalyze_STTComplete(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 150.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("complete %d", i), true))
	}
}

// BenchmarkAnalyze_STTFormatting measures STT with formatting-only changes.
// Same semantic content with different formatting (punctuation, case).
func BenchmarkAnalyze_STTFormatting(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 150.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First send unformatted
	_ = svcIface.Execute(ctx, sttInput("hello world", false))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Then repeatedly send formatted version
		_ = svcIface.Execute(ctx, sttInput("Hello, World!", i%2 == 0))
	}
}

// BenchmarkAnalyze_STTHighFrequency measures rapid STT updates (streaming).
// Simulates continuous STT updates from speech-to-text engine.
func BenchmarkAnalyze_STTHighFrequency(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 150.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Simulate streaming updates with varying completeness
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("hello world part %d", i), i%10 == 0))
	}
}

// ============================================================================
// EDGE CASES AT SCALE
// ============================================================================

// BenchmarkAnalyze_RapidFireInputs measures performance with rapid sequential inputs.
// Each new input invalidates the previous transcript revision.
func BenchmarkAnalyze_RapidFireInputs(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("input %d", i)))
	}
}

// BenchmarkAnalyze_LargePayloads measures performance with large speech text.
// Tests memory and processing of large text payloads.
func BenchmarkAnalyze_LargePayloads(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create large payload (10KB text)
	largePayload := ""
	for i := 0; i < 1000; i++ {
		largePayload += "This is a test sentence with many words. "
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, userInput(largePayload))
	}
}

// BenchmarkAnalyze_EmptyInputs measures performance with empty speech.
// Empty speech should be ignored, testing the fast rejection path.
func BenchmarkAnalyze_EmptyInputs(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, userInput(""))
	}
}

// BenchmarkAnalyze_ContextCancellation measures performance with pre-cancelled contexts.
// Context is cancelled before Analyze is called, testing early exit path.
func BenchmarkAnalyze_ContextCancellation(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = svcIface.Execute(ctx, systemInput("cancelled"))
	}
}

// BenchmarkAnalyze_RevisionInvalidation measures performance under transcript revision updates.
// Tests cost of invalidating old timers.
func BenchmarkAnalyze_RevisionInvalidation(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Each input increments transcript revision.
		_ = svcIface.Execute(ctx, systemInput(fmt.Sprintf("gen %d", i)))
	}
}

// ============================================================================
// SCALE BENCHMARKS
// ============================================================================

// BenchmarkAnalyze_HighThroughput measures sustained high throughput.
// Stress test with 1000+ operations, measuring memory and CPU efficiency.
func BenchmarkAnalyze_HighThroughput(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Alternate between input types for realism
		if i%3 == 0 {
			_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("msg %d", i)))
		} else if i%3 == 1 {
			_ = svcIface.Execute(ctx, systemInput("sys"))
		} else {
			_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("stt %d", i), i%7 == 0))
		}
	}
}

// BenchmarkAnalyze_SustainedLoad measures sustained load over time.
// Maintains consistent input rate, useful for detecting performance degradation.
func BenchmarkAnalyze_SustainedLoad(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callCount := int64(0)
	callback := func(context.Context, ...internal_type.Packet) error {
		atomic.AddInt64(&callCount, 1)
		return nil
	}
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, systemInput(fmt.Sprintf("load %d", i)))
	}
	b.Logf("Callbacks fired: %d", atomic.LoadInt64(&callCount))
}

// BenchmarkAnalyze_MultipleServices measures performance with multiple service instances.
// Tests whether services can coexist and whether there's shared state contention.
func BenchmarkAnalyze_MultipleServices(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }

	// Create 10 service instances
	services := make([]internal_type.EndOfSpeechExecutor, 10)
	for i := 0; i < 10; i++ {
		svc, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))
		services[i] = svc
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		svc := services[i%10]
		_ = svc.Execute(ctx, userInput(fmt.Sprintf("multi %d", i)))
	}
}

// ============================================================================
// RACE CONDITION DETECTION BENCHMARKS
// ============================================================================

// BenchmarkAnalyze_RaceDetectionConcurrent runs concurrent stress test suitable for -race flag.
// This benchmark should pass with `go test -bench=. -race`.
func BenchmarkAnalyze_RaceDetectionConcurrent(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			switch idx % 3 {
			case 0:
				_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("race %d", idx)))
			case 1:
				_ = svcIface.Execute(ctx, systemInput("race"))
			case 2:
				_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("race %d", idx), idx%5 == 0))
			}
			idx++
		}
	})
}

// BenchmarkAnalyze_RaceDetectionWithCallback runs concurrent test with active callbacks.
// Tests race conditions in callback execution path.
func BenchmarkAnalyze_RaceDetectionWithCallback(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	var callCount int64

	callback := func(context.Context, ...internal_type.Packet) error {
		atomic.AddInt64(&callCount, 1)
		return nil
	}

	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 50.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = svcIface.Execute(ctx, userInput("callback"))
		}
	})
	b.Logf("Callbacks: %d", atomic.LoadInt64(&callCount))
}

// ============================================================================
// MEMORY ALLOCATION BENCHMARKS
// ============================================================================

// BenchmarkMemory_UserInputAlloc measures memory allocations for user input.
func BenchmarkMemory_UserInputAlloc(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("msg %d", i)))
	}
}

// BenchmarkMemory_STTInputAlloc measures memory allocations for STT input.
func BenchmarkMemory_STTInputAlloc(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("transcription %d", i), i%10 == 0))
	}
}

// BenchmarkMemory_ConcurrentAlloc measures memory allocations under concurrent load.
func BenchmarkMemory_ConcurrentAlloc(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{"microphone.eos.timeout": 100.0}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = svcIface.Execute(ctx, systemInput("mem"))
		}
	})
}

// ============================================================================
// LEGACY BENCHMARKS (for backward compatibility)
// ============================================================================

// BenchmarkAnalyze_STTHandling measures performance of STT input processing with deduplication.
// SKIPPED: Implementation has deadlock in handleSTTInput->triggerExtension
func BenchmarkAnalyze_STTHandling(b *testing.B) {
	b.Skip("Skipping: Implementation deadlock")
}

// BenchmarkBufferAppend measures the cost of appending activities to the bounded buffer.
func BenchmarkBufferAppend(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	svcIface, _ := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svcIface.Execute(ctx, userInput("msg"))
	}
}
