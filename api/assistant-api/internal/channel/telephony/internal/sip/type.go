// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
)

type OutboundFailureReason string

func (r OutboundFailureReason) String() string {
	return string(r)
}

var (
	Rapida16kConfig = internal_audio.NewLinear16khzMonoAudioConfig()
	Mulaw8kConfig   = internal_audio.NewMulaw8khzMonoAudioConfig()
	Linear8kConfig  = internal_audio.NewLinear8khzMonoAudioConfig()
)

type AudioProcessorConfig struct {
	RTPHandler rtpHandler
	Logger     commons.Logger
	Record     func(...observability.Record) error
	Ringtone   string
	Ambient    *internal_ambient.Config
}

type rtpHandler interface {
	internal_type.SIPRTPBridgeTarget
	GetCodec() *sip_runtime.Codec
	LocalAddress() sip_runtime.RTPAddress
	SetInboundAudioSink(func(sip_runtime.InboundAudioFrame))
}
