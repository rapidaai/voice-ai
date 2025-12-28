// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_end_of_speech

import (
	"context"
	"time"
)

type EndOfSpeechCallback func(context.Context, *EndOfSpeechResult) error

type EndOfSpeechResult struct {
	StartAt float64
	EndAt   float64
	Speech  string
}

func (s *EndOfSpeechResult) GetSpeechStartAt() float64 { return s.StartAt }
func (s *EndOfSpeechResult) GetSpeechEndAt() float64   { return s.EndAt }
func (s *EndOfSpeechResult) GetDuration() float64      { return s.EndAt - s.StartAt }
func (s *EndOfSpeechResult) GetSpeech() string         { return s.Speech }

type EndOfSpeechInput interface {
	GetMessage() string
	GetTime() time.Time
}

type SystemEndOfSpeechInput struct {
	EndOfSpeechInput
	Time time.Time
}

func (m *SystemEndOfSpeechInput) GetMessage() string {
	return "<|activity|>"
}

func (m *SystemEndOfSpeechInput) GetTime() time.Time {
	return m.Time
}

type STTEndOfSpeechInput struct {
	EndOfSpeechInput
	Message    string
	Time       time.Time
	IsComplete bool
}

func (m *STTEndOfSpeechInput) GetMessage() string {
	return m.Message
}

func (m *STTEndOfSpeechInput) GetTime() time.Time {
	return m.Time
}

type UserEndOfSpeechInput struct {
	EndOfSpeechInput
	Message string
	Time    time.Time
}

func (m *UserEndOfSpeechInput) GetMessage() string {
	return m.Message
}

func (m *UserEndOfSpeechInput) GetTime() time.Time {
	return m.Time
}

type EndOfSpeech interface {
	Name() string
	Analyze(ctx context.Context, s EndOfSpeechInput) error
	Close() error
}
