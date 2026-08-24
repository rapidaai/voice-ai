// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/require"
)

func newPipelineTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(
		commons.Level("error"),
		commons.Name("sip-pipeline-test"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return l
}

func newPipelineTestSession(t *testing.T) *sip_infra.Session {
	t.Helper()
	s, err := sip_infra.NewSession(context.Background(),
		sip_infra.WithSessionConfig(&sip_infra.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_infra.WithSessionDirection(sip_infra.CallDirectionInbound),
	)
	require.NoError(t, err)
	return s
}

func TestHandleSessionEstablished_ConversationErrorEndsSession(t *testing.T) {
	t.Parallel()

	transferServer := &fakeTransferServer{}
	d := New(
		WithLogger(newPipelineTestLogger(t)),
		WithTransferServer(transferServer),
	)

	s := newPipelineTestSession(t)
	d.handleSessionEstablished(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-setup-fail",
		Session:     s,
		Direction:   sip_infra.CallDirectionInbound,
		AssistantID: 1,
	})

	require.Eventually(t, s.IsEnded, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, []sip_infra.LifecycleReason{sip_infra.LifecycleReasonPipelineConversationFailed}, transferServer.lifecycleEndReasons())
}

func TestDispatcherBackpressureAndTeardownStress(t *testing.T) {
	logger := newPipelineTestLogger(t)

	const calls = 400

	transferServer := &fakeTransferServer{}

	d := New(
		WithLogger(logger),
		WithTransferServer(transferServer),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	for i := 0; i < calls; i++ {
		s := newPipelineTestSession(t)
		d.OnPipeline(ctx, sip_infra.SessionEstablishedPipeline{
			ID:          fmt.Sprintf("call-%d", i),
			Session:     s,
			Direction:   sip_infra.CallDirectionInbound,
			AssistantID: 1,
		})
	}

	require.Eventually(t, func() bool {
		return len(transferServer.lifecycleEndReasons()) == calls
	}, 10*time.Second, 10*time.Millisecond)
}
