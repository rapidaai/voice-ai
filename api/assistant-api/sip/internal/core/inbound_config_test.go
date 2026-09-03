// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"testing"

	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInboundConfigClassifiesMiddlewareErrors(t *testing.T) {
	tests := []struct {
		name              string
		middlewareError   error
		wantClass         inboundFailureClass
		wantResponseClass internal_inbound.FailureClass
		wantResult        CallTerminationResult
		wantReason        string
		wantError         error
	}{
		{
			name:              "invalid configuration",
			middlewareError:   &SIPError{Code: 500, Message: "route resolver unavailable", Err: ErrInvalidConfig},
			wantClass:         inboundFailureConfig,
			wantResponseClass: internal_inbound.FailureConfig,
			wantResult:        CallTerminationServerError,
			wantReason:        "inbound_config",
			wantError:         ErrInvalidConfig,
		},
		{
			name:              "authentication required",
			middlewareError:   &SIPError{Code: 403, Message: "forbidden", Err: ErrAuthRequired},
			wantClass:         inboundFailureAuthRequired,
			wantResponseClass: internal_inbound.FailureAuth,
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

			_, failure := NewInboundConfig(server, inboundInviteIdentity{}, inboundMediaOffer{})

			require.NotNil(t, failure)
			assert.Equal(t, test.wantClass, failure.class)
			assert.Equal(t, test.wantResponseClass, failure.responseClass)
			assert.Equal(t, test.wantResult, failure.termination.Result)
			assert.Equal(t, test.wantReason, failure.termination.Reason)
			assert.ErrorIs(t, failure.err, test.wantError)
		})
	}
}

func TestNewInboundConfigPopulatesLegacyIdentityFields(t *testing.T) {
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

	_, failure := NewInboundConfig(server, identity, inboundMediaOffer{})

	require.NotNil(t, requestContext)
	assert.Equal(t, requestContext.CallAddress.FromURI, requestContext.FromIdentity)
	assert.Equal(t, requestContext.CallAddress.ToURI, requestContext.ToIdentity)
	assert.Equal(t, requestContext.CallAddress.FromURI, requestContext.FromURI)
	assert.Equal(t, requestContext.CallAddress.ToURI, requestContext.ToURI)
	assert.NotNil(t, failure)
}
