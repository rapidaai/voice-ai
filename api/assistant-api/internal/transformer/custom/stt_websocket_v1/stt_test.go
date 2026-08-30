// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	transformer_testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sttPacketCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
}

func (collector *sttPacketCollector) onPacket(pkt ...internal_type.Packet) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.packets = append(collector.packets, pkt...)
	return nil
}

func (collector *sttPacketCollector) all() []internal_type.Packet {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	out := make([]internal_type.Packet, len(collector.packets))
	copy(out, collector.packets)
	return out
}

func (collector *sttPacketCollector) hasSTTError() bool {
	for _, packet := range collector.all() {
		if _, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
			return true
		}
	}
	return false
}

func TestNewSpeechToText_ConfigErrorEmitsSTTErrorEvent(t *testing.T) {
	collector := &sttPacketCollector{}
	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: "wss://example.invalid/ws",
		}),
		collector.onPacket,
		utils.Option{},
	)

	require.Error(t, err)
	assert.Nil(t, transformer)
	assert.Contains(t, err.Error(), optionKeyRequestRules)
	assert.False(t, collector.hasSTTError(), "constructor should emit observability, not runtime SpeechToTextErrorPacket")

	var found bool
	for _, packet := range collector.all() {
		event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
		if !ok {
			continue
		}
		if event.Scope == internal_type.ObservabilityRecordScopeConversation &&
			event.Record.Component == observability.ComponentSTT &&
			event.Record.Event == observability.STTError &&
			event.Record.Attributes["provider"] == "custom-stt-websocket-v1" &&
			event.Record.Attributes["operation"] == "load_config" &&
			strings.Contains(event.Record.Attributes["error"], optionKeyRequestRules) {
			found = true
		}
	}
	assert.True(t, found, "expected stt.error observability event for config failure")
}

type blockingResampler struct {
	started chan struct{}
	release chan struct{}
}

func (resampler *blockingResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	select {
	case resampler.started <- struct{}{}:
	default:
	}
	<-resampler.release
	return append([]byte(nil), data...), nil
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition")
}

func TestSpeechToText_WebsocketFlow_JSONRequestRules(t *testing.T) {
	var (
		gotAuthHeader string
		gotModel      string
		gotStartReq   map[string]any
		gotFlushReq   map[string]any
		gotAudioReq   map[string]any
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthHeader = request.Header.Get("Authorization")
		gotModel = request.URL.Query().Get("model")

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.TextMessage, messageType)
		require.NoError(t, json.Unmarshal(payload, &gotStartReq))

		messageType, payload, err = conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.TextMessage, messageType)
		require.NoError(t, json.Unmarshal(payload, &gotFlushReq))

		messageType, payload, err = conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.TextMessage, messageType)
		require.NoError(t, json.Unmarshal(payload, &gotAudioReq))

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"partial","text":"hello","confidence":0.4,"language":"en-US"}`)))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"hello world","confidence":0.9,"language":"en-US"}`)))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
			credentialKeyHeaders:      `{"Authorization":"Bearer abc"}`,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyModel:       "model-a",
			optionKeyLanguage:    "en-US",
			optionKeyQueryParams: `{"model":{"$var":"model"}}`,
			optionKeyRequestRules: `[
				{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start","language":{"$path":"config.language"}}}},
				{"when":{"packet":"interrupt"},"send":{"frame":"json","body":{"type":"flush"}}},
				{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.base64"},"encoding":{"$path":"config.audio.encoding"},"sample_rate":{"$cast":"number","value":{"$path":"config.audio.sample_rate"}}}}}
			]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"json","path":"type","equals":"partial"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":true}},
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-1"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextEndPacket{ContextID: "ctx-1"}))

	audio := []byte{0x01, 0x02, 0x03, 0x04}
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-1",
		Audio:     audio,
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "hello world" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, "Bearer abc", gotAuthHeader)
	assert.Equal(t, "model-a", gotModel)
	require.NotEmpty(t, gotStartReq)
	assert.Equal(t, "start", gotStartReq["type"])
	assert.Equal(t, "en-US", gotStartReq["language"])
	require.NotEmpty(t, gotFlushReq)
	assert.Equal(t, "flush", gotFlushReq["type"])
	require.NotEmpty(t, gotAudioReq)
	assert.Equal(t, "LINEAR16", gotAudioReq["encoding"])
	assert.Equal(t, float64(16000), gotAudioReq["sample_rate"])

	encoded, ok := gotAudioReq["audio"].(string)
	require.True(t, ok)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Equal(t, audio, decoded)

	var (
		hasInterimTranscript     bool
		hasFinalTranscript       bool
		hasInterimEmptyConcat    bool
		hasFinalEmptyConcat      bool
		hasLatencyMetric         bool
		hasSpeechToTextError     bool
		interruptionPacketCount  int
		latencyMetricPacketCount int
	)

	for _, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.SpeechToTextPacket:
			if typed.Interim && typed.Script == "hello" {
				hasInterimTranscript = true
				if typed.Concat != nil && *typed.Concat == "" {
					hasInterimEmptyConcat = true
				}
			}
			if !typed.Interim && typed.Script == "hello world" {
				hasFinalTranscript = true
				if typed.Concat != nil && *typed.Concat == "" {
					hasFinalEmptyConcat = true
				}
			}
		case internal_type.InterruptionDetectedPacket:
			interruptionPacketCount++
		case internal_type.ObservabilityMetricRecordPacket:
			for _, metric := range typed.Record.Metrics {
				if metric.GetName() == "stt.latency_ms" {
					hasLatencyMetric = true
					latencyMetricPacketCount++
				}
			}
		case internal_type.SpeechToTextErrorPacket:
			hasSpeechToTextError = true
		}
	}

	assert.True(t, hasInterimTranscript, "expected interim transcript")
	assert.True(t, hasFinalTranscript, "expected final transcript")
	assert.True(t, hasInterimEmptyConcat, "expected interim transcript with explicit empty concat")
	assert.True(t, hasFinalEmptyConcat, "expected final transcript with explicit empty concat")
	assert.GreaterOrEqual(t, interruptionPacketCount, 2, "expected interruption packet for interim and final transcripts")
	assert.True(t, hasLatencyMetric, "expected stt.latency_ms metric")
	assert.Equal(t, 1, latencyMetricPacketCount, "expected one latency metric per interruption window")
	assert.False(t, hasSpeechToTextError, "did not expect stt error packet")
}

func TestSpeechToText_BinaryAudioResampledWithBinaryRequestRule(t *testing.T) {
	var (
		gotMessageType int
		gotAudioChunk  []byte
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		gotMessageType = messageType
		gotAudioChunk = append([]byte(nil), payload...)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"ok","language":"en-US","confidence":0.8}`)))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyEncoding:     "MuLaw8",
			optionKeySampleRate:   "8000",
			optionKeyRequestRules: `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-2"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextEndPacket{ContextID: "ctx-2"}))

	audio := transformer_testutil.SineTonePCM(440, 1.0)
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-2",
		Audio:     audio,
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "ok" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, websocket.BinaryMessage, gotMessageType)
	assert.Greater(t, len(gotAudioChunk), 0, "expected non-empty resampled chunk")
	assert.NotEqual(t, audio, gotAudioChunk)
}

func TestSpeechToText_WebsocketFlow_TextTranscriptFrames(t *testing.T) {
	var (
		gotMessageType int
		gotAudioChunk  []byte
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		gotMessageType = messageType
		gotAudioChunk = append([]byte(nil), payload...)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("namaste duniya")))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyRequestRules: `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"language":"hi","interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-text"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextEndPacket{ContextID: "ctx-text"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-text",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "namaste duniya" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, websocket.BinaryMessage, gotMessageType)
	assert.NotEmpty(t, gotAudioChunk)

	var (
		hasTranscript bool
		hasLatency    bool
	)

	for _, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.SpeechToTextPacket:
			if !typed.Interim && typed.Script == "namaste duniya" && typed.Language == "hi" {
				hasTranscript = true
			}
		case internal_type.ObservabilityMetricRecordPacket:
			for _, metric := range typed.Record.Metrics {
				if metric.GetName() == "stt.latency_ms" {
					hasLatency = true
				}
			}
		}
	}

	assert.True(t, hasTranscript, "expected transcript from text response frame")
	assert.True(t, hasLatency, "expected final transcript latency metric")
}

func TestSpeechToText_EmitsLatencyMetricFromFirstAudioWithoutStart(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer connection.Close()

		_, _, err = connection.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"hello","confidence":0.8,"language":"en-US"}`)))
		require.NoError(t, connection.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyRequestRules: `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-no-interruption"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-no-interruption",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "hello" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	latencyMetricPacketCount := 0
	for _, packet := range collector.all() {
		metricPacket, ok := packet.(internal_type.ObservabilityMetricRecordPacket)
		if !ok {
			continue
		}
		for _, metric := range metricPacket.Record.Metrics {
			if metric.GetName() == "stt.latency_ms" {
				latencyMetricPacketCount++
			}
		}
	}

	assert.Equal(t, 1, latencyMetricPacketCount, "expected latency metric from first audio without explicit start")
}

func TestSpeechToText_LatencyUsesLatestStartInWindow(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer connection.Close()

		_, _, err = connection.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"hello","confidence":0.8,"language":"en-US"}`)))
		require.NoError(t, connection.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyRequestRules: `[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-first-interrupt"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextStartPacket{ContextID: "ctx-first-interrupt"}))

	time.Sleep(120 * time.Millisecond)

	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextStartPacket{ContextID: "ctx-first-interrupt"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-first-interrupt",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "hello" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	latencyMetricPacketCount := 0
	latencyMilliseconds := int64(-1)
	for _, packet := range collector.all() {
		metricPacket, ok := packet.(internal_type.ObservabilityMetricRecordPacket)
		if !ok {
			continue
		}
		for _, metric := range metricPacket.Record.Metrics {
			if metric.GetName() != "stt.latency_ms" {
				continue
			}
			latencyMetricPacketCount++
			latencyMilliseconds, err = strconv.ParseInt(metric.GetValue(), 10, 64)
			require.NoError(t, err)
		}
	}

	assert.Equal(t, 1, latencyMetricPacketCount, "expected one latency metric for the start window")
	assert.Less(t, latencyMilliseconds, int64(100), "expected latency to be measured from the latest start in the window")
}

func TestSpeechToText_AudioRequestContextStaysBoundToAudioPacket(t *testing.T) {
	var (
		gotAudioContextID string
		serverErr         error
	)

	audioWritten := make(chan struct{}, 1)
	releaseServer := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, payload, err := conn.ReadMessage()
		if err != nil {
			serverErr = err
			return
		}

		var body map[string]any
		if err := json.Unmarshal(payload, &body); err != nil {
			serverErr = err
			return
		}

		contextValue, ok := body["context_id"].(string)
		if !ok {
			serverErr = fmt.Errorf("context_id missing or not string in request body")
			return
		}
		gotAudioContextID = contextValue
		select {
		case audioWritten <- struct{}{}:
		default:
		}
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}
	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyRequestRules:  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"context_id":{"$path":"packet.context_id"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
		},
	)
	require.NoError(t, err)

	typed, ok := transformer.(*speechToText)
	require.True(t, ok)
	blocker := &blockingResampler{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	typed.resampler = blocker

	audioErrCh := make(chan error, 1)
	go func() {
		audioErrCh <- transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
			ContextID: "ctx-a",
			Audio:     []byte{0x01, 0x02, 0x03},
		})
	}()

	select {
	case <-blocker.started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for audio resampler")
	}

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-b"}))
	close(blocker.release)

	select {
	case audioErr := <-audioErrCh:
		require.NoError(t, audioErr)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for audio transform")
	}

	select {
	case <-audioWritten:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for audio request")
	}

	require.NoError(t, transformer.Close(context.Background()))
	require.NoError(t, serverErr)
	assert.Equal(t, "ctx-a", gotAudioContextID, "audio request scope must use the originating audio packet context")
	assert.False(t, collector.hasSTTError(), "did not expect stt error packet")
}
