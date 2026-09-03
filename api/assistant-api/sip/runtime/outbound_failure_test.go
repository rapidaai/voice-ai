// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
)

func TestOutboundFailureError(t *testing.T) {
	cause := fmt.Errorf("%w: unsupported answer payload", ErrCodecNotSupported)
	failure := OutboundFailure{
		Class:           OutboundFailureMedia,
		Reason:          cause.Error(),
		Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_answer_sdp_failed"},
		LifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
		Err:             cause,
	}

	assert.Equal(t, "codec not supported: unsupported answer payload", failure.Error())
	assert.ErrorIs(t, failure, ErrCodecNotSupported)
}

func TestNewOutboundSetupFailure(t *testing.T) {
	cause := errors.New("failed to create RTP handler")

	failure := NewOutboundSetupFailure(cause)

	assert.Equal(t, OutboundFailureSetup, failure.Class)
	assert.Equal(t, "failed to create RTP handler", failure.Reason)
	assert.Equal(t, CallTerminationServerError, failure.Termination.Result)
	assert.Equal(t, "outbound_setup_failed", failure.Termination.Reason)
	assert.Equal(t, LifecycleReasonOutboundSetupFailure, failure.LifecycleReason)
	assert.ErrorIs(t, failure, cause)
}

func TestOutboundFailure_StatusUpdateAndRecord(t *testing.T) {
	failure := OutboundFailure{
		StatusCode:      sip.StatusBusyHere,
		Class:           OutboundFailureBusy,
		Reason:          "Busy Here",
		Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_busy"},
		Retryable:       true,
		LifecycleReason: LifecycleReasonOutboundRejected,
		Err:             errors.New("busy"),
	}

	update := failure.StatusUpdate("call-1")

	assert.Equal(t, "call-1", update.ChannelUUID)
	assert.Equal(t, string(OutboundCallStatusFailed), update.CallStatus)
	assert.Equal(t, "busy", update.ErrorMessage)
	assert.Equal(t, string(OutboundFailureBusy), update.FailureClass)
	assert.Equal(t, "Busy Here", update.FailureReason)
	assert.Equal(t, string(CallTerminationClientError), update.SLIResult)
	assert.Equal(t, "outbound_busy", update.SLIReason)
	assert.Equal(t, LifecycleReasonOutboundRejected.String(), update.DisconnectReason)
	assert.True(t, update.Retryable)
	assert.Equal(t, sip.StatusBusyHere, update.ProviderStatusCode)

	session := &Session{}
	failure.Record(session)

	assertSessionMetadata(t, session, "sip.failure_class", string(OutboundFailureBusy))
	assertSessionMetadata(t, session, "sip.failure_reason", "Busy Here")
	assertSessionMetadata(t, session, "sip.sli_result", string(CallTerminationClientError))
	assertSessionMetadata(t, session, "sip.sli_reason", "outbound_busy")
	assertSessionMetadata(t, session, "sip.failure_retryable", true)
	assertSessionMetadata(t, session, "sip.failure_status_code", sip.StatusBusyHere)
}

func TestNewOutboundInviteFailure_MapsPreAnswerCancel(t *testing.T) {
	failure := NewOutboundInviteFailure(context.Canceled)

	assert.Equal(t, OutboundFailureCancelled, failure.Class)
	assert.Equal(t, "cancelled", failure.Reason)
	assert.Equal(t, CallTerminationClientError, failure.Termination.Result)
	assert.Equal(t, "outbound_cancelled", failure.Termination.Reason)
	assert.Equal(t, LifecycleReasonOutboundCancelledBeforeAnswer, failure.LifecycleReason)
}

func TestNewOutboundInviteFailure_MapsRingingTimeout(t *testing.T) {
	failure := NewOutboundInviteFailure(context.DeadlineExceeded)

	assert.Equal(t, OutboundFailureNoAnswer, failure.Class)
	assert.Equal(t, "ringing timeout", failure.Reason)
	assert.Equal(t, CallTerminationClientError, failure.Termination.Result)
	assert.Equal(t, "outbound_no_answer", failure.Termination.Reason)
	assert.Equal(t, LifecycleReasonOutboundNoAnswer, failure.LifecycleReason)
	assert.True(t, failure.Retryable)
}

func TestNewOutboundInviteFailure_MapsAuthRequired(t *testing.T) {
	failure := NewOutboundInviteFailure(fmt.Errorf("challenge rejected: %w", ErrAuthRequired))

	assert.Equal(t, OutboundFailureAuthRequired, failure.Class)
	assert.Equal(t, "auth credentials missing", failure.Reason)
	assert.Equal(t, CallTerminationClientError, failure.Termination.Result)
	assert.Equal(t, "outbound_auth_required", failure.Termination.Reason)
	assert.Equal(t, LifecycleReasonOutboundAuthFailed, failure.LifecycleReason)
}

func TestNewOutboundInviteFailure_MapsSIPStatus(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		reason          string
		class           OutboundFailureClass
		termination     CallTermination
		lifecycleReason LifecycleReason
		retryable       bool
	}{
		{
			name:            "forbidden",
			statusCode:      sip.StatusForbidden,
			reason:          "Forbidden",
			class:           OutboundFailureForbidden,
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_forbidden"},
			lifecycleReason: LifecycleReasonOutboundRejected,
		},
		{
			name:            "request timeout",
			statusCode:      sip.StatusRequestTimeout,
			reason:          "Request Timeout",
			class:           OutboundFailureNoAnswer,
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_request_timeout"},
			lifecycleReason: LifecycleReasonOutboundNoAnswer,
			retryable:       true,
		},
		{
			name:            "busy",
			statusCode:      sip.StatusBusyHere,
			reason:          "Busy Here",
			class:           OutboundFailureBusy,
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_busy"},
			lifecycleReason: LifecycleReasonOutboundRejected,
		},
		{
			name:            "media rejected",
			statusCode:      sip.StatusNotAcceptableHere,
			reason:          "Not Acceptable Here",
			class:           OutboundFailureMedia,
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_not_acceptable"},
			lifecycleReason: LifecycleReasonOutboundMediaRejected,
		},
		{
			name:            "upstream unavailable",
			statusCode:      sip.StatusServiceUnavailable,
			reason:          "Service Unavailable",
			class:           OutboundFailureUpstreamFailure,
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_upstream_server_error"},
			lifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			retryable:       true,
		},
		{
			name:            "generic global decline",
			statusCode:      606,
			reason:          "Not Acceptable",
			class:           OutboundFailureRejected,
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_global_decline"},
			lifecycleReason: LifecycleReasonOutboundRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &sipgo.ErrDialogResponse{Res: sip.NewResponse(tt.statusCode, tt.reason)}

			failure := NewOutboundInviteFailure(err)

			assert.Equal(t, tt.statusCode, failure.StatusCode)
			assert.Equal(t, tt.class, failure.Class)
			assert.Equal(t, tt.reason, failure.Reason)
			assert.Equal(t, tt.termination, failure.Termination)
			assert.Equal(t, tt.lifecycleReason, failure.LifecycleReason)
			assert.Equal(t, tt.retryable, failure.Retryable)
			assert.ErrorIs(t, failure, err)
		})
	}
}

func TestNewOutboundInviteFailure_MapsProviderCapacity5xx(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		reason      string
		body        string
		header      string
		failureText string
		sliReason   string
	}{
		{
			name:        "cps limit in body",
			statusCode:  sip.StatusServiceUnavailable,
			reason:      "Service Unavailable",
			body:        "CPS limit exceeded",
			failureText: "CPS limit exceeded",
			sliReason:   "outbound_cps_limit_exceeded",
		},
		{
			name:        "concurrent call limit in body",
			statusCode:  sip.StatusInternalServerError,
			reason:      "Internal Server Error",
			body:        "Concurrent call limit exceeded",
			failureText: "Concurrent call limit exceeded",
			sliReason:   "outbound_concurrent_limit_exceeded",
		},
		{
			name:        "cps limit in provider header",
			statusCode:  sip.StatusServiceUnavailable,
			reason:      "Service Unavailable",
			header:      "CPS limit exceeded",
			failureText: "CPS limit exceeded",
			sliReason:   "outbound_cps_limit_exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := sip.NewResponse(tt.statusCode, tt.reason)
			if tt.body != "" {
				response.SetBody([]byte(tt.body))
			}
			if tt.header != "" {
				response.AppendHeader(sip.NewHeader("X-Twilio-Error", tt.header))
			}
			err := &sipgo.ErrDialogResponse{Res: response}

			failure := NewOutboundInviteFailure(err)
			update := failure.StatusUpdate("call-1")

			assert.Equal(t, tt.statusCode, failure.StatusCode)
			assert.Equal(t, OutboundFailureTrunkCapacity, failure.Class)
			assert.Equal(t, tt.failureText, failure.Reason)
			assert.Equal(t, CallTerminationClientError, failure.Termination.Result)
			assert.Equal(t, tt.sliReason, failure.Termination.Reason)
			assert.Equal(t, LifecycleReasonOutboundTrunkCapacity, failure.LifecycleReason)
			assert.True(t, failure.Retryable)
			assert.ErrorIs(t, failure, err)

			assert.Equal(t, string(OutboundFailureTrunkCapacity), update.FailureClass)
			assert.Equal(t, tt.failureText, update.FailureReason)
			assert.Equal(t, string(CallTerminationClientError), update.SLIResult)
			assert.Equal(t, tt.sliReason, update.SLIReason)
			assert.Equal(t, LifecycleReasonOutboundTrunkCapacity.String(), update.DisconnectReason)
			assert.True(t, update.Retryable)
			assert.Equal(t, tt.statusCode, update.ProviderStatusCode)
		})
	}
}

func TestNewOutboundInviteFailure_MapsNetworkFailure(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		reason      string
		termination CallTermination
		retryable   bool
	}{
		{
			name:        "dns",
			err:         &net.DNSError{Err: "no such host", Name: "trunk.example.com"},
			reason:      "dns resolution failed",
			termination: CallTermination{Result: CallTerminationClientError, Reason: "outbound_dns_resolution"},
			retryable:   true,
		},
		{
			name:        "address",
			err:         &net.AddrError{Err: "missing port", Addr: "trunk.example.com"},
			reason:      "invalid SIP address",
			termination: CallTermination{Result: CallTerminationClientError, Reason: "outbound_invalid_address"},
		},
		{
			name:        "transport",
			err:         &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			reason:      "SIP transport failed",
			termination: CallTermination{Result: CallTerminationServerError, Reason: "outbound_network_error"},
			retryable:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := NewOutboundInviteFailure(tt.err)

			assert.Equal(t, OutboundFailureNetwork, failure.Class)
			assert.Equal(t, tt.reason, failure.Reason)
			assert.Equal(t, tt.termination, failure.Termination)
			assert.Equal(t, LifecycleReasonOutboundNetworkFailure, failure.LifecycleReason)
			assert.Equal(t, tt.retryable, failure.Retryable)
			assert.ErrorIs(t, failure, tt.err)
		})
	}
}
