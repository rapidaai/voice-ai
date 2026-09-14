// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package webrtc_internal

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
)

// Opus audio constants (WebRTC standard: 48kHz)
const (
	OpusSampleRate        = 48000
	OpusFrameDuration     = 20
	OpusFrameSamples      = 960
	OpusPCMBytesPerSample = 2
	OpusFrameBytes        = OpusFrameSamples * OpusPCMBytesPerSample
	OpusChannels          = 2
	OpusPayloadType       = 111
	OpusSDPFmtpLine       = "minptime=10;useinbandfec=1;stereo=0;sprop-stereo=0;ptime=20"
)

// Audio RTCP feedback negotiated for browser WebRTC Opus.
const (
	RTCPFeedbackNACK = "nack"
)

// WebRTCOutputPCM16kFrameBytes is the 20ms PCM16k frame size used for assistant audio.
var WebRTCOutputPCM16kFrameBytes = internal_audio.BytesPerMs(internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG) * OpusFrameDuration

// Opus codec tuning for Rapida voice audio.
const (
	OpusVoiceChannels             = 1
	OpusVoiceBitrate              = 32000
	OpusVoiceComplexity           = 8
	OpusExpectedPacketLossPercent = 10
	OpusEncoderOutputMaxBytes     = 1000
	OpusMaxFrameSamples           = 5760
)

// Channel and buffer sizes
const (
	InternalPCM16BytesPerMs    = 32
	InputBufferThresholdMs     = 40
	InputChannelSize           = 500
	OutputChannelSize          = 1500
	PeerEventChannelSize       = 128
	MediaLifecycleChannelSize  = 32
	WebRTCOperationChannelSize = 128
	RTPBufferSize              = 1500
	MaxConsecutiveReadErrors   = 50
	InputBufferThreshold       = InternalPCM16BytesPerMs * InputBufferThresholdMs

	OutputPaceInterval             = OpusFrameDuration // milliseconds (20ms per frame)
	OutputAudioQueueMaxFrames      = 4000
	PendingRemoteICECandidateLimit = 256
)

// Peer and media timing constants.
const (
	OutputPaceDuration             = time.Duration(OutputPaceInterval) * time.Millisecond
	OutputHealthReportInterval     = 5 * time.Second
	HealthWatchdogInterval         = 5 * time.Second
	HandshakeDeadlineCheckInterval = 500 * time.Millisecond
	SignalingAnswerDeadline        = 10 * time.Second
	ICEConnectedDeadline           = 15 * time.Second
	PeerConnectedDeadline          = 20 * time.Second
	ICEDisconnectedTimeout         = 10 * time.Second
	ICEFailedTimeout               = 5 * time.Second
	ICEKeepaliveInterval           = 2 * time.Second
	DTLSRetransmissionInterval     = 100 * time.Millisecond
	DTLSHandshakeTimeout           = time.Minute
	WebTalkSuccessCode             = 200
	OutputAudioDropOldestSize      = 1
	OutputAudioQueueEmptySize      = 0
	ICERestartAttemptLimit         = 1
	MediaRestartAttemptLimit       = 1
)

// Health watchdog thresholds.
const (
	ConnectedNoUserAudioThreshold  = 10 * time.Second
	AssistantQueuedNoSendThreshold = 2 * time.Second
	RTCPFeedbackMissingThreshold   = 15 * time.Second
	HealthWatchdogEventCooldown    = 30 * time.Second
	RepeatedWriteFailuresThreshold = 3
	MillisecondsPerSecond          = 1000
)

// Conversational WebRTC quality thresholds.
const (
	QualityGoodPacketLossPercent = 2.0
	QualityPoorPacketLossPercent = 10.0
	QualityGoodJitterMs          = 30.0
	QualityPoorJitterMs          = 80.0
	QualityGoodRoundTripMs       = 250
	QualityPoorRoundTripMs       = 600
)

// RTCP timing constants.
const (
	RTCPFractionLostDenominator = 256
	RTCPCompactNTPUnitsPerSec   = 65536
	RTCPPercentMultiplier       = 100
	RTCPNTPUnixEpochOffset      = 2208988800
	RTCPNTPFractionUnits        = 1 << 32
	RTCPCompactNTPSecondsMask   = 0xFFFF
	RTCPCompactNTPFractionShift = 16
)

// WebRTC event names and queue policy values.
const (
	EventOutputQueueOverflow         = "output_queue_overflow"
	EventOutputQueueCleared          = "output_queue_cleared"
	EventOutputPacerHealth           = "output_pacer_health"
	EventOutputSendError             = "output_send_error"
	EventPeerQuality                 = "transport_quality"
	EventSelectedICECandidatePair    = "selected_ice_candidate_pair"
	EventConnectedNoUserAudio        = "connected_no_user_audio"
	EventAssistantAudioQueuedNotSent = "assistant_audio_queued_not_sent"
	EventRTCPFeedbackMissing         = "rtcp_feedback_missing"
	EventRepeatedWriteFailures       = "repeated_write_failures"
	EventICEConnectionState          = "ice_connection_state"
	EventICERestarting               = "ice_restarting"
	EventNegotiationOfferSent        = "negotiation_offer_sent"
	EventNegotiationAnswerReceived   = "negotiation_answer_received"
	EventNegotiationRetryQueued      = "negotiation_retry_queued"
	EventNegotiationRetrySent        = "negotiation_retry_sent"
	EventICERestartDeferred          = "ice_restart_deferred"
	EventHandshakeDeadlineExceeded   = "handshake_deadline_exceeded"
	EventMediaSessionRestarting      = "media_session_restarting"

	OutputQueuePolicyDropOldest        = "drop_oldest"
	OutputQueueClearReasonFlush        = "flush_audio"
	OutputQueueClearReasonInterruption = "interruption"
)

// WebRTC event data keys.
const (
	DataType                        = "type"
	DataSessionID                   = "session_id"
	DataMediaSessionID              = "media_session_id"
	DataOperation                   = "operation"
	DataICERestart                  = "ice_restart"
	DataRetryPending                = "retry_pending"
	DataICELatencyMs                = "ice_latency_ms"
	DataReason                      = "reason"
	DataCodec                       = "codec"
	DataPolicy                      = "policy"
	DataClearedFrames               = "cleared_frames"
	DataRemainingQueueFrames        = "remaining_queue_frames"
	DataTicks                       = "ticks"
	DataLateTicks                   = "late_ticks"
	DataActiveTicks                 = "active_ticks"
	DataIdleTicks                   = "idle_ticks"
	DataSendErrors                  = "send_errors"
	DataSendErrorsDelta             = "send_errors_delta"
	DataTotalSendErrors             = "total_send_errors"
	DataIdleRatio                   = "idle_ratio"
	DataPendingAudioFrames          = "pending_audio_frames"
	DataPendingAudioOldestAgeMs     = "pending_audio_oldest_age_ms"
	DataLastAssistantFrameSentMsAgo = "last_assistant_frame_sent_ms_ago"
	DataTotalDroppedFrames          = "total_dropped_frames"
	DataReceiverReports             = "receiver_reports"
	DataPacketLossFraction          = "packet_loss_fraction"
	DataPacketLossPercent           = "packet_loss_percent"
	DataPacketLossTotal             = "packet_loss_total"
	DataJitterMs                    = "jitter_ms"
	DataLastFeedbackMsAgo           = "last_feedback_ms_ago"
	DataLastFeedbackUnixMs          = "last_feedback_unix_ms"
	DataLastFeedbackAvailable       = "last_feedback_available"
	DataRoundTripTimePresent        = "round_trip_time_present"
	DataRoundTripTimeMs             = "round_trip_time_ms"
	DataQualityState                = "quality_state"
	DataCandidatePairID             = "candidate_pair_id"
	DataLocalCandidateType          = "local_candidate_type"
	DataLocalProtocol               = "local_protocol"
	DataRemoteCandidateType         = "remote_candidate_type"
	DataRemoteProtocol              = "remote_protocol"
	DataCandidatePairRTTMs          = "candidate_pair_rtt_ms"
	DataAvailableOutgoingBitrateBps = "available_outgoing_bitrate_bps"
	DataConnectedMs                 = "connected_ms"
	DataThresholdMs                 = "threshold_ms"
	DataReadErrors                  = "read_errors"
	DataRTPEmptyPayloads            = "rtp_empty_payloads"
	DataRTPParseFailures            = "rtp_parse_failures"
	DataOpusDecodeFailures          = "opus_decode_failures"
	DataConsecutiveWriteFailures    = "consecutive_write_failures"
	DataTotalWriteFailures          = "total_write_failures"
	DataLastFailureMsAgo            = "last_failure_ms_ago"
	DataFailureThreshold            = "failure_threshold"
	DataDroppedFrames               = "dropped_frames"
	DataLimitFrames                 = "limit_frames"
	DataQueueDepthFrames            = "queue_depth_frames"
	DataICEConnectionState          = "ice_connection_state"
	DataPeerConnectionState         = "peer_connection_state"
	DataDeadline                    = "deadline"
	DataDeadlineMs                  = "deadline_ms"
	DataElapsedMs                   = "elapsed_ms"
	DataRestartAttempt              = "restart_attempt"
	DataRestartLimit                = "restart_limit"
)

// WebRTC event reason values.
const (
	ReasonICEFailed             = "ice_failed"
	ReasonPeerDisconnected      = "peer_disconnected"
	ReasonPeerFailed            = "peer_failed"
	ReasonRemoteAnswerDeadline  = "remote_answer_deadline"
	ReasonICEConnectedDeadline  = "ice_connected_deadline"
	ReasonPeerConnectedDeadline = "peer_connected_deadline"
	ReasonConnectedNoUserAudio  = "connected_no_user_audio"
)

// Quality state values emitted for conversation-level WebRTC quality.
const (
	QualityStateExcellent = "excellent"
	QualityStateGood      = "good"
	QualityStatePoor      = "poor"
	QualityStateLost      = "lost"
)

// Pion ICE state string values used in transport events.
const (
	ICEStateChecking  = "checking"
	ICEStateConnected = "connected"
	ICEStateCompleted = "completed"
	ICEStateFailed    = "failed"
)

// Config holds WebRTC configuration
type Config struct {
	ICEServers         []ICEServer
	ICETransportPolicy string // "all" or "relay"
}

const (
	ICETransportPolicyAll   = "all"
	ICETransportPolicyRelay = "relay"
)

type MediaState int32

const (
	MediaStateText MediaState = iota
	MediaStateAudioNegotiating
	MediaStateAudioConnected
)

type NegotiationState int32

const (
	NegotiationStateIdle NegotiationState = iota
	NegotiationStateOfferSent
	NegotiationStateRetryPending
)

type MediaLifecycleEventKind int

const (
	MediaLifecycleEventRestart MediaLifecycleEventKind = iota + 1
	MediaLifecycleEventRecover
)

type MediaLifecycleEvent struct {
	Kind           MediaLifecycleEventKind
	MediaSessionID uint64
	Reason         string
	RequestedAt    time.Time
}

// WebRTCOperationKind identifies serialized WebRTC signaling and ICE mutations.
type WebRTCOperationKind int

const (
	WebRTCOperationSendOffer WebRTCOperationKind = iota + 1
	WebRTCOperationApplyRemoteAnswer
	WebRTCOperationAddRemoteICECandidate
	WebRTCOperationSendLocalICECandidate
	WebRTCOperationRestartICE
	WebRTCOperationICEGatheringComplete
)

func (k WebRTCOperationKind) String() string {
	switch k {
	case WebRTCOperationSendOffer:
		return "send_offer"
	case WebRTCOperationApplyRemoteAnswer:
		return "apply_remote_answer"
	case WebRTCOperationAddRemoteICECandidate:
		return "add_remote_ice_candidate"
	case WebRTCOperationSendLocalICECandidate:
		return "send_local_ice_candidate"
	case WebRTCOperationRestartICE:
		return "restart_ice"
	case WebRTCOperationICEGatheringComplete:
		return "ice_gathering_complete"
	default:
		return "unknown"
	}
}

// WebRTCOperation carries one ordered WebRTC mutation for the operation loop.
type WebRTCOperation struct {
	Kind               WebRTCOperationKind
	MediaSessionID     uint64
	Reason             string
	RequestedAt        time.Time
	OfferOptions       *pionwebrtc.OfferOptions
	SignalMediaConfig  bool
	LocalICECandidate  pionwebrtc.ICECandidateInit
	RemoteAnswerSDP    string
	RemoteICECandidate pionwebrtc.ICECandidateInit
}

type WebRTCDeferredICERestart struct {
	MediaSessionID uint64
	Reason         string
	RequestedAt    time.Time
}

// WebRTCAudioBufferState owns input and output PCM buffers for a WebRTC streamer.
type WebRTCAudioBufferState struct {
	InputAudioBufferMu  sync.Mutex
	InputAudioBuffer    *bytes.Buffer
	OutputAudioBufferMu sync.Mutex
	OutputAudioBuffer   *bytes.Buffer
}

type WebRTCRemoteAudioTrack struct {
	TrackCodec     pionwebrtc.RTPCodecParameters
	ReceiverCodecs []pionwebrtc.RTPCodecParameters
}

func (t WebRTCRemoteAudioTrack) SelectedCodec() (pionwebrtc.RTPCodecParameters, bool) {
	if t.TrackCodec.MimeType != "" {
		return t.TrackCodec, true
	}
	for _, receiverCodec := range t.ReceiverCodecs {
		if strings.EqualFold(receiverCodec.MimeType, pionwebrtc.MimeTypeOpus) {
			return receiverCodec, true
		}
	}
	if len(t.ReceiverCodecs) > 0 {
		return t.ReceiverCodecs[0], true
	}
	return pionwebrtc.RTPCodecParameters{}, false
}

type PeerEventKind int

const (
	SignalEventClientMessage PeerEventKind = iota + 1
	PeerEventStateChanged
	PeerEventICEConnectionStateChanged
)

type PeerEvent struct {
	Kind                  PeerEventKind
	MediaSessionID        uint64
	SignalClientMessage   *protos.ClientSignaling
	PeerState             pionwebrtc.PeerConnectionState
	PeerStateChangedAt    time.Time
	PeerICEState          pionwebrtc.ICEConnectionState
	PeerICEStateChangedAt time.Time
}

// SessionState owns WebRTC lifecycle flags shared across goroutines.
type SessionState struct {
	Scope                             observability.Scope
	closeStarted                      atomic.Bool
	peerConnected                     atomic.Bool
	mediaState                        atomic.Int32
	negotiationState                  atomic.Int32
	negotiationRetryICE               atomic.Bool
	iceGatheringActive                atomic.Bool
	activeMediaSessionID              atomic.Uint64
	pacedAssistantFrameMediaSessionID atomic.Uint64
	remoteAudioReaderMediaSessionID   atomic.Uint64
	outputAudioDroppedFrames          atomic.Uint64
	iceRestartAttempts                atomic.Uint64
	mediaRestartAttempts              atomic.Uint64
	deferredICERestartMu              sync.Mutex
	deferredICERestart                WebRTCDeferredICERestart
}

func (s *SessionState) ChangeScope(scope observability.Scope) {
	s.Scope = scope
}

func (s *SessionState) BeginClose() bool {
	return s.closeStarted.CompareAndSwap(false, true)
}

func (s *SessionState) CloseStarted() bool {
	return s.closeStarted.Load()
}

func (s *SessionState) SetPeerConnected(connected bool) {
	s.peerConnected.Store(connected)
}

func (s *SessionState) PeerConnected() bool {
	return s.peerConnected.Load()
}

func (s *SessionState) SetMediaState(state MediaState) {
	s.mediaState.Store(int32(state))
}

func (s *SessionState) MediaState() MediaState {
	return MediaState(s.mediaState.Load())
}

func (s *SessionState) TryStartRemoteAudioReader(mediaSessionID uint64) bool {
	return s.remoteAudioReaderMediaSessionID.CompareAndSwap(0, mediaSessionID)
}

func (s *SessionState) RemoteAudioReaderMediaSessionID() uint64 {
	return s.remoteAudioReaderMediaSessionID.Load()
}

func (s *SessionState) ResetRemoteAudioReader() {
	s.remoteAudioReaderMediaSessionID.Store(0)
}

func (s *SessionState) InvalidateMediaSession() uint64 {
	s.SetPeerConnected(false)
	s.SetMediaState(MediaStateText)
	s.SetICEGatheringActive(false)
	s.ClearDeferredICERestart()
	s.ResetRemoteAudioReader()
	s.ResetNegotiation()
	return s.activeMediaSessionID.Add(1)
}

func (s *SessionState) StartMediaSession() uint64 {
	s.SetPeerConnected(false)
	s.SetMediaState(MediaStateAudioNegotiating)
	s.SetICEGatheringActive(false)
	s.ClearDeferredICERestart()
	s.ResetRemoteAudioReader()
	s.ResetNegotiation()
	return s.activeMediaSessionID.Add(1)
}

func (s *SessionState) ActiveMediaSessionID() uint64 {
	return s.activeMediaSessionID.Load()
}

func (s *SessionState) IsActiveMediaSession(mediaSessionID uint64) bool {
	return mediaSessionID == s.activeMediaSessionID.Load()
}

func (s *SessionState) StampPacedAssistantFrame(mediaSessionID uint64) {
	s.pacedAssistantFrameMediaSessionID.Store(mediaSessionID)
}

func (s *SessionState) PacedAssistantFrameMediaSessionID() uint64 {
	return s.pacedAssistantFrameMediaSessionID.Load()
}

func (s *SessionState) CanWritePacedAssistantFrame() bool {
	pacedMediaSessionID := s.pacedAssistantFrameMediaSessionID.Load()
	if pacedMediaSessionID == 0 {
		return true
	}
	return pacedMediaSessionID == s.activeMediaSessionID.Load() && s.PeerConnected()
}

func (s *SessionState) AddOutputAudioDroppedFrames(droppedFrames int) uint64 {
	return s.outputAudioDroppedFrames.Add(uint64(droppedFrames))
}

func (s *SessionState) OutputAudioDroppedFrames() uint64 {
	return s.outputAudioDroppedFrames.Load()
}

func (s *SessionState) BeginNegotiation(iceRestart bool) (bool, bool) {
	for {
		state := NegotiationState(s.negotiationState.Load())
		switch state {
		case NegotiationStateIdle:
			if s.negotiationState.CompareAndSwap(int32(NegotiationStateIdle), int32(NegotiationStateOfferSent)) {
				s.negotiationRetryICE.Store(false)
				return true, false
			}
		case NegotiationStateOfferSent:
			if s.negotiationState.CompareAndSwap(int32(NegotiationStateOfferSent), int32(NegotiationStateRetryPending)) {
				if iceRestart {
					s.negotiationRetryICE.Store(true)
				}
				return false, true
			}
		case NegotiationStateRetryPending:
			if iceRestart {
				s.negotiationRetryICE.Store(true)
			}
			return false, true
		default:
			s.ResetNegotiation()
		}
	}
}

func (s *SessionState) CompleteNegotiation() (bool, bool) {
	state := NegotiationState(s.negotiationState.Swap(int32(NegotiationStateIdle)))
	retryICE := s.negotiationRetryICE.Swap(false)
	return state == NegotiationStateRetryPending, retryICE
}

func (s *SessionState) NegotiationState() NegotiationState {
	return NegotiationState(s.negotiationState.Load())
}

func (s *SessionState) NegotiationRetryICE() bool {
	return s.negotiationRetryICE.Load()
}

func (s *SessionState) ResetNegotiation() {
	s.negotiationState.Store(int32(NegotiationStateIdle))
	s.negotiationRetryICE.Store(false)
}

func (s *SessionState) SetICEGatheringActive(active bool) {
	s.iceGatheringActive.Store(active)
}

func (s *SessionState) ICEGatheringActive() bool {
	return s.iceGatheringActive.Load()
}

func (s *SessionState) DeferICERestart(restart WebRTCDeferredICERestart) {
	if restart.MediaSessionID == 0 {
		return
	}
	s.deferredICERestartMu.Lock()
	defer s.deferredICERestartMu.Unlock()
	s.deferredICERestart = restart
}

func (s *SessionState) DeferredICERestartPending(mediaSessionID uint64) bool {
	if mediaSessionID == 0 {
		return false
	}
	s.deferredICERestartMu.Lock()
	defer s.deferredICERestartMu.Unlock()
	return s.deferredICERestart.MediaSessionID == mediaSessionID
}

func (s *SessionState) TakeDeferredICERestart(mediaSessionID uint64) (WebRTCDeferredICERestart, bool) {
	if mediaSessionID == 0 {
		return WebRTCDeferredICERestart{}, false
	}
	s.deferredICERestartMu.Lock()
	defer s.deferredICERestartMu.Unlock()
	if s.deferredICERestart.MediaSessionID != mediaSessionID {
		return WebRTCDeferredICERestart{}, false
	}
	deferredICERestart := s.deferredICERestart
	s.deferredICERestart = WebRTCDeferredICERestart{}
	return deferredICERestart, true
}

func (s *SessionState) ClearDeferredICERestart() {
	s.deferredICERestartMu.Lock()
	defer s.deferredICERestartMu.Unlock()
	s.deferredICERestart = WebRTCDeferredICERestart{}
}

func (s *SessionState) ICERestartPending() bool {
	return s.NegotiationState() == NegotiationStateRetryPending && s.NegotiationRetryICE()
}

func (s *SessionState) ResetMediaRestartAttempts() {
	s.iceRestartAttempts.Store(0)
	s.mediaRestartAttempts.Store(0)
}

func (s *SessionState) ResetICERestartAttempts() {
	s.iceRestartAttempts.Store(0)
}

func (s *SessionState) TryBeginICERestart(limit uint64) (uint64, bool) {
	for {
		current := s.iceRestartAttempts.Load()
		if current >= limit {
			return current, false
		}
		next := current + 1
		if s.iceRestartAttempts.CompareAndSwap(current, next) {
			return next, true
		}
	}
}

func (s *SessionState) TryBeginMediaRestart(limit uint64) (uint64, bool) {
	for {
		current := s.mediaRestartAttempts.Load()
		if current >= limit {
			return current, false
		}
		next := current + 1
		if s.mediaRestartAttempts.CompareAndSwap(current, next) {
			return next, true
		}
	}
}

// MediaHealthState stores WebRTC media timing and analysis state.
type MediaHealthState struct {
	ICEStartedAt                time.Time
	OfferSentAt                 time.Time
	RemoteDescriptionSetAt      time.Time
	ICEConnectionState          string
	ICEConnectionStateChangedAt time.Time
	ICECheckingStartedAt        time.Time
	ICEConnectedAt              time.Time
	ICECompletedAt              time.Time
	ICEFailedAt                 time.Time
	PeerConnectedAt             time.Time
	PeerDisconnectedAt          time.Time
	PeerFailedAt                time.Time
	PeerClosedAt                time.Time

	FirstUserAudioReceivedAt       time.Time
	LastUserAudioReceivedAt        time.Time
	UserAudioReadErrors            uint64
	UserAudioConsecutiveReadErrors int
	UserAudioEmptyRTPPayloads      uint64
	UserAudioRTPUnmarshalFailures  uint64
	UserAudioOpusDecodeFailures    uint64
	UserAudioResampleFailures      uint64

	FirstAssistantAudioQueuedAt            time.Time
	LastAssistantAudioQueuedAt             time.Time
	LastAssistantFrameSentAt               time.Time
	AssistantFrameWriteFailures            uint64
	ConsecutiveAssistantFrameWriteFailures uint64
	LastAssistantFrameWriteFailureAt       time.Time

	ReceiverReports                       uint64
	LastReceiverReportAt                  time.Time
	LastReceiverReportFractionLost        uint8
	LastReceiverReportPacketLossPercent   float64
	LastReceiverReportTotalLost           uint32
	LastReceiverReportJitterMs            float64
	LastReceiverReportRoundTripTimeMs     int64
	LastReceiverReportRoundTripTimeUsable bool

	SelectedICECandidatePairID             string
	SelectedICECandidatePairChangedAt      time.Time
	SelectedICELocalCandidateType          string
	SelectedICELocalProtocol               string
	SelectedICERemoteCandidateType         string
	SelectedICERemoteProtocol              string
	SelectedICECandidatePairRTTMs          int64
	SelectedICEAvailableOutgoingBitrateBps int64
}

func (s *MediaHealthState) Reset() {
	*s = MediaHealthState{}
}

func (s *MediaHealthState) StartICE(startedAt time.Time) {
	*s = MediaHealthState{ICEStartedAt: startedAt}
}

func (s *MediaHealthState) StartICERestart(startedAt time.Time) {
	s.ICEStartedAt = startedAt
	s.OfferSentAt = time.Time{}
	s.RemoteDescriptionSetAt = time.Time{}
	s.ICEConnectionState = ""
	s.ICEConnectionStateChangedAt = time.Time{}
	s.ICECheckingStartedAt = time.Time{}
	s.ICEConnectedAt = time.Time{}
	s.ICECompletedAt = time.Time{}
	s.ICEFailedAt = time.Time{}
	s.PeerConnectedAt = time.Time{}
	s.PeerDisconnectedAt = time.Time{}
	s.PeerFailedAt = time.Time{}
	s.PeerClosedAt = time.Time{}
}

func (s *MediaHealthState) RecordOfferSent(sentAt time.Time) {
	s.OfferSentAt = sentAt
}

func (s *MediaHealthState) RecordRemoteDescriptionSet(setAt time.Time) {
	s.RemoteDescriptionSetAt = setAt
}

func (s *MediaHealthState) RecordICEConnectionState(state string, changedAt time.Time) {
	s.ICEConnectionState = state
	s.ICEConnectionStateChangedAt = changedAt
	switch state {
	case ICEStateChecking:
		if s.ICECheckingStartedAt.IsZero() {
			s.ICECheckingStartedAt = changedAt
		}
	case ICEStateConnected:
		s.ICEConnectedAt = changedAt
	case ICEStateCompleted:
		s.ICECompletedAt = changedAt
	case ICEStateFailed:
		s.ICEFailedAt = changedAt
	}
}

func (s *MediaHealthState) RecordPeerConnected(connectedAt time.Time) {
	s.PeerConnectedAt = connectedAt
}

func (s *MediaHealthState) RecordPeerDisconnected(disconnectedAt time.Time) {
	s.PeerDisconnectedAt = disconnectedAt
}

func (s *MediaHealthState) RecordPeerFailed(failedAt time.Time) {
	s.PeerFailedAt = failedAt
}

func (s *MediaHealthState) RecordPeerClosed(closedAt time.Time) {
	s.PeerClosedAt = closedAt
}

func (s MediaHealthState) HandshakeDeadlineExceeded(now time.Time) (string, time.Duration, time.Duration, bool) {
	if s.OfferSentAt.IsZero() {
		return "", 0, 0, false
	}
	elapsed := now.Sub(s.OfferSentAt)
	if s.RemoteDescriptionSetAt.IsZero() && elapsed >= SignalingAnswerDeadline {
		return ReasonRemoteAnswerDeadline, SignalingAnswerDeadline, elapsed, true
	}
	if s.ICEConnectedAt.IsZero() && s.ICECompletedAt.IsZero() && elapsed >= ICEConnectedDeadline {
		return ReasonICEConnectedDeadline, ICEConnectedDeadline, elapsed, true
	}
	if s.PeerConnectedAt.IsZero() && elapsed >= PeerConnectedDeadline {
		return ReasonPeerConnectedDeadline, PeerConnectedDeadline, elapsed, true
	}
	return "", 0, 0, false
}

func (s *MediaHealthState) RecordUserAudioReceived(receivedAt time.Time) {
	if s.FirstUserAudioReceivedAt.IsZero() {
		s.FirstUserAudioReceivedAt = receivedAt
	}
	s.LastUserAudioReceivedAt = receivedAt
	s.UserAudioConsecutiveReadErrors = 0
}

func (s *MediaHealthState) RecordUserAudioReadError(consecutiveErrors int) {
	s.UserAudioReadErrors++
	s.UserAudioConsecutiveReadErrors = consecutiveErrors
}

func (s *MediaHealthState) RecordUserAudioReadRecovered() {
	s.UserAudioConsecutiveReadErrors = 0
}

func (s *MediaHealthState) RecordUserAudioRTPUnmarshalFailure() {
	s.UserAudioRTPUnmarshalFailures++
}

func (s *MediaHealthState) RecordUserAudioEmptyRTPPayload() {
	s.UserAudioEmptyRTPPayloads++
}

func (s *MediaHealthState) RecordUserAudioOpusDecodeFailure() {
	s.UserAudioOpusDecodeFailures++
}

func (s *MediaHealthState) RecordUserAudioResampleFailure() {
	s.UserAudioResampleFailures++
}

func (s *MediaHealthState) RecordAssistantAudioQueued(queuedAt time.Time) {
	if s.FirstAssistantAudioQueuedAt.IsZero() {
		s.FirstAssistantAudioQueuedAt = queuedAt
	}
	s.LastAssistantAudioQueuedAt = queuedAt
}

func (s *MediaHealthState) RecordAssistantFrameSent(sentAt time.Time) {
	s.LastAssistantFrameSentAt = sentAt
	s.ConsecutiveAssistantFrameWriteFailures = 0
}

func (s *MediaHealthState) RecordAssistantFrameWriteFailure(failedAt time.Time) {
	s.AssistantFrameWriteFailures++
	s.ConsecutiveAssistantFrameWriteFailures++
	s.LastAssistantFrameWriteFailureAt = failedAt
}

func (s *MediaHealthState) RecordReceiverReport(
	receivedAt time.Time,
	fractionLost uint8,
	totalLost uint32,
	jitter uint32,
	lastSenderReport uint32,
	delaySinceLastSenderReport uint32,
) {
	s.ReceiverReports++
	s.LastReceiverReportAt = receivedAt
	s.LastReceiverReportFractionLost = fractionLost
	s.LastReceiverReportPacketLossPercent = float64(fractionLost) * RTCPPercentMultiplier / RTCPFractionLostDenominator
	s.LastReceiverReportTotalLost = totalLost
	s.LastReceiverReportJitterMs = float64(jitter) * MillisecondsPerSecond / OpusSampleRate
	s.LastReceiverReportRoundTripTimeUsable = false
	s.LastReceiverReportRoundTripTimeMs = 0

	if lastSenderReport == 0 || delaySinceLastSenderReport == 0 {
		return
	}

	nowNTPCompact := CompactNTP(receivedAt)
	roundTripUnits := nowNTPCompact - lastSenderReport - delaySinceLastSenderReport
	s.LastReceiverReportRoundTripTimeMs = int64(roundTripUnits) * MillisecondsPerSecond / RTCPCompactNTPUnitsPerSec
	s.LastReceiverReportRoundTripTimeUsable = true
}

func (s *MediaHealthState) RecordSelectedICECandidatePair(pair SelectedICECandidatePair, selectedAt time.Time) bool {
	changed := s.SelectedICECandidatePairID != pair.ID
	if changed {
		s.SelectedICECandidatePairChangedAt = selectedAt
	}
	s.SelectedICECandidatePairID = pair.ID
	s.SelectedICELocalCandidateType = pair.LocalCandidateType
	s.SelectedICELocalProtocol = pair.LocalProtocol
	s.SelectedICERemoteCandidateType = pair.RemoteCandidateType
	s.SelectedICERemoteProtocol = pair.RemoteProtocol
	s.SelectedICECandidatePairRTTMs = pair.CurrentRoundTripTimeMs
	s.SelectedICEAvailableOutgoingBitrateBps = pair.AvailableOutgoingBitrateBps
	return changed
}

func (s MediaHealthState) QualityState(now time.Time) string {
	if s.ConsecutiveAssistantFrameWriteFailures >= RepeatedWriteFailuresThreshold {
		return QualityStateLost
	}
	if !s.LastAssistantFrameSentAt.IsZero() &&
		(s.LastReceiverReportAt.IsZero() || now.Sub(s.LastReceiverReportAt) >= RTCPFeedbackMissingThreshold) &&
		now.Sub(s.LastAssistantFrameSentAt) >= RTCPFeedbackMissingThreshold {
		return QualityStateLost
	}
	if s.LastReceiverReportPacketLossPercent >= QualityPoorPacketLossPercent ||
		s.LastReceiverReportJitterMs >= QualityPoorJitterMs ||
		(s.LastReceiverReportRoundTripTimeUsable && s.LastReceiverReportRoundTripTimeMs >= QualityPoorRoundTripMs) {
		return QualityStatePoor
	}
	if s.LastReceiverReportPacketLossPercent >= QualityGoodPacketLossPercent ||
		s.LastReceiverReportJitterMs >= QualityGoodJitterMs ||
		(s.LastReceiverReportRoundTripTimeUsable && s.LastReceiverReportRoundTripTimeMs >= QualityGoodRoundTripMs) {
		return QualityStateGood
	}
	return QualityStateExcellent
}

func CompactNTP(t time.Time) uint32 {
	seconds := uint64(t.Unix()) + RTCPNTPUnixEpochOffset
	fraction := uint64(t.Nanosecond()) * RTCPNTPFractionUnits / uint64(time.Second)
	return uint32(seconds&RTCPCompactNTPSecondsMask)<<RTCPCompactNTPFractionShift |
		uint32(fraction>>RTCPCompactNTPFractionShift)
}

// OutputAudioFrame stores assistant audio waiting for paced WebRTC delivery.
type OutputAudioFrame struct {
	Audio    []byte
	QueuedAt time.Time
	Playback *OutputPlayback
	Terminal bool
}

// OutputPlayback is shared by one response's frames, under the streamer's output state lock.
type OutputPlayback struct {
	ID             string
	MediaSessionID uint64
	Generation     uint64
	HasAudio       bool
	TerminalQueued bool
	Sent           bool
	Failed         bool
}

// SelectedICECandidatePair summarizes the active network path without exposing IPs.
type SelectedICECandidatePair struct {
	ID                          string
	LocalCandidateType          string
	LocalProtocol               string
	RemoteCandidateType         string
	RemoteProtocol              string
	CurrentRoundTripTimeMs      int64
	AvailableOutgoingBitrateBps int64
}

// ICEServer represents a STUN/TURN server
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// DefaultConfig returns default WebRTC configuration
func DefaultConfig() *Config {
	return &Config{
		ICEServers: []ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
		},
		ICETransportPolicy: ICETransportPolicyAll,
	}
}

// ICECandidate represents an ICE candidate for signaling
type ICECandidate struct {
	Candidate        string
	SDPMid           string
	SDPMLineIndex    int
	UsernameFragment string
}
