package internal_analyzers

import (
	"context"

	"github.com/rapidaai/pkg/utils"
)

type AnalyzeOptions struct {
	opts utils.Option
}

type Activity interface {
	// The relative timestamp in seconds of when a speech segment begins.
	GetSpeechStartAt() float64
	// The relative timestamp in seconds of when a speech segment ends.
	GetSpeechEndAt() float64
	// The duration of the speech segment in seconds.
	GetDuration() float64
}

type SpeechStartActivity interface {
	Activity
	// The energy level of the speech segment, used to filter out noise.
	GetEnergy() float64
	// A confidence score indicating the likelihood of valid speech (0 to 1).
	GetConfidence() float32
}

type SpeechEndActivity interface {
	Activity
	GetSpeech() string
}

type VoiceAnalyzerOptions struct {
	AnalyzeOptions
	OnAnalyze func(ctx context.Context, t Activity) error
}

func (ao *VoiceAnalyzerOptions) WithOptions(opts map[string]interface{}) *VoiceAnalyzerOptions {
	ao.opts = opts
	return ao
}

type TextAnalyzerOptions struct {
	AnalyzeOptions
	OnAnalyze func(ctx context.Context, t Activity) error
}

func (ao *TextAnalyzerOptions) WithOptions(opts map[string]interface{}) *TextAnalyzerOptions {
	ao.opts = opts
	return ao
}
