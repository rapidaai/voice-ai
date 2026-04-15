// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package inworld_internal contains the wire types for the Inworld TTS
// bidirectional WebSocket protocol at
// wss://api.inworld.ai/tts/v1/voice:streamBidirectional.
//
// The protocol is a context-scoped, bidirectional stream. Clients send a
// create frame to open a context, one or more send_text frames to feed text,
// and a close_context frame to signal end-of-turn. The server replies with
// result.audioChunk frames carrying base64 audio until it emits
// result.contextClosed (or done:true) to terminate the context.
package inworld_internal

import "encoding/json"

// AudioConfig controls Inworld's output audio encoding and sample rate.
// LINEAR16 @ 16000 Hz matches Rapida's pcm_16000 expectation used by
// elevenlabs and cartesia transformers.
type AudioConfig struct {
	AudioEncoding   string `json:"audio_encoding"`
	SampleRateHertz int    `json:"sample_rate_hertz"`
}

// CreateBody is the body of a "create" frame — opens a new context with the
// selected voice/model/audio configuration. AutoMode lets the server decide
// when to flush the text buffer for minimal latency while keeping quality.
type CreateBody struct {
	VoiceID     string      `json:"voice_id"`
	ModelID     string      `json:"model_id"`
	AudioConfig AudioConfig `json:"audio_config"`
	AutoMode    bool        `json:"auto_mode,omitempty"`
}

// CreateRequest opens a context. Must be sent once per context_id before any
// send_text frames can be issued against that context.
type CreateRequest struct {
	ContextID string     `json:"context_id"`
	Create    CreateBody `json:"create"`
}

// SendTextBody is the body of a "send_text" frame. FlushContext is optional —
// when present, the server flushes the current buffer immediately rather than
// waiting for additional text. It is sent as an empty object per the Inworld
// protocol spec.
type SendTextBody struct {
	Text         string                 `json:"text"`
	FlushContext map[string]interface{} `json:"flush_context,omitempty"`
}

// SendTextRequest pushes a text chunk into the given context.
type SendTextRequest struct {
	ContextID string       `json:"context_id"`
	SendText  SendTextBody `json:"send_text"`
}

// CloseContextRequest signals end-of-turn. The server will drain any buffered
// audio, then emit result.contextClosed and terminate the context.
type CloseContextRequest struct {
	ContextID    string                 `json:"context_id"`
	CloseContext map[string]interface{} `json:"close_context"`
}

// AudioChunk is the nested audio payload of a result frame. audioContent is
// base64-encoded audio bytes in the encoding specified by the create frame.
type AudioChunk struct {
	AudioContent string `json:"audioContent"`
}

// Result is the envelope for server-sent messages.
type Result struct {
	// ContextID echoes the context the frame belongs to. Useful for logs.
	ContextID string `json:"contextId,omitempty"`

	// AudioChunk carries one audio chunk. Present on streaming audio frames.
	AudioChunk *AudioChunk `json:"audioChunk,omitempty"`

	// ContextClosed is non-nil on the final frame for a context. Its contents
	// are unused by the transformer — presence alone signals end-of-stream.
	ContextClosed map[string]interface{} `json:"contextClosed,omitempty"`

	// FlushCompleted acknowledges a mid-stream flush. The context is still
	// open and more audio may follow — it is NOT an end-of-stream signal.
	FlushCompleted map[string]interface{} `json:"flushCompleted,omitempty"`

	// Status is a google.rpc.Status object the server includes alongside
	// ack/close frames. Unused by the transformer — kept raw so unmarshalling
	// never fails on shape variance.
	Status json.RawMessage `json:"status,omitempty"`
}

// ErrorBody is the body of a server-emitted error frame.
type ErrorBody struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// InworldTTSResponse is the top-level server response frame. Exactly one of
// Result/Error is populated on any given frame; Done may appear instead as a
// terminal signal on some server paths.
type InworldTTSResponse struct {
	Result *Result    `json:"result,omitempty"`
	Error  *ErrorBody `json:"error,omitempty"`
	Done   *bool      `json:"done,omitempty"`
}
