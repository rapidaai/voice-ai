// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
)

// Server state constants describe the process-level SIP server lifecycle.
const (
	ServerStateCreated ServerState = iota
	ServerStateRunning
	ServerStateStopped
)

// Inbound rejection cache constants bound final-response replay for completed INVITEs.
const (
	InboundRejectedInviteTTL  = time.Minute
	MaxInboundRejectedInvites = 1024
)

// Transport constants list supported SIP transport protocols.
const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
	TransportTLS Transport = "tls"
)

// Inbound answer modes control when an inbound INVITE receives 200 OK.
const (
	InboundAnswerModeImmediate            InboundAnswerMode = "answer_immediately"
	InboundAnswerModeAfterMinRingDuration InboundAnswerMode = "answer_after_min_ring_ms"
)

// Call states represent the session lifecycle stored on SessionInfo.
const (
	CallStateInitializing    CallState = "initializing"
	CallStateRinging         CallState = "ringing"
	CallStateConnected       CallState = "connected"
	CallStateOnHold          CallState = "on_hold"
	CallStateTransferring    CallState = "transferring"
	CallStateBridgeConnected CallState = "bridge_connected"
	CallStateEnding          CallState = "ending"
	CallStateEnded           CallState = "ended"
	CallStateFailed          CallState = "failed"
	CallStateCancelled       CallState = "cancelled"
)

// Call direction constants record whether this runtime originated or accepted the call.
const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// Inbound setup phases identify observable milestones during INVITE handling.
const (
	InboundSetupPhaseInviteReceived   InboundSetupPhase = "invite_received"
	InboundSetupPhaseTryingSent       InboundSetupPhase = "trying_sent"
	InboundSetupPhaseRingingSent      InboundSetupPhase = "ringing_sent"
	InboundSetupPhaseAuthenticated    InboundSetupPhase = "authenticated"
	InboundSetupPhaseRouted           InboundSetupPhase = "routed"
	InboundSetupPhaseMediaAllocated   InboundSetupPhase = "media_allocated"
	InboundSetupPhaseApplicationReady InboundSetupPhase = "application_ready"
	InboundSetupPhaseAnswerReady      InboundSetupPhase = "answer_ready"
	InboundSetupPhaseAnswered         InboundSetupPhase = "answered"
	InboundSetupPhaseACKConfirmed     InboundSetupPhase = "ack_confirmed"
	InboundSetupPhaseMediaFlowing     InboundSetupPhase = "media_flowing"
)

// Bridge timing constants bound transfer setup and bridge lifetime.
const (
	BridgeCallTimeout   = 30 * time.Second
	BridgeSafetyTimeout = 5 * time.Minute
)

// Bridge metadata keys are written to session metadata for transfer orchestration.
const (
	MetadataBridgeTransferTarget         = "bridge_transfer_target"
	MetadataBridgeTransferStatus         = "bridge_transfer_status"
	MetadataBridgeTransferDuration       = "bridge_transfer_duration"
	MetadataBridgeTransferOutboundCallID = "bridge_transfer_outbound_call_id"
)

// Disconnect metadata keys preserve the terminal SIP reason in session metadata.
const (
	MetadataDisconnectReason    = "disconnect_reason"
	MetadataDisconnectRawReason = "disconnect_raw_reason"
)

// Post-transfer actions control inbound behavior after the operator leg ends.
const (
	PostTransferActionEndCall  = "end_call"
	PostTransferActionResumeAI = "resume_ai"
)

// Disconnect reasons are canonical terminal reasons reported outside the SIP runtime.
const (
	DisconnectReasonRemoteHangup   = "remote_hangup"
	DisconnectReasonNormalClearing = "normal_clearing"
	DisconnectReasonBusy           = "busy"
	DisconnectReasonNoAnswer       = "no_answer"
	DisconnectReasonRejected       = "rejected"
	DisconnectReasonCancelled      = "cancelled"
	DisconnectReasonNetworkFailure = "network_failure"
	DisconnectReasonRemoteError    = "remote_error"
)

// Lifecycle reasons identify why a call lifecycle transition occurred.
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

// Call termination result constants classify final call completion outcome.
const (
	CallTerminationSuccess     CallTerminationResult = "success"
	CallTerminationClientError CallTerminationResult = "client_error"
	CallTerminationServerError CallTerminationResult = "server_error"
)

// Bridge end reasons identify which lifecycle signal ended a transfer bridge.
const (
	BridgeEndInboundBye BridgeEndReason = iota
	BridgeEndOutboundBye
	BridgeEndContext
	BridgeEndTimeout
)

// Outbound failure classes group call failures for reporting and retry policy.
const (
	OutboundFailureAuthRequired    OutboundFailureClass = "auth_required"
	OutboundFailureSetup           OutboundFailureClass = "setup"
	OutboundFailureForbidden       OutboundFailureClass = "forbidden"
	OutboundFailureNotFound        OutboundFailureClass = "not_found"
	OutboundFailureNoAnswer        OutboundFailureClass = "no_answer"
	OutboundFailureUnavailable     OutboundFailureClass = "unavailable"
	OutboundFailureBusy            OutboundFailureClass = "busy"
	OutboundFailureRejected        OutboundFailureClass = "rejected"
	OutboundFailureUpstreamFailure OutboundFailureClass = "upstream_failure"
	OutboundFailureTrunkCapacity   OutboundFailureClass = "trunk_capacity"
	OutboundFailureNetwork         OutboundFailureClass = "network"
	OutboundFailureCancelled       OutboundFailureClass = "cancelled"
	OutboundFailureMedia           OutboundFailureClass = "media"
	OutboundFailureApplication     OutboundFailureClass = "application"
	OutboundFailureUnknown         OutboundFailureClass = "unknown"
)

// Outbound modes select the dialing strategy for a call leg.
const (
	OutboundModeTrunkTermination OutboundMode = "trunk_termination"
)

// Outbound leg purposes distinguish primary calls from transfer bridge legs.
const (
	OutboundLegPurposePrimary        OutboundLegPurpose = "primary_outbound_call"
	OutboundLegPurposeTransferBridge OutboundLegPurpose = "transfer_bridge_call"
)

// Outbound dialog phases track INVITE dialog progress.
const (
	OutboundDialogPhaseInviting   OutboundDialogPhase = "inviting"
	OutboundDialogPhaseProceeding OutboundDialogPhase = "proceeding"
	OutboundDialogPhaseAnswered   OutboundDialogPhase = "answered"
	OutboundDialogPhaseConfirmed  OutboundDialogPhase = "confirmed"
	OutboundDialogPhaseTerminated OutboundDialogPhase = "terminated"
)

// Outbound timing constants bound the ringing phase.
const (
	defaultOutboundRingingTimeout = 60 * time.Second
)

// Outbound call statuses mirror provider-neutral callcontext statuses.
const (
	OutboundCallStatusInitiated OutboundCallStatus = callcontext.CallStatusInitiated
	OutboundCallStatusRinging   OutboundCallStatus = callcontext.CallStatusRinging
	OutboundCallStatusAnswered  OutboundCallStatus = callcontext.CallStatusAnswered
	OutboundCallStatusFailed    OutboundCallStatus = callcontext.CallStatusFailed
	OutboundCallStatusCompleted OutboundCallStatus = callcontext.CallStatusCompleted
	OutboundCallStatusCancelled OutboundCallStatus = callcontext.CallStatusCancelled
)

// Outbound metadata keys link transfer B-leg calls back to their parent context.
const (
	MetadataOutboundLegPurpose           = "outbound_leg_purpose"
	MetadataOutboundParentCallID         = "outbound_parent_call_id"
	MetadataOutboundParentContextID      = "outbound_parent_context_id"
	MetadataOutboundParentConversationID = "outbound_parent_conversation_id"
	MetadataOutboundTransferTarget       = "outbound_transfer_target"
	MetadataOutboundTransferAttempt      = "outbound_transfer_attempt"
	MetadataOutboundTransferTotal        = "outbound_transfer_total"
)

// Registration timing constants define REGISTER renewal and retry behavior.
const (
	defaultRegisterExpiry      uint32 = 3600
	renewalFraction                   = 0.8
	defaultRegisterTimeout            = 10 * time.Second
	renewRetryInterval                = 30 * time.Second
	maxRegistrationExpiryGrace        = 60 * time.Second
)

// Registration failure classes group registration errors by owner and retry policy.
const (
	RegistrationFailureClassConfig     RegistrationFailureClass = "config"
	RegistrationFailureClassAuth       RegistrationFailureClass = "auth"
	RegistrationFailureClassRejected   RegistrationFailureClass = "rejected"
	RegistrationFailureClassTransient  RegistrationFailureClass = "transient"
	RegistrationFailureClassNetwork    RegistrationFailureClass = "network"
	RegistrationFailureClassOwnership  RegistrationFailureClass = "ownership"
	RegistrationFailureClassDuplicate  RegistrationFailureClass = "duplicate"
	RegistrationFailureClassRenewal    RegistrationFailureClass = "renewal"
	RegistrationFailureClassUnregister RegistrationFailureClass = "unregister"
)

// Registration failure reasons provide stable machine-readable registration details.
const (
	RegistrationFailureReasonMissingDID              RegistrationFailureReason = "missing_did"
	RegistrationFailureReasonMissingCredentialID     RegistrationFailureReason = "missing_credential_id"
	RegistrationFailureReasonDuplicateDID            RegistrationFailureReason = "duplicate_did"
	RegistrationFailureReasonAssistantNotFound       RegistrationFailureReason = "assistant_not_found"
	RegistrationFailureReasonVaultCredentialNotFound RegistrationFailureReason = "vault_credential_not_found" // #nosec G101, registration state enum.
	RegistrationFailureReasonInvalidSIPConfig        RegistrationFailureReason = "invalid_sip_config"
	RegistrationFailureReasonMissingSIPServer        RegistrationFailureReason = "missing_sip_server"
	RegistrationFailureReasonOwnershipClaimFailed    RegistrationFailureReason = "ownership_claim_failed"
	RegistrationFailureReasonAuthFailed              RegistrationFailureReason = "auth_failed"
	RegistrationFailureReasonRegistrarRejected       RegistrationFailureReason = "registrar_rejected"
	RegistrationFailureReasonRegistrarUnreachable    RegistrationFailureReason = "registrar_unreachable"
	RegistrationFailureReasonTransportError          RegistrationFailureReason = "transport_error"
	RegistrationFailureReasonRegisterTimeout         RegistrationFailureReason = "register_timeout"
	RegistrationFailureReasonRenewalFailed           RegistrationFailureReason = "renewal_failed"
	RegistrationFailureReasonUnregisterFailed        RegistrationFailureReason = "unregister_failed"
	RegistrationFailureReasonInvalidContactAddress   RegistrationFailureReason = "invalid_contact_address"
)

// SDP parser constants identify line prefixes and packetization bounds.
const (
	sdpConnectionIPPrefix = "c=IN IP4 "
	sdpAudioPrefix        = "m=audio "
	sdpRTPMapPrefix       = "a=rtpmap:"
	sdpRTCPPrefix         = "a=rtcp:"
	sdpPTimePrefix        = "a=ptime:"
	sdpContentType        = "application/sdp"
	sdpDefaultPTimeMS     = 20
	sdpMinPTimeMS         = 5
	sdpMaxPTimeMS         = 60
)

// SDP directions represent media direction attributes.
const (
	SDPDirectionSendRecv SDPDirection = "sendrecv"
	SDPDirectionSendOnly SDPDirection = "sendonly"
	SDPDirectionRecvOnly SDPDirection = "recvonly"
	SDPDirectionInactive SDPDirection = "inactive"
)

// RTP packet and socket constants define baseline RTP transport behavior.
const (
	rtpVersion                  = 2
	rtpHeaderSize               = 12
	rtpDefaultClockRate         = 8000
	rtpMaxPort                  = 65535
	rtpReadBufferSize           = 65536
	rtpWriteBufferSize          = 65536
	rtpPacketMaxSize            = 1500
	rtpPacketInterval           = 20 * time.Millisecond
	rtpDefaultPacketizationTime = 20 * time.Millisecond
	rtpMinPacketizationTime     = 5 * time.Millisecond
	rtpMaxPacketizationTime     = 60 * time.Millisecond
	rtpMediaTimeoutInitial      = 30 * time.Second
	rtpMediaTimeout             = 15 * time.Second
	rtpAudioInBufferSize        = 100
	rtpAudioOutBufferSize       = 100
	rtpMediaTimeoutDisabledPark = time.Hour
)

// RTP network constants select UDP address family for socket creation.
const (
	rtpNetworkUDP4 = "udp4"
	rtpNetworkUDP6 = "udp6"
)

// RTP error format constants keep wrapped validation errors consistent.
const (
	rtpNewHandlerOperation   = "NewRTPHandler"
	rtpErrorIntFormat        = "%w: %d"
	rtpErrorMaxPortFormat    = "%w: max=%d"
	rtpErrorPortRangeFormat  = "%w: range=%d-%d tried=%d"
	rtpErrorSizeHeaderFormat = "%w: size=%d header=%d"
)

// RTP input jitter buffer constants bound reordering and missing audio duration.
const (
	rtpInputReorderWindow             = 80 * time.Millisecond
	rtpInputMaxLossGap                = 500 * time.Millisecond
	rtpInputMaxSilenceGap             = 500 * time.Millisecond
	rtpInputBufferedPacketMapCapacity = 5
)

// RTP inbound quality constants define rolling quality thresholds and labels.
const (
	rtpInboundQualityWindow       = 5 * time.Second
	rtpInboundQualityGoodLossRate = 0.05
	rtpInboundQualityPoorLossRate = 0.12
	rtpInboundQualityUnknown      = "unknown"
	rtpInboundQualityExcellent    = "excellent"
	rtpInboundQualityGood         = "good"
	rtpInboundQualityPoor         = "poor"
	rtpInboundQualityLost         = "lost"
)

// RTCP constants define companion port behavior, reporting cadence, and unit conversion.
const (
	rtcpPortOffset               = 1
	rtcpReadBufferSize           = 1500
	rtcpWriteBufferSize          = 1500
	rtcpReportInterval           = 5 * time.Second
	rtcpReadTimeout              = time.Second
	rtcpCNAMEFormat              = "rapida-%08x@%s"
	rtcpNanoseconds              = 1000000000
	rtcpNanosecondsUint64        = 1000000000
	rtcpNTPUnixOffset            = 2208988800
	rtcpNTPFractionUnit          = 1 << 32
	rtcpCompactUnit              = 65536
	rtcpMaxUint8                 = 255
	rtcpMaxUint32                = 1<<32 - 1
	rtcpMaxUint32Int64           = 1<<32 - 1
	rtpSequenceCycle             = 1 << 16
	rtpSequenceRolloverThreshold = 1 << 15
)

// Inbound request timing and retry constants bound delayed answers and final responses.
const (
	defaultInboundACKTimeout                = 5 * time.Second
	defaultInboundFinalResponseRetryInitial = 500 * time.Millisecond
	defaultInboundFinalResponseRetryMax     = 4 * time.Second
	defaultInboundRingingInterval           = time.Second
	sipHeaderRetryAfter                     = "Retry-After"
	sipCapacityRetryAfterSeconds            = 1
)

// Inbound failure classes group INVITE failures for response mapping and reporting.
const (
	inboundFailureConfig           inboundFailureClass = "config"
	inboundFailureAuth             inboundFailureClass = "auth"
	inboundFailureAuthRequired     inboundFailureClass = "auth_required"
	inboundFailureMedia            inboundFailureClass = "media"
	inboundFailureRTP              inboundFailureClass = "rtp"
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
