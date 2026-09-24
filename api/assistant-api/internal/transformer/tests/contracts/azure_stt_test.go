// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	azure "github.com/rapidaai/api/assistant-api/internal/transformer/azure"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("Azure STT contracts", ginkgo.Label("offline", "stt", "azure-speech-service", "sdk"), func() {
	ginkgo.It("preserves a loopback endpoint and applies recognition settings to native configuration", ginkgo.Label("configuration"), func() {
		ginkgo.By("constructing configuration without creating or starting a recognizer")
		endpoint := "ws://127.0.0.1:1/speech/recognition/conversation/cognitiveservices/v1"
		opts, err := azure.NewAzureOption(testutil.NewTestLogger(), testutil.BuildCredential(map[string]string{
			"subscription_key": "offline-key", "endpoint": endpoint,
		}), testutil.BuildOptions(map[string]interface{}{"listen.language": "de-DE"}))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		config, err := opts.SpeechToTextOption()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		ginkgo.DeferCleanup(config.Close)
		ginkgo.By("checking the actual SDK endpoint, language and detailed response format")
		gomega.Expect(config.GetProperty(common.SpeechServiceConnectionEndpoint)).To(gomega.Equal(endpoint))
		gomega.Expect(config.SpeechRecognitionLanguage()).To(gomega.Equal("de-DE"))
		gomega.Expect(config.OutputFormat()).To(gomega.Equal(common.Detailed))
	})

	for _, canceled := range []bool{false, true} {
		name := "before initialization"
		if canceled {
			name = "with an already-canceled parent"
		}
		ginkgo.It("closes repeatedly "+name+" without inventing usage or transcripts", ginkgo.Label("lifecycle"), func(ctx ginkgo.SpecContext) {
			ginkgo.By("constructing a provider without allocating native recognition resources")
			parent, cancel := context.WithCancel(ctx)
			defer cancel()
			if canceled {
				cancel()
			}
			collector := testutil.NewPacketCollector()
			stt, err := azure.NewSpeechToText(azure.WithContext(parent), azure.WithLogger(testutil.NewTestLogger()),
				azure.WithCredential(testutil.BuildCredential(map[string]string{
					"subscription_key": "offline-key", "endpoint": "ws://127.0.0.1:1",
				})), azure.WithOptions(testutil.BuildOptions(nil)), azure.WithOnPacket(collector.OnPacket))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			ginkgo.DeferCleanup(func() { gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed()) })
			ginkgo.By("retaining the current turn across non-audio control packets")
			gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "unstarted"})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{})).To(gomega.Succeed())
			gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{})).To(gomega.Succeed())
			gomega.Expect(collector.GetPackets()).To(gomega.BeEmpty())
			ginkgo.By("closing twice without duration billing or successful recognition")
			gomega.Expect(stt.Close(ctx)).To(gomega.Succeed())
			gomega.Expect(stt.Close(ctx)).To(gomega.Succeed())
			gomega.Expect(collector.EventPackets()).NotTo(gomega.BeEmpty())
			for _, packet := range collector.GetPackets() {
				gomega.Expect(packet).To(gomega.BeAssignableToTypeOf(internal_type.ObservabilityEventRecordPacket{}))
				gomega.Expect(packet).To(gomega.HaveField("ContextID", "unstarted"))
				gomega.Expect(packet).To(gomega.HaveField("Record.Event", observability.STTClosed))
			}
			ginkgo.AddReportEntry("packet-timeline", map[string]interface{}{
				"context_id": "unstarted", "closed_events": len(collector.EventPackets()), "initialized": false,
			}, ginkgo.ReportEntryVisibilityFailureOrVerbose)
		}, ginkgo.SpecTimeout(5*time.Second))
	}

	ginkgo.It("rejects missing endpoint configuration before allocating a recognizer", ginkgo.Label("configuration"), func(ctx ginkgo.SpecContext) {
		ginkgo.By("providing a synthetic subscription key without an endpoint")
		collector := testutil.NewPacketCollector()
		stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(testutil.NewTestLogger()),
			azure.WithCredential(testutil.BuildCredential(map[string]string{"subscription_key": "offline-key"})),
			azure.WithOptions(testutil.BuildOptions(nil)), azure.WithOnPacket(collector.OnPacket))
		ginkgo.By("returning an actionable error without lifecycle packets")
		gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("endpoint not found")))
		gomega.Expect(stt).To(gomega.BeNil())
		gomega.Expect(collector.GetPackets()).To(gomega.BeEmpty())
	})

	ginkgo.It("reports a local SDK WebSocket handshake rejection and closes", ginkgo.Label("websocket", "lifecycle"), func(ctx ginkgo.SpecContext) {
		if os.Getenv("RAPIDA_AZURE_STT_CONTRACT_CHILD") != "1" {
			ginkgo.By("isolating native SDK resources and ignored asynchronous result channels")
			childCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			command := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestTransformerContracts$",
				"-test.timeout=18s", "-ginkgo.focus=reports a local SDK WebSocket handshake rejection and closes", "-ginkgo.focus-file=azure_stt_test.go",
				"-ginkgo.no-color", "-ginkgo.v")
			for _, value := range os.Environ() {
				key, _, _ := strings.Cut(value, "=")
				switch strings.ToUpper(key) {
				case "RAPIDA_AZURE_STT_CONTRACT_CHILD", "TRANSFORMER_REPORT_DIR", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY":
					continue
				}
				command.Env = append(command.Env, value)
			}
			command.Env = append(command.Env, "RAPIDA_AZURE_STT_CONTRACT_CHILD=1", "NO_PROXY=127.0.0.1,localhost")
			output, err := command.CombinedOutput()
			gomega.Expect(childCtx.Err()).NotTo(gomega.HaveOccurred(), "native lifecycle exceeded its deadline")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "%s", output)
			ginkgo.By("recording only allowlisted lifecycle diagnostics from the child")
			var startWaiters, stopWaiters int
			found := false
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "AZURE_STT_DIAGNOSTIC ") {
					count, err := fmt.Sscanf(line, "AZURE_STT_DIAGNOSTIC %d %d", &startWaiters, &stopWaiters)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(count).To(gomega.Equal(2))
					found = true
				}
			}
			gomega.Expect(found).To(gomega.BeTrue(), "child must finish its lifecycle assertions")
			ginkgo.AddReportEntry("packet-timeline", map[string]interface{}{
				"context_id": "rejected", "transport": "loopback-websocket", "error_observed": true,
				"closed_observed": true, "start_result_waiters": startWaiters, "stop_result_waiters": stopWaiters,
			}, ginkgo.ReportEntryVisibilityAlways)
			return
		}

		ginkgo.By("creating a loopback server that rejects the SDK WebSocket upgrade")
		requests := make(chan bool, 8)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			select {
			case requests <- request.Method == http.MethodGet && strings.EqualFold(request.Header.Get("Upgrade"), "websocket"):
			default:
			}
			w.WriteHeader(http.StatusUnauthorized)
		}))
		ginkgo.DeferCleanup(server.Close)
		endpoint, err := url.Parse(server.URL)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(net.ParseIP(endpoint.Hostname()).IsLoopback()).To(gomega.BeTrue())
		endpoint.Scheme = "ws"
		endpoint.Path = "/speech/recognition/conversation/cognitiveservices/v1"
		credential := testutil.BuildCredential(map[string]string{"subscription_key": "offline-key", "endpoint": endpoint.String()})
		opts := testutil.BuildOptions(map[string]interface{}{"listen.language": "en-US"})
		providerOptions, err := azure.NewAzureOption(testutil.NewTestLogger(), credential, opts)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		config, err := providerOptions.SpeechToTextOption()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(config.GetProperty(common.SpeechServiceConnectionEndpoint)).To(gomega.Equal(endpoint.String()))
		config.Close()

		ginkgo.By("initializing the real SDK only after verifying its configured local target")
		collector := testutil.NewPacketCollector()
		stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(testutil.NewTestLogger()),
			azure.WithCredential(credential), azure.WithOptions(opts), azure.WithOnPacket(collector.OnPacket))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		closed := false
		ginkgo.DeferCleanup(func() {
			if !closed {
				gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed())
			}
		})
		gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "rejected"})).To(gomega.Succeed())
		gomega.Expect(stt.Initialize()).To(gomega.Succeed())
		gomega.Eventually(requests).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.Receive(gomega.BeTrue()))
		ginkgo.By("observing rejection rather than successful recognition")
		gomega.Eventually(collector.GetPackets).WithContext(ctx).WithTimeout(5 * time.Second).Should(gomega.ContainElement(gomega.SatisfyAll(
			gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{}),
			gomega.HaveField("ContextID", "rejected"), gomega.HaveField("Error", gomega.MatchError("azure-stt: recognition cancelled")))))
		gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
		gomega.Expect(collector.EventPackets()).NotTo(gomega.ContainElement(gomega.HaveField("Record.Event", observability.STTCompleted)))
		ginkgo.By("closing native resources and diagnosing unconsumed SDK results")
		gomega.Expect(stt.Close(ctx)).To(gomega.Succeed())
		closed = true
		gomega.Expect(collector.EventPackets()).To(gomega.ContainElement(gomega.HaveField("Record.Event", observability.STTClosed)))
		var stacks bytes.Buffer
		gomega.Expect(pprof.Lookup("goroutine").WriteTo(&stacks, 2)).To(gomega.Succeed())
		var startWaiters, stopWaiters int
		for _, stack := range strings.Split(stacks.String(), "\n\n") {
			if !strings.Contains(stack, "[chan send]") {
				continue
			}
			if strings.Contains(stack, "speech.SpeechRecognizer.StartContinuousRecognitionAsync.func1()") {
				startWaiters++
			}
			if strings.Contains(stack, "speech.SpeechRecognizer.StopContinuousRecognitionAsync.func1()") {
				stopWaiters++
			}
		}
		fmt.Fprintf(os.Stdout, "\nAZURE_STT_DIAGNOSTIC %d %d\n", startWaiters, stopWaiters)
	}, ginkgo.SpecTimeout(25*time.Second))
})
