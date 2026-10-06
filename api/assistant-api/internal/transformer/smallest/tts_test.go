package internal_transformer_smallest

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

type smallestTTSPipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *smallestTTSPipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

type smallestTTSWriteGate struct {
	entered chan struct{}
	result  chan error
}

type smallestTTSGatedConn struct {
	net.Conn
	gateMu      sync.Mutex
	writeGate   *smallestTTSWriteGate
	readEntered chan struct{}
	readRelease chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func (c *smallestTTSGatedConn) gateWrite() *smallestTTSWriteGate {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.writeGate = &smallestTTSWriteGate{entered: make(chan struct{}), result: make(chan error, 1)}
	return c.writeGate
}

func (c *smallestTTSGatedConn) gateRead() (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.readEntered, c.readRelease = make(chan struct{}), make(chan struct{})
	return c.readEntered, c.readRelease
}

func (c *smallestTTSGatedConn) Read(payload []byte) (int, error) {
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

func (c *smallestTTSGatedConn) Write(payload []byte) (int, error) {
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

func (c *smallestTTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

type smallestTTSPeer struct {
	conn      *websocket.Conn
	transport *smallestTTSGatedConn
	request   *http.Request
	messages  chan map[string]interface{}
	closed    chan struct{}
}

func newSmallestTTSSession(t *testing.T) (*smallestTTS, *testutil.PacketCollector, <-chan *smallestTTSPeer) {
	t.Helper()
	peers := make(chan *smallestTTSPeer, 16)
	var servers sync.WaitGroup
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		transport := &smallestTTSGatedConn{Conn: client, closed: make(chan struct{})}
		servers.Go(func() {
			defer server.Close()
			request, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				return
			}
			response := &smallestTTSPipeResponse{ResponseRecorder: httptest.NewRecorder(), conn: server}
			conn, err := (&websocket.Upgrader{}).Upgrade(response, request, nil)
			if err != nil {
				t.Errorf("upgrade websocket: %v", err)
				return
			}
			peer := &smallestTTSPeer{conn: conn, transport: transport, request: request, messages: make(chan map[string]interface{}, 32), closed: make(chan struct{})}
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
	transformer, err := NewSmallestTextToSpeech(context.Background(), newTestLogger(), newVaultCredential(map[string]interface{}{"key": "test-key"}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*smallestTTS)
	t.Cleanup(func() {
		require.NoError(t, tts.Close(context.Background()))
		servers.Wait()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, peers
}

func smallestTTSReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Smallest TTS test event")
		var zero T
		return zero
	}
}

func smallestTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var packets []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			packets = append(packets, packet)
		}
	}
	return packets
}

func TestSmallestTTSSequentialResponses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newSmallestTTSSession(t)
		require.NoError(t, tts.Initialize())
		peer := smallestTTSReceive(t, peers)
		require.NoError(t, tts.Initialize())
		require.Empty(t, peers)
		require.Equal(t, "Bearer test-key", peer.request.Header.Get("Authorization"))
		require.Equal(t, "rapida", peer.request.Header.Get("X-Source"))
		require.Equal(t, "/waves/v1/tts/live", peer.request.URL.Path)
		for index, id := range []string{"old", "fresh"} {
			for _, text := range []string{"hello", "more"} {
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: text}))
				if index == 1 && text == "hello" {
					peer = smallestTTSReceive(t, peers)
				}
				message := smallestTTSReceive(t, peer.messages)
				require.Equal(t, id, message["context_id"])
				require.NotContains(t, message, "session_id")
				require.Equal(t, text, message["text"])
				require.Equal(t, true, message["continue"])
				require.Equal(t, DefaultVoiceID, message["voice_id"])
				require.Equal(t, DefaultTextToSpeechModel, message["model"])
			}
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "complete"}))
			synctest.Wait()
			require.Len(t, collector.EndPackets(), index, "early complete must not close an open input context")
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "AQI="}}))
			synctest.Wait()
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
			message := smallestTTSReceive(t, peer.messages)
			require.Equal(t, id, message["context_id"])
			require.Equal(t, "", message["text"])
			require.Equal(t, false, message["continue"])
			require.Equal(t, true, message["flush"])
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "late"}))
			require.Empty(t, peer.messages)
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "AwQ="}}))
			require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "complete"}))
			synctest.Wait()
			require.Len(t, collector.AudioPackets(), (index+1)*2)
			require.Equal(t, id, collector.AudioPackets()[index*2+1].ContextID)
			require.Len(t, collector.EndPackets(), index+1)
			require.Equal(t, id, collector.EndPackets()[index].ContextID)
			smallestTTSReceive(t, peer.closed)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: id}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "retired"}))
			require.Empty(t, peers)
		}
		require.Empty(t, smallestTTSErrors(collector))
	})
}

func TestSmallestTTSCompleteWaitsForFlushResult(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				peer := smallestTTSReceive(t, peers)
				smallestTTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}()
				<-gate.entered
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "complete"}))
				synctest.Wait()
				require.Empty(t, collector.EndPackets())
				if failed {
					gate.result <- errors.New("flush failed")
				} else {
					gate.result <- nil
				}
				require.NoError(t, <-result)
				synctest.Wait()
				if failed {
					require.Empty(t, collector.EndPackets())
					require.Len(t, smallestTTSErrors(collector), 1)
				} else {
					require.Len(t, collector.EndPackets(), 1)
					require.Empty(t, smallestTTSErrors(collector))
				}
			})
		})
	}
}

func TestSmallestTTSGatedWriteCancellation(t *testing.T) {
	for _, command := range []string{"text", "flush"} {
		for _, cancellation := range []string{"interrupt", "caller", "session", "close"} {
			t.Run(command+"/"+cancellation, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					tts, collector, peers := newSmallestTTSSession(t)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
					peer := smallestTTSReceive(t, peers)
					smallestTTSReceive(t, peer.messages)
					gate := peer.transport.gateWrite()
					var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
					if command == "flush" {
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
					}
					err := <-result
					if cancellation == "interrupt" {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, context.Canceled)
					}
					smallestTTSReceive(t, peer.closed)
					require.Empty(t, smallestTTSErrors(collector))
					require.Empty(t, collector.EndPackets())
					if cancellation == "session" || cancellation == "close" {
						require.ErrorIs(t, tts.Initialize(), context.Canceled)
						return
					}
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					require.Empty(t, peers)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					fresh := smallestTTSReceive(t, peers)
					require.Equal(t, "fresh", smallestTTSReceive(t, fresh.messages)["context_id"])
					gate = fresh.transport.gateWrite()
					go func() {
						result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "blocked again"})
					}()
					<-gate.entered
					cancel()
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					synctest.Wait()
					require.Empty(t, result, "late cancellation must not release the new writer")
					select {
					case <-fresh.transport.closed:
						t.Fatal("stale cancellation closed the replacement socket")
					default:
					}
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
					require.NoError(t, <-result)
					require.Empty(t, smallestTTSErrors(collector))
				})
			})
		}
	}
}

func TestSmallestTTSReaderFailureClosesBeforeCallback(t *testing.T) {
	for _, command := range []string{"text", "flush"} {
		t.Run(command, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
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
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				peer := smallestTTSReceive(t, peers)
				smallestTTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if command == "flush" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				result := make(chan error, 1)
				go func() { result <- tts.Transform(context.Background(), packet) }()
				<-gate.entered
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "error", "message": "invalid voice"}))
				<-entered
				smallestTTSReceive(t, peer.closed)
				require.NoError(t, <-result, "reader failure must release the writer before its error callback returns")
				require.Len(t, smallestTTSErrors(collector), 1)
				require.Contains(t, smallestTTSErrors(collector)[0].ErrMessage(), "invalid voice")
				release()
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := smallestTTSReceive(t, peers)
				require.Equal(t, "fresh", smallestTTSReceive(t, fresh.messages)["context_id"])
				require.Len(t, smallestTTSErrors(collector), 1)
			})
		})
	}
}

func TestSmallestTTSStaleReadAfterReplacement(t *testing.T) {
	for _, response := range []string{"chunk", "error", "complete"} {
		t.Run(response, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
				old := smallestTTSReceive(t, peers)
				smallestTTSReceive(t, old.messages)
				if response == "complete" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					smallestTTSReceive(t, old.messages)
				}
				entered, unblock := old.transport.gateRead()
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"status": response, "message": "old failure", "data": map[string]string{"audio": "AQI="}}))
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := smallestTTSReceive(t, peers)
				smallestTTSReceive(t, fresh.messages)
				release()
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "AwQ="}}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
				smallestTTSReceive(t, fresh.messages)
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"status": "complete"}))
				synctest.Wait()
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				require.Empty(t, smallestTTSErrors(collector))
			})
		})
	}
}

func TestSmallestTTSBlockedAudioOrderingAndClose(t *testing.T) {
	for _, action := range []string{"complete", "close"} {
		t.Run(action, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
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
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
				peer := smallestTTSReceive(t, peers)
				smallestTTSReceive(t, peer.messages)
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "AQI="}}))
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				smallestTTSReceive(t, peer.messages)
				result := make(chan error, 1)
				if action == "complete" {
					go func() { result <- peer.conn.WriteJSON(map[string]interface{}{"status": "complete"}) }()
				} else {
					go func() { result <- tts.Close(context.Background()) }()
					smallestTTSReceive(t, peer.closed)
				}
				synctest.Wait()
				require.Empty(t, result)
				require.Empty(t, collector.EndPackets())
				release()
				require.NoError(t, <-result)
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				if action == "complete" {
					require.Len(t, collector.EndPackets(), 1)
				} else {
					require.Empty(t, collector.EndPackets())
				}
				require.Empty(t, smallestTTSErrors(collector))
			})
		})
	}
}

func TestSmallestTTSDialUpgradeCancellation(t *testing.T) {
	for _, cancellation := range []string{"caller", "session", "close"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				upgrading := make(chan struct{})
				peerClosed := make(chan error, 1)
				var remote net.Conn
				websocket.DefaultDialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
					client, server := net.Pipe()
					remote = server
					go func() {
						defer server.Close()
						_, err := http.ReadRequest(bufio.NewReader(server))
						if err != nil {
							peerClosed <- err
							return
						}
						close(upgrading)
						_, err = server.Read(make([]byte, 1))
						peerClosed <- err
					}()
					return client, nil
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
				}()
				<-upgrading
				t.Cleanup(func() { _ = remote.Close() })
				switch cancellation {
				case "caller":
					cancel()
				case "session":
					tts.ctxCancel()
				case "close":
					require.NoError(t, tts.Close(context.Background()))
				}
				require.ErrorIs(t, <-result, context.Canceled)
				require.Error(t, <-peerClosed, "canceling the HTTP upgrade must close the physical transport")
				require.Empty(t, smallestTTSErrors(collector))
				websocket.DefaultDialer.NetDialTLSContext = dial
				if cancellation == "caller" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.Empty(t, peers)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					fresh := smallestTTSReceive(t, peers)
					require.Equal(t, "fresh", smallestTTSReceive(t, fresh.messages)["context_id"])
				}
			})
		})
	}
}

func TestSmallestTTSFailureOnceAndIdleDisconnect(t *testing.T) {
	for _, failure := range []string{"provider", "encoding", "transport", "idle transport", "dial"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newSmallestTTSSession(t)
				if failure == "dial" {
					websocket.DefaultDialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
						return nil, errors.New("dial failed")
					}
				}
				if failure == "idle transport" {
					require.NoError(t, tts.Initialize())
				} else {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
				}
				if failure != "dial" {
					peer := smallestTTSReceive(t, peers)
					switch failure {
					case "provider":
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "error", "errors": []map[string]string{{"message": "voice rejected"}}}))
					case "encoding":
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "invalid-base64"}}))
					default:
						require.NoError(t, peer.conn.Close())
					}
				}
				synctest.Wait()
				require.Empty(t, collector.EndPackets())
				if failure == "idle transport" {
					require.Empty(t, smallestTTSErrors(collector))
					return
				}
				require.Len(t, smallestTTSErrors(collector), 1)
				require.Equal(t, "old", smallestTTSErrors(collector)[0].ContextID)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Len(t, smallestTTSErrors(collector), 1)
				require.Empty(t, peers)
			})
		})
	}
}

func TestSmallestTTSQueuedCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newSmallestTTSSession(t)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
		peer := smallestTTSReceive(t, peers)
		smallestTTSReceive(t, peer.messages)
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
		require.Empty(t, owner, "canceling a queued Done must not cancel the text writer")
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, <-owner)
		require.Empty(t, smallestTTSErrors(collector))
	})
}

func TestSmallestTTSQueuedTextCancellationRetiresInput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newSmallestTTSSession(t)
		require.NoError(t, tts.Initialize())
		peer := smallestTTSReceive(t, peers)
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
		tts.stateMu.Lock()
		contextID := tts.contextId
		tts.stateMu.Unlock()
		require.Equal(t, "old", contextID, "the text must be admitted before cancellation")
		cancel()
		require.ErrorIs(t, <-result, context.Canceled)
		releaseWriter()
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.Empty(t, peer.messages, "canceled queued text must not leave a flushable input context")
		require.Empty(t, peers, "late same-ID input must not reconnect")
		require.Empty(t, collector.EndPackets())
		require.Empty(t, smallestTTSErrors(collector))
		smallestTTSReceive(t, peer.closed)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		fresh := smallestTTSReceive(t, peers)
		require.Equal(t, "fresh", smallestTTSReceive(t, fresh.messages)["context_id"])
	})
}

func TestSmallestTTSQueuedTextCancellationPreservesReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newSmallestTTSSession(t)
		require.NoError(t, tts.Initialize())
		old := smallestTTSReceive(t, peers)
		require.NoError(t, tts.writeLock.Acquire(context.Background(), 1))
		releaseWriter := sync.OnceFunc(func() { tts.writeLock.Release(1) })
		t.Cleanup(releaseWriter)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		oldResult := make(chan error, 1)
		go func() {
			oldResult <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		synctest.Wait()
		require.Empty(t, oldResult)
		freshResult := make(chan error, 1)
		go func() {
			freshResult <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
		}()
		synctest.Wait()
		tts.stateMu.Lock()
		contextID := tts.contextId
		tts.stateMu.Unlock()
		require.Equal(t, "fresh", contextID, "replacement must own admission before the old caller cancels")
		cancel()
		require.ErrorIs(t, <-oldResult, context.Canceled)
		releaseWriter()
		require.NoError(t, <-freshResult)
		smallestTTSReceive(t, old.closed)
		fresh := smallestTTSReceive(t, peers)
		require.Equal(t, "fresh", smallestTTSReceive(t, fresh.messages)["context_id"])
		require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"status": "chunk", "data": map[string]string{"audio": "AQI="}}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
		smallestTTSReceive(t, fresh.messages)
		require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"status": "complete"}))
		synctest.Wait()
		require.Len(t, collector.AudioPackets(), 1)
		require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
		require.Len(t, collector.EndPackets(), 1)
		require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
		require.Empty(t, smallestTTSErrors(collector))
	})
}
