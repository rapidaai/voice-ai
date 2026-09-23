// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"time"

	"cloud.google.com/go/speech/apiv2/speechpb"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	google "github.com/rapidaai/api/assistant-api/internal/transformer/google"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	"github.com/rapidaai/pkg/commons"
)

var _ = ginkgo.Describe("Google STT contracts", ginkgo.Label("offline", "stt", "google-speech-service", "sdk", "configuration"), func() {
	ginkgo.It("maps multilingual recognition options without constructing a network client", func() {
		ginkgo.By("building public options with empty language segments and explicit model and region")
		opts, err := google.NewGoogleOption(testutil.NewTestLogger(),
			testutil.BuildCredential(map[string]string{"project_id": "offline-project"}),
			testutil.BuildOptions(map[string]interface{}{
				"listen.language": " en-US " + commons.SEPARATOR + " " + commons.SEPARATOR + " hi-IN ",
				"listen.model":    "chirp_2", "listen.region": "us-central1",
			}))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		ginkgo.By("checking the recognizer resource and wire configuration")
		gomega.Expect(opts.GetRecognizer()).To(gomega.Equal("projects/offline-project/locations/us-central1/recognizers/_"))
		config := opts.SpeechToTextOptions()
		gomega.Expect(config.GetConfig().GetLanguageCodes()).To(gomega.Equal([]string{"en-US", "hi-IN"}))
		gomega.Expect(config.GetConfig().GetModel()).To(gomega.Equal("chirp_2"))
		decoding := config.GetConfig().GetExplicitDecodingConfig()
		gomega.Expect(decoding.GetEncoding()).To(gomega.Equal(speechpb.ExplicitDecodingConfig_LINEAR16))
		gomega.Expect(decoding.GetSampleRateHertz()).To(gomega.Equal(int32(16000)))
		gomega.Expect(decoding.GetAudioChannelCount()).To(gomega.Equal(int32(1)))
		gomega.Expect(config.GetStreamingFeatures().GetInterimResults()).To(gomega.BeTrue())
		gomega.Expect(config.GetStreamingFeatures().GetEnableVoiceActivityEvents()).To(gomega.BeFalse())
	})

	for _, condition := range []string{"absent", "nil"} {
		ginkgo.It("uses default recognition settings when options are "+condition, func() {
			ginkgo.By("building options that cannot supply language, model or region strings")
			values := map[string]interface{}{}
			if condition == "nil" {
				values = map[string]interface{}{"listen.language": nil, "listen.model": nil, "listen.region": nil}
			}
			opts, err := google.NewGoogleOption(testutil.NewTestLogger(),
				testutil.BuildCredential(map[string]string{"project_id": "offline-project"}), testutil.BuildOptions(values))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			ginkgo.By("checking fallback language, model, global recognizer and recognition features")
			config := opts.SpeechToTextOptions()
			gomega.Expect(config.GetConfig().GetLanguageCodes()).To(gomega.Equal([]string{google.DefaultLanguageCode}))
			gomega.Expect(config.GetConfig().GetModel()).To(gomega.Equal(google.DefaultModel))
			gomega.Expect(opts.GetRecognizer()).To(gomega.Equal("projects/offline-project/locations/global/recognizers/_"))
			features := config.GetConfig().GetFeatures()
			gomega.Expect(features.GetEnableAutomaticPunctuation()).To(gomega.BeTrue())
			gomega.Expect(features.GetEnableWordConfidence()).To(gomega.BeTrue())
			gomega.Expect(features.GetProfanityFilter()).To(gomega.BeTrue())
			gomega.Expect(features.GetEnableSpokenPunctuation()).To(gomega.BeTrue())
		})
	}

	ginkgo.It("returns independent recognition configurations for successive streams", func() {
		ginkgo.By("building the first stream configuration")
		opts, err := google.NewGoogleOption(testutil.NewTestLogger(), testutil.BuildCredential(nil), testutil.BuildOptions(nil))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		first := opts.SpeechToTextOptions()
		ginkgo.By("mutating the returned protobuf without changing the next stream's configuration")
		first.Config.LanguageCodes[0] = "fr-FR"
		first.Config.Model = "changed-model"
		first.Config.GetExplicitDecodingConfig().SampleRateHertz = 8000
		first.StreamingFeatures.InterimResults = false
		second := opts.SpeechToTextOptions()
		gomega.Expect(second.GetConfig().GetLanguageCodes()).To(gomega.Equal([]string{google.DefaultLanguageCode}))
		gomega.Expect(second.GetConfig().GetModel()).To(gomega.Equal(google.DefaultModel))
		gomega.Expect(second.GetConfig().GetExplicitDecodingConfig().GetSampleRateHertz()).To(gomega.Equal(int32(16000)))
		gomega.Expect(second.GetStreamingFeatures().GetInterimResults()).To(gomega.BeTrue())
	})

	for _, credential := range []struct{ name, value, message string }{
		{"malformed service account JSON", `{`, "unexpected end of JSON input"},
		{"unsupported credential type", `{"type":"offline-unsupported"}`, "unsupported"},
	} {
		ginkgo.It("rejects "+credential.name+" before gRPC dialing or packet emission", func(ctx ginkgo.SpecContext) {
			ginkgo.By("supplying explicit invalid credentials without an API key or ADC fallback")
			collector := testutil.NewPacketCollector()
			// Valid credentials would start an eager gRPC dial, even with a canceled context.
			stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(testutil.NewTestLogger()),
				google.WithCredential(testutil.BuildCredential(map[string]string{"service_account_key": credential.value})),
				google.WithOptions(testutil.BuildOptions(nil)), google.WithOnPacket(collector.OnPacket))
			ginkgo.By("reporting setup failure before a transformer or stream exists")
			gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(credential.message)))
			gomega.Expect(stt).To(gomega.BeNil())
			gomega.Expect(collector.GetPackets()).To(gomega.BeEmpty())
		}, ginkgo.SpecTimeout(5*time.Second))
	}
})
