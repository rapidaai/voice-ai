// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeTTSResampler struct {
	out []byte
	err error
}

func (resampler *fakeTTSResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if resampler.err != nil {
		return nil, resampler.err
	}
	if resampler.out != nil {
		return append([]byte(nil), resampler.out...), nil
	}
	return append([]byte(nil), data...), nil
}

type packetCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
	endCh   chan struct{}
}

func newPacketCollector() *packetCollector {
	return &packetCollector{endCh: make(chan struct{}, 1)}
}

func (collector *packetCollector) onPacket(pkt ...internal_type.Packet) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.packets = append(collector.packets, pkt...)
	for _, packet := range pkt {
		if _, ok := packet.(internal_type.TextToSpeechEndPacket); ok {
			select {
			case collector.endCh <- struct{}{}:
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

func (collector *packetCollector) hasTTSError() bool {
	for _, packet := range collector.all() {
		if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			return true
		}
	}
	return false
}

func TestNewTextToSpeech_ConfigErrorEmitsTTSErrorEvent(t *testing.T) {
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: "wss://example.invalid/ws",
		}),
		collector.onPacket,
		utils.Option{},
	)

	require.Error(t, err)
	assert.Nil(t, transformer)
	assert.Contains(t, err.Error(), optionKeyRequestRules)
	assert.False(t, collector.hasTTSError(), "constructor should emit observability, not runtime TextToSpeechErrorPacket")

	var found bool
	for _, packet := range collector.all() {
		event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
		if !ok {
			continue
		}
		if event.Scope == internal_type.ObservabilityRecordScopeConversation &&
			event.Record.Component == observability.ComponentTTS &&
			event.Record.Event == observability.TTSError &&
			event.Record.Attributes["provider"] == "custom-tts-websocket-v1" &&
			event.Record.Attributes["operation"] == "load_config" &&
			strings.Contains(event.Record.Attributes["error"], optionKeyRequestRules) {
			found = true
		}
	}
	assert.True(t, found, "expected tts.error observability event for config failure")
}

func testWSCredential(t *testing.T, values map[string]any) *protos.VaultCredential {
	t.Helper()
	pb, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.VaultCredential{Value: pb}
}

func TestTextToSpeech_WebsocketFlow_RequestRules(t *testing.T) {
	var (
		gotAuthHeader  string
		gotMessageID   string
		gotTextRequest map[string]any
		gotDoneRequest map[string]any
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthHeader = request.Header.Get("Authorization")
		gotMessageID = request.URL.Query().Get("message_id")

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, firstMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(firstMessage, &gotTextRequest))

		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("pcm-audio")))

		_, doneMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(doneMessage, &gotDoneRequest))

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done","request_id":"ctx-1"}`)))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()

	opts := utils.Option{
		optionKeyVoiceID:     "virat",
		optionKeyQueryParams: `{"message_id":{"$var":"message_id"}}`,
		optionKeyRequestRules: `[
			{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"voice_id":{"$path":"config.voice.id"},"request_id":{"$path":"packet.message_id"}}}},
			{"when":{"packet":"done"},"send":{"frame":"json","body":{"request_id":{"$path":"packet.message_id"},"type":"done"}}}
		]`,
		optionKeyResponseRules: `[
			{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},
			{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"request_id"},"done":true}}
		]`,
	}

	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
			credentialKeyHeaders:      `{"Authorization":"Bearer abc"}`,
		}),
		collector.onPacket,
		opts,
	)
	require.NoError(t, err)
	typedTransformer, ok := transformer.(*textToSpeech)
	require.True(t, ok)
	require.NotNil(t, typedTransformer.resampler)
	require.NoError(t, transformer.Initialize())

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-1",
		Text:      "hello world",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-1",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, "Bearer abc", gotAuthHeader)
	assert.Equal(t, "ctx-1", gotMessageID)
	assert.Equal(t, "hello world", gotTextRequest["text"])
	assert.Equal(t, "virat", gotTextRequest["voice_id"])
	assert.Equal(t, "ctx-1", gotTextRequest["request_id"])
	assert.Equal(t, "done", gotDoneRequest["type"])
	assert.Equal(t, "ctx-1", gotDoneRequest["request_id"])

	packets := collector.all()
	var (
		hasAudio bool
		hasEnd   bool
	)
	for _, packet := range packets {
		switch typed := packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			if typed.ContextID == "ctx-1" && string(typed.AudioChunk) == "pcm-audio" {
				hasAudio = true
			}
		case internal_type.TextToSpeechEndPacket:
			if typed.ContextID == "ctx-1" {
				hasEnd = true
			}
		}
	}

	assert.True(t, hasAudio, "expected audio packet")
	assert.True(t, hasEnd, "expected end packet")
}

func TestTextToSpeech_NormalizesCustomAudioOutput(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("provider-audio")))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done","request_id":"ctx-normalize"}`)))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID:       "virat",
			optionKeySampleRate:    "8000",
			optionKeyRequestRules:  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"request_id"},"done":true}}]`,
		},
	)
	require.NoError(t, err)

	typed, ok := transformer.(*textToSpeech)
	require.True(t, ok)
	require.NotNil(t, typed.resampler, "non-internal custom TTS output should initialize a streaming resampler")
	typed.resampler = &fakeTTSResampler{out: []byte("internal-audio")}

	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-normalize",
		Text:      "hello",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))

	hasAudio := false
	for _, packet := range collector.all() {
		audio, ok := packet.(internal_type.TextToSpeechAudioPacket)
		if ok && audio.ContextID == "ctx-normalize" {
			hasAudio = true
			assert.Equal(t, []byte("internal-audio"), audio.AudioChunk)
			assert.NotEqual(t, []byte("provider-audio"), audio.AudioChunk)
		}
	}
	assert.True(t, hasAudio, "expected normalized audio packet")
	assert.False(t, collector.hasTTSError(), "did not expect TextToSpeechErrorPacket")
}

func TestTextToSpeech_NormalizeAudioErrorSkipsChunk(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("provider-audio")))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done","request_id":"ctx-error"}`)))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID:       "virat",
			optionKeyRequestRules:  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"request_id"},"done":true}}]`,
		},
	)
	require.NoError(t, err)

	typed, ok := transformer.(*textToSpeech)
	require.True(t, ok)
	typed.resampler = &fakeTTSResampler{err: errors.New("resample failed")}

	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-error",
		Text:      "hello",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))

	var (
		hasAudio bool
		hasError bool
	)
	for _, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			hasAudio = true
			assert.NotEqual(t, []byte("provider-audio"), typed.AudioChunk, "raw provider audio must not be emitted after normalization failure")
		case internal_type.TextToSpeechErrorPacket:
			hasError = true
			assert.Equal(t, "ctx-error", typed.ContextID)
			assert.Contains(t, typed.Error.Error(), "failed to resample audio")
		}
	}
	assert.False(t, hasAudio, "did not expect audio packet after normalization failure")
	assert.True(t, hasError, "expected TextToSpeechErrorPacket")
}

func TestTextToSpeech_EndsOnCleanServerClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("pcm-audio")))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID:       "virat",
			optionKeyRequestRules:  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-close",
		Text:      "hello",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))
	assert.False(t, collector.hasTTSError(), "did not expect TextToSpeechErrorPacket on clean close")
}

func TestTextToSpeech_InterruptRequestRule(t *testing.T) {
	var (
		gotTextRequest      map[string]any
		gotInterruptRequest map[string]any
	)

	interruptWritten := make(chan struct{}, 1)
	releaseServer := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, firstMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(firstMessage, &gotTextRequest))

		_, interruptMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(interruptMessage, &gotInterruptRequest))

		select {
		case interruptWritten <- struct{}{}:
		default:
		}
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()

	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID: "virat",
			optionKeyRequestRules: `[
				{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}},
				{"when":{"packet":"interrupt"},"send":{"frame":"json","body":{"type":"interrupt","request_id":{"$path":"packet.message_id"}}}}
			]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-int",
		Text:      "hello",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-int",
	}))

	select {
	case <-interruptWritten:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for interrupt request")
	}

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, "hello", gotTextRequest["text"])
	assert.Equal(t, "ctx-int", gotTextRequest["request_id"])
	assert.Equal(t, "interrupt", gotInterruptRequest["type"])
	assert.Equal(t, "ctx-int", gotInterruptRequest["request_id"])

	hasInterruptedEvent := false
	for _, packet := range collector.all() {
		event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
		if !ok {
			continue
		}
		if event.ContextID == "ctx-int" && event.Record.Component.String() == "tts" && event.Record.Attributes["type"] == "interrupted" {
			hasInterruptedEvent = true
		}
	}
	assert.True(t, hasInterruptedEvent, "expected interrupted conversation event")
}

func TestTextToSpeech_StaleInterruptDoesNotAffectActiveConnection(t *testing.T) {
	var (
		upgradeCount int32
		messagesMu   sync.Mutex
		messages     []map[string]any
	)

	secondTextSeen := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&upgradeCount, 1)

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var payload map[string]any
			require.NoError(t, json.Unmarshal(message, &payload))

			messagesMu.Lock()
			messages = append(messages, payload)
			messagesMu.Unlock()

			if payload["text"] == "hello-2" {
				select {
				case secondTextSeen <- struct{}{}:
				default:
				}
			}
		}
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()

	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID: "virat",
			optionKeyRequestRules: `[
				{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}},
				{"when":{"packet":"interrupt"},"send":{"frame":"json","body":{"type":"interrupt","request_id":{"$path":"packet.message_id"}}}}
			]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-active",
		Text:      "hello-1",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-stale",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-active",
		Text:      "hello-2",
	}))

	select {
	case <-secondTextSeen:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for second text request")
	}

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, int32(1), atomic.LoadInt32(&upgradeCount), "expected a single active websocket")

	messagesMu.Lock()
	defer messagesMu.Unlock()
	for _, payload := range messages {
		assert.NotEqual(t, "interrupt", payload["type"], "stale interrupt must not send interrupt request")
	}

	hasStaleInterruptedEvent := false
	for _, packet := range collector.all() {
		event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
		if !ok {
			continue
		}
		if event.ContextID == "ctx-stale" && event.Record.Component.String() == "tts" && event.Record.Attributes["type"] == "interrupted" {
			hasStaleInterruptedEvent = true
		}
	}
	assert.False(t, hasStaleInterruptedEvent, "stale interrupt must not emit interrupted event")
}

func TestTextToSpeech_ConcurrentTextUsesSingleConnection(t *testing.T) {
	const totalPackets = 24

	var (
		upgradeCount int32
		messageCount int32
	)

	allSeen := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&upgradeCount, 1)

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			count := atomic.AddInt32(&messageCount, 1)
			if count == totalPackets {
				select {
				case allSeen <- struct{}{}:
				default:
				}
			}
		}
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()

	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID:       "virat",
			optionKeyRequestRules:  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}]`,
			optionKeyResponseRules: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())

	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, totalPackets)
	for i := 0; i < totalPackets; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			errCh <- transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
				ContextID: "ctx-concurrent",
				Text:      fmt.Sprintf("hello-%d", index),
			})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	select {
	case <-allSeen:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for websocket requests")
	}

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, int32(1), atomic.LoadInt32(&upgradeCount), "expected only one websocket upgrade")
	assert.Equal(t, int32(totalPackets), atomic.LoadInt32(&messageCount), "expected all text packets to be written")
}

func TestTextToSpeech_CloseEmitsConversationDurationAfterDone(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("pcm-audio")))

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done","request_id":"ctx-metric"}`)))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID: "virat",
			optionKeyRequestRules: `[
				{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}},
				{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done","request_id":{"$path":"packet.message_id"}}}}
			]`,
			optionKeyResponseRules: `[
				{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},
				{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"request_id"},"done":true}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-metric",
		Text:      "hello",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-metric",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))

	hasDurationMetric := false
	hasDurationUsage := false
	for _, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.ObservabilityMetricRecordPacket:
			for _, metric := range typed.Record.Metrics {
				if metric.GetName() == observability.MetricConversationTTSDuration && strings.TrimSpace(metric.GetValue()) != "" {
					hasDurationMetric = true
				}
			}
		case internal_type.ObservabilityUsageRecordPacket:
			if typed.Record.Component == observability.ComponentName(observability.UsageConversationTTSDuration) &&
				typed.Record.Provider == "custom-tts-websocket-v1" &&
				typed.Record.Duration > 0 {
				hasDurationUsage = true
			}
		}
	}
	assert.True(t, hasDurationMetric, "expected CONVERSATION_TTS_DURATION metric after Close")
	assert.True(t, hasDurationUsage, "expected TTS duration usage after Close")
}
