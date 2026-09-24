// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	cartesia "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type cartesiaContractRequest struct {
	ContextID        string `json:"context_id"`
	Transcript       string `json:"transcript"`
	Continue         bool   `json:"continue"`
	Cancel           bool   `json:"cancel"`
	ConnectionNumber int32  `json:"-"`
}

var _ = ginkgo.Describe("Cartesia TTS WebSocket contract", ginkgo.Serial,
	ginkgo.Label("offline", "tts", "cartesia", "websocket"), func() {
		var (
			tts                 internal_type.TextToSpeechTransformer
			collector           *testutil.PacketCollector
			dropFirstConnection atomic.Bool
			connectionCount     atomic.Int32
			expectedConnections int32
			expectedRequests    []cartesiaContractRequest
			expectedAudio       []internal_type.TextToSpeechAudioPacket
			expectedEndIDs      []string
			expectedErrorIDs    []string
		)

		ginkgo.BeforeEach(func() {
			tts = nil
			collector = testutil.NewPacketCollector()
			dropFirstConnection.Store(false)
			connectionCount.Store(0)
			expectedConnections = 1
			expectedRequests = nil
			expectedAudio = nil
			expectedEndIDs = nil
			expectedErrorIDs = nil

			var handlers sync.WaitGroup
			var socketsMu sync.Mutex
			var sockets []*websocket.Conn
			var closing atomic.Bool
			requests := make(chan cartesiaContractRequest, 16)
			serverFailures := make(chan error, 16)
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				handlers.Add(1)
				defer handlers.Done()
				connection, err := upgrader.Upgrade(writer, request, nil)
				if err != nil {
					serverFailures <- fmt.Errorf("upgrade local socket: %w", err)
					return
				}
				defer connection.Close()
				socketsMu.Lock()
				if closing.Load() {
					socketsMu.Unlock()
					return
				}
				sockets = append(sockets, connection)
				socketsMu.Unlock()
				connectionNumber := connectionCount.Add(1)
				for {
					if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
						serverFailures <- fmt.Errorf("set local read deadline: %w", err)
						return
					}
					var message cartesiaContractRequest
					if err := connection.ReadJSON(&message); err != nil {
						// The spec context closes the client when its subject node finishes.
						if !closing.Load() && !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							serverFailures <- fmt.Errorf("read local request: %w", err)
						}
						return
					}
					message.ConnectionNumber = connectionNumber
					select {
					case requests <- message:
					default:
						serverFailures <- fmt.Errorf("local request capacity exceeded")
						return
					}
					if err := connection.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
						serverFailures <- fmt.Errorf("set local write deadline: %w", err)
						return
					}
					switch {
					case message.Cancel:
						// These frames precede the next message's completion on this socket.
						err = connection.WriteJSON(map[string]any{
							"type": "chunk", "context_id": message.ContextID,
							"data": base64.StdEncoding.EncodeToString([]byte{3, 4}),
						})
						if err == nil {
							err = connection.WriteJSON(map[string]any{"type": "done", "context_id": message.ContextID})
						}
					case message.Continue:
						err = connection.WriteJSON(map[string]any{
							"type": "chunk", "context_id": message.ContextID,
							"data": base64.StdEncoding.EncodeToString([]byte{1, 2}),
						})
					default:
						if dropFirstConnection.Load() && connectionNumber == 1 {
							return
						}
						err = connection.WriteJSON(map[string]any{"type": "done", "context_id": message.ContextID})
					}
					if err != nil {
						serverFailures <- fmt.Errorf("write local response: %w", err)
						return
					}
				}
			}))

			previousDialer := websocket.DefaultDialer
			dialer := *previousDialer
			dialer.Proxy = nil
			dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			websocket.DefaultDialer = &dialer

			ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
				defer func() { websocket.DefaultDialer = previousDialer }()
				closing.Store(true)
				socketsMu.Lock()
				for _, connection := range sockets {
					_ = connection.Close()
				}
				socketsMu.Unlock()
				closed := make(chan error, 1)
				go func() {
					var closeError error
					if tts != nil {
						closeError = tts.Close(ctx)
					}
					server.Close()
					handlers.Wait()
					closed <- closeError
				}()
				var closeError error
				gomega.Eventually(closed).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Receive(&closeError))

				packets := collector.GetPackets()
				timeline := make([]struct {
					Kind      string `json:"kind"`
					ContextID string `json:"context_id"`
					ByteCount int    `json:"byte_count"`
				}, len(packets))
				var endIDs, errorIDs []string
				var audioPackets []internal_type.TextToSpeechAudioPacket
				for index, packet := range packets {
					timeline[index].Kind = fmt.Sprintf("%T", packet)
					timeline[index].ContextID = packet.ContextId()
					switch packet := packet.(type) {
					case internal_type.TextToSpeechAudioPacket:
						timeline[index].ByteCount = len(packet.AudioChunk)
						audioPackets = append(audioPackets, packet)
					case internal_type.TextToSpeechEndPacket:
						endIDs = append(endIDs, packet.ContextID)
					case internal_type.TextToSpeechErrorPacket:
						errorIDs = append(errorIDs, packet.ContextID)
					}
				}
				ginkgo.AddReportEntry("packet-timeline", timeline)
				gomega.Expect(closeError).NotTo(gomega.HaveOccurred())
				gomega.Expect(serverFailures).To(gomega.BeEmpty())
				gomega.Expect(connectionCount.Load()).To(gomega.Equal(expectedConnections))
				close(requests)
				var receivedRequests []cartesiaContractRequest
				for request := range requests {
					receivedRequests = append(receivedRequests, request)
				}
				gomega.Expect(receivedRequests).To(gomega.Equal(expectedRequests))
				gomega.Expect(audioPackets).To(gomega.Equal(expectedAudio))
				gomega.Expect(endIDs).To(gomega.Equal(expectedEndIDs))
				gomega.Expect(errorIDs).To(gomega.Equal(expectedErrorIDs))
				for _, packet := range packets {
					if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
						gomega.Expect(failure.Error).To(gomega.HaveOccurred())
						gomega.Expect(failure.Type).To(gomega.Equal(internal_type.TTSNetworkTimeout))
					}
				}
			}, ginkgo.NodeTimeout(6*time.Second))
		})

		ginkgo.When("text arrives before Done", func() {
			ginkgo.It("emits audio before Done and exactly one end for the message", func(ctx ginkgo.SpecContext) {
				expectedRequests = []cartesiaContractRequest{
					{ContextID: "message", Transcript: "Hello.", Continue: true, ConnectionNumber: 1},
					{ContextID: "message", ConnectionNumber: 1},
				}
				expectedAudio = []internal_type.TextToSpeechAudioPacket{{ContextID: "message", AudioChunk: []byte{1, 2}}}
				expectedEndIDs = []string{"message"}

				ginkgo.By("opening the real transformer against the local WebSocket")
				var err error
				tts, err = cartesia.NewCartesiaTextToSpeech(ctx, testutil.NewTestLogger(),
					testutil.BuildCredential(map[string]string{"key": "test-key"}), collector.OnPacket, testutil.BuildOptions(nil))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(tts.Initialize()).To(gomega.Succeed())

				ginkgo.By("admitting text and observing audio before sending Done")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "message", Text: "Hello."})).To(gomega.Succeed())
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(expectedAudio))
				gomega.Expect(collector.EndPackets()).To(gomega.BeEmpty())

				ginkgo.By("finishing the message and observing its single completion")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "message"})).To(gomega.Succeed())
				gomega.Eventually(collector.EndPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(
					[]internal_type.TextToSpeechEndPacket{{ContextID: "message"}},
				))
			}, ginkgo.SpecTimeout(10*time.Second))
		})

		ginkgo.When("a message is interrupted", func() {
			ginkgo.It("discards late old audio and end while the next message completes on the same socket", func(ctx ginkgo.SpecContext) {
				expectedRequests = []cartesiaContractRequest{
					{ContextID: "interrupted", Transcript: "First.", Continue: true, ConnectionNumber: 1},
					{ContextID: "interrupted", Cancel: true, ConnectionNumber: 1},
					{ContextID: "next", Transcript: "Next.", Continue: true, ConnectionNumber: 1},
					{ContextID: "next", ConnectionNumber: 1},
				}
				expectedAudio = []internal_type.TextToSpeechAudioPacket{
					{ContextID: "interrupted", AudioChunk: []byte{1, 2}},
					{ContextID: "next", AudioChunk: []byte{1, 2}},
				}
				expectedEndIDs = []string{"next"}

				ginkgo.By("admitting the first message and observing its initial audio")
				var err error
				tts, err = cartesia.NewCartesiaTextToSpeech(ctx, testutil.NewTestLogger(),
					testutil.BuildCredential(map[string]string{"key": "test-key"}), collector.OnPacket, testutil.BuildOptions(nil))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(tts.Initialize()).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "interrupted", Text: "First."})).To(gomega.Succeed())
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(expectedAudio[:1]))

				ginkgo.By("interrupting the message so the server sends late old audio and end")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "interrupted"})).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "interrupted"})).To(gomega.Succeed())

				ginkgo.By("completing the next message after the late frames on the same ordered socket")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "next", Text: "Next."})).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "next"})).To(gomega.Succeed())
				gomega.Eventually(collector.EndPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(
					[]internal_type.TextToSpeechEndPacket{{ContextID: "next"}},
				))
				gomega.Expect(collector.AudioPackets()).To(gomega.Equal(expectedAudio))
				gomega.Expect(connectionCount.Load()).To(gomega.Equal(int32(1)))
			}, ginkgo.SpecTimeout(10*time.Second))
		})

		ginkgo.When("the first connection drops before provider completion", func() {
			ginkgo.It("emits an error instead of completion and reconnects the same transformer for the next message", func(ctx ginkgo.SpecContext) {
				dropFirstConnection.Store(true)
				expectedConnections = 2
				expectedRequests = []cartesiaContractRequest{
					{ContextID: "failed", Transcript: "First.", Continue: true, ConnectionNumber: 1},
					{ContextID: "failed", ConnectionNumber: 1},
					{ContextID: "recovered", Transcript: "Next.", Continue: true, ConnectionNumber: 2},
					{ContextID: "recovered", ConnectionNumber: 2},
				}
				expectedAudio = []internal_type.TextToSpeechAudioPacket{
					{ContextID: "failed", AudioChunk: []byte{1, 2}},
					{ContextID: "recovered", AudioChunk: []byte{1, 2}},
				}
				expectedEndIDs = []string{"recovered"}
				expectedErrorIDs = []string{"failed"}

				ginkgo.By("receiving initial audio before dropping the first connection on Done")
				var err error
				tts, err = cartesia.NewCartesiaTextToSpeech(ctx, testutil.NewTestLogger(),
					testutil.BuildCredential(map[string]string{"key": "test-key"}), collector.OnPacket, testutil.BuildOptions(nil))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(tts.Initialize()).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "failed", Text: "First."})).To(gomega.Succeed())
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(expectedAudio[:1]))
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"})).To(gomega.Succeed())

				ginkgo.By("observing the failed message's error without successful completion")
				gomega.Eventually(func() []string {
					var errorIDs []string
					for _, packet := range collector.GetPackets() {
						if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
							errorIDs = append(errorIDs, failure.ContextID)
						}
					}
					return errorIDs
				}).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(expectedErrorIDs))
				gomega.Expect(collector.EndPackets()).To(gomega.BeEmpty())

				ginkgo.By("reusing the transformer and ignoring stale Done and interruption for the failed message")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "failed"})).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "recovered", Text: "Next."})).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "failed"})).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "recovered"})).To(gomega.Succeed())
				gomega.Eventually(collector.EndPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Equal(
					[]internal_type.TextToSpeechEndPacket{{ContextID: "recovered"}},
				))
				gomega.Expect(collector.AudioPackets()).To(gomega.Equal(expectedAudio))
				gomega.Expect(connectionCount.Load()).To(gomega.Equal(int32(2)))
			}, ginkgo.SpecTimeout(10*time.Second))
		})
	})
