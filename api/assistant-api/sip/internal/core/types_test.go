// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rapidaai/protos"
)

func makeVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	s, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: s}
}

func TestParseConfigFromVault_NilCredential(t *testing.T) {
	_, err := ParseConfigFromVault(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vault credential is required")
}

func TestParseConfigFromVault_NilValue(t *testing.T) {
	_, err := ParseConfigFromVault(&protos.VaultCredential{})
	require.Error(t, err)
}

func TestParseConfigFromVault_BasicFields(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_username": "user1",
		"sip_password": "pass1",
		"sip_server":   "pbx.example.com",
		"sip_realm":    "example.com",
		"sip_domain":   "sip.example.com",
	}))
	require.NoError(t, err)
	assert.Equal(t, "user1", cfg.Username)
	assert.Equal(t, "pass1", cfg.Password)
	assert.Equal(t, "pbx.example.com", cfg.Server)
	assert.Equal(t, "example.com", cfg.Realm)
	assert.Equal(t, "sip.example.com", cfg.Domain)
}

func TestParseConfigFromVault_CurrentVaultAliases(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"host":     "trunk.example.com:5070",
		"user":     "auth-user",
		"password": "auth-pass",
		"headers": map[string]interface{}{
			"X-Custom": "value",
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, "trunk.example.com", cfg.Server)
	assert.Equal(t, 5070, cfg.Port)
	assert.Equal(t, "auth-user", cfg.Username)
	assert.Equal(t, "auth-pass", cfg.Password)
	assert.Equal(t, map[string]string{"X-Custom": "value"}, cfg.CustomHeaders)
}

func TestConfigValidate_AllowsEmptyAuth(t *testing.T) {
	cfg := &Config{
		Server:            "trunk.example.com",
		Port:              5060,
		Transport:         TransportUDP,
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10100,
	}
	require.NoError(t, cfg.Validate())
}

func TestConfigValidate_MinRingAnswerModeRequiresDuration(t *testing.T) {
	cfg := &Config{
		Server:            "trunk.example.com",
		Port:              5060,
		Transport:         TransportUDP,
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10100,
		InboundAnswerMode: InboundAnswerModeAfterMinRingDuration,
	}

	err := cfg.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "min_ring_duration is required for answer_after_min_ring_ms")

	cfg.InboundMinRingDuration = time.Second
	require.NoError(t, cfg.Validate())
}

func TestParseConfigFromVault_SIPURIWithPort(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_uri":      "sip:192.168.1.5:5060",
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.5", cfg.Server)
	assert.Equal(t, 5060, cfg.Port)
}

func TestParseConfigFromVault_SIPURIWithoutPort(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_uri":      "sip:pstn.twilio.com",
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	assert.Equal(t, "pstn.twilio.com", cfg.Server)
	assert.Equal(t, 0, cfg.Port)
}

func TestParseConfigFromVault_SIPSScheme(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_uri":      "sips:secure.example.com:5061",
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	assert.Equal(t, "secure.example.com", cfg.Server)
	assert.Equal(t, 5061, cfg.Port)
}

func TestParseConfigFromVault_ServerOverridesSIPURI(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_uri":      "sip:old.host.com:5060",
		"sip_server":   "new.host.com",
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	assert.Equal(t, "new.host.com", cfg.Server)
	// Port from URI is preserved
	assert.Equal(t, 5060, cfg.Port)
}

func TestParseConfigFromVault_ExplicitPort(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_server":   "host.com",
		"sip_port":     float64(5080),
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	assert.Equal(t, 5080, cfg.Port)
}

func TestParseConfigFromVault_PortStringFormat(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_server":   "host.com",
		"sip_port":     "5080",
		"sip_username": "u",
		"sip_password": "p",
	}))
	require.NoError(t, err)
	// structpb converts numbers to float64, but string port is tested via parsePortValue
	assert.Equal(t, 5080, cfg.Port)
}

func TestParseConfigFromVault_CallerID(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_server":    "host.com",
		"sip_username":  "u",
		"sip_password":  "p",
		"sip_caller_id": "+15551234567",
	}))
	require.NoError(t, err)
	assert.Equal(t, "+15551234567", cfg.CallerID)
}

func TestParseConfigFromVault_CustomHeaders(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_server":   "host.com",
		"sip_username": "u",
		"sip_password": "p",
		"sip_headers":  `{"X-Custom":"foo","X-Other":"bar"}`,
	}))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"X-Custom": "foo",
		"X-Other":  "bar",
	}, cfg.CustomHeaders)
}

func TestParseConfigFromVault_InvalidHeadersIgnored(t *testing.T) {
	cfg, err := ParseConfigFromVault(makeVaultCredential(map[string]interface{}{
		"sip_server":   "host.com",
		"sip_username": "u",
		"sip_password": "p",
		"sip_headers":  "not-valid-json",
	}))
	require.NoError(t, err)
	assert.Nil(t, cfg.CustomHeaders)
}

func TestExtractDIDFromURI_PhoneNumber(t *testing.T) {
	did := ExtractDIDFromURI("sip:15551234567@pstn.twilio.com")
	assert.Equal(t, "+15551234567", did)
}

func TestExtractDIDFromURI_AlreadyE164(t *testing.T) {
	did := ExtractDIDFromURI("sip:+15551234567@pstn.twilio.com")
	assert.Equal(t, "+15551234567", did)
}

func TestExtractDIDFromURI_SkipsCredentialPair(t *testing.T) {
	did := ExtractDIDFromURI("sip:12345:apikey@host.com")
	assert.Equal(t, "", did)
}

func TestExtractDIDFromURI_Empty(t *testing.T) {
	did := ExtractDIDFromURI("")
	assert.Equal(t, "", did)
}

func TestExtractDIDFromURI_ShortUser(t *testing.T) {
	// Users with 5 or fewer chars don't get "+" prefix
	did := ExtractDIDFromURI("sip:ext42@pbx.local")
	assert.Equal(t, "ext42", did)
}

func TestApplyOperationalDefaults_FillsUnset(t *testing.T) {
	cfg := &Config{Server: "host.com"}
	cfg.ApplyOperationalDefaults(5060, TransportTCP, 10000, 20000)
	assert.Equal(t, 5060, cfg.Port)
	assert.Equal(t, TransportTCP, cfg.Transport)
	assert.Equal(t, 10000, cfg.RTPPortRangeStart)
	assert.Equal(t, 20000, cfg.RTPPortRangeEnd)
}

func TestApplyOperationalDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		Server:            "host.com",
		Port:              5080,
		Transport:         TransportUDP,
		RTPPortRangeStart: 12000,
		RTPPortRangeEnd:   14000,
	}
	cfg.ApplyOperationalDefaults(5060, TransportTCP, 10000, 20000)
	assert.Equal(t, 5080, cfg.Port)
	assert.Equal(t, TransportUDP, cfg.Transport)
	assert.Equal(t, 12000, cfg.RTPPortRangeStart)
	assert.Equal(t, 14000, cfg.RTPPortRangeEnd)
}

func TestApplyTimeoutDefaults_FillsUnset(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyTimeoutDefaults(5*time.Second, 30*time.Second, 45*time.Minute)

	assert.Equal(t, 5*time.Second, cfg.RegisterTimeout)
	assert.Equal(t, 30*time.Second, cfg.InviteTimeout)
	assert.Equal(t, 45*time.Minute, cfg.SessionTimeout)
	assert.Equal(t, 5*time.Second, cfg.EffectiveRegisterTimeout())
}

func TestApplyTimeoutDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		RegisterTimeout: 3 * time.Second,
		InviteTimeout:   10 * time.Second,
		SessionTimeout:  20 * time.Minute,
	}
	cfg.ApplyTimeoutDefaults(5*time.Second, 30*time.Second, 45*time.Minute)

	assert.Equal(t, 3*time.Second, cfg.RegisterTimeout)
	assert.Equal(t, 10*time.Second, cfg.InviteTimeout)
	assert.Equal(t, 20*time.Minute, cfg.SessionTimeout)
}

func TestApplyMediaTimeoutDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyMediaTimeoutDefaults(30*time.Second, 15*time.Second)

	assert.Equal(t, 30*time.Second, cfg.MediaTimeoutInitial)
	assert.Equal(t, 15*time.Second, cfg.MediaTimeout)
}

func TestApplyMediaTimeoutDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		MediaTimeoutInitial: 10 * time.Second,
		MediaTimeout:        5 * time.Second,
	}
	cfg.ApplyMediaTimeoutDefaults(30*time.Second, 15*time.Second)

	assert.Equal(t, 10*time.Second, cfg.MediaTimeoutInitial)
	assert.Equal(t, 5*time.Second, cfg.MediaTimeout)
}

func TestApplyInboundAnswerDefaults(t *testing.T) {
	cfg := &Config{}

	cfg.ApplyInboundAnswerDefaults(
		InboundAnswerModeAfterMinRingDuration,
		50*time.Millisecond,
		5*time.Second,
		2*time.Second,
	)

	assert.Equal(t, InboundAnswerModeAfterMinRingDuration, cfg.InboundAnswerMode)
	assert.Equal(t, 50*time.Millisecond, cfg.InboundMinRingDuration)
	assert.Equal(t, 5*time.Second, cfg.InboundMaxRingDuration)
	assert.Equal(t, 2*time.Second, cfg.InboundACKTimeout)
}

func TestApplyInboundAnswerDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		InboundAnswerMode:      InboundAnswerModeImmediate,
		InboundMinRingDuration: 10 * time.Millisecond,
		InboundMaxRingDuration: time.Second,
		InboundACKTimeout:      500 * time.Millisecond,
	}

	cfg.ApplyInboundAnswerDefaults(
		InboundAnswerModeAfterMinRingDuration,
		50*time.Millisecond,
		5*time.Second,
		2*time.Second,
	)

	assert.Equal(t, InboundAnswerModeImmediate, cfg.InboundAnswerMode)
	assert.Equal(t, 10*time.Millisecond, cfg.InboundMinRingDuration)
	assert.Equal(t, time.Second, cfg.InboundMaxRingDuration)
	assert.Equal(t, 500*time.Millisecond, cfg.InboundACKTimeout)
}

func TestDefaultInboundAnswerPolicyAnswersImmediately(t *testing.T) {
	policy := DefaultInboundAnswerPolicy()

	assert.Equal(t, InboundAnswerModeImmediate, policy.Mode)
	assert.Equal(t, defaultInboundACKTimeout, policy.ACKTimeout)
}
