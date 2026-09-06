// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import "errors"

// General SIP runtime errors.
var (
	ErrInvalidConfig              = errors.New("invalid SIP configuration")
	ErrSessionNotFound            = errors.New("SIP session not found")
	ErrSessionClosed              = errors.New("SIP session is closed")
	ErrRTPNotInitialized          = errors.New("RTP handler not initialized")
	ErrRTPHandlerStopped          = errors.New("RTP handler is stopped")
	ErrRTPMediaTimeout            = errors.New("RTP media timeout")
	ErrRTPOutputQueueFull         = errors.New("RTP output queue is full")
	ErrRTPPortRangeExhausted      = errors.New("no RTP ports available")
	ErrSDPParseFailed             = errors.New("failed to parse SDP")
	ErrCodecNotSupported          = errors.New("codec not supported")
	ErrConnectionFailed           = errors.New("SIP connection failed")
	ErrAuthRequired               = errors.New("SIP auth required but credentials are missing")
	ErrOutboundFromUserRequired   = errors.New("outbound From user is required")
	ErrInboundACKTimeout          = errors.New("inbound ACK timeout")
	ErrInboundInviteCancelled     = errors.New("inbound INVITE cancelled")
	ErrInboundAnswerPolicyTimeout = errors.New("inbound answer policy timeout")
	ErrBridgeLifecycleRejected    = errors.New("bridge lifecycle transition rejected")
	ErrInvalidCallRoute           = errors.New("invalid SIP call route")
	ErrMiddlewareChainIncomplete  = errors.New("SIP middleware chain incomplete")
	ErrPhoneDeploymentRequired    = errors.New("SIP phone deployment is required")
	ErrVaultResolverRequired      = errors.New("SIP vault resolver is required")
	ErrCredentialIDRequired       = errors.New("SIP credential ID is required")
	ErrVaultCredentialResolution  = errors.New("SIP vault credential resolution failed")
	ErrVaultConfigInvalid         = errors.New("SIP vault configuration is invalid")
	ErrSIPCallCapacityExceeded    = errors.New("SIP call capacity exceeded")
	ErrSIPCallRateExceeded        = errors.New("SIP call setup rate exceeded")
)

// Registration errors.
var (
	ErrRegistrationFailed  = errors.New("SIP registration failed")
	ErrRegistrationExpired = errors.New("SIP registration expired")
	ErrDIDNotRegistered    = errors.New("DID is not registered")
	ErrMissingDID          = errors.New("DID is required for registration")
	ErrMissingServer       = errors.New("SIP server is required for registration")
	ErrAuthFailed          = errors.New("SIP authentication failed")
	ErrPermanentFailure    = errors.New("SIP registration permanently rejected")
)

// Media setup errors.
var (
	ErrInboundMediaNotPrepared  = errors.New("inbound media is not prepared")
	ErrInboundMediaNoSession    = errors.New("inbound media requires a session")
	ErrOutboundMediaNotPrepared = errors.New("outbound media is not prepared")
	ErrOutboundMediaNoSession   = errors.New("outbound media requires a session")
)

// RTP validation and packet parsing errors.
var (
	errRTPConfigRequired              = errors.New("rtp config is required")
	errRTPCreateSSRC                  = errors.New("failed to create rtp ssrc")
	errRTPCreateSocket                = errors.New("failed to create rtp socket")
	errRTPInvalidConfig               = errors.New("invalid configuration")
	errRTPInvalidLocalAddressPort     = errors.New("invalid local_address.port")
	errRTPInvalidPacketLength         = errors.New("invalid packet length")
	errRTPInvalidPacketizationTime    = errors.New("invalid packetization time")
	errRTPInvalidPaddingLength        = errors.New("invalid rtp padding length")
	errRTPLocalAddressIPRequired      = errors.New("local_address.ip is required")
	errRTPPacketShortCSRCHeader       = errors.New("packet shorter than csrc header")
	errRTPPacketShortExtension        = errors.New("packet shorter than rtp extension header")
	errRTPPacketShortExtensionPayload = errors.New("packet shorter than rtp extension payload")
	errRTPPacketTooSmall              = errors.New("packet too small")
	errRTPPaddingNoPayload            = errors.New("rtp padding has no payload")
	errRTPPortRangeInvalidOrder       = errors.New("rtp_port_range_start must be less than or equal to rtp_port_range_end")
	errRTPPortRangeRequired           = errors.New("rtp port range is required when local_address.port is not specified")
	errRTPPortRangeValue              = errors.New("rtp port range exceeds max port")
	errRTPUnsupportedVersion          = errors.New("unsupported rtp version")
)
