package internal_transformer_rime

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type rimeTTSPipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *rimeTTSPipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

type rimeTTSWriteGate struct {
	entered chan struct{}
	result  chan error
}

type rimeTTSGatedConn struct {
	net.Conn
	gateMu      sync.Mutex
	writeGate   *rimeTTSWriteGate
	readEntered chan struct{}
	readRelease chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func (c *rimeTTSGatedConn) gateWrite() *rimeTTSWriteGate {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.writeGate = &rimeTTSWriteGate{entered: make(chan struct{}), result: make(chan error, 1)}
	return c.writeGate
}

func (c *rimeTTSGatedConn) gateRead() (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.readEntered, c.readRelease = make(chan struct{}), make(chan struct{})
	return c.readEntered, c.readRelease
}

func (c *rimeTTSGatedConn) Read(payload []byte) (int, error) {
	count, err := c.Conn.Read(payload)
	c.gateMu.Lock()
	entered, release := c.readEntered, c.readRelease
	c.readEntered, c.readRelease = nil, nil
	c.gateMu.Unlock()
	if entered != nil {
		close(entered)
		<-release
	}
	return count, err
}

func (c *rimeTTSGatedConn) Write(payload []byte) (int, error) {
	c.gateMu.Lock()
	gate := c.writeGate
	c.writeGate = nil
	c.gateMu.Unlock()
	if gate != nil {
		close(gate.entered)
		select {
		case err := <-gate.result:
			if err != nil {
				return 0, err
			}
		case <-c.closed:
			return 0, net.ErrClosed
		}
	}
	return c.Conn.Write(payload)
}

func (c *rimeTTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

type rimeTTSPeer struct {
	conn      *websocket.Conn
	transport *rimeTTSGatedConn
	request   *http.Request
	messages  chan map[string]interface{}
	closed    chan struct{}
}

func newRimeTTSSession(t *testing.T) (*rimeTTS, *testutil.PacketCollector, <-chan *rimeTTSPeer) {
	t.Helper()
	peers := make(chan *rimeTTSPeer, 16)
	var servers sync.WaitGroup
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		transport := &rimeTTSGatedConn{Conn: client, closed: make(chan struct{})}
		servers.Go(func() {
			defer server.Close()
			request, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				return
			}
			response := &rimeTTSPipeResponse{ResponseRecorder: httptest.NewRecorder(), conn: server}
			conn, err := (&websocket.Upgrader{}).Upgrade(response, request, nil)
			if err != nil {
				t.Errorf("upgrade websocket: %v", err)
				return
			}
			peer := &rimeTTSPeer{conn: conn, transport: transport, request: request, messages: make(chan map[string]interface{}, 32), closed: make(chan struct{})}
			defer close(peer.closed)
			peers <- peer
			for {
				var message map[string]interface{}
				if err := conn.ReadJSON(&message); err != nil {
					return
				}
				peer.messages <- message
			}
		})
		return transport, nil
	}
	websocket.DefaultDialer = &dialer
	collector := testutil.NewPacketCollector()
	transformer, err := NewRimeTextToSpeech(context.Background(), newTestLogger(), newVaultCredential(map[string]interface{}{"key": "test-key"}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*rimeTTS)
	t.Cleanup(func() {
		require.NoError(t, tts.Close(context.Background()))
		servers.Wait()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, peers
}

func rimeTTSReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Rime TTS test event")
		var zero T
		return zero
	}
}

func rimeTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var packets []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			packets = append(packets, packet)
		}
	}
	return packets
}

func TestRimeTTSSequentialResponsesAndStaleControl(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newRimeTTSSession(t)
		require.NoError(t, tts.Initialize())
		peer := rimeTTSReceive(t, peers)
		require.NoError(t, tts.Initialize())
		require.Empty(t, peers)
		require.Equal(t, "Bearer test-key", peer.request.Header.Get("Authorization"))
		require.Equal(t, "/ws3", peer.request.URL.Path)
		for index, id := range []string{"old", "fresh"} {
			for _, text := range []string{"hello", "more"} {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: text}))
				if index == 1 && text == "hello" {
					peer = rimeTTSReceive(t, peers)
				}
				require.Equal(t, map[string]interface{}{"text": text}, rimeTTSReceive(t, peer.messages))
			}
			for _, stale := range []string{"", "stale"} {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: stale}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: stale}))
			}
			require.Empty(t, peer.messages)
			require.Empty(t, peers)
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "done"}))
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AQI="}))
			synctest.Wait()
			require.Len(t, collector.EndPackets(), index)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
			require.Equal(t, map[string]interface{}{"operation": "eos"}, rimeTTSReceive(t, peer.messages))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "late"}))
			require.Empty(t, peer.messages)
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "done"}))
			synctest.Wait()
			require.Len(t, collector.EndPackets(), index, "a late per-batch done must not terminate EOS")
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AwQ="}))
			require.NoError(t, peer.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
			synctest.Wait()
			require.Len(t, collector.AudioPackets(), (index+1)*2)
			require.Equal(t, id, collector.AudioPackets()[index*2+1].ContextID)
			require.Len(t, collector.EndPackets(), index+1)
			require.Equal(t, id, collector.EndPackets()[index].ContextID)
			rimeTTSReceive(t, peer.closed)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: id}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "retired"}))
			require.Empty(t, peers)
		}
		require.Empty(t, rimeTTSErrors(collector))
	})
}

func TestRimeTTSCleanCloseWaitsForEOSResult(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newRimeTTSSession(t)
				require.NoError(t, tts.Initialize())
				peer := rimeTTSReceive(t, peers)
				// Suppress the automatic close reply so it cannot serialize behind the gated EOS.
				tts.connection.SetCloseHandler(func(int, string) error { return nil })
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				rimeTTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}()
				<-gate.entered
				require.NoError(t, peer.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
				synctest.Wait()
				require.Empty(t, collector.EndPackets())
				require.Empty(t, result)
				if failed {
					gate.result <- errors.New("EOS failed")
				} else {
					gate.result <- nil
				}
				require.NoError(t, <-result)
				synctest.Wait()
				if failed {
					require.Empty(t, collector.EndPackets())
					require.Len(t, rimeTTSErrors(collector), 1)
				} else {
					require.Len(t, collector.EndPackets(), 1)
					require.Empty(t, rimeTTSErrors(collector))
				}
			})
		})
	}
}

func TestRimeTTSGatedWriteCancellation(t *testing.T) {
	for _, command := range []string{"text", "eos"} {
		for _, cancellation := range []string{"interrupt", "caller", "session", "close", "replacement"} {
			t.Run(command+"/"+cancellation, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					tts, collector, peers := newRimeTTSSession(t)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
					peer := rimeTTSReceive(t, peers)
					rimeTTSReceive(t, peer.messages)
					gate := peer.transport.gateWrite()
					var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
					if command == "eos" {
						packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					result := make(chan error, 1)
					go func() { result <- tts.Transform(ctx, packet) }()
					<-gate.entered
					synctest.Wait()
					require.Empty(t, result)
					switch cancellation {
					case "interrupt":
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					case "caller":
						cancel()
					case "session":
						tts.ctxCancel()
					case "close":
						require.NoError(t, tts.Close(context.Background()))
					case "replacement":
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					}
					err := <-result
					if cancellation == "interrupt" || cancellation == "replacement" {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, context.Canceled)
					}
					rimeTTSReceive(t, peer.closed)
					require.Empty(t, rimeTTSErrors(collector))
					require.Empty(t, collector.EndPackets())
					if cancellation == "session" || cancellation == "close" {
						require.ErrorIs(t, tts.Initialize(), context.Canceled)
						return
					}
					if cancellation != "replacement" {
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
						require.Empty(t, peers)
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					}
					fresh := rimeTTSReceive(t, peers)
					require.Equal(t, "fresh", rimeTTSReceive(t, fresh.messages)["text"])
					gate = fresh.transport.gateWrite()
					go func() {
						result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "blocked again"})
					}()
					<-gate.entered
					cancel()
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					synctest.Wait()
					require.Empty(t, result, "late cancellation must not release the replacement writer")
					select {
					case <-fresh.transport.closed:
						t.Fatal("stale cancellation closed replacement")
					default:
					}
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
					require.NoError(t, <-result)
					require.Empty(t, rimeTTSErrors(collector))
				})
			})
		}
	}
}

func TestRimeTTSReaderFailureClosesBeforeCallback(t *testing.T) {
	for _, command := range []string{"text", "eos"} {
		for _, failure := range []string{"provider", "decode", "json", "transport", "early_close"} {
			t.Run(command+"/"+failure, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					tts, collector, peers := newRimeTTSSession(t)
					entered := make(chan struct{}, 2)
					unblock := make(chan struct{})
					release := sync.OnceFunc(func() { close(unblock) })
					t.Cleanup(release)
					tts.onPacket = func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
								collector.OnPacket(packet)
								entered <- struct{}{}
								<-unblock
								return nil
							}
						}
						return collector.OnPacket(packets...)
					}
					require.NoError(t, tts.Initialize())
					peer := rimeTTSReceive(t, peers)
					tts.connection.SetCloseHandler(func(int, string) error { return nil })
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
					rimeTTSReceive(t, peer.messages)
					gate := peer.transport.gateWrite()
					var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
					if command == "eos" {
						packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
					}
					result := make(chan error, 1)
					go func() { result <- tts.Transform(context.Background(), packet) }()
					<-gate.entered
					switch failure {
					case "provider":
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "error", "message": "rejected"}))
					case "decode":
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "!!!"}))
					case "json":
						require.NoError(t, peer.conn.WriteMessage(websocket.TextMessage, []byte("{")))
					case "transport":
						require.NoError(t, peer.conn.Close())
					case "early_close":
						code := websocket.CloseNormalClosure
						if command == "eos" {
							code = websocket.CloseGoingAway
						}
						require.NoError(t, peer.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, ""), time.Now().Add(time.Second)))
					}
					rimeTTSReceive(t, entered)
					rimeTTSReceive(t, peer.transport.closed)
					require.NoError(t, rimeTTSReceive(t, result), "the reader must close before blocking in error publication")
					require.Len(t, rimeTTSErrors(collector), 1)
					require.Equal(t, "old", rimeTTSErrors(collector)[0].ContextID)
					release()
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "retry"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					require.Empty(t, peers)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					fresh := rimeTTSReceive(t, peers)
					require.Equal(t, "fresh", rimeTTSReceive(t, fresh.messages)["text"])
					synctest.Wait()
					require.Len(t, rimeTTSErrors(collector), 1)
					require.Empty(t, collector.EndPackets())
				})
			})
		}
	}
}

func TestRimeTTSStaleReaderCannotAffectReplacement(t *testing.T) {
	for _, frame := range []string{"audio", "error", "decode", "done", "close"} {
		t.Run(frame, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newRimeTTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				old := rimeTTSReceive(t, peers)
				rimeTTSReceive(t, old.messages)
				entered, unblock := old.transport.gateRead()
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				switch frame {
				case "audio":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AQI="}))
				case "error":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "error", "message": "old failure"}))
				case "decode":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "!!!"}))
				case "done":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "done"}))
				case "close":
					require.NoError(t, old.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
				}
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := rimeTTSReceive(t, peers)
				rimeTTSReceive(t, fresh.messages)
				release()
				synctest.Wait()
				require.Empty(t, collector.AudioPackets())
				require.Empty(t, collector.EndPackets())
				require.Empty(t, rimeTTSErrors(collector))
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AwQ="}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
				rimeTTSReceive(t, fresh.messages)
				require.NoError(t, fresh.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				require.Empty(t, rimeTTSErrors(collector))
			})
		})
	}
}

func TestRimeTTSQueuedCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newRimeTTSSession(t)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
		peer := rimeTTSReceive(t, peers)
		rimeTTSReceive(t, peer.messages)
		gate := peer.transport.gateWrite()
		owner := make(chan error, 1)
		go func() {
			owner <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
		}()
		<-gate.entered
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		queued := make(chan error, 1)
		go func() { queued <- tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}) }()
		synctest.Wait()
		require.Empty(t, queued)
		cancel()
		require.ErrorIs(t, <-queued, context.Canceled)
		require.Empty(t, owner, "canceling queued Done must not cancel the text writer")
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, <-owner)
		require.Empty(t, rimeTTSErrors(collector))
	})
}

func TestRimeTTSQueuedTextCancellation(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "retires_input"
		if replacement {
			name = "preserves_replacement"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newRimeTTSSession(t)
				require.NoError(t, tts.Initialize())
				old := rimeTTSReceive(t, peers)
				require.NoError(t, tts.writeLock.Acquire(context.Background(), 1))
				releaseWriter := sync.OnceFunc(func() { tts.writeLock.Release(1) })
				t.Cleanup(releaseWriter)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "queued"})
				}()
				synctest.Wait()
				require.Empty(t, result)
				freshResult := make(chan error, 1)
				if replacement {
					go func() {
						freshResult <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
					}()
					synctest.Wait()
				}
				cancel()
				require.ErrorIs(t, <-result, context.Canceled)
				releaseWriter()
				if replacement {
					require.NoError(t, <-freshResult)
				} else {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.Empty(t, peers, "canceled queued text must not leave input open for retry")
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				}
				require.Empty(t, old.messages)
				rimeTTSReceive(t, old.closed)
				fresh := rimeTTSReceive(t, peers)
				require.Equal(t, "fresh", rimeTTSReceive(t, fresh.messages)["text"])
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AQI="}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.Empty(t, rimeTTSErrors(collector))
			})
		})
	}
}

func TestRimeTTSAudioCallbackOrderingAndCloseJoin(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		name := "end_after_audio"
		if shutdown {
			name = "close_joins_audio"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newRimeTTSSession(t)
				entered, unblock := make(chan struct{}), make(chan struct{})
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				tts.onPacket = func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
							close(entered)
							<-unblock
						}
					}
					return collector.OnPacket(packets...)
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				peer := rimeTTSReceive(t, peers)
				rimeTTSReceive(t, peer.messages)
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "chunk", "data": "AQI="}))
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				rimeTTSReceive(t, peer.messages)
				result := make(chan error, 1)
				if shutdown {
					go func() { result <- tts.Close(context.Background()) }()
					rimeTTSReceive(t, peer.transport.closed)
				} else {
					go func() {
						result <- peer.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
					}()
				}
				synctest.Wait()
				require.Empty(t, result)
				require.Empty(t, collector.EndPackets())
				release()
				require.NoError(t, rimeTTSReceive(t, result))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				if shutdown {
					require.Empty(t, collector.EndPackets())
				} else {
					require.Len(t, collector.EndPackets(), 1)
					audioIndex, endIndex := -1, -1
					for i, packet := range collector.GetPackets() {
						switch packet.(type) {
						case internal_type.TextToSpeechAudioPacket:
							audioIndex = i
						case internal_type.TextToSpeechEndPacket:
							endIndex = i
						}
					}
					require.Less(t, audioIndex, endIndex)
				}
				require.Empty(t, rimeTTSErrors(collector))
			})
		})
	}
}

func TestRimeTTSDialUpgradeCancellation(t *testing.T) {
	for _, cancellation := range []string{"caller", "session", "close", "initialize"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, _ := newRimeTTSSession(t)
				sessionDialer := websocket.DefaultDialer
				dialer := *sessionDialer
				requested, closed := make(chan struct{}), make(chan struct{})
				dialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
					client, server := net.Pipe()
					go func() {
						defer close(closed)
						defer server.Close()
						_, err := http.ReadRequest(bufio.NewReader(server))
						if err != nil {
							t.Errorf("read upgrade: %v", err)
							return
						}
						close(requested)
						_, err = server.Read(make([]byte, 1))
						if err == nil {
							t.Error("expected canceled upgrade transport to close")
						}
					}()
					return client, nil
				}
				websocket.DefaultDialer = &dialer
				t.Cleanup(func() { websocket.DefaultDialer = sessionDialer })
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() {
					if cancellation == "initialize" {
						result <- tts.Initialize()
						return
					}
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"})
				}()
				<-requested
				switch cancellation {
				case "caller":
					cancel()
				case "session", "initialize":
					tts.ctxCancel()
				case "close":
					require.NoError(t, tts.Close(context.Background()))
				}
				require.ErrorIs(t, rimeTTSReceive(t, result), context.Canceled)
				rimeTTSReceive(t, closed)
				require.Empty(t, collector.EndPackets())
				require.Empty(t, rimeTTSErrors(collector))
			})
		})
	}
}

func TestRimeTTSInterruptedConnectDoesNotSendRetiredText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newRimeTTSSession(t)
		sessionDialer := websocket.DefaultDialer
		dialer := *sessionDialer
		entered, unblock := make(chan struct{}), make(chan struct{})
		release := sync.OnceFunc(func() { close(unblock) })
		t.Cleanup(release)
		dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			close(entered)
			<-unblock
			return sessionDialer.NetDialTLSContext(ctx, network, address)
		}
		websocket.DefaultDialer = &dialer
		t.Cleanup(func() { websocket.DefaultDialer = sessionDialer })
		result := make(chan error, 1)
		go func() {
			result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"})
		}()
		<-entered
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		release()
		require.NoError(t, <-result)
		websocket.DefaultDialer = sessionDialer
		peer := rimeTTSReceive(t, peers)
		require.Empty(t, peer.messages)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Empty(t, peer.messages)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		require.Equal(t, "fresh", rimeTTSReceive(t, peer.messages)["text"])
		require.Empty(t, peers, "an unused connection can serve the replacement")
		require.Empty(t, rimeTTSErrors(collector))
	})
}

func TestRimeTTSSendFailureIsTerminalOnce(t *testing.T) {
	for _, command := range []string{"text", "eos"} {
		t.Run(command, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newRimeTTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				peer := rimeTTSReceive(t, peers)
				rimeTTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				gate.result <- errors.New("send failed")
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "failed"}
				if command == "eos" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				require.NoError(t, tts.Transform(context.Background(), packet))
				rimeTTSReceive(t, peer.transport.closed)
				require.NoError(t, tts.Transform(context.Background(), packet))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
				synctest.Wait()
				require.Empty(t, peers)
				require.Empty(t, peer.messages)
				require.Len(t, rimeTTSErrors(collector), 1)
				require.Equal(t, "old", rimeTTSErrors(collector)[0].ContextID)
				require.Empty(t, collector.EndPackets())
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := rimeTTSReceive(t, peers)
				require.Equal(t, map[string]interface{}{"text": "fresh"}, rimeTTSReceive(t, fresh.messages))
			})
		})
	}
}

func TestRimeTTSConnectFailureIsNotReplayed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newRimeTTSSession(t)
		sessionDialer := websocket.DefaultDialer
		dialer := *sessionDialer
		attempts := 0
		dialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
			attempts++
			return nil, errors.New("dial failed")
		}
		websocket.DefaultDialer = &dialer
		t.Cleanup(func() { websocket.DefaultDialer = sessionDialer })
		for range 2 {
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "failed"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		}
		require.Equal(t, 1, attempts)
		require.Len(t, rimeTTSErrors(collector), 1)
		require.Equal(t, "old", rimeTTSErrors(collector)[0].ContextID)
		require.Empty(t, collector.EndPackets())
		websocket.DefaultDialer = sessionDialer
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		fresh := rimeTTSReceive(t, peers)
		require.Equal(t, map[string]interface{}{"text": "fresh"}, rimeTTSReceive(t, fresh.messages))
	})
}
