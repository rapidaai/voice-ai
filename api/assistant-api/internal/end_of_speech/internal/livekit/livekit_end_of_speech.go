// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type vadState uint8

type transcriptState uint8

type speechSegment struct {
	Revision  uint64
	ContextID string
	FinalText string
	Text      string
	Timestamp time.Time
	Chunks    []internal_type.SpeechToTextPacket
}

type workerCommand struct {
	ctx             context.Context
	timeout         time.Duration
	deadline        time.Time
	segment         speechSegment
	confidence      float64
	fireImmediately bool
	predict         bool
	emitInterim     bool
	emitStarted     bool
}

type endOfSpeechState struct {
	segment       speechSegment
	confidence    float64
	started       bool
	vadState      vadState
	transcript    transcriptState
	lastSpeechEnd time.Time
}

type turnPredictor interface {
	PredictContext(context.Context, string) (float64, error)
}

// livekitEndOfSpeech predicts completion from committed text after speech stops.
// The worker owns delivery and records completed user turns before invoking callbacks.
type livekitEndOfSpeech struct {
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	opts     utils.Option

	// Model-based turn detection
	predictor   turnPredictor
	predictorMu sync.Mutex

	// Conversation history built from packets (protected by mu)
	history []chatMessage

	// Configuration
	threshold      float64
	quickTimeout   time.Duration
	silenceTimeout time.Duration
	maxHistory     int
	modelType      string

	// Worker orchestration
	commandCh  chan struct{}
	commands   []workerCommand
	stopCh     chan struct{}
	workerDone chan struct{}
	closedCh   chan struct{}
	closeOnce  sync.Once

	// State
	mu               sync.RWMutex
	state            *endOfSpeechState
	eosStartedAt     time.Time
	audioSamples     uint64
	closed           bool
	cancelPrediction context.CancelFunc
}

// New validates provider options before loading the model and starting the EOS worker.
// Initialization failures are returned to the caller, not emitted as packets.
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
		return nil, fmt.Errorf("%s: %w", eosName, errLivekitOnPacketRequired)
	}
	start := time.Now()

	endOfSpeech := &livekitEndOfSpeech{
		logger:         options.logger,
		onPacket:       options.onPacket,
		opts:           options.options,
		threshold:      defaultThreshold,
		quickTimeout:   time.Duration(defaultQuickTimeout) * time.Millisecond,
		silenceTimeout: time.Duration(defaultSilenceTimeout) * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		modelType:      defaultModelType,
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		workerDone:     make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}

	if options.options[optKeyThreshold] != nil {
		threshold, err := options.options.GetFloat64(optKeyThreshold)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errLivekitInvalidOption, optKeyThreshold, err)
		}
		if math.IsNaN(threshold) || threshold < 0 || threshold > 1 {
			return nil, fmt.Errorf("%w: %s must be between 0 and 1", errLivekitInvalidOption, optKeyThreshold)
		}
		endOfSpeech.threshold = threshold
	}
	for _, optionKey := range []string{optKeyQuickTimeout, optKeyExtendedTimeout} {
		if options.options[optionKey] == nil {
			continue
		}
		timeoutMillis, err := options.options.GetUint64(optionKey)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errLivekitInvalidOption, optionKey, err)
		}
		if timeoutMillis > uint64(math.MaxInt64/int64(time.Millisecond)) {
			return nil, fmt.Errorf("%w: %s exceeds the supported millisecond duration", errLivekitInvalidOption, optionKey)
		}
		switch optionKey {
		case optKeyQuickTimeout:
			endOfSpeech.quickTimeout = time.Duration(timeoutMillis) * time.Millisecond
		case optKeyExtendedTimeout:
			endOfSpeech.silenceTimeout = time.Duration(timeoutMillis) * time.Millisecond
		}
	}
	if options.options[optKeyMaxHistory] != nil {
		maxHistory, err := options.options.GetUint64(optKeyMaxHistory)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errLivekitInvalidOption, optKeyMaxHistory, err)
		}
		if maxHistory > uint64(math.MaxInt) {
			return nil, fmt.Errorf("%w: %s exceeds the supported history count", errLivekitInvalidOption, optKeyMaxHistory)
		}
		endOfSpeech.maxHistory = int(maxHistory)
	}
	detectorConfig := TurnDetectorConfig{ModelType: defaultModelType}
	if modelType, err := options.options.GetString(optKeyModel); err == nil && modelType != "" {
		detectorConfig.ModelType = modelType
	}
	if modelPath, err := options.options.GetString(optKeyModelPath); err == nil {
		detectorConfig.ModelPath = modelPath
	}
	if tokenizerPath, err := options.options.GetString(optKeyTokenizerPath); err == nil {
		detectorConfig.TokenizerPath = tokenizerPath
	}
	detector, err := NewTurnDetector(detectorConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLivekitInitTurnDetector, err)
	}
	endOfSpeech.predictor = detector
	endOfSpeech.modelType = detectorConfig.ModelType
	endOfSpeech.eosStartedAt = time.Now()

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

func (endOfSpeech *livekitEndOfSpeech) Name() string {
	return eosName
}

func (endOfSpeech *livekitEndOfSpeech) Options() utils.Option {
	return endOfSpeech.opts
}

func (endOfSpeech *livekitEndOfSpeech) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

// Execute applies packet state in order and queues inference and delivery without waiting for them.
// A canceled caller context is returned immediately; inference failures are reported asynchronously.
func (endOfSpeech *livekitEndOfSpeech) Execute(ctx context.Context, packet internal_type.Packet) error {
	endOfSpeech.mu.Lock()
	defer endOfSpeech.mu.Unlock()
	if endOfSpeech.closed {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch packet := packet.(type) {
	case internal_type.EndOfSpeechAudioPacket:
		// Slice lengths produce non-negative PCM16 sample counts.
		audioSampleCount, _ := utils.IntToUint64(len(packet.Audio) / 2)
		endOfSpeech.audioSamples += audioSampleCount
		return nil
	case internal_type.UserTextReceivedPacket:
		if packet.Text == "" {
			return nil
		}
		if endOfSpeech.cancelPrediction != nil {
			endOfSpeech.cancelPrediction()
		}
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
			emitInterim:     true,
			emitStarted:     true,
		}
		endOfSpeech.state.segment = segment
		endOfSpeech.state.confidence = 0
		endOfSpeech.state.transcript = transcriptStateUserText
		endOfSpeech.enqueueCommand(command)

	case internal_type.EndOfSpeechInterruptionPacket:
		if endOfSpeech.state.vadState != vadStateIdle || endOfSpeech.state.transcript == transcriptStateUserText {
			return nil
		}
		if packet.ContextID != "" &&
			endOfSpeech.state.segment.ContextID != "" &&
			packet.ContextID != endOfSpeech.state.segment.ContextID {
			return nil
		}
		command := workerCommand{
			ctx:        ctx,
			segment:    endOfSpeech.state.segment,
			confidence: endOfSpeech.state.confidence,
			timeout:    endOfSpeech.silenceTimeout,
		}
		command.segment.Text = command.segment.FinalText
		if command.segment.Text == "" {
			return nil
		}
		endOfSpeech.enqueueCommand(command)

	case internal_type.InterruptionDetectedPacket:
		if packet.Source != internal_type.InterruptionSourceVad {
			return nil
		}
		switch packet.Event {
		case internal_type.InterruptionEventStart:
			if endOfSpeech.cancelPrediction != nil {
				endOfSpeech.cancelPrediction()
			}
			endOfSpeech.state.vadState = vadStateSpeaking
			endOfSpeech.state.lastSpeechEnd = time.Time{}
			// Invalidate armed EOS timers without dropping transcript accumulated so far.
			endOfSpeech.state.segment.Revision++
			if endOfSpeech.state.transcript == transcriptStateUserText {
				endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision}
				endOfSpeech.state.transcript = transcriptStateIdle
				endOfSpeech.state.started = false
			}
			return nil
		case internal_type.InterruptionEventEnd:
			if endOfSpeech.state.vadState == vadStateEnded {
				return nil
			}
			endOfSpeech.state.vadState = vadStateEnded
			if endOfSpeech.cancelPrediction != nil {
				endOfSpeech.cancelPrediction()
			}
			endOfSpeech.state.segment.Revision++
			endOfSpeech.state.lastSpeechEnd = time.Now()
			if packet.EndAt > 0 && packet.EndAt <= float64(endOfSpeech.audioSamples)/livekitAudioSampleRate {
				stopSilence := float64(endOfSpeech.audioSamples)/livekitAudioSampleRate - packet.EndAt
				endOfSpeech.state.lastSpeechEnd = endOfSpeech.state.lastSpeechEnd.Add(-time.Duration(stopSilence * float64(time.Second)))
			}
			if endOfSpeech.state.transcript == transcriptStateUserText {
				endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision}
				endOfSpeech.state.transcript = transcriptStateIdle
				endOfSpeech.state.started = false
			}
			if endOfSpeech.state.segment.FinalText != "" {
				segment := endOfSpeech.state.segment
				segment.Text = segment.FinalText
				deadline := endOfSpeech.state.lastSpeechEnd
				endOfSpeech.enqueueCommand(workerCommand{
					ctx: ctx, segment: segment, deadline: deadline, predict: true,
				})
				return nil
			}
		}
		return nil

	case internal_type.SpeechToTextPacket:
		if packet.Script == "" {
			return nil
		}
		if endOfSpeech.state.transcript == transcriptStateUserText {
			endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision + 1}
			endOfSpeech.state.started = false
		}
		if packet.Interim {
			previous := endOfSpeech.state.segment
			timestamp := time.Now()
			if previous.FinalText != "" && !previous.Timestamp.IsZero() {
				timestamp = previous.Timestamp
			}
			segment := speechSegment{
				Revision:  previous.Revision,
				ContextID: packet.ContextId(),
				FinalText: previous.FinalText,
				Timestamp: timestamp,
				Chunks:    append([]internal_type.SpeechToTextPacket(nil), previous.Chunks...),
			}
			segment.Chunks = append(segment.Chunks, packet)
			segment.Text = segment.FinalText
			if segment.Text != "" {
				segment.Text += packet.GetConcat()
			}
			segment.Text += packet.Script
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
			endOfSpeech.enqueueCommand(workerCommand{ctx: ctx, segment: segment, emitInterim: true, emitStarted: emitStarted})
			return nil
		}

		// Final transcript: accumulate text
		if endOfSpeech.cancelPrediction != nil {
			endOfSpeech.cancelPrediction()
		}
		previous := endOfSpeech.state.segment
		segment := speechSegment{
			Revision:  previous.Revision + 1,
			ContextID: packet.ContextId(),
			FinalText: previous.FinalText,
			Timestamp: time.Now(),
			Chunks:    append([]internal_type.SpeechToTextPacket(nil), previous.Chunks...),
		}
		segment.Chunks = append(segment.Chunks, packet)
		if segment.FinalText != "" {
			segment.FinalText += packet.GetConcat()
		}
		segment.FinalText += packet.Script
		segment.Text = segment.FinalText
		emitStarted := segment.Text != "" && !endOfSpeech.state.started
		if emitStarted {
			endOfSpeech.state.started = true
		}
		endOfSpeech.state.transcript = transcriptStateFinalized
		endOfSpeech.state.segment = segment
		endOfSpeech.state.confidence = 0
		shouldPredict := endOfSpeech.state.vadState != vadStateSpeaking
		if endOfSpeech.state.vadState == vadStateIdle {
			endOfSpeech.state.lastSpeechEnd = time.Now()
		}
		deadline := endOfSpeech.state.lastSpeechEnd
		segment.Text = segment.FinalText
		endOfSpeech.enqueueCommand(workerCommand{
			ctx: ctx, segment: segment, deadline: deadline, predict: shouldPredict, emitInterim: true, emitStarted: emitStarted,
		})
		return nil

	case internal_type.LLMResponseDonePacket:
		if packet.Text != "" {
			endOfSpeech.history = append(endOfSpeech.history, chatMessage{Role: "assistant", Content: packet.Text})
		}
	}

	return nil
}

func (endOfSpeech *livekitEndOfSpeech) predictEOU(ctx context.Context, currentText string) (float64, error) {
	endOfSpeech.mu.RLock()
	chatText := formatChatTemplateFromHistory(
		endOfSpeech.history,
		currentText,
		endOfSpeech.maxHistory,
		endOfSpeech.modelType,
	)
	endOfSpeech.mu.RUnlock()

	if chatText == "" {
		return 0, errTurnDetectorEmptyTokenSequence
	}

	endOfSpeech.predictorMu.Lock()
	defer endOfSpeech.predictorMu.Unlock()

	if endOfSpeech.predictor == nil {
		return 0, errTurnDetectorNil
	}

	probability, err := endOfSpeech.predictor.PredictContext(ctx, chatText)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errTurnDetectorRunInference, err)
	}

	if endOfSpeech.logger != nil {
		endOfSpeech.logger.Debugf("livekit_eos: P(eou)=%.4f threshold=%.4f text=%q", probability, endOfSpeech.threshold, currentText)
	}

	return probability, nil
}

// enqueueCommand publishes state-ordered work while mu is held; callbacks may reenter Execute.
func (endOfSpeech *livekitEndOfSpeech) enqueueCommand(command workerCommand) {
	if endOfSpeech == nil || endOfSpeech.commandCh == nil || endOfSpeech.stopCh == nil {
		return
	}

	select {
	case <-endOfSpeech.stopCh:
		return
	default:
	}

	endOfSpeech.commands = append(endOfSpeech.commands, command)
	select {
	case endOfSpeech.commandCh <- struct{}{}:
	default:
	}
}

func (endOfSpeech *livekitEndOfSpeech) worker() {
	if endOfSpeech.workerDone != nil {
		defer close(endOfSpeech.workerDone)
	}
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
		// Retire the completed revision before callbacks can submit another turn.
		endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision + 1}
		endOfSpeech.state.confidence = 0
		endOfSpeech.state.started = false
		endOfSpeech.state.vadState = vadStateIdle
		endOfSpeech.state.transcript = transcriptStateIdle
		endOfSpeech.state.lastSpeechEnd = time.Time{}
	}

	for {
		select {
		case <-endOfSpeech.stopCh:
			stopTimer()
			return

		case <-endOfSpeech.commandCh:
			endOfSpeech.mu.Lock()
			if len(endOfSpeech.commands) == 0 {
				endOfSpeech.mu.Unlock()
				continue
			}
			command := endOfSpeech.commands[0]
			endOfSpeech.commands[0] = workerCommand{}
			endOfSpeech.commands = endOfSpeech.commands[1:]
			if len(endOfSpeech.commands) > 0 {
				select {
				case endOfSpeech.commandCh <- struct{}{}:
				default:
				}
			} else {
				endOfSpeech.commands = nil
			}
			if endOfSpeech.closed || command.ctx.Err() != nil {
				endOfSpeech.mu.Unlock()
				continue
			}
			if command.emitInterim {
				endOfSpeech.mu.Unlock()
				if command.emitStarted {
					_ = endOfSpeech.onPacket(command.ctx,
						internal_type.InterimEndOfSpeechPacket{Speech: command.segment.Text, ContextID: command.segment.ContextID},
						internal_type.ObservabilityEventRecordPacket{
							ContextID: command.segment.ContextID,
							Scope:     internal_type.ObservabilityRecordScopeUserMessage,
							Record: observability.RecordEvent{
								Component: observability.ComponentEOS, Event: observability.EOSStarted, OccurredAt: time.Now(),
								Attributes: observability.Attributes{
									"provider": endOfSpeech.Name(), "context_id": command.segment.ContextID, "speech": command.segment.Text,
								},
							},
						},
					)
				} else {
					_ = endOfSpeech.onPacket(command.ctx, internal_type.InterimEndOfSpeechPacket{
						Speech: command.segment.Text, ContextID: command.segment.ContextID,
					})
				}
				if !command.predict && !command.fireImmediately {
					continue
				}
				endOfSpeech.mu.Lock()
				if endOfSpeech.closed || command.ctx.Err() != nil {
					endOfSpeech.mu.Unlock()
					continue
				}
			}

			// Newer text/STT packets supersede older timer commands. Completing a
			// stale snapshot can shrink the final user message.
			if !command.fireImmediately && command.segment.Revision != endOfSpeech.state.segment.Revision {
				endOfSpeech.mu.Unlock()
				continue
			}

			if command.fireImmediately {
				if command.segment.Revision == endOfSpeech.state.segment.Revision {
					stopTimer()
					resetState()
				}
				endOfSpeech.history = append(endOfSpeech.history, chatMessage{Role: "user", Content: command.segment.Text})
				endOfSpeech.mu.Unlock()
				endOfSpeech.fire(command, time.Now())
				continue
			}

			if endOfSpeech.state.vadState == vadStateSpeaking || endOfSpeech.state.transcript == transcriptStateUserText {
				endOfSpeech.mu.Unlock()
				continue
			}
			if command.predict {
				stopTimer()
				predictionContext, cancelPrediction := context.WithTimeout(command.ctx, predictionTimeout)
				endOfSpeech.cancelPrediction = cancelPrediction
				endOfSpeech.mu.Unlock()
				probability, predictionError := endOfSpeech.predictEOU(predictionContext, command.segment.Text)
				cancelPrediction()
				if predictionError != nil && !errors.Is(predictionError, context.Canceled) && !errors.Is(predictionError, context.DeadlineExceeded) {
					_ = endOfSpeech.onPacket(command.ctx, internal_type.ObservabilityLogRecordPacket{
						ContextID: command.segment.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeConversation,
						Record: observability.RecordLog{
							Level:      observability.LevelError,
							Message:    "turn prediction failed",
							OccurredAt: time.Now(),
							Attributes: observability.Attributes{
								"component": observability.ComponentEOS.String(),
								"provider":  endOfSpeech.Name(),
								"operation": "predict_end_of_turn",
								"error":     predictionError.Error(),
							},
						},
					})
				}
				endOfSpeech.mu.Lock()
				endOfSpeech.cancelPrediction = nil
				if endOfSpeech.closed || command.ctx.Err() != nil || command.segment.Revision != endOfSpeech.state.segment.Revision {
					endOfSpeech.mu.Unlock()
					continue
				}
				command.confidence = probability
				endOfSpeech.state.confidence = probability
				if predictionError != nil || probability >= endOfSpeech.threshold {
					command.deadline = command.deadline.Add(endOfSpeech.quickTimeout)
				} else {
					command.deadline = command.deadline.Add(endOfSpeech.silenceTimeout)
				}
			}
			currentCommand = command
			stopTimer()
			timerArmedAt = time.Now()
			timeout := command.timeout
			if !command.deadline.IsZero() {
				timeout = max(0, time.Until(command.deadline))
			}
			timer = time.NewTimer(timeout)
			timerCh = timer.C
			endOfSpeech.mu.Unlock()

		case <-timerCh:
			endOfSpeech.mu.Lock()
			if len(endOfSpeech.commands) > 0 && !endOfSpeech.closed {
				// Deliver admitted previews before the corresponding final callback.
				timer.Reset(0)
				endOfSpeech.mu.Unlock()
				continue
			}
			if endOfSpeech.closed || currentCommand.ctx.Err() != nil {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			if currentCommand.segment.Revision != endOfSpeech.state.segment.Revision {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			if endOfSpeech.state.vadState == vadStateSpeaking {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			if endOfSpeech.state.segment.FinalText == "" || endOfSpeech.state.transcript == transcriptStateUserText {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}

			command := currentCommand
			command.segment = endOfSpeech.state.segment
			command.segment.Text = command.segment.FinalText
			armedAt := timerArmedAt
			stopTimer()
			resetState()
			endOfSpeech.history = append(endOfSpeech.history, chatMessage{Role: "user", Content: command.segment.Text})
			endOfSpeech.mu.Unlock()
			endOfSpeech.fire(command, armedAt)
		}
	}
}

func (endOfSpeech *livekitEndOfSpeech) fire(command workerCommand, timerArmedAt time.Time) {
	if endOfSpeech == nil {
		return
	}

	ctx := command.ctx
	segment := command.segment
	confidence := command.confidence
	speech := segment.Text
	if speech == "" {
		return
	}

	if ctx.Err() != nil {
		return
	}
	if endOfSpeech.onPacket == nil {
		return
	}

	wordCount := len(strings.Fields(speech))
	triggerAt := time.Now()
	textToTriggerMs := triggerAt.Sub(segment.Timestamp).Milliseconds()
	waitToTriggerMs := textToTriggerMs
	if !timerArmedAt.IsZero() {
		waitToTriggerMs = triggerAt.Sub(timerArmedAt).Milliseconds()
	}
	_ = endOfSpeech.onPacket(ctx,
		internal_type.EndOfSpeechPacket{
			Speech:    speech,
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
					"provider":           eosName,
					"context_id":         segment.ContextID,
					"speech":             speech,
					"confidence":         fmt.Sprintf("%.4f", confidence),
					"word_count":         fmt.Sprintf("%d", wordCount),
					"char_count":         fmt.Sprintf("%d", len(speech)),
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
					{Name: observability.MetricEOSConfidence, Value: fmt.Sprintf("%.4f", confidence)},
				},
			},
		})
}

// Close cancels pending work and waits for worker shutdown and native resource cleanup.
// If ctx expires, cleanup continues; a later Close can wait for its completion.
func (endOfSpeech *livekitEndOfSpeech) Close(ctx context.Context) error {
	if endOfSpeech == nil {
		return nil
	}

	endOfSpeech.closeOnce.Do(func() {
		endOfSpeech.mu.Lock()
		endOfSpeech.closed = true
		clear(endOfSpeech.commands)
		endOfSpeech.commands = nil
		endOfSpeech.closedCh = make(chan struct{})
		if endOfSpeech.cancelPrediction != nil {
			endOfSpeech.cancelPrediction()
		}
		if endOfSpeech.stopCh != nil {
			close(endOfSpeech.stopCh)
		}
		eosStartedAt := endOfSpeech.eosStartedAt
		endOfSpeech.eosStartedAt = time.Time{}
		endOfSpeech.mu.Unlock()

		// Cleanup keeps ownership if the closing caller's deadline expires first.
		go func() {
			defer close(endOfSpeech.closedCh)
			if endOfSpeech.workerDone != nil {
				<-endOfSpeech.workerDone
			}
			endOfSpeech.predictorMu.Lock()
			if predictor, ok := endOfSpeech.predictor.(interface{ Destroy() }); ok {
				predictor.Destroy()
			}
			endOfSpeech.predictor = nil
			endOfSpeech.predictorMu.Unlock()

			if endOfSpeech.onPacket != nil {
				if !eosStartedAt.IsZero() {
					endOfSpeech.onPacket(ctx, internal_type.ObservabilityUsageRecordPacket{
						Scope: internal_type.ObservabilityRecordScopeConversation,
						Record: observability.NewEOSDurationUsageRecord(endOfSpeech.Name(), time.Since(eosStartedAt), observability.Attributes{
							"provider": endOfSpeech.Name(),
						}),
					})
				}
				endOfSpeech.onPacket(ctx, internal_type.ObservabilityEventRecordPacket{
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

		}()
	})

	select {
	case <-endOfSpeech.closedCh:
		return nil
	default:
	}
	select {
	case <-endOfSpeech.closedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
