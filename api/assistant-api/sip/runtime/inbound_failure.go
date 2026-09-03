// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"

type inboundFailureClass string

const (
	inboundFailureConfig           inboundFailureClass = "config"
	inboundFailureAuthRequired     inboundFailureClass = "auth_required"
	inboundFailureMedia            inboundFailureClass = "media"
	inboundFailureUnsupportedMedia inboundFailureClass = "unsupported_media"
	inboundFailureRTPUnavailable   inboundFailureClass = "rtp_unavailable"
	inboundFailureMediaTimeout     inboundFailureClass = "media_timeout"
	inboundFailureDialog           inboundFailureClass = "dialog"
	inboundFailureNoAnswer         inboundFailureClass = "no_answer"
	inboundFailureNoACK            inboundFailureClass = "no_ack"
	inboundFailureApplication      inboundFailureClass = "application"
	inboundFailureSetup            inboundFailureClass = "setup"
	inboundFailureCancelled        inboundFailureClass = "cancelled"
	inboundFailureUnknown          inboundFailureClass = "unknown"
)

type inboundFailure struct {
	statusCode      int
	class           inboundFailureClass
	responseClass   internal_inbound.FailureClass
	reason          string
	termination     CallTermination
	retryable       bool
	lifecycleReason LifecycleReason
	err             error
}

func (failure inboundFailure) Error() string {
	if failure.err != nil {
		return failure.err.Error()
	}
	if failure.reason != "" {
		return failure.reason
	}
	return "inbound failure"
}

func (failure inboundFailure) Unwrap() error {
	return failure.err
}

func newInboundSessionFailure(err error) inboundFailure {
	return inboundFailure{
		statusCode:      500,
		class:           inboundFailureSetup,
		responseClass:   internal_inbound.FailureSetup,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_setup"},
		lifecycleReason: LifecycleReasonInboundInviteFailed,
		err:             err,
	}
}

func newInboundDialogFailure(statusCode int, err error) inboundFailure {
	return inboundFailure{
		statusCode:      statusCode,
		class:           inboundFailureDialog,
		responseClass:   internal_inbound.FailureDialog,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_dialog"},
		lifecycleReason: LifecycleReasonInboundInviteFailed,
		err:             err,
	}
}

func newInboundRTPUnavailableFailure(err error, lifecycleReason LifecycleReason) inboundFailure {
	return inboundFailure{
		statusCode:      503,
		class:           inboundFailureRTPUnavailable,
		responseClass:   internal_inbound.FailureRTP,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_rtp_unavailable"},
		lifecycleReason: lifecycleReason,
		err:             err,
	}
}

func newInboundApplicationFailure(err error) inboundFailure {
	return inboundFailure{
		class:           inboundFailureApplication,
		responseClass:   internal_inbound.FailureSetup,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_application"},
		lifecycleReason: LifecycleReasonPipelineSetupFailed,
		err:             err,
	}
}

func newInboundNoAnswerFailure(err error) inboundFailure {
	return inboundFailure{
		statusCode:      408,
		class:           inboundFailureNoAnswer,
		responseClass:   internal_inbound.FailureSetup,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_no_answer"},
		lifecycleReason: LifecycleReasonInboundAnswerPolicyTimeout,
		err:             err,
	}
}

func newInboundNoACKFailure(err error) inboundFailure {
	return inboundFailure{
		class:           inboundFailureNoACK,
		responseClass:   internal_inbound.FailureDialog,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_no_ack"},
		retryable:       true,
		lifecycleReason: LifecycleReasonInboundACKTimeout,
		err:             err,
	}
}

func newInboundMediaTimeoutFailure() inboundFailure {
	return inboundFailure{
		class:           inboundFailureMediaTimeout,
		responseClass:   internal_inbound.FailureRTP,
		reason:          ErrRTPMediaTimeout.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_media_timeout"},
		retryable:       true,
		lifecycleReason: LifecycleReasonInboundMediaTimeout,
		err:             ErrRTPMediaTimeout,
	}
}
