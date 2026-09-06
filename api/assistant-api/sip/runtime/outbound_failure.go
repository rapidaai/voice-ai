// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type OutboundFailureClass string

type OutboundFailure struct {
	StatusCode      int
	Class           OutboundFailureClass
	Reason          string
	Termination     CallTermination
	Retryable       bool
	LifecycleReason LifecycleReason
	Err             error
}

// NewOutboundSetupFailure maps outbound call setup errors before INVITE completion.
// It uses the same failure shape as invite, media, and application failures.
func NewOutboundSetupFailure(err error) OutboundFailure {
	reason := "outbound setup failed"
	if err != nil {
		reason = err.Error()
	}
	return OutboundFailure{
		Class:           OutboundFailureSetup,
		Reason:          reason,
		Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_setup_failed"},
		LifecycleReason: LifecycleReasonOutboundSetupFailure,
		Err:             err,
	}
}

// NewOutboundInviteFailure maps a failed outbound INVITE to lifecycle and SLI state.
// It stays pure so outbound call orchestration owns reporting and teardown side effects.
func NewOutboundInviteFailure(err error) OutboundFailure {
	failure := OutboundFailure{
		Class:           OutboundFailureUnknown,
		Reason:          "unknown",
		Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_invite_failed"},
		LifecycleReason: LifecycleReasonOutboundWaitAnswerFailed,
		Err:             err,
	}

	switch {
	case errors.Is(err, ErrAuthRequired):
		failure.Class = OutboundFailureAuthRequired
		failure.Reason = "auth credentials missing"
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_auth_required"}
		failure.LifecycleReason = LifecycleReasonOutboundAuthFailed
	case errors.Is(err, context.Canceled):
		failure.Class = OutboundFailureCancelled
		failure.Reason = "cancelled"
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_cancelled"}
		failure.LifecycleReason = LifecycleReasonOutboundCancelledBeforeAnswer
	case errors.Is(err, context.DeadlineExceeded):
		failure.Class = OutboundFailureNoAnswer
		failure.Reason = "ringing timeout"
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_no_answer"}
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundNoAnswer
	default:
		var dialogErr *sipgo.ErrDialogResponse
		if errors.As(err, &dialogErr) && dialogErr.Res != nil {
			return NewOutboundSIPResponseFailure(dialogErr.Res, err)
		}

		var dnsErr *net.DNSError
		var addressErr *net.AddrError
		var operationErr *net.OpError
		switch {
		case errors.As(err, &dnsErr):
			failure.Class = OutboundFailureNetwork
			failure.Reason = "dns resolution failed"
			failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_dns_resolution"}
			failure.Retryable = true
			failure.LifecycleReason = LifecycleReasonOutboundNetworkFailure
		case errors.As(err, &addressErr):
			failure.Class = OutboundFailureNetwork
			failure.Reason = "invalid SIP address"
			failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_invalid_address"}
			failure.LifecycleReason = LifecycleReasonOutboundNetworkFailure
		case errors.As(err, &operationErr):
			failure.Class = OutboundFailureNetwork
			failure.Reason = "SIP transport failed"
			failure.Termination = CallTermination{Result: CallTerminationServerError, Reason: "outbound_network_error"}
			failure.Retryable = true
			failure.LifecycleReason = LifecycleReasonOutboundNetworkFailure
		}
	}

	return failure
}

// NewOutboundSIPStatusFailure maps a final SIP response to an outbound failure.
// The caller still owns whether that failure is reported, retried, or used to end the session.
func NewOutboundSIPStatusFailure(statusCode int, reason string, err error) OutboundFailure {
	return newOutboundSIPStatusFailure(statusCode, reason, "", err)
}

// NewOutboundSIPResponseFailure maps a final SIP response to an outbound failure.
// It inspects narrow provider response details where 5xx means customer trunk capacity,
// not upstream outage.
func NewOutboundSIPResponseFailure(response *sip.Response, err error) OutboundFailure {
	if response == nil {
		return NewOutboundSIPStatusFailure(0, "unknown", err)
	}

	detail := response.Reason
	if body := response.Body(); len(body) > 0 {
		detail = string(body)
	} else if twilioError := response.GetHeader("X-Twilio-Error"); twilioError != nil {
		detail = twilioError.Value()
	}
	return newOutboundSIPStatusFailure(response.StatusCode, response.Reason, detail, err)
}

func newOutboundSIPStatusFailure(statusCode int, reason string, detail string, err error) OutboundFailure {
	if statusCode >= 500 && statusCode < 600 {
		providerReason := strings.TrimSpace(detail)
		if providerReason == "" {
			providerReason = reason
		}

		switch normalizedReason := strings.ToLower(providerReason); {
		case strings.Contains(normalizedReason, "cps limit exceeded"):
			return OutboundFailure{
				StatusCode:      statusCode,
				Class:           OutboundFailureTrunkCapacity,
				Reason:          providerReason,
				Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_cps_limit_exceeded"},
				Retryable:       true,
				LifecycleReason: LifecycleReasonOutboundTrunkCapacity,
				Err:             err,
			}
		case strings.Contains(normalizedReason, "concurrent call limit exceeded"):
			return OutboundFailure{
				StatusCode:      statusCode,
				Class:           OutboundFailureTrunkCapacity,
				Reason:          providerReason,
				Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_concurrent_limit_exceeded"},
				Retryable:       true,
				LifecycleReason: LifecycleReasonOutboundTrunkCapacity,
				Err:             err,
			}
		}
	}

	failure := OutboundFailure{
		StatusCode:      statusCode,
		Class:           OutboundFailureRejected,
		Reason:          reason,
		Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_rejected"},
		LifecycleReason: LifecycleReasonOutboundRejected,
		Err:             err,
	}

	switch statusCode {
	case sip.StatusUnauthorized, sip.StatusProxyAuthRequired:
		failure.Class = OutboundFailureAuthRequired
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_auth_required"}
		failure.LifecycleReason = LifecycleReasonOutboundAuthFailed
	case sip.StatusForbidden:
		failure.Class = OutboundFailureForbidden
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_forbidden"}
	case sip.StatusNotFound:
		failure.Class = OutboundFailureNotFound
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_not_found"}
	case sip.StatusRequestTimeout:
		failure.Class = OutboundFailureNoAnswer
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_request_timeout"}
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundNoAnswer
	case sip.StatusTemporarilyUnavailable:
		failure.Class = OutboundFailureUnavailable
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_unavailable"}
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundUnavailable
	case sip.StatusBusyHere, sip.StatusGlobalBusyEverywhere:
		failure.Class = OutboundFailureBusy
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_busy"}
	case sip.StatusNotAcceptableHere:
		failure.Class = OutboundFailureMedia
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_not_acceptable"}
		failure.LifecycleReason = LifecycleReasonOutboundMediaRejected
	case sip.StatusGlobalDecline:
		failure.Class = OutboundFailureRejected
		failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_global_decline"}
	case sip.StatusInternalServerError, sip.StatusBadGateway, sip.StatusServiceUnavailable, sip.StatusGatewayTimeout:
		failure.Class = OutboundFailureUpstreamFailure
		failure.Termination = CallTermination{Result: CallTerminationServerError, Reason: "outbound_upstream_server_error"}
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundUpstreamFailure
	default:
		switch {
		case statusCode >= 400 && statusCode < 500:
			failure.Class = OutboundFailureRejected
			failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_client_error"}
		case statusCode >= 500 && statusCode < 600:
			failure.Class = OutboundFailureUpstreamFailure
			failure.Termination = CallTermination{Result: CallTerminationServerError, Reason: "outbound_upstream_server_error"}
			failure.Retryable = true
			failure.LifecycleReason = LifecycleReasonOutboundUpstreamFailure
		case statusCode >= 600 && statusCode < 700:
			failure.Class = OutboundFailureRejected
			failure.Termination = CallTermination{Result: CallTerminationClientError, Reason: "outbound_global_decline"}
		}
	}

	return failure
}

func (failure OutboundFailure) Error() string {
	if failure.Err != nil {
		return failure.Err.Error()
	}
	if failure.Reason != "" {
		return failure.Reason
	}
	return "outbound failure"
}

func (failure OutboundFailure) Unwrap() error {
	return failure.Err
}

// StatusUpdate converts this failure into the provider-neutral status payload.
func (failure OutboundFailure) StatusUpdate(callID string) internal_type.ProviderCallStatusUpdate {
	status := OutboundCallStatusFailed
	if failure.Class == OutboundFailureCancelled {
		status = OutboundCallStatusCancelled
	}
	errorMessage := ""
	if failure.Err != nil {
		errorMessage = failure.Err.Error()
	}
	return internal_type.ProviderCallStatusUpdate{
		ChannelUUID:        callID,
		CallStatus:         string(status),
		ErrorMessage:       errorMessage,
		FailureClass:       string(failure.Class),
		FailureReason:      failure.Reason,
		SLIResult:          string(failure.Termination.Result),
		SLIReason:          failure.Termination.Reason,
		DisconnectReason:   failure.LifecycleReason.String(),
		Retryable:          failure.Retryable,
		ProviderStatusCode: failure.StatusCode,
	}
}

// Record persists the normalized SIP failure fields on the call session.
func (failure OutboundFailure) Record(session *Session) {
	if session == nil {
		return
	}
	session.SetMetadata("sip.failure_class", string(failure.Class))
	session.SetMetadata("sip.failure_reason", failure.Reason)
	session.SetMetadata("sip.sli_result", string(failure.Termination.Result))
	session.SetMetadata("sip.sli_reason", failure.Termination.Reason)
	session.SetMetadata("sip.failure_retryable", failure.Retryable)
	if failure.StatusCode > 0 {
		session.SetMetadata("sip.failure_status_code", failure.StatusCode)
	}
}
