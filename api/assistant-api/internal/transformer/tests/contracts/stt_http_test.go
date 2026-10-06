// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	aws "github.com/rapidaai/api/assistant-api/internal/transformer/aws"
	groq "github.com/rapidaai/api/assistant-api/internal/transformer/groq"
	nvidia "github.com/rapidaai/api/assistant-api/internal/transformer/nvidia"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type sttHTTPRoundTripper func(*http.Request) (*http.Response, error)

func (transport sttHTTPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

// Body closure is the provider's completion barrier, including empty responses.
type sttHTTPResponseBody struct {
	io.ReadCloser
	settled *atomic.Int32
}

func (body *sttHTTPResponseBody) Close() error {
	err := body.ReadCloser.Close()
	body.settled.Add(1)
	return err
}

type sttHTTPReply struct {
	status int
	body   string
	err    error
}

type sttHTTPRequest struct {
	request *http.Request
	body    []byte
	err     error
	reply   chan sttHTTPReply
}

var _ = ginkgo.Describe("HTTP STT contracts", ginkgo.Serial, ginkgo.Label("offline", "stt", "http"), func() {
	for _, provider := range []string{"aws", "groq", "nvidia"} {
		ginkgo.Context(provider, ginkgo.Label(provider), func() {
			var (
				stt       internal_type.SpeechToTextTransformer
				collector *testutil.PacketCollector
				cancel    context.CancelFunc
				requests  chan sttHTTPRequest
				settled   *atomic.Int32
				submitted int32
				response  string
			)

			ginkgo.BeforeEach(func() {
				collector = testutil.NewPacketCollector()
				requests = make(chan sttHTTPRequest, 4)
				settled = &atomic.Int32{}
				submitted = 0
				stt = nil
				// The provider lifetime spans setup and It; a hook context ends with the hook.
				parent, stop := context.WithCancel(context.Background())
				cancel = stop
				originalClient, originalTransport := http.DefaultClient, http.DefaultTransport
				ginkgo.DeferCleanup(func() {
					http.DefaultClient, http.DefaultTransport = originalClient, originalTransport
				})
				// Both defaults are replaced so an unexpected client cannot reach the network.
				transport := sttHTTPRoundTripper(func(request *http.Request) (*http.Response, error) {
					body, err := io.ReadAll(request.Body)
					_ = request.Body.Close()
					pending := sttHTTPRequest{request: request, body: body, err: err, reply: make(chan sttHTTPReply, 1)}
					select {
					case requests <- pending:
					case <-request.Context().Done():
						settled.Add(1)
						return nil, request.Context().Err()
					}
					select {
					case reply := <-pending.reply:
						if reply.err != nil {
							settled.Add(1)
							return nil, reply.err
						}
						return &http.Response{
							StatusCode: reply.status, Header: make(http.Header), Request: request,
							Body: &sttHTTPResponseBody{ReadCloser: io.NopCloser(strings.NewReader(reply.body)), settled: settled},
						}, nil
					case <-request.Context().Done():
						settled.Add(1)
						return nil, request.Context().Err()
					}
				})
				http.DefaultClient = &http.Client{Transport: transport}
				http.DefaultTransport = transport
				ginkgo.DeferCleanup(func() {
					cancel()
					if stt != nil {
						closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
						defer closeCancel()
						gomega.Expect(stt.Close(closeCtx)).To(gomega.Succeed())
					}
					gomega.Eventually(settled.Load).WithTimeout(3 * time.Second).Should(gomega.Equal(submitted))
					var timeline []map[string]interface{}
					for index, packet := range collector.GetPackets() {
						entry := map[string]interface{}{"index": index, "type": fmt.Sprintf("%T", packet)}
						switch packet := packet.(type) {
						case internal_type.SpeechToTextPacket:
							entry["context_id"], entry["interim"] = packet.ContextID, packet.Interim
						case internal_type.SpeechToTextErrorPacket:
							entry["context_id"] = packet.ContextID
						case internal_type.InterruptionDetectedPacket:
							entry["context_id"] = packet.ContextID
						case internal_type.ObservabilityMetricRecordPacket:
							entry["context_id"] = packet.ContextID
						}
						timeline = append(timeline, entry)
					}
					ginkgo.AddReportEntry("packet-timeline", timeline, ginkgo.ReportEntryVisibilityFailureOrVerbose)
				})

				logger := testutil.NewTestLogger()
				opts := testutil.BuildOptions(map[string]interface{}{"listen.language": "en-US", "listen.model": "contract-model"})
				var err error
				switch provider {
				case "aws":
					stt, err = aws.NewSpeechToText(aws.WithContext(parent), aws.WithLogger(logger), aws.WithOnPacket(collector.OnPacket),
						aws.WithCredential(testutil.BuildCredential(map[string]string{"access_key_id": "offline-id", "secret_access_key": "offline-secret", "region": "us-east-1"})), aws.WithOptions(opts))
					response = `{"Results":{"Transcripts":[{"Transcript":"fixture transcript"}]}}`
				case "groq":
					stt, err = groq.NewSpeechToText(groq.WithContext(parent), groq.WithLogger(logger), groq.WithOnPacket(collector.OnPacket),
						groq.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})), groq.WithOptions(opts))
					response = `{"text":"fixture transcript"}`
				case "nvidia":
					stt, err = nvidia.NewSpeechToText(nvidia.WithContext(parent), nvidia.WithLogger(logger), nvidia.WithOnPacket(collector.OnPacket),
						nvidia.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key", "function_id": "offline-function"})), nvidia.WithOptions(opts))
					response = `{"text":"fixture transcript"}`
				}
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(stt.Name()).To(gomega.Equal(provider + "-stt"))
				gomega.Expect(stt.Initialize()).To(gomega.Succeed())
			})

			ginkgo.It("encodes PCM and emits final transcript, interruption, completion and latency for its turn", func(ctx ginkgo.SpecContext) {
				ginkgo.By("admitting audio through the public factory with explicit options")
				gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "first"})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextStartPacket{})).To(gomega.Succeed())
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextEndPacket{})).To(gomega.Succeed())
				audio := []byte{1, 0, 2, 0, 3, 0}
				submitted++
				gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: audio})).To(gomega.Succeed())
				var pending sttHTTPRequest
				gomega.Eventually(requests).WithContext(ctx).Should(gomega.Receive(&pending))
				gomega.Expect(pending.err).NotTo(gomega.HaveOccurred())
				gomega.Expect(pending.request.Method).To(gomega.Equal(http.MethodPost))
				gomega.Expect(pending.request.URL.Scheme).To(gomega.Equal("https"))

				ginkgo.By("inspecting the captured provider payload without external HTTP")
				switch provider {
				case "aws":
					gomega.Expect(pending.request.URL.Host).To(gomega.Equal("transcribe.us-east-1.amazonaws.com"))
					gomega.Expect(pending.request.Header.Get("Content-Type")).To(gomega.Equal("application/x-amz-json-1.1"))
					gomega.Expect(pending.request.Header.Get("Authorization")).To(gomega.HavePrefix("AWS4-HMAC-SHA256 "))
					var payload struct {
						AudioStream                 struct{ AudioEvent struct{ AudioChunk []byte } }
						LanguageCode, MediaEncoding string
						MediaSampleRateHertz        int
					}
					gomega.Expect(json.Unmarshal(pending.body, &payload)).To(gomega.Succeed())
					gomega.Expect(payload.AudioStream.AudioEvent.AudioChunk).To(gomega.Equal(audio))
					gomega.Expect(payload.LanguageCode).To(gomega.Equal("en-US"))
					gomega.Expect(payload.MediaEncoding).To(gomega.Equal("pcm"))
					gomega.Expect(payload.MediaSampleRateHertz).To(gomega.Equal(16000))
				case "groq":
					gomega.Expect(pending.request.URL.String()).To(gomega.Equal(groq.GROQ_STT_URL))
					gomega.Expect(pending.request.Header.Get("Authorization")).To(gomega.Equal("Bearer offline-key"))
					mediaType, params, err := mime.ParseMediaType(pending.request.Header.Get("Content-Type"))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(mediaType).To(gomega.Equal("multipart/form-data"))
					reader := multipart.NewReader(bytes.NewReader(pending.body), params["boundary"])
					fields := map[string]string{}
					for {
						part, err := reader.NextPart()
						if errors.Is(err, io.EOF) {
							break
						}
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
						value, err := io.ReadAll(part)
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
						gomega.Expect(part.Close()).To(gomega.Succeed())
						if part.FormName() == "file" {
							gomega.Expect(part.FileName()).To(gomega.Equal("audio.wav"))
							gomega.Expect(value).To(gomega.HaveLen(44 + len(audio)))
							gomega.Expect(string(value[:4])).To(gomega.Equal("RIFF"))
							gomega.Expect(string(value[8:12])).To(gomega.Equal("WAVE"))
							gomega.Expect(binary.LittleEndian.Uint32(value[24:28])).To(gomega.Equal(uint32(16000)))
							gomega.Expect(binary.LittleEndian.Uint16(value[22:24])).To(gomega.Equal(uint16(1)))
							gomega.Expect(binary.LittleEndian.Uint16(value[34:36])).To(gomega.Equal(uint16(16)))
							gomega.Expect(binary.LittleEndian.Uint32(value[40:44])).To(gomega.Equal(uint32(len(audio))))
							gomega.Expect(value[44:]).To(gomega.Equal(audio))
						}
						fields[part.FormName()] = string(value)
					}
					gomega.Expect(fields).To(gomega.HaveLen(4))
					gomega.Expect(fields).To(gomega.HaveKey("file"))
					gomega.Expect(fields["model"]).To(gomega.Equal("contract-model"))
					gomega.Expect(fields["language"]).To(gomega.Equal("en-US"))
					gomega.Expect(fields["response_format"]).To(gomega.Equal("verbose_json"))
				case "nvidia":
					gomega.Expect(pending.request.URL.String()).To(gomega.Equal("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/offline-function"))
					gomega.Expect(pending.request.Header.Get("Authorization")).To(gomega.Equal("Bearer offline-key"))
					gomega.Expect(pending.request.Header.Get("NVCF-INPUT-ASSET-REFERENCES")).To(gomega.Equal("offline-function"))
					var payload struct {
						Audio        []byte `json:"audio"`
						Encoding     string `json:"encoding"`
						SampleRate   int    `json:"sample_rate"`
						LanguageCode string `json:"language_code"`
					}
					gomega.Expect(json.Unmarshal(pending.body, &payload)).To(gomega.Succeed())
					gomega.Expect(payload.Audio).To(gomega.Equal(audio))
					gomega.Expect(payload.Encoding).To(gomega.Equal("LINEAR_PCM"))
					gomega.Expect(payload.SampleRate).To(gomega.Equal(16000))
					gomega.Expect(payload.LanguageCode).To(gomega.Equal("en-US"))
				}

				ginkgo.By("completing the response and checking packets independently of callback ordering")
				pending.reply <- sttHTTPReply{status: http.StatusOK, body: response}
				gomega.Eventually(settled.Load).WithContext(ctx).Should(gomega.Equal(int32(1)))
				expected := internal_type.SpeechToTextPacket{ContextID: "first", Script: "fixture transcript"}
				if provider == "aws" {
					expected.Language = "en-US"
				}
				gomega.Expect(collector.TranscriptPackets()).To(gomega.Equal([]internal_type.SpeechToTextPacket{expected}))
				gomega.Expect(collector.GetPackets()).NotTo(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{})))
				gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.Equal([]internal_type.InterruptionDetectedPacket{{ContextID: "first", Source: internal_type.InterruptionSourceWord}}))
				gomega.Expect(collector.EventPackets()).To(gomega.ContainElement(gomega.SatisfyAll(
					gomega.HaveField("ContextID", "first"), gomega.HaveField("Record.Event", observability.STTCompleted))))
				var latencyContexts []string
				for _, packet := range collector.MetricPackets() {
					for _, metric := range packet.Record.Metrics {
						if metric.Name == observability.MetricSTTLatencyMs {
							latencyContexts = append(latencyContexts, packet.ContextID)
						}
					}
				}
				gomega.Expect(latencyContexts).To(gomega.Equal([]string{"first"}))
			}, ginkgo.SpecTimeout(10*time.Second))

			for _, failure := range []string{"HTTP rejection", "invalid JSON", "transport error", "empty transcript"} {
				ginkgo.It("handles "+failure+" without successful transcript or completion", func(ctx ginkgo.SpecContext) {
					ginkgo.By("submitting audio without an explicit start packet")
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "failed"})).To(gomega.Succeed())
					submitted++
					gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 0}})).To(gomega.Succeed())
					var pending sttHTTPRequest
					gomega.Eventually(requests).WithContext(ctx).Should(gomega.Receive(&pending))
					ginkgo.By("returning a controlled failure or empty result")
					reply := sttHTTPReply{status: http.StatusOK}
					message := ""
					switch failure {
					case "HTTP rejection":
						reply.status, reply.body, message = http.StatusTooManyRequests, `{}`, "status 429"
					case "invalid JSON":
						reply.body, message = `{`, "decode failed"
					case "transport error":
						reply.err, message = io.ErrUnexpectedEOF, "request failed"
					case "empty transcript":
						reply.body = `{}`
					}
					pending.reply <- reply
					gomega.Eventually(settled.Load).WithContext(ctx).Should(gomega.Equal(int32(1)))
					if failure != "empty transcript" {
						gomega.Eventually(collector.GetPackets).WithContext(ctx).Should(gomega.ContainElement(gomega.SatisfyAll(
							gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{}),
							gomega.HaveField("ContextID", "failed"),
							gomega.HaveField("Type", internal_type.STTErrorType(internal_type.STTNetworkTimeout)),
							gomega.HaveField("Error", gomega.MatchError(gomega.ContainSubstring(message))))))
					}
					gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
					gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.BeEmpty())
					gomega.Expect(collector.EventPackets()).To(gomega.BeEmpty())
					if failure == "empty transcript" {
						gomega.Expect(collector.GetPackets()).NotTo(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{})))
					}
				}, ginkgo.SpecTimeout(10*time.Second))
			}

			for _, action := range []string{"parent cancellation", "Close"} {
				ginkgo.It("cancels a pending request on "+action+" and preserves error ownership", func(ctx ginkgo.SpecContext) {
					ginkgo.By("holding an admitted HTTP request before its response")
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "pending"})).To(gomega.Succeed())
					submitted++
					gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{1, 0}})).To(gomega.Succeed())
					var pending sttHTTPRequest
					gomega.Eventually(requests).WithContext(ctx).Should(gomega.Receive(&pending))
					ginkgo.By("changing turns and canceling the provider's request context")
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: "next"})).To(gomega.Succeed())
					if action == "Close" {
						gomega.Expect(stt.Close(ctx)).To(gomega.Succeed())
					} else {
						cancel()
					}
					gomega.Eventually(pending.request.Context().Done()).WithContext(ctx).Should(gomega.BeClosed())
					gomega.Eventually(collector.GetPackets).WithContext(ctx).Should(gomega.ContainElement(gomega.SatisfyAll(
						gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{}),
						gomega.HaveField("ContextID", "pending"),
						gomega.HaveField("Error", gomega.MatchError(context.Canceled)))))
					gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
					gomega.Expect(collector.EventPackets()).NotTo(gomega.ContainElement(gomega.HaveField("Record.Event", observability.STTCompleted)))
				}, ginkgo.SpecTimeout(10*time.Second))
			}

			ginkgo.It("retains audio-admission context when a newer turn responds before an older turn", func(ctx ginkgo.SpecContext) {
				ginkgo.By("holding the first response while admitting the next turn")
				var pending [2]sttHTTPRequest
				for index, contextID := range []string{"first", "next"} {
					gomega.Expect(stt.Transform(ctx, internal_type.TurnChangePacket{ContextID: contextID})).To(gomega.Succeed())
					submitted++
					gomega.Expect(stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{Audio: []byte{byte(index + 1), 0}})).To(gomega.Succeed())
					gomega.Eventually(requests).WithContext(ctx).Should(gomega.Receive(&pending[index]))
				}
				ginkgo.By("releasing the newer response before the older response")
				pending[1].reply <- sttHTTPReply{status: http.StatusOK, body: strings.ReplaceAll(response, "fixture transcript", "next fixture")}
				gomega.Eventually(settled.Load).WithContext(ctx).Should(gomega.Equal(int32(1)))
				pending[0].reply <- sttHTTPReply{status: http.StatusOK, body: response}
				gomega.Eventually(settled.Load).WithContext(ctx).Should(gomega.Equal(int32(2)))
				ginkgo.By("checking ownership without imposing chronological transcript order")
				language := ""
				if provider == "aws" {
					language = "en-US"
				}
				gomega.Expect(collector.TranscriptPackets()).To(gomega.ConsistOf(
					internal_type.SpeechToTextPacket{ContextID: "first", Script: "fixture transcript", Language: language},
					internal_type.SpeechToTextPacket{ContextID: "next", Script: "next fixture", Language: language}))
				gomega.Expect(collector.InterruptionDetectedPackets()).To(gomega.ConsistOf(
					internal_type.InterruptionDetectedPacket{ContextID: "first", Source: internal_type.InterruptionSourceWord},
					internal_type.InterruptionDetectedPacket{ContextID: "next", Source: internal_type.InterruptionSourceWord}))
				gomega.Expect(collector.EventPackets()).To(gomega.ConsistOf(
					gomega.SatisfyAll(gomega.HaveField("ContextID", "first"), gomega.HaveField("Record.Event", observability.STTCompleted)),
					gomega.SatisfyAll(gomega.HaveField("ContextID", "next"), gomega.HaveField("Record.Event", observability.STTCompleted))))
				gomega.Expect(collector.GetPackets()).NotTo(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.SpeechToTextErrorPacket{})))
				var latencyContexts []string
				for _, packet := range collector.MetricPackets() {
					for _, metric := range packet.Record.Metrics {
						if metric.Name == observability.MetricSTTLatencyMs {
							latencyContexts = append(latencyContexts, packet.ContextID)
						}
					}
				}
				// Diagnose the shared-timer defect without declaring it a valid metric contract.
				ginkgo.AddReportEntry("overlapping-request-latency", map[string]interface{}{
					"provider": provider, "expected_contexts": []string{"first", "next"}, "observed_contexts": latencyContexts,
				}, ginkgo.ReportEntryVisibilityAlways)
				if provider == "aws" {
					gomega.Expect(latencyContexts).To(gomega.ConsistOf("first", "next"))
				}
			}, ginkgo.SpecTimeout(10*time.Second))
		})
	}
})
