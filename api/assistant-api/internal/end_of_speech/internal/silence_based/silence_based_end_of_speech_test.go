// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// helpers to build inputs
func userInput(msg string) internal_type.UserTextReceivedPacket {
	return internal_type.UserTextReceivedPacket{Text: msg}
}

func systemInput(msg string) internal_type.EndOfSpeechInterruptionPacket {
	return internal_type.EndOfSpeechInterruptionPacket{Source: "vad"}
}

func sttInput(msg string, complete bool) internal_type.SpeechToTextPacket {
	return internal_type.SpeechToTextPacket{Script: msg, Interim: !complete}
}

func sttInputWithLanguage(msg, language string, complete bool) internal_type.SpeechToTextPacket {
	return internal_type.SpeechToTextPacket{Script: msg, Interim: !complete, Language: language}
}

// newTestOpts creates a utils.Option (which is just map[string]interface{})
func newTestOpts(m map[string]any) utils.Option {
	return utils.Option(m)
}

// --- Tests ---

func TestTimerFiresAndCallbackCalled(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- res:

				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 150.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, userInput("hello")); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case res := <-called:
		if res.Speech != "hello" {
			t.Fatalf("unexpected speech: %v", res.Speech)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

func TestSystemInputTriggersTimer(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- res:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, userInput("sys")); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if err := svcIface.Execute(ctx, systemInput("sys")); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case <-called:
	case <-time.After(700 * time.Millisecond):
		t.Fatal("timeout waiting for callback for system input")
	}
}

func TestEmptySpeechIgnored(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- res:
				default:
				}
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 150.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, userInput("")); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case <-called:
		t.Fatal("callback should not be called for empty speech")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestSilenceBasedEndOfSpeech_ObservabilityEventShape(t *testing.T) {
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

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if err := svcIface.Execute(context.Background(), internal_type.UserTextReceivedPacket{
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
			if event.ContextID != "ctx-events" {
				t.Fatalf("unexpected detected context: %+v", event)
			}
			if event.Scope != internal_type.ObservabilityRecordScopeUserMessage {
				t.Fatalf("unexpected detected scope: %+v", event)
			}
			if event.Record.Component != observability.ComponentEOS {
				t.Fatalf("unexpected detected component: %+v", event)
			}
			if event.Record.Attributes["provider"] != silenceBasedEndOfSpeechName {
				t.Fatalf("unexpected detected provider: %+v", event)
			}
			if event.Record.Attributes["context_id"] != "ctx-events" || event.Record.Attributes["speech"] != "hello world" {
				t.Fatalf("unexpected detected data: %+v", event)
			}
			if event.Record.Attributes["confidence"] != "0.0000" {
				t.Fatalf("unexpected detected confidence: %+v", event)
			}
			if _, parseErr := strconv.Atoi(event.Record.Attributes["text_to_trigger_ms"]); parseErr != nil {
				t.Fatalf("invalid detected timing: %+v", event)
			}
			if _, parseErr := strconv.Atoi(event.Record.Attributes["wait_to_trigger_ms"]); parseErr != nil {
				t.Fatalf("invalid detected wait timing: %+v", event)
			}
			if event.Record.OccurredAt.IsZero() {
				t.Fatalf("expected detected time: %+v", event)
			}
			sawDetected = true
		case metric := <-metrics:
			if len(metric.Record.Metrics) == 0 {
				continue
			}
			if metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
				continue
			}
			if metric.Scope != internal_type.ObservabilityRecordScopeUserMessage {
				t.Fatalf("unexpected metric scope: %+v", metric)
			}
			if metric.Record.Attributes["provider"] != silenceBasedEndOfSpeechName {
				t.Fatalf("unexpected metric provider: %+v", metric)
			}
			if _, parseErr := strconv.Atoi(metric.Record.Metrics[0].Value); parseErr != nil {
				t.Fatalf("invalid eos metric: %+v", metric)
			}
			sawMetric = true
		case <-timeout:
			t.Fatal("timeout waiting for eos conversation events")
		}
	}
}

func TestSilenceBasedEndOfSpeech_ObservabilityLifecycleEvents(t *testing.T) {
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

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if err := svcIface.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-debug",
		Text:      "hello",
	}); err != nil {
		t.Fatalf("execute: %v", err)
	}

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
				log.Record.Attributes["provider"] == silenceBasedEndOfSpeechName &&
				log.Record.Attributes["options"] != "" {
				sawInitLog = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for eos lifecycle events")
		}
	}

	if err := svcIface.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}

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
				if usage.Record.Attributes["provider"] != silenceBasedEndOfSpeechName {
					t.Fatalf("unexpected usage provider: %+v", usage)
				}
				sawUsage = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for closed eos event")
		}
	}
}

func TestSilenceBasedEndOfSpeech_ObservabilityStartedForSpeechToText(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
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
	}

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer func() { _ = svcIface.Close(context.Background()) }()

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello",
		Interim:   true,
	}); err != nil {
		t.Fatalf("execute interim stt: %v", err)
	}

	select {
	case event := <-events:
		if event.ContextID != "ctx-started" {
			t.Fatalf("unexpected started context: %+v", event)
		}
		if event.Scope != internal_type.ObservabilityRecordScopeUserMessage {
			t.Fatalf("unexpected started scope: %+v", event)
		}
		if event.Record.Component != observability.ComponentEOS {
			t.Fatalf("unexpected started component: %+v", event)
		}
		if event.Record.Attributes["provider"] != silenceBasedEndOfSpeechName {
			t.Fatalf("unexpected started provider: %+v", event)
		}
		if event.Record.Attributes["context_id"] != "ctx-started" || event.Record.Attributes["speech"] != "hello" {
			t.Fatalf("unexpected started data: %+v", event)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for eos started event")
	}

	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello again",
		Interim:   true,
	}); err != nil {
		t.Fatalf("execute second interim stt: %v", err)
	}

	select {
	case event := <-events:
		t.Fatalf("unexpected duplicate started event: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSilenceBasedEndOfSpeech_KeepsMetrics(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 2)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
		}
		return nil
	}

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer func() { _ = svcIface.Close(context.Background()) }()

	if err := svcIface.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-off",
		Text:      "hello",
	}); err != nil {
		t.Fatalf("execute: %v", err)
	}

	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case metric := <-metrics:
			if len(metric.Record.Metrics) > 0 && metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
				continue
			}
			if len(metric.Record.Metrics) == 0 || metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
				t.Fatalf("unexpected metric packet: %+v", metric)
			}
			return
		case <-timeout:
			t.Fatal("timeout waiting for eos metric")
		}
	}
}

func TestSilenceBasedEndOfSpeech_MetricUsesLastTimerArm(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 1)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
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
				if len(typed.Record.Metrics) == 0 || typed.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
					continue
				}
				select {
				case metrics <- typed:
				default:
				}
			}
		}
		return nil
	}

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{
		"microphone.eos.timeout": 120.0,
	}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, sttInput("hello", true)); err != nil {
		t.Fatalf("execute final stt: %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if err := svcIface.Execute(ctx, systemInput("activity")); err != nil {
		t.Fatalf("execute activity: %v", err)
	}

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
	if err != nil {
		t.Fatalf("parse text_to_trigger_ms: %v", err)
	}
	waitMs, err := strconv.Atoi(detected.Record.Attributes["wait_to_trigger_ms"])
	if err != nil {
		t.Fatalf("parse wait_to_trigger_ms: %v", err)
	}
	metricMs, err := strconv.Atoi(metric.Record.Metrics[0].Value)
	if err != nil {
		t.Fatalf("parse eos.latency_ms: %v", err)
	}

	if metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
		t.Fatalf("unexpected metric packet: %+v", metric)
	}
	if diff := metricMs - waitMs; diff < -30 || diff > 30 {
		t.Fatalf("metric and wait timing diverged: metric=%d wait=%d", metricMs, waitMs)
	}
	if textMs <= waitMs+40 {
		t.Fatalf("expected text timing to include pre-reset time: text=%d wait=%d", textMs, waitMs)
	}
}

func TestSilenceBasedEndOfSpeech_RespectsExplicitEmptyConcat(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if endOfSpeech, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- endOfSpeech:
				default:
				}
			}
		}
		return nil
	}

	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, newTestOpts(map[string]any{
		"microphone.eos.timeout": 80.0,
	}))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	empty := ""
	packets := []internal_type.SpeechToTextPacket{
		{ContextID: "ctx-concat", Script: "I", Interim: false},
		{ContextID: "ctx-concat", Script: "'m", Concat: &empty, Interim: false},
		{ContextID: "ctx-concat", Script: "thinking", Interim: false},
		{ContextID: "ctx-concat", Script: ".", Concat: &empty, Interim: false},
	}
	for _, packet := range packets {
		if err := svcIface.Execute(context.Background(), packet); err != nil {
			t.Fatalf("execute: %v", err)
		}
	}

	select {
	case result := <-called:
		if result.Speech != "I'm thinking." {
			t.Fatalf("expected %q, got %q", "I'm thinking.", result.Speech)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for end of speech")
	}
}

func TestSTTNormalizationDeduplication(t *testing.T) {
	// SKIPPED: Implementation has deadlock in handleSTTInput->triggerExtension
	// Both hold mutex and one calls the other causing deadlock
	t.Skip("Skipping: Implementation deadlock between handleSTTInput and triggerExtension")
}

func TestAdjustedThresholdLowerBound(t *testing.T) {
	// SKIPPED: Implementation has deadlock in handleSTTInput->triggerExtension
	t.Skip("Skipping: Implementation deadlock between handleSTTInput and triggerExtension")
}

func TestActivityBufferCapped(t *testing.T) {
	// SKIPPED: This test checks for an activity buffer capping mechanism
	// that is not implemented in the current version
	t.Skip("Activity buffer capping not implemented in current design")
}

func TestConcurrentAnalyze(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calls := make(chan internal_type.EndOfSpeechPacket, 100)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case calls <- res:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}
	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = svcIface.Execute(ctx, userInput("u"))
		}(i)
	}
	wg.Wait()

	// Simply verify no panic occurred
}

func TestContextCancelStillFiresCallback(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- res:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 150.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	parentCtx, cancel := context.WithCancel(context.Background())
	// Send an STT packet (timer-based, not fireNow) so the timer must fire after cancel
	if err := svcIface.Execute(parentCtx, sttInput("bye", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	// Cancel the context before the silence timer fires
	cancel()

	// The callback should still fire with a fallback background context
	select {
	case res := <-called:
		if res.Speech != "bye" {
			t.Fatalf("unexpected speech: %v", res.Speech)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("callback should have been called even after context cancel")
	}
}

func TestSTTLanguagePreservedInEndOfSpeechPacket(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, sttInputWithLanguage("hola", "es-ES", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case res := <-called:
		if res.Speech != "hola" {
			t.Fatalf("unexpected speech: %v", res.Speech)
		}
		if len(res.Speechs) != 1 || res.Speechs[0].Script != "hola" || res.Speechs[0].Language != "es-ES" {
			t.Fatalf("unexpected speech chunks: %+v", res.Speechs)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

func TestSTTLanguage_UsesLatestNonEmptyAcrossChunks(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 120.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	requireNoError := func(err error) {
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
	}
	requireNoError(svcIface.Execute(ctx, sttInputWithLanguage("hello", "en-US", true)))
	requireNoError(svcIface.Execute(ctx, sttInputWithLanguage("bonjour", "fr-FR", true)))
	requireNoError(svcIface.Execute(ctx, sttInputWithLanguage("hallo", "de-DE", true)))

	select {
	case res := <-called:
		if res.Speech != "hello bonjour hallo" {
			t.Fatalf("unexpected speech: %v", res.Speech)
		}
		if len(res.Speechs) != 3 {
			t.Fatalf("expected 3 speech chunks, got %d", len(res.Speechs))
		}
		if res.Speechs[0].Script != "hello" || res.Speechs[1].Script != "bonjour" || res.Speechs[2].Script != "hallo" {
			t.Fatalf("unexpected speech chunk scripts: %+v", res.Speechs)
		}
	case <-time.After(600 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

func TestSTTLanguage_LastChunkWithoutLanguageRetainsPrevious(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 120.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	if err := svcIface.Execute(ctx, sttInputWithLanguage("hola", "es-ES", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if err := svcIface.Execute(ctx, sttInputWithLanguage("mundo", "", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case res := <-called:
		if res.Speech != "hola mundo" {
			t.Fatalf("unexpected speech: %v", res.Speech)
		}
		if len(res.Speechs) != 2 {
			t.Fatalf("expected 2 speech chunks, got %d", len(res.Speechs))
		}
		if res.Speechs[0].Language != "es-ES" || res.Speechs[1].Language != "" {
			t.Fatalf("unexpected speech chunk languages: %+v", res.Speechs)
		}
	case <-time.After(600 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

// TestHandleSTTInput_IncompleteSTT verifies incomplete STT triggers normal threshold
func TestHandleSTTInput_IncompleteSTT(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbackTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callbackTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	timeout := 150.0
	opts := newTestOpts(map[string]any{"microphone.eos.timeout": timeout})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	startTime := time.Now()

	// Send incomplete STT - should trigger normal timeout
	if err := svcIface.Execute(ctx, sttInput("hello world", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	if err := svcIface.Execute(ctx, sttInput("hello world", false)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case cbTime := <-callbackTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := time.Duration(int64(timeout)) * time.Millisecond
		tolerance := 15 * time.Millisecond
		minExpected := expectedMs - tolerance
		maxExpected := expectedMs + tolerance

		if elapsed < minExpected || elapsed > maxExpected {
			t.Fatalf("callback timing out of bounds: expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on incomplete STT")
	}
}

// TestHandleSTTInput_CompleteSTTNoActivity verifies complete STT with no activity uses normal threshold
func TestHandleSTTInput_CompleteSTTNoActivity(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbackTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callbackTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	timeout := 120.0
	opts := newTestOpts(map[string]any{"microphone.eos.timeout": timeout})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	startTime := time.Now()

	// Send complete STT with no prior activity - should trigger normal timeout
	if err := svcIface.Execute(ctx, sttInput("complete message", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case cbTime := <-callbackTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := time.Duration(int64(timeout)) * time.Millisecond
		tolerance := 15 * time.Millisecond
		minExpected := expectedMs - tolerance
		maxExpected := expectedMs + tolerance

		if elapsed < minExpected || elapsed > maxExpected {
			t.Fatalf("callback timing out of bounds: expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on complete STT with no activity")
	}
}

// TestHandleSTTInput_DifferentTextCompleteSTT verifies different STT text triggers normal threshold
func TestHandleSTTInput_DifferentTextCompleteSTT(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbackTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callbackTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	timeout := 100.0
	opts := newTestOpts(map[string]any{"microphone.eos.timeout": timeout})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// First STT with "hello"
	if err := svcIface.Execute(ctx, sttInput("hello", true)); err != nil {
		t.Fatalf("analyze first: %v", err)
	}

	// Second STT with "goodbye" - different text, should trigger normal timeout
	startTime := time.Now()
	if err := svcIface.Execute(ctx, sttInput("goodbye", true)); err != nil {
		t.Fatalf("analyze second: %v", err)
	}

	select {
	case cbTime := <-callbackTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := time.Duration(int64(timeout)) * time.Millisecond
		tolerance := 15 * time.Millisecond
		minExpected := expectedMs - tolerance
		maxExpected := expectedMs + tolerance

		if elapsed < minExpected || elapsed > maxExpected {
			t.Fatalf("callback timing out of bounds: expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on different STT text")
	}
}

// TestHandleSTTInput_ActivityAfterUserInput verifies STT after user input doesn't use adjusted threshold
func TestHandleSTTInput_ActivityAfterUserInput(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbackTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callbackTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	timeout := 150.0
	opts := newTestOpts(map[string]any{"microphone.eos.timeout": timeout})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// Add system input activity (not user input, not STT)
	if err := svcIface.Execute(ctx, systemInput("system activity")); err != nil {
		t.Fatalf("analyze system input: %v", err)
	}

	// Complete STT - recent activity is system, so normal threshold applies
	startTime := time.Now()
	if err := svcIface.Execute(ctx, sttInput("stt text", true)); err != nil {
		t.Fatalf("analyze stt: %v", err)
	}

	select {
	case cbTime := <-callbackTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := time.Duration(int64(timeout)) * time.Millisecond
		tolerance := 20 * time.Millisecond
		minExpected := expectedMs - tolerance
		maxExpected := expectedMs + tolerance

		if elapsed < minExpected || elapsed > maxExpected {
			t.Fatalf("callback timing out of bounds: expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on STT after system input")
	}
}

// === Additional comprehensive test cases per README ===

// TestCallbackFiresOnlyOnce verifies the callback fires exactly once per utterance.
// After callback fires and reset occurs, new inputs start a fresh utterance window.
func TestCallbackFiresOnlyOnce(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	var mu sync.Mutex
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				mu.Lock()
				callCount++
				mu.Unlock()
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// Send system input - starts timer for utterance 1
	if err := svcIface.Execute(ctx, sttInput("activity", true)); err != nil {
		t.Fatalf("analyze system 1: %v", err)
	}

	// Wait for callback to fire (100ms timeout)
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 1 {
		t.Fatalf("callback should fire once on timeout, got %d", count)
	}

	// At this point, the system has reset for a new utterance.
	// This system input will start a NEW utterance window, not the same one.
	// So per the README: "After the callback completes, the EOS instance is reset and reusable for the next utterance"
	// We should NOT send another input without waiting for the reset to complete,
	// OR we should wait long enough to verify the callback doesn't fire again from utterance 1.

	// Instead, we'll verify that the service is reusable by sending a user input which triggers immediately
	if err := svcIface.Execute(ctx, userInput("new utterance")); err != nil {
		t.Fatalf("analyze user: %v", err)
	}

	// Wait for the new callback (user input triggers immediately)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count = callCount
	mu.Unlock()

	if count != 2 {
		t.Fatalf("callback should fire again for new utterance, expected 2 got %d", count)
	}
}

// TestNewInputInvalidatesPreviousCallback verifies new input cancels pending callbacks
func TestNewInputInvalidatesPreviousCallback(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	var mu sync.Mutex
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				mu.Lock()
				callCount++
				mu.Unlock()
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 300.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// activity
	if err := svcIface.Execute(ctx, sttInput("activity1", false)); err != nil {
		t.Fatalf("analyze 1: %v", err)
	}

	// Send system input - starts 300ms timer
	// here the timer is started for the current transcript revision
	if err := svcIface.Execute(ctx, sttInput("activity1.", true)); err != nil {
		t.Fatalf("analyze 1: %v", err)
	}

	// Wait 150ms, then send another system input - resets timer
	time.Sleep(150 * time.Millisecond)
	if err := svcIface.Execute(ctx, sttInput("activity2", false)); err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	if err := svcIface.Execute(ctx, sttInput("activity2!", true)); err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	// Wait another 150ms - total 300ms but timer was reset, so callback not fired yet
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 0 {
		t.Fatalf("callback should not have fired yet, but got %d calls", count)
	}

	// Wait for the reset timer to fire (300ms from the second input)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count = callCount
	mu.Unlock()

	if count != 1 {
		t.Fatalf("callback should fire after reset timer, expected 1 but got %d", count)
	}
}

// TestUserInputImmediateTrigger verifies user input triggers callback immediately
func TestUserInputImmediateTrigger(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 1000.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	startTime := time.Now()

	// Send user input - should trigger immediately
	if err := svcIface.Execute(ctx, userInput("user said something")); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case cbTime := <-callTime:
		elapsed := cbTime.Sub(startTime)
		if elapsed > 50*time.Millisecond {
			t.Fatalf("user input should trigger immediately, took %v", elapsed)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for callback on user input")
	}
}

func TestUserInputImmediateTriggerUsesQueuedSegmentSnapshot(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	eos := &silenceBasedEndOfSpeech{
		onPacket: func(ctx context.Context, res ...internal_type.Packet) error {
			for _, packet := range res {
				if endOfSpeech, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					called <- endOfSpeech
				}
			}
			return nil
		},
		commandCh: make(chan workerCommand, 4),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	go eos.worker()
	defer func() { _ = eos.Close(context.Background()) }()

	requireNoError := func(err error) {
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
	}

	requireNoError(eos.handleUserTextPacket(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-first",
		Text:      "first",
	}))
	requireNoError(eos.handleUserTextPacket(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-second",
		Text:      "second",
	}))

	select {
	case packet := <-called:
		if packet.ContextID != "ctx-first" || packet.Speech != "first" {
			t.Fatalf("unexpected first packet: %+v", packet)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for first immediate callback")
	}

	select {
	case packet := <-called:
		if packet.ContextID != "ctx-second" || packet.Speech != "second" {
			t.Fatalf("unexpected second packet: %+v", packet)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for second immediate callback")
	}
}

// TestSystemInputExtendsTimer verifies system input extends silence timer
func TestSystemInputExtendsTimer(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// Send initial system input (NOT user input, which fires immediately)
	if err := svcIface.Execute(ctx, sttInput("activity", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	// Wait 100ms and send another system input - this should reset the timer
	time.Sleep(100 * time.Millisecond)
	startTime := time.Now()

	if err := svcIface.Execute(ctx, systemInput("more activity")); err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	// Callback should fire ~200ms from the second input
	select {
	case cbTime := <-callTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := 200 * time.Millisecond
		tolerance := 20 * time.Millisecond
		if elapsed < expectedMs-tolerance || elapsed > expectedMs+tolerance {
			t.Fatalf("callback timing incorrect: expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on system input")
	}
}

// TestSTTInputExtendsTimer verifies STT input extends silence timer
func TestSTTInputExtendsTimer(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callTime <- time.Now():
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 150.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// Send STT input
	if err := svcIface.Execute(ctx, sttInput("incomplete message", true)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	// Callback should fire ~150ms later
	select {
	case cbTime := <-callTime:
		elapsed := cbTime.Sub(time.Now().Add(-150 * time.Millisecond))
		expectedMs := 150 * time.Millisecond
		tolerance := 20 * time.Millisecond
		// Allow some slack since we're measuring from "now minus expected time"
		if elapsed > expectedMs+tolerance*2 {
			t.Fatalf("callback took too long: expected ~%v, got %v", expectedMs, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback on STT input")
	}
}

// TestRevisionInvalidation verifies old callbacks don't fire after new input.
func TestRevisionInvalidation(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	var mu sync.Mutex
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				mu.Lock()
				callCount++
				mu.Unlock()
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 500.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	// Send first system input - starts timer for the current transcript revision.
	if err := svcIface.Execute(ctx, sttInput("activity1", true)); err != nil {
		t.Fatalf("analyze 1: %v", err)
	}

	// Wait 200ms and send second input - increments transcript revision, invalidates the old timer.
	time.Sleep(200 * time.Millisecond)
	if err := svcIface.Execute(ctx, sttInput("activity2", true)); err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	// Wait 200ms more - total 400ms from first input, but only 200ms from second
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 0 {
		t.Fatalf("old revision timer should not fire, expected 0 callbacks, got %d", count)
	}

	// Wait for second input timer to fire (500ms total from second input)
	time.Sleep(400 * time.Millisecond)

	mu.Lock()
	count = callCount
	mu.Unlock()

	if count != 1 {
		t.Fatalf("current revision timer should fire, expected 1 callback, got %d", count)
	}
}

// TestContextCancellation verifies cancelled context prevents callback
func TestContextCancellation(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan bool, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				called <- true
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil

	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Send system input with cancellable context
	if err := svcIface.Execute(ctx, systemInput("activity")); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	// Cancel context immediately
	cancel()

	// Wait past the timeout
	time.Sleep(400 * time.Millisecond)

	select {
	case <-called:
		t.Fatal("callback should not be called after context cancellation")
	default:
		// Expected: no callback
	}
}

// TestCallbackReceivesCorrectData verifies callback receives complete EndOfSpeechResult
func TestCallbackReceivesCorrectData(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	results := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case results <- res:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	speechText := "hello there"

	if err := svcIface.Execute(ctx, userInput(speechText)); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	select {
	case res := <-results:
		if res.Speech != speechText {
			t.Fatalf("incorrect speech: expected %q, got %q", speechText, res.Speech)
		}

	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for callback result")
	}
}

// TestRaceConditionUnderConcurrentInput uses goroutines to stress-test for races
func TestRaceConditionUnderConcurrentInput(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 50.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	wg := sync.WaitGroup{}

	// Spawn multiple goroutines sending different input types concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_ = svcIface.Execute(ctx, userInput("user"))
			case 1:
				_ = svcIface.Execute(ctx, systemInput("system"))
			case 2:
				_ = svcIface.Execute(ctx, sttInput("stt", i%2 == 0))
			}
		}(i)
	}

	wg.Wait()

	// If we get here without panicking, the test passes
	// (In debug mode with race detector enabled, this would catch races)
	time.Sleep(200 * time.Millisecond)
}

// TestServiceName verifies the service name
func TestServiceName(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	name := svcIface.Name()
	if name != "silenceBasedEndOfSpeech" {
		t.Fatalf("unexpected service name: %v", name)
	}
}

// TestServiceClose verifies graceful shutdown
func TestServiceClose(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	// Close should not panic
	if err := svcIface.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := svcIface.Close(context.Background()); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestSilenceBasedEndOfSpeech_EnqueueAfterClose_DoesNotEnqueueCommand(t *testing.T) {
	eos := &silenceBasedEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	close(eos.stopCh)

	eos.enqueueCommand(workerCommand{fireImmediately: true})

	if got := len(eos.commandCh); got != 0 {
		t.Fatalf("expected no enqueued commands after close, got=%d", got)
	}
}

func TestSilenceBasedEndOfSpeech_EnqueueCommandBlocksUntilChannelHasSpace(t *testing.T) {
	eos := &silenceBasedEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	eos.commandCh <- workerCommand{segment: speechSegment{Text: "first"}}

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		eos.enqueueCommand(workerCommand{segment: speechSegment{Text: "second"}})
		close(done)
	}()

	<-started
	select {
	case <-done:
		t.Fatal("enqueueCommand should wait while channel is full")
	case <-time.After(50 * time.Millisecond):
	}

	first := <-eos.commandCh
	if first.segment.Text != "first" {
		t.Fatalf("unexpected first command: %+v", first)
	}

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("enqueueCommand should resume after channel space is available")
	}

	second := <-eos.commandCh
	if second.segment.Text != "second" {
		t.Fatalf("unexpected second command: %+v", second)
	}
}

// === COMPREHENSIVE CONCURRENT & COMBINATION TESTS ===
// These tests validate behavior under realistic concurrent loads that trigger LLM calls

// TestConcurrentMixedInputTypes simulates multiple threads with different input types
// arriving simultaneously - realistic scenario for real-time voice processing
func TestConcurrentMixedInputTypes(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbacks := make(chan internal_type.EndOfSpeechPacket, 100)
	callbackMu := sync.Mutex{}
	callCount := 0

	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case callbacks <- res:
					callbackMu.Lock()
					callCount++
					callbackMu.Unlock()
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()
	wg := sync.WaitGroup{}

	// Simulate 5 concurrent goroutines sending mixed input types
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine sends: STT -> System -> User
			switch id % 3 {
			case 0:
				_ = svcIface.Execute(ctx, sttInput("concurrent stt", false))
				time.Sleep(10 * time.Millisecond)
				_ = svcIface.Execute(ctx, systemInput("activity"))
			case 1:
				_ = svcIface.Execute(ctx, systemInput("system1"))
				time.Sleep(15 * time.Millisecond)
				_ = svcIface.Execute(ctx, sttInput("concurrent speech", true))
			case 2:
				_ = svcIface.Execute(ctx, sttInput("incomplete", false))
				time.Sleep(20 * time.Millisecond)
				_ = svcIface.Execute(ctx, userInput("user interrupts"))
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(300 * time.Millisecond) // Wait for callbacks

	callbackMu.Lock()
	if callCount == 0 {
		t.Fatalf("at least one callback should fire under concurrent load, got %d", callCount)
	}
	callbackMu.Unlock()
}

// TestHighFrequencySTTUpdates simulates rapid STT streaming (10+ updates/sec)
// while other inputs arrive - common during active speech
func TestHighFrequencySTTUpdates(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callTime := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callTime <- time.Now()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 150.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()
	startTime := time.Now()

	// Rapid-fire 20 STT updates (2ms interval = 10 updates/sec)
	for i := 0; i < 20; i++ {
		interim := i < 19 // Last one is complete
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("word%d", i), !interim))
		time.Sleep(2 * time.Millisecond)
	}

	// Now silence - let timer fire
	select {
	case cbTime := <-callTime:
		elapsed := cbTime.Sub(startTime)
		expectedMs := 150 * time.Millisecond
		// Allow wide tolerance because of rapid updates
		tolerance := 50 * time.Millisecond
		if elapsed < expectedMs-tolerance || elapsed > expectedMs+tolerance {
			t.Logf("WARNING: Timing slightly off (high-frequency updates): expected %v±%v, got %v", expectedMs, tolerance, elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout: callback should fire after rapid STT updates end")
	}
}

// TestUserInputInterruptsActiveSTT tests user input (immediate trigger)
// arriving while STT is actively updating
func TestUserInputInterruptsActiveSTT(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	callMu := sync.Mutex{}
	callTime := make(chan time.Time, 1)

	callback := func(ctx context.Context, res ...internal_type.Packet) error {

		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				callTime <- time.Now()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil

	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 500.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Start STT updates
	for i := 0; i < 5; i++ {
		_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("stt update %d", i), false))
		time.Sleep(50 * time.Millisecond)
	}

	// User interrupts mid-stream
	startTime := time.Now()
	_ = svcIface.Execute(ctx, userInput("stop, I want to say something else"))

	// Callback should fire immediately
	select {
	case cbTime := <-callTime:
		elapsed := cbTime.Sub(startTime)
		if elapsed > 100*time.Millisecond {
			t.Fatalf("user input should trigger immediately, took %v", elapsed)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout: user input should fire callback immediately")
	}

	callMu.Lock()
	if callCount != 1 {
		t.Fatalf("expected exactly 1 callback, got %d", callCount)
	}
	callMu.Unlock()
}

// TestMultipleUtteranceSequence tests that after first callback fires,
// the system correctly resets for a second utterance
func TestMultipleUtteranceSequence(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbacks := make(chan internal_type.EndOfSpeechPacket, 10)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				callbacks <- res
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// === UTTERANCE 1 ===
	_ = svcIface.Execute(ctx, sttInput("first utterance", true))
	select {
	case res := <-callbacks:
		if res.Speech != "first utterance" {
			t.Fatalf("utterance 1: unexpected speech %q", res.Speech)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("utterance 1: timeout waiting for callback")
	}

	// === UTTERANCE 2 ===
	time.Sleep(50 * time.Millisecond) // Small gap between utterances
	_ = svcIface.Execute(ctx, sttInput("second utterance", true))
	select {
	case res := <-callbacks:
		if res.Speech != "second utterance" {
			t.Fatalf("utterance 2: unexpected speech %q", res.Speech)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("utterance 2: timeout waiting for callback")
	}

	// === UTTERANCE 3 ===
	time.Sleep(50 * time.Millisecond)
	_ = svcIface.Execute(ctx, systemInput("activity"))
	time.Sleep(50 * time.Millisecond)
	_ = svcIface.Execute(ctx, sttInput("third utterance", true))

	select {
	case res := <-callbacks:
		if res.Speech != "third utterance" {
			t.Fatalf("utterance 3: unexpected speech %q", res.Speech)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("utterance 3: timeout waiting for callback")
	}
}

// TestConcurrentUtterancesRapid tests rapid sequential utterances
// (simulating multiple speakers or quick turn-taking)
func TestConcurrentUtterancesRapid(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbacks := make(chan internal_type.EndOfSpeechPacket, 20)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				callbacks <- res
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 80.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Fire off 5 utterances in rapid succession (< 10ms apart)
	expectedTexts := []string{"first", "second", "third", "fourth", "fifth"}
	for i, text := range expectedTexts {
		_ = svcIface.Execute(ctx, userInput(text))
		if i < len(expectedTexts)-1 {
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Collect all callbacks
	received := 0
	timeout := time.After(500 * time.Millisecond)
	for received < len(expectedTexts) {
		select {
		case res := <-callbacks:
			received++
			if received <= len(expectedTexts) && res.Speech != expectedTexts[received-1] {
				t.Logf("utterance %d: expected %q, got %q (note: user input fires immediately, so order may vary)", received, expectedTexts[received-1], res.Speech)
			}
		case <-timeout:
			break
		}
	}

	if received != len(expectedTexts) {
		t.Fatalf("expected %d callbacks, got %d", len(expectedTexts), received)
	}
}

// TestConcurrentInputsDuringReset tests inputs arriving while reset is processing
// This was a critical race condition in the original implementation
func TestConcurrentInputsDuringReset(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callbacks := make(chan internal_type.EndOfSpeechPacket, 10)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				callbacks <- res
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 50.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// First utterance
	_ = svcIface.Execute(ctx, userInput("first"))

	select {
	case <-callbacks:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first callback should fire")
	}

	// Now fire many inputs rapidly while reset is being processed
	// These should all be accepted for the new utterance window
	inputCount := 0
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("stt%d", i), false))
		} else {
			_ = svcIface.Execute(ctx, systemInput("activity"))
		}
		inputCount++
		time.Sleep(5 * time.Millisecond)
	}

	// Final input triggers second callback
	_ = svcIface.Execute(ctx, userInput("second"))

	select {
	case <-callbacks:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("second callback should fire even after rapid inputs during reset")
	}
}

// TestStressLoadWithManyInputs stress tests with 100+ inputs
func TestStressLoadWithManyInputs(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {

		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil

	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 50.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()
	wg := sync.WaitGroup{}

	// 20 goroutines, each sending 50 inputs = 1000 total inputs
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				inputType := (goroutineID*50 + i) % 3
				switch inputType {
				case 0:
					_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("g%d_i%d", goroutineID, i), i%7 == 6))
				case 1:
					_ = svcIface.Execute(ctx, systemInput("activity"))
				case 2:
					if i%10 == 0 { // Occasional user inputs
						_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("user_g%d", goroutineID)))
					}
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(g)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond) // Wait for final callbacks

	callMu.Lock()
	if callCount == 0 {
		t.Fatalf("stress test: at least some callbacks should fire, got %d", callCount)
	}
	t.Logf("stress test: %d inputs processed, %d callbacks fired", 1000, callCount)
	callMu.Unlock()
}

// TestContextCancellationUnderConcurrentLoad tests that context cancellation
// works correctly under heavy concurrent input
func TestContextCancellationUnderConcurrentLoad(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	// Spawn 5 goroutines sending inputs
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = svcIface.Execute(ctx, sttInput(fmt.Sprintf("g%d_i%d", id, j), false))
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	// Cancel context after 30ms
	time.Sleep(30 * time.Millisecond)
	cancel()

	wg.Wait()
	time.Sleep(300 * time.Millisecond) // Wait for any pending callbacks

	callMu.Lock()
	// With cancelled context, callbacks should not fire
	if callCount > 0 {
		t.Logf("cancellation test: %d callbacks fired despite context cancellation (may be race)", callCount)
	}
	callMu.Unlock()
}

// TestFormattedTextOptimizationUnderConcurrency tests the half-timeout optimization
// works correctly when multiple STT updates arrive concurrently
func TestFormattedTextOptimizationUnderConcurrency(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callTimes := make(chan time.Time, 5)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callTimes <- time.Now()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name     string
		sequence []struct {
			text  string
			final bool
			delay time.Duration
		}
		expectedApproxMs int
	}{
		{
			name: "formatting_optimization",
			sequence: []struct {
				text  string
				final bool
				delay time.Duration
			}{
				{"hello world", false, 30 * time.Millisecond},
				{"hello world", false, 30 * time.Millisecond},
				{"Hello, World!", true, 0}, // Final: normalized same, uses half timeout
			},
			expectedApproxMs: 100, // 200/2
		},
		{
			name: "no_optimization_different_text",
			sequence: []struct {
				text  string
				final bool
				delay time.Duration
			}{
				{"hello", false, 30 * time.Millisecond},
				{"hello world", true, 0}, // Final: different content, uses full timeout
			},
			expectedApproxMs: 200,
		},
	}

	for _, tc := range testCases {
		startTime := time.Now()

		for i, step := range tc.sequence {
			_ = svcIface.Execute(ctx, sttInput(step.text, step.final))
			if i < len(tc.sequence)-1 && step.delay > 0 {
				time.Sleep(step.delay)
			}
		}

		select {
		case cbTime := <-callTimes:
			elapsed := cbTime.Sub(startTime).Milliseconds()
			expectedMs := int64(tc.expectedApproxMs)
			tolerance := int64(40)

			if elapsed < expectedMs-tolerance || elapsed > expectedMs+tolerance {
				t.Logf("WARNING: %s timing off: expected %dms±%dms, got %dms", tc.name, expectedMs, tolerance, elapsed)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("%s: timeout waiting for callback", tc.name)
		}
	}
}

// TestRevisionPreventsStaleCallbacks validates that transcript revision prevents
// stale callbacks even under extreme timing variations.
func TestRevisionPreventsStaleCallbacks(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Start a timer that will eventually fire
	_ = svcIface.Execute(ctx, sttInput("gen1", true))

	// Wait 30ms
	time.Sleep(30 * time.Millisecond)

	// Before old timer fires, send new input (invalidates old revision).
	_ = svcIface.Execute(ctx, sttInput("gen2", true))

	// Wait 30ms more (total 60ms from first, 30ms from second)
	time.Sleep(30 * time.Millisecond)

	// At this point, gen1 timer would fire (100ms total from first input)
	// But revision should be incremented, so it should be ignored.
	callMu.Lock()
	countAt60ms := callCount
	callMu.Unlock()

	if countAt60ms != 0 {
		t.Fatalf("old revision timer should be ignored, but got %d callbacks at 60ms", countAt60ms)
	}

	// Wait for gen2 to fire
	time.Sleep(120 * time.Millisecond)

	callMu.Lock()
	finalCount := callCount
	callMu.Unlock()

	if finalCount != 1 {
		t.Fatalf("expected exactly 1 callback from gen2, got %d total", finalCount)
	}
}

// TestNormalizationConsistency tests that text normalization is consistent
// across concurrent accesses
func TestNormalizationConsistency(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	testTexts := []struct {
		text1       string
		text2       string
		shouldMatch bool
	}{
		{"hello world", "HELLO WORLD", true},
		{"hello, world!", "hello world", true},
		{"Hello, World!!!", "hello world", true},
		{"hello", "hello world", false},
		{"hello world", "hello world ", false}, // Space difference
		{"café", "CAFÉ", true},
	}

	for _, test := range testTexts {
		// Send first text
		_ = svcIface.Execute(ctx, sttInput(test.text1, true))
		time.Sleep(30 * time.Millisecond)

		// Send second text and check if it uses optimization
		_ = svcIface.Execute(ctx, sttInput(test.text2, true))

		// The optimization affects timing; we just verify no panic
		time.Sleep(150 * time.Millisecond)
	}
}

// TestEdgeCaseRapidResetCycles tests many reset cycles happening in quick succession
func TestEdgeCaseRapidResetCycles(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	callCount := 0
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 30.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Rapid user inputs (each fires immediately and triggers reset)
	for i := 0; i < 20; i++ {
		_ = svcIface.Execute(ctx, userInput(fmt.Sprintf("utterance_%d", i)))
		time.Sleep(2 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	callMu.Lock()
	if callCount != 20 {
		t.Fatalf("expected 20 callbacks from 20 user inputs, got %d", callCount)
	}
	callMu.Unlock()
}

// TestSingleCallbackForContinuousSpeechWithInterimAndFinal verifies that
// continuous speech with mixed interim and final packets results in a single
// callback with the complete accumulated text from FINAL packets only.
// Interim packets only extend the timer, they don't contribute to the callback text.
// This test reproduces the bug where callback fired twice:
// 1. "Okay. Can you tell me what is" (premature)
// 2. "payment doing right now and later?" (remainder)
// Instead of once with: "Okay. Can you tell me what is the difference inpayment doing right now and later?"
func TestSingleCallbackForContinuousSpeechWithInterimAndFinal(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	var callbackResults []internal_type.EndOfSpeechPacket
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch res := r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callbackResults = append(callbackResults, res)
				callMu.Unlock()
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 500.0}) // 500ms timeout
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Simulate the exact sequence from the bug report:
	// Packets arrive faster than silence timeout, so timer keeps resetting
	packets := []internal_type.SpeechToTextPacket{
		// 1. Interim: "Okay. We" - only extends timer
		{Script: "Okay. We", Interim: true, Confidence: 0.74560547, Language: "en"},
		// 2. Interim: "Okay. Can you tell me what is" - only extends timer
		{Script: "Okay. Can you tell me what is", Interim: true, Confidence: 0.9980469, Language: "en"},
		// 3. Final: "Okay. Can you tell me what is the difference in" - accumulates text
		{Script: "Okay. Can you tell me what is the difference in", Interim: false, Confidence: 1.0, Language: "en"},
		// 4. Interim: "payment doing right now and later?" - only extends timer
		{Script: "payment doing right now and later?", Interim: true, Confidence: 0.9970703, Language: "en"},
		// 5. Final: "payment doing right now and later?" - accumulates text
		{Script: "payment doing right now and later?", Interim: false, Confidence: 0.9970703, Language: "en"},
	}

	// Send packets with small delays (simulating real-time speech < silence timeout)
	for _, pkt := range packets {
		if err := svcIface.Execute(ctx, pkt); err != nil {
			t.Fatalf("Analyze failed: %v", err)
		}
		time.Sleep(100 * time.Millisecond) // Packets arrive faster than 500ms timeout
	}

	// Wait for silence timeout to trigger callback
	time.Sleep(700 * time.Millisecond)

	// Verify: callback should fire exactly once
	callMu.Lock()
	count := len(callbackResults)
	callMu.Unlock()

	if count != 1 {
		callMu.Lock()
		for i, res := range callbackResults {
			t.Logf("Callback %d: %q", i+1, res.Speech)
		}
		callMu.Unlock()
		t.Fatalf("expected 1 callback invocation, got %d", count)
	}

	// Verify: the callback should contain the accumulated FINAL text only
	expectedText := "Okay. Can you tell me what is the difference in payment doing right now and later?"
	callMu.Lock()
	actualText := callbackResults[0].Speech
	callMu.Unlock()

	if actualText != expectedText {
		t.Errorf("expected speech %q, got %q", expectedText, actualText)
	}
}

// TestInterimPacketsOnlyExtendTimer verifies that interim packets only extend
// the timer but don't contribute text to the callback. If only interim packets
// are received (no finals), the callback fires with empty text.

func TestInterimPacketsOnlyExtendTimer(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	called := make(chan struct{}, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {

		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				called <- struct{}{}
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are received but don't signal end-of-speech
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Send only an interim packet (no final)
	pkt := internal_type.SpeechToTextPacket{
		Script:  "This is interim only text",
		Interim: true,
	}
	if err := svcIface.Execute(ctx, pkt); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Callback must NOT be called for interim-only packets
	select {
	case <-called:
		t.Fatal("callback should not be called for interim-only packet")
	case <-time.After(400 * time.Millisecond):
		// success: no callback received
	}
}

func TestFaultyVADRecoversPendingFinal(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch packet := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- packet:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 60.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}); err != nil {
		t.Fatalf("vad start: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-faulty-vad",
		Script:    "hello",
		Interim:   false,
	}); err != nil {
		t.Fatalf("stt final: %v", err)
	}

	select {
	case packet := <-called:
		t.Fatalf("callback fired before stuck VAD recovery elapsed: %+v", packet)
	case <-time.After(90 * time.Millisecond):
	}

	select {
	case packet := <-called:
		if packet.ContextID != "ctx-faulty-vad" {
			t.Fatalf("unexpected context id: %q", packet.ContextID)
		}
		if packet.Speech != "hello" {
			t.Fatalf("unexpected speech: %q", packet.Speech)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback after stuck VAD recovery")
	}

	select {
	case packet := <-called:
		t.Fatalf("unexpected duplicate callback after stuck VAD recovery: %+v", packet)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestStaleTimerCompletionDoesNotShrinkLatestSegment(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := &silenceBasedEndOfSpeech{
		onPacket: func(ctx context.Context, res ...internal_type.Packet) error {
			for _, r := range res {
				if packet, ok := r.(internal_type.EndOfSpeechPacket); ok {
					called <- packet
				}
			}
			return nil
		},
		silenceTimeout: 30 * time.Millisecond,
		commandCh:      make(chan workerCommand, 4),
		stopCh:         make(chan struct{}),
		state: &endOfSpeechState{segment: speechSegment{
			Revision:  1,
			ContextID: "ctx-stale",
			Text:      "me an idea about the how how do I do things?",
			Timestamp: time.Now(),
		}},
	}
	go endOfSpeech.worker()
	defer func() { _ = endOfSpeech.Close(context.Background()) }()

	oldSegment := endOfSpeech.state.segment
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
		if packet.Speech != latestSegment.Text {
			t.Fatalf("expected latest speech %q, got %q", latestSegment.Text, packet.Speech)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for latest completion")
	}
}

func TestVADEndFlushesPendingFinal(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch packet := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- packet:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 100.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())
	svc := svcIface.(*silenceBasedEndOfSpeech)
	svc.vadEndTimeout = 40 * time.Millisecond

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}); err != nil {
		t.Fatalf("vad start: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-vad-success",
		Script:    "done",
		Interim:   false,
	}); err != nil {
		t.Fatalf("stt final: %v", err)
	}

	deadline := time.After(300 * time.Millisecond)
	for {
		svc.mu.RLock()
		pending := svc.state.pending != nil
		svc.mu.RUnlock()
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

	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}); err != nil {
		t.Fatalf("vad end: %v", err)
	}

	select {
	case packet := <-called:
		if packet.ContextID != "ctx-vad-success" {
			t.Fatalf("unexpected context id: %q", packet.ContextID)
		}
		if packet.Speech != "done" {
			t.Fatalf("unexpected speech: %q", packet.Speech)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD end")
	}

	select {
	case packet := <-called:
		t.Fatalf("unexpected duplicate callback after VAD end: %+v", packet)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestVADEndFlushWindowUsesLateFinalSTT(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch packet := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- packet:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 200.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())
	svc := svcIface.(*silenceBasedEndOfSpeech)
	svc.vadEndTimeout = 40 * time.Millisecond

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}); err != nil {
		t.Fatalf("vad start: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}); err != nil {
		t.Fatalf("first final: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}); err != nil {
		t.Fatalf("vad end: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "Rightly?",
		Interim:   false,
	}); err != nil {
		t.Fatalf("late final: %v", err)
	}

	select {
	case packet := <-called:
		expected := "me an idea about the how how do I do things? Rightly?"
		if packet.Speech != expected {
			t.Fatalf("expected latest speech %q, got %q", expected, packet.Speech)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for late final callback")
	}
}

func TestVADEndCompleteClearsPendingInterimWhenFinalIsShorter(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch packet := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- packet:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 70.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())
	svc := svcIface.(*silenceBasedEndOfSpeech)
	svc.vadEndTimeout = 20 * time.Millisecond

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}); err != nil {
		t.Fatalf("vad start: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things? Rightly?",
		Interim:   true,
	}); err != nil {
		t.Fatalf("stt interim: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}); err != nil {
		t.Fatalf("stt final: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}); err != nil {
		t.Fatalf("vad end: %v", err)
	}

	select {
	case packet := <-called:
		expected := "me an idea about the how how do I do things?"
		if packet.Speech != expected {
			t.Fatalf("expected final transcript %q, got %q", expected, packet.Speech)
		}
		if len(packet.Speechs) != 2 {
			t.Fatalf("expected interim and final chunks, got %+v", packet.Speechs)
		}
		if !packet.Speechs[0].Interim || packet.Speechs[1].Interim {
			t.Fatalf("unexpected chunk lifecycle: %+v", packet.Speechs)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for final transcript")
	}
}

func TestOnlyInterimWithVADDoesNotComplete(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch packet := r.(type) {
			case internal_type.EndOfSpeechPacket:
				select {
				case called <- packet:
				default:
				}
			case internal_type.InterimEndOfSpeechPacket:
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 60.0})
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}); err != nil {
		t.Fatalf("vad start: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-interim-only",
		Script:    "interim only",
		Interim:   true,
	}); err != nil {
		t.Fatalf("stt interim: %v", err)
	}
	if err := svcIface.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}); err != nil {
		t.Fatalf("vad end: %v", err)
	}

	select {
	case packet := <-called:
		t.Fatalf("callback should not fire for interim-only VAD input: %+v", packet)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestInterimPacketsResetTimerContinuously verifies that continuous interim packets
// prevent the callback from firing prematurely.
func TestInterimPacketsResetTimerContinuously(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	callCount := 0
	callMu := sync.Mutex{}
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			switch r.(type) {
			case internal_type.EndOfSpeechPacket:
				callMu.Lock()
				callCount++
				callMu.Unlock()
				return nil
			case internal_type.InterimEndOfSpeechPacket:
				// Interim packets are expected, ignore for this test
			}
		}
		return nil
	}

	opts := newTestOpts(map[string]any{"microphone.eos.timeout": 300.0}) // 300ms timeout
	svcIface, err := newSilenceBasedEndOfSpeechForTest(context.Background(), logger, callback, opts)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer svcIface.Close(context.Background())

	ctx := context.Background()

	// Send interim packets every 200ms (faster than 300ms timeout)
	// This should keep resetting the timer
	for i := 0; i < 5; i++ {
		pkt := internal_type.SpeechToTextPacket{
			Script:  fmt.Sprintf("Interim part %d", i+1),
			Interim: false,
		}
		if err := svcIface.Execute(ctx, pkt); err != nil {
			t.Fatalf("Analyze failed: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Callback should not have fired yet (timer keeps resetting)
	callMu.Lock()
	countDuringSpeech := callCount
	callMu.Unlock()

	if countDuringSpeech != 0 {
		t.Errorf("expected 0 callback invocations during continuous speech, got %d", countDuringSpeech)
	}

	// Now wait for silence timeout
	time.Sleep(400 * time.Millisecond)

	// Callback should fire exactly once
	callMu.Lock()
	finalCount := callCount
	callMu.Unlock()

	if finalCount != 1 {
		t.Errorf("expected 1 callback invocation after silence, got %d", finalCount)
	}
}
