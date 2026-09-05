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

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHOutboundUsernamePassword(t *testing.T) {
	trunk := loadOutboundTrunkConfig(t)
	harness := newFreeSWITCHHarness(t, trunk.sipCredentialConfig)

	session, err := harness.server.MakeCall(
		context.Background(),
		harness.sipConfig,
		trunk.answerUser,
		trunk.fromUser,
		sip_runtime.MakeCallOptions{},
	)
	require.NoErrorf(t, err, "outbound call to %s failed", freeSWITCHOutboundTargetDescription(harness.config, trunk.answerUser))
	require.NotNil(t, session)

	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)
	require.NoError(t, harness.server.EndCallWithReason(session, sip_runtime.LifecycleReasonEndCall))
	waitForTerminalCallState(t, session, callTeardownTimeout)
}

func TestFreeSWITCHOutboundRequestContextCancellationDoesNotCancelCall(t *testing.T) {
	trunk := loadOutboundTrunkConfig(t)
	harness := newFreeSWITCHHarness(t, trunk.sipCredentialConfig)
	requestContext, cancelRequest := context.WithCancel(context.Background())

	session, err := harness.server.MakeCall(
		requestContext,
		harness.sipConfig,
		trunk.answerUser,
		trunk.fromUser,
		sip_runtime.MakeCallOptions{},
	)
	require.NoErrorf(t, err, "outbound call to %s failed", freeSWITCHOutboundTargetDescription(harness.config, trunk.answerUser))
	require.NotNil(t, session)
	cancelRequest()

	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)
	require.NoError(t, harness.server.EndCallWithReason(session, sip_runtime.LifecycleReasonEndCall))
	waitForTerminalCallState(t, session, callTeardownTimeout)
}

func TestFreeSWITCHOutboundUsernamePasswordHeaders(t *testing.T) {
	trunk := loadOutboundTrunkConfig(t)
	if trunk.headerAssertUser == "" {
		t.Skip("FREESWITCH_OUTBOUND_HEADER_ASSERT_USER is required to assert custom INVITE headers")
	}
	harness := newFreeSWITCHHarness(t, trunk.sipCredentialConfig)
	harness.sipConfig.CustomHeaders = map[string]string{
		"X-Rapida-Integration-Test": "headers-ok",
		"X-Rapida-Test-Trace":       "freeswitch-outbound",
	}

	session, err := harness.server.MakeCall(
		context.Background(),
		harness.sipConfig,
		trunk.headerAssertUser,
		trunk.fromUser,
		sip_runtime.MakeCallOptions{},
	)
	require.NoErrorf(t, err, "outbound call to %s failed", freeSWITCHOutboundTargetDescription(harness.config, trunk.headerAssertUser))
	require.NotNil(t, session)

	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)
	require.NoError(t, harness.server.EndCallWithReason(session, sip_runtime.LifecycleReasonEndCall))
	waitForTerminalCallState(t, session, callTeardownTimeout)
}

func TestFreeSWITCHOutboundCancelBeforeAnswer(t *testing.T) {
	trunk := loadOutboundTrunkConfig(t)
	if trunk.ringOnlyUser == "" {
		t.Skip("FREESWITCH_OUTBOUND_RING_ONLY_USER is required to validate pre-answer CANCEL")
	}
	harness := newFreeSWITCHHarness(t, trunk.sipCredentialConfig)

	session, err := harness.server.MakeCall(
		context.Background(),
		harness.sipConfig,
		trunk.ringOnlyUser,
		trunk.fromUser,
		sip_runtime.MakeCallOptions{},
	)
	require.NoErrorf(t, err, "outbound call to %s failed", freeSWITCHOutboundTargetDescription(harness.config, trunk.ringOnlyUser))
	require.NotNil(t, session)

	waitForCallState(t, session, sip_runtime.CallStateRinging, callSetupTimeout)
	require.NoError(t, harness.server.CancelCall(session, sip_runtime.LifecycleReasonOutboundCancelledBeforeAnswer))
	waitForTerminalCallState(t, session, callTeardownTimeout)
	require.Equal(t, sip_runtime.CallStateCancelled, session.GetInfo().State)
}
