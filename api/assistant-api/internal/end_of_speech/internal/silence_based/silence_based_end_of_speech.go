// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	silenceBasedEndOfSpeechName = "silenceBasedEndOfSpeech"
	optSilenceTimeout           = "microphone.eos.timeout"
	defaultSilenceTimeout       = 1000 * time.Millisecond
	defaultVadEndTimeout        = 250 * time.Millisecond
)

type vadState uint8

const (
	vadStateIdle vadState = iota
	vadStateSpeaking
	vadStateEnded
)

type transcriptState uint8

const (
	transcriptStateIdle transcriptState = iota
	transcriptStateInterimPending
	transcriptStateFinalized
	transcriptStateFinalizedWithPendingInterim
)

type speechSegment struct {
	Revision  uint64
	ContextID string
	FinalText string
	Text      string
	Chunks    []internal_type.SpeechToTextPacket
	Timestamp time.Time
}

type workerCommand struct {
	ctx             context.Context
	timeout         time.Duration
	segment         speechSegment
	fireImmediately bool
}

type endOfSpeechState struct {
	segment       speechSegment
	pending       *workerCommand
	started       bool
	callbackFired bool
	vadState      vadState
	transcript    transcriptState
}

type silenceBasedEndOfSpeech struct {
	onPacket       func(context.Context, ...internal_type.Packet) error
	opts           utils.Option
	silenceTimeout time.Duration
	vadEndTimeout  time.Duration

	commandCh chan workerCommand
	stopCh    chan struct{}
	closeOnce sync.Once

	mu    sync.RWMutex
	state *endOfSpeechState

	eosStartedAt time.Time
}

type options struct {
	ctx      context.Context
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	options  utils.Option
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(opts utils.Option) Option {
	return func(options *options) {
		options.options = opts
	}
}

func New(opts ...Option) (internal_type.EndOfSpeechExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.onPacket == nil {
		return nil, fmt.Errorf("%s: onPacket is required", silenceBasedEndOfSpeechName)
	}
	start := time.Now()
	silenceTimeout := defaultSilenceTimeout
	if value, err := options.options.GetFloat64(optSilenceTimeout); err == nil {
		silenceTimeout = time.Duration(value) * time.Millisecond
	}

	endOfSpeech := &silenceBasedEndOfSpeech{
		onPacket:       options.onPacket,
		opts:           options.options,
		silenceTimeout: silenceTimeout,
		vadEndTimeout:  defaultVadEndTimeout,
		commandCh:      make(chan workerCommand, 32),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
		eosStartedAt:   time.Now(),
	}

	go endOfSpeech.worker()
	_ = endOfSpeech.onPacket(options.ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": endOfSpeech.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricEOSInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "EOS initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: fmt.Sprintf("%s: initialization completed", endOfSpeech.Name()),
				Attributes: observability.Attributes{
					"component": observability.ComponentEOS.String(),
					"provider":  endOfSpeech.Name(),
					"options":   observability.AttributeValue(endOfSpeech.Options()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return endOfSpeech, nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) Name() string {
	return silenceBasedEndOfSpeechName
}

func (endOfSpeech *silenceBasedEndOfSpeech) Options() utils.Option {
	return endOfSpeech.opts
}

func (endOfSpeech *silenceBasedEndOfSpeech) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) Execute(ctx context.Context, packet internal_type.Packet) error {
	switch packet := packet.(type) {
	case internal_type.UserTextReceivedPacket:
		return endOfSpeech.handleUserTextPacket(ctx, packet)
	case internal_type.EndOfSpeechInterruptionPacket:
		return endOfSpeech.handleInterruptionPacket(ctx)
	case internal_type.InterruptionDetectedPacket:
		if packet.Source != internal_type.InterruptionSourceVad {
			return nil
		}
		endOfSpeech.mu.Lock()
		switch packet.Event {
		case internal_type.InterruptionEventStart:
			// VAD start only marks speech as active. Transcript revision remains
			// the only token used to reject stale EOS timers.
			endOfSpeech.state.vadState = vadStateSpeaking
			endOfSpeech.state.pending = nil
			endOfSpeech.mu.Unlock()
			return nil
		case internal_type.InterruptionEventEnd:
			endOfSpeech.state.vadState = vadStateEnded
			if endOfSpeech.state.segment.Text != "" &&
				!endOfSpeech.state.callbackFired &&
				(endOfSpeech.state.transcript == transcriptStateFinalized ||
					endOfSpeech.state.transcript == transcriptStateIdle) {
				command := workerCommand{
					ctx:     ctx,
					segment: endOfSpeech.state.segment,
					timeout: endOfSpeech.vadEndTimeout,
				}
				endOfSpeech.state.pending = nil
				endOfSpeech.mu.Unlock()
				// VAD end is not transcript finalization. It only opens the
				// flush window after STT has acknowledged with a final packet.
				endOfSpeech.enqueueCommand(command)
				return nil
			}
			if endOfSpeech.state.segment.Text != "" &&
				!endOfSpeech.state.callbackFired &&
				endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim {
				command := workerCommand{
					ctx:     ctx,
					segment: endOfSpeech.state.segment,
					timeout: endOfSpeech.silenceTimeout,
				}
				endOfSpeech.state.pending = nil
				endOfSpeech.mu.Unlock()
				// A newer interim after a final means STT finalization is still
				// catching up. Wait longer, then use the best visible transcript.
				endOfSpeech.enqueueCommand(command)
				return nil
			}
		}
		endOfSpeech.mu.Unlock()
		return nil
	case internal_type.SpeechToTextPacket:
		return endOfSpeech.handleSpeechToTextPacket(ctx, packet)
	}

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleUserTextPacket(
	ctx context.Context,
	packet internal_type.UserTextReceivedPacket,
) error {
	if packet.Text == "" {
		return nil
	}

	endOfSpeech.mu.Lock()
	segment := speechSegment{
		Revision:  endOfSpeech.state.segment.Revision + 1,
		ContextID: packet.ContextId(),
		FinalText: packet.Text,
		Text:      packet.Text,
		Timestamp: time.Now(),
	}
	command := workerCommand{
		ctx:             ctx,
		segment:         segment,
		fireImmediately: true,
	}
	endOfSpeech.state.segment = segment
	endOfSpeech.state.transcript = transcriptStateFinalized
	endOfSpeech.mu.Unlock()

	_ = endOfSpeech.onPacket(ctx,
		internal_type.InterimEndOfSpeechPacket{
			Speech:    command.segment.Text,
			ContextID: command.segment.ContextID,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: command.segment.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentEOS,
				Event:      observability.EOSStarted,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   endOfSpeech.Name(),
					"context_id": command.segment.ContextID,
					"speech":     command.segment.Text,
				},
			},
		},
	)
	endOfSpeech.enqueueCommand(command)

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleInterruptionPacket(ctx context.Context) error {
	return endOfSpeech.extendCurrentSegment(ctx, endOfSpeech.silenceTimeout)
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleSpeechToTextPacket(
	ctx context.Context,
	packet internal_type.SpeechToTextPacket,
) error {
	endOfSpeech.mu.Lock()
	if packet.Interim {
		previous := endOfSpeech.state.segment
		if packet.Script == "" {
			endOfSpeech.mu.Unlock()
			return nil
		}
		timestamp := time.Now()
		if previous.FinalText != "" && !previous.Timestamp.IsZero() {
			timestamp = previous.Timestamp
		}
		segment := speechSegment{
			Revision:  previous.Revision + 1,
			ContextID: packet.ContextId(),
			Chunks:    append([]internal_type.SpeechToTextPacket(nil), previous.Chunks...),
			Timestamp: timestamp,
		}
		segment.Chunks = append(segment.Chunks, packet)
		pendingTranscript := ""
		for _, chunk := range segment.Chunks {
			if chunk.Script == "" {
				continue
			}
			if chunk.Interim {
				pendingTranscript = chunk.Script
				if segment.FinalText != "" {
					pendingTranscript = chunk.GetConcat() + pendingTranscript
				}
				continue
			}
			if segment.FinalText != "" {
				segment.FinalText += chunk.GetConcat()
			}
			segment.FinalText += chunk.Script
			pendingTranscript = ""
		}
		segment.Text = segment.FinalText + pendingTranscript
		emitStarted := segment.Text != "" && !endOfSpeech.state.started
		if emitStarted {
			endOfSpeech.state.started = true
		}
		if previous.FinalText == "" {
			endOfSpeech.state.transcript = transcriptStateInterimPending
		} else {
			endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
		}
		endOfSpeech.state.segment = segment
		if endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim ||
			endOfSpeech.state.vadState == vadStateEnded {
			command := workerCommand{
				ctx:     ctx,
				segment: segment,
				timeout: endOfSpeech.silenceTimeout,
			}
			endOfSpeech.mu.Unlock()

			if emitStarted {
				_ = endOfSpeech.onPacket(ctx,
					internal_type.InterimEndOfSpeechPacket{
						Speech:    command.segment.Text,
						ContextID: command.segment.ContextID,
					},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: command.segment.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentEOS,
							Event:      observability.EOSStarted,
							OccurredAt: time.Now(),
							Attributes: observability.Attributes{
								"provider":   endOfSpeech.Name(),
								"context_id": command.segment.ContextID,
								"speech":     command.segment.Text,
							},
						},
					},
				)
			} else {
				_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
					Speech:    command.segment.Text,
					ContextID: command.segment.ContextID,
				})
			}
			endOfSpeech.enqueueCommand(command)
			return nil
		}
		endOfSpeech.mu.Unlock()

		if emitStarted {
			_ = endOfSpeech.onPacket(ctx,
				internal_type.InterimEndOfSpeechPacket{
					Speech:    segment.Text,
					ContextID: segment.ContextID,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: segment.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component:  observability.ComponentEOS,
						Event:      observability.EOSStarted,
						OccurredAt: time.Now(),
						Attributes: observability.Attributes{
							"provider":   endOfSpeech.Name(),
							"context_id": segment.ContextID,
							"speech":     segment.Text,
						},
					},
				},
			)
		} else {
			_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
				Speech:    segment.Text,
				ContextID: segment.ContextID,
			})
		}
		return nil
	}

	previous := endOfSpeech.state.segment
	segment := speechSegment{
		Revision:  previous.Revision + 1,
		ContextID: packet.ContextId(),
		Timestamp: time.Now(),
		Chunks:    append([]internal_type.SpeechToTextPacket(nil), previous.Chunks...),
	}
	segment.Chunks = append(segment.Chunks, packet)
	pendingTranscript := ""
	for _, chunk := range segment.Chunks {
		if chunk.Script == "" {
			continue
		}
		if chunk.Interim {
			pendingTranscript = chunk.Script
			if segment.FinalText != "" {
				pendingTranscript = chunk.GetConcat() + pendingTranscript
			}
			continue
		}
		if segment.FinalText != "" {
			segment.FinalText += chunk.GetConcat()
		}
		segment.FinalText += chunk.Script
		pendingTranscript = ""
	}
	segment.Text = segment.FinalText + pendingTranscript
	emitStarted := segment.Text != "" && !endOfSpeech.state.started
	if emitStarted {
		endOfSpeech.state.started = true
	}
	endOfSpeech.state.transcript = transcriptStateFinalized
	if segment.Text != segment.FinalText {
		endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
	}
	endOfSpeech.state.segment = segment
	if endOfSpeech.state.vadState == vadStateEnded {
		timeout := endOfSpeech.vadEndTimeout
		if endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim {
			timeout = endOfSpeech.silenceTimeout
		}
		command := workerCommand{
			ctx:     ctx,
			segment: segment,
			timeout: timeout,
		}
		endOfSpeech.mu.Unlock()

		if emitStarted {
			_ = endOfSpeech.onPacket(ctx,
				internal_type.InterimEndOfSpeechPacket{
					Speech:    command.segment.Text,
					ContextID: command.segment.ContextID,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: command.segment.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component:  observability.ComponentEOS,
						Event:      observability.EOSStarted,
						OccurredAt: time.Now(),
						Attributes: observability.Attributes{
							"provider":   endOfSpeech.Name(),
							"context_id": command.segment.ContextID,
							"speech":     command.segment.Text,
						},
					},
				},
			)
		} else {
			_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
				Speech:    command.segment.Text,
				ContextID: command.segment.ContextID,
			})
		}
		endOfSpeech.enqueueCommand(command)
		return nil
	}
	command := workerCommand{
		ctx:     ctx,
		segment: segment,
		timeout: endOfSpeech.silenceTimeout,
	}
	endOfSpeech.mu.Unlock()

	if emitStarted {
		_ = endOfSpeech.onPacket(ctx,
			internal_type.InterimEndOfSpeechPacket{
				Speech:    command.segment.Text,
				ContextID: command.segment.ContextID,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: command.segment.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentEOS,
					Event:      observability.EOSStarted,
					OccurredAt: time.Now(),
					Attributes: observability.Attributes{
						"provider":   endOfSpeech.Name(),
						"context_id": command.segment.ContextID,
						"speech":     command.segment.Text,
					},
				},
			},
		)
	} else {
		_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
			Speech:    command.segment.Text,
			ContextID: command.segment.ContextID,
		})
	}
	endOfSpeech.enqueueCommand(command)

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) extendCurrentSegment(
	ctx context.Context,
	timeout time.Duration,
) error {
	endOfSpeech.mu.RLock()
	command := workerCommand{
		ctx:     ctx,
		segment: endOfSpeech.state.segment,
		timeout: timeout,
	}
	endOfSpeech.mu.RUnlock()

	if command.segment.Text == "" {
		return nil
	}

	endOfSpeech.enqueueCommand(command)

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) enqueueCommand(command workerCommand) {
	if endOfSpeech == nil || endOfSpeech.commandCh == nil || endOfSpeech.stopCh == nil {
		return
	}

	select {
	case <-endOfSpeech.stopCh:
		return
	default:
	}

	select {
	case endOfSpeech.commandCh <- command:
	case <-endOfSpeech.stopCh:
	}
}

func (endOfSpeech *silenceBasedEndOfSpeech) worker() {
	var (
		timer          *time.Timer
		timerCh        <-chan time.Time
		timerArmedAt   time.Time
		currentCommand workerCommand
	)

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerCh = nil
		}
		timerArmedAt = time.Time{}
	}
	resetState := func() {
		endOfSpeech.state.callbackFired = false
		// Bump revision after a completed turn so late timer work from the old
		// message cannot complete against the next user turn.
		endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision + 1}
		endOfSpeech.state.pending = nil
		endOfSpeech.state.started = false
		endOfSpeech.state.vadState = vadStateIdle
		endOfSpeech.state.transcript = transcriptStateIdle
	}

	for {
		select {
		case <-endOfSpeech.stopCh:
			stopTimer()
			return

		case command := <-endOfSpeech.commandCh:
			endOfSpeech.mu.Lock()

			if endOfSpeech.state.callbackFired {
				endOfSpeech.mu.Unlock()
				continue
			}
			// Newer text/STT packets supersede older timer commands. Completing a
			// stale snapshot can shrink the final user message.
			if !command.fireImmediately && command.segment.Revision != endOfSpeech.state.segment.Revision {
				endOfSpeech.mu.Unlock()
				continue
			}

			if command.fireImmediately {
				endOfSpeech.state.callbackFired = true
				endOfSpeech.state.pending = nil
				stopTimer()
				endOfSpeech.mu.Unlock()
				endOfSpeech.emitEndOfSpeech(command, time.Now())
				endOfSpeech.mu.Lock()
				resetState()
				endOfSpeech.mu.Unlock()
				continue
			}

			endOfSpeech.state.pending = nil
			currentCommand = command
			stopTimer()
			timerArmedAt = time.Now()
			timer = time.NewTimer(command.timeout)
			timerCh = timer.C
			endOfSpeech.mu.Unlock()

		case <-timerCh:
			endOfSpeech.mu.Lock()
			if endOfSpeech.state.callbackFired {
				endOfSpeech.mu.Unlock()
				continue
			}

			// While VAD says the user is speaking, defer once. The fallback timer
			// recovers if VAD end is lost because of noise or provider failure.
			if endOfSpeech.state.vadState == vadStateSpeaking {
				if endOfSpeech.state.pending == nil {
					command := currentCommand
					endOfSpeech.state.pending = &command
					stopTimer()
					timerArmedAt = time.Now()
					timer = time.NewTimer(endOfSpeech.silenceTimeout)
					timerCh = timer.C
					endOfSpeech.mu.Unlock()
					continue
				}

				command := *endOfSpeech.state.pending
				endOfSpeech.state.pending = nil
				endOfSpeech.state.vadState = vadStateIdle
				if command.segment.Revision != endOfSpeech.state.segment.Revision {
					stopTimer()
					endOfSpeech.mu.Unlock()
					continue
				}
				if endOfSpeech.state.transcript == transcriptStateInterimPending {
					stopTimer()
					endOfSpeech.mu.Unlock()
					continue
				}

				endOfSpeech.state.callbackFired = true
				armedAt := timerArmedAt
				stopTimer()
				endOfSpeech.mu.Unlock()
				endOfSpeech.emitEndOfSpeech(command, armedAt)
				endOfSpeech.mu.Lock()
				resetState()
				endOfSpeech.mu.Unlock()
				continue
			}
			if currentCommand.segment.Revision != endOfSpeech.state.segment.Revision {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			if endOfSpeech.state.transcript == transcriptStateInterimPending {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}

			endOfSpeech.state.callbackFired = true
			command := currentCommand
			armedAt := timerArmedAt
			stopTimer()
			endOfSpeech.mu.Unlock()
			endOfSpeech.emitEndOfSpeech(command, armedAt)
			endOfSpeech.mu.Lock()
			resetState()
			endOfSpeech.mu.Unlock()
		}
	}
}

func (endOfSpeech *silenceBasedEndOfSpeech) emitEndOfSpeech(command workerCommand, timerArmedAt time.Time) {
	if endOfSpeech == nil || endOfSpeech.onPacket == nil {
		return
	}

	ctx := command.ctx
	segment := command.segment
	if segment.Text == "" {
		return
	}
	if ctx != nil && ctx.Err() != nil {
		ctx = context.Background()
	}

	wordCount := len(strings.Fields(segment.Text))
	triggerAt := time.Now()
	textToTriggerMs := triggerAt.Sub(segment.Timestamp).Milliseconds()
	waitToTriggerMs := textToTriggerMs
	if !timerArmedAt.IsZero() {
		waitToTriggerMs = triggerAt.Sub(timerArmedAt).Milliseconds()
	}
	_ = endOfSpeech.onPacket(ctx,
		internal_type.EndOfSpeechPacket{
			Speech:    segment.Text,
			ContextID: segment.ContextID,
			Speechs:   append([]internal_type.SpeechToTextPacket(nil), segment.Chunks...),
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: segment.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentEOS,
				Event:      observability.EOSCompleted,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":           endOfSpeech.Name(),
					"context_id":         segment.ContextID,
					"speech":             segment.Text,
					"confidence":         "0.0000",
					"word_count":         fmt.Sprintf("%d", wordCount),
					"char_count":         fmt.Sprintf("%d", len(segment.Text)),
					"text_to_trigger_ms": fmt.Sprintf("%d", textToTriggerMs),
					"wait_to_trigger_ms": fmt.Sprintf("%d", waitToTriggerMs),
				},
			},
		},
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: segment.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordMetric{
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider": endOfSpeech.Name(),
				},
				Metrics: []*protos.Metric{
					{Name: observability.MetricEOSLatencyMs, Value: fmt.Sprintf("%d", waitToTriggerMs)},
					{Name: observability.MetricEOSTextToTriggerMs, Value: fmt.Sprintf("%d", textToTriggerMs)},
					{Name: observability.MetricEOSWordCount, Value: fmt.Sprintf("%d", wordCount)},
					{Name: observability.MetricEOSConfidence, Value: "0.0000"},
				},
			},
		},
	)
}

func (endOfSpeech *silenceBasedEndOfSpeech) Close(ctx context.Context) error {
	if endOfSpeech == nil {
		return nil
	}

	endOfSpeech.closeOnce.Do(func() {
		endOfSpeech.mu.Lock()
		eosStartedAt := endOfSpeech.eosStartedAt
		endOfSpeech.eosStartedAt = time.Time{}
		endOfSpeech.mu.Unlock()

		if endOfSpeech.onPacket != nil {
			if !eosStartedAt.IsZero() {
				_ = endOfSpeech.onPacket(ctx, internal_type.ObservabilityUsageRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.NewEOSDurationUsageRecord(endOfSpeech.Name(), time.Since(eosStartedAt), observability.Attributes{
						"provider": endOfSpeech.Name(),
					}),
				})
			}
			_ = endOfSpeech.onPacket(ctx, internal_type.ObservabilityEventRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component: observability.ComponentEOS,
					Event:     observability.EOSClosed,
					Attributes: observability.Attributes{
						"provider": endOfSpeech.Name(),
					},
					OccurredAt: time.Now(),
				},
			})
		}
		if endOfSpeech.stopCh != nil {
			close(endOfSpeech.stopCh)
		}
	})
	return nil
}
