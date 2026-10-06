package internal_transformer_resembleai

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

type resembleAITTSPipeResponse struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (w *resembleAITTSPipeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

type resembleAITTSWriteGate struct {
	entered chan struct{}
	result  chan error
}

type resembleAITTSGatedConn struct {
	net.Conn
	gateMu      sync.Mutex
	writeGate   *resembleAITTSWriteGate
	readEntered chan struct{}
	readRelease chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func (c *resembleAITTSGatedConn) gateWrite() *resembleAITTSWriteGate {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.writeGate = &resembleAITTSWriteGate{entered: make(chan struct{}), result: make(chan error, 1)}
	return c.writeGate
}

func (c *resembleAITTSGatedConn) gateRead() (<-chan struct{}, chan<- struct{}) {
	c.gateMu.Lock()
	defer c.gateMu.Unlock()
	c.readEntered, c.readRelease = make(chan struct{}), make(chan struct{})
	return c.readEntered, c.readRelease
}

func (c *resembleAITTSGatedConn) Read(payload []byte) (int, error) {
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

func (c *resembleAITTSGatedConn) Write(payload []byte) (int, error) {
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

func (c *resembleAITTSGatedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() { close(c.closed) })
	return err
}

type resembleAITTSPeer struct {
	conn      *websocket.Conn
	transport *resembleAITTSGatedConn
	request   *http.Request
	messages  chan map[string]interface{}
	closed    chan struct{}
}

func newResembleAITTSSession(t *testing.T) (*resembleaiTTS, *testutil.PacketCollector, <-chan *resembleAITTSPeer) {
	t.Helper()
	peers := make(chan *resembleAITTSPeer, 16)
	var servers sync.WaitGroup
	previousDialer := websocket.DefaultDialer
	dialer := *previousDialer
	dialer.Proxy = nil
	dialer.NetDialTLSContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		transport := &resembleAITTSGatedConn{Conn: client, closed: make(chan struct{})}
		servers.Go(func() {
			defer server.Close()
			request, err := http.ReadRequest(bufio.NewReader(server))
			if err != nil {
				return
			}
			response := &resembleAITTSPipeResponse{ResponseRecorder: httptest.NewRecorder(), conn: server}
			conn, err := (&websocket.Upgrader{}).Upgrade(response, request, nil)
			if err != nil {
				t.Errorf("upgrade websocket: %v", err)
				return
			}
			peer := &resembleAITTSPeer{conn: conn, transport: transport, request: request, messages: make(chan map[string]interface{}, 32), closed: make(chan struct{})}
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
	transformer, err := NewResembleAITextToSpeech(context.Background(), testutil.NewTestLogger(), testutil.BuildCredential(map[string]string{"key": "test-key"}), collector.OnPacket, utils.Option{"speak.voice.id": "test-voice"})
	require.NoError(t, err)
	tts := transformer.(*resembleaiTTS)
	t.Cleanup(func() {
		require.NoError(t, tts.Close(context.Background()))
		servers.Wait()
		websocket.DefaultDialer = previousDialer
	})
	return tts, collector, peers
}

func resembleAITTSReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ResembleAI TTS test event")
		var zero T
		return zero
	}
}

func resembleAITTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var packets []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			packets = append(packets, packet)
		}
	}
	return packets
}

func TestResembleAITTSSequentialResponsesAndPendingDrains(t *testing.T) {
	for _, doneFirst := range []bool{false, true} {
		name := "drained_before_done"
		if doneFirst {
			name = "done_before_drain"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Initialize())
				peer := resembleAITTSReceive(t, peers)
				require.NoError(t, tts.Initialize())
				require.Empty(t, peers)
				require.Equal(t, "Bearer test-key", peer.request.Header.Get("Authorization"))
				require.Equal(t, "/stream", peer.request.URL.Path)
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Empty(t, collector.EndPackets(), "an idle audio_end must be ignored")
				for index, id := range []string{"old", "fresh"} {
					for _, text := range []string{"first", "second"} {
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: text}))
						if index == 1 && text == "first" {
							peer = resembleAITTSReceive(t, peers)
						}
						require.Equal(t, map[string]interface{}{
							"voice_uuid": "test-voice", "data": text, "output_format": "wav",
							"sample_rate": float64(16000), "precision": "PCM_16", "no_audio_header": true,
						}, resembleAITTSReceive(t, peer.messages))
					}
					for _, stale := range []string{"", "stale"} {
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: stale}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: stale}))
					}
					if doneFirst {
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
					}
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
					synctest.Wait()
					require.Len(t, collector.EndPackets(), index, "the second request has not drained")
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AwQ="}))
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
					synctest.Wait()
					if !doneFirst {
						require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
						synctest.Wait()
						require.Len(t, collector.EndPackets(), index, "an extra audio_end must not close open text")
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
					}
					synctest.Wait()
					require.Len(t, collector.AudioPackets(), (index+1)*2)
					require.Equal(t, id, collector.AudioPackets()[index*2+1].ContextID)
					require.Len(t, collector.EndPackets(), index+1)
					require.Equal(t, id, collector.EndPackets()[index].ContextID)
					resembleAITTSReceive(t, peer.closed)
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: id}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "late"}))
					require.Empty(t, peers)
					require.Empty(t, peer.messages, "Done must not write a protocol command")
				}
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSCompletionWaitsForTextWrite(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				peer := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				written, done := make(chan error, 1), make(chan error, 1)
				go func() {
					written <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "second"})
				}()
				<-gate.entered
				go func() {
					done <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}()
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Empty(t, written)
				require.Empty(t, done)
				require.Empty(t, collector.EndPackets())
				if failed {
					gate.result <- errors.New("text write failed")
				} else {
					gate.result <- nil
				}
				require.NoError(t, <-written)
				require.NoError(t, <-done)
				synctest.Wait()
				if failed {
					require.Empty(t, collector.EndPackets())
					require.Len(t, resembleAITTSErrors(collector), 1)
				} else {
					require.Len(t, collector.EndPackets(), 1)
					require.Empty(t, resembleAITTSErrors(collector))
					require.Equal(t, "second", resembleAITTSReceive(t, peer.messages)["data"])
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "retry"}))
				require.Empty(t, peers)
			})
		})
	}
}

func TestResembleAITTSGatedWriteCancellation(t *testing.T) {
	for _, cancellation := range []string{"interrupt", "caller", "session", "close", "replacement"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				old := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, old.messages)
				gate := old.transport.gateWrite()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
				}()
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
				resembleAITTSReceive(t, old.closed)
				require.Empty(t, resembleAITTSErrors(collector))
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
				fresh := resembleAITTSReceive(t, peers)
				require.Equal(t, "fresh", resembleAITTSReceive(t, fresh.messages)["data"])
				gate = fresh.transport.gateWrite()
				go func() {
					result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "blocked again"})
				}()
				<-gate.entered
				cancel()
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				synctest.Wait()
				require.Empty(t, result, "stale cancellation must not release the replacement writer")
				select {
				case <-fresh.transport.closed:
					t.Fatal("stale cancellation closed replacement")
				default:
				}
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
				require.NoError(t, <-result)
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSDoneBeforeAdmittedTextWrites(t *testing.T) {
	for _, warm := range []bool{false, true} {
		name := "cold"
		if warm {
			name = "warm"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				if warm {
					require.NoError(t, tts.Initialize())
				}
				require.NoError(t, tts.writeLock.Acquire(context.Background(), 1))
				releaseWriter := sync.OnceFunc(func() { tts.writeLock.Release(1) })
				t.Cleanup(releaseWriter)
				done, written := make(chan error, 1), make(chan error, 1)
				go func() {
					done <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}()
				synctest.Wait()
				require.Empty(t, done)
				go func() {
					written <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "queued"})
				}()
				synctest.Wait()
				require.Empty(t, written)
				releaseWriter()
				require.NoError(t, <-done)
				require.NoError(t, <-written)
				require.Empty(t, collector.EndPackets(), "Done must wait for the admitted text to drain")
				peer := resembleAITTSReceive(t, peers)
				require.Equal(t, "queued", resembleAITTSReceive(t, peer.messages)["data"])
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
				require.Empty(t, peer.messages, "new text after Done must not be admitted")
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "old", collector.AudioPackets()[0].ContextID)
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "old", collector.EndPackets()[0].ContextID)
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSFinalDrainWaitsForAdmittedWriteResult(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Initialize())
				peer := resembleAITTSReceive(t, peers)
				gate := peer.transport.gateWrite()
				require.NoError(t, tts.writeLock.Acquire(context.Background(), 1))
				releaseWriter := sync.OnceFunc(func() { tts.writeLock.Release(1) })
				t.Cleanup(releaseWriter)
				done, written := make(chan error, 1), make(chan error, 1)
				go func() {
					done <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
				}()
				synctest.Wait()
				go func() {
					written <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "queued"})
				}()
				synctest.Wait()
				releaseWriter()
				require.NoError(t, <-done)
				<-gate.entered
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Empty(t, collector.EndPackets(), "audio_end cannot turn a failed pending write into success")
				require.Empty(t, written)
				if failed {
					gate.result <- errors.New("text write failed after audio_end")
				} else {
					gate.result <- nil
				}
				require.NoError(t, <-written)
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				if failed {
					require.Empty(t, collector.EndPackets())
					require.Len(t, resembleAITTSErrors(collector), 1)
				} else {
					require.Len(t, collector.EndPackets(), 1)
					require.Empty(t, resembleAITTSErrors(collector))
				}
			})
		})
	}
}

func TestResembleAITTSReaderFailureClosesBeforeCallback(t *testing.T) {
	for _, failure := range []string{"provider", "decode", "json", "transport"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
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
				peer := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, peer.messages)
				gate := peer.transport.gateWrite()
				result := make(chan error, 1)
				go func() {
					result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
				}()
				<-gate.entered
				switch failure {
				case "provider":
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "error", "message": "rejected"}))
				case "decode":
					require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "!!!"}))
				case "json":
					require.NoError(t, peer.conn.WriteMessage(websocket.TextMessage, []byte("{")))
				case "transport":
					require.NoError(t, peer.conn.Close())
				}
				resembleAITTSReceive(t, entered)
				resembleAITTSReceive(t, peer.transport.closed)
				require.NoError(t, resembleAITTSReceive(t, result), "the reader must close before blocking in error publication")
				require.Len(t, resembleAITTSErrors(collector), 1)
				require.Equal(t, "old", resembleAITTSErrors(collector)[0].ContextID)
				if failure == "provider" {
					require.ErrorContains(t, resembleAITTSErrors(collector)[0].Error, "rejected")
				}
				release()
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "retry"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Empty(t, peers)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := resembleAITTSReceive(t, peers)
				require.Equal(t, "fresh", resembleAITTSReceive(t, fresh.messages)["data"])
				synctest.Wait()
				require.Len(t, resembleAITTSErrors(collector), 1)
				require.Empty(t, collector.EndPackets())
			})
		})
	}
}

func TestResembleAITTSStaleReaderCannotAffectReplacement(t *testing.T) {
	for _, frame := range []string{"audio", "error", "decode", "audio_end", "transport"} {
		t.Run(frame, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				old := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, old.messages)
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				entered, unblock := old.transport.gateRead()
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				switch frame {
				case "audio":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
				case "error":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "error", "message": "old failure"}))
				case "decode":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "!!!"}))
				case "audio_end":
					require.NoError(t, old.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				case "transport":
					require.NoError(t, old.conn.Close())
				}
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, fresh.messages)
				release()
				synctest.Wait()
				require.Empty(t, collector.AudioPackets())
				require.Empty(t, collector.EndPackets())
				require.Empty(t, resembleAITTSErrors(collector))
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AwQ="}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSAudioCallbackOrderingAndCloseJoin(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		name := "end_after_audio"
		if shutdown {
			name = "close_joins_audio"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
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
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
				peer := resembleAITTSReceive(t, peers)
				resembleAITTSReceive(t, peer.messages)
				require.NoError(t, peer.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
				<-entered
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Empty(t, peer.messages)
				result := make(chan error, 1)
				if shutdown {
					go func() { result <- tts.Close(context.Background()) }()
					resembleAITTSReceive(t, peer.transport.closed)
				} else {
					go func() { result <- peer.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}) }()
				}
				synctest.Wait()
				require.Empty(t, result)
				require.Empty(t, collector.EndPackets())
				release()
				require.NoError(t, resembleAITTSReceive(t, result))
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
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSQueuedCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newResembleAITTSSession(t)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
		peer := resembleAITTSReceive(t, peers)
		resembleAITTSReceive(t, peer.messages)
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
		require.Empty(t, owner, "canceling queued Done must not cancel the active text writer")
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, <-owner)
		require.Empty(t, resembleAITTSErrors(collector))
	})
}

func TestResembleAITTSQueuedTextCancellation(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "retires_input"
		if replacement {
			name = "preserves_replacement"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, peers := newResembleAITTSSession(t)
				require.NoError(t, tts.Initialize())
				old := resembleAITTSReceive(t, peers)
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
				resembleAITTSReceive(t, old.closed)
				fresh := resembleAITTSReceive(t, peers)
				require.Equal(t, "fresh", resembleAITTSReceive(t, fresh.messages)["data"])
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "audio", "audio_content": "AQI="}))
				require.NoError(t, fresh.conn.WriteJSON(map[string]interface{}{"type": "audio_end"}))
				synctest.Wait()
				require.Len(t, collector.AudioPackets(), 1)
				require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				require.Len(t, collector.EndPackets(), 1)
				require.Equal(t, "fresh", collector.EndPackets()[0].ContextID)
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSDialUpgradeCancellation(t *testing.T) {
	for _, cancellation := range []string{"caller", "session", "close", "initialize"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tts, collector, _ := newResembleAITTSSession(t)
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
					result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"})
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
				require.ErrorIs(t, resembleAITTSReceive(t, result), context.Canceled)
				resembleAITTSReceive(t, closed)
				require.Empty(t, collector.EndPackets())
				require.Empty(t, resembleAITTSErrors(collector))
			})
		})
	}
}

func TestResembleAITTSInterruptedConnectDoesNotSendRetiredText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newResembleAITTSSession(t)
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
			result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"})
		}()
		<-entered
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		release()
		require.NoError(t, <-result)
		websocket.DefaultDialer = sessionDialer
		peer := resembleAITTSReceive(t, peers)
		require.Empty(t, peer.messages)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Empty(t, peer.messages)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		require.Equal(t, "fresh", resembleAITTSReceive(t, peer.messages)["data"])
		require.Empty(t, peers, "an unused connection can serve the replacement")
		require.Empty(t, resembleAITTSErrors(collector))
	})
}

func TestResembleAITTSConnectFailureIsNotReplayed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector, peers := newResembleAITTSSession(t)
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
		require.Len(t, resembleAITTSErrors(collector), 1)
		require.Equal(t, "old", resembleAITTSErrors(collector)[0].ContextID)
		require.Empty(t, collector.EndPackets())
		websocket.DefaultDialer = sessionDialer
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		fresh := resembleAITTSReceive(t, peers)
		require.Equal(t, "fresh", resembleAITTSReceive(t, fresh.messages)["data"])
	})
}
