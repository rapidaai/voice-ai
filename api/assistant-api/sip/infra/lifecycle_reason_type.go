// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type LifecycleReason = internal_core.LifecycleReason

const (
	LifecycleReasonEndCall                        = internal_core.LifecycleReasonEndCall
	LifecycleReasonSessionEnd                     = internal_core.LifecycleReasonSessionEnd
	LifecycleReasonServerStop                     = internal_core.LifecycleReasonServerStop
	LifecycleReasonRemoteBye                      = internal_core.LifecycleReasonRemoteBye
	LifecycleReasonCancelReceived                 = internal_core.LifecycleReasonCancelReceived
	LifecycleReasonInviteCancelled                = internal_core.LifecycleReasonInviteCancelled
	LifecycleReasonInviteCancelledBeforeAnswer    = internal_core.LifecycleReasonInviteCancelledBeforeAnswer
	LifecycleReasonInboundInviteReceived          = internal_core.LifecycleReasonInboundInviteReceived
	LifecycleReasonInboundAuthenticated           = internal_core.LifecycleReasonInboundAuthenticated
	LifecycleReasonInboundRouted                  = internal_core.LifecycleReasonInboundRouted
	LifecycleReasonInboundInviteRinging           = internal_core.LifecycleReasonInboundInviteRinging
	LifecycleReasonInboundMediaAllocated          = internal_core.LifecycleReasonInboundMediaAllocated
	LifecycleReasonInboundApplicationReady        = internal_core.LifecycleReasonInboundApplicationReady
	LifecycleReasonInboundAnswerPolicyReady       = internal_core.LifecycleReasonInboundAnswerPolicyReady
	LifecycleReasonInboundInviteAnswered          = internal_core.LifecycleReasonInboundInviteAnswered
	LifecycleReasonInboundInviteFailed            = internal_core.LifecycleReasonInboundInviteFailed
	LifecycleReasonInboundInviteACKReceived       = internal_core.LifecycleReasonInboundInviteACKReceived
	LifecycleReasonInboundMediaFlowing            = internal_core.LifecycleReasonInboundMediaFlowing
	LifecycleReasonInboundMediaFailed             = internal_core.LifecycleReasonInboundMediaFailed
	LifecycleReasonInboundMediaTimeout            = internal_core.LifecycleReasonInboundMediaTimeout
	LifecycleReasonInboundFirstRTPReceived        = internal_core.LifecycleReasonInboundFirstRTPReceived
	LifecycleReasonInboundAssistantAudioReady     = internal_core.LifecycleReasonInboundAssistantAudioReady
	LifecycleReasonInboundFirstAssistantAudioSent = internal_core.LifecycleReasonInboundFirstAssistantAudioSent
	LifecycleReasonInboundReinviteACKReceived     = internal_core.LifecycleReasonInboundReinviteACKReceived
	LifecycleReasonInboundReinviteSDPRejected     = internal_core.LifecycleReasonInboundReinviteSDPRejected
	LifecycleReasonInboundUpdateSDPRejected       = internal_core.LifecycleReasonInboundUpdateSDPRejected
	LifecycleReasonInboundACKTimeout              = internal_core.LifecycleReasonInboundACKTimeout
	LifecycleReasonInboundAnswerPolicyTimeout     = internal_core.LifecycleReasonInboundAnswerPolicyTimeout
	LifecycleReasonInboundLateACK                 = internal_core.LifecycleReasonInboundLateACK
	LifecycleReasonPipelineCallbacksMissing       = internal_core.LifecycleReasonPipelineCallbacksMissing
	LifecycleReasonPipelineConversationMissing    = internal_core.LifecycleReasonPipelineConversationMissing
	LifecycleReasonPipelineConversationFailed     = internal_core.LifecycleReasonPipelineConversationFailed
	LifecycleReasonPipelineSetupFailed            = internal_core.LifecycleReasonPipelineSetupFailed
	LifecycleReasonPipelineTalkCompleted          = internal_core.LifecycleReasonPipelineTalkCompleted
	LifecycleReasonPipelineCallEnd                = internal_core.LifecycleReasonPipelineCallEnd
	LifecycleReasonStreamerEndSession             = internal_core.LifecycleReasonStreamerEndSession
	LifecycleReasonStreamerClosed                 = internal_core.LifecycleReasonStreamerClosed
	LifecycleReasonOutboundCancelledBeforeAnswer  = internal_core.LifecycleReasonOutboundCancelledBeforeAnswer
	LifecycleReasonOutboundSetupFailure           = internal_core.LifecycleReasonOutboundSetupFailure
	LifecycleReasonOutboundProgressRinging        = internal_core.LifecycleReasonOutboundProgressRinging
	LifecycleReasonOutboundWaitAnswerFailed       = internal_core.LifecycleReasonOutboundWaitAnswerFailed
	LifecycleReasonOutboundAuthFailed             = internal_core.LifecycleReasonOutboundAuthFailed
	LifecycleReasonOutboundNoAnswer               = internal_core.LifecycleReasonOutboundNoAnswer
	LifecycleReasonOutboundUnavailable            = internal_core.LifecycleReasonOutboundUnavailable
	LifecycleReasonOutboundRejected               = internal_core.LifecycleReasonOutboundRejected
	LifecycleReasonOutboundMediaRejected          = internal_core.LifecycleReasonOutboundMediaRejected
	LifecycleReasonOutboundMediaTimeout           = internal_core.LifecycleReasonOutboundMediaTimeout
	LifecycleReasonOutboundUpstreamFailure        = internal_core.LifecycleReasonOutboundUpstreamFailure
	LifecycleReasonOutboundTrunkCapacity          = internal_core.LifecycleReasonOutboundTrunkCapacity
	LifecycleReasonOutboundNetworkFailure         = internal_core.LifecycleReasonOutboundNetworkFailure
	LifecycleReasonOutboundACKSent                = internal_core.LifecycleReasonOutboundACKSent
	LifecycleReasonOutboundAnswerSDPFailed        = internal_core.LifecycleReasonOutboundAnswerSDPFailed
	LifecycleReasonOutboundACKFailed              = internal_core.LifecycleReasonOutboundACKFailed
	LifecycleReasonOutboundMaxDuration            = internal_core.LifecycleReasonOutboundMaxDuration
	LifecycleReasonOutboundTeardownTimeout        = internal_core.LifecycleReasonOutboundTeardownTimeout
	LifecycleReasonBridgeSetupFailed              = internal_core.LifecycleReasonBridgeSetupFailed
	LifecycleReasonBridgeTransferStarted          = internal_core.LifecycleReasonBridgeTransferStarted
	LifecycleReasonBridgeMediaConnected           = internal_core.LifecycleReasonBridgeMediaConnected
	LifecycleReasonBridgeRTPUnavailable           = internal_core.LifecycleReasonBridgeRTPUnavailable
	LifecycleReasonTransferModeStarted            = internal_core.LifecycleReasonTransferModeStarted
	LifecycleReasonTransferModeEnded              = internal_core.LifecycleReasonTransferModeEnded
	LifecycleReasonTransferOutboundEnded          = internal_core.LifecycleReasonTransferOutboundEnded
)

type LifecycleController interface {
	TransitionCall(session *Session, next CallState, reason LifecycleReason) bool
	EndCallWithReason(session *Session, reason LifecycleReason) error
	FailCall(session *Session, reason LifecycleReason, err error) error
	CancelCall(session *Session, reason LifecycleReason) error
}
