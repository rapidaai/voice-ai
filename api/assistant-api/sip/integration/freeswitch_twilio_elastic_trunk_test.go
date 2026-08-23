// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package sip_integration

import (
	"context"
	"testing"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHTwilioElasticTrunkOutbound(t *testing.T) {
	twilioProfile := loadTwilioElasticTrunkConfig(t)
	harness := newFreeSWITCHHarness(t, twilioProfile.sipCredentialConfig)
	harness.sipConfig.CustomHeaders = map[string]string{
		"X-Twilio-AccountSid":           twilioProfile.accountSID,
		"X-Twilio-Elastic-Trunk-SID":    twilioProfile.trunkSID,
		"X-Rapida-Twilio-Trunk-Profile": "elastic-sip-trunk",
	}

	session, err := harness.server.MakeCall(
		context.Background(),
		harness.sipConfig,
		twilioProfile.outboundUser,
		twilioProfile.fromUser,
		sip_infra.MakeCallOptions{},
	)
	require.NoErrorf(t, err, "Twilio-style outbound call to %s failed", freeSWITCHOutboundTargetDescription(harness.config, twilioProfile.outboundUser))
	require.NotNil(t, session)

	waitForCallState(t, session, sip_infra.CallStateConnected, callSetupTimeout)
	require.NoError(t, harness.server.EndCallWithReason(session, sip_infra.LifecycleReasonEndCall))
	waitForTerminalCallState(t, session, callTeardownTimeout)
}

func TestFreeSWITCHTwilioElasticTrunkInbound(t *testing.T) {
	twilioProfile := loadTwilioElasticTrunkConfig(t)
	harness := newFreeSWITCHHarness(t, twilioProfile.sipCredentialConfig)
	answeredSessions := make(chan *sip_infra.Session, 1)
	remoteByeSessions := make(chan *sip_infra.Session, 1)

	harness.server.SetOnInvite(func(session *sip_infra.Session, fromURI, toURI string) error {
		require.Contains(t, fromURI, freeSWITCHUser(twilioProfile.callerUser))
		require.Contains(t, toURI, freeSWITCHUser(twilioProfile.inboundDID))
		answeredSessions <- session
		return nil
	})
	harness.server.SetOnBye(func(session *sip_infra.Session) error {
		remoteByeSessions <- session
		return nil
	})

	freeSWITCHCallUUID := harness.originateTwilioElasticInboundCall(twilioProfile)
	t.Cleanup(func() {
		_, _ = harness.runFreeSWITCHCommand("uuid_kill " + freeSWITCHCallUUID)
	})

	session := receiveInboundSession(t, answeredSessions, callSetupTimeout)
	require.Equal(t, sip_infra.CallDirectionInbound, session.GetInfo().Direction)
	require.Equal(t, sip_infra.InboundSetupPhaseMediaFlowing, session.GetInboundSetupPhase())
	waitForCallState(t, session, sip_infra.CallStateConnected, callSetupTimeout)

	harness.hangupFreeSWITCHCall(freeSWITCHCallUUID)
	remoteByeSession := receiveInboundSession(t, remoteByeSessions, callTeardownTimeout)
	require.Equal(t, session.GetCallID(), remoteByeSession.GetCallID())
	waitForTerminalCallState(t, session, callTeardownTimeout)
}
