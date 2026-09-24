// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"testing"
	"time"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
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

func TestDefaultInboundAnswerPolicyAnswersImmediately(t *testing.T) {
	policy := DefaultInboundAnswerPolicy()

	assert.Equal(t, InboundAnswerModeImmediate, policy.Mode)
	assert.Equal(t, defaultInboundACKTime, policy.ACKTimeout)
}
