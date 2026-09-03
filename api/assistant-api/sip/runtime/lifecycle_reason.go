// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

type LifecycleReason string

const (
	LifecycleReasonEndCall                        LifecycleReason = "end_call"
	LifecycleReasonSessionEnd                     LifecycleReason = "session_end"
	LifecycleReasonServerStop                     LifecycleReason = "server_stop"
	LifecycleReasonRemoteBye                      LifecycleReason = "remote_bye"
	LifecycleReasonCancelReceived                 LifecycleReason = "cancel_received"
	LifecycleReasonInviteCancelled                LifecycleReason = "invite_cancelled"
	LifecycleReasonInviteCancelledBeforeAnswer    LifecycleReason = "invite_cancelled_before_answer"
	LifecycleReasonInboundInviteReceived          LifecycleReason = "inbound_invite_received"
	LifecycleReasonInboundAuthenticated           LifecycleReason = "inbound_authenticated"
	LifecycleReasonInboundRouted                  LifecycleReason = "inbound_routed"
	LifecycleReasonInboundInviteRinging           LifecycleReason = "inbound_invite_ringing"
	LifecycleReasonInboundMediaAllocated          LifecycleReason = "inbound_media_allocated"
	LifecycleReasonInboundApplicationReady        LifecycleReason = "inbound_application_ready"
	LifecycleReasonInboundAnswerPolicyReady       LifecycleReason = "inbound_answer_policy_ready"
	LifecycleReasonInboundInviteAnswered          LifecycleReason = "inbound_invite_answered"
	LifecycleReasonInboundInviteFailed            LifecycleReason = "inbound_invite_failed"
	LifecycleReasonInboundInviteACKReceived       LifecycleReason = "inbound_invite_ack_received"
	LifecycleReasonInboundMediaFlowing            LifecycleReason = "inbound_media_flowing"
	LifecycleReasonInboundMediaFailed             LifecycleReason = "inbound_media_failed"
	LifecycleReasonInboundMediaTimeout            LifecycleReason = "inbound_media_timeout"
	LifecycleReasonInboundFirstRTPReceived        LifecycleReason = "inbound_first_rtp_received"
	LifecycleReasonInboundAssistantAudioReady     LifecycleReason = "inbound_assistant_audio_ready"
	LifecycleReasonInboundFirstAssistantAudioSent LifecycleReason = "inbound_first_assistant_audio_sent"
	LifecycleReasonInboundReinviteACKReceived     LifecycleReason = "inbound_reinvite_ack_received"
	LifecycleReasonInboundReinviteSDPRejected     LifecycleReason = "inbound_reinvite_sdp_rejected"
	LifecycleReasonInboundUpdateSDPRejected       LifecycleReason = "inbound_update_sdp_rejected"
	LifecycleReasonInboundACKTimeout              LifecycleReason = "inbound_ack_timeout"
	LifecycleReasonInboundAnswerPolicyTimeout     LifecycleReason = "inbound_answer_policy_timeout"
	LifecycleReasonInboundLateACK                 LifecycleReason = "inbound_late_ack"
	LifecycleReasonPipelineCallbacksMissing       LifecycleReason = "pipeline_callbacks_missing"
	LifecycleReasonPipelineConversationMissing    LifecycleReason = "pipeline_conversation_missing"
	LifecycleReasonPipelineConversationFailed     LifecycleReason = "pipeline_conversation_failed"
	LifecycleReasonPipelineSetupFailed            LifecycleReason = "pipeline_setup_failed"
	LifecycleReasonPipelineTalkCompleted          LifecycleReason = "pipeline_talk_completed"
	LifecycleReasonPipelineCallEnd                LifecycleReason = "pipeline_call_end"
	LifecycleReasonStreamerEndSession             LifecycleReason = "streamer_end_session"
	LifecycleReasonStreamerClosed                 LifecycleReason = "streamer_closed"
	LifecycleReasonOutboundCancelledBeforeAnswer  LifecycleReason = "outbound_cancelled_before_answer"
	LifecycleReasonOutboundSetupFailure           LifecycleReason = "outbound_setup_failure"
	LifecycleReasonOutboundProgressRinging        LifecycleReason = "outbound_progress_ringing"
	LifecycleReasonOutboundWaitAnswerFailed       LifecycleReason = "outbound_wait_answer_failed"
	LifecycleReasonOutboundAuthFailed             LifecycleReason = "outbound_auth_failed"
	LifecycleReasonOutboundNoAnswer               LifecycleReason = "outbound_no_answer"
	LifecycleReasonOutboundUnavailable            LifecycleReason = "outbound_unavailable"
	LifecycleReasonOutboundRejected               LifecycleReason = "outbound_rejected"
	LifecycleReasonOutboundMediaRejected          LifecycleReason = "outbound_media_rejected"
	LifecycleReasonOutboundMediaTimeout           LifecycleReason = "outbound_media_timeout"
	LifecycleReasonOutboundUpstreamFailure        LifecycleReason = "outbound_upstream_failure"
	LifecycleReasonOutboundTrunkCapacity          LifecycleReason = "outbound_trunk_capacity"
	LifecycleReasonOutboundNetworkFailure         LifecycleReason = "outbound_network_failure"
	LifecycleReasonOutboundACKSent                LifecycleReason = "outbound_ack_sent"
	LifecycleReasonOutboundAnswerSDPFailed        LifecycleReason = "outbound_answer_sdp_failed"
	LifecycleReasonOutboundACKFailed              LifecycleReason = "outbound_ack_failed"
	LifecycleReasonOutboundMaxDuration            LifecycleReason = "outbound_max_duration"
	LifecycleReasonOutboundTeardownTimeout        LifecycleReason = "outbound_teardown_timeout"
	LifecycleReasonBridgeSetupFailed              LifecycleReason = "bridge_setup_failed"
	LifecycleReasonBridgeTransferStarted          LifecycleReason = "bridge_transfer_started"
	LifecycleReasonBridgeMediaConnected           LifecycleReason = "bridge_media_connected"
	LifecycleReasonBridgeRTPUnavailable           LifecycleReason = "bridge_rtp_unavailable"
	LifecycleReasonTransferModeStarted            LifecycleReason = "transfer_mode_started"
	LifecycleReasonTransferModeEnded              LifecycleReason = "transfer_mode_ended"
	LifecycleReasonTransferOutboundEnded          LifecycleReason = "transfer_outbound_ended"
)

func (r LifecycleReason) String() string {
	return string(r)
}

type LifecycleController interface {
	TransitionCall(session *Session, next CallState, reason LifecycleReason) bool
	EndCallWithReason(session *Session, reason LifecycleReason) error
	FailCall(session *Session, reason LifecycleReason, err error) error
	CancelCall(session *Session, reason LifecycleReason) error
}
