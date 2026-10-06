// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

var _ = ginkgo.Describe("AssemblyAI STT pre-initialization contract",
	ginkgo.Label("offline", "stt", "assemblyai", "lifecycle"), func() {
		ginkgo.It("closes an unused session without transcript output or usage", func(ctx ginkgo.SpecContext) {
			ginkgo.By("constructing without starting the provider connection")
			collector := testutil.NewPacketCollector()
			stt, err := transformer.NewSpeechToText(
				transformer.WithContext(ctx), transformer.WithLogger(testutil.NewTestLogger()),
				transformer.WithProvider("assemblyai"),
				transformer.WithCredential(testutil.BuildCredential(map[string]string{"key": "offline-key"})),
				transformer.WithOnPacket(collector.OnPacket))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(stt).NotTo(gomega.BeNil())
			ginkgo.DeferCleanup(func() { gomega.Expect(stt.Close(context.Background())).To(gomega.Succeed()) })

			ginkgo.By("closing with an already canceled caller and repeating cleanup")
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			closed := make(chan error, 1)
			go func() { closed <- stt.Close(canceled) }()
			gomega.Eventually(closed).WithContext(ctx).WithTimeout(time.Second).Should(gomega.Receive(gomega.BeNil()))
			gomega.Expect(stt.Close(canceled)).To(gomega.Succeed())
			gomega.Expect(collector.TranscriptPackets()).To(gomega.BeEmpty())
			gomega.Expect(collector.MetricPackets()).To(gomega.BeEmpty())
			gomega.Expect(collector.GetPackets()).NotTo(gomega.ContainElement(gomega.BeAssignableToTypeOf(internal_type.ObservabilityUsageRecordPacket{})))
			gomega.Expect(collector.EventPackets()).To(gomega.ContainElement(gomega.HaveField("Record.Event", observability.STTClosed)))
		}, ginkgo.SpecTimeout(5*time.Second))
	})
