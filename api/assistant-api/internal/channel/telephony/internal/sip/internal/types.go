// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"errors"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
)

const (
	Provider               = "sip"
	WebhookEvent           = "webhook"
	DefaultOutboundSIPPort = 5060
	DefaultRingtone        = "ringtone_us"

	AudioChannelSize = 100
	ChunkDuration    = 20 * time.Millisecond
	MulawFrameSize   = 160
	MulawSilenceByte = 0xFF

	Linear16BytesPerMs    = 32
	BridgeOutputFrameSize = Linear16BytesPerMs * 20
	InputBufferThreshold  = Linear16BytesPerMs * 40
)

type OutboundFailureReason string

const (
	OutboundFailureReasonInvalidConfiguration OutboundFailureReason = "sip_outbound_invalid_configuration"
	OutboundFailureReasonServerNotInitialized OutboundFailureReason = "sip_outbound_server_not_initialized"
	OutboundFailureReasonServerNotRunning     OutboundFailureReason = "sip_outbound_server_not_running"
	OutboundFailureReasonHealthGateFailed     OutboundFailureReason = "sip_outbound_health_gate_failed"
	OutboundFailureReasonSetupFailed          OutboundFailureReason = "sip_outbound_setup_failed"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

var (
	Rapida16kConfig = internal_audio.NewLinear16khzMonoAudioConfig()
	Mulaw8kConfig   = internal_audio.NewMulaw8khzMonoAudioConfig()
	Linear8kConfig  = internal_audio.NewLinear8khzMonoAudioConfig()

	ErrInboundCallerMissing           = errors.New("missing caller information")
	ErrSIPServerNotInitialized        = errors.New("SIP server not initialized")
	ErrSIPServerNotRunning            = errors.New("SIP server not running")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to mulaw 8kHz failed")
	ErrRTPOutputQueueFull             = sip_runtime.ErrRTPOutputQueueFull
)

type AudioProcessorConfig struct {
	RTPHandler rtpHandler
	Resampler  internal_type.AudioResampler
	PushInput  func(internal_type.Stream)
	Record     func(...observability.Record) error
	Ringtone   string
	Ambient    *internal_ambient.Config
}

type rtpHandler interface {
	internal_type.SIPRTPBridgeTarget
	AudioIn() <-chan []byte
	ClearFallbackAudioSource()
	FlushAudioOut()
	GetCodec() *sip_runtime.Codec
	LocalAddr() (string, int)
	SetFallbackAudioSource(sip_runtime.RTPFallbackAudioSource)
}
