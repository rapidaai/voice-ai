// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package cartesia_internal

type TextToSpeechVoice struct {
	Mode string `json:"mode"`
	ID   string `json:"id"`
}

type TextToSpeechExperimentalControls struct {
	Speed   string   `json:"speed"`
	Emotion []string `json:"emotion"`
}

type TextToSpeechOutputFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

type TextToSpeechInput struct {
	ModelID              string                           `json:"model_id"`
	ContextID            string                           `json:"context_id"`
	Transcript           string                           `json:"transcript"`
	Voice                TextToSpeechVoice                `json:"voice"`
	ExperimentalControls TextToSpeechExperimentalControls `json:"__experimental_controls"`
	OutputFormat         TextToSpeechOutputFormat         `json:"output_format"`
	Language             string                           `json:"language"`
	Continue             bool                             `json:"continue"`
	AddTimestamps        bool                             `json:"add_timestamps"`
}

type TextToSpeechOuput struct {
	Type       string `json:"type"`
	Data       string `json:"data"`
	Done       bool   `json:"done"`
	StatusCode int    `json:"status_code"`
	ContextID  string `json:"context_id"`
}

type TranscriptWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type SpeechToTextOutput struct {
	Type      string           `json:"type"`
	IsFinal   bool             `json:"is_final"`
	RequestID string           `json:"request_id"`
	Text      string           `json:"text"`
	Duration  float64          `json:"duration"`
	Language  string           `json:"language"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Words     []TranscriptWord `json:"words"`
}
