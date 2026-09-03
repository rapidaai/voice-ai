// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type Codec = internal_core.Codec

var (
	CodecPCMU           = codecValueFromCore(internal_core.CodecPCMU)
	CodecPCMA           = codecValueFromCore(internal_core.CodecPCMA)
	CodecG722           = codecValueFromCore(internal_core.CodecG722)
	CodecTelephoneEvent = codecValueFromCore(internal_core.CodecTelephoneEvent)
)

var SupportedCodecs = []Codec{CodecPCMU, CodecPCMA}

type SDPDirection = internal_core.SDPDirection

const (
	SDPDirectionSendRecv SDPDirection = "sendrecv"
	SDPDirectionSendOnly SDPDirection = "sendonly"
	SDPDirectionRecvOnly SDPDirection = "recvonly"
	SDPDirectionInactive SDPDirection = "inactive"
)

type SDPMediaInfo = internal_core.SDPMediaInfo
type SDPConfig = internal_core.SDPConfig

func DefaultSDPConfig(localIP string, rtpPort int) *SDPConfig {
	coreConfig := internal_core.DefaultSDPConfig(localIP, rtpPort)
	return sdpConfigFromCore(coreConfig)
}

func GetCodecByPayloadType(pt uint8) *Codec {
	coreCodec := internal_core.GetCodecByPayloadType(pt)
	return codecFromCore(coreCodec)
}

func GetCodecByName(name string) *Codec {
	coreCodec := internal_core.GetCodecByName(name)
	return codecFromCore(coreCodec)
}

func codecFromCore(codec *internal_core.Codec) *Codec {
	if codec == nil {
		return nil
	}
	codecValue := codecValueFromCore(*codec)
	return &codecValue
}

func codecValueFromCore(codec internal_core.Codec) Codec {
	return Codec{
		Name:        codec.Name,
		PayloadType: codec.PayloadType,
		ClockRate:   codec.ClockRate,
		Channels:    codec.Channels,
	}
}

func codecToCore(codec *Codec) *internal_core.Codec {
	if codec == nil {
		return nil
	}
	return &internal_core.Codec{
		Name:        codec.Name,
		PayloadType: codec.PayloadType,
		ClockRate:   codec.ClockRate,
		Channels:    codec.Channels,
	}
}

func sdpInfoFromCore(info *internal_core.SDPMediaInfo) *SDPMediaInfo {
	if info == nil {
		return nil
	}
	return &SDPMediaInfo{
		ConnectionIP:   info.ConnectionIP,
		AudioPort:      info.AudioPort,
		PayloadTypes:   append([]uint8(nil), info.PayloadTypes...),
		PreferredCodec: codecFromCore(info.PreferredCodec),
		Direction:      SDPDirection(info.Direction),
	}
}

func sdpConfigFromCore(config *internal_core.SDPConfig) *SDPConfig {
	if config == nil {
		return nil
	}
	return &SDPConfig{
		SessionID:   config.SessionID,
		SessionName: config.SessionName,
		LocalIP:     config.LocalIP,
		RTPPort:     config.RTPPort,
		Codecs:      codecsFromCore(config.Codecs),
		PTime:       config.PTime,
	}
}

func sdpConfigToCore(config *SDPConfig) *internal_core.SDPConfig {
	if config == nil {
		return nil
	}
	return &internal_core.SDPConfig{
		SessionID:   config.SessionID,
		SessionName: config.SessionName,
		LocalIP:     config.LocalIP,
		RTPPort:     config.RTPPort,
		Codecs:      codecsToCore(config.Codecs),
		PTime:       config.PTime,
	}
}

func codecsFromCore(codecs []internal_core.Codec) []Codec {
	if len(codecs) == 0 {
		return nil
	}
	result := make([]Codec, 0, len(codecs))
	for _, codec := range codecs {
		result = append(result, codecValueFromCore(codec))
	}
	return result
}

func codecsToCore(codecs []Codec) []internal_core.Codec {
	if len(codecs) == 0 {
		return nil
	}
	result := make([]internal_core.Codec, 0, len(codecs))
	for _, codec := range codecs {
		result = append(result, *codecToCore(&codec))
	}
	return result
}
