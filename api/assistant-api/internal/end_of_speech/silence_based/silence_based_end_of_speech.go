// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based_end_of_speech

import (
	"context"
	"strings"
	"time"
	"unicode"

	internal_end_of_speech "github.com/rapidaai/api/assistant-api/internal/end_of_speech"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type silenceBasedEndOfSpeech struct {
	logger            commons.Logger
	onCallback        internal_end_of_speech.EndOfSpeechCallback
	thresholdDuration time.Duration
	activities        []internal_end_of_speech.EndOfSpeechInput
	currentCtx        context.Context
	currentCancel     context.CancelFunc
}

func NewSilenceBasedEndOfSpeech(
	logger commons.Logger,
	onCallback internal_end_of_speech.EndOfSpeechCallback,
	opts utils.Option,
) (internal_end_of_speech.EndOfSpeech, error) {
	duration := time.Duration(1000) * time.Millisecond
	timeOut, err := opts.GetFloat64("microphone.eos.timeout")
	if err == nil {
		logger.Debugf("overriding default duration of timeout for silence based eos.")
		duration = time.Duration(timeOut) * time.Millisecond
	}
	uea := &silenceBasedEndOfSpeech{
		logger:            logger,
		onCallback:        onCallback,
		thresholdDuration: duration,
	}
	return uea, nil
}

func (a *silenceBasedEndOfSpeech) Name() string {
	return "silenceBasedEndOfSpeech"
}

func (alyzer *silenceBasedEndOfSpeech) Analyze(ctx context.Context, msg internal_end_of_speech.EndOfSpeechInput) error {
	switch input := msg.(type) {
	case *internal_end_of_speech.UserEndOfSpeechInput:
		alyzer.triggerExtension(ctx, input.GetMessage(), alyzer.thresholdDuration)

	case *internal_end_of_speech.SystemEndOfSpeechInput:
		alyzer.triggerExtension(ctx, input.GetMessage(), alyzer.thresholdDuration)

	case *internal_end_of_speech.STTEndOfSpeechInput:
		alyzer.handleSTTInput(ctx, input)
	}
	alyzer.activities = append(alyzer.activities, msg)
	return nil
}

func (a *silenceBasedEndOfSpeech) handleSTTInput(ctx context.Context, input *internal_end_of_speech.STTEndOfSpeechInput) error {
	if (len(a.activities)) == 0 || !input.IsComplete {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}
	recentActivity := a.activities[len(a.activities)-1]
	// if last activity is not stt's activity then skip
	sActivity, ok := recentActivity.(*internal_end_of_speech.STTEndOfSpeechInput)
	if !ok {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}

	if normalizeMessage(sActivity.GetMessage()) != normalizeMessage(input.GetMessage()) {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}
	adjustedThreshold := a.thresholdDuration - 200
	if adjustedThreshold < 0 {
		adjustedThreshold = 100
	}
	return a.triggerExtension(ctx, input.GetMessage(), adjustedThreshold)
}

func (alyzer *silenceBasedEndOfSpeech) triggerExtension(ctx context.Context, speech string, duration time.Duration) error {
	if alyzer.currentCtx != nil {
		alyzer.currentCancel()
	}
	alyzer.currentCtx, alyzer.currentCancel = context.WithCancel(ctx)

	// cancle existing context as it is not required to push the activity to upstream services
	select {
	case <-alyzer.currentCtx.Done():
		return alyzer.currentCtx.Err()
	default:
		alyzer.extendTimer(alyzer.currentCtx, speech, duration)
		return nil
	}
}

func (a *silenceBasedEndOfSpeech) extendTimer(ctx context.Context, speech string, duration time.Duration) {
	start := time.Now()
	timer := time.NewTimer(duration)

	select {
	case <-ctx.Done():
		timer.Stop()
		return
	case <-timer.C:
		end := time.Now()
		if speech != "" {
			seg := buildSpeechSegment(start, end, speech)
			a.logger.Debugf("Analyzer interrupted. Detected silence. Speech segment: '%s', Silence duration: %.2f ms", speech, seg.GetDuration()*1000)
			if err := a.onCallback(ctx, seg); err != nil {
				a.logger.Errorf("interrupt: Error in onAnalyze: %v", err)
			}
		}
	}
}

func buildSpeechSegment(start, end time.Time, speech string) *internal_end_of_speech.EndOfSpeechResult {
	return &internal_end_of_speech.EndOfSpeechResult{
		StartAt: float64(start.UnixNano()) / 1e9,
		EndAt:   float64(end.UnixNano()) / 1e9,
		Speech:  speech,
	}
}

// Utility function for normalizing messages by removing punctuation and symbols
func normalizeMessage(message string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, message)
}

func (a *silenceBasedEndOfSpeech) Close() error {
	return nil
}
