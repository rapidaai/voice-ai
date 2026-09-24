// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package sip_runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSIPRequestContextResolveRoute(t *testing.T) {
	tests := []struct {
		name       string
		requestURI string
		expected   CallRoute
		expectsErr bool
	}{
		{name: "agent", requestURI: "sip:agent-42@sip.rapida.ai", expected: AgentCallRoute{AssistantID: 42}},
		{name: "agent parameters", requestURI: "sip:agent-42;transport=tcp@sip.rapida.ai", expected: AgentCallRoute{AssistantID: 42}},
		{name: "prefixed DID", requestURI: "sips:did-+15551234567@sip.rapida.ai", expected: DIDCallRoute{DID: "+15551234567"}},
		{name: "plain DID", requestURI: "sip:+15551234568@sip.rapida.ai", expected: DIDCallRoute{DID: "+15551234568"}},
		{name: "invalid assistant", requestURI: "sip:agent-invalid@sip.rapida.ai", expectsErr: true},
		{name: "zero assistant", requestURI: "sip:agent-0@sip.rapida.ai", expectsErr: true},
		{name: "invalid DID", requestURI: "sip:did-+1-555@sip.rapida.ai", expectsErr: true},
		{name: "empty", requestURI: "", expectsErr: true},
		{name: "credential pair", requestURI: "sip:12345:apikey@sip.rapida.ai", expectsErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &SIPRequestContext{RequestURI: test.requestURI}
			actual, err := ctx.ResolveRoute()
			if test.expectsErr {
				require.ErrorIs(t, err, ErrInvalidCallRoute)
				assert.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestSIPRequestContextResolveRouteNilContext(t *testing.T) {
	var ctx *SIPRequestContext
	route, err := ctx.ResolveRoute()
	require.ErrorIs(t, err, ErrInvalidCallRoute)
	assert.Nil(t, route)
}

func TestRTPConfigValidateRejectsNil(t *testing.T) {
	var config *RTPConfig

	assert.ErrorIs(t, config.Validate(), errRTPConfigRequired)
}

func TestRTPConfigValidateReturnsSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  *RTPConfig
		wantErr error
	}{
		{
			name:    "missing local ip",
			config:  &RTPConfig{},
			wantErr: errRTPLocalAddressIPRequired,
		},
		{
			name: "blank local ip",
			config: &RTPConfig{
				LocalAddress: RTPAddress{IP: " "},
			},
			wantErr: errRTPLocalAddressIPRequired,
		},
		{
			name: "invalid local port",
			config: &RTPConfig{
				LocalAddress: RTPAddress{IP: "127.0.0.1", Port: rtpMaxPort + 1},
			},
			wantErr: errRTPInvalidLocalAddressPort,
		},
		{
			name: "missing port range",
			config: &RTPConfig{
				LocalAddress: RTPAddress{IP: "127.0.0.1"},
			},
			wantErr: errRTPPortRangeRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.ErrorIs(t, test.config.Validate(), test.wantErr)
		})
	}
}

func TestNilSDPMediaInfoIsNotHold(t *testing.T) {
	var mediaInfo *SDPMediaInfo

	if mediaInfo.IsHold() {
		t.Fatal("nil SDP media info should not report hold")
	}
}
