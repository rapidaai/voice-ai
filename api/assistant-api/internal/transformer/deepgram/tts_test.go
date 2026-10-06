package internal_transformer_deepgram

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type deepgramTTSGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *deepgramTTSGatedConn) gateNextWrite() (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.entered = make(chan struct{})
	c.release = make(chan struct{})
	return c.entered, c.release
}

func (c *deepgramTTSGatedConn) Write(p []byte) (int, error) {
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
	return c.Conn.Write(p)
}

func (c *deepgramTTSGatedConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func TestDeepgramTTSStalledWriteCancellation(t *testing.T) {
	for _, operation := range []string{"Speak", "Flush", "Clear interrupt", "Clear replacement"} {
		for _, cancellation := range []string{"interrupt", "caller", "close", "session", "provider error"} {
			if operation != "Speak" && operation != "Flush" && cancellation != "caller" {
				continue
			}
			t.Run(operation+"/"+cancellation, func(t *testing.T) {
				peers := make(chan *websocket.Conn, 4)
				transports := make(chan *deepgramTTSGatedConn, 4)
				upgrader := websocket.Upgrader{}
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					peer, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						t.Errorf("upgrade: %v", err)
						return
					}
					peers <- peer
				}))
				t.Cleanup(server.Close)
				previousDialer := websocket.DefaultDialer
				dialer := *previousDialer
				dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := (&tls.Dialer{
						Config: server.Client().Transport.(*http.Transport).TLSClientConfig,
					}).DialContext(ctx, network, address)
					if err != nil {
						return nil, err
					}
					transport := &deepgramTTSGatedConn{Conn: conn, closed: make(chan struct{})}
					t.Cleanup(func() { _ = transport.Close() })
					transports <- transport
					return transport, nil
				}
				websocket.DefaultDialer = &dialer
				t.Cleanup(func() { websocket.DefaultDialer = previousDialer })
				var calls sync.WaitGroup
				t.Cleanup(calls.Wait)
				sessionCtx, cancelSession := context.WithCancel(t.Context())
				t.Cleanup(cancelSession)
				packets := make(chan internal_type.Packet, 16)
				providerErrors := make(chan internal_type.TextToSpeechErrorPacket, 16)
				releaseErrorCallback := make(chan struct{})
				var errors atomic.Int32
				transformer, err := NewDeepgramTextToSpeech(sessionCtx, testutil.NewTestLogger(), newVaultCredential(map[string]interface{}{
					"key": "test", "endpoint": strings.TrimPrefix(server.URL, "https://"),
				}), func(emitted ...internal_type.Packet) error {
					for _, packet := range emitted {
						switch packet := packet.(type) {
						case internal_type.TextToSpeechErrorPacket:
							errors.Add(1)
							if cancellation == "provider error" {
								providerErrors <- packet
								<-releaseErrorCallback
							}
						case internal_type.TextToSpeechAudioPacket, internal_type.TextToSpeechEndPacket:
							packets <- packet
						}
					}
					return nil
				}, utils.Option{})
				require.NoError(t, err)
				provider := transformer.(*deepgramTTS)
				t.Cleanup(func() { _ = provider.Close(t.Context()) })
				t.Cleanup(func() {
					select {
					case <-releaseErrorCallback:
					default:
						close(releaseErrorCallback)
					}
				})
				require.NoError(t, provider.Initialize())
				transport := <-transports
				peer := <-peers
				t.Cleanup(func() { _ = peer.Close() })
				require.NoError(t, peer.SetReadDeadline(time.Now().Add(3*time.Second)))
				var request map[string]string
				var input internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "stalled text"}
				if operation != "Speak" {
					require.NoError(t, provider.Transform(t.Context(), input))
					require.NoError(t, peer.ReadJSON(&request))
					require.Equal(t, "Speak", request["type"])
				}
				switch operation {
				case "Flush":
					input = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				case "Clear interrupt":
					input = internal_type.TextToSpeechInterruptPacket{ContextID: "old"}
				case "Clear replacement":
					input = internal_type.TextToSpeechTextPacket{ContextID: "replacement", Text: "replaces old speech"}
				}
				entered, _ := transport.gateNextWrite()
				writeCtx, cancelWrite := context.WithCancel(t.Context())
				t.Cleanup(cancelWrite)
				pending := make(chan error, 1)
				calls.Go(func() { pending <- provider.Transform(writeCtx, input) })
				select {
				case <-entered:
				case <-time.After(time.Second):
					t.Fatal("write did not reach the transport gate")
				}
				select {
				case err := <-pending:
					t.Fatalf("gated write returned before cancellation: %v", err)
				default:
				}
				switch cancellation {
				case "interrupt":
					interrupted := make(chan error, 1)
					calls.Go(func() {
						interrupted <- provider.Transform(t.Context(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
					})
					select {
					case err := <-interrupted:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("matching interrupt waited behind the stalled write")
					}
				case "caller":
					cancelWrite()
				case "close":
					closed := make(chan error, 1)
					calls.Go(func() { closed <- provider.Close(t.Context()) })
					select {
					case err := <-closed:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("Close did not unblock the stalled write and join readers")
					}
				case "session":
					cancelSession()
				case "provider error":
					require.NoError(t, peer.WriteJSON(map[string]string{
						"type": "Error", "code": "TEST_PROVIDER_ERROR", "description": "synthesis rejected",
					}))
					select {
					case failure := <-providerErrors:
						require.Equal(t, "old", failure.ContextID)
						require.ErrorContains(t, failure.Error, "TEST_PROVIDER_ERROR")
					case <-time.After(time.Second):
						t.Fatal("provider error callback did not start")
					}
					require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
					_, _, err := peer.ReadMessage()
					require.True(t, websocket.IsCloseError(err, websocket.CloseAbnormalClosure),
						"failed socket must close before its error callback returns: %v", err)
				}
				select {
				case err := <-pending:
					if cancellation == "caller" {
						require.ErrorIs(t, err, context.Canceled)
					} else if cancellation == "interrupt" || cancellation == "provider error" {
						require.NoError(t, err)
					} else if err != nil {
						require.ErrorIs(t, err, context.Canceled)
					}
				case <-time.After(time.Second):
					t.Fatal("cancellation did not release the stalled write")
				}
				if cancellation == "provider error" {
					require.EqualValues(t, 1, errors.Load(), "writer must not duplicate the blocked provider error")
					close(releaseErrorCallback)
				}
				select {
				case <-transport.closed:
				default:
					t.Fatal("cancelled write left its transport open")
				}
				if cancellation == "close" || cancellation == "session" {
					require.NoError(t, provider.Close(t.Context()))
					require.ErrorIs(t, provider.Initialize(), context.Canceled)
					require.Zero(t, errors.Load(), "session shutdown must not emit a TTS failure")
					return
				}
				if operation == "Speak" || operation == "Flush" {
					require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "must not restart"}))
					require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					require.Empty(t, transports, "repeated interrupt must not permit same-context reconnection")
					require.Empty(t, peers, "retired input must not open another socket")
				}
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "fresh text"}))
				nextTransport := <-transports
				nextPeer := <-peers
				t.Cleanup(func() { _ = nextPeer.Close() })
				require.NotSame(t, transport, nextTransport)
				require.NoError(t, nextPeer.SetReadDeadline(time.Now().Add(3*time.Second)))
				require.NoError(t, nextPeer.ReadJSON(&request))
				require.Equal(t, "Speak", request["type"])
				require.Equal(t, "fresh text", request["text"])
				var nextInput internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "more text"}
				if operation == "Flush" {
					nextInput = internal_type.TextToSpeechDonePacket{ContextID: "new"}
				}
				nextEntered, release := nextTransport.gateNextWrite()
				nextPending := make(chan error, 1)
				calls.Go(func() { nextPending <- provider.Transform(t.Context(), nextInput) })
				select {
				case <-nextEntered:
				case <-time.After(time.Second):
					t.Fatal("new response write did not reach the transport gate")
				}
				staleInterrupt := make(chan error, 1)
				calls.Go(func() {
					staleInterrupt <- provider.Transform(t.Context(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
				})
				select {
				case err := <-staleInterrupt:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("stale interrupt waited behind a newer write")
				}
				select {
				case <-nextTransport.closed:
					t.Fatal("stale interrupt closed the newer transport")
				case err := <-nextPending:
					t.Fatalf("stale interrupt released the newer stalled write: %v", err)
				default:
				}
				close(release)
				select {
				case err := <-nextPending:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("released newer write did not complete")
				}
				require.NoError(t, nextPeer.ReadJSON(&request))
				if operation == "Flush" {
					require.Equal(t, "Flush", request["type"])
				} else {
					require.Equal(t, "Speak", request["type"])
					require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
					require.NoError(t, nextPeer.ReadJSON(&request))
					require.Equal(t, "Flush", request["type"])
				}
				require.NoError(t, nextPeer.WriteMessage(websocket.BinaryMessage, []byte{3, 4}))
				require.NoError(t, nextPeer.WriteJSON(map[string]string{"type": "Flushed"}))
				for _, expected := range []internal_type.Packet{
					internal_type.TextToSpeechAudioPacket{ContextID: "new", AudioChunk: []byte{3, 4}},
					internal_type.TextToSpeechEndPacket{ContextID: "new"},
				} {
					select {
					case packet := <-packets:
						require.Equal(t, expected, packet)
					case <-time.After(time.Second):
						t.Fatal("new response output missing")
					}
				}
				require.NoError(t, provider.Close(t.Context()))
				if cancellation == "provider error" {
					require.EqualValues(t, 1, errors.Load(), "provider failure must emit exactly one error")
					require.Empty(t, providerErrors)
				} else {
					require.Zero(t, errors.Load(), "cancelled synthesis must not emit a TTS failure")
				}
			})
		}
	}
}

func TestDeepgramTTSSessionCancellationDuringDial(t *testing.T) {
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	websocket.DefaultDialer = &dialer
	t.Cleanup(func() { websocket.DefaultDialer = previousDialer })
	for range 32 {
		sessionCtx, cancelSession := context.WithCancel(t.Context())
		t.Cleanup(cancelSession)
		dialed := false
		dialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
			dialed = true
			cancelSession()
			return nil, net.ErrClosed
		}
		var errors atomic.Int32
		transformer, err := NewDeepgramTextToSpeech(sessionCtx, testutil.NewTestLogger(), newVaultCredential(map[string]interface{}{
			"key": "test",
		}), func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
					errors.Add(1)
				}
			}
			return nil
		}, utils.Option{})
		require.NoError(t, err)
		provider := transformer.(*deepgramTTS)
		t.Cleanup(func() { _ = provider.Close(t.Context()) })
		require.ErrorIs(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{
			ContextID: "cancelled", Text: "never sent",
		}), context.Canceled)
		require.True(t, dialed, "session must be cancelled during the dial, not before admission")
		require.NoError(t, provider.Close(t.Context()))
		require.Zero(t, errors.Load(), "session cancellation must not become a synthesis error")
		require.Nil(t, provider.connection)
	}
}

func TestDeepgramTTSPersistentConnection(t *testing.T) {
	for _, scenario := range []string{"completed response", "interrupted response", "new text before old interrupt", "connection failure", "server failure", "close during clear", "close after connection failure"} {
		t.Run(scenario, func(t *testing.T) {
			peers := make(chan *websocket.Conn, 16)
			var connections atomic.Int32
			upgrader := websocket.Upgrader{}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connection, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade: %v", err)
					return
				}
				connections.Add(1)
				peers <- connection
			}))
			defer server.Close()
			dialer := *websocket.DefaultDialer
			previousDialer := websocket.DefaultDialer
			dialer.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig
			websocket.DefaultDialer = &dialer
			defer func() { websocket.DefaultDialer = previousDialer }()
			packets := make(chan internal_type.Packet, 16)
			var latencyMetrics atomic.Int32
			transformer, err := NewDeepgramTextToSpeech(t.Context(), testutil.NewTestLogger(), newVaultCredential(map[string]interface{}{
				"key": "test", "endpoint": strings.TrimPrefix(server.URL, "https://"),
			}), func(emitted ...internal_type.Packet) error {
				for _, packet := range emitted {
					switch packet := packet.(type) {
					case internal_type.TextToSpeechAudioPacket, internal_type.TextToSpeechEndPacket, internal_type.TextToSpeechErrorPacket:
						packets <- packet
					case internal_type.ObservabilityMetricRecordPacket:
						if packet.ContextID != "" {
							latencyMetrics.Add(1)
						}
					}
				}
				return nil
			}, utils.Option{})
			require.NoError(t, err)
			provider := transformer.(*deepgramTTS)
			defer provider.Close(t.Context())
			var initializers sync.WaitGroup
			initializationErrors := make(chan error, 8)
			for range 8 {
				initializers.Go(func() { initializationErrors <- provider.Initialize() })
			}
			initializers.Wait()
			for range 8 {
				require.NoError(t, <-initializationErrors)
			}
			require.EqualValues(t, 1, connections.Load())
			peer := <-peers
			defer peer.Close()
			require.NoError(t, peer.SetReadDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "first", Text: "Hello"}))
			var request map[string]interface{}
			require.NoError(t, peer.ReadJSON(&request))
			require.Equal(t, "Speak", request["type"])
			for range 2 {
				require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte{0, 1}))
				select {
				case packet := <-packets:
					require.Equal(t, internal_type.TextToSpeechAudioPacket{ContextID: "first", AudioChunk: []byte{0, 1}}, packet)
				case <-time.After(3 * time.Second):
					t.Fatal("streamed audio missing")
				}
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "first", Text: "more"}))
				require.NoError(t, peer.ReadJSON(&request))
				require.Equal(t, "Speak", request["type"])
			}
			require.EqualValues(t, 1, latencyMetrics.Load(), "later text chunks must not restart first-audio metrics")
			require.NoError(t, provider.Transform(t.Context(), internal_type.TurnChangePacket{ContextID: "second", PreviousContextID: "first"}))

			switch scenario {
			case "completed response":
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "first"}))
				require.NoError(t, peer.ReadJSON(&request))
				require.Equal(t, "Flush", request["type"])
				require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte{1, 2}))
				require.NoError(t, peer.WriteJSON(map[string]string{"type": "Flushed"}))
				select {
				case packet := <-packets:
					require.Equal(t, internal_type.TextToSpeechAudioPacket{ContextID: "first", AudioChunk: []byte{1, 2}}, packet)
				case <-time.After(3 * time.Second):
					t.Fatal("first audio missing")
				}
				select {
				case packet := <-packets:
					require.Equal(t, internal_type.TextToSpeechEndPacket{ContextID: "first"}, packet)
				case <-time.After(3 * time.Second):
					t.Fatal("first completion missing")
				}
			case "connection failure", "server failure", "close after connection failure":
				if scenario == "server failure" {
					require.NoError(t, peer.WriteJSON(map[string]string{"type": "Error", "code": "INVALID_TEXT", "description": "cannot synthesize"}))
				} else {
					require.NoError(t, peer.Close())
				}
				select {
				case packet := <-packets:
					failure, ok := packet.(internal_type.TextToSpeechErrorPacket)
					require.True(t, ok)
					require.Equal(t, "first", failure.ContextID)
				case <-time.After(3 * time.Second):
					t.Fatal("connection failure missing")
				}
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "first", Text: "must not replay"}))
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "first"}))
				require.EqualValues(t, 1, connections.Load())
				if scenario == "close after connection failure" {
					closed := make(chan error, 1)
					go func() { closed <- provider.Close(t.Context()) }()
					select {
					case err := <-closed:
						require.NoError(t, err)
					case <-time.After(3 * time.Second):
						t.Fatal("close did not finish after connection failure")
					}
					require.ErrorIs(t, provider.Initialize(), context.Canceled)
					return
				}
			default:
				if scenario != "new text before old interrupt" {
					require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechInterruptPacket{ContextID: "first"}))
				}
				pending := make(chan error, 1)
				go func() {
					pending <- provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "second", Text: "Next"})
				}()
				require.NoError(t, peer.ReadJSON(&request))
				require.Equal(t, "Clear", request["type"])
				if scenario == "close during clear" {
					require.NoError(t, provider.Close(t.Context()))
					select {
					case err := <-pending:
						require.ErrorIs(t, err, context.Canceled)
					case <-time.After(3 * time.Second):
						t.Fatal("close did not release pending synthesis")
					}
					require.ErrorIs(t, provider.Initialize(), context.Canceled)
					return
				}
				require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte{9, 9}))
				require.NoError(t, peer.WriteJSON(map[string]string{"type": "Flushed"}))
				require.NoError(t, peer.WriteJSON(map[string]string{"type": "Cleared"}))
				select {
				case err := <-pending:
					require.NoError(t, err)
				case <-time.After(3 * time.Second):
					t.Fatal("Clear acknowledgement did not release new synthesis")
				}
				require.Empty(t, packets, "interrupted audio and completion must be discarded")
			}
			if scenario == "completed response" || scenario == "connection failure" || scenario == "server failure" {
				require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "second", Text: "Next"}))
			}
			if scenario == "connection failure" || scenario == "server failure" {
				peer = <-peers
				defer peer.Close()
				require.NoError(t, peer.SetReadDeadline(time.Now().Add(5*time.Second)))
				require.EqualValues(t, 2, connections.Load())
			} else {
				require.EqualValues(t, 1, connections.Load())
			}
			require.NoError(t, peer.ReadJSON(&request))
			require.Equal(t, "Speak", request["type"])
			require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechInterruptPacket{ContextID: "first"}))
			require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "second"}))
			require.NoError(t, peer.ReadJSON(&request))
			require.Equal(t, "Flush", request["type"], "late old interrupt must not clear new speech")
			require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte{3, 4}))
			require.NoError(t, peer.WriteJSON(map[string]string{"type": "Flushed"}))
			select {
			case packet := <-packets:
				require.Equal(t, internal_type.TextToSpeechAudioPacket{ContextID: "second", AudioChunk: []byte{3, 4}}, packet)
			case <-time.After(3 * time.Second):
				t.Fatal("second audio missing")
			}
			select {
			case packet := <-packets:
				require.Equal(t, internal_type.TextToSpeechEndPacket{ContextID: "second"}, packet)
			case <-time.After(3 * time.Second):
				t.Fatal("second completion missing")
			}
			require.EqualValues(t, 2, latencyMetrics.Load(), "each response emits exactly one first-audio metric")
			require.NoError(t, provider.Close(t.Context()))
			require.ErrorIs(t, provider.Initialize(), context.Canceled)
		})
	}
}

func TestDeepgramTTSDiscardsObsoleteConnectionOutput(t *testing.T) {
	upgrader := websocket.Upgrader{}
	peers := make(chan *websocket.Conn, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		peers <- connection
	}))
	defer server.Close()
	oldConnection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer oldConnection.Close()
	oldPeer := <-peers
	defer oldPeer.Close()
	currentConnection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer currentConnection.Close()
	currentPeer := <-peers
	defer currentPeer.Close()
	var packets []internal_type.Packet
	provider := &deepgramTTS{
		ctx: t.Context(), contextId: "unclear-prompt", connection: currentConnection,
		logger:   testutil.NewTestLogger(),
		onPacket: func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil },
	}
	done := make(chan struct{})
	go func() { defer close(done); provider.readLoop(oldConnection) }()
	require.NoError(t, oldPeer.WriteMessage(websocket.BinaryMessage, []byte{0, 0}))
	require.NoError(t, oldPeer.WriteJSON(map[string]string{"type": "Flushed"}))
	require.NoError(t, oldPeer.Close())
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = oldConnection.Close()
		<-done
		t.Fatal("old reader did not exit")
	}
	require.Empty(t, packets, "obsolete connection must not emit audio or completion for the unclear prompt")
	require.Same(t, currentConnection, provider.connection, "obsolete Flushed must not detach the current connection")
}

func TestDeepgramTTSDoesNotReplayFailedConnection(t *testing.T) {
	provider := &deepgramTTS{ctx: t.Context(), contextId: "failed"}
	require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "do not retry"}))
	require.NoError(t, provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "failed"}))
	require.Nil(t, provider.connection)
}

func TestDeepgramTTSParentCancellationClosesInstalledConnection(t *testing.T) {
	peers := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		peers <- connection
	}))
	defer server.Close()
	dialer := *websocket.DefaultDialer
	previousDialer := websocket.DefaultDialer
	dialer.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig
	websocket.DefaultDialer = &dialer
	defer func() { websocket.DefaultDialer = previousDialer }()
	for range 32 {
		ctx, cancel := context.WithCancel(t.Context())
		transformer, err := NewDeepgramTextToSpeech(ctx, testutil.NewTestLogger(), newVaultCredential(map[string]interface{}{
			"key": "test", "endpoint": strings.TrimPrefix(server.URL, "https://"),
		}), func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
					cancel()
				}
			}
			return nil
		}, utils.Option{})
		require.NoError(t, err)
		provider := transformer.(*deepgramTTS)
		t.Cleanup(func() { _ = provider.Close(t.Context()) })
		require.NoError(t, provider.Initialize())
		peer := <-peers
		t.Cleanup(func() { _ = peer.Close() })
		require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
		_, _, err = peer.ReadMessage()
		require.True(t, websocket.IsCloseError(err, websocket.CloseAbnormalClosure),
			"parent cancellation did not close the installed socket: %v", err)
		require.ErrorIs(t, provider.Initialize(), context.Canceled)
	}
}

func TestDeepgramTTSBlockedAudioDoesNotBlockInputClosure(t *testing.T) {
	upgrader := websocket.Upgrader{}
	peers := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		peers <- connection
	}))
	defer server.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	peer := <-peers
	defer peer.Close()
	ctx, cancel := context.WithCancel(t.Context())
	blocked := make(chan struct{})
	release := make(chan struct{})
	provider := &deepgramTTS{
		ctx: ctx, ctxCancel: cancel, contextId: "active", connection: connection,
		logger: testutil.NewTestLogger(),
		onPacket: func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
					close(blocked)
					<-release
				}
			}
			return nil
		},
	}
	defer provider.Close(t.Context())
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	provider.readers.Go(func() { provider.readLoop(connection) })
	require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte{1, 2}))
	select {
	case <-blocked:
	case <-time.After(3 * time.Second):
		t.Fatal("audio callback did not start")
	}
	done := make(chan error, 1)
	go func() {
		done <- provider.Transform(t.Context(), internal_type.TextToSpeechDonePacket{ContextID: "active"})
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("blocked output must not hold the synthesis state lock")
	}
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
	var request map[string]string
	require.NoError(t, peer.ReadJSON(&request))
	require.Equal(t, "Flush", request["type"])
	closed := make(chan error, 1)
	go func() { closed <- provider.Close(t.Context()) }()
	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("close did not cancel the reader")
	}
	select {
	case <-closed:
		t.Fatal("close returned before the audio callback finished")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("close did not finish after the audio callback returned")
	}
}
