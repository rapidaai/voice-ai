package internal_transformer_cartesia

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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

func newCartesiaTTSSession(t *testing.T) (*cartesiaTTS, *testutil.PacketCollector, <-chan *websocket.Conn, <-chan map[string]interface{}) {
	t.Helper()
	connections := make(chan *websocket.Conn, 32)
	requests := make(chan map[string]interface{}, 128)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer connection.Close()
		connections <- connection
		for {
			var request map[string]interface{}
			if err := connection.ReadJSON(&request); err != nil {
				return
			}
			requests <- request
		}
	}))
	t.Cleanup(server.Close)
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	websocket.DefaultDialer = &dialer
	t.Cleanup(func() { websocket.DefaultDialer = previousDialer })
	collector := testutil.NewPacketCollector()
	transformer, err := NewCartesiaTextToSpeech(context.Background(), newTestLogger(),
		newVaultCredential(map[string]interface{}{"key": "local-test"}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, transformer.Close(context.Background())) })
	return transformer.(*cartesiaTTS), collector, connections, requests
}

func cartesiaTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var packets []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			packets = append(packets, failure)
		}
	}
	return packets
}

func TestCartesiaTTSPersistentConnection(t *testing.T) {
	transformer, collector, connections, requests := newCartesiaTTSSession(t)
	ctx := context.Background()
	require.NoError(t, transformer.Initialize())
	remote := waitCartesiaTTSServerConnection(t, connections)
	for index, contextID := range []string{"first", "second"} {
		require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "hello"}))
		request := waitCartesiaTTSRequest(t, requests)
		assert.Equal(t, contextID, request["context_id"])
		assert.Equal(t, true, request["continue"])
		require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		request = waitCartesiaTTSRequest(t, requests)
		assert.Equal(t, false, request["continue"])
		require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "flush_done", "done": true, "context_id": contextID}))
		require.NoError(t, remote.WriteJSON(map[string]interface{}{
			"type": "chunk", "context_id": contextID, "data": base64.StdEncoding.EncodeToString([]byte{1, 2}),
		}))
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "context_id": contextID}))
		collector.WaitFor(t, time.Second, "synthesis end", func() bool { return len(collector.EndPackets()) == index+1 })
		require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "done": true, "context_id": contextID}))
		require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "late"}))
	}
	assert.Empty(t, requests)
	assert.Empty(t, connections, "completion must reuse the same socket")
	assert.Empty(t, cartesiaTTSErrors(collector))
	require.Len(t, collector.AudioPackets(), 2)
	require.Len(t, collector.EndPackets(), 2)
	assert.Equal(t, "first", collector.EndPackets()[0].ContextID)
	assert.Equal(t, "second", collector.EndPackets()[1].ContextID)
	var ordered []string
	for _, packet := range collector.GetPackets() {
		switch packet := packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			ordered = append(ordered, "audio:"+packet.ContextID)
		case internal_type.TextToSpeechEndPacket:
			ordered = append(ordered, "end:"+packet.ContextID)
		case internal_type.PlaybackCompletedPacket:
			t.Fatal("provider completion must not manufacture a playback receipt")
		}
	}
	assert.Equal(t, []string{"audio:first", "end:first", "audio:second", "end:second"}, ordered)
	var initMetrics, latencyMetrics int
	for _, packet := range collector.MetricPackets() {
		for _, metric := range packet.Record.Metrics {
			switch metric.Name {
			case observability.MetricTTSInitLatencyMs:
				initMetrics++
			case observability.MetricTTSLatencyMs:
				latencyMetrics++
			}
		}
	}
	assert.Equal(t, 1, initMetrics)
	assert.Equal(t, 2, latencyMetrics)
}

func TestCartesiaTTSInterruptContextOrdering(t *testing.T) {
	for _, order := range []string{"interrupt before text", "text before interrupt"} {
		t.Run(order, func(t *testing.T) {
			transformer, collector, connections, requests := newCartesiaTTSSession(t)
			ctx := context.Background()
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
			remote := waitCartesiaTTSServerConnection(t, connections)
			assert.Equal(t, "first", waitCartesiaTTSRequest(t, requests)["transcript"])
			require.NoError(t, transformer.Transform(ctx, internal_type.TurnChangePacket{ContextID: "user-turn", PreviousContextID: "old"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "still speaking"}))
			assert.Equal(t, "still speaking", waitCartesiaTTSRequest(t, requests)["transcript"])
			if order == "interrupt before text" {
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				assert.Equal(t, map[string]interface{}{"context_id": "old", "cancel": true}, waitCartesiaTTSRequest(t, requests))
			}
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "next"}))
			if order == "text before interrupt" {
				assert.Equal(t, map[string]interface{}{"context_id": "old", "cancel": true}, waitCartesiaTTSRequest(t, requests))
			}
			assert.Equal(t, "next", waitCartesiaTTSRequest(t, requests)["transcript"])
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late text"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			for _, contextID := range []string{"old", "unknown", ""} {
				for _, kind := range []string{"chunk", "done", "error"} {
					require.NoError(t, remote.WriteJSON(map[string]interface{}{
						"type": kind, "context_id": contextID, "done": kind == "done",
						"data": base64.StdEncoding.EncodeToString([]byte{9}),
					}))
				}
			}
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "new"}))
			assert.Equal(t, "new", waitCartesiaTTSRequest(t, requests)["context_id"])
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "chunk", "context_id": "new", "data": base64.StdEncoding.EncodeToString([]byte{2}),
			}))
			require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "done": true, "context_id": "new"}))
			collector.WaitForTTSEnd(t, time.Second)
			require.Len(t, collector.AudioPackets(), 1)
			assert.Equal(t, "new", collector.AudioPackets()[0].ContextID)
			require.Len(t, collector.EndPackets(), 1)
			assert.Equal(t, "new", collector.EndPackets()[0].ContextID)
			assert.Empty(t, cartesiaTTSErrors(collector))
			assert.Empty(t, connections)
			assert.Empty(t, requests, "late interrupts and retired text must not send anything")
		})
	}
}

func TestCartesiaTTSInterruptBeforeFirstText(t *testing.T) {
	for _, state := range []string{"idle", "active"} {
		t.Run(state, func(t *testing.T) {
			transformer, collector, connections, requests := newCartesiaTTSSession(t)
			ctx := context.Background()
			if state == "active" {
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "live", Text: "first"}))
				waitCartesiaTTSServerConnection(t, connections)
				waitCartesiaTTSRequest(t, requests)
			}
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "retired"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "retired", Text: "late"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "retired"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "live", Text: "next"}))
			request := waitCartesiaTTSRequest(t, requests)
			assert.Equal(t, "live", request["context_id"])
			assert.Equal(t, "next", request["transcript"])
			if state == "idle" {
				waitCartesiaTTSServerConnection(t, connections)
			}
			assert.Empty(t, requests)
			assert.Empty(t, connections)
			assert.Empty(t, cartesiaTTSErrors(collector))
		})
	}
}

type cartesiaTTSDelayedCancelConn struct {
	net.Conn
	cancel        context.CancelFunc
	armed         atomic.Bool
	delayClose    atomic.Bool
	closeStarted  chan struct{}
	releaseClose  chan struct{}
	writeFinished chan struct{}
}

func (connection *cartesiaTTSDelayedCancelConn) Write(payload []byte) (int, error) {
	n, err := connection.Conn.Write(payload)
	if err == nil && connection.armed.CompareAndSwap(true, false) {
		connection.cancel()
		<-connection.closeStarted
		close(connection.writeFinished)
	}
	return n, err
}

func (connection *cartesiaTTSDelayedCancelConn) Close() error {
	if connection.delayClose.CompareAndSwap(true, false) {
		close(connection.closeStarted)
		<-connection.releaseClose
	}
	return connection.Conn.Close()
}

func TestCartesiaTTSJoinsWriteCancellation(t *testing.T) {
	for _, kind := range []string{"text", "done", "interrupt", "replacement"} {
		t.Run(kind, func(t *testing.T) {
			transformer, collector, connections, requests := newCartesiaTTSSession(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dial := websocket.DefaultDialer.NetDialTLSContext
			clients := make(chan *cartesiaTTSDelayedCancelConn, 4)
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				connection, err := dial(ctx, network, address)
				if err != nil {
					return nil, err
				}
				client := &cartesiaTTSDelayedCancelConn{
					Conn: connection, cancel: cancel, closeStarted: make(chan struct{}),
					releaseClose: make(chan struct{}), writeFinished: make(chan struct{}),
				}
				clients <- client
				return client, nil
			}
			require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
			waitCartesiaTTSServerConnection(t, connections)
			waitCartesiaTTSRequest(t, requests)
			client := <-clients
			release := sync.OnceFunc(func() { close(client.releaseClose) })
			t.Cleanup(release)
			client.delayClose.Store(true)
			client.armed.Store(true)
			finished := make(chan error, 1)
			go func() {
				switch kind {
				case "text":
					finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "second"})
				case "done":
					finished <- transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"})
				case "interrupt":
					finished <- transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
				case "replacement":
					finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "replacement", Text: "second"})
				}
			}()
			select {
			case <-client.writeFinished:
			case <-time.After(time.Second):
				t.Fatal("write did not reach the delayed cancellation barrier")
			}
			waitCartesiaTTSRequest(t, requests)
			select {
			case <-finished:
				t.Fatal("Transform returned before its cancellation callback completed")
			case <-time.After(50 * time.Millisecond):
			}
			release()
			select {
			case err := <-finished:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("Transform did not join cancellation")
			}
			transformer.stateMu.Lock()
			connection := transformer.connection
			transformer.stateMu.Unlock()
			assert.Nil(t, connection, "cancellation must detach the socket before the next text")
			require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
			require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"}))
			remote := waitCartesiaTTSServerConnection(t, connections)
			assert.Equal(t, "fresh text", waitCartesiaTTSRequest(t, requests)["transcript"])
			require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
			waitCartesiaTTSRequest(t, requests)
			require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "chunk", "context_id": "fresh", "data": "AQI="}))
			require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "context_id": "fresh"}))
			collector.WaitForTTSEnd(t, time.Second)
			require.NoError(t, transformer.Close(context.Background()))
			require.Len(t, collector.EndPackets(), 1)
			assert.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
			assert.Empty(t, cartesiaTTSErrors(collector))
			assert.Empty(t, requests)
			assert.Empty(t, connections)
		})
	}
}

func TestCartesiaTTSConcurrentInitializeAndReconnect(t *testing.T) {
	transformer, collector, connections, requests := newCartesiaTTSSession(t)
	for phase := range 2 {
		contextID := fmt.Sprintf("context-%d", phase)
		var workers sync.WaitGroup
		for range 16 {
			workers.Add(1)
			go func() {
				defer workers.Done()
				assert.NoError(t, transformer.Initialize())
				assert.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "hello"}))
			}()
		}
		workers.Wait()
		remote := waitCartesiaTTSServerConnection(t, connections)
		for range 16 {
			assert.Equal(t, contextID, waitCartesiaTTSRequest(t, requests)["context_id"])
		}
		assert.Empty(t, connections, "concurrent initialization must dial once")
		require.NoError(t, remote.Close())
		collector.WaitFor(t, time.Second, "connection error", func() bool { return len(cartesiaTTSErrors(collector)) == phase+1 })
		assert.Equal(t, contextID, cartesiaTTSErrors(collector)[phase].ContextID)
		require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "no replay"}))
		require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		assert.Empty(t, connections, "failed text must not reconnect or replay")
		assert.Empty(t, requests)
	}
	assert.Empty(t, collector.EndPackets())
}

type cartesiaTTSWriteFailureConn struct {
	net.Conn
	fail atomic.Bool
}

func (connection *cartesiaTTSWriteFailureConn) Write(payload []byte) (int, error) {
	if connection.fail.Load() {
		return 0, errors.New("injected write failure")
	}
	return connection.Conn.Write(payload)
}

func TestCartesiaTTSWriteFailure(t *testing.T) {
	for _, kind := range []string{"text", "done", "interrupt", "replacement"} {
		t.Run(kind, func(t *testing.T) {
			transformer, collector, connections, requests := newCartesiaTTSSession(t)
			dial := websocket.DefaultDialer.NetDialTLSContext
			var client *cartesiaTTSWriteFailureConn
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				connection, err := dial(ctx, network, address)
				if err != nil {
					return nil, err
				}
				client = &cartesiaTTSWriteFailureConn{Conn: connection}
				return client, nil
			}
			ctx := context.Background()
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "first"}))
			remote := waitCartesiaTTSServerConnection(t, connections)
			waitCartesiaTTSRequest(t, requests)
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "chunk", "context_id": "failed", "data": base64.StdEncoding.EncodeToString([]byte{1}),
			}))
			collector.WaitForAudio(t, time.Second)
			client.fail.Store(true)
			switch kind {
			case "text":
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "second"}))
			case "done":
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
			case "interrupt":
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "failed"}))
			case "replacement":
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"}))
			}
			require.Len(t, cartesiaTTSErrors(collector), 1, "send failure must be reported before Transform returns")
			assert.Equal(t, "failed", cartesiaTTSErrors(collector)[0].ContextID)
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "late"}))
			require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
			if kind != "replacement" {
				assert.Empty(t, connections)
				require.NoError(t, transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"}))
			}
			waitCartesiaTTSServerConnection(t, connections)
			assert.Equal(t, "fresh text", waitCartesiaTTSRequest(t, requests)["transcript"])
			require.NoError(t, transformer.Close(ctx))
			assert.Len(t, cartesiaTTSErrors(collector), 1, "reader exit must not duplicate the writer error")
			assert.Empty(t, collector.EndPackets())
		})
	}
}

func TestCartesiaTTSDialCancellation(t *testing.T) {
	for _, mode := range []string{"deadline", "caller cancellation", "close"} {
		t.Run(mode, func(t *testing.T) {
			transformer, collector, _, _ := newCartesiaTTSSession(t)
			entered := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				select {
				case <-release:
				case <-r.Context().Done():
				}
			}))
			t.Cleanup(server.Close)
			t.Cleanup(func() { close(release) })
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan error, 1)
			started := time.Now()
			go func() {
				finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "dial", Text: "hello"})
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("dial did not reach HTTP upgrade")
			}
			switch mode {
			case "caller cancellation":
				cancel()
			case "close":
				require.NoError(t, transformer.Close(context.Background()))
			}
			select {
			case err := <-finished:
				if mode == "deadline" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, context.Canceled)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("dial did not stop")
			}
			if mode == "deadline" {
				assert.Less(t, time.Since(started), 3*time.Second)
			} else {
				assert.Less(t, time.Since(started), time.Second)
			}
			if mode != "deadline" {
				assert.Empty(t, cartesiaTTSErrors(collector))
				if mode == "close" {
					require.Error(t, transformer.Initialize())
				}
			} else {
				require.Len(t, cartesiaTTSErrors(collector), 1)
				assert.Equal(t, "dial", cartesiaTTSErrors(collector)[0].ContextID)
				require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "dial", Text: "no retry"}))
				assert.Len(t, cartesiaTTSErrors(collector), 1)
			}
		})
	}
}

type cartesiaTTSStalledWriteConn struct {
	net.Conn
	stalled net.Conn
	started chan struct{}
	block   atomic.Bool
}

func (connection *cartesiaTTSStalledWriteConn) Write(payload []byte) (int, error) {
	if connection.block.Load() {
		close(connection.started)
		return connection.stalled.Write(payload)
	}
	return connection.Conn.Write(payload)
}

func (connection *cartesiaTTSStalledWriteConn) SetWriteDeadline(deadline time.Time) error {
	_ = connection.stalled.SetWriteDeadline(deadline)
	return connection.Conn.SetWriteDeadline(deadline)
}

func (connection *cartesiaTTSStalledWriteConn) Close() error {
	_ = connection.stalled.Close()
	return connection.Conn.Close()
}

func TestCartesiaTTSWriteCancellation(t *testing.T) {
	for _, mode := range []string{"caller cancellation", "close"} {
		t.Run(mode, func(t *testing.T) {
			transformer, collector, connections, requests := newCartesiaTTSSession(t)
			dial := websocket.DefaultDialer.NetDialTLSContext
			stalled, peer := net.Pipe()
			t.Cleanup(func() { _ = stalled.Close(); _ = peer.Close() })
			var client *cartesiaTTSStalledWriteConn
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				connection, err := dial(ctx, network, address)
				if err != nil {
					return nil, err
				}
				client = &cartesiaTTSStalledWriteConn{Conn: connection, stalled: stalled, started: make(chan struct{})}
				return client, nil
			}
			require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "write", Text: "first"}))
			waitCartesiaTTSServerConnection(t, connections)
			waitCartesiaTTSRequest(t, requests)
			client.block.Store(true)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan error, 1)
			started := time.Now()
			go func() {
				finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "write", Text: "second"})
			}()
			select {
			case <-client.started:
			case <-time.After(time.Second):
				t.Fatal("writer did not block")
			}
			switch mode {
			case "caller cancellation":
				cancel()
			case "close":
				require.NoError(t, transformer.Close(context.Background()))
			}
			select {
			case err := <-finished:
				if mode == "caller cancellation" {
					require.ErrorIs(t, err, context.Canceled)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("writer did not stop")
			}
			assert.Less(t, time.Since(started), time.Second)
			require.NoError(t, transformer.Close(context.Background()))
			assert.Empty(t, cartesiaTTSErrors(collector))
			assert.Empty(t, collector.EndPackets())
			assert.Empty(t, connections)
		})
	}
}

type cartesiaTTSGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func (connection *cartesiaTTSGatedConn) gateNextWrite() (<-chan struct{}, chan<- struct{}) {
	connection.gateMu.Lock()
	defer connection.gateMu.Unlock()
	connection.entered = make(chan struct{})
	connection.release = make(chan struct{})
	return connection.entered, connection.release
}

func (connection *cartesiaTTSGatedConn) Write(payload []byte) (int, error) {
	connection.gateMu.Lock()
	entered, release := connection.entered, connection.release
	connection.entered, connection.release = nil, nil
	connection.gateMu.Unlock()
	if entered != nil {
		close(entered)
		select {
		case <-release:
		case <-connection.closed:
			return 0, net.ErrClosed
		}
	}
	return connection.Conn.Write(payload)
}

func (connection *cartesiaTTSGatedConn) Close() error {
	err := connection.Conn.Close()
	connection.closeOnce.Do(func() { close(connection.closed) })
	return err
}

func TestCartesiaTTSGatedWriteCancellation(t *testing.T) {
	for _, kind := range []string{"text", "done", "interrupt", "replacement"} {
		for _, mode := range []string{"interrupt", "caller", "session", "close", "provider error", "transport error"} {
			if (kind == "interrupt" || kind == "replacement") && mode != "caller" && mode != "transport error" {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				transformer, collector, connections, requests := newCartesiaTTSSession(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				clients := make(chan *cartesiaTTSGatedConn, 4)
				websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					connection, err := dial(ctx, network, address)
					if err != nil {
						return nil, err
					}
					client := &cartesiaTTSGatedConn{Conn: connection, closed: make(chan struct{})}
					clients <- client
					return client, nil
				}
				errorEntered := make(chan struct{})
				enterError := sync.OnceFunc(func() { close(errorEntered) })
				releaseError := make(chan struct{})
				release := sync.OnceFunc(func() { close(releaseError) })
				if mode == "provider error" || mode == "transport error" {
					transformer.onPacket = func(packets ...internal_type.Packet) error {
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
				require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				remote := waitCartesiaTTSServerConnection(t, connections)
				waitCartesiaTTSRequest(t, requests)
				client := <-clients
				var workers sync.WaitGroup
				t.Cleanup(func() {
					release()
					transformer.ctxCancel()
					_ = client.Close()
					workers.Wait()
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered, _ := client.gateNextWrite()
				finished := make(chan error, 1)
				workers.Go(func() {
					switch kind {
					case "text":
						finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
					case "done":
						finished <- transformer.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"})
					case "interrupt":
						finished <- transformer.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					case "replacement":
						finished <- transformer.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "replacement", Text: "blocked"})
					}
				})
				select {
				case <-entered:
				case <-time.After(time.Second):
					t.Fatal("write did not reach gate")
				}
				closed := make(chan error, 1)
				switch mode {
				case "interrupt":
					interrupted := make(chan error, 1)
					workers.Go(func() {
						interrupted <- transformer.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					})
					select {
					case err := <-interrupted:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("interrupt waited for the stalled writer")
					}
				case "caller":
					cancel()
				case "session":
					transformer.ctxCancel()
				case "close":
					workers.Go(func() { closed <- transformer.Close(context.Background()) })
				case "provider error":
					require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "error", "context_id": "old", "status_code": 500}))
				case "transport error":
					require.NoError(t, remote.Close())
				}
				if mode == "provider error" || mode == "transport error" {
					select {
					case <-errorEntered:
					case <-time.After(time.Second):
						t.Fatal("failure did not enter error callback")
					}
				}
				select {
				case <-client.closed:
				case <-time.After(time.Second):
					t.Fatal("socket was not closed while the write was gated")
				}
				select {
				case err := <-finished:
					if mode == "caller" {
						require.ErrorIs(t, err, context.Canceled)
					} else {
						require.NoError(t, err)
					}
				case <-time.After(time.Second):
					t.Fatal("stalled Transform did not exit before callback release")
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
				if mode != "session" && mode != "close" {
					if kind == "replacement" && mode == "transport error" {
						remote = waitCartesiaTTSServerConnection(t, connections)
						request := waitCartesiaTTSRequest(t, requests)
						assert.Equal(t, "replacement", request["context_id"])
						assert.Equal(t, "blocked", request["transcript"])
					}
					require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					assert.Empty(t, connections, "retired input must not reconnect")
					assert.Empty(t, requests)
					require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"}))
					if kind == "replacement" && mode == "transport error" {
						assert.Equal(t, map[string]interface{}{"context_id": "replacement", "cancel": true}, waitCartesiaTTSRequest(t, requests))
					} else {
						remote = waitCartesiaTTSServerConnection(t, connections)
					}
					assert.Equal(t, "fresh text", waitCartesiaTTSRequest(t, requests)["transcript"])
					fresh := <-clients
					entered, unblock := fresh.gateNextWrite()
					workers.Go(func() {
						finished <- transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "still fresh"})
					})
					select {
					case <-entered:
					case <-time.After(time.Second):
						t.Fatal("replacement write did not reach gate")
					}
					stale := make(chan error, 1)
					workers.Go(func() {
						stale <- transformer.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					})
					select {
					case err := <-stale:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("stale interrupt waited for the new writer")
					}
					select {
					case <-fresh.closed:
						t.Fatal("stale interrupt closed the replacement socket")
					case <-finished:
						t.Fatal("stale interrupt released the new writer")
					default:
					}
					close(unblock)
					select {
					case err := <-finished:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("replacement writer did not finish")
					}
					assert.Equal(t, "still fresh", waitCartesiaTTSRequest(t, requests)["transcript"])
					require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
					waitCartesiaTTSRequest(t, requests)
					require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "chunk", "context_id": "fresh", "data": "AQI="}))
					require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "context_id": "fresh"}))
					collector.WaitForTTSEnd(t, time.Second)
					require.Len(t, collector.EndPackets(), 1)
					assert.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				}
				require.NoError(t, transformer.Close(context.Background()))
				if mode == "provider error" || mode == "transport error" {
					require.Len(t, cartesiaTTSErrors(collector), 1)
					assert.Equal(t, "old", cartesiaTTSErrors(collector)[0].ContextID)
				} else {
					assert.Empty(t, cartesiaTTSErrors(collector))
				}
				assert.Empty(t, connections)
				assert.Empty(t, requests)
			})
		}
	}
}

func TestCartesiaTTSBlockedAudioDoesNotBlockDone(t *testing.T) {
	transformer, collector, connections, requests := newCartesiaTTSSession(t)
	entered := make(chan struct{})
	unblock := make(chan struct{})
	release := sync.OnceFunc(func() { close(unblock) })
	var workers sync.WaitGroup
	t.Cleanup(func() {
		release()
		transformer.ctxCancel()
		workers.Wait()
	})
	transformer.onPacket = func(packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
				close(entered)
				<-unblock
			}
		}
		return collector.OnPacket(packets...)
	}
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "hello"}))
	remote := waitCartesiaTTSServerConnection(t, connections)
	waitCartesiaTTSRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "chunk", "context_id": "active", "data": "AQI="}))
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("audio callback did not block")
	}
	finished := make(chan error, 1)
	workers.Go(func() {
		finished <- transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"})
	})
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Done waited for the blocked audio callback")
	}
	assert.Equal(t, false, waitCartesiaTTSRequest(t, requests)["continue"])
	workers.Go(func() { finished <- transformer.Close(context.Background()) })
	select {
	case <-transformer.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Close did not cancel the session")
	}
	select {
	case <-finished:
		t.Fatal("Close returned before the reader callback finished")
	default:
	}
	release()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Close did not join the released callback")
	}
	assert.Empty(t, cartesiaTTSErrors(collector))
}

func TestCartesiaTTSBlockedProviderErrorReusesIdleSocket(t *testing.T) {
	transformer, collector, connections, requests := newCartesiaTTSSession(t)
	entered := make(chan struct{})
	enter := sync.OnceFunc(func() { close(entered) })
	unblock := make(chan struct{})
	release := sync.OnceFunc(func() { close(unblock) })
	var workers sync.WaitGroup
	t.Cleanup(func() {
		release()
		transformer.ctxCancel()
		workers.Wait()
	})
	transformer.onPacket = func(packets ...internal_type.Packet) error {
		err := collector.OnPacket(packets...)
		for _, packet := range packets {
			if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
				enter()
				<-unblock
			}
		}
		return err
	}
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
	remote := waitCartesiaTTSServerConnection(t, connections)
	waitCartesiaTTSRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "error", "context_id": "old", "status_code": 500}))
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("provider error callback did not block")
	}
	finished := make(chan error, 1)
	workers.Go(func() {
		finished <- transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"})
	})
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("new input waited for the blocked error callback")
	}
	assert.Equal(t, "fresh text", waitCartesiaTTSRequest(t, requests)["transcript"])
	assert.Empty(t, connections, "idle provider error must preserve the healthy socket")
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
	waitCartesiaTTSRequest(t, requests)
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "error", "context_id": "old", "status_code": 500}))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "context_id": "old"}))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "chunk", "context_id": "fresh", "data": "AQI="}))
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "context_id": "fresh"}))
	release()
	collector.WaitForTTSEnd(t, time.Second)
	require.NoError(t, transformer.Close(context.Background()))
	require.Len(t, cartesiaTTSErrors(collector), 1)
	assert.Equal(t, "old", cartesiaTTSErrors(collector)[0].ContextID)
	require.Len(t, collector.EndPackets(), 1)
	assert.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
	require.Len(t, collector.AudioPackets(), 1)
	assert.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
}

func TestCartesiaTTSIdleDisconnectAndClose(t *testing.T) {
	transformer, collector, connections, requests := newCartesiaTTSSession(t)
	require.NoError(t, transformer.Initialize())
	remote := waitCartesiaTTSServerConnection(t, connections)
	require.NoError(t, remote.Close())
	require.Eventually(t, func() bool {
		transformer.stateMu.Lock()
		defer transformer.stateMu.Unlock()
		return transformer.connection == nil
	}, time.Second, time.Millisecond)
	assert.Empty(t, cartesiaTTSErrors(collector))
	assert.Empty(t, connections)
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "hello"}))
	waitCartesiaTTSServerConnection(t, connections)
	assert.Equal(t, "hello", waitCartesiaTTSRequest(t, requests)["transcript"])
	require.NoError(t, transformer.Close(context.Background()))
	require.NoError(t, transformer.Close(context.Background()))
	require.Error(t, transformer.Initialize())
	require.Error(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "closed", Text: "hello"}))
	assert.Empty(t, cartesiaTTSErrors(collector))
	var closed, usage int
	for _, packet := range collector.GetPackets() {
		switch packet := packet.(type) {
		case internal_type.ObservabilityEventRecordPacket:
			if packet.Record.Event == observability.TTSClosed {
				closed++
			}
		case internal_type.ObservabilityUsageRecordPacket:
			usage++
		}
	}
	assert.Equal(t, 1, closed)
	assert.Equal(t, 1, usage)
}

func TestCartesiaTTSReadLoopConnectionOwnership(t *testing.T) {
	for _, state := range []string{"current", "interrupted", "replaced"} {
		t.Run(state, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			serverConnections := make(chan *websocket.Conn, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connection, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade websocket: %v", err)
					return
				}
				serverConnections <- connection
			}))
			t.Cleanup(server.Close)
			connectionURL := "ws" + strings.TrimPrefix(server.URL, "http")
			connection, _, err := websocket.DefaultDialer.Dial(connectionURL, nil)
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			remote := waitCartesiaTTSServerConnection(t, serverConnections)
			t.Cleanup(func() { _ = remote.Close() })

			ctx, cancel := context.WithCancel(context.Background())
			collector := testutil.NewPacketCollector()
			transformer := &cartesiaTTS{
				ctx:          ctx,
				ctxCancel:    cancel,
				connection:   connection,
				contextId:    "original-message",
				ttsStartedAt: time.Now(),
				textClosed:   true,
				retired:      make(map[string]struct{}),
				logger:       newTestLogger(),
				onPacket:     collector.OnPacket,
			}
			readerReady := make(chan struct{})
			releaseReader := make(chan struct{})
			release := sync.OnceFunc(func() { close(releaseReader) })
			// Hold the real reader while audio queues and connection ownership changes.
			connection.SetPingHandler(func(string) error {
				close(readerReady)
				<-releaseReader
				return nil
			})
			readerDone := make(chan struct{})
			go func() {
				defer close(readerDone)
				transformer.readLoop(connection)
			}()
			t.Cleanup(func() {
				cancel()
				release()
				_ = connection.Close()
				select {
				case <-readerDone:
				case <-time.After(3 * time.Second):
					t.Error("TTS reader did not exit")
				}
			})
			require.NoError(t, remote.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
			select {
			case <-readerReady:
			case <-time.After(3 * time.Second):
				t.Fatal("TTS reader did not reach the queued frame barrier")
			}
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "chunk", "context_id": "original-message",
				"data": base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
			}))
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "done", "context_id": "original-message", "done": true,
			}))
			require.NoError(t, remote.Close())

			switch state {
			case "interrupted":
				transformer.stateMu.Lock()
				transformer.connection = nil
				transformer.retired[transformer.contextId] = struct{}{}
				transformer.contextId = ""
				transformer.stateMu.Unlock()
			case "replaced":
				replacement, _, err := websocket.DefaultDialer.Dial(connectionURL, nil)
				require.NoError(t, err)
				t.Cleanup(func() { _ = replacement.Close() })
				replacementRemote := waitCartesiaTTSServerConnection(t, serverConnections)
				t.Cleanup(func() { _ = replacementRemote.Close() })
				transformer.stateMu.Lock()
				transformer.connection = replacement
				transformer.retired[transformer.contextId] = struct{}{}
				transformer.contextId = "replacement-message"
				transformer.textClosed = false
				transformer.ttsStartedAt = time.Now()
				transformer.stateMu.Unlock()
			}
			release()
			select {
			case <-readerDone:
			case <-time.After(3 * time.Second):
				t.Fatal("TTS reader did not finish queued frames")
			}

			if state != "current" {
				assert.Empty(t, collector.AudioPackets(), "obsolete connection must not emit audio")
				assert.Empty(t, collector.MetricPackets(), "obsolete connection must not emit latency")
				assert.Empty(t, collector.EndPackets(), "obsolete connection must not finish the replacement message")
				return
			}
			require.Len(t, collector.AudioPackets(), 1)
			assert.Equal(t, "original-message", collector.AudioPackets()[0].ContextID)
			assert.Equal(t, []byte{1, 2, 3, 4}, collector.AudioPackets()[0].AudioChunk)
			require.Len(t, collector.MetricPackets(), 1)
			assert.Equal(t, "original-message", collector.MetricPackets()[0].ContextID)
			require.Len(t, collector.EndPackets(), 1)
			assert.Equal(t, "original-message", collector.EndPackets()[0].ContextID)
		})
	}
}
