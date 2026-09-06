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

func TestInboundResolveConfigClassifiesMiddlewareErrors(t *testing.T) {
	tests := []struct {
		name              string
		middlewareError   error
		wantClass         inboundFailureClass
		wantResponseClass inboundFailureClass
		wantResult        CallTerminationResult
		wantReason        string
		wantError         error
	}{
		{
			name:              "invalid configuration",
			middlewareError:   &SIPError{Code: 500, Message: "route resolver unavailable", Err: ErrInvalidConfig},
			wantClass:         inboundFailureConfig,
			wantResponseClass: inboundFailureConfig,
			wantResult:        CallTerminationServerError,
			wantReason:        "inbound_config",
			wantError:         ErrInvalidConfig,
		},
		{
			name:              "authentication required",
			middlewareError:   &SIPError{Code: 403, Message: "forbidden", Err: ErrAuthRequired},
			wantClass:         inboundFailureAuthRequired,
			wantResponseClass: inboundFailureAuth,
			wantResult:        CallTerminationClientError,
			wantReason:        "inbound_auth_required",
			wantError:         ErrAuthRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := &Server{middlewares: []Middleware{func(*SIPRequestContext) error {
				return test.middlewareError
			}}}

			inboundCall := &Inbound{server: server}
			failure := inboundCall.resolveConfig()

			require.NotNil(t, failure)
			assert.Equal(t, test.wantClass, failure.class)
			assert.Equal(t, test.wantResponseClass, failure.responseClass)
			assert.Equal(t, test.wantResult, failure.termination.Result)
			assert.Equal(t, test.wantReason, failure.termination.Reason)
			assert.ErrorIs(t, failure.err, test.wantError)
		})
	}
}

func TestInboundResolveConfigPopulatesCallAddress(t *testing.T) {
	var requestContext *SIPRequestContext
	server := &Server{middlewares: []Middleware{func(ctx *SIPRequestContext) error {
		requestContext = ctx
		return nil
	}}}
	identity := inboundInviteIdentity{
		callAddress: CallAddress{
			FromURI: "sip:+14155550100@carrier.example.com",
			ToURI:   "sip:agent-42@sip.rapida.ai",
		},
	}

	inboundCall := &Inbound{server: server, identity: identity}
	failure := inboundCall.resolveConfig()

	require.NotNil(t, requestContext)
	assert.Equal(t, identity.callAddress, requestContext.CallAddress)
	assert.NotNil(t, failure)
}

func TestInboundResolveConfigProtectsCapturedCallAddress(t *testing.T) {
	server := &Server{middlewares: []Middleware{func(ctx *SIPRequestContext) error {
		ctx.CallAddress.From = "+14155550999"
		ctx.CallAddress.To = "+14155550300"
		ctx.CallAddress.FromURI = "sip:attacker@example.com"
		ctx.CallAddress.ToURI = "sip:attacker@example.net"
		ctx.CallAddress.Headers["x-original-called-number"] = "+14155550999"
		ctx.Config = &Config{Server: "trunk.example.com"}
		return nil
	}}}
	identity := inboundInviteIdentity{
		callAddress: CallAddress{
			From:    "+14155550100",
			FromURI: "sip:+14155550100@carrier.example.com",
			ToURI:   "sip:agent-42@sip.rapida.ai",
			Headers: map[string]string{"x-original-called-number": "+14155550200"},
		},
	}

	inboundCall := &Inbound{server: server, identity: identity}
	failure := inboundCall.resolveConfig()

	require.Nil(t, failure)
	assert.Equal(t, "+14155550100", inboundCall.resolvedConfig.callAddress.From)
	assert.Equal(t, "+14155550300", inboundCall.resolvedConfig.callAddress.To)
	assert.Equal(t, "sip:+14155550100@carrier.example.com", inboundCall.resolvedConfig.callAddress.FromURI)
	assert.Equal(t, "sip:agent-42@sip.rapida.ai", inboundCall.resolvedConfig.callAddress.ToURI)
	assert.Equal(t, "+14155550200", inboundCall.resolvedConfig.callAddress.Headers["x-original-called-number"])
	assert.Equal(t, inboundCall.resolvedConfig.callAddress, inboundCall.identity.callAddress)
	assert.Equal(t, "+14155550200", identity.callAddress.Headers["x-original-called-number"])
}
