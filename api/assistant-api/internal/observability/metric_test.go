// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"testing"
	"time"

	"github.com/rapidaai/protos"
)

func TestMetricNames_MirrorCurrentImplementation(t *testing.T) {
	tests := []struct {
		actual   string
		expected string
	}{
		{MetricConversationStatus, "status"},
		{MetricConversationDuration, "conversation.duration_ms"},
		{MetricConversationSTTDuration, "stt.duration_ms"},
		{MetricConversationTTSDuration, "tts.duration_ms"},
		{MetricCallDurationMs, "call.duration_ms"},
		{MetricCallStatus, "call.status"},
		{MetricCallPrice, "call.price"},
		{MetricSIPRegistrationStatus, "sip.registration.status"},
		{MetricCallTransferDurationMs, "call.transfer.bridge_duration_ms"},
		{MetricICELatencyMs, "webrtc.ice_latency_ms"},
		{MetricWebRTCOutputQueueDrops, "webrtc.output_queue_dropped_frames"},
		{MetricCallStatusComplete, "COMPLETE"},
		{MetricCallStatusFailed, "FAILED"},
		{MetricCallStatusInProgress, "INPROGRESS"},
		{MetricCallStatusRinging, "RINGING"},
		{MetricCallStatusCancelled, "CANCELLED"},
		{MetricUserTurn, "user_turn"},
		{MetricAssistantTurn, "assistant_turn"},
		{MetricSTTInitLatencyMs, "stt_init_ms"},
		{MetricSTTLatencyMs, "stt_latency_ms"},
		{MetricTTSInitLatencyMs, "tts_init_ms"},
		{MetricTTSLatencyMs, "tts_latency_ms"},
		{MetricVADInitLatencyMs, "vad.init_ms"},
		{MetricEOSInitLatencyMs, "eos.init_ms"},
		{MetricEOSLatencyMs, "eos_latency_ms"},
		{MetricEOSTextToTriggerMs, "eos_text_to_trigger_ms"},
		{MetricEOSWordCount, "eos_word_count"},
		{MetricEOSCharCount, "eos_char_count"},
		{MetricEOSConfidence, "eos_confidence"},
		{MetricDenoiseInitLatencyMs, "denoise.init_ms"},
		{MetricLLMInitLatencyMs, "agent.init_ms"},
		{MetricLLMLatencyMs, "llm_latency_ms"},
		{MetricAgentTTFTMs, "agent.ttft_ms"},
		{MetricAgentTRTMs, "agent.trt_ms"},
		{MetricStorageInitLatencyMs, "storage.init_ms"},
		{MetricAnalysisInitLatencyMs, "analysis.init_ms"},
		{MetricAuthenticationInitLatencyMs, "authentication.init_ms"},
		{MetricAuthenticationLatencyMs, "authentication.latency_ms"},
		{MetricRecordingInitLatencyMs, "recording.init_ms"},
		{MetricKnowledgeLatencyMs, "knowledge_latency_ms"},
		{MetricLLMError, "llm_error"},
		{MetricSTTError, "stt_error"},
		{MetricTTSError, "tts_error"},
		{MetricDiscardedTTSChunk, "discarded_tts_chunk"},
		{MetricDiscardedTTS, "discarded_tts"},
		{MetricTimeTaken, "time_taken"},
		{MetricStatus, "status"},
		{MetricInputToken, "input_token"},
		{MetricOutputToken, "output_token"},
		{MetricTotalToken, "total_token"},
		{MetricCachedContentToken, "cached_content_token"},
		{MetricCost, "cost"},
		{MetricInputCost, "input_cost"},
		{MetricOutputCost, "output_cost"},
		{MetricLLMRequestID, "llm_request_id"},
		{MetricTokenPerSecond, "token_pre_second"},
		{MetricTimeToFirstToken, "time_to_first_token"},
		{MetricProviderTotalTime, "provider_total_time"},
		{MetricProviderGenerateTime, "provider_generate_time"},
	}

	for _, test := range tests {
		if test.actual != test.expected {
			t.Fatalf("expected metric name %q, got %q", test.expected, test.actual)
		}
	}
}

func TestNewMetricSTTInitLatencyMs(t *testing.T) {
	record := NewMetricSTTInitLatencyMs(123*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricSTTInitLatencyMs {
		t.Fatalf("expected metric name %q, got %q", MetricSTTInitLatencyMs, metric.Name)
	}
	if metric.Value != "123" {
		t.Fatalf("expected metric value %q, got %q", "123", metric.Value)
	}
	if metric.Description != "STT initialization latency in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "STT initialization latency in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func TestNewMetricSTTLatencyMs(t *testing.T) {
	record := NewMetricSTTLatencyMs(456*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricSTTLatencyMs {
		t.Fatalf("expected metric name %q, got %q", MetricSTTLatencyMs, metric.Name)
	}
	if metric.Value != "456" {
		t.Fatalf("expected metric value %q, got %q", "456", metric.Value)
	}
	if metric.Description != "STT latency from speech end to final transcript in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "STT latency from speech end to final transcript in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func TestNewMetricTTSInitLatencyMs(t *testing.T) {
	record := NewMetricTTSInitLatencyMs(123*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricTTSInitLatencyMs {
		t.Fatalf("expected metric name %q, got %q", MetricTTSInitLatencyMs, metric.Name)
	}
	if metric.Value != "123" {
		t.Fatalf("expected metric value %q, got %q", "123", metric.Value)
	}
	if metric.Description != "TTS initialization latency in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "TTS initialization latency in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func TestNewMetricTTSLatencyMs(t *testing.T) {
	record := NewMetricTTSLatencyMs(456*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricTTSLatencyMs {
		t.Fatalf("expected metric name %q, got %q", MetricTTSLatencyMs, metric.Name)
	}
	if metric.Value != "456" {
		t.Fatalf("expected metric value %q, got %q", "456", metric.Value)
	}
	if metric.Description != "TTS latency from text input to first audio in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "TTS latency from text input to first audio in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func TestNewMetricSTTDuration(t *testing.T) {
	record := NewMetricSTTDuration(789*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricConversationSTTDuration {
		t.Fatalf("expected metric name %q, got %q", MetricConversationSTTDuration, metric.Name)
	}
	if metric.Value != "789" {
		t.Fatalf("expected metric value %q, got %q", "789", metric.Value)
	}
	if metric.Description != "Total STT connection duration in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "Total STT connection duration in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func TestNewMetricTTSDuration(t *testing.T) {
	record := NewMetricTTSDuration(987*time.Millisecond, Attributes{"provider": "deepgram"})
	metric := singleMetric(t, record)

	if metric.Name != MetricConversationTTSDuration {
		t.Fatalf("expected metric name %q, got %q", MetricConversationTTSDuration, metric.Name)
	}
	if metric.Value != "987" {
		t.Fatalf("expected metric value %q, got %q", "987", metric.Value)
	}
	if metric.Description != "Total TTS connection duration in milliseconds" {
		t.Fatalf("expected metric description %q, got %q", "Total TTS connection duration in milliseconds", metric.Description)
	}
	assertRecordAttribute(t, record, "provider", "deepgram")
}

func singleMetric(t *testing.T, record RecordMetric) *protos.Metric {
	t.Helper()

	if len(record.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(record.Metrics))
	}
	return record.Metrics[0]
}

func assertRecordAttribute(t *testing.T, record RecordMetric, key string, want string) {
	t.Helper()

	if record.Attributes[key] != want {
		t.Fatalf("expected record attribute %q=%q, got %q", key, want, record.Attributes[key])
	}
}
