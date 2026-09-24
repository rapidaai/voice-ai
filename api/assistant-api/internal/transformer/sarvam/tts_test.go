package internal_transformer_sarvam

import (
	"bufio"
	"context"
	"fmt"
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
	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type sarvamTTSPeer struct {
	conn     *websocket.Conn
	requests chan map[string]interface{}
	closed   chan struct{}
}

func newSarvamTTSSession(t *testing.T) (*sarvamTextToSpeech, *testutil.PacketCollector, <-chan *sarvamTTSPeer) {
	t.Helper()
	peers := make(chan *sarvamTTSPeer, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Api-Subscription-Key") != "local-key" || r.URL.Query().Get("send_completion_event") != "true" {
			t.Error("missing Sarvam authentication or completion setting")
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		peer := &sarvamTTSPeer{conn: conn, requests: make(chan map[string]interface{}, 128), closed: make(chan struct{})}
		defer close(peer.closed)
		peers <- peer
		for {
			var request map[string]interface{}
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			peer.requests <- request
		}
	}))
	t.Cleanup(server.Close)
	collector := testutil.NewPacketCollector()
	transformer, err := NewSarvamTextToSpeech(context.Background(), newTestLogger(),
		newVaultCredential(map[string]interface{}{"key": "local-key"}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*sarvamTextToSpeech)
	// Keep the production URL/config and redirect only the transport to a local WebSocket.
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	websocket.DefaultDialer = &dialer
	t.Cleanup(func() { websocket.DefaultDialer = previousDialer })
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
	}
	t.Cleanup(func() { require.NoError(t, tts.Close(context.Background())) })
	return tts, collector, peers
}

func nextSarvamTTSPeer(t *testing.T, peers <-chan *sarvamTTSPeer) *sarvamTTSPeer {
	t.Helper()
	select {
	case peer := <-peers:
		request := waitSarvamTTSRequest(t, peer.requests)
		require.Equal(t, "config", request["type"])
		require.Equal(t, map[string]interface{}{
			"target_language_code": "en-IN", "speaker": "anushka",
			"speech_sample_rate": float64(16000), "output_audio_codec": "linear16",
		}, request["data"])
		return peer
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Sarvam connection")
		return nil
	}
}

func writeSarvamTTSFinal(t *testing.T, peer *sarvamTTSPeer) {
	t.Helper()
	require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{
		"type": "event", "data": map[string]string{"event_type": "final", "request_id": "provider-request-not-caller-id"},
	}))
}

func TestSarvamTTSPersistentTurns(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	require.NoError(t, tts.Initialize())
	peer := nextSarvamTTSPeer(t, peers)
	for i, id := range []string{"first", "second"} {
		require.NoError(t, tts.Initialize())
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "hello"}))
		require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
		writeSarvamTTSAudio(t, peer.conn, []byte{byte(i)})
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
		require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
		require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "event", "data": map[string]string{"event_type": "ready"}}))
		writeSarvamTTSAudio(t, peer.conn, []byte{byte(i + 10)})
		collector.WaitFor(t, time.Second, "audio before final", func() bool { return len(collector.AudioPackets()) == (i+1)*2 })
		require.Len(t, collector.EndPackets(), i)
		writeSarvamTTSFinal(t, peer)
		writeSarvamTTSFinal(t, peer)
		collector.WaitFor(t, time.Second, "end", func() bool { return len(collector.EndPackets()) == i+1 })
	}
	var order []string
	latencies := 0
	for _, packet := range collector.GetPackets() {
		switch packet := packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			order = append(order, "audio:"+packet.ContextID)
		case internal_type.TextToSpeechEndPacket:
			order = append(order, "end:"+packet.ContextID)
		case internal_type.ObservabilityMetricRecordPacket:
			for _, metric := range packet.Record.Metrics {
				if metric.Name == observability.MetricTTSLatencyMs {
					latencies++
				}
			}
		}
	}
	require.Equal(t, []string{"audio:first", "audio:first", "end:first", "audio:second", "audio:second", "end:second"}, order)
	require.Equal(t, 2, latencies)
	require.Empty(t, peers)
	require.Empty(t, peer.requests)
}

func TestSarvamTTSInterruptClosesSocketAndDiscardsQueuedAudio(t *testing.T) {
	for _, done := range []bool{false, true} {
		t.Run(fmt.Sprintf("done=%t", done), func(t *testing.T) {
			tts, collector, peers := newSarvamTTSSession(t)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old text"}))
			peer := nextSarvamTTSPeer(t, peers)
			require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			if done {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
			}
			readerEntered := make(chan struct{})
			releaseReader := make(chan struct{})
			release := sync.OnceFunc(func() { close(releaseReader) })
			t.Cleanup(release)
			tts.stateMu.Lock()
			connection := tts.connection
			tts.stateMu.Unlock()
			connection.SetPingHandler(func(string) error {
				close(readerEntered)
				<-releaseReader
				return nil
			})
			require.NoError(t, peer.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
			select {
			case <-readerEntered:
			case <-time.After(time.Second):
				t.Fatal("old reader did not reach the queued audio barrier")
			}
			writeSarvamTTSAudio(t, peer.conn, []byte{1})
			writeSarvamTTSFinal(t, peer)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
			select {
			case <-peer.closed:
			case <-time.After(time.Second):
				t.Fatal("interrupt did not close the old socket")
			}
			require.Empty(t, peer.requests)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late text"}))
			result := make(chan error, 1)
			go func() {
				result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new text"})
			}()
			select {
			case err := <-result:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("new text waited for the old reader")
			}
			peer = nextSarvamTTSPeer(t, peers)
			require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			release()
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
			require.Empty(t, peer.requests)
			writeSarvamTTSAudio(t, peer.conn, []byte{2})
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
			require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
			writeSarvamTTSFinal(t, peer)
			collector.WaitForTTSEnd(t, time.Second)
			require.NoError(t, tts.Close(context.Background()))
			require.Len(t, collector.AudioPackets(), 1)
			require.Equal(t, "new", collector.AudioPackets()[0].ContextID)
			require.Equal(t, []byte{2}, collector.AudioPackets()[0].AudioChunk)
			require.Equal(t, "new", collector.EndPackets()[0].ContextID)
			require.Empty(t, peers)
		})
	}
}

func TestSarvamTTSNewContextBeforeOldInterrupt(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	peer := nextSarvamTTSPeer(t, peers)
	require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
	require.NoError(t, tts.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "turn"}))
	writeSarvamTTSAudio(t, peer.conn, []byte{1})
	collector.WaitFor(t, time.Second, "old audio", func() bool { return len(collector.AudioPackets()) == 1 })
	require.Equal(t, "old", collector.AudioPackets()[0].ContextID)
	result := make(chan error, 1)
	go func() {
		result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"})
	}()
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("replacement waited for an old final event")
	}
	select {
	case <-peer.closed:
	case <-time.After(time.Second):
		t.Fatal("replacement did not close the old socket")
	}
	require.Empty(t, peer.requests)
	peer = nextSarvamTTSPeer(t, peers)
	require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	require.Empty(t, peer.requests)
	writeSarvamTTSAudio(t, peer.conn, []byte{3})
	collector.WaitFor(t, time.Second, "new audio", func() bool { return len(collector.AudioPackets()) == 2 })
	require.Equal(t, "new", collector.AudioPackets()[1].ContextID)
	require.Equal(t, []byte{3}, collector.AudioPackets()[1].AudioChunk)
	require.Empty(t, collector.EndPackets())
}

func TestSarvamTTSFailureDoesNotReplayPartialSpeech(t *testing.T) {
	for _, failure := range []string{"drop", "provider", "json", "audio"} {
		t.Run(failure, func(t *testing.T) {
			tts, collector, peers := newSarvamTTSSession(t)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "partial"}))
			peer := nextSarvamTTSPeer(t, peers)
			require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			writeSarvamTTSAudio(t, peer.conn, []byte{1})
			collector.WaitFor(t, time.Second, "partial audio", func() bool { return len(collector.AudioPackets()) == 1 })
			switch failure {
			case "drop":
				require.NoError(t, peer.conn.Close())
			case "provider":
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "error", "data": map[string]string{"message": "rejected"}}))
			case "json":
				require.NoError(t, peer.conn.WriteMessage(websocket.TextMessage, []byte("{")))
			case "audio":
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "data": map[string]string{"audio": "!"}}))
			}
			collector.WaitFor(t, time.Second, "active error", func() bool {
				for _, packet := range collector.GetPackets() {
					if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
						return packet.ContextID == "old"
					}
				}
				return false
			})
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "must not replay"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.Empty(t, peers)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "fresh"}))
			peer = nextSarvamTTSPeer(t, peers)
			require.Equal(t, map[string]interface{}{"text": "fresh"}, waitSarvamTTSRequest(t, peer.requests)["data"])
			writeSarvamTTSAudio(t, peer.conn, []byte{2})
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
			require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
			writeSarvamTTSFinal(t, peer)
			collector.WaitForTTSEnd(t, time.Second)
			require.Len(t, collector.EndPackets(), 1)
			require.Equal(t, "new", collector.EndPackets()[0].ContextID)
		})
	}
}

func TestSarvamTTSReconnectDoesNotWaitForFinal(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	old := nextSarvamTTSPeer(t, peers)
	require.Equal(t, "text", waitSarvamTTSRequest(t, old.requests)["type"])
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
	result := make(chan error, 1)
	go func() {
		result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "fresh"})
	}()
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("replacement waited for a final event or drain timeout")
	}
	peer := nextSarvamTTSPeer(t, peers)
	require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
	select {
	case <-old.closed:
	case <-time.After(time.Second):
		t.Fatal("interrupted socket remained open")
	}
	writeSarvamTTSAudio(t, peer.conn, []byte{2})
	collector.WaitFor(t, time.Second, "fresh audio", func() bool { return len(collector.AudioPackets()) == 1 })
	require.Equal(t, "new", collector.AudioPackets()[0].ContextID)
}

func TestSarvamTTSCancelReplacementDialAndClose(t *testing.T) {
	for _, closeSession := range []bool{false, true} {
		t.Run(fmt.Sprintf("close=%t", closeSession), func(t *testing.T) {
			tts, collector, peers := newSarvamTTSSession(t)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
			peer := nextSarvamTTSPeer(t, peers)
			require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dial := websocket.DefaultDialer.NetDialTLSContext
			entered := make(chan struct{})
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
				close(entered)
				<-ctx.Done()
				return nil, ctx.Err()
			}
			result := make(chan error, 1)
			go func() {
				result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"})
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("replacement did not reach dial")
			}
			if closeSession {
				require.NoError(t, tts.Close(context.Background()))
			} else {
				cancel()
			}
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("replacement dial ignored cancellation")
			}
			select {
			case <-peer.closed:
			case <-time.After(time.Second):
				t.Fatal("replacement left the old socket open")
			}
			if closeSession {
				require.ErrorIs(t, tts.Initialize(), context.Canceled)
			} else {
				websocket.DefaultDialer.NetDialTLSContext = dial
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				nextSarvamTTSPeer(t, peers)
			}
			require.NoError(t, tts.Close(context.Background()))
			for _, packet := range collector.GetPackets() {
				_, isError := packet.(internal_type.TextToSpeechErrorPacket)
				require.False(t, isError)
			}
		})
	}
}

func TestSarvamTTSSerialInitializeAndWrites(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	var calls sync.WaitGroup
	errors := make(chan error, 32)
	for range 16 {
		calls.Add(1)
		go func() { defer calls.Done(); errors <- tts.Initialize() }()
	}
	calls.Wait()
	peer := nextSarvamTTSPeer(t, peers)
	for range 16 {
		require.NoError(t, <-errors)
		calls.Add(1)
		go func() {
			defer calls.Done()
			errors <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "one", Text: "text"})
		}()
	}
	calls.Wait()
	for range 16 {
		require.NoError(t, <-errors)
	}
	for range 16 {
		require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
	}
	require.Empty(t, peers)
	require.Empty(t, collector.EndPackets())
	require.NoError(t, tts.Close(context.Background()))
	select {
	case <-peer.closed:
	case <-time.After(time.Second):
		t.Fatal("close did not stop socket reader")
	}
}

func TestSarvamTTSCloseCancelsDial(t *testing.T) {
	tts, _, peers := newSarvamTTSSession(t)
	started := make(chan struct{})
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	result := make(chan error, 1)
	go func() { result <- tts.Initialize() }()
	<-started
	require.NoError(t, tts.Close(context.Background()))
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("dial ignored Close")
	}
	require.Empty(t, peers)
}

func TestSarvamTTSWriterChecksCancellationAfterLock(t *testing.T) {
	tts, _, _ := newSarvamTTSSession(t)
	tts.writeMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := make(chan error, 1)
	go func() {
		result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "waiting", Text: "hello"})
	}()
	tts.writeMu.Unlock()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("writer did not check cancellation after acquiring lock")
	}
}

func TestSarvamTTSDialDeadlineAndFailure(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 2*time.Second)
		return nil, fmt.Errorf("local dial failure")
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "hello"}))
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			require.Equal(t, "failed", packet.ContextID)
			require.Contains(t, packet.Error.Error(), "local dial failure")
			require.Empty(t, peers)
			return
		}
	}
	t.Fatal("dial failure did not emit an active error")
}

func TestSarvamTTSWriteHonorsCallerDeadline(t *testing.T) {
	tts, collector, _ := newSarvamTTSSession(t)
	serverConn := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			serverConn <- conn
		}
	}))
	defer server.Close()
	websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		if err == nil {
			err = conn.(*net.TCPConn).SetWriteBuffer(1024)
		}
		return conn, err
	}
	require.NoError(t, tts.Initialize())
	peer := waitSarvamTTSServerConnection(t, serverConn)
	defer peer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "blocked", Text: strings.Repeat("x", 8<<20)}), context.DeadlineExceeded)
	require.NoError(t, tts.Close(context.Background()))
	for _, packet := range collector.GetPackets() {
		_, isError := packet.(internal_type.TextToSpeechErrorPacket)
		require.False(t, isError)
	}
	require.Empty(t, collector.EndPackets())
}

type sarvamTTSPipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *sarvamTTSPipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestSarvamTTSKeepalive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()
		previousDialer := websocket.DefaultDialer
		dialer := *previousDialer
		dialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) { return client, nil }
		websocket.DefaultDialer = &dialer
		defer func() { websocket.DefaultDialer = previousDialer }()
		requests := make(chan map[string]interface{}, 16)
		go func() {
			req, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				t.Error(err)
				return
			}
			conn, err := (&websocket.Upgrader{}).Upgrade(&sarvamTTSPipeResponse{httptest.NewRecorder(), server}, req, nil)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			for {
				var request map[string]interface{}
				if err := conn.ReadJSON(&request); err != nil {
					return
				}
				requests <- request
			}
		}()
		collector := testutil.NewPacketCollector()
		tts, err := NewSarvamTextToSpeech(context.Background(), newTestLogger(),
			newVaultCredential(map[string]interface{}{"key": "local-key"}), collector.OnPacket, utils.Option{})
		require.NoError(t, err)
		defer tts.Close(context.Background())
		require.NoError(t, tts.Initialize())
		synctest.Wait()
		require.Equal(t, "config", waitSarvamTTSRequest(t, requests)["type"])
		time.Sleep(19 * time.Second)
		synctest.Wait()
		require.Empty(t, requests)
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, "ping", waitSarvamTTSRequest(t, requests)["type"])
		time.Sleep(20 * time.Second)
		synctest.Wait()
		require.Equal(t, "ping", waitSarvamTTSRequest(t, requests)["type"])
		require.Empty(t, collector.EndPackets())
		require.NoError(t, tts.Close(context.Background()))
		time.Sleep(20 * time.Second)
		synctest.Wait()
		require.Empty(t, requests)
	})
}

type sarvamTTSCancelConn struct {
	net.Conn
	armed        atomic.Bool
	cancel       context.CancelFunc
	closeStarted chan struct{}
	allowClose   chan struct{}
	closeOnce    sync.Once
}

func (c *sarvamTTSCancelConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	// Trigger after a successful WebSocket write, not the HTTP handshake.
	if err == nil && len(p) > 0 && p[0]&0x80 != 0 && c.armed.CompareAndSwap(true, false) {
		c.cancel()
		<-c.closeStarted
	}
	return n, err
}

func (c *sarvamTTSCancelConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closeStarted)
		<-c.allowClose
	})
	return c.Conn.Close()
}

func TestSarvamTTSJoinsWriteCancellationBeforeNextResponse(t *testing.T) {
	for _, boundary := range []string{"config", "text", "done", "replacement-config"} {
		t.Run(boundary, func(t *testing.T) {
			tts, collector, peers := newSarvamTTSSession(t)
			if boundary == "replacement-config" {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "previous", Text: "previous"}))
				peer := nextSarvamTTSPeer(t, peers)
				require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn := &sarvamTTSCancelConn{
				cancel: cancel, closeStarted: make(chan struct{}), allowClose: make(chan struct{}),
			}
			release := sync.OnceFunc(func() { close(conn.allowClose) })
			t.Cleanup(release)
			dial := websocket.DefaultDialer.NetDialTLSContext
			websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				socket, err := dial(ctx, network, address)
				if err == nil && conn.Conn == nil {
					conn.Conn = socket
					return conn, nil
				}
				return socket, err
			}
			var peer *sarvamTTSPeer
			var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}
			if boundary != "config" && boundary != "replacement-config" {
				require.NoError(t, tts.Initialize())
				peer = nextSarvamTTSPeer(t, peers)
			}
			if boundary == "done" {
				require.NoError(t, tts.Transform(context.Background(), packet))
				require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			}
			switch boundary {
			case "done":
				packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
			}
			conn.armed.Store(true)
			result := make(chan error, 1)
			go func() { result <- tts.Transform(ctx, packet) }()
			select {
			case <-conn.closeStarted:
			case <-time.After(time.Second):
				t.Fatal("write cancellation callback did not start")
			}
			if boundary == "config" || boundary == "replacement-config" {
				peer = nextSarvamTTSPeer(t, peers)
			} else if boundary == "text" {
				require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
			} else {
				require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
			}
			fresh := make(chan error, 1)
			go func() {
				fresh <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
			}()
			select {
			case <-result:
				t.Fatal("write returned while its cancellation callback was still running")
			case <-fresh:
				t.Fatal("fresh response started before cancellation finished")
			case <-time.After(30 * time.Millisecond):
			}
			require.Empty(t, peers)
			release()
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("write did not finish after cancellation callback was released")
			}
			select {
			case err := <-fresh:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("fresh response did not reconnect promptly")
			}
			peer = nextSarvamTTSPeer(t, peers)
			require.Equal(t, map[string]interface{}{"text": "fresh"}, waitSarvamTTSRequest(t, peer.requests)["data"])
			writeSarvamTTSAudio(t, peer.conn, []byte{1})
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
			require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
			writeSarvamTTSFinal(t, peer)
			collector.WaitForTTSEnd(t, time.Second)
			require.Len(t, collector.AudioPackets(), 1)
			require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
			require.Len(t, collector.EndPackets(), 1)
			require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
			require.NoError(t, tts.Close(context.Background()))
			for _, packet := range collector.GetPackets() {
				_, isError := packet.(internal_type.TextToSpeechErrorPacket)
				require.False(t, isError)
			}
		})
	}
}

type sarvamTTSGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
	fail      atomic.Bool
}

func (c *sarvamTTSGatedConn) gateNextWrite() (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.entered = make(chan struct{})
	c.release = make(chan struct{})
	return c.entered, c.release
}

func (c *sarvamTTSGatedConn) Write(payload []byte) (int, error) {
	c.gateMu.Lock()
	entered, release := c.entered, c.release
	c.entered, c.release = nil, nil
	c.gateMu.Unlock()
	if entered != nil {
		close(entered)
		select {
		case <-release:
		case <-c.closed:
			return 0, net.ErrClosed
		}
	}
	if c.fail.Load() {
		return 0, fmt.Errorf("injected write failure")
	}
	return c.Conn.Write(payload)
}

func (c *sarvamTTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

func TestSarvamTTSGatedWriteCancellation(t *testing.T) {
	for _, kind := range []string{"text", "done"} {
		for _, mode := range []string{"interrupt", "caller", "session", "close", "provider", "transport", "write failure"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				tts, collector, peers := newSarvamTTSSession(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				clients := make(chan *sarvamTTSGatedConn, 4)
				websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := dial(ctx, network, address)
					if err != nil {
						return nil, err
					}
					client := &sarvamTTSGatedConn{Conn: conn, closed: make(chan struct{})}
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
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				peer := nextSarvamTTSPeer(t, peers)
				require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
				client := <-clients
				var calls sync.WaitGroup
				t.Cleanup(func() {
					release()
					tts.ctxCancel()
					_ = client.Close()
					calls.Wait()
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered, unblock := client.gateNextWrite()
				finished := make(chan error, 1)
				calls.Go(func() {
					switch kind {
					case "text":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
					case "done":
						finished <- tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"})
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
					calls.Go(func() {
						interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					})
					select {
					case err := <-interrupted:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("interrupt waited behind the blocked writer")
					}
				case "caller":
					cancel()
				case "session":
					tts.ctxCancel()
				case "close":
					calls.Go(func() { closed <- tts.Close(context.Background()) })
				case "provider":
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "error", "data": map[string]string{"message": "rejected"}}))
				case "transport":
					require.NoError(t, peer.conn.Close())
				case "write failure":
					client.fail.Store(true)
					close(unblock)
				}
				if mode == "provider" || mode == "transport" {
					select {
					case <-errorEntered:
					case <-time.After(time.Second):
						t.Fatal("reader did not enter error callback")
					}
				}
				select {
				case <-peer.closed:
				case <-time.After(time.Second):
					t.Fatal("peer did not observe socket closure")
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
					require.Empty(t, peers, "old input must not reconnect")
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh text"}))
					peer = nextSarvamTTSPeer(t, peers)
					require.Equal(t, map[string]interface{}{"text": "fresh text"}, waitSarvamTTSRequest(t, peer.requests)["data"])
					fresh := <-clients
					entered, unblock := fresh.gateNextWrite()
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
						t.Fatal("stale interrupt waited for the new writer")
					}
					select {
					case <-fresh.closed:
						t.Fatal("old cancellation closed the fresh socket")
					case <-finished:
						t.Fatal("old cancellation released the fresh writer")
					default:
					}
					close(unblock)
					select {
					case err := <-finished:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("fresh writer did not finish")
					}
					require.Equal(t, map[string]interface{}{"text": "still fresh"}, waitSarvamTTSRequest(t, peer.requests)["data"])
					writeSarvamTTSAudio(t, peer.conn, []byte{2})
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
					require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
					writeSarvamTTSFinal(t, peer)
					collector.WaitForTTSEnd(t, time.Second)
					require.Len(t, collector.AudioPackets(), 1)
					require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
					require.Len(t, collector.EndPackets(), 1)
					require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				}
				require.NoError(t, tts.Close(context.Background()))
				errorCount := 0
				for _, packet := range collector.GetPackets() {
					if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
						errorCount++
						require.Equal(t, "old", packet.ContextID)
					}
				}
				if mode == "provider" || mode == "transport" || mode == "write failure" {
					require.Equal(t, 1, errorCount)
				} else {
					require.Zero(t, errorCount)
				}
				require.Empty(t, peers)
			})
		}
	}
}

func TestSarvamTTSBlockedAudioDoesNotBlockDone(t *testing.T) {
	tts, collector, peers := newSarvamTTSSession(t)
	entered := make(chan struct{})
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
			if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
				close(entered)
				<-unblock
			}
		}
		return collector.OnPacket(packets...)
	}
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "hello"}))
	peer := nextSarvamTTSPeer(t, peers)
	require.Equal(t, "text", waitSarvamTTSRequest(t, peer.requests)["type"])
	writeSarvamTTSAudio(t, peer.conn, []byte{1})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("audio callback did not block")
	}
	finished := make(chan error, 1)
	calls.Go(func() {
		finished <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"})
	})
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Done waited for the blocked audio callback")
	}
	require.Equal(t, "flush", waitSarvamTTSRequest(t, peer.requests)["type"])
	calls.Go(func() { finished <- tts.Close(context.Background()) })
	select {
	case <-peer.closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not close the socket while audio callback was blocked")
	}
	select {
	case <-finished:
		t.Fatal("Close returned before its reader callback finished")
	default:
	}
	release()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Close did not join the released callback")
	}
	for _, packet := range collector.GetPackets() {
		_, isError := packet.(internal_type.TextToSpeechErrorPacket)
		require.False(t, isError)
	}
}

func TestSarvamTTSInterruptDuringConnect(t *testing.T) {
	for _, dialFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("dialFails=%t", dialFails), func(t *testing.T) {
			tts, collector, peers := newSarvamTTSSession(t)
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
				if dialFails {
					return nil, fmt.Errorf("interrupted dial failed")
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
			var peer *sarvamTTSPeer
			if !dialFails {
				peer = nextSarvamTTSPeer(t, peers)
				require.Empty(t, peer.requests, "interrupted connect must not send old text")
			}
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.Empty(t, peers)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			if dialFails {
				peer = nextSarvamTTSPeer(t, peers)
			}
			require.Equal(t, map[string]interface{}{"text": "fresh"}, waitSarvamTTSRequest(t, peer.requests)["data"])
			require.Empty(t, peers)
			require.NoError(t, tts.Close(context.Background()))
			for _, packet := range collector.GetPackets() {
				_, isError := packet.(internal_type.TextToSpeechErrorPacket)
				require.False(t, isError)
			}
		})
	}
}
