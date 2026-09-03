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

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

type sipCredentialConfig struct {
	username string
	password string
	realm    string
	domain   string
	fromUser string
}

type outboundTrunkConfig struct {
	sipCredentialConfig
	answerUser       string
	headerAssertUser string
	ringOnlyUser     string
}

type outboundFailureEndpointConfig struct {
	sipCredentialConfig
	busyUser        string
	rejectedUser    string
	noAnswerUser    string
	unavailableUser string
	mediaRejectUser string
}

type registrationInboundConfig struct {
	sipCredentialConfig
	registeredDID string
	callerUser    string
}

type twilioElasticTrunkConfig struct {
	sipCredentialConfig
	outboundUser string
	inboundDID   string
	callerUser   string
	accountSID   string
	trunkSID     string
	callSID      string
}

func loadOutboundTrunkConfig(t *testing.T) outboundTrunkConfig {
	t.Helper()
	return outboundTrunkConfig{
		sipCredentialConfig: loadSIPCredentialConfig(t),
		answerUser:          requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_ANSWER_USER"),
		headerAssertUser:    integrationEnv("FREESWITCH_OUTBOUND_HEADER_ASSERT_USER", ""),
		ringOnlyUser:        integrationEnv("FREESWITCH_OUTBOUND_RING_ONLY_USER", ""),
	}
}

func loadOutboundFailureEndpointConfig(t *testing.T) outboundFailureEndpointConfig {
	t.Helper()
	return outboundFailureEndpointConfig{
		sipCredentialConfig: loadSIPCredentialConfig(t),
		busyUser:            requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_BUSY_USER"),
		rejectedUser:        requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_REJECTED_USER"),
		noAnswerUser:        requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_NO_ANSWER_USER"),
		unavailableUser:     requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_UNAVAILABLE_USER"),
		mediaRejectUser:     requiredIntegrationEnv(t, "FREESWITCH_OUTBOUND_MEDIA_REJECT_USER"),
	}
}

func loadRegistrationInboundConfig(t *testing.T) registrationInboundConfig {
	t.Helper()
	return registrationInboundConfig{
		sipCredentialConfig: loadSIPCredentialConfig(t),
		registeredDID:       requiredIntegrationEnv(t, "FREESWITCH_REGISTER_DID"),
		callerUser:          requiredIntegrationEnv(t, "FREESWITCH_INBOUND_CALLER_USER"),
	}
}

func loadTwilioElasticTrunkConfig(t *testing.T) twilioElasticTrunkConfig {
	t.Helper()
	return twilioElasticTrunkConfig{
		sipCredentialConfig: loadSIPCredentialConfig(t),
		outboundUser:        requiredIntegrationEnv(t, "FREESWITCH_TWILIO_OUTBOUND_USER"),
		inboundDID:          requiredIntegrationEnv(t, "FREESWITCH_TWILIO_INBOUND_DID"),
		callerUser:          integrationEnv("FREESWITCH_TWILIO_CALLER_USER", "+15551239999"),
		accountSID:          integrationEnv("FREESWITCH_TWILIO_ACCOUNT_SID", "ACrapidafreeswitchintegration"),
		trunkSID:            integrationEnv("FREESWITCH_TWILIO_TRUNK_SID", "TKrapidafreeswitchintegration"),
		callSID:             integrationEnv("FREESWITCH_TWILIO_CALL_SID", "CArapidafreeswitchintegration"),
	}
}

func loadSIPCredentialConfig(t *testing.T) sipCredentialConfig {
	t.Helper()
	username := requiredIntegrationEnv(t, "FREESWITCH_SIP_USERNAME")
	return sipCredentialConfig{
		username: username,
		password: requiredIntegrationEnv(t, "FREESWITCH_SIP_PASSWORD"),
		realm:    integrationEnv("FREESWITCH_SIP_REALM", ""),
		domain:   integrationEnv("FREESWITCH_SIP_DOMAIN", ""),
		fromUser: integrationEnv("RAPIDA_SIP_FROM_USER", username),
	}
}

func registerFreeSWITCHInboundDID(
	t *testing.T,
	registrationClient *sip_runtime.RegistrationClient,
	inboundConfig registrationInboundConfig,
	sipConfig *sip_runtime.Config,
) {
	t.Helper()
	registerContext, cancelRegister := context.WithTimeout(context.Background(), callSetupTimeout)
	defer cancelRegister()
	require.NoError(t, registrationClient.Register(registerContext, &sip_runtime.Registration{
		DID:         inboundConfig.registeredDID,
		Config:      sipConfig,
		AssistantID: 1001,
		ExpiresIn:   120,
	}))
	t.Cleanup(func() {
		unregisterContext, cancelUnregister := context.WithTimeout(context.Background(), callTeardownTimeout)
		defer cancelUnregister()
		require.NoError(t, registrationClient.Unregister(unregisterContext, inboundConfig.registeredDID))
	})
}
