package internal_transformer_custom_tts_websocket_v1

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type lifecyclePipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *lifecyclePipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

type lifecycleWriteGate struct {
	entered chan struct{}
	result  chan error
}

type lifecycleGatedConn struct {
	net.Conn
	gateMu    sync.Mutex
	gate      *lifecycleWriteGate
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *lifecycleGatedConn) gateWrite() *lifecycleWriteGate {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.gate = &lifecycleWriteGate{entered: make(chan struct{}), result: make(chan error, 1)}
	return c.gate
}

func (c *lifecycleGatedConn) Write(payload []byte) (int, error) {
	c.gateMu.Lock()
	gate := c.gate
	c.gate = nil
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

func (c *lifecycleGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

type lifecyclePeer struct {
	conn      *websocket.Conn
	transport *lifecycleGatedConn
	messages  chan map[string]any
	closed    chan struct{}
}

func newLifecycleSession(t *testing.T, configuredInterrupt bool) (*textToSpeech, *testutil.PacketCollector, <-chan *lifecyclePeer) {
	t.Helper()
	requests := `[
		{"when":{"packet":"text"},"send":{"frame":"json","body":{"type":"text","text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}},
		{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done","request_id":{"$path":"packet.message_id"}}}}`
	if configuredInterrupt {
		requests += `,{"when":{"packet":"interrupt"},"send":{"frame":"json","body":{"type":"interrupt","request_id":{"$path":"packet.message_id"}}}}`
	}
	requests += `]`
	config, err := NewConfig(testWSCredential(t, map[string]any{credentialKeyBaseURLCamel: "wss://custom.invalid/ws"}), utils.Option{
		optionKeyRequestRules: requests,
		optionKeyResponseRules: `[
			{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},
			{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"request_id"},"done":true}}
		]`,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	collector := testutil.NewPacketCollector()
	tts := &textToSpeech{config: config, engine: config.newEngine(), ctx: ctx, cancel: cancel, logger: testutil.NewTestLogger(), onPacket: collector.OnPacket, resampler: &fakeTTSResampler{}}
	peers := make(chan *lifecyclePeer, 8)
	var servers sync.WaitGroup
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		transport := &lifecycleGatedConn{Conn: client, closed: make(chan struct{})}
		servers.Go(func() {
			defer server.Close()
			request, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				return
			}
			response := &lifecyclePipeResponse{ResponseRecorder: httptest.NewRecorder(), conn: server}
			conn, err := (&websocket.Upgrader{}).Upgrade(response, request, nil)
			if err != nil {
				t.Errorf("upgrade websocket: %v", err)
				return
			}
			peer := &lifecyclePeer{conn: conn, transport: transport, messages: make(chan map[string]any, 16), closed: make(chan struct{})}
			defer close(peer.closed)
			peers <- peer
			for {
				var message map[string]any
				if err := conn.ReadJSON(&message); err != nil {
					return
				}
				peer.messages <- message
			}
		})
		return transport, nil
	}
	websocket.DefaultDialer = &dialer
	t.Cleanup(func() {
		require.NoError(t, tts.Close(context.Background()))
		servers.Wait()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, peers
}

func lifecycleReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for custom TTS lifecycle event")
		var zero T
		return zero
	}
}

func TestTextToSpeech_BusyWriteInterrupt(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "without_interrupt_rule"
		if configured {
			name = "with_interrupt_rule"
		}
		for _, command := range []string{"text", "done"} {
			t.Run(name+"/"+command, func(t *testing.T) {
				tts, collector, peers := newLifecycleSession(t, configured)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				old := lifecycleReceive(t, peers)
				lifecycleReceive(t, old.messages)
				gate := old.transport.gateWrite()
				writer, interrupted := make(chan error, 1), make(chan error, 1)
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if command == "done" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				go func() { writer <- tts.Transform(context.Background(), packet) }()
				<-gate.entered
				go func() {
					interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
				}()
				select {
				case err := <-interrupted:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Error("interrupt waited for the stalled writer")
					// Release the original implementation too, so a regression cannot strand cleanup.
					require.NoError(t, old.transport.Close())
					require.NoError(t, lifecycleReceive(t, interrupted))
				}
				require.NoError(t, lifecycleReceive(t, writer))
				lifecycleReceive(t, old.closed)
				require.Empty(t, old.messages, "a busy interrupt must not write an optional request")
				require.Empty(t, collector.EndPackets())
				for _, packet := range collector.GetPackets() {
					_, failed := packet.(internal_type.TextToSpeechErrorPacket)
					require.False(t, failed, "intentional interruption must not emit TTS error")
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := lifecycleReceive(t, peers)
				require.Equal(t, "fresh", lifecycleReceive(t, fresh.messages)["request_id"])
				gate = fresh.transport.gateWrite()
				go func() {
					writer <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "more"})
				}()
				<-gate.entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				select {
				case <-fresh.transport.closed:
					t.Fatal("stale interrupt closed the new writer")
				default:
				}
				gate.result <- nil
				require.NoError(t, lifecycleReceive(t, writer))
				require.Equal(t, "more", lifecycleReceive(t, fresh.messages)["text"])
				require.NoError(t, fresh.conn.WriteMessage(websocket.BinaryMessage, []byte{1, 2}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
				require.Equal(t, "done", lifecycleReceive(t, fresh.messages)["type"])
				require.NoError(t, fresh.conn.WriteJSON(map[string]any{"type": "done", "request_id": "fresh"}))
				collector.WaitForTTSEnd(t, time.Second)
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
			})
		}
	}
}

func TestTextToSpeech_CloseUnblocksWrites(t *testing.T) {
	for _, command := range []string{"text", "interrupt"} {
		for _, shutdown := range []string{"session", "close"} {
			t.Run(command+"/"+shutdown, func(t *testing.T) {
				tts, collector, peers := newLifecycleSession(t, true)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				peer := lifecycleReceive(t, peers)
				lifecycleReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				writer := make(chan error, 1)
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if command == "interrupt" {
					packet = internal_type.TextToSpeechInterruptPacket{ContextID: "old"}
				}
				go func() { writer <- tts.Transform(context.Background(), packet) }()
				<-gate.entered
				closed := make(chan error, 1)
				if shutdown == "session" {
					tts.cancel()
				} else {
					go func() { closed <- tts.Close(context.Background()) }()
				}
				select {
				case <-peer.transport.closed:
				case <-time.After(time.Second):
					t.Error("shutdown waited for the stalled writer before closing its socket")
					_ = peer.transport.Close()
				}
				require.NoError(t, lifecycleReceive(t, writer))
				if shutdown == "close" {
					require.NoError(t, lifecycleReceive(t, closed))
				}
				require.Empty(t, collector.EndPackets())
				for _, packet := range collector.GetPackets() {
					_, failed := packet.(internal_type.TextToSpeechErrorPacket)
					require.False(t, failed, "shutdown must not report intentional write failure")
				}
			})
		}
	}
}

func TestTextToSpeech_WriteFailureClosesBeforeCallback(t *testing.T) {
	for _, command := range []string{"text", "done", "interrupt"} {
		t.Run(command, func(t *testing.T) {
			tts, collector, peers := newLifecycleSession(t, true)
			entered, unblock := make(chan struct{}, 2), make(chan struct{})
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
			peer := lifecycleReceive(t, peers)
			lifecycleReceive(t, peer.messages)
			gate := peer.transport.gateWrite()
			gate.result <- errors.New("injected write failure")
			writer := make(chan error, 1)
			var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "failed"}
			switch command {
			case "done":
				packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
			case "interrupt":
				packet = internal_type.TextToSpeechInterruptPacket{ContextID: "old"}
			}
			go func() { writer <- tts.Transform(context.Background(), packet) }()
			lifecycleReceive(t, entered)
			select {
			case <-peer.transport.closed:
			case <-time.After(time.Second):
				t.Error("failed socket remains open during the blocked error callback")
				_ = peer.transport.Close()
			}
			release()
			require.NoError(t, lifecycleReceive(t, writer))
			require.NoError(t, tts.Close(context.Background()))
			errorCount := 0
			for _, packet := range collector.GetPackets() {
				if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
					errorCount++
					require.Equal(t, "old", packet.ContextID)
				}
			}
			require.Equal(t, 1, errorCount)
			require.Empty(t, collector.EndPackets())
		})
	}
}

func TestTextToSpeech_InterruptDuringDial(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "successful_dial"
		if failed {
			name = "failed_dial"
		}
		t.Run(name, func(t *testing.T) {
			tts, collector, peers := newLifecycleSession(t, true)
			sessionDialer := websocket.DefaultDialer
			dialer := *sessionDialer
			entered, unblock := make(chan struct{}), make(chan struct{})
			release := sync.OnceFunc(func() { close(unblock) })
			t.Cleanup(release)
			dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				close(entered)
				<-unblock
				if failed {
					return nil, errors.New("dial failed after interruption")
				}
				return sessionDialer.NetDialTLSContext(ctx, network, address)
			}
			websocket.DefaultDialer = &dialer
			t.Cleanup(func() { websocket.DefaultDialer = sessionDialer })
			writer, interrupted := make(chan error, 1), make(chan error, 1)
			go func() {
				writer <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"})
			}()
			<-entered
			go func() {
				interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
			}()
			select {
			case err := <-interrupted:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Error("interrupt waited for connectMu")
				release()
				require.NoError(t, lifecycleReceive(t, interrupted))
			}
			release()
			require.NoError(t, lifecycleReceive(t, writer))
			websocket.DefaultDialer = sessionDialer
			if !failed {
				old := lifecycleReceive(t, peers)
				lifecycleReceive(t, old.closed)
				require.Empty(t, old.messages, "retired dial must not publish or send text")
			}
			for _, packet := range collector.GetPackets() {
				_, failed := packet.(internal_type.TextToSpeechErrorPacket)
				require.False(t, failed, "an interrupted dial must not report a synthetic failure")
			}
			require.Empty(t, collector.EndPackets())
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			fresh := lifecycleReceive(t, peers)
			require.Equal(t, "fresh", lifecycleReceive(t, fresh.messages)["request_id"])
		})
	}
}

func TestTextToSpeech_CloseCancelsHTTPUpgrade(t *testing.T) {
	tts, collector, _ := newLifecycleSession(t, true)
	sessionDialer := websocket.DefaultDialer
	dialer := *sessionDialer
	requested, transportClosed := make(chan struct{}), make(chan struct{})
	dialer.NetDialTLSContext = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			defer close(transportClosed)
			if _, err := http.ReadRequest(bufio.NewReader(server)); err != nil {
				t.Errorf("read upgrade: %v", err)
				return
			}
			close(requested)
			if _, err := server.Read(make([]byte, 1)); err == nil {
				t.Error("expected canceled upgrade transport to close")
			}
		}()
		return client, nil
	}
	websocket.DefaultDialer = &dialer
	t.Cleanup(func() { websocket.DefaultDialer = sessionDialer })
	writer, closed := make(chan error, 1), make(chan error, 1)
	go func() {
		writer <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"})
	}()
	<-requested
	go func() { closed <- tts.Close(context.Background()) }()
	lifecycleReceive(t, transportClosed)
	require.NoError(t, lifecycleReceive(t, writer))
	require.NoError(t, lifecycleReceive(t, closed))
	for _, packet := range collector.GetPackets() {
		_, failed := packet.(internal_type.TextToSpeechErrorPacket)
		require.False(t, failed)
	}
	require.Empty(t, collector.EndPackets())
}

func TestTextToSpeech_RetiredAudioDoesNotAbortInterruptRequest(t *testing.T) {
	tts, collector, peers := newLifecycleSession(t, true)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
	peer := lifecycleReceive(t, peers)
	lifecycleReceive(t, peer.messages)
	gate := peer.transport.gateWrite()
	interrupted := make(chan error, 1)
	go func() {
		interrupted <- tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"})
	}()
	<-gate.entered
	// The second pipe write proves the reader advanced past the first retired frame.
	require.NoError(t, peer.conn.WriteMessage(websocket.BinaryMessage, []byte{1, 2}))
	require.NoError(t, peer.conn.WriteMessage(websocket.BinaryMessage, []byte{3, 4}))
	select {
	case <-peer.transport.closed:
		t.Fatal("discarding retired audio closed the interrupt writer's socket")
	default:
	}
	gate.result <- nil
	require.NoError(t, lifecycleReceive(t, interrupted))
	require.Equal(t, map[string]any{"type": "interrupt", "request_id": "old"}, lifecycleReceive(t, peer.messages))
	lifecycleReceive(t, peer.closed)
	require.NoError(t, tts.Close(context.Background()))
	require.Empty(t, collector.AudioPackets())
	require.Empty(t, collector.EndPackets())
	for _, packet := range collector.GetPackets() {
		_, failed := packet.(internal_type.TextToSpeechErrorPacket)
		require.False(t, failed, "discarded audio must not abort a configured interrupt request")
	}
}
