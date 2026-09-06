// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package freeswitch_test

import (
	"testing"
	"time"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHInboundRemoteByeNormalClearing(t *testing.T) {
	inboundConfig := loadRegistrationInboundConfig(t)
	harness := newFreeSWITCHHarness(t, inboundConfig.sipCredentialConfig)
	registrationClient := harness.registrationClient()
	answeredSessions := make(chan *sip_runtime.Session, 1)
	remoteByeSessions := make(chan *sip_runtime.Session, 1)

	harness.server.SetOnInvite(func(session *sip_runtime.Session, _ string, _ sip_runtime.CallAddress) error {
		answeredSessions <- session
		return nil
	})
	harness.server.SetOnBye(func(session *sip_runtime.Session) error {
		remoteByeSessions <- session
		return nil
	})
	registerFreeSWITCHInboundDID(t, registrationClient, inboundConfig, harness.sipConfig)

	freeSWITCHCallUUID := harness.originateRegisteredInboundCall(inboundConfig.registeredDID, inboundConfig.callerUser)
	t.Cleanup(func() {
		_, _ = harness.runFreeSWITCHCommand("uuid_kill " + freeSWITCHCallUUID)
	})

	session := receiveInboundSession(t, answeredSessions, callSetupTimeout)
	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)

	harness.hangupFreeSWITCHCallWithCause(freeSWITCHCallUUID, "NORMAL_CLEARING")
	remoteByeSession := receiveInboundSession(t, remoteByeSessions, callTeardownTimeout)
	require.Equal(t, session.GetCallID(), remoteByeSession.GetCallID())
	waitForTerminalCallState(t, session, callTeardownTimeout)

	metadata := session.GetDisconnectMetadata()
	require.NotEmpty(t, metadata.Reason)
	require.Contains(t, []string{
		sip_runtime.DisconnectReasonNormalClearing,
		sip_runtime.DisconnectReasonRemoteHangup,
	}, metadata.Reason)
}

func TestFreeSWITCHSystemDisconnectSendsBye(t *testing.T) {
	inboundConfig := loadRegistrationInboundConfig(t)
	harness := newFreeSWITCHHarness(t, inboundConfig.sipCredentialConfig)
	registrationClient := harness.registrationClient()
	answeredSessions := make(chan *sip_runtime.Session, 1)

	harness.server.SetOnInvite(func(session *sip_runtime.Session, _ string, _ sip_runtime.CallAddress) error {
		answeredSessions <- session
		return nil
	})
	registerFreeSWITCHInboundDID(t, registrationClient, inboundConfig, harness.sipConfig)

	freeSWITCHCallUUID := harness.originateRegisteredInboundCall(inboundConfig.registeredDID, inboundConfig.callerUser)
	t.Cleanup(func() {
		_, _ = harness.runFreeSWITCHCommand("uuid_kill " + freeSWITCHCallUUID)
	})

	session := receiveInboundSession(t, answeredSessions, callSetupTimeout)
	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)

	require.NoError(t, harness.server.EndCallWithReason(session, sip_runtime.LifecycleReasonEndCall))
	waitForTerminalCallState(t, session, callTeardownTimeout)
	require.Eventually(t, func() bool {
		return !harness.freeSWITCHCallExists(freeSWITCHCallUUID)
	}, callTeardownTimeout, 100*time.Millisecond)
}
