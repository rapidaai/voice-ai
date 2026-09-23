package internal_transformer_elevenlabs

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newElevenLabsTTSServer(t *testing.T) (*elevenlabsTTS, *testutil.PacketCollector, <-chan *websocket.Conn, <-chan map[string]interface{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	connections := make(chan *websocket.Conn, 16)
	requests := make(chan map[string]interface{}, 128)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "local-test-key", r.Header.Get("xi-api-key"))
		assert.Contains(t, r.URL.Path, "/multi-stream-input")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		connections <- conn
		for {
			var request map[string]interface{}
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			select {
			case requests <- request:
			case <-ctx.Done():
				return
			}
		}
	}))
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}
	websocket.DefaultDialer = &dialer
	collector := testutil.NewPacketCollector()
	transformer, err := NewElevenlabsTextToSpeech(ctx, testutil.NewTestLogger(),
		newVaultCredential(map[string]interface{}{"key": "local-test-key"}), collector.OnPacket,
		utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*elevenlabsTTS)
	t.Cleanup(func() {
		cancel()
		require.NoError(t, tts.Close(context.Background()))
		server.Close()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, connections, requests
}

func elevenLabsTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var errors []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			errors = append(errors, failure)
		}
	}
	return errors
}

func TestElevenLabsTTSPersistentContexts(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	ctx := context.Background()
	require.NoError(t, tts.Initialize())
	remote := waitElevenLabsServerConnection(t, connections)
	require.NoError(t, tts.Initialize())
	for _, contextID := range []string{"first", "second"} {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "hello"}))
		init := waitElevenLabsRequest(t, requests)
		assert.Equal(t, contextID, init["context_id"])
		assert.Equal(t, " ", init["text"])
		assert.Equal(t, map[string]interface{}{"stability": 0.5, "similarity_boost": 0.75}, init["voice_settings"])
		assert.Equal(t, map[string]interface{}{"context_id": contextID, "text": "hello"}, waitElevenLabsRequest(t, requests))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: " again"}))
		assert.Equal(t, map[string]interface{}{"context_id": contextID, "text": " again"}, waitElevenLabsRequest(t, requests))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		assert.Equal(t, map[string]interface{}{"context_id": contextID, "flush": true}, waitElevenLabsRequest(t, requests))
		assert.Equal(t, map[string]interface{}{"context_id": contextID, "close_context": true}, waitElevenLabsRequest(t, requests))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		require.NoError(t, remote.WriteJSON(map[string]interface{}{
			"contextId": contextID, "audio": base64.StdEncoding.EncodeToString([]byte{1, 2}), "isFinal": true,
		}))
		collector.WaitFor(t, time.Second, "context completion", func() bool {
			for _, end := range collector.EndPackets() {
				if end.ContextID == contextID {
					return true
				}
			}
			return false
		})
		// Completed contexts must not be reopened by duplicate text or final frames.
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "late"}))
		if contextID == "first" {
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"contextId": contextID, "audio": "AQI=", "isFinal": true,
			}))
		}
	}
	require.Len(t, collector.AudioPackets(), 2)
	require.Len(t, collector.EndPackets(), 2)
	assert.Equal(t, "first", collector.AudioPackets()[0].ContextID)
	assert.Equal(t, "second", collector.AudioPackets()[1].ContextID)
	latencies := 0
	for _, packet := range collector.MetricPackets() {
		for _, metric := range packet.Record.Metrics {
			if metric.Name == observability.MetricTTSLatencyMs {
				latencies++
			}
		}
	}
	assert.Equal(t, 2, latencies)
	assert.Empty(t, requests)
	assert.Empty(t, connections)
}

func TestElevenLabsTTSInterruptOwnership(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	ctx := context.Background()
	for _, contextID := range []string{"old", "new"} {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: contextID}))
		if contextID == "new" {
			assert.Equal(t, map[string]interface{}{"context_id": "old", "close_context": true}, waitElevenLabsRequest(t, requests))
		}
		assert.Equal(t, contextID, waitElevenLabsRequest(t, requests)["context_id"])
		assert.Equal(t, contextID, waitElevenLabsRequest(t, requests)["text"])
	}
	remote := waitElevenLabsServerConnection(t, connections)
	require.NoError(t, tts.Transform(ctx, internal_type.TurnChangePacket{ContextID: "user-turn"}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
	for _, contextID := range []string{"old", "unknown", ""} {
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": contextID, "audio": "AQI=", "isFinal": true}))
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": contextID, "audio": "invalid!"}))
	}
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"audio": "AQI=", "isFinal": true}))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"audio": "invalid!"}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "new"}))
	assert.Equal(t, true, waitElevenLabsRequest(t, requests)["flush"])
	assert.Equal(t, true, waitElevenLabsRequest(t, requests)["close_context"])
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "new", "audio": "AwQ=", "isFinal": true}))
	collector.WaitForTTSEnd(t, time.Second)
	require.Len(t, collector.AudioPackets(), 1)
	assert.Equal(t, "new", collector.AudioPackets()[0].ContextID)
	assert.Equal(t, []byte{3, 4}, collector.AudioPackets()[0].AudioChunk)
	require.Len(t, collector.EndPackets(), 1)
	assert.Equal(t, "new", collector.EndPackets()[0].ContextID)
	assert.Empty(t, requests)
	assert.Empty(t, connections)
}

func TestElevenLabsTTSOverlappingCompletions(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	ctx := context.Background()
	for _, contextID := range []string{"first", "second"} {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: contextID}))
		if contextID == "second" {
			assert.Equal(t, map[string]interface{}{"context_id": "first", "close_context": true}, waitElevenLabsRequest(t, requests))
		}
		waitElevenLabsRequest(t, requests)
		waitElevenLabsRequest(t, requests)
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		waitElevenLabsRequest(t, requests)
		waitElevenLabsRequest(t, requests)
	}
	remote := waitElevenLabsServerConnection(t, connections)
	for _, contextID := range []string{"first", "second"} {
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": contextID, "audio": "AQI=", "isFinal": true}))
	}
	collector.WaitForTTSEnd(t, time.Second)
	require.Len(t, collector.AudioPackets(), 1)
	require.Len(t, collector.EndPackets(), 1)
	assert.Equal(t, "second", collector.AudioPackets()[0].ContextID)
	assert.Equal(t, "second", collector.EndPackets()[0].ContextID)
	assert.Empty(t, elevenLabsTTSErrors(collector))
}

func TestElevenLabsTTSInterruptAfterDone(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	ctx := context.Background()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	remote := waitElevenLabsServerConnection(t, connections)
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
	assert.Equal(t, map[string]interface{}{"context_id": "old", "close_context": true}, waitElevenLabsRequest(t, requests))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "old", "audio": "AQI=", "isFinal": true}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "new"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "new", "isFinal": true}))
	collector.WaitForTTSEnd(t, time.Second)
	assert.Empty(t, collector.AudioPackets())
	require.Len(t, collector.EndPackets(), 1)
	assert.Equal(t, "new", collector.EndPackets()[0].ContextID)
	assert.Empty(t, connections)
}

func TestElevenLabsTTSContextErrorOwnership(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	ctx := context.Background()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "failed"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	remote := waitElevenLabsServerConnection(t, connections)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "failed", "error": "rejected"}))
	assert.Equal(t, map[string]interface{}{"context_id": "failed", "close_context": true}, waitElevenLabsRequest(t, requests))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "live", Text: "live"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "failed", "error": "duplicate"}))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "failed", "audio": "AQI=", "isFinal": true}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "live"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "live", "audio": "AQI=", "isFinal": true}))
	collector.WaitForTTSEnd(t, time.Second)
	require.Len(t, elevenLabsTTSErrors(collector), 1)
	assert.Equal(t, "failed", elevenLabsTTSErrors(collector)[0].ContextID)
	require.Len(t, collector.AudioPackets(), 1)
	assert.Equal(t, "live", collector.AudioPackets()[0].ContextID)
	assert.Empty(t, connections)
}

func TestElevenLabsTTSReconnectDoesNotReplay(t *testing.T) {
	for _, failure := range []string{"read", "write", "provider", "bad-audio"} {
		t.Run(failure, func(t *testing.T) {
			tts, collector, connections, requests := newElevenLabsTTSServer(t)
			ctx := context.Background()
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "first"}))
			remote := waitElevenLabsServerConnection(t, connections)
			waitElevenLabsRequest(t, requests)
			waitElevenLabsRequest(t, requests)
			switch failure {
			case "read":
				require.NoError(t, remote.Close())
			case "write":
				tts.stateMu.Lock()
				require.NoError(t, tts.connection.Close())
				tts.stateMu.Unlock()
				require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "second"}))
			case "provider":
				require.NoError(t, remote.WriteJSON(map[string]interface{}{"error": "provider rejected request"}))
			case "bad-audio":
				require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "failed", "audio": "invalid!"}))
			}
			collector.WaitFor(t, time.Second, "contextual failure", func() bool { return len(elevenLabsTTSErrors(collector)) == 1 })
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "late"}))
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
			assert.Empty(t, connections)
			assert.Empty(t, requests)
			assert.Empty(t, collector.EndPackets())
			assert.Equal(t, "failed", elevenLabsTTSErrors(collector)[0].ContextID)
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			remote = waitElevenLabsServerConnection(t, connections)
			assert.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
			assert.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
			waitElevenLabsRequest(t, requests)
			waitElevenLabsRequest(t, requests)
			require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "fresh", "audio": "AQI=", "isFinal": true}))
			collector.WaitForTTSEnd(t, time.Second)
			assert.Len(t, elevenLabsTTSErrors(collector), 1)
			assert.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
		})
	}
}

func TestElevenLabsTTSConcurrentInitializeAndText(t *testing.T) {
	tts, _, connections, requests := newElevenLabsTTSServer(t)
	var workers sync.WaitGroup
	for range 12 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			assert.NoError(t, tts.Initialize())
			assert.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "same", Text: "chunk"}))
		}()
	}
	workers.Wait()
	assert.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	for range 12 {
		assert.Equal(t, map[string]interface{}{"context_id": "same", "text": "chunk"}, waitElevenLabsRequest(t, requests))
	}
	waitElevenLabsServerConnection(t, connections)
	assert.Empty(t, connections)
	assert.Empty(t, requests)
}

func TestElevenLabsTTSInitializeFailureAndLazyRetry(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	dial := websocket.DefaultDialer.NetDialTLSContext
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("local dial failure")
	}
	require.ErrorContains(t, tts.Initialize(), "local dial failure")
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "hello"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "late"}))
	require.Len(t, elevenLabsTTSErrors(collector), 1)
	assert.Equal(t, "failed", elevenLabsTTSErrors(collector)[0].ContextID)
	websocket.DefaultDialer.NetDialTLSContext = dial
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
	waitElevenLabsServerConnection(t, connections)
	assert.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	assert.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
}

func TestElevenLabsTTSDialCancellation(t *testing.T) {
	tts, _, _, _ := newElevenLabsTTSServer(t)
	dialStarted := make(chan struct{})
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.LessOrEqual(t, time.Until(deadline), 2*time.Second)
		close(dialStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	result := make(chan error, 1)
	go func() { result <- tts.Initialize() }()
	<-dialStarted
	require.NoError(t, tts.Close(context.Background()))
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("session cancellation did not stop dial")
	}
}

func TestElevenLabsTTSStalledUpgradeCancellation(t *testing.T) {
	for _, mode := range []string{"deadline", "caller cancellation", "close"} {
		t.Run(mode, func(t *testing.T) {
			tts, collector, _, _ := newElevenLabsTTSServer(t)
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				<-release
			}))
			t.Cleanup(func() { close(release); server.Close() })
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "dial", Text: "hello"})
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("HTTP upgrade did not start")
			}
			start := time.Now()
			switch mode {
			case "caller cancellation":
				cancel()
			case "close":
				require.NoError(t, tts.Close(context.Background()))
			}
			select {
			case err := <-result:
				if mode != "deadline" {
					require.ErrorIs(t, err, context.Canceled)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("HTTP upgrade did not stop")
			}
			if mode != "deadline" {
				assert.Less(t, time.Since(start), time.Second)
			}
			if mode != "deadline" {
				assert.Empty(t, elevenLabsTTSErrors(collector))
			} else {
				require.Len(t, elevenLabsTTSErrors(collector), 1)
				assert.Equal(t, "dial", elevenLabsTTSErrors(collector)[0].ContextID)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "dial", Text: "late"}))
				assert.Len(t, elevenLabsTTSErrors(collector), 1)
			}
		})
	}
}

type elevenLabsTTSStalledWriteConn struct {
	net.Conn
	stalled net.Conn
	started chan struct{}
	block   atomic.Bool
}

func (connection *elevenLabsTTSStalledWriteConn) Write(payload []byte) (int, error) {
	if connection.block.Load() {
		close(connection.started)
		return connection.stalled.Write(payload)
	}
	return connection.Conn.Write(payload)
}

func (connection *elevenLabsTTSStalledWriteConn) SetWriteDeadline(deadline time.Time) error {
	_ = connection.stalled.SetWriteDeadline(deadline)
	return connection.Conn.SetWriteDeadline(deadline)
}

func (connection *elevenLabsTTSStalledWriteConn) Close() error {
	_ = connection.stalled.Close()
	return connection.Conn.Close()
}

func TestElevenLabsTTSWriteCancellation(t *testing.T) {
	for _, packet := range []internal_type.Packet{
		internal_type.TextToSpeechTextPacket{ContextID: "write", Text: "second"},
		internal_type.TextToSpeechDonePacket{ContextID: "write"},
		internal_type.TextToSpeechInterruptPacket{ContextID: "write"},
	} {
		for _, mode := range []string{"caller cancellation", "socket failure", "close"} {
			t.Run(string(packet.PacketName())+"/"+mode, func(t *testing.T) {
				tts, collector, connections, requests := newElevenLabsTTSServer(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				stalled, peer := net.Pipe()
				t.Cleanup(func() { _ = stalled.Close(); _ = peer.Close() })
				var client *elevenLabsTTSStalledWriteConn
				websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					connection, err := dial(ctx, network, address)
					if err != nil {
						return nil, err
					}
					client = &elevenLabsTTSStalledWriteConn{Conn: connection, stalled: stalled, started: make(chan struct{})}
					return client, nil
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "write", Text: "first"}))
				waitElevenLabsServerConnection(t, connections)
				waitElevenLabsRequest(t, requests)
				waitElevenLabsRequest(t, requests)
				client.block.Store(true)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				finished := make(chan error, 1)
				go func() { finished <- tts.Transform(ctx, packet) }()
				select {
				case <-client.started:
				case <-time.After(time.Second):
					t.Fatal("writer did not block")
				}
				start := time.Now()
				switch mode {
				case "caller cancellation":
					cancel()
				case "socket failure":
					require.NoError(t, peer.Close())
				case "close":
					require.NoError(t, tts.Close(context.Background()))
				}
				select {
				case err := <-finished:
					if mode == "socket failure" {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, context.Canceled)
					}
				case <-time.After(time.Second):
					t.Fatal("writer did not stop")
				}
				assert.Less(t, time.Since(start), time.Second)
				tts.stateMu.Lock()
				assert.Nil(t, tts.connection)
				tts.stateMu.Unlock()
				require.NoError(t, tts.Close(context.Background()))
				if mode != "socket failure" || packet.PacketName() == internal_type.PacketNameTextToSpeechInterrupt {
					assert.Empty(t, elevenLabsTTSErrors(collector))
				} else {
					require.Len(t, elevenLabsTTSErrors(collector), 1)
					assert.Equal(t, "write", elevenLabsTTSErrors(collector)[0].ContextID)
				}
				assert.Empty(t, collector.EndPackets())
				assert.Empty(t, connections)
				assert.Empty(t, requests)
			})
		}
	}
}

type elevenLabsTTSCancelWriteConn struct {
	net.Conn
	cancel       context.CancelFunc
	armed        atomic.Bool
	writes       atomic.Int32
	closeEntered chan struct{}
	releaseClose chan struct{}
	closeOnce    sync.Once
}

func (connection *elevenLabsTTSCancelWriteConn) Write(payload []byte) (int, error) {
	n, err := connection.Conn.Write(payload)
	if connection.armed.Load() {
		connection.writes.Add(1)
		connection.cancel()
		if connection.closeEntered != nil {
			<-connection.closeEntered
		}
	}
	return n, err
}

func (connection *elevenLabsTTSCancelWriteConn) Close() error {
	if connection.closeEntered != nil {
		connection.closeOnce.Do(func() {
			close(connection.closeEntered)
			<-connection.releaseClose
		})
	}
	return connection.Conn.Close()
}

func TestElevenLabsTTSCancelAfterContextInit(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	dial := websocket.DefaultDialer.NetDialTLSContext
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var client *elevenLabsTTSCancelWriteConn
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		connection, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		client = &elevenLabsTTSCancelWriteConn{Conn: connection, cancel: cancel}
		return client, nil
	}
	require.NoError(t, tts.Initialize())
	waitElevenLabsServerConnection(t, connections)
	client.armed.Store(true)
	require.ErrorIs(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "canceled", Text: "must not send"}), context.Canceled)
	assert.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	assert.Equal(t, int32(1), client.writes.Load(), "cancellation after init must prevent the text write")
	require.Empty(t, elevenLabsTTSErrors(collector))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "canceled", Text: "late"}))
	assert.Empty(t, connections)
	assert.Empty(t, requests)
}

func TestElevenLabsTTSCancelBeforeContextInit(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	dial := websocket.DefaultDialer.NetDialTLSContext
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var client *elevenLabsTTSCancelWriteConn
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		client = &elevenLabsTTSCancelWriteConn{Conn: conn, cancel: cancel}
		return client, nil
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	waitElevenLabsServerConnection(t, connections)
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	client.armed.Store(true)
	require.ErrorIs(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "canceled", Text: "never sent"}), context.Canceled)
	require.Equal(t, map[string]interface{}{"context_id": "old", "close_context": true}, waitElevenLabsRequest(t, requests))
	require.Equal(t, int32(1), client.writes.Load(), "cancellation after old-context close must prevent context-init")
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "canceled", Text: "late"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "canceled"}))
	require.Empty(t, connections)
	require.Empty(t, requests)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
	waitElevenLabsServerConnection(t, connections)
	require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
	require.NoError(t, tts.Close(context.Background()))
	require.Empty(t, elevenLabsTTSErrors(collector))
}

func TestElevenLabsTTSInterruptDuringConnect(t *testing.T) {
	for _, outcome := range []string{"success", "failure"} {
		t.Run(outcome, func(t *testing.T) {
			tts, collector, connections, requests := newElevenLabsTTSServer(t)
			dial := websocket.DefaultDialer.NetDialTLSContext
			entered := make(chan struct{})
			unblock := make(chan struct{})
			release := sync.OnceFunc(func() { close(unblock) })
			var calls sync.WaitGroup
			t.Cleanup(func() {
				release()
				tts.ctxCancel()
				calls.Wait()
			})
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				close(entered)
				select {
				case <-unblock:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				if outcome == "failure" {
					return nil, errors.New("interrupted dial failed")
				}
				return dial(ctx, network, address)
			}
			finished := make(chan error, 1)
			calls.Go(func() {
				finished <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "never sent"})
			})
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("dial did not reach barrier")
			}
			interrupted := make(chan error, 1)
			calls.Go(func() {
				interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
			})
			select {
			case err := <-interrupted:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("interrupt waited for connect")
			}
			release()
			select {
			case err := <-finished:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("interrupted connect did not finish")
			}
			websocket.DefaultDialer.NetDialTLSContext = dial
			if outcome == "success" {
				waitElevenLabsServerConnection(t, connections)
			}
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.Empty(t, requests, "interrupted context must never send context-init or text")
			require.Empty(t, connections)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			if outcome == "failure" {
				waitElevenLabsServerConnection(t, connections)
			}
			require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
			require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
			require.Empty(t, connections)
			require.NoError(t, tts.Close(context.Background()))
			require.Empty(t, elevenLabsTTSErrors(collector))
		})
	}
}

func TestElevenLabsTTSReadFailureBeforeFirstText(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	var armed atomic.Bool
	armed.Store(true)
	errorDelivered := make(chan struct{})
	markError := sync.OnceFunc(func() { close(errorDelivered) })
	tts.onPacket = func(packets ...internal_type.Packet) error {
		err := collector.OnPacket(packets...)
		for _, packet := range packets {
			switch packet := packet.(type) {
			case internal_type.TextToSpeechErrorPacket:
				markError()
			case internal_type.ObservabilityMetricRecordPacket:
				if packet.Scope != internal_type.ObservabilityRecordScopeConversation || !armed.CompareAndSwap(true, false) {
					continue
				}
				remote := waitElevenLabsServerConnection(t, connections)
				require.NoError(t, remote.Close())
				select {
				case <-errorDelivered:
				case <-time.After(time.Second):
					t.Error("reader failure before context-init did not report the admitted context")
				}
			}
		}
		return err
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "never sent"}))
	require.Len(t, elevenLabsTTSErrors(collector), 1)
	require.Equal(t, "failed", elevenLabsTTSErrors(collector)[0].ContextID)
	require.Empty(t, requests)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "late"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
	require.Empty(t, connections)
	require.Empty(t, requests)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
	waitElevenLabsServerConnection(t, connections)
	require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
	require.NoError(t, tts.Close(context.Background()))
	require.Len(t, elevenLabsTTSErrors(collector), 1)
}

func TestElevenLabsTTSJoinsWriteCancellation(t *testing.T) {
	for _, kind := range []string{"init", "flush"} {
		t.Run(kind, func(t *testing.T) {
			tts, collector, connections, requests := newElevenLabsTTSServer(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &elevenLabsTTSCancelWriteConn{
				cancel: cancel, closeEntered: make(chan struct{}), releaseClose: make(chan struct{}),
			}
			release := sync.OnceFunc(func() { close(client.releaseClose) })
			var calls sync.WaitGroup
			t.Cleanup(func() {
				release()
				tts.ctxCancel()
				calls.Wait()
			})
			dial := websocket.DefaultDialer.NetDialTLSContext
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				conn, err := dial(ctx, network, address)
				if err == nil && client.Conn == nil {
					client.Conn = conn
					return client, nil
				}
				return conn, err
			}
			require.NoError(t, tts.Initialize())
			waitElevenLabsServerConnection(t, connections)
			if kind == "flush" {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				waitElevenLabsRequest(t, requests)
				waitElevenLabsRequest(t, requests)
			}
			client.armed.Store(true)
			finished := make(chan error, 1)
			calls.Go(func() {
				if kind == "init" {
					finished <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "never sent"})
				} else {
					finished <- tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}
			})
			select {
			case <-client.closeEntered:
			case <-time.After(time.Second):
				t.Fatal("cancellation did not reach close barrier")
			}
			request := waitElevenLabsRequest(t, requests)
			if kind == "init" {
				require.Contains(t, request, "voice_settings")
			} else {
				require.Equal(t, true, request["flush"])
			}
			fresh := make(chan error, 1)
			calls.Go(func() {
				fresh <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
			})
			select {
			case <-finished:
				t.Fatal("writer returned before cancellation callback completed")
			case <-fresh:
				t.Fatal("fresh writer started before cancellation callback completed")
			case <-time.After(30 * time.Millisecond):
			}
			require.Empty(t, connections)
			release()
			select {
			case err := <-finished:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("writer did not join cancellation")
			}
			select {
			case err := <-fresh:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("fresh writer did not reconnect")
			}
			waitElevenLabsServerConnection(t, connections)
			require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
			require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
			require.NoError(t, tts.Close(context.Background()))
			require.Empty(t, elevenLabsTTSErrors(collector))
			require.Empty(t, requests)
		})
	}
}

func TestElevenLabsTTSSessionCancellation(t *testing.T) {
	tts, collector, connections, _ := newElevenLabsTTSServer(t)
	require.NoError(t, tts.Initialize())
	remote := waitElevenLabsServerConnection(t, connections)
	tts.ctxCancel()
	stopped := make(chan struct{})
	go func() { tts.workers.Wait(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("session cancellation did not stop reader and keepalive")
	}
	require.Eventually(t, func() bool {
		return remote.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)) != nil
	}, time.Second, time.Millisecond)
	assert.Empty(t, elevenLabsTTSErrors(collector))
	require.ErrorIs(t, tts.Initialize(), context.Canceled)
}

func TestElevenLabsTTSWriteCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		<-ctx.Done()
	}))
	defer func() { cancel(); server.Close() }()
	previousDialer := websocket.DefaultDialer
	websocket.DefaultDialer = &websocket.Dialer{
		NetDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
		},
	}
	defer func() { websocket.DefaultDialer = previousDialer }()
	collector := testutil.NewPacketCollector()
	tts, err := NewElevenlabsTextToSpeech(ctx, testutil.NewTestLogger(),
		newVaultCredential(map[string]interface{}{"key": "local-test-key"}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	defer tts.Close(ctx)
	require.NoError(t, tts.Initialize())
	writeCtx, stopWrite := context.WithTimeout(ctx, 50*time.Millisecond)
	defer stopWrite()
	start := time.Now()
	require.ErrorIs(t, tts.Transform(writeCtx, internal_type.TextToSpeechTextPacket{ContextID: "blocked", Text: strings.Repeat("x", 16<<20)}), context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 3*time.Second)
	require.Empty(t, elevenLabsTTSErrors(collector))
	assert.Empty(t, collector.EndPackets())
	cancel()
}

type elevenLabsTTSGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	skip      int
	entered   chan struct{}
	release   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *elevenLabsTTSGatedConn) gateWrite(skip int) (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.skip = skip
	c.entered = make(chan struct{})
	c.release = make(chan struct{})
	return c.entered, c.release
}

func (c *elevenLabsTTSGatedConn) Write(payload []byte) (int, error) {
	c.gateMu.Lock()
	var entered, release chan struct{}
	if c.skip > 0 {
		c.skip--
	} else {
		entered, release = c.entered, c.release
		c.entered, c.release = nil, nil
	}
	c.gateMu.Unlock()
	if entered != nil {
		close(entered)
		select {
		case <-release:
		case <-c.closed:
			return 0, net.ErrClosed
		}
	}
	return c.Conn.Write(payload)
}

func (c *elevenLabsTTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

func TestElevenLabsTTSGatedWriteCancellation(t *testing.T) {
	for _, kind := range []string{"init", "text", "flush", "close", "interrupt", "replacement-close"} {
		for _, mode := range []string{"interrupt", "caller", "session", "close", "provider", "transport"} {
			if (kind == "close" || kind == "interrupt") && mode != "caller" {
				continue
			}
			if kind == "replacement-close" && mode != "caller" && mode != "interrupt" && mode != "transport" {
				continue
			}
			if kind == "init" && (mode == "provider" || mode == "transport") {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				tts, collector, connections, requests := newElevenLabsTTSServer(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				clients := make(chan *elevenLabsTTSGatedConn, 4)
				websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := dial(ctx, network, address)
					if err != nil {
						return nil, err
					}
					client := &elevenLabsTTSGatedConn{Conn: conn, closed: make(chan struct{})}
					clients <- client
					return client, nil
				}
				errorEntered := make(chan struct{})
				enterError := sync.OnceFunc(func() { close(errorEntered) })
				releaseError := make(chan struct{})
				release := sync.OnceFunc(func() { close(releaseError) })
				if mode == "provider" || mode == "transport" {
					tts.onPacket = func(packets ...internal_type.Packet) error {
						err := collector.OnPacket(packets...)
						for _, packet := range packets {
							if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
								enterError()
								<-releaseError
							}
						}
						return err
					}
				}
				require.NoError(t, tts.Initialize())
				remote := waitElevenLabsServerConnection(t, connections)
				client := <-clients
				if kind != "init" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
					waitElevenLabsRequest(t, requests)
					waitElevenLabsRequest(t, requests)
				}
				var calls sync.WaitGroup
				t.Cleanup(func() {
					release()
					tts.ctxCancel()
					_ = client.Close()
					calls.Wait()
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				skip := 0
				if kind == "close" {
					skip = 1
				}
				entered, _ := client.gateWrite(skip)
				finished := make(chan error, 1)
				calls.Go(func() {
					switch kind {
					case "init", "text":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
					case "flush", "close":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"})
					case "interrupt":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					case "replacement-close":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "replacement", Text: "blocked"})
					}
				})
				select {
				case <-entered:
				case <-time.After(time.Second):
					t.Fatal("write did not reach gate")
				}
				if kind == "close" {
					require.Equal(t, true, waitElevenLabsRequest(t, requests)["flush"])
				}
				closed := make(chan error, 1)
				switch mode {
				case "interrupt":
					interrupted := make(chan error, 1)
					contextID := "old"
					if kind == "replacement-close" {
						contextID = "replacement"
					}
					calls.Go(func() {
						interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: contextID})
					})
					select {
					case err := <-interrupted:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("interrupt waited for the blocked writer")
					}
				case "caller":
					cancel()
				case "session":
					tts.ctxCancel()
				case "close":
					calls.Go(func() { closed <- tts.Close(context.Background()) })
				case "provider":
					require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "old", "error": "rejected"}))
				case "transport":
					require.NoError(t, remote.Close())
				}
				if mode == "provider" || mode == "transport" {
					select {
					case <-errorEntered:
					case <-time.After(time.Second):
						t.Fatal("reader did not enter error callback")
					}
				}
				select {
				case <-client.closed:
				case <-time.After(time.Second):
					t.Fatal("socket was not closed")
				}
				select {
				case err := <-finished:
					if mode == "caller" || mode == "session" || mode == "close" {
						require.ErrorIs(t, err, context.Canceled)
					} else {
						require.NoError(t, err)
					}
				case <-time.After(time.Second):
					t.Fatal("writer did not exit before error callback release")
				}
				release()
				if mode == "close" {
					select {
					case err := <-closed:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("Close did not finish")
					}
				}
				if mode != "close" && mode != "session" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					if kind == "replacement-close" {
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "replacement", Text: "late"}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "replacement"}))
					}
					require.Empty(t, requests)
					require.Empty(t, connections)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					remote = waitElevenLabsServerConnection(t, connections)
					require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
					require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
					fresh := <-clients
					entered, unblock := fresh.gateWrite(0)
					calls.Go(func() {
						finished <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "still fresh"})
					})
					select {
					case <-entered:
					case <-time.After(time.Second):
						t.Fatal("fresh write did not reach gate")
					}
					cancel()
					stale := make(chan error, 1)
					calls.Go(func() {
						stale <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					})
					select {
					case err := <-stale:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("stale interrupt waited for fresh writer")
					}
					select {
					case <-fresh.closed:
						t.Fatal("old cancellation closed fresh socket")
					case <-finished:
						t.Fatal("old cancellation released fresh writer")
					default:
					}
					close(unblock)
					select {
					case err := <-finished:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("fresh writer did not finish")
					}
					require.Equal(t, "still fresh", waitElevenLabsRequest(t, requests)["text"])
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
					waitElevenLabsRequest(t, requests)
					waitElevenLabsRequest(t, requests)
					require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "fresh", "audio": "AQI=", "isFinal": true}))
					collector.WaitForTTSEnd(t, time.Second)
					require.Len(t, collector.AudioPackets(), 1)
					require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
					require.Len(t, collector.EndPackets(), 1)
				}
				require.NoError(t, tts.Close(context.Background()))
				if mode == "provider" || mode == "transport" {
					require.Len(t, elevenLabsTTSErrors(collector), 1)
					if kind == "replacement-close" {
						require.Equal(t, "replacement", elevenLabsTTSErrors(collector)[0].ContextID)
					} else {
						require.Equal(t, "old", elevenLabsTTSErrors(collector)[0].ContextID)
					}
				} else {
					require.Empty(t, elevenLabsTTSErrors(collector))
				}
				require.Empty(t, connections)
				require.Empty(t, requests)
			})
		}
	}
}

func TestElevenLabsTTSBlockedCallbacks(t *testing.T) {
	for _, action := range []string{"done", "interrupt", "provider-error"} {
		t.Run(action, func(t *testing.T) {
			tts, collector, connections, requests := newElevenLabsTTSServer(t)
			entered := make(chan struct{})
			enter := sync.OnceFunc(func() { close(entered) })
			unblock := make(chan struct{})
			release := sync.OnceFunc(func() { close(unblock) })
			var calls sync.WaitGroup
			t.Cleanup(func() {
				release()
				tts.ctxCancel()
				calls.Wait()
			})
			tts.onPacket = func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					switch packet.(type) {
					case internal_type.TextToSpeechAudioPacket, internal_type.TextToSpeechErrorPacket:
						enter()
						<-unblock
					}
				}
				return collector.OnPacket(packets...)
			}
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
			remote := waitElevenLabsServerConnection(t, connections)
			waitElevenLabsRequest(t, requests)
			waitElevenLabsRequest(t, requests)
			if action == "provider-error" {
				require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "old", "error": "rejected"}))
			} else {
				require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "old", "audio": "AQI="}))
			}
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("reader callback did not block")
			}
			if action == "provider-error" {
				require.Equal(t, true, waitElevenLabsRequest(t, requests)["close_context"])
			}
			finished := make(chan error, 1)
			calls.Go(func() {
				switch action {
				case "done":
					finished <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				case "interrupt":
					finished <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
				case "provider-error":
					finished <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
				}
			})
			select {
			case err := <-finished:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("input waited for the blocked callback")
			}
			waitElevenLabsRequest(t, requests)
			if action != "interrupt" {
				waitElevenLabsRequest(t, requests)
			}
			require.Empty(t, connections, "healthy idle socket must be reused")
			calls.Go(func() { finished <- tts.Close(context.Background()) })
			select {
			case <-tts.ctx.Done():
			case <-time.After(time.Second):
				t.Fatal("Close did not cancel session")
			}
			select {
			case <-finished:
				t.Fatal("Close returned before reader callback finished")
			default:
			}
			release()
			select {
			case err := <-finished:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("Close did not join released callback")
			}
			if action == "provider-error" {
				require.Len(t, elevenLabsTTSErrors(collector), 1)
			} else {
				require.Empty(t, elevenLabsTTSErrors(collector))
			}
		})
	}
}

func TestElevenLabsTTSInterruptsProviderErrorClose(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	dial := websocket.DefaultDialer.NetDialTLSContext
	clients := make(chan *elevenLabsTTSGatedConn, 4)
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		client := &elevenLabsTTSGatedConn{Conn: conn, closed: make(chan struct{})}
		clients <- client
		return client, nil
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	remote := waitElevenLabsServerConnection(t, connections)
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	client := <-clients
	var calls sync.WaitGroup
	t.Cleanup(func() {
		tts.ctxCancel()
		_ = client.Close()
		calls.Wait()
	})
	entered, _ := client.gateWrite(0)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "old", "error": "rejected"}))
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("provider-error close write did not reach gate")
	}
	finished := make(chan error, 1)
	calls.Go(func() {
		finished <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
	})
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("interrupt waited for the provider-error close write")
	}
	select {
	case <-client.closed:
	case <-time.After(time.Second):
		t.Fatal("interrupt did not close the provider-error socket")
	}
	collector.WaitFor(t, time.Second, "provider error", func() bool { return len(elevenLabsTTSErrors(collector)) == 1 })
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	require.Empty(t, requests)
	require.Empty(t, connections)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
	remote = waitElevenLabsServerConnection(t, connections)
	require.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	require.Equal(t, "fresh", waitElevenLabsRequest(t, requests)["text"])
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
	waitElevenLabsRequest(t, requests)
	waitElevenLabsRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"contextId": "fresh", "audio": "AQI=", "isFinal": true}))
	collector.WaitForTTSEnd(t, time.Second)
	require.NoError(t, tts.Close(context.Background()))
	require.Len(t, elevenLabsTTSErrors(collector), 1)
	require.Equal(t, "old", elevenLabsTTSErrors(collector)[0].ContextID)
	require.Len(t, collector.EndPackets(), 1)
	require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
}

func TestElevenLabsTTSKeepaliveAndClose(t *testing.T) {
	tts, collector, connections, requests := newElevenLabsTTSServer(t)
	require.NoError(t, tts.Initialize())
	remote := waitElevenLabsServerConnection(t, connections)
	select {
	case request := <-requests:
		assert.Equal(t, map[string]interface{}{"text": ""}, request)
	case <-time.After(12 * time.Second):
		t.Fatal("idle protocol keepalive missing")
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "hello"}))
	assert.Contains(t, waitElevenLabsRequest(t, requests), "voice_settings")
	waitElevenLabsRequest(t, requests)
	select {
	case request := <-requests:
		assert.Equal(t, map[string]interface{}{"context_id": "active", "text": ""}, request)
	case <-time.After(12 * time.Second):
		t.Fatal("context protocol keepalive missing")
	}
	require.NoError(t, tts.Close(context.Background()))
	require.NoError(t, tts.Close(context.Background()))
	require.ErrorIs(t, tts.Initialize(), context.Canceled)
	require.ErrorIs(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "closed", Text: "hello"}), context.Canceled)
	assert.Empty(t, elevenLabsTTSErrors(collector))
	assert.Empty(t, collector.EndPackets())
	closedEvents := 0
	for _, packet := range collector.EventPackets() {
		if packet.Record.Event == observability.TTSClosed {
			closedEvents++
		}
	}
	assert.Equal(t, 1, closedEvents)
	assert.Empty(t, connections)
	assert.Empty(t, requests)
	require.Eventually(t, func() bool {
		return remote.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)) != nil
	}, time.Second, time.Millisecond)
}
