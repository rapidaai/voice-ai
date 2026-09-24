// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
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
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("Deepgram TTS Clear acknowledgement contract", ginkgo.Serial,
	ginkgo.Label("offline", "tts", "deepgram", "websocket"), func() {
		ginkgo.DescribeTable("when the server receives Clear but withholds Cleared",
			func(ctx ginkgo.SpecContext, cancellation string) {
				session, cancelSession := context.WithCancel(ctx)
				collector := testutil.NewPacketCollector()
				var tts internal_type.TextToSpeechTransformer
				var handlers, calls sync.WaitGroup
				var socketsMu sync.Mutex
				var sockets []*websocket.Conn
				var closing atomic.Bool
				var connections atomic.Int32
				clearReceived := make(chan time.Time, 1)
				oldSocketClosed := make(chan time.Time, 1)
				requests := make(chan string, 16)
				serverFailures := make(chan error, 8)
				upgrader := websocket.Upgrader{}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlers.Add(1)
					defer handlers.Done()
					conn, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						serverFailures <- fmt.Errorf("upgrade local socket: %w", err)
						return
					}
					defer conn.Close()
					socketsMu.Lock()
					if closing.Load() {
						socketsMu.Unlock()
						return
					}
					sockets = append(sockets, conn)
					socketsMu.Unlock()
					number := connections.Add(1)
					for {
						if err := conn.SetReadDeadline(time.Now().Add(8 * time.Second)); err != nil {
							serverFailures <- err
							return
						}
						var request struct {
							Type string `json:"type"`
						}
						if err := conn.ReadJSON(&request); err != nil {
							if number == 1 {
								oldSocketClosed <- time.Now()
							}
							if timeout, ok := err.(net.Error); ok && timeout.Timeout() && !closing.Load() {
								serverFailures <- fmt.Errorf("local socket did not close: %w", err)
							}
							return
						}
						if request.Type == "Close" {
							return
						}
						select {
						case requests <- fmt.Sprintf("%d:%s", number, request.Type):
						default:
							serverFailures <- fmt.Errorf("local request capacity exceeded")
							return
						}
						if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
							serverFailures <- err
							return
						}
						switch request.Type {
						case "Speak":
							err = conn.WriteMessage(websocket.BinaryMessage, []byte{byte(number), 0})
						case "Clear":
							// Late output must not complete or feed the pending replacement.
							err = conn.WriteMessage(websocket.BinaryMessage, []byte{9, 0})
							if err == nil {
								err = conn.WriteJSON(map[string]string{"type": "Flushed"})
							}
							select {
							case clearReceived <- time.Now():
							case <-session.Done():
								return
							}
						case "Flush":
							err = conn.WriteJSON(map[string]string{"type": "Flushed"})
						default:
							err = fmt.Errorf("unexpected local command %q", request.Type)
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
				dialer.NetDialTLSContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
				}
				websocket.DefaultDialer = &dialer
				expectedAudio := []internal_type.TextToSpeechAudioPacket{{ContextID: "old", AudioChunk: []byte{1, 0}}}
				var expectedEnds []internal_type.TextToSpeechEndPacket
				expectedRequests := []string{"1:Speak", "1:Clear"}
				ginkgo.DeferCleanup(func(cleanup ginkgo.SpecContext) {
					defer func() { websocket.DefaultDialer = previousDialer }()
					closing.Store(true)
					cancelSession()
					socketsMu.Lock()
					for _, conn := range sockets {
						_ = conn.Close()
					}
					socketsMu.Unlock()
					closed := make(chan error, 1)
					go func() {
						var err error
						if tts != nil {
							err = tts.Close(cleanup)
						}
						calls.Wait()
						server.Close()
						handlers.Wait()
						closed <- err
					}()
					var closeError error
					gomega.Eventually(closed).WithContext(cleanup).WithTimeout(4 * time.Second).Should(gomega.Receive(&closeError))
					packets := collector.GetPackets()
					timeline := make([]struct {
						Kind      string `json:"kind"`
						ContextID string `json:"context_id"`
						ByteCount int    `json:"byte_count"`
					}, len(packets))
					var errorIDs []string
					for i, packet := range packets {
						timeline[i].Kind = fmt.Sprintf("%T", packet)
						timeline[i].ContextID = packet.ContextId()
						switch packet := packet.(type) {
						case internal_type.TextToSpeechAudioPacket:
							timeline[i].ByteCount = len(packet.AudioChunk)
						case internal_type.TextToSpeechErrorPacket:
							errorIDs = append(errorIDs, packet.ContextID)
						}
					}
					ginkgo.AddReportEntry("packet-timeline", timeline)
					gomega.Expect(closeError).NotTo(gomega.HaveOccurred())
					gomega.Expect(serverFailures).To(gomega.BeEmpty())
					gomega.Expect(errorIDs).To(gomega.BeEmpty())
					gomega.Expect(collector.AudioPackets()).To(gomega.Equal(expectedAudio))
					gomega.Expect(collector.EndPackets()).To(gomega.Equal(expectedEnds))
					close(requests)
					var received []string
					for request := range requests {
						received = append(received, request)
					}
					gomega.Expect(received).To(gomega.Equal(expectedRequests))
				}, ginkgo.NodeTimeout(5*time.Second))

				ginkgo.By("opening the public factory transformer and receiving old-message audio")
				var err error
				tts, err = transformer.GetTextToSpeechTransformer(session, testutil.NewTestLogger(), "deepgram",
					testutil.BuildCredential(map[string]string{"key": "offline-key"}), collector.OnPacket, testutil.BuildOptions(nil))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(tts.Initialize()).To(gomega.Succeed())
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "First."})).To(gomega.Succeed())
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expectedAudio))

				ginkgo.By("requesting replacement text and observing Clear without acknowledging it")
				caller, cancelCaller := context.WithCancel(ctx)
				defer cancelCaller()
				result := make(chan error, 1)
				started := time.Now()
				calls.Go(func() {
					result <- tts.Transform(caller, internal_type.TextToSpeechTextPacket{ContextID: "next", Text: "Next."})
				})
				var clearAt time.Time
				gomega.Eventually(clearReceived).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&clearAt))
				gomega.Consistently(result).WithContext(ctx).WithTimeout(100 * time.Millisecond).ShouldNot(gomega.Receive())
				gomega.Expect(collector.EndPackets()).To(gomega.BeEmpty())

				if cancellation != "none" {
					ginkgo.By("canceling the waiting caller or session before the Clear timeout")
					if cancellation == "caller" {
						cancelCaller()
					} else {
						cancelSession()
					}
					var callError error
					gomega.Eventually(result).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&callError))
					gomega.Expect(callError).To(gomega.MatchError(context.Canceled))
					gomega.Expect(connections.Load()).To(gomega.Equal(int32(1)))
					if cancellation == "session" {
						ginkgo.By("observing session socket shutdown and rejecting further text")
						gomega.Eventually(oldSocketClosed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive())
						gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "after-cancel", Text: "Later."})).To(gomega.MatchError(context.Canceled))
						return
					}
					ginkgo.By("retrying on the same live session after caller cancellation")
					started = time.Now()
					calls.Go(func() {
						result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "next", Text: "Next."})
					})
				}

				ginkgo.By("observing the two-second timeout close the old socket and admit replacement text")
				var callError error
				gomega.Eventually(result).WithContext(ctx).WithTimeout(4 * time.Second).Should(gomega.Receive(&callError))
				gomega.Expect(callError).NotTo(gomega.HaveOccurred())
				var closedAt time.Time
				gomega.Eventually(oldSocketClosed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&closedAt))
				gomega.Expect(closedAt.Sub(started)).To(gomega.BeNumerically(">=", 1900*time.Millisecond))
				gomega.Expect(closedAt.Sub(started)).To(gomega.BeNumerically("<", 4*time.Second))
				gomega.Expect(closedAt.After(clearAt)).To(gomega.BeTrue())
				gomega.Expect(connections.Load()).To(gomega.Equal(int32(2)))
				expectedRequests = append(expectedRequests, "2:Speak", "2:Flush")
				expectedAudio = append(expectedAudio, internal_type.TextToSpeechAudioPacket{ContextID: "next", AudioChunk: []byte{2, 0}})
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expectedAudio))

				ginkgo.By("completing only the replacement message with correctly owned audio and end")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "next"})).To(gomega.Succeed())
				expectedEnds = []internal_type.TextToSpeechEndPacket{{ContextID: "next"}}
				gomega.Eventually(collector.EndPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expectedEnds))
			},
			ginkgo.Entry("replaces the socket after the implemented two-second timeout", "none", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("returns caller cancellation and allows the same session to recover", "caller", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("returns session cancellation and closes the waiting socket", "session", ginkgo.SpecTimeout(10*time.Second)),
		)
	})
