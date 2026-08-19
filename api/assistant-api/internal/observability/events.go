// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import "strings"

// ComponentName is the first segment of a stable event name, such as "call" in
// "call.ringing".
type ComponentName string

const (
	ComponentUnknown        ComponentName = ""
	ComponentCall           ComponentName = "call"
	ComponentConversation   ComponentName = "conversation"
	ComponentTurn           ComponentName = "turn"
	ComponentSTT            ComponentName = "stt"
	ComponentTTS            ComponentName = "tts"
	ComponentAgent          ComponentName = "agent"
	ComponentVAD            ComponentName = "vad"
	ComponentEOS            ComponentName = "eos"
	ComponentDenoise        ComponentName = "denoise"
	ComponentTool           ComponentName = "tool"
	ComponentWebhook        ComponentName = "webhook"
	ComponentAnalysis       ComponentName = "analysis"
	ComponentAuthentication ComponentName = "authentication"
	ComponentRecording      ComponentName = "recording"
	ComponentStorage        ComponentName = "storage"
	ComponentSIP            ComponentName = "sip"
	ComponentWebRTC         ComponentName = "webrtc"
	ComponentUsage          ComponentName = "usage"
	ComponentLog            ComponentName = "log"
	ComponentMetric         ComponentName = "metric"
	ComponentMetadata       ComponentName = "metadata"
)

func (c ComponentName) String() string {
	return string(c)
}

// EventName is the stable domain event name emitted by the assistant runtime.
type EventName string

const (
	CallStatus                 EventName = "call.status"
	CallReceived               EventName = "call.received"
	CallRinging                EventName = "call.ringing"
	CallStarted                EventName = "call.started"
	CallMediaStarted           EventName = "call.media_started"
	CallHangup                 EventName = "call.hangup"
	CallEnded                  EventName = "call.ended"
	CallFailed                 EventName = "call.failed"
	CallCancelled              EventName = "call.cancelled"
	CallOutboundRequested      EventName = "call.outbound_requested"
	CallOutboundDispatched     EventName = "call.outbound_dispatched"
	CallOutboundDispatchFailed EventName = "call.outbound_dispatch_failed"
	CallProviderAnswered       EventName = "call.provider_answered"
	CallSessionConnected       EventName = "call.session_connected"
	CallAssistantLoaded        EventName = "call.assistant_loaded"
	CallConversationCreated    EventName = "call.conversation_created"
	CallContextSaved           EventName = "call.context_saved"
)

const (
	ConversationBegin                 EventName = "conversation.begin"
	ConversationResume                EventName = "conversation.resume"
	ConversationInitializing          EventName = "conversation.initializing"
	ConversationInitialized           EventName = "conversation.initialized"
	ConversationAuthenticationStarted EventName = "conversation.authentication_started"
	ConversationCompleted             EventName = "conversation.completed"
	ConversationCleanup               EventName = "conversation.cleanup"
	ConversationError                 EventName = "conversation.error"
	ConversationAgentStateChanged     EventName = "conversation.agent_state_changed"
	ConversationModeSwitchFailed      EventName = "conversation.mode_switch_failed"
)

const (
	TurnInterrupted EventName = "turn.interrupted"
	TurnChange      EventName = "turn.change"
	TurnStarted     EventName = "turn.started"
)

const (
	STTInterim       EventName = "stt.interim"
	STTCompleted     EventName = "stt.completed"
	STTLowConfidence EventName = "stt.low_confidence"
	STTClosed        EventName = "stt.closed"
	STTError         EventName = "stt.error"
)

const (
	TTSSpeaking    EventName = "tts.speaking"
	TTSCompleted   EventName = "tts.completed"
	TTSInterrupted EventName = "tts.interrupted"
	TTSClosed      EventName = "tts.closed"
	TTSError       EventName = "tts.error"
)

const (
	AgentStarted   EventName = "agent.started"
	AgentCompleted EventName = "agent.completed"
	AgentDiscarded EventName = "agent.discarded"
	AgentError     EventName = "agent.error"
)

const (
	AgentTransitionTriggered   EventName = "agent.transition.triggered"
	AgentTransitionMatched     EventName = "agent.transition.matched"
	AgentTransitionMissingEdge EventName = "agent.transition.missing_edge"
)

const (
	VADSpeechStarted EventName = "vad.speech_started"
	VADSpeechEnded   EventName = "vad.speech_ended"
	VADClosed        EventName = "vad.closed"
	VADError         EventName = "vad.error"
)

const (
	EOSStarted   EventName = "eos.started"
	EOSCompleted EventName = "eos.completed"
	EOSClosed    EventName = "eos.closed"
)

const (
	DenoiseClosed EventName = "denoise.closed"
	DenoiseError  EventName = "denoise.error"
)

const (
	ToolCallStarted   EventName = "tool.call_started"
	ToolCallCompleted EventName = "tool.call_completed"
	ToolCallFailed    EventName = "tool.call_failed"
)

const (
	RecordingStarted   EventName = "recording.started"
	RecordingCompleted EventName = "recording.completed"
)

const (
	SIPTransferRequested     EventName = "sip.transfer_requested"
	SIPTransferring          EventName = "sip.transferring"
	SIPRegisterStarted       EventName = "sip.register_started"
	SIPRegisterActive        EventName = "sip.register_active"
	SIPRegisterFailed        EventName = "sip.register_failed"
	SIPRegisterRenewed       EventName = "sip.register_renewed"
	SIPRegisterRenewalFailed EventName = "sip.register_renewal_failed"
	SIPRegisterExpired       EventName = "sip.register_expired"
	SIPUnregisterFailed      EventName = "sip.unregister_failed"
)

const (
	WebRTCConnecting                EventName = "webrtc.connecting"
	WebRTCConnected                 EventName = "webrtc.connected"
	WebRTCReconnecting              EventName = "webrtc.reconnecting"
	WebRTCDisconnected              EventName = "webrtc.disconnected"
	WebRTCFailed                    EventName = "webrtc.failed"
	WebRTCICEConnectionState        EventName = "webrtc.ice_connection_state"
	WebRTCICEConnected              EventName = "webrtc.ice_connected"
	WebRTCICEFailed                 EventName = "webrtc.ice_failed"
	WebRTCAudioTrackReceived        EventName = "webrtc.audio_track_received"
	WebRTCPeerQuality               EventName = "webrtc.peer_quality"
	WebRTCSelectedICECandidatePair  EventName = "webrtc.selected_ice_candidate_pair"
	WebRTCNegotiationOfferSent      EventName = "webrtc.negotiation_offer_sent"
	WebRTCNegotiationAnswerReceived EventName = "webrtc.negotiation_answer_received"
	WebRTCNegotiationRetryQueued    EventName = "webrtc.negotiation_retry_queued"
	WebRTCNegotiationRetrySent      EventName = "webrtc.negotiation_retry_sent"
	WebRTCICERestartDeferred        EventName = "webrtc.ice_restart_deferred"
)

var eventsByComponent = map[ComponentName][]EventName{
	ComponentCall: {
		CallStatus,
		CallReceived,
		CallRinging,
		CallStarted,
		CallMediaStarted,
		CallHangup,
		CallEnded,
		CallFailed,
		CallCancelled,
		CallOutboundRequested,
		CallOutboundDispatched,
		CallOutboundDispatchFailed,
		CallProviderAnswered,
		CallSessionConnected,
		CallAssistantLoaded,
		CallConversationCreated,
		CallContextSaved,
	},
	ComponentConversation: {
		ConversationBegin,
		ConversationResume,
		ConversationInitializing,
		ConversationInitialized,
		ConversationAuthenticationStarted,
		ConversationCompleted,
		ConversationCleanup,
		ConversationError,
		ConversationAgentStateChanged,
		ConversationModeSwitchFailed,
	},
	ComponentTurn: {
		TurnInterrupted,
		TurnChange,
		TurnStarted,
	},

	ComponentSTT: {
		STTInterim,
		STTCompleted,
		STTLowConfidence,
		STTClosed,
		STTError,
	},
	ComponentTTS: {
		TTSSpeaking,
		TTSCompleted,
		TTSInterrupted,
		TTSClosed,
		TTSError,
	},
	ComponentAgent: {
		AgentStarted,
		AgentCompleted,
		AgentDiscarded,
		AgentError,
		AgentTransitionTriggered,
		AgentTransitionMatched,
		AgentTransitionMissingEdge,
	},
	ComponentVAD: {
		VADSpeechStarted,
		VADSpeechEnded,
		VADClosed,
		VADError,
	},
	ComponentEOS: {
		EOSStarted,
		EOSCompleted,
		EOSClosed,
	},
	ComponentDenoise: {
		DenoiseClosed,
		DenoiseError,
	},
	ComponentTool: {
		ToolCallStarted,
		ToolCallCompleted,
		ToolCallFailed,
	},
	ComponentRecording: {
		RecordingStarted,
		RecordingCompleted,
	},
	ComponentSIP: {
		SIPTransferRequested,
		SIPTransferring,
		SIPRegisterStarted,
		SIPRegisterActive,
		SIPRegisterFailed,
		SIPRegisterRenewed,
		SIPRegisterRenewalFailed,
		SIPRegisterExpired,
		SIPUnregisterFailed,
	},
	ComponentWebRTC: {
		WebRTCConnecting,
		WebRTCConnected,
		WebRTCReconnecting,
		WebRTCDisconnected,
		WebRTCFailed,
		WebRTCICEConnectionState,
		WebRTCICEConnected,
		WebRTCICEFailed,
		WebRTCAudioTrackReceived,
		WebRTCPeerQuality,
		WebRTCSelectedICECandidatePair,
		WebRTCNegotiationOfferSent,
		WebRTCNegotiationAnswerReceived,
		WebRTCNegotiationRetryQueued,
		WebRTCNegotiationRetrySent,
		WebRTCICERestartDeferred,
	},
}

func (e EventName) String() string {
	return string(e)
}

func (e EventName) Component() ComponentName {
	value := e.String()
	idx := strings.IndexByte(value, '.')
	if idx <= 0 {
		return ComponentUnknown
	}
	return ComponentName(value[:idx])
}

func (e EventName) HasComponent(component ComponentName) bool {
	return e.Component() == component
}

func (e EventName) HasCategory(component ComponentName) bool {
	return e.HasComponent(component)
}

func (e EventName) IsKnown() bool {
	for _, known := range Events(e.Component()) {
		if known == e {
			return true
		}
	}
	return false
}

func Events(component ComponentName) []EventName {
	return append([]EventName(nil), eventsByComponent[component]...)
}

func CallEvents() []EventName {
	return Events(ComponentCall)
}

func ConversationEvents() []EventName {
	return Events(ComponentConversation)
}

func AllEvents() []EventName {
	var total int
	for _, events := range eventsByComponent {
		total += len(events)
	}
	all := make([]EventName, 0, total)
	for _, events := range eventsByComponent {
		all = append(all, events...)
	}
	return all
}
