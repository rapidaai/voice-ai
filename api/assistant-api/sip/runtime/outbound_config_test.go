// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOutboundConfig() *Config {
	return &Config{
		Server:            "trunk.example.com",
		Port:              5060,
		Transport:         TransportUDP,
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10100,
		Username:          "auth-user",
		Password:          "auth-pass",
	}
}

func TestNewOutboundInviteRequest_TrunkTermination(t *testing.T) {
	request, err := NewOutboundInviteRequest(testOutboundConfig(), "+15551234567", "+15557654321")
	require.NoError(t, err)

	assert.Equal(t, OutboundModeTrunkTermination, request.Config.Mode)
	assert.Equal(t, "trunk.example.com", request.Config.Address)
	assert.Equal(t, "+15551234567", request.Identity.ToUser)
	assert.Equal(t, "+15557654321", request.Identity.FromUser)
	assert.Equal(t, "auth-user", request.Config.Auth.Username)
}

func TestOutboundConfig_MapsLifecycleTimeouts(t *testing.T) {
	cfg := testOutboundConfig()
	cfg.InviteTimeout = 30 * time.Second
	cfg.SessionTimeout = 45 * time.Minute
	cfg.MediaTimeoutInitial = 20 * time.Second
	cfg.MediaTimeout = 10 * time.Second

	outboundConfig := cfg.ToOutboundConfig()

	assert.Equal(t, 30*time.Second, outboundConfig.RingingTimeout)
	assert.Equal(t, 45*time.Minute, outboundConfig.MaxCallDuration)
	assert.Equal(t, 20*time.Second, outboundConfig.MediaTimeoutInitial)
	assert.Equal(t, 10*time.Second, outboundConfig.MediaTimeout)
}

func TestOutboundConfig_EffectiveTimeouts(t *testing.T) {
	assert.Equal(t, defaultOutboundRingingTimeout, OutboundConfig{}.EffectiveRingingTimeout())
	assert.Zero(t, OutboundConfig{}.EffectiveMaxCallDuration())

	outboundConfig := OutboundConfig{
		RingingTimeout:  5 * time.Second,
		MaxCallDuration: 10 * time.Minute,
	}
	assert.Equal(t, 5*time.Second, outboundConfig.EffectiveRingingTimeout())
	assert.Equal(t, 10*time.Minute, outboundConfig.EffectiveMaxCallDuration())
}

func TestOutboundDialogPhase_IsPreAnswer(t *testing.T) {
	assert.True(t, OutboundDialogPhaseInviting.IsPreAnswer())
	assert.True(t, OutboundDialogPhaseProceeding.IsPreAnswer())
	assert.False(t, OutboundDialogPhaseAnswered.IsPreAnswer())
	assert.False(t, OutboundDialogPhaseConfirmed.IsPreAnswer())
	assert.False(t, OutboundDialogPhaseTerminated.IsPreAnswer())
}

func TestNewOutboundInviteRequest_RequiresFromUser(t *testing.T) {
	_, err := NewOutboundInviteRequest(testOutboundConfig(), "+15551234567", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outbound From user is required")
}

func TestNewOutboundInviteRequest_DoesNotFallbackFromUserToAuthUsername(t *testing.T) {
	cfg := testOutboundConfig()

	_, err := NewOutboundInviteRequest(cfg, "+15551234567", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outbound From user is required")
}

func TestNewOutboundInviteRequest_AllowsEmptyAuth(t *testing.T) {
	cfg := testOutboundConfig()
	cfg.Username = ""
	cfg.Password = ""

	request, err := NewOutboundInviteRequest(cfg, "+15551234567", "+15557654321")
	require.NoError(t, err)
	assert.Empty(t, request.Config.Auth.Username)
	assert.Empty(t, request.Config.Auth.Password)
}

func TestOutboundAuthMissingForChallenge(t *testing.T) {
	assert.False(t, outboundAuthMissingForChallenge(SIPAuthConfig{}, 180))
	assert.True(t, outboundAuthMissingForChallenge(SIPAuthConfig{}, 401))
	assert.True(t, outboundAuthMissingForChallenge(SIPAuthConfig{Username: "u"}, 407))
	assert.False(t, outboundAuthMissingForChallenge(SIPAuthConfig{Username: "u", Password: "p"}, 407))
}

func TestTransferBridgeCallOptions_DoesNotClaimParentCallContext(t *testing.T) {
	options := TransferBridgeCallOptions{
		ConversationID: 1001,
		ContextID:      "parent-context-id",
	}

	makeCallOptions := options.makeCallOptions()

	assert.Zero(t, makeCallOptions.ConversationID)
	assert.Empty(t, makeCallOptions.ContextID)
}
