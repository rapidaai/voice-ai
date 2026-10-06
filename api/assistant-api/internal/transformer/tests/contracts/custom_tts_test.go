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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

var _ = ginkgo.Describe("Custom TTS WebSocket response DSL contract",
	ginkgo.Label("offline", "tts", "custom", "websocket"), func() {
		ginkgo.DescribeTable("response context ownership",
			func(ctx ginkgo.SpecContext, mode string) {
				session, cancelSession := context.WithCancel(ctx)
				collector := testutil.NewPacketCollector()
				var tts internal_type.TextToSpeechTransformer
				var handlers sync.WaitGroup
				var socketsMu sync.Mutex
				var sockets []*websocket.Conn
				var closing atomic.Bool
				var connections atomic.Int32
				requests := make(chan string, 8)
				serverFailures := make(chan error, 8)
				peerClosed := make(chan struct{}, 1)
				expectedID := "provider-context"
				if mode == "absent" || mode == "unmapped" || mode == "empty" {
					expectedID = "input-context"
				}
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
					connections.Add(1)
					for {
						if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
							serverFailures <- err
							return
						}
						var request struct {
							Type      string `json:"type"`
							RequestID string `json:"request_id"`
						}
						if err := conn.ReadJSON(&request); err != nil {
							select {
							case peerClosed <- struct{}{}:
							case <-session.Done():
							}
							return
						}
						select {
						case requests <- request.Type + ":" + request.RequestID:
						default:
							serverFailures <- fmt.Errorf("local request capacity exceeded")
							return
						}
						if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
							serverFailures <- err
							return
						}
						if mode == "filtered" {
							// The configured predicate, not an implicit ID registry, rejects these frames.
							for _, id := range []string{"stale-context", "unknown-context"} {
								if err := conn.WriteJSON(map[string]any{
									"type": "result", "request_id": id, "done": true,
									"audio": base64.StdEncoding.EncodeToString([]byte{9, 0}),
								}); err != nil {
									serverFailures <- fmt.Errorf("write unmatched response: %w", err)
									return
								}
							}
						}
						response := map[string]any{"type": "result", "done": false, "audio": ""}
						if mode != "absent" {
							response["request_id"] = "provider-context"
						}
						if mode == "empty" {
							response["request_id"] = ""
						}
						if mode == "retired" && request.RequestID == "stale-context" {
							response["request_id"] = "stale-context"
						}
						switch request.Type {
						case "text":
							response["audio"] = base64.StdEncoding.EncodeToString([]byte{1, 0, 2, 0})
						case "done":
							response["done"] = true
						default:
							serverFailures <- fmt.Errorf("unexpected local command %q", request.Type)
							return
						}
						if err := conn.WriteJSON(response); err != nil {
							serverFailures <- fmt.Errorf("write matched response: %w", err)
							return
						}
					}
				}))
				expectedAudio := []internal_type.TextToSpeechAudioPacket{{ContextID: expectedID, AudioChunk: []byte{1, 0, 2, 0}}}
				expectedEnds := []internal_type.TextToSpeechEndPacket{{ContextID: expectedID}}
				expectedConnections := int32(1)
				expectedRequests := []string{"text:input-context", "done:input-context"}
				ginkgo.DeferCleanup(func(cleanup ginkgo.SpecContext) {
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
					gomega.Expect(connections.Load()).To(gomega.Equal(expectedConnections))
					close(requests)
					var received []string
					for request := range requests {
						received = append(received, request)
					}
					gomega.Expect(received).To(gomega.Equal(expectedRequests))
				}, ginkgo.NodeTimeout(5*time.Second))

				ginkgo.By("configuring the public factory with response mapping and a local WebSocket endpoint")
				when := `"path":"type","equals":"result"`
				if mode == "filtered" || mode == "absent" {
					when = `"path":"request_id","equals":"provider-context"`
				}
				mapping := `,"message_id":{"$path":"request_id"}`
				if mode == "unmapped" {
					mapping = ""
				}
				fallback := ""
				if mode == "absent" {
					// Missing emit paths are errors; absence is handled by a later matching rule.
					fallback = `,{"when":{"frame":"json","path":"type","equals":"result"},"emit":{
						"audio":{"$decode":"base64","value":{"$path":"audio"}},"done":{"$path":"done"}}}`
				}
				var err error
				tts, err = transformer.GetTextToSpeechTransformer(session, testutil.NewTestLogger(), "custom-tts",
					testutil.BuildCredential(map[string]string{
						"base_url": "ws" + strings.TrimPrefix(server.URL, "http"), "api_compatibility": "websocket_v1",
					}), collector.OnPacket, utils.Option{
						internal_options.SpeakOptionRequestRules: `[
							{"when":{"packet":"text"},"send":{"frame":"json","body":{"type":"text","text":{"$path":"packet.text"},"request_id":{"$path":"packet.message_id"}}}},
							{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done","request_id":{"$path":"packet.message_id"}}}}
						]`,
						internal_options.SpeakOptionResponseRules: `[{"when":{"frame":"json",` + when + `},"emit":{
							"audio":{"$decode":"base64","value":{"$path":"audio"}},"done":{"$path":"done"}` + mapping + `}}` + fallback + `]`,
					})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(tts.Initialize()).To(gomega.Succeed())

				if mode == "retired" {
					ginkgo.By("retiring an actual prior owner by interruption before admitting the next message")
					priorAudio := []internal_type.TextToSpeechAudioPacket{{ContextID: "stale-context", AudioChunk: []byte{1, 0, 2, 0}}}
					expectedAudio = append(priorAudio, expectedAudio...)
					expectedConnections = 2
					expectedRequests = append([]string{"text:stale-context"}, expectedRequests...)
					gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "stale-context", Text: "Earlier."})).To(gomega.Succeed())
					gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(priorAudio))
					gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "stale-context"})).To(gomega.Succeed())
					gomega.Eventually(peerClosed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive())
					gomega.Expect(collector.EndPackets()).To(gomega.BeEmpty())
				}

				ginkgo.By("receiving only matched audio under the configured or fallback context before Done")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "input-context", Text: "First."})).To(gomega.Succeed())
				gomega.Eventually(collector.AudioPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expectedAudio))
				gomega.Expect(collector.EndPackets()).To(gomega.BeEmpty())

				ginkgo.By("ignoring stale and unknown input controls without retiring the active input owner")
				for _, id := range []string{"stale-context", "unknown-context"} {
					gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: id})).To(gomega.Succeed())
					gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: id})).To(gomega.Succeed())
				}

				ginkgo.By("accepting valid completion using the same response mapping and closing the completed socket")
				gomega.Expect(tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "input-context"})).To(gomega.Succeed())
				gomega.Eventually(collector.EndPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expectedEnds))
				gomega.Eventually(peerClosed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive())
			},
			ginkgo.Entry("maps response IDs independently of the wire input ID", "mapped", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("falls through to the configured input-owner rule when the response ID is absent", "absent", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("falls back to the input owner when the mapped response ID is empty", "empty", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("uses the input owner when no response ID mapping is configured", "unmapped", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("filters stale and unknown response IDs through the configured DSL predicate", "filtered", ginkgo.SpecTimeout(8*time.Second)),
			ginkgo.Entry("ignores controls for an interrupted prior owner while completing the next message", "retired", ginkgo.SpecTimeout(8*time.Second)),
		)
	})
