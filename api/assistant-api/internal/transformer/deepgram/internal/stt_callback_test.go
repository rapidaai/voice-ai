// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package deepgram_internal

import (
	"sync"
	"testing"
	"time"

	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/websocket/interfaces"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Packet Collector - Helper for capturing OnPacket calls
// =============================================================================

type packetCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
}

func newPacketCollector() *packetCollector {
	return &packetCollector{
		packets: make([]internal_type.Packet, 0),
	}
}

func (pc *packetCollector) OnPacket(pkts ...internal_type.Packet) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.packets = append(pc.packets, pkts...)
	return nil
}

func (pc *packetCollector) GetPackets() []internal_type.Packet {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return append([]internal_type.Packet{}, pc.packets...)
}

func (pc *packetCollector) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.packets = make([]internal_type.Packet, 0)
}

func countMetricPackets(packets []internal_type.Packet, metricName string) int {
	count := 0
	for _, packet := range packets {
		metricPacket, ok := packet.(internal_type.ObservabilityMetricRecordPacket)
		if !ok {
			continue
		}
		for _, metric := range metricPacket.Record.Metrics {
			if metric.Name == metricName {
				count++
			}
		}
	}
	return count
}

func findSpeechToTextPacket(packets []internal_type.Packet) (internal_type.SpeechToTextPacket, bool) {
	for _, packet := range packets {
		sttPacket, ok := packet.(internal_type.SpeechToTextPacket)
		if ok {
			return sttPacket, true
		}
	}
	return internal_type.SpeechToTextPacket{}, false
}

func findSpeechToTextPackets(packets []internal_type.Packet) []internal_type.SpeechToTextPacket {
	sttPackets := []internal_type.SpeechToTextPacket{}
	for _, packet := range packets {
		sttPacket, ok := packet.(internal_type.SpeechToTextPacket)
		if ok {
			sttPackets = append(sttPackets, sttPacket)
		}
	}
	return sttPackets
}

func findEventPacket(packets []internal_type.Packet) (internal_type.ObservabilityEventRecordPacket, bool) {
	for _, packet := range packets {
		eventPacket, ok := packet.(internal_type.ObservabilityEventRecordPacket)
		if ok {
			return eventPacket, true
		}
	}
	return internal_type.ObservabilityEventRecordPacket{}, false
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func createTestCallback(opts utils.Option) (*packetCollector, commons.Logger, msginterfaces.LiveMessageCallback) {
	return createTestCallbackWithSpeechEndedAt(opts, time.Now().Add(-100*time.Millisecond))
}

func createTestCallbackWithSpeechEndedAt(opts utils.Option, speechEndedAt time.Time) (*packetCollector, commons.Logger, msginterfaces.LiveMessageCallback) {
	logger, _ := commons.NewApplicationLogger()
	collector := newPacketCollector()
	metrics := &SttSessionMetrics{}
	metrics.SetSpeechEndedAt(speechEndedAt)
	callback := NewDeepgramSttCallback(logger, collector.OnPacket, opts, func() string { return "ctx-test" }, "deepgram-stt", metrics)
	return collector, logger, callback
}

func createTestCallbackWithSpeechWindow(opts utils.Option, speechStartedAt, speechEndedAt time.Time) (*packetCollector, commons.Logger, msginterfaces.LiveMessageCallback) {
	logger, _ := commons.NewApplicationLogger()
	collector := newPacketCollector()
	metrics := &SttSessionMetrics{}
	metrics.ResetSpeech(speechStartedAt)
	metrics.SetSpeechEndedAt(speechEndedAt)
	callback := NewDeepgramSttCallback(logger, collector.OnPacket, opts, func() string { return "ctx-test" }, "deepgram-stt", metrics)
	return collector, logger, callback
}

func createMessageResponse(transcript string, confidence float64, isFinal bool, languages []string) *msginterfaces.MessageResponse {
	return &msginterfaces.MessageResponse{
		Channel: msginterfaces.Channel{
			Alternatives: []msginterfaces.Alternative{
				{
					Transcript: transcript,
					Confidence: confidence,
					Languages:  languages,
				},
			},
		},
		IsFinal: isFinal,
	}
}

func createMultiAlternativeResponse(alternatives []msginterfaces.Alternative, isFinal bool) *msginterfaces.MessageResponse {
	return &msginterfaces.MessageResponse{
		Channel: msginterfaces.Channel{
			Alternatives: alternatives,
		},
		IsFinal: isFinal,
	}
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewDeepgramSttCallback(t *testing.T) {
	t.Run("creates callback with valid options", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		assert.NotNil(t, callback)
		assert.NotNil(t, collector)
	})

	t.Run("creates callback with model options", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 0.8,
		}
		_, _, callback := createTestCallback(opts)

		assert.NotNil(t, callback)
	})
}

// =============================================================================
// Open Handler Tests
// =============================================================================

func TestOpen(t *testing.T) {
	t.Run("returns nil on successful open", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.Open(&msginterfaces.OpenResponse{})

		assert.NoError(t, err)
	})

	t.Run("handles nil OpenResponse", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.Open(nil)

		assert.NoError(t, err)
	})
}

// =============================================================================
// Message Handler Tests
// =============================================================================

func TestMessage(t *testing.T) {
	t.Run("processes transcript successfully", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("hello world", 0.95, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final: metric packet + InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket.
		require.Len(t, packets, 4)

		// Second packet should be InterruptionDetectedPacket
		interruption, ok := packets[1].(internal_type.InterruptionDetectedPacket)
		assert.True(t, ok, "first packet should be InterruptionDetectedPacket")
		assert.Equal(t, internal_type.InterruptionSourceWord, interruption.Source)

		// Third packet should be SpeechToTextPacket
		stt, ok := packets[2].(internal_type.SpeechToTextPacket)
		assert.True(t, ok, "second packet should be SpeechToTextPacket")
		assert.Equal(t, "hello world", stt.Script)
		assert.Equal(t, 0.95, stt.Confidence)
		assert.Equal(t, "en", stt.Language)
		assert.False(t, stt.Interim) // IsFinal=true means Interim=false
		assert.Greater(t, stt.Latency, time.Duration(0))
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTLatencyMs))
	})

	t.Run("omits latency metric when speech end time is missing", func(t *testing.T) {
		collector, _, callback := createTestCallbackWithSpeechEndedAt(utils.Option{}, time.Time{})

		mr := createMessageResponse("hello world", 0.95, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final without a speech end timestamp: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket
		require.Len(t, packets, 3)

		_, ok := findEventPacket(packets)
		assert.True(t, ok, "expected ObservabilityEventRecordPacket")
		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, time.Duration(0), stt.Latency)
	})

	t.Run("sets interim true when IsFinal is false", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("hello", 0.9, false, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Interim: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket (no metric)
		require.Len(t, packets, 3)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.True(t, stt.Interim)
	})

	t.Run("ignores empty transcript", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("", 0.95, true, []string{"en"})
		err := callback.Message(mr)

		assert.NoError(t, err)
		assert.Empty(t, collector.GetPackets())
	})

	t.Run("processes only first non-empty alternative", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMultiAlternativeResponse([]msginterfaces.Alternative{
			{Transcript: "", Confidence: 0.5, Languages: []string{"en"}},
			{Transcript: "second transcript", Confidence: 0.8, Languages: []string{"en"}},
			{Transcript: "third transcript", Confidence: 0.9, Languages: []string{"en"}},
		}, true)

		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final: metric packet + InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket.
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "second transcript", stt.Script)
		assert.Equal(t, 0.8, stt.Confidence)
	})

	t.Run("handles empty alternatives", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := &msginterfaces.MessageResponse{
			Channel: msginterfaces.Channel{
				Alternatives: []msginterfaces.Alternative{},
			},
			IsFinal: true,
		}

		err := callback.Message(mr)

		assert.NoError(t, err)
		assert.Empty(t, collector.GetPackets())
	})

	t.Run("handles nil channel alternatives", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := &msginterfaces.MessageResponse{
			IsFinal: true,
		}

		err := callback.Message(mr)

		assert.NoError(t, err)
		assert.Empty(t, collector.GetPackets())
	})

	t.Run("emits latency metric once per speech end", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		require.NoError(t, callback.Message(createMessageResponse("first final", 0.95, true, []string{"en"})))
		require.NoError(t, callback.Message(createMessageResponse("second final", 0.95, true, []string{"en"})))

		assert.Equal(t, 1, countMetricPackets(collector.GetPackets(), observability.MetricSTTLatencyMs))
	})

	t.Run("emits deepgram timing metrics for final transcript", func(t *testing.T) {
		startedAt := time.Now().Add(-250 * time.Millisecond)
		endedAt := time.Now().Add(-100 * time.Millisecond)
		collector, _, callback := createTestCallbackWithSpeechWindow(utils.Option{}, startedAt, endedAt)

		require.NoError(t, callback.Message(createMessageResponse("final timing", 0.95, true, []string{"en"})))

		packets := collector.GetPackets()
		require.Len(t, packets, 6)
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTTimeToFirstTokenMs))
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTTimeToLastTokenMs))
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTLatencyMs))
	})
}

// =============================================================================
// Confidence Threshold Tests
// =============================================================================

func TestMessageWithConfidenceThreshold(t *testing.T) {
	t.Run("emits low_confidence event when confidence below threshold", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 0.9,
		}
		collector, _, callback := createTestCallback(opts)

		// Confidence 0.7 is below threshold 0.9
		mr := createMessageResponse("low confidence text", 0.7, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Below threshold: transcript is filtered, but final provider latency is still recorded.
		require.Len(t, packets, 2)

		evt, ok := findEventPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "stt", evt.Record.Component.String())
		assert.Equal(t, "low_confidence", evt.Record.Attributes["type"])
		assert.Equal(t, "low confidence text", evt.Record.Attributes["script"])
		assert.Equal(t, "0.7000", evt.Record.Attributes["confidence"])
		assert.Equal(t, "0.9000", evt.Record.Attributes["threshold"])
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTLatencyMs))
	})

	t.Run("respects IsFinal when confidence above threshold", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 0.5,
		}
		collector, _, callback := createTestCallback(opts)

		// Confidence 0.9 is above threshold 0.5
		mr := createMessageResponse("high confidence text", 0.9, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final above threshold: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket + ObservabilityMetricRecordPacket
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.False(t, stt.Interim, "should use IsFinal value when confidence is above threshold")
	})

	t.Run("handles exact threshold boundary", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 0.8,
		}
		collector, _, callback := createTestCallback(opts)

		// Confidence exactly at threshold
		mr := createMessageResponse("boundary text", 0.8, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final at boundary: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket + ObservabilityMetricRecordPacket
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		// When confidence equals threshold, it should NOT be below threshold
		assert.False(t, stt.Interim)
	})

	t.Run("handles missing threshold option", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("no threshold text", 0.3, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final without threshold: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket + ObservabilityMetricRecordPacket
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		// Without threshold, should use IsFinal directly
		assert.False(t, stt.Interim)
	})

	t.Run("handles zero threshold", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 0.0,
		}
		collector, _, callback := createTestCallback(opts)

		mr := createMessageResponse("zero threshold text", 0.1, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		// Final above zero threshold: InterruptionDetectedPacket + SpeechToTextPacket + ObservabilityEventRecordPacket + ObservabilityMetricRecordPacket
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.False(t, stt.Interim)
	})

	t.Run("handles threshold of 1.0", func(t *testing.T) {
		opts := utils.Option{
			"listen.threshold": 1.0,
		}
		collector, _, callback := createTestCallback(opts)

		// Any confidence below 1.0 should emit low_confidence event and provider latency.
		mr := createMessageResponse("max threshold text", 0.99, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 2)

		evt, ok := findEventPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "low_confidence", evt.Record.Attributes["type"])
		assert.Equal(t, 1, countMetricPackets(packets, observability.MetricSTTLatencyMs))
	})
}

// =============================================================================
// Language Detection Tests
// =============================================================================

func TestGetMostUsedLanguage(t *testing.T) {
	callback := &deepgramSttCallback{}

	t.Run("returns 'en' for empty languages", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{})
		assert.Equal(t, "en", result)
	})

	t.Run("returns 'en' for nil languages", func(t *testing.T) {
		result := callback.GetMostUsedLanguage(nil)
		assert.Equal(t, "en", result)
	})

	t.Run("returns single language", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{"fr"})
		assert.Equal(t, "fr", result)
	})

	t.Run("returns most frequent language", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{"en", "fr", "en", "de", "en", "fr"})
		assert.Equal(t, "en", result)
	})

	t.Run("handles tie by returning any winner", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{"en", "fr", "en", "fr"})
		// Either "en" or "fr" is acceptable since they're tied
		assert.Contains(t, []string{"en", "fr"}, result)
	})

	t.Run("handles all same language", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{"de", "de", "de", "de"})
		assert.Equal(t, "de", result)
	})

	t.Run("handles mixed case scenarios", func(t *testing.T) {
		result := callback.GetMostUsedLanguage([]string{"EN", "en", "En"})
		// These are treated as different languages
		assert.NotEmpty(t, result)
	})
}

func TestMessageLanguageDetection(t *testing.T) {
	t.Run("detects language from single language array", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("bonjour", 0.95, true, []string{"fr"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "fr", stt.Language)
	})

	t.Run("detects most common language from multiple", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("guten tag", 0.95, true, []string{"de", "de", "en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "de", stt.Language)
	})

	t.Run("defaults to 'en' for empty languages", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("hello", 0.95, true, []string{})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "en", stt.Language)
	})
}

// =============================================================================
// UtteranceEnd Handler Tests
// =============================================================================

func TestUtteranceEnd(t *testing.T) {
	t.Run("returns nil on utterance end", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.UtteranceEnd(&msginterfaces.UtteranceEndResponse{})

		assert.NoError(t, err)
	})

	t.Run("handles nil UtteranceEndResponse", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.UtteranceEnd(nil)

		assert.NoError(t, err)
	})
}

// =============================================================================
// Metadata Handler Tests
// =============================================================================

func TestMetadata(t *testing.T) {
	t.Run("emits empty stt request id metadata", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		err := callback.Metadata(&msginterfaces.MetadataResponse{})

		assert.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 1)
		metadataPacket, ok := packets[0].(internal_type.ObservabilityMetadataRecordPacket)
		require.True(t, ok)
		require.Len(t, metadataPacket.Record.Metadata, 1)
		assert.Equal(t, observability.MetadataSTTRequestID, metadataPacket.Record.Metadata[0].Key)
		assert.Equal(t, "", metadataPacket.Record.Metadata[0].Value)
	})

	t.Run("handles nil MetadataResponse", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.Metadata(nil)

		assert.NoError(t, err)
	})

	t.Run("emits stt request id metadata", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		err := callback.Metadata(&msginterfaces.MetadataResponse{RequestID: "dg-request-1"})

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 1)
		metadataPacket, ok := packets[0].(internal_type.ObservabilityMetadataRecordPacket)
		require.True(t, ok)
		assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, metadataPacket.Scope)
		require.Len(t, metadataPacket.Record.Metadata, 1)
		assert.Equal(t, observability.MetadataSTTRequestID, metadataPacket.Record.Metadata[0].Key)
		assert.Equal(t, "dg-request-1", metadataPacket.Record.Metadata[0].Value)
	})
}

// =============================================================================
// SpeechStarted Handler Tests
// =============================================================================

func TestSpeechStarted(t *testing.T) {
	t.Run("returns nil on speech started", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.SpeechStarted(&msginterfaces.SpeechStartedResponse{})

		assert.NoError(t, err)
	})

	t.Run("handles nil SpeechStartedResponse", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.SpeechStarted(nil)

		assert.NoError(t, err)
	})
}

// =============================================================================
// Close Handler Tests
// =============================================================================

func TestClose(t *testing.T) {
	t.Run("returns nil on close", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.Close(&msginterfaces.CloseResponse{})

		assert.NoError(t, err)
	})

	t.Run("handles nil CloseResponse", func(t *testing.T) {
		_, _, callback := createTestCallback(utils.Option{})

		err := callback.Close(nil)

		assert.NoError(t, err)
	})
}

func TestError(t *testing.T) {
	t.Run("records stt error metric at conversation scope", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		err := callback.Error(&msginterfaces.ErrorResponse{ErrMsg: "deepgram error"})

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 2)

		metric := packets[0].(internal_type.ObservabilityMetricRecordPacket)
		assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, metric.Scope)
		require.Len(t, metric.Record.Metrics, 1)
		assert.Equal(t, observability.MetricSTTError, metric.Record.Metrics[0].Name)
		assert.Equal(t, "deepgram-stt", metric.Record.Attributes["provider"])
		assert.NotContains(t, metric.Record.Attributes, "messageId")
	})
}

// =============================================================================
// Integration / Scenario Tests
// =============================================================================

func TestMessageSequence(t *testing.T) {
	t.Run("handles sequence of interim and final messages", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		// Simulate typical Deepgram behavior: interim results followed by final
		messages := []*msginterfaces.MessageResponse{
			createMessageResponse("hel", 0.7, false, []string{"en"}),
			createMessageResponse("hello", 0.8, false, []string{"en"}),
			createMessageResponse("hello wo", 0.75, false, []string{"en"}),
			createMessageResponse("hello world", 0.95, true, []string{"en"}),
		}

		for _, mr := range messages {
			err := callback.Message(mr)
			require.NoError(t, err)
		}

		packets := collector.GetPackets()
		// 3 interim × 3 packets + final latency metric + final transcript packets = 13 packets
		assert.Len(t, packets, 13)

		// Verify the last SpeechToTextPacket is marked as final (Interim=false)
		sttPackets := findSpeechToTextPackets(packets)
		require.NotEmpty(t, sttPackets)
		lastStt := sttPackets[len(sttPackets)-1]
		assert.False(t, lastStt.Interim)
		assert.Equal(t, "hello world", lastStt.Script)
	})

	t.Run("handles mixed empty and non-empty transcripts", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		messages := []*msginterfaces.MessageResponse{
			createMessageResponse("", 0.0, false, []string{}), // Empty - ignored
			createMessageResponse("hello", 0.9, false, []string{"en"}),
			createMessageResponse("", 0.0, false, []string{}), // Empty - ignored
			createMessageResponse("world", 0.95, true, []string{"en"}),
		}

		for _, mr := range messages {
			err := callback.Message(mr)
			require.NoError(t, err)
		}

		packets := collector.GetPackets()
		// 1 interim × 3 packets + 1 final × 4 packets = 7 packets
		assert.Len(t, packets, 7)
	})
}

func TestConcurrentMessageProcessing(t *testing.T) {
	t.Run("handles concurrent message calls safely", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				mr := createMessageResponse("concurrent message", 0.9, true, []string{"en"})
				err := callback.Message(mr)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		packets := collector.GetPackets()
		// Each final message generates 3 transcript packets; the session emits one STT latency metric.
		assert.Len(t, packets, numGoroutines*3+1)
	})
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestEdgeCases(t *testing.T) {
	t.Run("handles very long transcript", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		longText := string(make([]byte, 10000))
		for i := range longText {
			longText = longText[:i] + "a" + longText[i+1:]
		}

		mr := createMessageResponse(longText, 0.9, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Len(t, stt.Script, 10000)
	})

	t.Run("handles unicode transcript", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		unicodeText := "こんにちは世界 🌍 مرحبا العالم"
		mr := createMessageResponse(unicodeText, 0.9, true, []string{"ja", "ar"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, unicodeText, stt.Script)
	})

	t.Run("handles special characters in transcript", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		specialText := "Hello! @#$%^&*()_+{}|:<>?~`-=[]\\;',./"
		mr := createMessageResponse(specialText, 0.9, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, specialText, stt.Script)
	})

	t.Run("handles zero confidence", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMessageResponse("zero confidence", 0.0, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, 0.0, stt.Confidence)
	})

	t.Run("handles confidence greater than 1", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		// Edge case: confidence > 1 (shouldn't happen, but test handling)
		mr := createMessageResponse("high confidence", 1.5, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, 1.5, stt.Confidence)
	})

	t.Run("handles negative confidence", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		// Edge case: negative confidence (shouldn't happen, but test handling)
		mr := createMessageResponse("negative confidence", -0.5, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, -0.5, stt.Confidence)
	})

	t.Run("handles whitespace-only transcript", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		// Whitespace is not empty string, so should be processed
		mr := createMessageResponse("   ", 0.9, true, []string{"en"})
		err := callback.Message(mr)

		require.NoError(t, err)
		packets := collector.GetPackets()
		require.Len(t, packets, 4)

		stt, ok := findSpeechToTextPacket(packets)
		require.True(t, ok)
		assert.Equal(t, "   ", stt.Script)
	})

	t.Run("handles many alternatives with all empty transcripts", func(t *testing.T) {
		collector, _, callback := createTestCallback(utils.Option{})

		mr := createMultiAlternativeResponse([]msginterfaces.Alternative{
			{Transcript: "", Confidence: 0.9},
			{Transcript: "", Confidence: 0.8},
			{Transcript: "", Confidence: 0.7},
		}, true)

		err := callback.Message(mr)

		require.NoError(t, err)
		assert.Empty(t, collector.GetPackets())
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMessage(b *testing.B) {
	collector, _, callback := createTestCallback(utils.Option{})
	mr := createMessageResponse("benchmark transcript", 0.95, true, []string{"en"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callback.Message(mr)
	}

	_ = collector.GetPackets()
}

func BenchmarkMessageWithThreshold(b *testing.B) {
	opts := utils.Option{
		"listen.threshold": 0.8,
	}
	collector, _, callback := createTestCallback(opts)
	mr := createMessageResponse("benchmark transcript", 0.95, true, []string{"en"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callback.Message(mr)
	}

	_ = collector.GetPackets()
}

func BenchmarkGetMostUsedLanguage(b *testing.B) {
	callback := &deepgramSttCallback{}
	languages := []string{"en", "fr", "en", "de", "en", "fr", "ja", "en", "de"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callback.GetMostUsedLanguage(languages)
	}
}
