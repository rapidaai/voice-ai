// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package freeswitch_test

import (
	"context"
	"testing"
	"time"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHOutboundBusy(t *testing.T) {
	endpoints := loadOutboundFailureEndpointConfig(t)
	harness := newFreeSWITCHHarness(t, endpoints.sipCredentialConfig)

	assertFreeSWITCHOutboundFailure(t, harness, endpoints.busyUser, expectedOutboundFailure{
		class:      "busy",
		reason:     sip_runtime.LifecycleReasonOutboundRejected.String(),
		statusCode: 486,
	})
}

func TestFreeSWITCHOutboundNoAnswer(t *testing.T) {
	endpoints := loadOutboundFailureEndpointConfig(t)
	harness := newFreeSWITCHHarness(t, endpoints.sipCredentialConfig)
	harness.sipConfig.InviteTimeout = 250 * time.Millisecond

	assertFreeSWITCHOutboundFailure(t, harness, endpoints.noAnswerUser, expectedOutboundFailure{
		class:      "no_answer",
		reason:     sip_runtime.LifecycleReasonOutboundNoAnswer.String(),
		statusCode: 0,
		retryable:  true,
	})
}

func TestFreeSWITCHOutboundRejected(t *testing.T) {
	endpoints := loadOutboundFailureEndpointConfig(t)
	harness := newFreeSWITCHHarness(t, endpoints.sipCredentialConfig)

	assertFreeSWITCHOutboundFailure(t, harness, endpoints.rejectedUser, expectedOutboundFailure{
		class:      "rejected",
		reason:     sip_runtime.LifecycleReasonOutboundRejected.String(),
		statusCode: 603,
	})
}

func TestFreeSWITCHOutboundUnavailable(t *testing.T) {
	endpoints := loadOutboundFailureEndpointConfig(t)
	harness := newFreeSWITCHHarness(t, endpoints.sipCredentialConfig)

	assertFreeSWITCHOutboundFailure(t, harness, endpoints.unavailableUser, expectedOutboundFailure{
		class:      "unavailable",
		reason:     sip_runtime.LifecycleReasonOutboundUnavailable.String(),
		statusCode: 480,
		retryable:  true,
	})
}

func TestFreeSWITCHOutboundMediaRejected(t *testing.T) {
	endpoints := loadOutboundFailureEndpointConfig(t)
	harness := newFreeSWITCHHarness(t, endpoints.sipCredentialConfig)

	assertFreeSWITCHOutboundFailure(t, harness, endpoints.mediaRejectUser, expectedOutboundFailure{
		class:      "media",
		reason:     sip_runtime.LifecycleReasonOutboundMediaRejected.String(),
		statusCode: 488,
	})
}

type expectedOutboundFailure struct {
	class      string
	reason     string
	statusCode int
	retryable  bool
}

func assertFreeSWITCHOutboundFailure(t *testing.T, harness *freeSWITCHHarness, toUser string, expected expectedOutboundFailure) {
	t.Helper()
	statusRecorder := newIntegrationOutboundStatusRecorder()

	session, err := harness.server.MakeCall(
		context.Background(),
		harness.sipConfig,
		toUser,
		harness.sipConfig.Username,
		sip_runtime.MakeCallOptions{CallStatusObserver: statusRecorder.Record},
	)
	require.NoErrorf(t, err, "outbound failure scenario %s did not create a SIP session", freeSWITCHOutboundTargetDescription(harness.config, toUser))
	require.NotNil(t, session)

	waitForCallState(t, session, sip_runtime.CallStateFailed, callTeardownTimeout)
	waitForTerminalCallState(t, session, callTeardownTimeout)

	status := statusRecorder.LastStatus(t, "failed")
	require.Equal(t, expected.class, status.FailureClass)
	require.Equal(t, expected.reason, status.DisconnectReason)
	require.Equal(t, expected.statusCode, status.ProviderStatusCode)
	require.Equal(t, expected.retryable, status.Retryable)

	failureClass, ok := session.GetMetadata("sip.failure_class")
	require.True(t, ok)
	require.Equal(t, expected.class, failureClass)
}
