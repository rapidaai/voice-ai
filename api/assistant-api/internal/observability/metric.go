// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"strconv"
	"time"

	"github.com/rapidaai/protos"
)

// Conversation metric names mirror the current observe/type_enums names.
const (
	MetricConversationStatus      = "status"
	MetricConversationDuration    = "conversation.duration_ms"
	MetricConversationSTTDuration = "stt.duration_ms"
	MetricConversationTTSDuration = "tts.duration_ms"
)

// Current call, SIP, and WebRTC metric names.
const (
	MetricCallDurationMs         = "call.duration_ms"
	MetricCallStatus             = "call.status"
	MetricCallPrice              = "call.price"
	MetricCallTransferDurationMs = "call.transfer.bridge_duration_ms"
	MetricSIPRegistrationStatus  = "sip.registration.status"
	MetricICELatencyMs           = "webrtc.ice_latency_ms"
	MetricWebRTCOutputQueueDrops = "webrtc.output_queue_dropped_frames"
)

// Current call.status metric values.
const (
	MetricCallStatusComplete   = "COMPLETE"
	MetricCallStatusFailed     = "FAILED"
	MetricCallStatusInProgress = "INPROGRESS"
	MetricCallStatusRinging    = "RINGING"
	MetricCallStatusCancelled  = "CANCELLED"
)

// Current turn and provider metric names.
const (
	MetricUserTurn      = "user_turn"
	MetricAssistantTurn = "assistant_turn"

	MetricSTTInitLatencyMs      = "stt.init_ms"
	MetricSTTLatencyMs          = "stt.latency_ms"
	MetricSTTTimeToFirstTokenMs = "stt.ttft_ms"
	MetricSTTTimeToLastTokenMs  = "stt.ttlt_ms"
	MetricSTTError              = "stt.error"

	MetricTTSInitLatencyMs  = "tts_init_ms"
	MetricTTSLatencyMs      = "tts.latency_ms"
	MetricTTSError          = "tts.error"
	MetricDiscardedTTSChunk = "tts.discard_chunk_count"

	MetricVADInitLatencyMs = "vad.init_ms"

	MetricEOSInitLatencyMs   = "eos.init_ms"
	MetricEOSLatencyMs       = "eos.latency_ms"
	MetricEOSTextToTriggerMs = "eos.trigger_ms"
	MetricEOSWordCount       = "eos.word_count"
	MetricEOSConfidence      = "eos.confidence"

	MetricDenoiseInitLatencyMs = "denoise.init_ms"

	MetricLLMInitLatencyMs = "agent.init_ms"
	MetricAgentLatencyMs   = "agent.latency_ms"
	MetricAgentTTFTMs      = "agent.ttft_ms"
	MetricAgentTRTMs       = "agent.trt_ms"
	MetricAgentError       = "agent.error"

	MetricStorageInitLatencyMs        = "storage.init_ms"
	MetricAnalysisInitLatencyMs       = "analysis.init_ms"
	MetricAuthenticationInitLatencyMs = "authentication.init_ms"
	MetricAuthenticationLatencyMs     = "authentication.latency_ms"
	MetricRecordingInitLatencyMs      = "recording.init_ms"

	MetricKnowledgeLatencyMs = "knowledge_latency_ms"

	MetricAgentTotalToken         = "agent.total_token"
	MetricAgentCachedContentToken = "agent.cached_content_token"
	MetricAgentCost               = "agent.cost"
	MetricAgentInputCost          = "agent.input_cost"
	MetricAgentOutputCost         = "agent.output_cost"
	MetricAgentLLMRequestID       = "agent.llm_request_id"
	MetricAgentTokenPerSecond     = "agent.token_pre_second"
)

const (
	MetricValueYes = "1"
)

func NewMetricSTTLatencyMs(duration time.Duration, attr Attributes) RecordMetric {
	record := NewConversationMetricRecord([]*protos.Metric{{
		Name:        MetricSTTLatencyMs,
		Value:       strconv.FormatInt(duration.Milliseconds(), 10),
		Description: "STT latency from speech end to final transcript in milliseconds",
	}})
	record.Attributes = attr
	return record
}

func NewMetricTTSInitLatencyMs(duration time.Duration, attr Attributes) RecordMetric {
	record := NewConversationMetricRecord([]*protos.Metric{{
		Name:        MetricTTSInitLatencyMs,
		Value:       strconv.FormatInt(duration.Milliseconds(), 10),
		Description: "TTS initialization latency in milliseconds",
	}})
	record.Attributes = attr
	return record
}

func NewMetricTTSLatencyMs(duration time.Duration, attr Attributes) RecordMetric {
	record := NewConversationMetricRecord([]*protos.Metric{{
		Name:        MetricTTSLatencyMs,
		Value:       strconv.FormatInt(duration.Milliseconds(), 10),
		Description: "TTS latency from text input to first audio in milliseconds",
	}})
	record.Attributes = attr
	return record
}

func NewMetricSTTDuration(duration time.Duration, attr Attributes) RecordMetric {
	record := NewConversationMetricRecord([]*protos.Metric{{
		Name:        MetricConversationSTTDuration,
		Value:       strconv.FormatInt(duration.Milliseconds(), 10),
		Description: "Total STT connection duration in milliseconds",
	}})
	record.Attributes = attr
	return record
}

func NewMetricTTSDuration(duration time.Duration, attr Attributes) RecordMetric {
	record := NewConversationMetricRecord([]*protos.Metric{{
		Name:        MetricConversationTTSDuration,
		Value:       strconv.FormatInt(duration.Milliseconds(), 10),
		Description: "Total TTS connection duration in milliseconds",
	}})
	record.Attributes = attr
	return record
}
