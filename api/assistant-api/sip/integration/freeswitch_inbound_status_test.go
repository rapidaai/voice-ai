// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package sip_integration

import (
	"testing"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHRegistrationBasedInboundCall(t *testing.T) {
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
	require.Equal(t, sip_runtime.CallDirectionInbound, session.GetInfo().Direction)
	require.Equal(t, sip_runtime.InboundSetupPhaseMediaFlowing, session.GetInboundSetupPhase())
	waitForCallState(t, session, sip_runtime.CallStateConnected, callSetupTimeout)

	harness.hangupFreeSWITCHCall(freeSWITCHCallUUID)
	remoteByeSession := receiveInboundSession(t, remoteByeSessions, callTeardownTimeout)
	require.Equal(t, session.GetCallID(), remoteByeSession.GetCallID())
	waitForTerminalCallState(t, session, callTeardownTimeout)
}
