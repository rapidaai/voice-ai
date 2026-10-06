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
	"time"

	"github.com/gorilla/websocket"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	deepgram "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("Deepgram STT WebSocket contract", ginkgo.Serial,
	ginkgo.Label("offline", "stt", "deepgram", "websocket"), func() {
		var (
			stt         internal_type.SpeechToTextTransformer
			collector   *testutil.PacketCollector
			connections chan *websocket.Conn
			endpoint    string
			closeDone   chan struct{}
		)

		ginkgo.BeforeEach(func() {
			stt = nil
			closeDone = nil
			collector = testutil.NewPacketCollector()
			connections = make(chan *websocket.Conn, 1)
			stop := make(chan struct{})
			failures := make(chan error, 16)
			var handlers sync.WaitGroup
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlers.Add(1)
				defer handlers.Done()
				if r.URL.Path != "/v1/listen" || r.URL.Query().Get("encoding") != "linear16" || r.URL.Query().Get("sample_rate") != "16000" {
					failures <- fmt.Errorf("unexpected Deepgram listen path or audio format")
					http.Error(w, "unexpected listen request", http.StatusBadRequest)
					return
				}
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					failures <- fmt.Errorf("upgrade Deepgram socket: %w", err)
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
			endpoint = "ws://" + server.Listener.Addr().String()
			// SDK environment options override credential endpoints and authentication.
			for key, value := range map[string]string{
				"DEEPGRAM_HOST": endpoint, "DEEPGRAM_ACCESS_TOKEN": "", "DEEPGRAM_API_VERSION": "v1",
				"DEEPGRAM_API_PATH": "listen", "DEEPGRAM_WEBSOCKET_REDIRECT": "false",
				"DEEPGRAM_WEBSOCKET_REPLY_AUTO_FLUSH": "0",
			} {
				ginkgo.GinkgoT().Setenv(key, value)
			}
			ginkgo.DeferCleanup(func() {
				close(stop)
				server.Close()
				handlers.Wait()
				if closeDone != nil {
					gomega.Eventually(closeDone, 3*time.Second).Should(gomega.BeClosed())
				}
				if stt != nil {
					gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed())
				}
				gomega.Expect(failures).To(gomega.BeEmpty())
			})
		})

		ginkgo.It("streams audio, finalizes speech, and assigns delayed wire results to the current turn", func(ctx ginkgo.SpecContext) {
			ginkgo.By("selecting Deepgram through the root factory with the SDK's local endpoint")
			var err error
			stt, err = transformer.NewSpeechToText(
				transformer.WithProvider("deepgram"),
				transformer.WithContext(ctx), transformer.WithLogger(testutil.NewTestLogger()),
				transformer.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key", "endpoint": endpoint})),
				transformer.WithOnPacket(collector.OnPacket),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt.Name()).To(gomega.Equal("deepgram-stt"))
			gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())

			ginkgo.By("sending audio and receiving an interim transcript for the active speech")
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 2, 3, 4}})).To(gomega.Succeed())
			kind, audio, err := conn.ReadMessage()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
			gomega.Expect(audio).To(gomega.Equal([]byte{1, 2, 3, 4}))
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Results","is_final":false,"channel":{"alternatives":[{"transcript":"Hello","confidence":0.9}]}}`))).To(gomega.Succeed())
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).Should(gomega.Equal([]internal_type.SpeechToTextPacket{
				{ContextID: "first", Script: "Hello", Confidence: 0.9, Language: "en", Interim: true},
			}))

			ginkgo.By("observing Finalize on the wire before delivering the final transcript")
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: "first"})).To(gomega.Succeed())
			var finalize map[string]interface{}
			gomega.Expect(conn.ReadJSON(&finalize)).To(gomega.Succeed())
			gomega.Expect(finalize).To(gomega.Equal(map[string]interface{}{"type": "Finalize"}))
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Results","is_final":true,"from_finalize":true,"channel":{"alternatives":[{"transcript":"Hello there.","confidence":0.9}]}}`))).To(gomega.Succeed())
			gomega.Eventually(collector.FinalTranscripts).WithContext(ctx).Should(gomega.HaveExactElements(
				gstruct.MatchAllFields(gstruct.Fields{
					"ContextID": gomega.Equal("first"), "Script": gomega.Equal("Hello there."), "Concat": gomega.BeNil(),
					"Confidence": gomega.Equal(0.9), "Language": gomega.Equal("en"), "Interim": gomega.BeFalse(),
					"Latency": gomega.BeNumerically(">=", 0),
				}),
			))
			firstFinal := collector.FinalTranscripts()[0]

			ginkgo.By("changing turns before a delayed result arrives on the session socket")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Results","is_final":true,"channel":{"alternatives":[{"transcript":"Delayed result.","confidence":0.9}]}}`))).To(gomega.Succeed())
			gomega.Eventually(collector.FinalTranscripts).WithContext(ctx).Should(gomega.HaveExactElements(
				gomega.Equal(firstFinal),
				gstruct.MatchAllFields(gstruct.Fields{
					"ContextID": gomega.Equal("next"), "Script": gomega.Equal("Delayed result."), "Concat": gomega.BeNil(),
					"Confidence": gomega.Equal(0.9), "Language": gomega.Equal("en"), "Interim": gomega.BeFalse(),
					"Latency": gomega.BeNumerically(">=", 0),
				}),
			))

			ginkgo.By("closing within a bound despite an already canceled caller context")
			closeCtx, cancel := context.WithCancel(context.Background())
			cancel()
			closed := make(chan error, 1)
			closeDone = make(chan struct{})
			go func() {
				defer close(closeDone)
				closed <- stt.Close(closeCtx)
			}()
			gomega.Eventually(closed).WithContext(ctx).WithTimeout(3 * time.Second).Should(gomega.Receive(gomega.BeNil()))
			var closeStream map[string]interface{}
			gomega.Expect(conn.ReadJSON(&closeStream)).To(gomega.Succeed())
			gomega.Expect(closeStream).To(gomega.Equal(map[string]interface{}{"type": "CloseStream"}))
			_, _, err = conn.ReadMessage()
			var networkErr net.Error
			if errors.As(err, &networkErr) {
				gomega.Expect(networkErr.Timeout()).To(gomega.BeFalse(), "shutdown must close the peer socket, not time out")
			}
			gomega.Expect(websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure)).To(gomega.BeTrue(), "expected peer closure: %v", err)
			gomega.Expect(stt.Close(closeCtx)).To(gomega.Succeed())
		}, ginkgo.Label("factory"), ginkgo.SpecTimeout(10*time.Second))

		ginkgo.It("retains the captured turn while an admitted callback is delayed across a turn change", func(ctx ginkgo.SpecContext) {
			ginkgo.By("holding an admitted transcript callback before recording its packets")
			entered := make(chan struct{})
			release := make(chan struct{}, 1)
			callbackCollector := collector
			ginkgo.DeferCleanup(func() { close(release) })
			var err error
			stt, err = deepgram.NewSpeechToText(
				deepgram.WithContext(ctx), deepgram.WithLogger(testutil.NewTestLogger()),
				deepgram.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key", "endpoint": endpoint})),
				deepgram.WithOnPacket(func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if transcript, ok := packet.(internal_type.SpeechToTextPacket); ok && transcript.Interim {
							close(entered)
							select {
							case <-release:
							case <-ctx.Done():
							}
						}
					}
					return callbackCollector.OnPacket(packets...)
				}),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Results","is_final":false,"channel":{"alternatives":[{"transcript":"Held","confidence":0.9}]}}`))).To(gomega.Succeed())
			gomega.Eventually(entered).WithContext(ctx).Should(gomega.BeClosed())

			ginkgo.By("changing the active turn before releasing the already captured callback")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
			release <- struct{}{}
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).Should(gomega.Equal([]internal_type.SpeechToTextPacket{
				{ContextID: "first", Script: "Held", Confidence: 0.9, Language: "en", Interim: true},
			}))
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{
				{ContextID: "first", Source: "word"},
			}))
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.It("rejects canceled setup without opening a socket or emitting a transcript", func(ctx ginkgo.SpecContext) {
			ginkgo.By("canceling the session before constructing the SDK-backed transformer")
			setupCtx, cancel := context.WithCancel(ctx)
			cancel()
			candidate, err := deepgram.NewSpeechToText(
				deepgram.WithContext(setupCtx), deepgram.WithLogger(testutil.NewTestLogger()),
				deepgram.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key", "endpoint": endpoint})),
				deepgram.WithOnPacket(collector.OnPacket),
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(candidate).To(gomega.BeNil())
			gomega.Expect(connections).To(gomega.BeEmpty())
			gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
			gomega.Expect(collector.MetricPackets()).NotTo(gomega.BeEmpty())
		}, ginkgo.SpecTimeout(5*time.Second))
	})
