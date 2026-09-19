// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ttsContractTransport func(*http.Request) (*http.Response, error)

func (transport ttsContractTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type synchronousTTSFixture struct {
	onPacket             func(...internal_type.Packet) error
	interruptedContextID atomic.Value
	closeCount           atomic.Int32
}

func (*synchronousTTSFixture) Name() string      { return "synchronous-test" }
func (*synchronousTTSFixture) Initialize() error { return nil }

func (fixture *synchronousTTSFixture) Transform(ctx context.Context, packet internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch packet := packet.(type) {
	case internal_type.TextToSpeechTextPacket:
		return fixture.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: packet.ContextID, AudioChunk: []byte{1, 2}})
	case internal_type.TextToSpeechDonePacket:
		if fixture.interruptedContextID.Load() == packet.ContextID {
			return nil
		}
		return fixture.onPacket(internal_type.TextToSpeechEndPacket{ContextID: packet.ContextID})
	case internal_type.TextToSpeechInterruptPacket:
		fixture.interruptedContextID.Store(packet.ContextID)
		return nil
	default:
		return fmt.Errorf("unexpected test packet %T", packet)
	}
}

func (fixture *synchronousTTSFixture) Close(context.Context) error {
	fixture.closeCount.Add(1)
	return nil
}

func TestTTSSynchronousInterruptionContract(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fixture := &synchronousTTSFixture{}
		testTTSInterruption(t, func(_ context.Context, onPacket func(...internal_type.Packet) error) (internal_type.TextToSpeechTransformer, error) {
			fixture.onPacket = onPacket
			return fixture, nil
		})
		require.Equal(t, "test-tts-interrupt", fixture.interruptedContextID.Load())
		require.EqualValues(t, 1, fixture.closeCount.Load())
	})
}

// TestTTSHTTPContract checks the shared TTS scenarios using buffered HTTP responses without network access.
func TestTTSHTTPContract(t *testing.T) {
	client := http.DefaultClient
	var requests atomic.Int32
	http.DefaultClient = &http.Client{Transport: ttsContractTransport(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, []string{"api.groq.com", "polly.us-east-1.amazonaws.com"}, request.URL.Host)
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(strings.Repeat("audio", 2048))),
		}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = client })

	for _, scenario := range []struct {
		name     string
		run      func(*testing.T, ttsTestFactory)
		requests int32
	}{
		{name: "basic", run: testTTSBasic, requests: 1},
		{name: "metrics", run: testTTSMetricsAndEvents, requests: 2},
		{name: "chunks", run: testTTSMultiChunk, requests: 1},
		{name: "interruption", run: testTTSInterruption, requests: 2},
		{name: "new-session", run: testTTSNewSession, requests: 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			for _, provider := range []string{"aws", "groq"} {
				t.Run(provider, func(t *testing.T) {
					requests.Store(0)
					scenario.run(t, func(ctx context.Context, onPacket func(...internal_type.Packet) error) (internal_type.TextToSpeechTransformer, error) {
						return transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), provider,
							testutil.BuildCredential(map[string]string{"key": "test-key", "access_key_id": "test-access", "secret_access_key": "test-secret"}),
							onPacket, testutil.BuildOptions(nil))
					})
					require.Equal(t, scenario.requests, requests.Load(), "scenario must reach the provider")
				})
			}
		})
	}
}

// TestTTSWebSocketContract exercises early audio and canceled-context responses on a reused socket.
func TestTTSWebSocketContract(t *testing.T) {
	for _, scenario := range []struct {
		name                   string
		run                    func(*testing.T, ttsTestFactory)
		connections            int32
		texts                  int32
		completions            int32
		interrupts             int32
		disconnectFirstMessage bool
	}{
		{name: "basic", run: testTTSBasic, connections: 1, texts: 1, completions: 1},
		{name: "metrics", run: testTTSMetricsAndEvents, connections: 1, texts: 2, completions: 2},
		{name: "chunks", run: testTTSMultiChunk, connections: 1, texts: 5, completions: 1},
		{name: "interruption", run: testTTSInterruption, connections: 1, texts: 2, completions: 2, interrupts: 1},
		{name: "new-session", run: testTTSNewSession, connections: 2, texts: 2, completions: 2},
		{
			name: "connection-loss", connections: 2, texts: 2, completions: 2, disconnectFirstMessage: true,
			run: func(t *testing.T, newTransformer ttsTestFactory) {
				collector := testutil.NewPacketCollector()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				tts, err := newTransformer(ctx, collector.OnPacket)
				require.NoError(t, err)
				t.Cleanup(func() {
					cancel()
					if tts != nil {
						assert.NoError(t, tts.Close(ctx))
					}
				})
				require.NoError(t, tts.Initialize())
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "This connection will drop."}))
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
				collector.WaitFor(t, 5*time.Second, "connection failure", func() bool {
					for _, packet := range collector.GetPackets() {
						if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok && failure.ContextID == "failed" {
							return true
						}
					}
					return false
				})
				require.Empty(t, collector.EndPackets(), "connection failure must not complete the message")

				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "Do not retry this text."}))
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "recovered", Text: "Continue on the new connection."}))
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "failed"}))
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "recovered"}))
				collector.WaitForTTSEnd(t, 5*time.Second)
				cancel()
				require.NoError(t, tts.Close(ctx))
				tts = nil

				require.Len(t, collector.EndPackets(), 1)
				assert.Equal(t, "recovered", collector.EndPackets()[0].ContextID)
				require.Len(t, collector.AudioPackets(), 2)
				assert.Equal(t, "failed", collector.AudioPackets()[0].ContextID)
				assert.Equal(t, "recovered", collector.AudioPackets()[1].ContextID)
				for _, packet := range collector.AudioPackets() {
					assert.Equal(t, []byte{1, 2}, packet.AudioChunk)
				}
				failures := 0
				for _, packet := range collector.GetPackets() {
					if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
						failures++
						assert.Equal(t, "failed", failure.ContextID)
						assert.Error(t, failure.Error)
					}
				}
				assert.Equal(t, 1, failures)
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var connections, texts, completions, interrupts atomic.Int32
			var handlers sync.WaitGroup
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				handlers.Add(1)
				defer handlers.Done()
				connection, err := upgrader.Upgrade(writer, request, nil)
				if !assert.NoError(t, err) {
					return
				}
				defer connection.Close()
				stop := context.AfterFunc(t.Context(), func() { _ = connection.Close() })
				defer stop()
				connectionNumber := connections.Add(1)
				for {
					var message struct {
						ContextID  string `json:"context_id"`
						Transcript string `json:"transcript"`
						Continue   bool   `json:"continue"`
						Cancel     bool   `json:"cancel"`
					}
					if err := connection.ReadJSON(&message); err != nil {
						return
					}
					assert.NotEmpty(t, message.ContextID)
					switch {
					case message.Cancel:
						interrupts.Add(1)
						// Late audio and completion must not reach the canceled message.
						if err := connection.WriteJSON(map[string]interface{}{
							"type": "chunk", "context_id": message.ContextID,
							"data": base64.StdEncoding.EncodeToString([]byte{3, 4}),
						}); !assert.NoError(t, err) {
							return
						}
						if err := connection.WriteJSON(map[string]interface{}{
							"type": "done", "context_id": message.ContextID,
						}); !assert.NoError(t, err) {
							return
						}
					case message.Continue:
						texts.Add(1)
						assert.NotEmpty(t, message.Transcript)
						// Send audio on text admission, before reading the final input packet.
						if err := connection.WriteJSON(map[string]interface{}{
							"type": "chunk", "context_id": message.ContextID,
							"data": base64.StdEncoding.EncodeToString([]byte{1, 2}),
						}); !assert.NoError(t, err) {
							return
						}
					default:
						completions.Add(1)
						assert.Empty(t, message.Transcript)
						if scenario.disconnectFirstMessage && connectionNumber == 1 {
							return
						}
						if err := connection.WriteJSON(map[string]interface{}{
							"type": "done", "context_id": message.ContextID,
						}); !assert.NoError(t, err) {
							return
						}
					}
				}
			}))
			t.Cleanup(func() { server.Close(); handlers.Wait() })

			previousDialer := websocket.DefaultDialer
			dialer := *previousDialer
			dialer.Proxy = nil
			dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			websocket.DefaultDialer = &dialer
			t.Cleanup(func() { websocket.DefaultDialer = previousDialer })

			scenario.run(t, func(ctx context.Context, onPacket func(...internal_type.Packet) error) (internal_type.TextToSpeechTransformer, error) {
				return transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), "cartesia",
					testutil.BuildCredential(map[string]string{"key": "test-key"}), onPacket, testutil.BuildOptions(nil))
			})
			assert.Equal(t, scenario.connections, connections.Load(), "interruption must reuse the socket; reconnect must create a new one")
			assert.Equal(t, scenario.texts, texts.Load())
			if scenario.name == "interruption" {
				// The interrupted message's Done may be suppressed; recovery must still complete.
				assert.GreaterOrEqual(t, completions.Load(), int32(1))
				assert.LessOrEqual(t, completions.Load(), scenario.completions)
			} else {
				assert.Equal(t, scenario.completions, completions.Load())
			}
			assert.Equal(t, scenario.interrupts, interrupts.Load())
		})
	}
}
