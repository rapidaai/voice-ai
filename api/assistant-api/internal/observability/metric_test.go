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
		{MetricSTTInitLatencyMs, "stt.init_ms"},
		{MetricSTTLatencyMs, "stt.latency_ms"},
		{MetricSTTTimeToFirstTokenMs, "stt.ttft_ms"},
		{MetricSTTTimeToLastTokenMs, "stt.ttlt_ms"},
		{MetricTTSInitLatencyMs, "tts_init_ms"},
		{MetricTTSLatencyMs, "tts.latency_ms"},
		{MetricVADInitLatencyMs, "vad.init_ms"},
		{MetricEOSInitLatencyMs, "eos.init_ms"},
		{MetricEOSLatencyMs, "eos.latency_ms"},
		{MetricEOSTextToTriggerMs, "eos.trigger_ms"},
		{MetricEOSWordCount, "eos.word_count"},
		{MetricEOSConfidence, "eos.confidence"},
		{MetricDenoiseInitLatencyMs, "denoise.init_ms"},
		{MetricLLMInitLatencyMs, "agent.init_ms"},
		{MetricAgentLatencyMs, "agent.latency_ms"},
		{MetricAgentTTFTMs, "agent.ttft_ms"},
		{MetricAgentTRTMs, "agent.trt_ms"},
		{MetricStorageInitLatencyMs, "storage.init_ms"},
		{MetricAnalysisInitLatencyMs, "analysis.init_ms"},
		{MetricAuthenticationInitLatencyMs, "authentication.init_ms"},
		{MetricAuthenticationLatencyMs, "authentication.latency_ms"},
		{MetricRecordingInitLatencyMs, "recording.init_ms"},
		{MetricKnowledgeLatencyMs, "knowledge_latency_ms"},
		{MetricAgentError, "agent.error"},
		{MetricValueYes, "1"},
		{MetricSTTError, "stt.error"},
		{MetricTTSError, "tts.error"},
		{MetricDiscardedTTSChunk, "tts.discard_chunk_count"},
		{MetricAgentMessageCount, "agent.message_count"},
		{MetricAgentMessageCharCount, "agent.message_char_count"},
		{MetricAgentResponseCharCount, "agent.response_char_count"},
		{MetricAgentTotalToken, "agent.total_token"},
		{MetricAgentCachedContentToken, "agent.cached_content_token"},
		{MetricAgentCost, "agent.cost"},
		{MetricAgentInputCost, "agent.input_cost"},
		{MetricAgentOutputCost, "agent.output_cost"},
		{MetricAgentLLMRequestID, "agent.llm_request_id"},
		{MetricAgentTokenPerSecond, "agent.token_pre_second"},
	}

	for _, test := range tests {
		if test.actual != test.expected {
			t.Fatalf("expected metric name %q, got %q", test.expected, test.actual)
		}
	}
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
