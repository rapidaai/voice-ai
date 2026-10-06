package internal_transformer_neuphonic

import (
	"bufio"
	"context"
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
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type neuphonicTTSPipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *neuphonicTTSPipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

type neuphonicTTSPeer struct {
	conn      *websocket.Conn
	transport *neuphonicTTSGatedConn
	requests  chan map[string]interface{}
	closed    chan struct{}
}

type neuphonicTTSGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

type neuphonicTTSObservedContext struct {
	context.Context
	checked chan struct{}
	once    sync.Once
}

func (ctx *neuphonicTTSObservedContext) Err() error {
	err := ctx.Context.Err()
	ctx.once.Do(func() { close(ctx.checked) })
	return err
}

func (c *neuphonicTTSGatedConn) gateNextWrite() <-chan struct{} {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.entered = make(chan struct{})
	c.release = make(chan struct{})
	return c.entered
}

func (c *neuphonicTTSGatedConn) Write(payload []byte) (int, error) {
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
	return c.Conn.Write(payload)
}

func (c *neuphonicTTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

func newNeuphonicTTSSession(t *testing.T) (*neuphonicTTS, *testutil.PacketCollector, <-chan *neuphonicTTSPeer) {
	t.Helper()
	peers := make(chan *neuphonicTTSPeer, 16)
	var servers sync.WaitGroup
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		transport := &neuphonicTTSGatedConn{Conn: client, closed: make(chan struct{})}
		servers.Go(func() {
			defer server.Close()
			request, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				return
			}
			if request.Header.Get("X-Api-Key") != "local-key" {
				t.Error("missing Neuphonic authentication")
			}
			response := &neuphonicTTSPipeResponse{ResponseRecorder: httptest.NewRecorder(), conn: server}
			conn, err := (&websocket.Upgrader{}).Upgrade(response, request, nil)
			if err != nil {
				t.Errorf("upgrade websocket: %v", err)
				return
			}
			peer := &neuphonicTTSPeer{conn: conn, transport: transport, requests: make(chan map[string]interface{}, 128), closed: make(chan struct{})}
			defer close(peer.closed)
			peers <- peer
			for {
				var message map[string]interface{}
				if err := conn.ReadJSON(&message); err != nil {
					return
				}
				peer.requests <- message
			}
		})
		return transport, nil
	}
	websocket.DefaultDialer = &dialer
	ctx, cancel := context.WithCancel(context.Background())
	collector := testutil.NewPacketCollector()
	value, err := structpb.NewStruct(map[string]interface{}{"key": "local-key"})
	require.NoError(t, err)
	transformer, err := NewNeuPhonicTextToSpeech(ctx, neuphonicTestLogger(), &protos.VaultCredential{Value: value}, collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*neuphonicTTS)
	t.Cleanup(func() {
		cancel()
		require.NoError(t, tts.Close(context.Background()))
		servers.Wait()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, peers
}

func nextNeuphonicTTSPeer(t *testing.T, peers <-chan *neuphonicTTSPeer) *neuphonicTTSPeer {
	t.Helper()
	select {
	case peer := <-peers:
		return peer
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Neuphonic connection")
		return nil
	}
}

func TestNeuphonicTTSReadIdleCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newNeuphonicTTSSession(t)
		require.NoError(t, tts.Initialize())
		peer := nextNeuphonicTTSPeer(t, peers)
		require.NoError(t, tts.Initialize())
		require.Empty(t, peers)
		for _, text := range []string{"first", "second"} {
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: text}))
			require.Equal(t, text+" <STOP>", waitNeuphonicTTSRequest(t, peer.requests)["text"])
		}
		require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AQI="}}))
		synctest.Wait()
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Equal(t, "<STOP>", waitNeuphonicTTSRequest(t, peer.requests)["text"])
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Empty(t, collector.EndPackets())
		require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AwQ="}}))
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Empty(t, collector.EndPackets(), "post-Done audio must renew the idle interval")
		time.Sleep(time.Second)
		synctest.Wait()
		require.Len(t, collector.AudioPackets(), 2)
		require.Len(t, collector.EndPackets(), 1)
		require.Equal(t, "old", collector.EndPackets()[0].ContextID)
		require.Empty(t, neuphonicTTSErrors(collector))
		select {
		case <-peer.closed:
		default:
			t.Fatal("idle completion did not close the socket")
		}
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Empty(t, peers)
		require.Empty(t, peer.requests)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		fresh := nextNeuphonicTTSPeer(t, peers)
		require.Equal(t, "fresh <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
	})
}

func TestNeuphonicTTSNoAudioAndDropAreErrors(t *testing.T) {
	for _, failure := range []string{"no audio", "drop after audio"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newNeuphonicTTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
				peer := nextNeuphonicTTSPeer(t, peers)
				waitNeuphonicTTSRequest(t, peer.requests)
				if failure == "drop after audio" {
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AQI="}}))
					synctest.Wait()
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				waitNeuphonicTTSRequest(t, peer.requests)
				if failure == "no audio" {
					time.Sleep(3 * time.Second)
				} else {
					require.NoError(t, peer.conn.Close())
				}
				synctest.Wait()
				require.Empty(t, collector.EndPackets())
				require.Len(t, neuphonicTTSErrors(collector), 1)
				require.Equal(t, "old", neuphonicTTSErrors(collector)[0].ContextID)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
				require.Empty(t, peers)
			})
		})
	}
}

func TestNeuphonicTTSBlockedAudioRenewsReadDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newNeuphonicTTSSession(t)
		entered := make(chan struct{})
		enter := sync.OnceFunc(func() { close(entered) })
		unblock := make(chan struct{})
		release := sync.OnceFunc(func() { close(unblock) })
		t.Cleanup(release)
		tts.onPacket = func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
					enter()
					<-unblock
				}
			}
			return collector.OnPacket(packets...)
		}
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
		peer := nextNeuphonicTTSPeer(t, peers)
		waitNeuphonicTTSRequest(t, peer.requests)
		require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AQI="}}))
		<-entered
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		waitNeuphonicTTSRequest(t, peer.requests)
		queued := make(chan error, 1)
		go func() {
			queued <- peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AwQ="}})
		}()
		time.Sleep(4 * time.Second)
		synctest.Wait()
		require.Empty(t, collector.EndPackets())
		require.Empty(t, neuphonicTTSErrors(collector))
		release()
		require.NoError(t, <-queued)
		synctest.Wait()
		require.Len(t, collector.AudioPackets(), 2, "queued audio must survive an elapsed deadline during callback")
		time.Sleep(3 * time.Second)
		synctest.Wait()
		var order []string
		for _, packet := range collector.GetPackets() {
			switch packet.(type) {
			case internal_type.TextToSpeechAudioPacket:
				order = append(order, "audio")
			case internal_type.TextToSpeechEndPacket:
				order = append(order, "end")
			}
		}
		require.Equal(t, []string{"audio", "audio", "end"}, order)
		require.Empty(t, neuphonicTTSErrors(collector))
	})
}

func TestNeuphonicTTSGatedWriteCancellation(t *testing.T) {
	for _, command := range []string{"text", "stop"} {
		for _, cancellation := range []string{"interrupt", "caller", "session", "close"} {
			t.Run(command+"/"+cancellation, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					tts, collector, peers := newNeuphonicTTSSession(t)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
					peer := nextNeuphonicTTSPeer(t, peers)
					waitNeuphonicTTSRequest(t, peer.requests)
					entered := peer.transport.gateNextWrite()
					var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
					if command == "stop" {
						packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					result := make(chan error, 1)
					go func() { result <- tts.Transform(ctx, packet) }()
					<-entered
					synctest.Wait()
					require.Empty(t, result, "write must be blocked at the transport gate")
					if command == "stop" && cancellation == "interrupt" {
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AQI="}}))
						synctest.Wait()
						time.Sleep(4 * time.Second)
						synctest.Wait()
						require.Empty(t, collector.EndPackets(), "STOP must finish writing before the completion interval starts")
						require.Empty(t, neuphonicTTSErrors(collector))
						require.Empty(t, result)
					}
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
					synctest.Wait()
					select {
					case err := <-result:
						if cancellation == "interrupt" {
							require.NoError(t, err)
						} else {
							require.ErrorIs(t, err, context.Canceled)
						}
					default:
						t.Fatal("cancellation did not unblock Transform")
					}
					select {
					case <-peer.closed:
					default:
						t.Fatal("cancellation did not physically close the socket")
					}
					require.Empty(t, neuphonicTTSErrors(collector))
					if cancellation == "session" || cancellation == "close" {
						require.ErrorIs(t, tts.Initialize(), context.Canceled)
						return
					}
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					require.Empty(t, peers)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
					fresh := nextNeuphonicTTSPeer(t, peers)
					require.Equal(t, "fresh <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
					cancel()
					synctest.Wait()
					entered = fresh.transport.gateNextWrite()
					go func() {
						result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "blocked again"})
					}()
					<-entered
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					synctest.Wait()
					require.Empty(t, result, "stale interrupt canceled the newer writer")
					select {
					case <-fresh.transport.closed:
						t.Fatal("late caller cancellation or stale interrupt closed the replacement socket")
					default:
					}
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
					require.NoError(t, <-result)
					require.Empty(t, neuphonicTTSErrors(collector))
				})
			})
		}
	}
}

func TestNeuphonicTTSReadFailureClosesBeforeCallback(t *testing.T) {
	for _, command := range []string{"text", "stop"} {
		t.Run(command, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newNeuphonicTTSSession(t)
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
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				peer := nextNeuphonicTTSPeer(t, peers)
				waitNeuphonicTTSRequest(t, peer.requests)
				writing := peer.transport.gateNextWrite()
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if command == "stop" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				result := make(chan error, 1)
				go func() { result <- tts.Transform(context.Background(), packet) }()
				<-writing
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "invalid-base64"}}))
				<-entered
				synctest.Wait()
				select {
				case <-peer.closed:
				default:
					t.Fatal("failed socket remained open during error callback")
				}
				select {
				case err := <-result:
					require.NoError(t, err)
				default:
					t.Fatal("reader error did not release the stalled writer before callback returned")
				}
				require.Len(t, neuphonicTTSErrors(collector), 1)
				release()
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := nextNeuphonicTTSPeer(t, peers)
				require.Equal(t, "fresh <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
				synctest.Wait()
				require.Len(t, neuphonicTTSErrors(collector), 1)
			})
		})
	}
}

func TestNeuphonicTTSBlockedAudioOwnership(t *testing.T) {
	for _, action := range []string{"interrupt", "replacement", "close"} {
		t.Run(action, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newNeuphonicTTSSession(t)
				entered := make(chan struct{})
				unblock := make(chan struct{})
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				tts.onPacket = func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if audio, ok := packet.(internal_type.TextToSpeechAudioPacket); ok && audio.ContextID == "old" {
							close(entered)
							<-unblock
						}
					}
					return collector.OnPacket(packets...)
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
				peer := nextNeuphonicTTSPeer(t, peers)
				waitNeuphonicTTSRequest(t, peer.requests)
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AQI="}}))
				<-entered
				if action == "close" {
					closed := make(chan error, 1)
					go func() { closed <- tts.Close(context.Background()) }()
					synctest.Wait()
					require.Empty(t, closed, "Close must join the reader callback")
					select {
					case <-peer.closed:
					default:
						t.Fatal("Close must close the socket before waiting for the callback")
					}
					release()
					require.NoError(t, <-closed)
					require.Empty(t, neuphonicTTSErrors(collector))
					return
				}
				if action == "interrupt" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := nextNeuphonicTTSPeer(t, peers)
				require.Equal(t, "fresh <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"data": map[string]string{"audio": "AwQ="}}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				release()
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 2)
				require.Equal(t, "old", collector.AudioPackets()[1].ContextID, "in-flight audio must retain its original context")
				time.Sleep(4 * time.Second)
				synctest.Wait()
				require.Empty(t, collector.EndPackets())
				require.Empty(t, neuphonicTTSErrors(collector), "detached reader failure must not fail the replacement")
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "more"}))
				require.Equal(t, "more <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
			})
		})
	}
}

func TestNeuphonicTTSDialCancellation(t *testing.T) {
	for _, cancellation := range []string{"caller", "session", "close", "interrupt"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newNeuphonicTTSSession(t)
				dial := websocket.DefaultDialer.NetDialTLSContext
				entered := make(chan struct{})
				unblock := make(chan struct{})
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				websocket.DefaultDialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					close(entered)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-unblock:
						return dial(ctx, network, address)
					}
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
				}()
				<-entered
				switch cancellation {
				case "caller":
					cancel()
				case "session":
					tts.ctxCancel()
				case "close":
					require.NoError(t, tts.Close(context.Background()))
				case "interrupt":
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					release()
				}
				if cancellation == "interrupt" {
					require.NoError(t, <-result)
					peer := nextNeuphonicTTSPeer(t, peers)
					require.Empty(t, peer.requests, "interruption during connect must prevent the original text write")
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
					require.Empty(t, peer.requests)
				} else {
					require.ErrorIs(t, <-result, context.Canceled)
					require.Empty(t, peers)
				}
				synctest.Wait()
				require.Empty(t, neuphonicTTSErrors(collector))
				websocket.DefaultDialer.NetDialTLSContext = dial
				if cancellation == "caller" || cancellation == "interrupt" {
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				}
			})
		})
	}
}

func TestNeuphonicTTSIdleSocketAndLateCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newNeuphonicTTSSession(t)
		require.NoError(t, tts.Initialize())
		idle := nextNeuphonicTTSPeer(t, peers)
		time.Sleep(10 * time.Second)
		require.Equal(t, "", waitNeuphonicTTSRequest(t, idle.requests)["text"])
		require.NoError(t, idle.conn.Close())
		synctest.Wait()
		require.Empty(t, neuphonicTTSErrors(collector), "a warmed idle socket has no synthesis to fail")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
		old := nextNeuphonicTTSPeer(t, peers)
		require.Equal(t, "old <STOP>", waitNeuphonicTTSRequest(t, old.requests)["text"])
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		fresh := nextNeuphonicTTSPeer(t, peers)
		waitNeuphonicTTSRequest(t, fresh.requests)
		cancel()
		synctest.Wait()
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "more"}))
		require.Equal(t, "more <STOP>", waitNeuphonicTTSRequest(t, fresh.requests)["text"])
		require.Empty(t, peers)
		require.Empty(t, neuphonicTTSErrors(collector))
	})
}

func TestNeuphonicTTSCallerCancellationBehindKeepalive(t *testing.T) {
	tts, collector, peers := newNeuphonicTTSSession(t)
	require.NoError(t, tts.Initialize())
	peer := nextNeuphonicTTSPeer(t, peers)
	entered := peer.transport.gateNextWrite()
	select {
	case <-entered:
	case <-time.After(12 * time.Second):
		t.Fatal("keepalive did not reach the transport gate")
	}
	caller, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &neuphonicTTSObservedContext{Context: caller, checked: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"})
	}()
	// Cancel after the entry check, while the keepalive still owns the writer.
	<-ctx.checked
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Error("caller cancellation cannot release Transform waiting behind the stalled idle keepalive")
		require.NoError(t, tts.Close(context.Background()))
		require.ErrorIs(t, <-result, context.Canceled)
	}
	require.Empty(t, neuphonicTTSErrors(collector))
}
