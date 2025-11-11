package internal_analyzers

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/rapidaai/pkg/commons"
)

type EndOfSpeechSegment struct {
	startAt float64
	endAt   float64
	speech  string
}

func (s *EndOfSpeechSegment) GetSpeechStartAt() float64 { return s.startAt }
func (s *EndOfSpeechSegment) GetSpeechEndAt() float64   { return s.endAt }
func (s *EndOfSpeechSegment) GetDuration() float64      { return s.endAt - s.startAt }
func (s *EndOfSpeechSegment) GetSpeech() string         { return s.speech }

type utteranceEndAnalyzer struct {
	logger            commons.Logger
	onAnalyze         func(context.Context, Activity) error
	thresholdDuration time.Duration
	activities        []TextAnalyzerInput
	currentCtx        context.Context
	currentCancel     context.CancelFunc
}

func NewTextUtteranceEndAnalyzer(
	logger commons.Logger,
	opts *TextAnalyzerOptions,
) (TextAnalyzer, error) {
	duration := time.Duration(1500) * time.Millisecond
	timeOut, err := opts.AnalyzeOptions.opts.GetFloat64("microphone.eos.timeout")
	if err == nil {
		duration = time.Duration(timeOut) * time.Millisecond
	}
	uea := &utteranceEndAnalyzer{
		logger:            logger,
		onAnalyze:         opts.OnAnalyze,
		thresholdDuration: duration,
	}
	return uea, nil
}

func (a *utteranceEndAnalyzer) Name() string {
	return "text-utterance-end-analyzer"
}

func (alyzer *utteranceEndAnalyzer) Analyze(ctx context.Context, msg TextAnalyzerInput) error {
	alyzer.logger.Infof("utteranceEndAnalyzer: analyze %s", msg.GetMessage())
	switch input := msg.(type) {
	case *UserTextAnalyzerInput:
		alyzer.triggerExtension(ctx, input.GetMessage(), alyzer.thresholdDuration)

	case *SystemTextAnalyzerInput:
		alyzer.triggerExtension(ctx, input.GetMessage(), alyzer.thresholdDuration)

	case *STTTextAnalyzerInput:
		alyzer.handleSTTInput(ctx, input)
	}
	alyzer.activities = append(alyzer.activities, msg)
	return nil
}

func (a *utteranceEndAnalyzer) handleSTTInput(ctx context.Context, input *STTTextAnalyzerInput) error {
	if (len(a.activities)) == 0 || !input.IsComplete {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}
	recentActivity := a.activities[len(a.activities)-1]
	// if last activity is not stt's activity then skip
	sActivity, ok := recentActivity.(*STTTextAnalyzerInput)
	if !ok {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}

	if normalizeMessage(sActivity.GetMessage()) != normalizeMessage(input.GetMessage()) {
		return a.triggerExtension(ctx, input.GetMessage(), a.thresholdDuration)
	}
	// skip in case of current message is complete of previour partial message
	a.logger.Infof("utteranceEndAnalyzer: Saving time Partial Message %s completed %v", sActivity.GetMessage(), sActivity.IsComplete)
	a.logger.Infof("utteranceEndAnalyzer: Saving time Complete Message %s completed %v", input.GetMessage(), input.IsComplete)
	a.logger.Infof("utteranceEndAnalyzer: Saving time from utterance %d", a.thresholdDuration)
	// no saving
	adjustedThreshold := a.thresholdDuration - 200
	if adjustedThreshold < 0 {
		adjustedThreshold = 100
	}
	return a.triggerExtension(ctx, input.GetMessage(), adjustedThreshold)
}

func (alyzer *utteranceEndAnalyzer) triggerExtension(ctx context.Context, speech string, duration time.Duration) error {
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

func (a *utteranceEndAnalyzer) extendTimer(ctx context.Context, speech string, duration time.Duration) {
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
			if err := a.onAnalyze(ctx, seg); err != nil {
				a.logger.Errorf("interrupt: Error in onAnalyze: %v", err)
			}
		}
	}
}

func buildSpeechSegment(start, end time.Time, speech string) *EndOfSpeechSegment {
	return &EndOfSpeechSegment{
		startAt: float64(start.UnixNano()) / 1e9,
		endAt:   float64(end.UnixNano()) / 1e9,
		speech:  speech,
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

func (a *utteranceEndAnalyzer) Close() error {
	return nil
}
