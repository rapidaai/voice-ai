// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	"github.com/rapidaai/protos"
)

// TalkInput defines the interface for incoming conversation messages from clients.
// It represents messages that can be sent to the assistant during a conversation stream,
// including initialization parameters, configuration updates, user messages, and metadata.
type Stream interface {
	// GetInitialization returns the conversation initialization message if present.
	// Contains initial setup parameters for the conversation stream.
	ProtoMessage()
}

// Streamer defines a bidirectional streaming interface for real-time conversation with the assistant.
// It manages the lifecycle of a conversation stream, allowing clients to send input messages
// and receive output responses asynchronously. The stream persists until explicitly closed
// or an error occurs.
type Streamer interface {
	// Context returns the context associated with this stream.
	// The context can be used to manage cancellation, timeouts, and deadlines.
	Context() context.Context

	// Recv receives the next output message from the stream.
	// It blocks until a message is available, the stream is closed, or an error occurs.
	// Returns the received message and any error encountered. If the stream is closed,
	// it should return (nil, io.EOF).
	Recv() (Stream, error)

	// Send sends an input message to the stream.
	// It returns an error if the send operation fails (e.g., stream closed, network error).
	Send(Stream) error
}

// SIPRTPBridgeTarget is the minimum RTP behavior needed to connect SIP bridge
// audio without coupling generic stream contracts to SIP infra packages.
type SIPRTPBridgeTarget interface {
	// WriteAudio writes one complete encoded frame to the RTP transport.
	WriteAudio([]byte) error
}

// SIPStreamer extends the generic streamer contract with SIP media behavior
// required directly by the SIP call runtime.
type SIPStreamer interface {
	Streamer

	// StartAssistantOutput opens assistant audio delivery after SIP answer ownership is ready.
	// Inbound calls use this to queue pre-answer audio without sending RTP early.
	StartAssistantOutput()

	// Close releases SIP streamer media resources and cancels stream context.
	// Session lifecycle ownership remains with the SIP call owner.
	Close() error
}

// SIPTransferStreamer exposes SIP transfer behavior used by the transfer owner.
// Keep transfer-specific methods out of the base SIPStreamer runtime contract.
type SIPTransferStreamer interface {
	Streamer

	// SetTransferRequestHandler registers the callback fired when assistant tooling requests transfer.
	// The callback owns SIP transfer orchestration outside the media streamer.
	SetTransferRequestHandler(func(targets []string, postTransferAction string))

	// ConnectTransferMedia connects caller audio to the answered transfer B-leg RTP output.
	// outputCodecName tells the media layer whether bridge audio needs codec conversion.
	ConnectTransferMedia(target SIPRTPBridgeTarget, outputCodecName string)

	// DisconnectTransferMedia disconnects active bridge RTP output from the streamer.
	// It must be safe to call during transfer teardown and session close.
	DisconnectTransferMedia()

	// StopTransferRingback stops locally generated transfer ringback audio.
	// Transfer lifecycle calls this once the B-leg is answered or cancelled.
	StopTransferRingback()

	// ResumeAssistant returns media ownership from bridge/ringback back to the assistant.
	// Implementations should leave the SIP call connected when AI resumes.
	ResumeAssistant()

	// RecordTransferOperatorAudio records transfer target audio into the conversation pipeline.
	// This is used only after bridge media is connected.
	RecordTransferOperatorAudio([]byte)

	// RecordTransferDurationMetric records the completed SIP bridge duration as a conversation metric.
	RecordTransferDurationMetric(durationMs string)

	// SendTransferToolResult reports transfer tool completion back to the assistant pipeline.
	// It preserves the original tool identifiers so the caller can correlate the result.
	SendTransferToolResult(contextID, toolID, toolName string, action protos.ToolCallAction, result map[string]string)

	// SendTransferEvent pushes a transfer event into the same queue used by transport media.
	// SIP transfer callbacks use it for structured telephony lifecycle events.
	SendTransferEvent(Stream)
}

// SIPCallStreamer combines SIP call runtime and transfer behavior returned by
// the SIP streamer constructor. Call owners can depend on narrower interfaces.
type SIPCallStreamer interface {
	SIPStreamer
	SIPTransferStreamer
}
