// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	transformer_testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type packetCollector struct {
	mu       sync.Mutex
	packets  []internal_type.Packet
	speechCh chan internal_type.SpeechToTextPacket
}

func newPacketCollector() *packetCollector {
	return &packetCollector{speechCh: make(chan internal_type.SpeechToTextPacket, 1)}
}

func (collector *packetCollector) onPacket(packets ...internal_type.Packet) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.packets = append(collector.packets, packets...)
	for _, packet := range packets {
		if speech, ok := packet.(internal_type.SpeechToTextPacket); ok {
			select {
			case collector.speechCh <- speech:
			default:
			}
		}
	}
	return nil
}

func (collector *packetCollector) all() []internal_type.Packet {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	out := make([]internal_type.Packet, len(collector.packets))
	copy(out, collector.packets)
	return out
}

func (collector *packetCollector) waitForSTTError(t *testing.T) internal_type.SpeechToTextErrorPacket {
	t.Helper()
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SpeechToTextErrorPacket")
		case <-ticker.C:
			for _, packet := range collector.all() {
				if sttErr, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
					return sttErr
				}
			}
		}
	}
}

func TestSpeechToText_HTTPFlow_FlushesBufferedSpeechOnVADEnd(t *testing.T) {
	requestSeen := make(chan struct{}, 1)
	var (
		gotAuthorization string
		gotLanguageQuery string
		gotRequestBody   map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotLanguageQuery = request.URL.Query().Get("language")
		require.Equal(t, http.MethodPost, request.Method)
		require.NoError(t, json.NewDecoder(request.Body).Decode(&gotRequestBody))

		audioBase64, ok := gotRequestBody["audio"].(string)
		require.True(t, ok)
		wavAudio, err := base64.StdEncoding.DecodeString(audioBase64)
		require.NoError(t, err)
		require.Greater(t, len(wavAudio), 44)
		assert.Equal(t, "RIFF", string(wavAudio[0:4]))
		assert.Equal(t, "WAVE", string(wavAudio[8:12]))
		assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, wavAudio[44:])

		select {
		case requestSeen <- struct{}{}:
		default:
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"text":"namaste","confidence":"0.88","language":"hi"}`))
	}))
	defer server.Close()

	collector := newPacketCollector()
	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: server.URL,
			credentialKeyHeaders:      `{"Authorization":"Bearer test-token"}`,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyLanguage:     "hi",
			optionKeyQueryParams:  `{"language":{"$var":"language"}}`,
			optionKeyRequestRules: `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.wav_base64"},"language":{"$path":"config.language"},"stream":false}}}]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)

	typedTransformer, ok := transformer.(*speechToText)
	require.True(t, ok)
	require.NotNil(t, typedTransformer.resampler)
	typedTransformer.httpClient = server.Client()

	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-http"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextStartPacket{ContextID: "ctx-http"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-http",
		Audio:     []byte{0x01, 0x02},
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-http",
		Audio:     []byte{0x03, 0x04},
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextEndPacket{ContextID: "ctx-http"}))

	select {
	case <-requestSeen:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for HTTP STT request")
	}

	var speech internal_type.SpeechToTextPacket
	select {
	case speech = <-collector.speechCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for SpeechToTextPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))
	assert.Equal(t, "Bearer test-token", gotAuthorization)
	assert.Equal(t, "hi", gotLanguageQuery)
	assert.Equal(t, "hi", gotRequestBody["language"])
	assert.Equal(t, false, gotRequestBody["stream"])
	assert.Equal(t, "ctx-http", speech.ContextID)
	assert.Equal(t, "namaste", speech.Script)
	assert.Equal(t, 0.88, speech.Confidence)
	assert.Equal(t, "hi", speech.Language)
	assert.False(t, speech.Interim)

	var hasLatencyMetric bool
	for _, packet := range collector.all() {
		if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
			for _, item := range metric.Record.Metrics {
				if item.GetName() == "stt.latency_ms" {
					hasLatencyMetric = true
				}
			}
		}
	}
	assert.True(t, hasLatencyMetric, "expected stt.latency_ms metric")
}

func TestSpeechToText_HTTPStatusError_EmitsErrorLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	collector := newPacketCollector()
	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: server.URL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyRequestRules:  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.base64"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
		},
	)
	require.NoError(t, err)

	typedTransformer, ok := transformer.(*speechToText)
	require.True(t, ok)
	typedTransformer.httpClient = server.Client()

	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextStartPacket{ContextID: "ctx-http-500"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-http-500",
		Audio:     []byte{0x01, 0x02},
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextEndPacket{ContextID: "ctx-http-500"}))

	sttErr := collector.waitForSTTError(t)
	assert.Equal(t, "stt: custom-stt http_v1: status 500: Internal Server Error", sttErr.ErrMessage())
	assert.Equal(t, internal_type.STTErrorType(internal_type.STTNetworkTimeout), sttErr.Type)

	var errorLog *internal_type.ObservabilityLogRecordPacket
	for _, packet := range collector.all() {
		if logPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok && logPacket.Record.Level == observability.LevelError {
			errorLog = &logPacket
			break
		}
	}
	require.NotNil(t, errorLog, "expected custom STT HTTP error log")
	assert.Equal(t, "ctx-http-500", errorLog.ContextID)
	assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, errorLog.Scope)
	assert.Equal(t, "stt: custom-stt http_v1: status 500: Internal Server Error", errorLog.Record.Message)
	assert.Equal(t, observability.ComponentSTT.String(), errorLog.Record.Attributes["component"])
	assert.Equal(t, "custom-stt-http-v1", errorLog.Record.Attributes["provider"])
	assert.Equal(t, "http_transcribe", errorLog.Record.Attributes["operation"])
	assert.Equal(t, "500", errorLog.Record.Attributes["http_status"])
	assert.Equal(t, "true", errorLog.Record.Attributes["recoverable"])
	assert.Equal(t, "stt: custom-stt http_v1: status 500: Internal Server Error", errorLog.Record.Attributes["error"])
}
