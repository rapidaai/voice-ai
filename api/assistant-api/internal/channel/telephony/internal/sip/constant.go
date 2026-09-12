// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import "time"

// SIP defaults identify the provider and fill optional call configuration.
const (
	Provider               = "sip"
	WebhookEvent           = "webhook"
	DefaultOutboundSIPPort = 5060
	DefaultRingtone        = "ringtone_us"
)

// Realtime media constants preserve 20 ms SIP audio frames and bounded queues.
const (
	RealtimeInputChannelCapacity   = 1000
	BridgeRecordingChannelCapacity = 100
	ChunkDuration                  = 20 * time.Millisecond
	MulawFrameSize                 = 160
	MulawSilenceByte               = 0xFF
	Linear16BytesPerMs             = 32
	BridgeOutputFrameSize          = Linear16BytesPerMs * 20
)

// Outbound failure reasons provide stable values for call status reporting.
const (
	OutboundFailureReasonInvalidConfiguration OutboundFailureReason = "sip_outbound_invalid_configuration"
	OutboundFailureReasonServerNotInitialized OutboundFailureReason = "sip_outbound_server_not_initialized"
	OutboundFailureReasonServerNotRunning     OutboundFailureReason = "sip_outbound_server_not_running"
	OutboundFailureReasonHealthGateFailed     OutboundFailureReason = "sip_outbound_health_gate_failed"
	OutboundFailureReasonSetupFailed          OutboundFailureReason = "sip_outbound_setup_failed"
)
