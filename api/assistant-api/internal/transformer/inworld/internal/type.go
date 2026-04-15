// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package inworld_internal contains the wire types for Inworld's HTTP
// streaming TTS endpoint at https://api.inworld.ai/tts/v1/voice:stream.
//
// The endpoint accepts a single POST request per synthesis and replies with
// an NDJSON stream (one JSON object per line). Each response line carries
// either a result envelope with base64 audio, or a top-level error body.
// Closing the response body ends the synthesis — there is no explicit
// end-of-stream sentinel.
package inworld_internal

// AudioConfig controls Inworld's output audio encoding and sample rate.
// LINEAR16 @ 16000 Hz matches Rapida's pcm_16000 expectation used by
// elevenlabs and cartesia transformers.
type AudioConfig struct {
	AudioEncoding   string `json:"audio_encoding"`
	SampleRateHertz int    `json:"sample_rate_hertz"`
}

// StreamRequest is the body posted to voice:stream. One request synthesizes
// one piece of text — Rapida's aggregator chunks LLM deltas into sentences
// before they reach the TTS transformer, so one request per sentence is the
// natural granularity.
type StreamRequest struct {
	Text        string      `json:"text"`
	VoiceID     string      `json:"voice_id"`
	ModelID     string      `json:"model_id"`
	AudioConfig AudioConfig `json:"audio_config"`
}

// StreamChunk is one NDJSON line of the streaming response. Inworld nests
// the audio payload under `result.audioContent` (base64 in the encoding
// requested by AudioConfig). An error line has a top-level `error` body
// instead of a result.
type StreamChunk struct {
	Result *StreamResult `json:"result,omitempty"`
	Error  *ErrorBody    `json:"error,omitempty"`
}

// StreamResult is the per-chunk result envelope. audioContent is
// base64-encoded audio bytes in the encoding specified by AudioConfig.
type StreamResult struct {
	AudioContent string `json:"audioContent,omitempty"`
}

// ErrorBody is the body of a server-emitted error line.
type ErrorBody struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}
