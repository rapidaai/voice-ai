// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

type inboundFailureClass string

type inboundFailure struct {
	statusCode      int
	class           inboundFailureClass
	responseClass   inboundFailureClass
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
		responseClass:   inboundFailureSetup,
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
		responseClass:   inboundFailureDialog,
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
		responseClass:   inboundFailureRTP,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_rtp_unavailable"},
		lifecycleReason: lifecycleReason,
		err:             err,
	}
}

func newInboundApplicationFailure(err error) inboundFailure {
	return inboundFailure{
		class:           inboundFailureApplication,
		responseClass:   inboundFailureSetup,
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
		responseClass:   inboundFailureSetup,
		reason:          err.Error(),
		termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_no_answer"},
		lifecycleReason: LifecycleReasonInboundAnswerPolicyTimeout,
		err:             err,
	}
}

func newInboundNoACKFailure(err error) inboundFailure {
	return inboundFailure{
		class:           inboundFailureNoACK,
		responseClass:   inboundFailureDialog,
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
		responseClass:   inboundFailureRTP,
		reason:          ErrRTPMediaTimeout.Error(),
		termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_media_timeout"},
		retryable:       true,
		lifecycleReason: LifecycleReasonInboundMediaTimeout,
		err:             ErrRTPMediaTimeout,
	}
}
