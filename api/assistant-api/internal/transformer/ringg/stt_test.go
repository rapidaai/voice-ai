// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_ringg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRinggSpeechToTextLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	collector := testutil.NewPacketCollector()
	logger := testutil.NewTestLogger()

	startSeen := make(chan map[string]any, 1)
	audioSeen := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, startMsg, err := conn.ReadMessage()
		require.NoError(t, err)
		var startPayload map[string]any
		require.NoError(t, json.Unmarshal(startMsg, &startPayload))
		startSeen <- startPayload

		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":                     "ready",
			"request_id":               "req-123",
			"sample_rate":              16000,
			"language":                 "en",
			"mode":                     "stream",
			"enable_cap_punc":          true,
			"vad_tail_sil_ms":          200,
			"vad_confidence":           0.55,
			"accept_client_vad_events": false,
		}))

		for {
			messageType, audioMsg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType != websocket.BinaryMessage {
				continue
			}
			require.NotEmpty(t, audioMsg)
			audioSeen <- struct{}{}

			require.NoError(t, conn.WriteJSON(map[string]any{
				"type":               "transcript",
				"transcription":      "hello from ringg",
				"is_final":           false,
				"language":           "en",
				"request_id":         "req-123",
				"segment_idx":        1,
				"segments":           2,
				"compute_latency_ms": 12.5,
			}))
			require.NoError(t, conn.WriteJSON(map[string]any{
				"type":                           "transcript",
				"transcription":                  "hello from ringg",
				"is_final":                       true,
				"language":                       "en",
				"request_id":                     "req-123",
				"segment_idx":                    2,
				"segments":                       2,
				"compute_latency_ms":             18.0,
				"audio_duration_sec":             0.4,
				"transcribed_audio_duration_sec": 0.4,
				"processing_time_ms":             21.0,
			}))
			return
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	stt, err := NewRinggSpeechToText(
		ctx,
		logger,
		testutil.BuildCredential(map[string]string{"key": "rk_test_key"}),
		collector.OnPacket,
		testutil.BuildOptions(map[string]interface{}{"listen.language": "en"}),
	)
	require.NoError(t, err)
	require.NotNil(t, stt)
	assert.Equal(t, "ringg-speech-to-text", stt.Name())

	ringgSTT := stt.(*ringgSTT)
	ringgSTT.dialWS = func(
		ctx context.Context,
		urlStr string,
		requestHeader http.Header,
	) (*websocket.Conn, *http.Response, error) {
		assert.Equal(t, RINGG_STT_URL, urlStr)
		return websocket.DefaultDialer.DialContext(ctx, wsURL, requestHeader)
	}

	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	startPayload := <-startSeen
	assert.Equal(t, "start", startPayload["type"])
	assert.Equal(t, float64(16000), startPayload["sample_rate"])
	assert.Equal(t, "int16", startPayload["encoding"])
	assert.Equal(t, "en", startPayload["language"])
	assert.Equal(t, "stream", startPayload["mode"])
	assert.Equal(t, true, startPayload["enable_cap_punc"])
	assert.Equal(t, false, startPayload["accept_client_vad_events"])
	assert.Equal(t, "rk_test_key", startPayload["api_key"])

	events := collector.EventPackets()
	require.NotEmpty(t, events)
	assert.Equal(t, "stt", events[0].Name)
	assert.Equal(t, "initialized", events[0].Data["type"])
	assert.Equal(t, "ringg-speech-to-text", events[0].Data["provider"])

	require.NoError(t, stt.Transform(ctx, internal_type.TurnChangePacket{
		ContextID: "ringg-ctx-1",
	}))
	require.NoError(t, stt.Transform(ctx, internal_type.SpeechToTextEndPacket{
		ContextID: "ringg-ctx-1",
	}))
	require.NoError(t, stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
		ContextID: "ringg-ctx-1",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	select {
	case <-audioSeen:
	case <-ctx.Done():
		t.Fatal("context cancelled before Ringg server observed audio")
	}

	collector.WaitForFinalTranscript(t, 10*time.Second)

	finals := collector.FinalTranscripts()
	require.NotEmpty(t, finals)
	assert.Equal(t, "hello from ringg", finals[0].Script)
	assert.Equal(t, "en", finals[0].Language)
	assert.False(t, finals[0].Interim)

	interims := collector.InterimTranscripts()
	require.NotEmpty(t, interims)
	assert.Equal(t, "hello from ringg", interims[0].Script)
	assert.True(t, interims[0].Interim)

	interruptions := collector.InterruptionDetectedPackets()
	require.NotEmpty(t, interruptions)

	var latencyMetricFound bool
	for _, pkt := range collector.GetPackets() {
		metricPkt, ok := pkt.(internal_type.UserMessageMetricPacket)
		if !ok {
			continue
		}
		for _, metric := range metricPkt.Metrics {
			if metric.GetName() == "stt_latency_ms" {
				latencyMetricFound = true
			}
		}
	}
	assert.True(t, latencyMetricFound, "expected stt_latency_ms metric to be emitted")
}

func TestRinggSpeechToTextHandshakeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":        "error",
			"detail":      "Invalid encoding 'LINEAR16'",
			"code":        "INVALID_ENCODING",
			"status_code": 400,
		}))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	stt, err := NewRinggSpeechToText(
		ctx,
		logger,
		testutil.BuildCredential(map[string]string{"key": "rk_test_key"}),
		collector.OnPacket,
		testutil.BuildOptions(map[string]interface{}{"listen.language": "en"}),
	)
	require.NoError(t, err)

	ringgSTT := stt.(*ringgSTT)
	ringgSTT.dialWS = func(
		ctx context.Context,
		urlStr string,
		requestHeader http.Header,
	) (*websocket.Conn, *http.Response, error) {
		assert.Equal(t, RINGG_STT_URL, urlStr)
		return websocket.DefaultDialer.DialContext(ctx, wsURL, requestHeader)
	}

	err = stt.Initialize()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid encoding")
}
