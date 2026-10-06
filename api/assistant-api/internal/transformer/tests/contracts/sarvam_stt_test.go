// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/onsi/gomega/gstruct"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("Sarvam STT WebSocket contract", ginkgo.Serial,
	ginkgo.Label("offline", "stt", "sarvamai", "websocket"), func() {
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
				if r.URL.Path != "/speech-to-text/ws" || query.Get("sample_rate") != "16000" ||
					query.Get("input_audio_codec") != "pcm_s16le" || query.Get("language-code") != "en-IN" ||
					query.Get("model") != "saarika:v2.5" || r.Header.Get("Api-Subscription-Key") != "offline-key" {
					failures <- fmt.Errorf("unexpected Sarvam startup path, configuration, or authentication")
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
					failures <- fmt.Errorf("upgrade Sarvam socket: %w", err)
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
				transformer.WithProvider("sarvamai"), transformer.WithContext(sessionCtx),
				transformer.WithLogger(testutil.NewTestLogger()),
				transformer.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})),
				transformer.WithOptions(testutil.BuildOptions(map[string]interface{}{
					"listen.language": "en-IN", "listen.model": "saarika:v2.5",
				})), transformer.WithOnPacket(collector.OnPacket),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt).NotTo(gomega.BeNil())
			gomega.Expect(stt.Name()).To(gomega.Equal("sarvam-stt"))
		})

		ginkgo.It("frames PCM audio and emits final transcripts across speech ends and turn changes", ginkgo.Label("factory"), func(ctx ginkgo.SpecContext) {
			ginkgo.By("opening the configured stream and checking the JSON audio envelope")
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
				gomega.Expect(kind).To(gomega.Equal(websocket.TextMessage))
				var envelope map[string]interface{}
				gomega.Expect(json.Unmarshal(audio, &envelope)).To(gomega.Succeed())
				gomega.Expect(envelope).To(gomega.Equal(map[string]interface{}{"audio": map[string]interface{}{
					"data": base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}), "sample_rate": float64(16000),
					"encoding": "audio/wav", "input_audio_codec": "pcm_s16le",
				}}))
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: turn})).To(gomega.Succeed())
				gomega.Expect(conn.WriteJSON(map[string]interface{}{
					"type": "events", "data": map[string]string{"signal_type": "END_SPEECH"},
				})).To(gomega.Succeed())
				gomega.Expect(conn.WriteJSON(map[string]interface{}{
					"type": "data", "data": map[string]string{"transcript": turn, "language_code": "en-IN"},
				})).To(gomega.Succeed())
				gomega.Eventually(collector.FinalTranscripts).WithContext(ctx).Should(gomega.ContainElement(internal_type.SpeechToTextPacket{
					ContextID: turn, Script: turn, Language: "en-IN", Confidence: 0.9,
				}))
			}
			ginkgo.By("observing exactly one final transcript and word interruption per turn")
			gomega.Expect(collector.TranscriptPackets()).To(gomega.HaveLen(2))
			gomega.Expect(collector.InterimTranscripts()).To(gomega.BeEmpty())
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{
				{ContextID: "first", Source: "word"}, {ContextID: "next", Source: "word"},
			}))

			ginkgo.By("canceling the session and closing without waiting for a provider reply")
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
			_, _, err := conn.ReadMessage()
			var networkErr net.Error
			if errors.As(err, &networkErr) {
				gomega.Expect(networkErr.Timeout()).To(gomega.BeFalse())
			}
			gomega.Expect(websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure)).To(gomega.BeTrue(), "expected peer closure: %v", err)
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.It("recovers after malformed frames and reports provider errors for the active turn", func(ctx ginkgo.SpecContext) {
			ginkgo.By("sending invalid JSON and invalid typed payloads before a valid provider error")
			gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "first"})).To(gomega.Succeed())
			for _, frame := range []string{`{`, `{"type":"data","data":42}`, `{"type":"error","data":42}`, `{"type":"unknown"}`} {
				gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(frame))).To(gomega.Succeed())
			}
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","data":{"error":"offline rejection","code":"invalid_audio"}}`))).To(gomega.Succeed())
			gomega.Eventually(func() []internal_type.SpeechToTextErrorPacket {
				var failures []internal_type.SpeechToTextErrorPacket
				for _, packet := range collector.GetPackets() {
					if failure, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
						failures = append(failures, failure)
					}
				}
				return failures
			}).WithContext(ctx).Should(gomega.HaveExactElements(gstruct.MatchAllFields(gstruct.Fields{
				"ContextID": gomega.Equal("first"), "Type": gomega.Equal(internal_type.STTErrorType(internal_type.STTNetworkTimeout)),
				"Error": gomega.MatchError(gomega.ContainSubstring("offline rejection")),
			})))
			ginkgo.By("using the error callback as a barrier before admitting the next turn")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"data","data":{"transcript":"Recovered","language_code":"en-IN"}}`))).To(gomega.Succeed())
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).Should(gomega.Equal([]internal_type.SpeechToTextPacket{
				{ContextID: "next", Script: "Recovered", Language: "en-IN", Confidence: 0.9},
			}))
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
