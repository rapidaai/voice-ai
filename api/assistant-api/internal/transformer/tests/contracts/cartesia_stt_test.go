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

var _ = ginkgo.Describe("Cartesia STT WebSocket lifecycle", ginkgo.Serial,
	ginkgo.Label("offline", "stt", "cartesia", "websocket"), func() {
		ginkgo.DescribeTable("public factory session",
			func(ctx ginkgo.SpecContext, scenario string) {
				session, cancel := context.WithCancel(ctx)
				collector := testutil.NewPacketCollector()
				peers := make(chan *websocket.Conn, 1)
				stop := make(chan struct{})
				failures := make(chan error, 2)
				entered, released, callbackDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var callbackHeld atomic.Bool
				var handlers sync.WaitGroup
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlers.Add(1)
					defer handlers.Done()
					if r.URL.Path != "/stt/websocket" || r.URL.Query().Get("encoding") != "pcm_s16le" || r.URL.Query().Get("sample_rate") != "16000" {
						failures <- fmt.Errorf("unexpected Cartesia path or PCM format")
						http.Error(w, "invalid format", http.StatusBadRequest)
						return
					}
					conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
					if err != nil {
						failures <- err
						return
					}
					defer conn.Close()
					select {
					case peers <- conn:
					case <-stop:
						return
					}
					<-stop
				}))
				previousDialer := websocket.DefaultDialer
				dialer := *previousDialer
				dialer.Proxy = nil
				dialer.HandshakeTimeout = 2 * time.Second
				dialer.NetDialTLSContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
				}
				websocket.DefaultDialer = &dialer
				var stt internal_type.SpeechToTextTransformer
				var expected []internal_type.SpeechToTextPacket
				expectedErrors := 0
				ginkgo.DeferCleanup(func(cleanup ginkgo.SpecContext) {
					defer func() { websocket.DefaultDialer = previousDialer }()
					cancel()
					closed := make(chan error, 1)
					go func() {
						var err error
						if stt != nil {
							err = stt.Close(cleanup)
						}
						close(stop)
						server.Close()
						handlers.Wait()
						closed <- err
					}()
					gomega.Eventually(closed).WithContext(cleanup).WithTimeout(3 * time.Second).Should(gomega.Receive(gomega.BeNil()))
					if callbackHeld.Load() {
						gomega.Eventually(callbackDone).WithContext(cleanup).WithTimeout(time.Second).Should(gomega.BeClosed())
					}
					packets := collector.GetPackets()
					timeline := make([]struct {
						Kind      string `json:"kind"`
						ContextID string `json:"context_id"`
						Interim   bool   `json:"interim"`
					}, len(packets))
					var errors []internal_type.SpeechToTextErrorPacket
					for i, packet := range packets {
						timeline[i].Kind, timeline[i].ContextID = fmt.Sprintf("%T", packet), packet.ContextId()
						switch packet := packet.(type) {
						case internal_type.SpeechToTextPacket:
							timeline[i].Interim = packet.Interim
						case internal_type.SpeechToTextErrorPacket:
							errors = append(errors, packet)
						}
					}
					ginkgo.AddReportEntry("packet-timeline", timeline)
					gomega.Expect(failures).To(gomega.BeEmpty())
					gomega.Expect(collector.TranscriptPackets()).To(gomega.Equal(expected))
					gomega.Expect(errors).To(gomega.HaveLen(expectedErrors))
					for _, packet := range errors {
						gomega.Expect(packet.ContextID).To(gomega.Equal("first"))
						gomega.Expect(packet.Type).To(gomega.Equal(internal_type.STTErrorType(internal_type.STTNetworkTimeout)))
						gomega.Expect(packet.Error).To(gomega.HaveOccurred())
					}
				}, ginkgo.NodeTimeout(5*time.Second))

				ginkgo.By("opening Cartesia through the public factory against the local server")
				var err error
				stt, err = transformer.NewSpeechToText(
					transformer.WithProvider("cartesia"), transformer.WithContext(session),
					transformer.WithLogger(testutil.NewTestLogger()),
					transformer.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})),
					transformer.WithOnPacket(func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							if transcript, ok := packet.(internal_type.SpeechToTextPacket); ok && transcript.Interim && scenario == "callback" && callbackHeld.CompareAndSwap(false, true) {
								close(entered)
								defer close(callbackDone)
								select {
								case <-released:
								case <-session.Done():
								}
							}
						}
						return collector.OnPacket(packets...)
					}))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(stt.Initialize()).To(gomega.Succeed())
				var peer *websocket.Conn
				gomega.Eventually(peers).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&peer))
				gomega.Expect(peer.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
				gomega.Expect(peer.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())

				ginkgo.By("sending exact PCM bytes for the first message")
				gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "first"})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 0, 2, 0}})).To(gomega.Succeed())
				kind, pcm, err := peer.ReadMessage()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
				gomega.Expect(pcm).To(gomega.Equal([]byte{1, 0, 2, 0}))

				if scenario == "provider" || scenario == "malformed" || scenario == "disconnect" {
					ginkgo.By("reporting a provider or transport failure without fabricating a transcript")
					expectedErrors = 1
					switch scenario {
					case "provider":
						gomega.Expect(peer.WriteJSON(map[string]string{"type": "error", "message": "offline failure"})).To(gomega.Succeed())
					case "malformed":
						gomega.Expect(peer.WriteMessage(websocket.TextMessage, []byte("{"))).To(gomega.Succeed())
					case "disconnect":
						gomega.Expect(peer.Close()).To(gomega.Succeed())
					}
					gomega.Eventually(collector.GetPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{})))
					return
				}

				ginkgo.By("admitting an interim result before changing ownership")
				gomega.Expect(peer.WriteJSON(map[string]any{"type": "transcript", "text": "Partial", "language": "en", "is_final": false})).To(gomega.Succeed())
				expected = []internal_type.SpeechToTextPacket{{ContextID: "first", Script: "Partial", Language: "en", Interim: true}}
				if scenario == "callback" {
					gomega.Eventually(entered).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
					ginkgo.By("changing turns while the admitted callback is held, then releasing its captured owner")
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
					close(released)
					gomega.Eventually(callbackDone).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
				}
				gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expected))
				if scenario == "flow" {
					ginkgo.By("sending EOS as a finalize frame and waiting for actual final transcription")
					gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: "first"})).To(gomega.Succeed())
					kind, payload, err := peer.ReadMessage()
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(kind).To(gomega.Equal(websocket.TextMessage))
					gomega.Expect(string(payload)).To(gomega.Equal("finalize"))
					gomega.Expect(peer.WriteJSON(map[string]any{"type": "flush_done"})).To(gomega.Succeed())
					gomega.Expect(peer.WriteJSON(map[string]any{"type": "transcript", "text": "Final", "language": "en", "is_final": true})).To(gomega.Succeed())
					expected = append(expected, internal_type.SpeechToTextPacket{ContextID: "first", Script: "Final", Language: "en"})
					gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expected))
					ginkgo.By("assigning a later wire result to the new current turn")
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
				}
				gomega.Expect(peer.WriteJSON(map[string]any{"type": "transcript", "text": "Later", "language": "en", "is_final": true})).To(gomega.Succeed())
				expected = append(expected, internal_type.SpeechToTextPacket{ContextID: "next", Script: "Later", Language: "en"})
				gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expected))

				ginkgo.By("closing the session with a canceled caller and observing socket shutdown")
				cancel()
				closed := make(chan error, 1)
				go func() { closed <- stt.Close(session) }()
				gomega.Eventually(closed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(gomega.BeNil()))
				gomega.Expect(peer.SetReadDeadline(time.Now().Add(time.Second))).To(gomega.Succeed())
				_, _, err = peer.ReadMessage()
				gomega.Expect(websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure)).To(gomega.BeTrue())
			},
			ginkgo.Entry("frames PCM, finalizes EOS, and assigns delayed wire results to the current turn", "flow", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("retains ownership of a callback admitted before a turn change", "callback", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("reports provider failure with message ownership", "provider", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("reports malformed response failure without transcript output", "malformed", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("reports unexpected disconnect without transcript output", "disconnect", ginkgo.SpecTimeout(8*time.Second)),
		)
	})
