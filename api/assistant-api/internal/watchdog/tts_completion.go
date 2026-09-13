// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

const (
	DefaultWordsPerMinute = 120
	DefaultMinimumTimeout = 2 * time.Second
	DefaultGracePeriod    = 1500 * time.Millisecond
)

type TTSCompletionOptions = WatchdogOptions
type TTSCompletionOption = Option

type ttsCompletionOption struct {
	applyFunc func(*TTSCompletionOptions)
}

func (option ttsCompletionOption) applyWatchdogOptions(options *WatchdogOptions) {
	if option.applyFunc == nil {
		return
	}
	option.applyFunc(options)
}

func WithWordsPerMinute(wordsPerMinute int) TTSCompletionOption {
	return ttsCompletionOption{applyFunc: func(options *TTSCompletionOptions) {
		options.WordsPerMinute = wordsPerMinute
	}}
}

func WithMinimumTimeout(minimumTimeout time.Duration) TTSCompletionOption {
	return ttsCompletionOption{applyFunc: func(options *TTSCompletionOptions) {
		options.MinimumTimeout = minimumTimeout
	}}
}

func WithGracePeriod(gracePeriod time.Duration) TTSCompletionOption {
	return ttsCompletionOption{applyFunc: func(options *TTSCompletionOptions) {
		options.GracePeriod = gracePeriod
	}}
}

type TTSCompletionEvent struct {
	ContextID              string
	Deadline               time.Time
	EstimatedAudioDuration time.Duration
	ObservedAudioDuration  time.Duration
}

type TTSCompletionWatchdog struct {
	mu      sync.Mutex
	options TTSCompletionOptions

	timer *time.Timer

	generation             uint64
	active                 bool
	contextID              string
	deadline               time.Time
	firstAudioAt           time.Time
	estimatedAudioDuration time.Duration
	observedAudioDuration  time.Duration
}

func NewTTSCompletionWatchdog(opts ...TTSCompletionOption) *TTSCompletionWatchdog {
	options := TTSCompletionOptions{
		WordsPerMinute: DefaultWordsPerMinute,
		MinimumTimeout: DefaultMinimumTimeout,
		GracePeriod:    DefaultGracePeriod,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.applyWatchdogOptions(&options)
	}

	if options.WordsPerMinute <= 0 {
		options.WordsPerMinute = DefaultWordsPerMinute
	}
	if options.MinimumTimeout <= 0 {
		options.MinimumTimeout = DefaultMinimumTimeout
	}
	if options.GracePeriod <= 0 {
		options.GracePeriod = DefaultGracePeriod
	}
	if options.PacketContext == nil {
		options.PacketContext = context.Background()
	}
	if options.RecordScope == "" {
		options.RecordScope = internal_type.ObservabilityRecordScopeConversation
	}

	watchdog := &TTSCompletionWatchdog{
		options: options,
	}

	if options.OnPacket != nil {
		_ = options.OnPacket(
			options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				Scope: options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "tts-completion-watchdog: initialization completed",
					Attributes: observability.Attributes{
						"component":          observability.ComponentTTS.String(),
						"watchdog":           "tts_completion",
						"words_per_minute":   observability.AttributeValue(options.WordsPerMinute),
						"minimum_timeout_ms": observability.AttributeValue(options.MinimumTimeout.Milliseconds()),
						"grace_period_ms":    observability.AttributeValue(options.GracePeriod.Milliseconds()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return watchdog
}

func (w *TTSCompletionWatchdog) StartFromText(contextID, text string) bool {
	estimatedDuration := EstimateSpeechDuration(text, w.options.WordsPerMinute)
	if estimatedDuration < w.options.MinimumTimeout {
		estimatedDuration = w.options.MinimumTimeout
	}

	return w.Start(contextID, estimatedDuration)
}

func (w *TTSCompletionWatchdog) Start(contextID string, timeout time.Duration) bool {
	if timeout < w.options.MinimumTimeout {
		timeout = w.options.MinimumTimeout
	}
	timeout += w.options.GracePeriod

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = true
	w.contextID = contextID
	w.firstAudioAt = time.Time{}
	w.estimatedAudioDuration = timeout - w.options.GracePeriod
	w.observedAudioDuration = 0
	w.deadline = time.Now().Add(timeout)

	generation := w.generation
	w.timer = time.AfterFunc(timeout, func() {
		w.expire(generation)
	})

	return true
}

func (w *TTSCompletionWatchdog) Extend(contextID string, duration time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active || w.contextID != contextID || duration <= 0 {
		return false
	}

	w.observedAudioDuration += duration
	if w.firstAudioAt.IsZero() {
		w.firstAudioAt = time.Now()
	}
	w.deadline = w.firstAudioAt.Add(w.observedAudioDuration).Add(w.options.GracePeriod)
	if remaining := time.Until(w.deadline); remaining > 0 {
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
		w.generation++
		generation := w.generation
		w.timer = time.AfterFunc(remaining, func() {
			w.expire(generation)
		})
	}

	return true
}

func (w *TTSCompletionWatchdog) Complete(contextID string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active || w.contextID != contextID {
		return false
	}

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.firstAudioAt = time.Time{}
	w.estimatedAudioDuration = 0
	w.observedAudioDuration = 0
	return true
}

func (w *TTSCompletionWatchdog) Cancel() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active {
		return false
	}

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.firstAudioAt = time.Time{}
	w.estimatedAudioDuration = 0
	w.observedAudioDuration = 0
	return true
}

func EstimateSpeechDuration(text string, wordsPerMinute int) time.Duration {
	if wordsPerMinute <= 0 {
		wordsPerMinute = DefaultWordsPerMinute
	}

	wordCount := len(strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}))
	if wordCount == 0 {
		return 0
	}

	return time.Duration(wordCount) * time.Minute / time.Duration(wordsPerMinute)
}

func (w *TTSCompletionWatchdog) expire(generation uint64) {
	w.mu.Lock()
	if !w.active || w.generation != generation {
		w.mu.Unlock()
		return
	}

	event := TTSCompletionEvent{
		ContextID:              w.contextID,
		Deadline:               w.deadline,
		EstimatedAudioDuration: w.estimatedAudioDuration,
		ObservedAudioDuration:  w.observedAudioDuration,
	}
	w.timer = nil
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.firstAudioAt = time.Time{}
	w.estimatedAudioDuration = 0
	w.observedAudioDuration = 0
	w.mu.Unlock()

	if w.options.OnPacket != nil {
		_ = w.options.OnPacket(
			w.options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				ContextID: event.ContextID,
				Scope:     w.options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "tts-completion-watchdog: deadline expired",
					Attributes: observability.Attributes{
						"component":                   observability.ComponentTTS.String(),
						"watchdog":                    "tts_completion",
						"estimated_audio_duration_ms": observability.AttributeValue(event.EstimatedAudioDuration.Milliseconds()),
						"observed_audio_duration_ms":  observability.AttributeValue(event.ObservedAudioDuration.Milliseconds()),
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.TextToSpeechErrorPacket{
				ContextID: event.ContextID,
				Error:     errors.New("tts-completion-watchdog: deadline expired"),
				Type:      internal_type.TTSNetworkTimeout,
			},
		)
	}
}
