// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

var _ = ginkgo.Describe("Transformer factory contract", ginkgo.Label("offline", "factory"), func() {
	for _, provider := range []struct {
		id, ttsName, sttName string
		credential           map[string]string
	}{
		// Deepgram STT opens its socket at construction; its local transport specs cover selection.
		{"deepgram", "deepgram-tts", "", map[string]string{"key": "test-key"}},
		{"azure-speech-service", "azure-tts", "azure-stt", map[string]string{"subscription_key": "test-key", "endpoint": "https://example.invalid"}},
		{"cartesia", "cartesia-tts", "cartesia-stt", map[string]string{"key": "test-key"}},
		{"google-speech-service", "google-tts", "google-stt", map[string]string{"key": "test-key", "project_id": "test-project"}},
		{"sarvamai", "sarvam-tts", "sarvam-stt", map[string]string{"key": "test-key"}},
		{"elevenlabs", "elevenlabs-tts", "", map[string]string{"key": "test-key"}},
		{"rime", "rime-tts", "", map[string]string{"key": "test-key"}},
		{"resembleai", "resembleai-tts", "", map[string]string{"key": "test-key"}},
		{"neuphonic", "neuphonic-tts", "", map[string]string{"key": "test-key"}},
		{"minimax", "minimax-tts", "", map[string]string{"key": "test-key", "group_id": "test-group"}},
		{"groq", "groq-tts", "groq-stt", map[string]string{"key": "test-key"}},
		{"speechmatics", "speechmatics-tts", "speechmatics-stt", map[string]string{"key": "test-key"}},
		{"nvidia", "nvidia-tts", "nvidia-stt", map[string]string{"key": "test-key", "function_id": "test-function"}},
		{"aws", "aws-tts", "aws-stt", map[string]string{"access_key_id": "test-access", "secret_access_key": "test-secret"}},
		{"smallest", "smallest-tts", "smallest-stt", map[string]string{"key": "test-key"}},
		{"assemblyai", "", "assemblyai-stt", map[string]string{"key": "test-key"}},
	} {
		ginkgo.Context(provider.id, ginkgo.Label(provider.id), func() {
			if provider.ttsName != "" {
				ginkgo.It("selects the expected TTS implementation", ginkgo.Label("tts"), func(ctx ginkgo.SpecContext) {
					ginkgo.By("constructing with valid synthetic credentials without opening a connection")
					tts, err := transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), provider.id,
						testutil.BuildCredential(provider.credential), func(...internal_type.Packet) error { return nil }, utils.Option{})
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(tts).NotTo(gomega.BeNil())
					ginkgo.DeferCleanup(func() { gomega.Expect(tts.Close(context.Background())).To(gomega.Succeed()) })
					gomega.Expect(tts.Name()).To(gomega.Equal(provider.ttsName))
				})
			}
			if provider.sttName != "" {
				ginkgo.It("selects the expected STT implementation", ginkgo.Label("stt"), func(ctx ginkgo.SpecContext) {
					ginkgo.By("constructing with valid synthetic credentials without opening a connection")
					stt, err := transformer.NewSpeechToText(
						transformer.WithContext(ctx), transformer.WithLogger(testutil.NewTestLogger()),
						transformer.WithProvider(provider.id), transformer.WithCredential(testutil.BuildCredential(provider.credential)),
						transformer.WithOnPacket(func(...internal_type.Packet) error { return nil }), transformer.WithOptions(utils.Option{}))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(stt).NotTo(gomega.BeNil())
					ginkgo.DeferCleanup(func() { gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed()) })
					gomega.Expect(stt.Name()).To(gomega.Equal(provider.sttName))
				})
			}
		})
	}

	ginkgo.It("selects custom WebSocket TTS", ginkgo.Label("tts", "custom-tts"), func(ctx ginkgo.SpecContext) {
		ginkgo.By("supplying a valid custom transport and packet mapping")
		tts, err := transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), "custom-tts",
			testutil.BuildCredential(map[string]string{"baseUrl": "wss://example.invalid", "apiCompatibility": "websocket_v1"}),
			func(...internal_type.Packet) error { return nil }, utils.Option{
				"speak.request_rules":  `[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"}}}}]`,
				"speak.response_rules": `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
			})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(tts).NotTo(gomega.BeNil())
		ginkgo.DeferCleanup(func() { gomega.Expect(tts.Close(context.Background())).To(gomega.Succeed()) })
		gomega.Expect(tts.Name()).To(gomega.Equal("custom-tts-websocket-v1"))
	})

	for _, transport := range []struct{ compatibility, endpoint, name string }{
		{"http_v1", "https://example.invalid", "custom-stt-http-v1"},
		{"websocket_v1", "wss://example.invalid", "custom-stt-websocket-v1"},
	} {
		ginkgo.It("selects custom STT "+transport.compatibility, ginkgo.Label("stt", "custom-stt"), func(ctx ginkgo.SpecContext) {
			ginkgo.By("selecting the configured custom transport")
			stt, err := transformer.NewSpeechToText(
				transformer.WithContext(ctx), transformer.WithLogger(testutil.NewTestLogger()), transformer.WithProvider("custom-stt"),
				transformer.WithCredential(testutil.BuildCredential(map[string]string{"baseUrl": transport.endpoint, "apiCompatibility": transport.compatibility})),
				transformer.WithOnPacket(func(...internal_type.Packet) error { return nil }), transformer.WithOptions(utils.Option{
					"listen.request_rules":  `[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.wav_base64"}}}}]`,
					"listen.response_rules": `[{"when":{"frame":"json"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
				}))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt).NotTo(gomega.BeNil())
			ginkgo.DeferCleanup(func() { gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed()) })
			gomega.Expect(stt.Name()).To(gomega.Equal(transport.name))
		})
	}

	ginkgo.It("rejects unknown providers and unsupported RevAI synthesis", func(ctx ginkgo.SpecContext) {
		ginkgo.By("rejecting an unknown TTS provider")
		tts, err := transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), "unknown", nil, nil, nil)
		gomega.Expect(err).To(gomega.MatchError("illegal text to speech idenitfier"))
		gomega.Expect(tts).To(gomega.BeNil())
		ginkgo.By("rejecting an unknown STT provider")
		stt, err := transformer.NewSpeechToText(transformer.WithContext(ctx), transformer.WithProvider("unknown"))
		gomega.Expect(err).To(gomega.HaveOccurred())
		gomega.Expect(stt).To(gomega.BeNil())
		ginkgo.By("rejecting a provider without synthesis support")
		tts, err = transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), "revai", nil, nil, nil)
		gomega.Expect(err).To(gomega.MatchError("revai-tts: text-to-speech is not supported"))
		gomega.Expect(tts).To(gomega.BeNil())
	})
})
