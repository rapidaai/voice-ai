// Copyright (c) 2023-2026 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import (
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

// SIPRTPBridgeTarget exposes RTP output needed by SIP transfer media.
type SIPRTPBridgeTarget interface {
	WriteAudio([]byte) error
}

// SIPStreamer retains lifecycle capabilities used by the SIP call runtime.
type SIPStreamer interface {
	Streamer

	// StartAssistantOutput enables queued audio after SIP answer ownership is ready.
	StartAssistantOutput()

	// Close releases media resources; the call owner retains session lifecycle ownership.
	Close() error
}

// SIPTransferStreamer exposes media operations to the existing transfer owner.
type SIPTransferStreamer interface {
	Streamer

	SetTransferRequestHandler(func(targets []string, postTransferAction string))
	ConnectTransferMedia(target SIPRTPBridgeTarget, outputCodecName string)
	DisconnectTransferMedia()
	StopTransferRingback()
	ResumeAssistant()
	RecordTransferOperatorAudio([]byte)
	RecordTransferDurationMetric(durationMs string)
	SendTransferToolResult(contextID, toolID, toolName string, action protos.ToolCallAction, result map[string]string)
	SendTransferEvent(proto.Message)
}

// SIPCallStreamer combines the runtime and transfer capabilities returned by SIP.
type SIPCallStreamer interface {
	SIPStreamer
	SIPTransferStreamer
}
