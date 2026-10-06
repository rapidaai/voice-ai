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
	options "github.com/rapidaai/api/assistant-api/internal/options"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

var _ = ginkgo.Describe("Custom STT WebSocket lifecycle", ginkgo.Label("offline", "stt", "custom-stt", "websocket"), func() {
	ginkgo.DescribeTable("public factory session", func(ctx ginkgo.SpecContext, failure bool) {
		session, cancel := context.WithCancel(ctx)
		collector := testutil.NewPacketCollector()
		peers := make(chan *websocket.Conn, 1)
		stop, entered, release, callbackDone := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
		var held atomic.Bool
		var handlers sync.WaitGroup
		serverErrors := make(chan error, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.Add(1)
			defer handlers.Done()
			conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
			if err != nil {
				serverErrors <- err
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
		var stt internal_type.SpeechToTextTransformer
		var expected []internal_type.SpeechToTextPacket
		expectedErrors := 0
		ginkgo.DeferCleanup(func(cleanup ginkgo.SpecContext) {
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
			if held.Load() {
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
			gomega.Expect(serverErrors).To(gomega.BeEmpty())
			gomega.Expect(collector.TranscriptPackets()).To(gomega.Equal(expected))
			gomega.Expect(errors).To(gomega.HaveLen(expectedErrors))
			for _, packet := range errors {
				gomega.Expect(packet.ContextID).To(gomega.Equal("first"))
				gomega.Expect(packet.Type).To(gomega.Equal(internal_type.STTErrorType(internal_type.STTSystemPanic)))
				gomega.Expect(packet.Error).To(gomega.HaveOccurred())
			}
		}, ginkgo.NodeTimeout(5*time.Second))

		ginkgo.By("opening custom websocket_v1 through the public factory with PCM and EOS rules")
		var err error
		stt, err = transformer.NewSpeechToText(
			transformer.WithProvider("custom-stt"), transformer.WithContext(session),
			transformer.WithLogger(testutil.NewTestLogger()),
			transformer.WithCredential(testutil.BuildCredential(map[string]string{"base_url": "ws" + strings.TrimPrefix(server.URL, "http"), "api_compatibility": "websocket_v1"})),
			transformer.WithOptions(utils.Option{
				options.ListenOptionRequestRules: `[
					{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}},
					{"when":{"packet":"interrupt"},"send":{"frame":"json","body":{"type":"finish","owner":{"$path":"packet.context_id"}}}}
				]`,
				options.ListenOptionResponseRules: `[
					{"when":{"frame":"json","path":"type","equals":"error"},"emit":{"error":{"$path":"message"}}},
					{"when":{"frame":"json","path":"type","equals":"result"},"emit":{"script":{"$path":"text"},"interim":{"$path":"interim"},"language":"en"}}
				]`,
			}),
			transformer.WithOnPacket(func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if transcript, ok := packet.(internal_type.SpeechToTextPacket); ok && transcript.Interim && held.CompareAndSwap(false, true) {
						close(entered)
						defer close(callbackDone)
						select {
						case <-release:
						case <-session.Done():
						}
					}
				}
				return collector.OnPacket(packets...)
			}))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(stt.Initialize()).To(gomega.Succeed())
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 0, 2, 0}})).To(gomega.Succeed())
		var peer *websocket.Conn
		gomega.Eventually(peers).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&peer))
		gomega.Expect(peer.SetReadDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
		gomega.Expect(peer.SetWriteDeadline(time.Now().Add(3 * time.Second))).To(gomega.Succeed())
		kind, pcm, err := peer.ReadMessage()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(kind).To(gomega.Equal(websocket.BinaryMessage))
		gomega.Expect(pcm).To(gomega.Equal([]byte{1, 0, 2, 0}))
		owner := "first"
		if failure {
			ginkgo.By("reporting missing response fields and provider errors before a valid recovery result")
			expectedErrors = 2
			gomega.Expect(peer.WriteJSON(map[string]any{"type": "result", "interim": false})).To(gomega.Succeed())
			gomega.Expect(peer.WriteJSON(map[string]string{"type": "error", "message": "offline failure"})).To(gomega.Succeed())
		} else {
			ginkgo.By("holding an admitted interim callback across a turn change")
			gomega.Expect(peer.WriteJSON(map[string]any{"type": "result", "text": "Partial", "interim": true})).To(gomega.Succeed())
			gomega.Eventually(entered).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
			close(release)
			gomega.Eventually(callbackDone).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
			expected = []internal_type.SpeechToTextPacket{{ContextID: "first", Script: "Partial", Language: "en", Interim: true, Concat: utils.Ptr("")}}
			gomega.Expect(collector.TranscriptPackets()).To(gomega.Equal(expected))
			owner = "next"
		}
		ginkgo.By("sending EOS through the configured interrupt rule and receiving a final for its owner")
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: owner})).To(gomega.Succeed())
		var finish map[string]string
		gomega.Expect(peer.ReadJSON(&finish)).To(gomega.Succeed())
		gomega.Expect(finish).To(gomega.Equal(map[string]string{"type": "finish", "owner": owner}))
		gomega.Expect(peer.WriteJSON(map[string]any{"type": "result", "text": "Final", "interim": false})).To(gomega.Succeed())
		expected = append(expected, internal_type.SpeechToTextPacket{ContextID: owner, Script: "Final", Language: "en", Concat: utils.Ptr("")})
		gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expected))

		ginkgo.By("closing with a canceled caller and observing peer shutdown")
		cancel()
		closed := make(chan error, 1)
		go func() { closed <- stt.Close(session) }()
		gomega.Eventually(closed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(gomega.BeNil()))
		gomega.Expect(peer.SetReadDeadline(time.Now().Add(time.Second))).To(gomega.Succeed())
		_, _, err = peer.ReadMessage()
		gomega.Expect(websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure)).To(gomega.BeTrue())
	},
		ginkgo.Entry("preserves an admitted callback owner and completes the new turn", false, ginkgo.SpecTimeout(10*time.Second)),
		ginkgo.Entry("reports response failures and accepts the next valid final", true, ginkgo.SpecTimeout(10*time.Second)),
	)
})

var _ = ginkgo.Describe("Custom STT HTTP lifecycle", ginkgo.Label("offline", "stt", "custom-stt", "http"), func() {
	ginkgo.DescribeTable("public factory buffered request", func(ctx ginkgo.SpecContext, scenario string) {
		session, cancel := context.WithCancel(ctx)
		collector := testutil.NewPacketCollector()
		requests := make(chan struct {
			Owner string `json:"owner"`
			Audio string `json:"audio"`
		}, 1)
		release, handlerDone := make(chan struct{}), make(chan struct{})
		failures := make(chan error, 1)
		var handlers sync.WaitGroup
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.Add(1)
			defer handlers.Done()
			if requestCount.Add(1) != 1 {
				http.Error(w, "unexpected duplicate request", http.StatusConflict)
				return
			}
			defer close(handlerDone)
			var request struct {
				Owner string `json:"owner"`
				Audio string `json:"audio"`
			}
			if r.Method != http.MethodPost {
				failures <- fmt.Errorf("expected POST")
				return
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				failures <- err
				return
			}
			select {
			case requests <- request:
			case <-session.Done():
				return
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			case <-session.Done():
				return
			}
			if scenario == "failure" {
				http.Error(w, "offline failure", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"text": "Final", "confidence": 0.8}); err != nil {
				failures <- err
			}
		}))
		var stt internal_type.SpeechToTextTransformer
		var expected []internal_type.SpeechToTextPacket
		expectedErrors := 0
		ginkgo.DeferCleanup(func(cleanup ginkgo.SpecContext) {
			cancel()
			closed := make(chan error, 1)
			go func() {
				var err error
				if stt != nil {
					err = stt.Close(cleanup)
				}
				server.Close()
				handlers.Wait()
				closed <- err
			}()
			gomega.Eventually(closed).WithContext(cleanup).WithTimeout(3 * time.Second).Should(gomega.Receive(gomega.BeNil()))
			packets := collector.GetPackets()
			timeline := make([]struct {
				Kind      string `json:"kind"`
				ContextID string `json:"context_id"`
			}, len(packets))
			var errors []internal_type.SpeechToTextErrorPacket
			for i, packet := range packets {
				timeline[i].Kind, timeline[i].ContextID = fmt.Sprintf("%T", packet), packet.ContextId()
				if packet, ok := packet.(internal_type.SpeechToTextErrorPacket); ok {
					errors = append(errors, packet)
				}
			}
			ginkgo.AddReportEntry("packet-timeline", timeline)
			gomega.Expect(requestCount.Load()).To(gomega.Equal(int32(1)), "EOS must submit exactly one HTTP request")
			gomega.Expect(failures).To(gomega.BeEmpty())
			gomega.Expect(collector.TranscriptPackets()).To(gomega.Equal(expected))
			gomega.Expect(errors).To(gomega.HaveLen(expectedErrors))
			for _, packet := range errors {
				gomega.Expect(packet.ContextID).To(gomega.Equal("first"))
				gomega.Expect(packet.Type).To(gomega.Equal(internal_type.STTErrorType(internal_type.STTNetworkTimeout)))
				gomega.Expect(packet.Error).To(gomega.HaveOccurred())
			}
		}, ginkgo.NodeTimeout(5*time.Second))

		ginkgo.By("constructing http_v1 with a PCM JSON request and a final-response rule")
		var err error
		stt, err = transformer.NewSpeechToText(
			transformer.WithProvider("custom-stt"), transformer.WithContext(session), transformer.WithLogger(testutil.NewTestLogger()),
			transformer.WithCredential(testutil.BuildCredential(map[string]string{"base_url": server.URL, "api_compatibility": "http_v1"})),
			transformer.WithOnPacket(collector.OnPacket), transformer.WithOptions(utils.Option{
				options.ListenOptionRequestRules:  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"owner":{"$path":"packet.context_id"},"audio":{"$path":"packet.audio.pcm_base64"}}}}]`,
				options.ListenOptionResponseRules: `[{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"confidence":{"$path":"confidence"},"language":"en","interim":false}}]`,
			}))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(stt.Initialize()).To(gomega.Succeed())
		ginkgo.By("buffering PCM chunks until EOS submits one request")
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{ContextID: "first"})).To(gomega.Succeed())
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 0}})).To(gomega.Succeed())
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{2, 0}})).To(gomega.Succeed())
		gomega.Consistently(requests).WithContext(ctx).WithTimeout(50 * time.Millisecond).ShouldNot(gomega.Receive())
		gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{ContextID: "first"})).To(gomega.Succeed())
		var request struct {
			Owner string `json:"owner"`
			Audio string `json:"audio"`
		}
		gomega.Eventually(requests).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(&request))
		gomega.Expect(request.Owner).To(gomega.Equal("first"))
		pcm, err := base64.StdEncoding.DecodeString(request.Audio)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(pcm).To(gomega.Equal([]byte{1, 0, 2, 0}))

		ginkgo.By("changing the current turn while the old HTTP response is delayed")
		gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
		if scenario == "close" {
			ginkgo.By("closing the in-flight request without admitting a final response")
			closed := make(chan error, 1)
			go func() { closed <- stt.Close(ctx) }()
			gomega.Eventually(closed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(gomega.BeNil()))
			gomega.Eventually(handlerDone).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
			return
		}
		close(release)
		if scenario == "failure" {
			expectedErrors = 1
			gomega.Eventually(collector.GetPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{})))
		} else {
			expected = []internal_type.SpeechToTextPacket{{ContextID: "first", Script: "Final", Confidence: 0.8, Language: "en"}}
			gomega.Eventually(collector.TranscriptPackets).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Equal(expected))
		}
		gomega.Eventually(handlerDone).WithContext(ctx).WithTimeout(time.Second).Should(gomega.BeClosed())
	},
		ginkgo.Entry("retains the flushed request owner across a delayed response and turn change", "flow", ginkgo.SpecTimeout(8*time.Second)),
		ginkgo.Entry("attributes HTTP failure to the flushed owner rather than the new turn", "failure", ginkgo.SpecTimeout(8*time.Second)),
		ginkgo.Entry("cancels an in-flight request on Close", "close", ginkgo.SpecTimeout(8*time.Second)),
	)
})
