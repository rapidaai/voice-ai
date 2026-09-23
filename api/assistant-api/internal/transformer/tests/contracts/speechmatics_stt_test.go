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
	speechmatics "github.com/rapidaai/api/assistant-api/internal/transformer/speechmatics"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("Speechmatics STT WebSocket contract", ginkgo.Serial,
	ginkgo.Label("offline", "stt", "speechmatics", "websocket"), func() {
		var (
			stt         internal_type.SpeechToTextTransformer
			collector   *testutil.PacketCollector
			connections chan *websocket.Conn
			initialized chan error
			initDone    chan struct{}
			closeDone   chan struct{}
		)

		ginkgo.BeforeEach(func() {
			collector = testutil.NewPacketCollector()
			connections = make(chan *websocket.Conn, 1)
			initialized = make(chan error, 1)
			initDone = nil
			closeDone = nil
			stop := make(chan struct{})
			failures := make(chan error, 16)
			var handlers sync.WaitGroup
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlers.Add(1)
				defer handlers.Done()
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					failures <- fmt.Errorf("upgrade Speechmatics socket: %w", err)
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
			ctx, cancel := context.WithCancel(context.Background())
			stt = nil
			ginkgo.DeferCleanup(func() {
				defer func() { websocket.DefaultDialer = previousDialer }()
				cancel()
				close(stop)
				server.Close()
				handlers.Wait()
				if initDone != nil {
					gomega.Eventually(initDone, 3*time.Second).Should(gomega.BeClosed())
				}
				if closeDone != nil {
					gomega.Eventually(closeDone, 3*time.Second).Should(gomega.BeClosed())
				}
				if stt != nil {
					gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed())
				}
				gomega.Expect(failures).To(gomega.BeEmpty())
			})
			var err error
			stt, err = speechmatics.NewSpeechToText(
				speechmatics.WithContext(ctx), speechmatics.WithLogger(testutil.NewTestLogger()),
				speechmatics.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})),
				speechmatics.WithOnPacket(collector.OnPacket),
				speechmatics.WithOptions(testutil.BuildOptions(map[string]interface{}{
					"listen.language": "en", "listen.operating_point": "standard",
				})),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("waits for startup acknowledgement, emits transcripts, and closes without a final server acknowledgement", func(ctx ginkgo.SpecContext) {
			ginkgo.By("starting recognition and inspecting the requested audio and transcription formats")
			initDone = make(chan struct{})
			go func() {
				defer close(initDone)
				initialized <- stt.Initialize()
			}()
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			var start map[string]interface{}
			gomega.Expect(conn.ReadJSON(&start)).To(gomega.Succeed())
			gomega.Expect(start).To(gomega.Equal(map[string]interface{}{
				"message":      "StartRecognition",
				"audio_format": map[string]interface{}{"type": "raw", "encoding": "pcm_s16le", "sample_rate": float64(16000)},
				"transcription_config": map[string]interface{}{
					"language": "en", "operating_point": "standard", "enable_partials": true, "max_delay": float64(2),
				},
			}))
			gomega.Expect(initialized).NotTo(gomega.Receive())
			gomega.Expect(conn.WriteJSON(map[string]string{"message": "Info"})).To(gomega.Succeed())
			gomega.Expect(conn.WriteJSON(map[string]string{"message": "RecognitionStarted"})).To(gomega.Succeed())
			gomega.Eventually(initialized).WithContext(ctx).Should(gomega.Receive(gomega.BeNil()))

			ginkgo.By("streaming binary audio and recovering from a malformed transcript frame")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "first"})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 2, 3, 4}})).To(gomega.Succeed())
			kind, audio, err := conn.ReadMessage()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
			gomega.Expect(audio).To(gomega.Equal([]byte{1, 2, 3, 4}))
			gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte("{"))).To(gomega.Succeed())
			gomega.Expect(conn.WriteJSON(map[string]interface{}{
				"message": "AddPartialTranscript", "metadata": map[string]string{"transcript": "Hello"},
			})).To(gomega.Succeed())
			gomega.Expect(conn.WriteJSON(map[string]interface{}{
				"message": "AddTranscript", "metadata": map[string]string{"transcript": "Hello there."},
			})).To(gomega.Succeed())
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).Should(gomega.Equal([]internal_type.SpeechToTextPacket{
				{ContextID: "first", Script: "Hello", Interim: true},
				{ContextID: "first", Script: "Hello there."},
			}))
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{
				{ContextID: "first", Source: "word"}, {ContextID: "first", Source: "word"},
			}))

			ginkgo.By("attributing a provider error to the next turn without emitting a transcript")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
			gomega.Expect(conn.WriteJSON(map[string]string{"message": "Error"})).To(gomega.Succeed())
			gomega.Eventually(func() []internal_type.SpeechToTextErrorPacket {
				var failures []internal_type.SpeechToTextErrorPacket
				for _, packet := range collector.GetPackets() {
					if failure, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
						failures = append(failures, failure)
					}
				}
				return failures
			}).WithContext(ctx).Should(gomega.HaveExactElements(gstruct.MatchAllFields(gstruct.Fields{
				"ContextID": gomega.Equal("next"), "Type": gomega.Equal(internal_type.STTErrorType(internal_type.STTNetworkTimeout)), "Error": gomega.HaveOccurred(),
			})))
			gomega.Expect(collector.TranscriptPackets()).To(gomega.HaveLen(2))

			ginkgo.By("closing with a canceled caller context while the peer sends no EndOfTranscript")
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
			gomega.Expect(end).To(gomega.Equal(map[string]interface{}{"message": "EndOfStream", "last_seq_no": float64(0)}))
			_, _, err = conn.ReadMessage()
			var networkErr net.Error
			if errors.As(err, &networkErr) {
				gomega.Expect(networkErr.Timeout()).To(gomega.BeFalse(), "shutdown must close the peer socket, not time out")
			}
			gomega.Expect(websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure)).To(gomega.BeTrue(), "expected peer closure: %v", err)
			gomega.Expect(stt.Close(closeCtx)).To(gomega.Succeed())
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.It("reports an established connection loss for the active turn and closes without a transcript", func(ctx ginkgo.SpecContext) {
			ginkgo.By("acknowledging recognition and accepting audio for the active turn")
			initDone = make(chan struct{})
			go func() {
				defer close(initDone)
				initialized <- stt.Initialize()
			}()
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			var start map[string]interface{}
			gomega.Expect(conn.ReadJSON(&start)).To(gomega.Succeed())
			gomega.Expect(start["message"]).To(gomega.Equal("StartRecognition"))
			gomega.Expect(conn.WriteJSON(map[string]string{"message": "RecognitionStarted"})).To(gomega.Succeed())
			gomega.Eventually(initialized).WithContext(ctx).Should(gomega.Receive(gomega.BeNil()))
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "interrupted"})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 2}})).To(gomega.Succeed())
			kind, audio, err := conn.ReadMessage()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
			gomega.Expect(audio).To(gomega.Equal([]byte{1, 2}))

			ginkgo.By("dropping the socket before a transcript and observing the active turn's error")
			gomega.Expect(conn.Close()).To(gomega.Succeed())
			gomega.Eventually(func() []internal_type.SpeechToTextErrorPacket {
				var failures []internal_type.SpeechToTextErrorPacket
				for _, packet := range collector.GetPackets() {
					if failure, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
						failures = append(failures, failure)
					}
				}
				return failures
			}).WithContext(ctx).Should(gomega.HaveExactElements(gstruct.MatchAllFields(gstruct.Fields{
				"ContextID": gomega.Equal("interrupted"),
				"Type":      gomega.Equal(internal_type.STTErrorType(internal_type.STTNetworkTimeout)),
				"Error":     gomega.MatchError(gomega.ContainSubstring("connection lost")),
			})))

			ginkgo.By("closing the disconnected session within a bound without successful output")
			closed := make(chan error, 1)
			closeDone = make(chan struct{})
			go func() {
				defer close(closeDone)
				closed <- stt.Close(ctx)
			}()
			gomega.Eventually(closed).WithContext(ctx).WithTimeout(3 * time.Second).Should(gomega.Receive(gomega.BeNil()))
			gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
			gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.BeEmpty())
		}, ginkgo.SpecTimeout(10*time.Second))

		ginkgo.DescribeTable("failed recognition startup", func(ctx ginkgo.SpecContext, response, expectedError string) {
			ginkgo.By("starting a real local socket and rejecting the startup exchange")
			initDone = make(chan struct{})
			go func() {
				defer close(initDone)
				initialized <- stt.Initialize()
			}()
			var conn *websocket.Conn
			gomega.Eventually(connections).WithContext(ctx).Should(gomega.Receive(&conn))
			gomega.Expect(conn.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			gomega.Expect(conn.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
			var start map[string]interface{}
			gomega.Expect(conn.ReadJSON(&start)).To(gomega.Succeed())
			gomega.Expect(start["message"]).To(gomega.Equal("StartRecognition"))
			if response == "" {
				gomega.Expect(conn.Close()).To(gomega.Succeed())
			} else {
				gomega.Expect(conn.WriteMessage(websocket.TextMessage, []byte(response))).To(gomega.Succeed())
			}
			gomega.Eventually(initialized).WithContext(ctx).Should(gomega.Receive(gomega.MatchError(gomega.ContainSubstring(expectedError))))

			ginkgo.By("observing peer cleanup and no successful transcription after failed initialization")
			if response != "" {
				_, _, err := conn.ReadMessage()
				gomega.Expect(websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure)).To(gomega.BeTrue())
			}
			gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
			gomega.Expect(stt.Close(ctx)).To(gomega.Succeed())
		}, ginkgo.Entry("provider rejects recognition", `{"message":"Error","reason":"offline rejection"}`, "server error during init", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("provider sends malformed JSON", `{`, "failed parsing RecognitionStarted", ginkgo.SpecTimeout(10*time.Second)),
			ginkgo.Entry("provider disconnects before acknowledgement", "", "failed reading RecognitionStarted", ginkgo.SpecTimeout(10*time.Second)))
	})
