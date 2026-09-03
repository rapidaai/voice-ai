// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type RTPPacket struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CC             uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Payload        []byte
}

type RTPFallbackAudioSource = internal_core.RTPFallbackAudioSource

type RTPHandler struct {
	inner *internal_core.RTPHandler

	codec          *Codec
	audioInChan    chan []byte
	audioOutChan   chan []byte
	flushAudioCh   chan struct{}
	fallbackSource RTPFallbackAudioSource
}

type RTPConfig = internal_core.RTPConfig
