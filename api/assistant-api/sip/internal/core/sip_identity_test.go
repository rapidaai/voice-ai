// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import "testing"

// The registration identity (AOR), the Digest auth identity, and the outbound
// caller ID are three different things. Mixing them is what made a
// username-keyed registrar answer REGISTER with 404, so each resolver is
// pinned here.

func TestGetAuthUsernameFallsBackToUsername(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		authUsername string
		want         string
	}{
		{name: "falls back to sip username", username: "sip7f2a91", want: "sip7f2a91"},
		{name: "prefers explicit auth username", username: "sip7f2a91", authUsername: "auth-user", want: "auth-user"},
		{name: "ignores blank auth username", username: "sip7f2a91", authUsername: "   ", want: "sip7f2a91"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Username: tt.username, AuthUsername: tt.authUsername}
			if got := cfg.GetAuthUsername(); got != tt.want {
				t.Fatalf("GetAuthUsername() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetAORUserFallsBackToUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		aorUser  string
		want     string
	}{
		{name: "falls back to sip username", username: "sip7f2a91", want: "sip7f2a91"},
		{name: "prefers explicit aor user", username: "sip7f2a91", aorUser: "1044", want: "1044"},
		{name: "ignores blank aor user", username: "sip7f2a91", aorUser: "  ", want: "sip7f2a91"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Username: tt.username, AORUser: tt.aorUser}
			if got := cfg.GetAORUser(); got != tt.want {
				t.Fatalf("GetAORUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A caller ID is outbound presentation only. It must never leak into the AOR,
// otherwise REGISTER goes out as the phone number instead of the account.
func TestRegistrationAORUserIgnoresCallerID(t *testing.T) {
	cfg := &Config{Username: "sip7f2a91", CallerID: "+15551234567"}
	reg := &Registration{DID: "+15551234567"}

	if got := registrationAORUser(cfg, reg); got != "sip7f2a91" {
		t.Fatalf("registrationAORUser() = %q, want the SIP username", got)
	}
}

// Credentials saved before the split have no username; those deployments must
// keep registering off the DID rather than breaking on upgrade.
func TestRegistrationAORUserLegacyDIDFallback(t *testing.T) {
	cfg := &Config{}
	reg := &Registration{DID: "+15551234567"}

	if got := registrationAORUser(cfg, reg); got != "15551234567" {
		t.Fatalf("registrationAORUser() = %q, want the normalized DID", got)
	}
}

// Alphanumeric account names are not phone numbers and must survive verbatim.
func TestRegistrationAORUserDoesNotNormalizeAlphanumeric(t *testing.T) {
	cfg := &Config{Username: "+abc-user"}

	if got := registrationAORUser(cfg, &Registration{DID: "+15551234567"}); got != "+abc-user" {
		t.Fatalf("registrationAORUser() = %q, want the username unchanged", got)
	}
}

func TestGetOutboundTargetPrefersProxy(t *testing.T) {
	cfg := &Config{Server: "at1.provider.com"}
	if got := cfg.GetOutboundTarget(); got != "at1.provider.com" {
		t.Fatalf("GetOutboundTarget() = %q, want the server", got)
	}

	cfg.OutboundProxy = "sbc.provider.com"
	if got := cfg.GetOutboundTarget(); got != "sbc.provider.com" {
		t.Fatalf("GetOutboundTarget() = %q, want the outbound proxy", got)
	}
}

// Outbound auth uses the Digest identity, and routing honours the proxy.
func TestToOutboundConfigUsesAuthIdentityAndProxy(t *testing.T) {
	cfg := &Config{
		Server:        "at1.provider.com",
		Username:      "sip7f2a91",
		AuthUsername:  "auth-user",
		OutboundProxy: "sbc.provider.com",
		Port:          5060,
		Transport:     TransportUDP,
	}

	outbound := cfg.ToOutboundConfig()
	if outbound.Auth.Username != "auth-user" {
		t.Fatalf("outbound auth username = %q, want the Digest identity", outbound.Auth.Username)
	}
	if outbound.Address != "sbc.provider.com" {
		t.Fatalf("outbound address = %q, want the outbound proxy", outbound.Address)
	}
}
