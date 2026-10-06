// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"errors"
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

var _ = ginkgo.Describe("Smallest STT WebSocket contract", ginkgo.Serial,
	ginkgo.Label("offline", "stt", "smallest", "websocket"), func() {
		var (
			stt           internal_type.SpeechToTextTransformer
			collector     *testutil.PacketCollector
			connections   chan *websocket.Conn
			rejectUpgrade atomic.Bool
			cancelSession context.CancelFunc
			closeDone     chan struct{}
		)

		ginkgo.BeforeEach(func() {
			collector = testutil.NewPacketCollector()
			connections = make(chan *websocket.Conn, 1)
			rejectUpgrade.Store(false)
			closeDone = nil
			stop := make(chan struct{})
			failures := make(chan error, 8)
			var handlers sync.WaitGroup
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlers.Add(1)
				defer handlers.Done()
				query := r.URL.Query()
				if r.URL.Path != "/waves/v1/pulse/get_text" || query.Get("encoding") != "linear16" ||
					query.Get("sample_rate") != "16000" || query.Get("language") != "en" || query.Get("word_timestamps") != "true" ||
					r.Header.Get("Authorization") != "Bearer offline-key" || r.Header.Get("X-Source") != "rapida" {
					failures <- fmt.Errorf("unexpected Smallest startup path, configuration, or authentication")
					http.Error(w, "invalid startup", http.StatusBadRequest)
					return
				}
				if rejectUpgrade.Load() {
					http.Error(w, "offline rejection", http.StatusForbidden)
					return
				}
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					failures <- fmt.Errorf("upgrade Smallest socket: %w", err)
					return
				}
				defer conn.Close()
				select {
				case connections <- conn:
				case <-stop:
					return
				}
				<-stop
			}))
			previousDialer := websocket.DefaultDialer
			dialer := *previousDialer
			dialer.Proxy = nil
			dialer.NetDialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			websocket.DefaultDialer = &dialer
			var sessionCtx context.Context
			sessionCtx, cancelSession = context.WithCancel(context.Background())
			stt = nil
			ginkgo.DeferCleanup(func() {
				defer func() { websocket.DefaultDialer = previousDialer }()
				cancelSession()
				close(stop)
				server.Close()
				handlers.Wait()
				if closeDone != nil {
					gomega.Eventually(closeDone, 3*time.Second).Should(gomega.BeClosed())
				}
				if stt != nil {
					gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed())
				}
				packets := collector.GetPackets()
				timeline := make([]struct {
					Kind      string `json:"kind"`
					ContextID string `json:"context_id"`
					Interim   bool   `json:"interim"`
				}, len(packets))
				for i, packet := range packets {
					timeline[i].Kind = string(packet.PacketName())
					timeline[i].ContextID = packet.ContextId()
					if transcript, ok := packet.(internal_type.SpeechToTextPacket); ok {
						timeline[i].Interim = transcript.Interim
					}
				}
				ginkgo.AddReportEntry("packet-timeline", timeline, ginkgo.ReportEntryVisibilityNever)
				gomega.Expect(failures).To(gomega.BeEmpty())
			})
			var err error
			stt, err = transformer.NewSpeechToText(
				transformer.WithProvider("smallest"), transformer.WithContext(sessionCtx),
				transformer.WithLogger(testutil.NewTestLogger()),
				transformer.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})),
				transformer.WithOptions(testutil.BuildOptions(map[string]interface{}{
					"listen.language": "en", "listen.word_timestamps": true,
				})), transformer.WithOnPacket(collector.OnPacket),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt).NotTo(gomega.BeNil())
			gomega.Expect(stt.Name()).To(gomega.Equal("smallest-stt"))
		})

		ginkgo.It("streams binary audio and emits interim then final transcripts across turn changes", ginkgo.Label("factory"), func(ctx ginkgo.SpecContext) {
			ginkgo.By("opening the configured Pulse stream and sending binary PCM")
			gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			for _, turn := range []string{"first", "next"} {
				gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: turn})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: turn})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 2, 3, 4}})).To(gomega.Succeed())
				kind, audio, err := conn.ReadMessage()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
				gomega.Expect(audio).To(gomega.Equal([]byte{1, 2, 3, 4}))
				gomega.Expect(conn.WriteJSON(map[string]interface{}{"transcript": turn, "is_final": false, "language": "en"})).To(gomega.Succeed())
				gomega.Eventually(collector.InterimTranscripts).WithContext(ctx).Should(gomega.ContainElement(internal_type.SpeechToTextPacket{
					ContextID: turn, Script: turn, Language: "en", Interim: true,
				}))
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: turn})).To(gomega.Succeed())
				gomega.Expect(conn.WriteJSON(map[string]interface{}{"transcript": turn, "is_final": true, "is_last": true, "language": "en"})).To(gomega.Succeed())
				gomega.Eventually(collector.FinalTranscripts).WithContext(ctx).Should(gomega.ContainElement(internal_type.SpeechToTextPacket{
					ContextID: turn, Script: turn, Language: "en",
				}))
			}
			ginkgo.By("observing exactly one interim and final transcript per turn")
			gomega.Expect(collector.TranscriptPackets()).To(gomega.HaveLen(4))
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{
				{ContextID: "first", Source: "word"}, {ContextID: "first", Source: "word"},
				{ContextID: "next", Source: "word"}, {ContextID: "next", Source: "word"},
			}))

			ginkgo.By("canceling and sending close_stream without waiting for a provider acknowledgement")
			cancelSession()
			closeCtx, cancel := context.WithCancel(context.Background())
			cancel()
			closed := make(chan error, 1)
			closeDone = make(chan struct{})
			go func() {
				defer close(closeDone)
				closed <- stt.Close(closeCtx)
			}()
			gomega.Eventually(closed).WithContext(ctx).WithTimeout(3 * time.Second).Should(gomega.Receive(gomega.BeNil()))
			var end map[string]interface{}
			gomega.Expect(conn.ReadJSON(&end)).To(gomega.Succeed())
			gomega.Expect(end).To(gomega.Equal(map[string]interface{}{"type": "close_stream"}))
			_, _, err := conn.ReadMessage()
			var networkErr net.Error
			if errors.As(err, &networkErr) {
				gomega.Expect(networkErr.Timeout()).To(gomega.BeFalse())
			}
			gomega.Expect(websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure)).To(gomega.BeTrue(), "expected peer closure: %v", err)
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.It("ignores malformed frames and provider VAD markers without losing the next transcript", func(ctx ginkgo.SpecContext) {
			ginkgo.By("sending invalid payloads and speech markers before a final transcript barrier")
			gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "active"})).To(gomega.Succeed())
			for _, frame := range []string{`{`, `{"transcript":42}`, `{"transcript":""}`, `{"type":"speech_started","timestamp":0}`, `{"type":"speech_ended","timestamp":1}`} {
				gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(frame))).To(gomega.Succeed())
			}
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"transcript":"Recovered","is_final":true,"language":"en","words":[{"word":"Recovered","confidence":0.8}]}`))).To(gomega.Succeed())
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).Should(gomega.Equal([]internal_type.SpeechToTextPacket{
				{ContextID: "active", Script: "Recovered", Language: "en"},
			}))
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{{ContextID: "active", Source: "word"}}))
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.DescribeTable("connection failures", func(ctx ginkgo.SpecContext, reject bool) {
			rejectUpgrade.Store(reject)
			ginkgo.By("initializing against the local provider")
			if reject {
				gomega.Expect(stt.Initialize()).To(gomega.MatchError(gomega.ContainSubstring("bad handshake")))
				gomega.Expect(connections).To(gomega.BeEmpty())
			} else {
				gomega.Expect(stt.Initialize()).To(gomega.Succeed())
				var conn *websocket.Conn
				gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
				gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "lost"})).To(gomega.Succeed())
				ginkgo.By("dropping the peer and observing the active turn's connection error")
				gomega.Expect(conn.Close()).To(gomega.Succeed())
				gomega.Eventually(func() []string {
					var contexts []string
					for _, packet := range collector.GetPackets() {
						if failure, ok := packet.(internal_type.SpeechToTextErrorPacket); ok && failure.Error != nil {
							contexts = append(contexts, failure.ContextID)
						}
					}
					return contexts
				}).WithContext(ctx).Should(gomega.Equal([]string{"lost"}))
			}
			gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
		}, ginkgo.Entry("upgrade rejected", true, ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("established socket lost", false, ginkgo.SpecTimeout(10*time.Second)))
	})
