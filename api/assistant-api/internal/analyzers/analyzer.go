package internal_analyzers

import (
	"context"
	"time"
)

/*
Analyzer is the base interface for all analyzer types.
It defines the common method that all analyzers must implement.

Methods:
  - Name() string: Returns the name of the analyzer.
*/
type Analyzer[t any] interface {
	Name() string
	Analyze(ctx context.Context, s t) error
	Close() error
}

/*
TextAnalyzer is an interface for analyzers that work with string input.
It extends the base Analyzer interface.

Methods:
  - Name() string: Inherited from Analyzer, returns the name of the analyzer.
  - Analyze(s string): Performs analysis on the given string input.
*/
type TextAnalyzerInput interface {
	GetMessage() string
	GetTime() time.Time
}

type SystemTextAnalyzerInput struct {
	TextAnalyzerInput
	Time time.Time
}

func (m *SystemTextAnalyzerInput) GetMessage() string {
	return "<|activity|>"
}

func (m *SystemTextAnalyzerInput) GetTime() time.Time {
	return m.Time
}

type STTTextAnalyzerInput struct {
	TextAnalyzerInput
	Message    string
	Time       time.Time
	IsComplete bool
}

func (m *STTTextAnalyzerInput) GetMessage() string {
	return m.Message
}

func (m *STTTextAnalyzerInput) GetTime() time.Time {
	return m.Time
}

type UserTextAnalyzerInput struct {
	TextAnalyzerInput
	Message string
	Time    time.Time
}

func (m *UserTextAnalyzerInput) GetMessage() string {
	return m.Message
}

func (m *UserTextAnalyzerInput) GetTime() time.Time {
	return m.Time
}

type TextAnalyzer interface {
	Analyzer[TextAnalyzerInput]
}

/*
AudioAnalyzer is an interface for analyzers that work with audio data.
It extends the base Analyzer interface.

Methods:
  - Name() string: Inherited from Analyzer, returns the name of the analyzer.
  - Analyze(s []byte): Performs analysis on the given audio data represented as a byte slice.
*/
type AudioAnalyzer interface {
	Analyzer[[]byte]
}
