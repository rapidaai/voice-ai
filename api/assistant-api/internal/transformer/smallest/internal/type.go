// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package smallest_internal

// TextToSpeechInput is the JSON body sent per-message to Smallest's Lightning
// streaming endpoint (wss://api.smallest.ai/waves/v1/tts/live).
type TextToSpeechInput struct {
	VoiceID    string  `json:"voice_id"`
	Text       string  `json:"text"`
	Model      string  `json:"model,omitempty"`
	Language   string  `json:"language,omitempty"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Speed      float64 `json:"speed,omitempty"`
	ContextID  string  `json:"context_id,omitempty"`
	Continue   bool    `json:"continue"`
	Flush      bool    `json:"flush,omitempty"`
}

// TextToSpeechOutputData is the frame-specific payload of a TTS response
// message; its shape depends on the parent message's Status.
type TextToSpeechOutputData struct {
	// Audio is base64-encoded PCM, present when Status == "chunk".
	Audio string `json:"audio"`

	// ID/Word/Start/End are present when Status == "word_timestamp".
	ID    int     `json:"id"`
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// TextToSpeechOutputError is one entry of the "errors" array on an
// error-status TTS response frame (e.g. a voice_id/model pairing rejected by
// the server).
type TextToSpeechOutputError struct {
	Code    string   `json:"code"`
	Path    []string `json:"path"`
	Message string   `json:"message"`
}

// TextToSpeechOutput mirrors Smallest's TtsResponseMessage. Status is the
// frame discriminator: "chunk" | "word_timestamp" | "complete" | "error".
// Message/Errors are only populated on "error" frames; the server sends
// exactly one such frame and then holds the connection open without ever
// sending "complete", so callers must treat "error" as terminal.
type TextToSpeechOutput struct {
	SessionID         string                    `json:"session_id"`
	RequestID         string                    `json:"request_id"`
	ExternalSessionID string                    `json:"external_session_id"`
	Status            string                    `json:"status"`
	Message           string                    `json:"message"`
	Errors            []TextToSpeechOutputError `json:"errors"`
	Data              TextToSpeechOutputData    `json:"data"`
}

// SpeechToTextWord is a per-word timestamp entry (present when
// word_timestamps=true was requested on connect).
type SpeechToTextWord struct {
	Word              string  `json:"word"`
	Start             float64 `json:"start"`
	End               float64 `json:"end"`
	Confidence        float64 `json:"confidence"`
	Speaker           int     `json:"speaker"`
	SpeakerConfidence float64 `json:"speaker_confidence"`
}

// SpeechToTextUtterance is a sentence-level segment (present when
// sentence_timestamps=true was requested on connect).
type SpeechToTextUtterance struct {
	Text    string  `json:"text"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker int     `json:"speaker"`
}

// SpeechToTextOutput mirrors Smallest's Pulse realtime WebSocket messages.
// Two shapes share this struct (extra fields are simply left zero-valued):
//   - Transcription messages (no "type" field): SessionID, Transcript,
//     IsFinal, IsLast, Words, Language, ...
//   - VAD events ("type": "speech_started" | "speech_ended"): Type, SessionID,
//     Timestamp.
type SpeechToTextOutput struct {
	Type             string                  `json:"type"`
	SessionID        string                  `json:"session_id"`
	Transcript       string                  `json:"transcript"`
	IsFinal          bool                    `json:"is_final"`
	IsLast           bool                    `json:"is_last"`
	FromFinalize     bool                    `json:"from_finalize"`
	Words            []SpeechToTextWord      `json:"words"`
	Utterances       []SpeechToTextUtterance `json:"utterances"`
	Language         string                  `json:"language"`
	Languages        []string                `json:"languages"`
	RedactedEntities []string                `json:"redacted_entities"`
	Timestamp        float64                 `json:"timestamp"`
}
