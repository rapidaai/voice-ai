// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

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
	eosName = "livekitEndOfSpeech"

	optKeyThreshold       = "microphone.eos.threshold"
	optKeyQuickTimeout    = "microphone.eos.quick_timeout"
	optKeyExtendedTimeout = "microphone.eos.extended_timeout"
	optKeyFallbackTimeout = "microphone.eos.fallback_timeout"
	optKeyMaxHistory      = "microphone.eos.max_history_turns"

	// Backward-compatible aliases.
	optKeyLegacySilenceTimeout = "microphone.eos.silence_timeout"
	optKeyLegacyTimeout        = "microphone.eos.timeout"

	// defaultThreshold is the English "unlikely_threshold" from LiveKit's
	// languages.json. Probabilities below this → user still speaking.
	defaultThreshold = 0.0289

	// defaultSilenceTimeout (max_endpointing_delay) — used when model predicts
	// user is still speaking (prob < threshold). LiveKit default: 3.0s.
	defaultSilenceTimeout = 3000.0

	// defaultQuickTimeout — short buffer after model says YES before firing.
	defaultQuickTimeout = 250.0

	// defaultMaxHistory matches LiveKit's MAX_HISTORY_TURNS = 6.
	defaultMaxHistory = 6.0

	// defaultFallbackTimeout is the silence timeout for interim STT and inference failures.
	defaultFallbackTimeout = 500.0
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
	Timestamp time.Time
	Chunks    []internal_type.SpeechToTextPacket
}

type workerCommand struct {
	ctx             context.Context
	timeout         time.Duration
	segment         speechSegment
	confidence      float64
	fireImmediately bool
}

type endOfSpeechState struct {
	segment       speechSegment
	pending       *workerCommand
	confidence    float64
	callbackFired bool
	vadState      vadState
	transcript    transcriptState
}

type turnPredictor interface {
	Predict(string) (float64, error)
}

// livekitEndOfSpeech detects end-of-speech using the LiveKit turn detector model
// with a hybrid approach: ONNX inference determines whether to use a quick
// or extended silence timeout, with fallback to standard silence on failure.
//
// Conversation history is built internally from packets flowing through
// Execute — user turns are recorded when EOS fires, and assistant turns
// are recorded from LLMResponseDonePacket.
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
	threshold       float64
	quickTimeout    time.Duration
	silenceTimeout  time.Duration
	fallbackTimeout time.Duration
	maxHistory      int

	// Worker orchestration
	commandCh chan workerCommand
	stopCh    chan struct{}
	closeOnce sync.Once

	// State
	mu           sync.RWMutex
	state        *endOfSpeechState
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
		return nil, fmt.Errorf("%s: onPacket is required", eosName)
	}
	start := time.Now()

	cfg := TurnDetectorConfig{ModelType: "en"}
	if v, err := options.options.GetString("microphone.eos.model"); err == nil && v != "" {
		cfg.ModelType = v
	}
	if v, err := options.options.GetString("microphone.eos.livekit.model_path"); err == nil {
		cfg.ModelPath = v
	}
	if v, err := options.options.GetString("microphone.eos.livekit.tokenizer_path"); err == nil {
		cfg.TokenizerPath = v
	}

	detector, err := NewTurnDetector(cfg)
	if err != nil {
		_ = options.onPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", eosName, err.Error()),
				Attributes: observability.Attributes{
					"component": observability.ComponentEOS.String(),
					"provider":  eosName,
					"options":   observability.AttributeValue(options.options),
				},
				OccurredAt: time.Now(),
			},
		})
		return nil, fmt.Errorf("livekit_eos: init turn detector: %w", err)
	}

	endOfSpeech := &livekitEndOfSpeech{
		logger:          options.logger,
		onPacket:        options.onPacket,
		opts:            options.options,
		predictor:       detector,
		threshold:       defaultThreshold,
		quickTimeout:    time.Duration(defaultQuickTimeout) * time.Millisecond,
		silenceTimeout:  time.Duration(defaultSilenceTimeout) * time.Millisecond,
		fallbackTimeout: time.Duration(defaultFallbackTimeout) * time.Millisecond,
		maxHistory:      int(defaultMaxHistory),
		commandCh:       make(chan workerCommand, 32),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
		eosStartedAt:    time.Now(),
	}

	if v, err := options.options.GetFloat64(optKeyThreshold); err == nil {
		endOfSpeech.threshold = v
	}
	if v, err := options.options.GetFloat64(optKeyExtendedTimeout); err == nil {
		endOfSpeech.silenceTimeout = time.Duration(v) * time.Millisecond
	} else if v, err := options.options.GetFloat64(optKeyLegacySilenceTimeout); err == nil {
		endOfSpeech.silenceTimeout = time.Duration(v) * time.Millisecond
	}
	if v, err := options.options.GetFloat64(optKeyQuickTimeout); err == nil {
		endOfSpeech.quickTimeout = time.Duration(v) * time.Millisecond
	}
	if v, err := options.options.GetFloat64(optKeyMaxHistory); err == nil {
		endOfSpeech.maxHistory = int(v)
	}
	if v, err := options.options.GetFloat64(optKeyFallbackTimeout); err == nil {
		endOfSpeech.fallbackTimeout = time.Duration(v) * time.Millisecond
	} else if v, err := options.options.GetFloat64(optKeyLegacyTimeout); err == nil {
		endOfSpeech.fallbackTimeout = time.Duration(v) * time.Millisecond
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

func (endOfSpeech *livekitEndOfSpeech) Name() string {
	return eosName
}

func (endOfSpeech *livekitEndOfSpeech) Options() utils.Option {
	return endOfSpeech.opts
}

func (endOfSpeech *livekitEndOfSpeech) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (endOfSpeech *livekitEndOfSpeech) Execute(ctx context.Context, packet internal_type.Packet) error {
	switch packet := packet.(type) {
	case internal_type.EndOfSpeechAudioPacket:
		return nil
	case internal_type.UserTextReceivedPacket:
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
		endOfSpeech.state.confidence = 0
		endOfSpeech.state.transcript = transcriptStateFinalized
		endOfSpeech.mu.Unlock()

		packets := []internal_type.Packet{internal_type.InterimEndOfSpeechPacket{
			Speech:    command.segment.Text,
			ContextID: command.segment.ContextID,
		}}
		_ = endOfSpeech.onPacket(ctx, packets...)
		endOfSpeech.enqueueCommand(command)

	case internal_type.EndOfSpeechInterruptionPacket:
		endOfSpeech.mu.RLock()
		command := workerCommand{
			ctx:        ctx,
			segment:    endOfSpeech.state.segment,
			confidence: endOfSpeech.state.confidence,
			timeout:    endOfSpeech.silenceTimeout,
		}
		endOfSpeech.mu.RUnlock()
		if command.segment.Text == "" {
			return nil
		}
		endOfSpeech.enqueueCommand(command)

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
					ctx:        ctx,
					segment:    endOfSpeech.state.segment,
					confidence: endOfSpeech.state.confidence,
					timeout:    endOfSpeech.quickTimeout,
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
					ctx:        ctx,
					segment:    endOfSpeech.state.segment,
					confidence: endOfSpeech.state.confidence,
					timeout:    endOfSpeech.fallbackTimeout,
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
		endOfSpeech.mu.Lock()
		if packet.Interim {
			if packet.Script == "" {
				endOfSpeech.mu.Unlock()
				return nil
			}
			previous := endOfSpeech.state.segment
			timestamp := time.Now()
			if previous.FinalText != "" && !previous.Timestamp.IsZero() {
				timestamp = previous.Timestamp
			}
			segment := speechSegment{
				Revision:  previous.Revision + 1,
				ContextID: packet.ContextId(),
				Timestamp: timestamp,
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
			if previous.FinalText == "" {
				endOfSpeech.state.transcript = transcriptStateInterimPending
			} else {
				endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
			}
			endOfSpeech.state.segment = segment
			if endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim ||
				endOfSpeech.state.vadState == vadStateEnded {
				command := workerCommand{
					ctx:        ctx,
					segment:    segment,
					confidence: endOfSpeech.state.confidence,
					timeout:    endOfSpeech.fallbackTimeout,
				}
				endOfSpeech.mu.Unlock()

				_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
					Speech:    command.segment.Text,
					ContextID: command.segment.ContextID,
				})
				endOfSpeech.enqueueCommand(command)
				return nil
			}
			endOfSpeech.mu.Unlock()

			_ = endOfSpeech.onPacket(ctx, internal_type.InterimEndOfSpeechPacket{
				Speech:    segment.Text,
				ContextID: segment.ContextID,
			})
			return nil
		}

		// Final transcript: accumulate text
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
		endOfSpeech.state.transcript = transcriptStateFinalized
		if segment.Text != segment.FinalText {
			endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
		}
		endOfSpeech.state.segment = segment
		endOfSpeech.state.confidence = 0
		fullText := segment.Text
		if endOfSpeech.state.vadState == vadStateEnded {
			timeout := endOfSpeech.quickTimeout
			if endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim {
				timeout = endOfSpeech.fallbackTimeout
			}
			command := workerCommand{
				ctx:     ctx,
				segment: segment,
				timeout: timeout,
			}
			endOfSpeech.mu.Unlock()

			if fullText == "" {
				return nil
			}
			packets := []internal_type.Packet{internal_type.InterimEndOfSpeechPacket{
				Speech:    fullText,
				ContextID: command.segment.ContextID,
			}}
			_ = endOfSpeech.onPacket(ctx, packets...)
			endOfSpeech.enqueueCommand(command)
			return nil
		}
		endOfSpeech.mu.Unlock()

		if fullText == "" {
			return nil
		}

		// Emit interim update (same as silence-based on final STT)
		packets := []internal_type.Packet{internal_type.InterimEndOfSpeechPacket{
			Speech:    fullText,
			ContextID: segment.ContextID,
		}}
		_ = endOfSpeech.onPacket(ctx, packets...)

		// Run model inference on accumulated final text.
		// YES (prob >= threshold) → quick_timeout buffer, then fire.
		// NO  (prob <  threshold) → keep accumulating, safety timer as fallback.
		probability := endOfSpeech.predictEOU(fullText)
		if probability < 0 {
			endOfSpeech.enqueueCommand(workerCommand{
				ctx:     ctx,
				segment: segment,
				timeout: endOfSpeech.fallbackTimeout,
			})
			return nil
		}

		endOfSpeech.mu.Lock()
		if endOfSpeech.state.segment.Revision == segment.Revision {
			endOfSpeech.state.confidence = probability
		}
		endOfSpeech.mu.Unlock()

		if probability >= endOfSpeech.threshold {
			endOfSpeech.enqueueCommand(workerCommand{
				ctx:        ctx,
				segment:    segment,
				confidence: probability,
				timeout:    endOfSpeech.quickTimeout,
			})
			return nil
		}

		endOfSpeech.enqueueCommand(workerCommand{
			ctx:        ctx,
			segment:    segment,
			confidence: probability,
			timeout:    endOfSpeech.silenceTimeout,
		})
		return nil

	case internal_type.LLMResponseDonePacket:
		if packet.Text != "" {
			endOfSpeech.mu.Lock()
			endOfSpeech.history = append(endOfSpeech.history, chatMessage{Role: "assistant", Content: packet.Text})
			endOfSpeech.mu.Unlock()
		}
	}

	return nil
}

// predictEOU runs the turn detection model and returns the end-of-utterance
// probability. Returns -1 on failure (caller should treat as "not done").
func (endOfSpeech *livekitEndOfSpeech) predictEOU(currentText string) float64 {
	endOfSpeech.mu.RLock()
	history := make([]chatMessage, len(endOfSpeech.history))
	copy(history, endOfSpeech.history)
	endOfSpeech.mu.RUnlock()

	chatText := formatChatTemplateFromHistory(history, currentText, endOfSpeech.maxHistory)
	if chatText == "" {
		return -1
	}

	endOfSpeech.predictorMu.Lock()
	defer endOfSpeech.predictorMu.Unlock()

	if endOfSpeech.predictor == nil {
		return -1
	}

	probability, err := endOfSpeech.predictor.Predict(chatText)
	if err != nil {
		if endOfSpeech.logger != nil {
			endOfSpeech.logger.Debugf("livekit_eos: inference failed: %v", err)
		}
		return -1
	}

	if endOfSpeech.logger != nil {
		endOfSpeech.logger.Debugf("livekit_eos: P(eou)=%.4f threshold=%.4f text=%q", probability, endOfSpeech.threshold, currentText)
	}

	return probability
}

func (endOfSpeech *livekitEndOfSpeech) enqueueCommand(command workerCommand) {
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

func (endOfSpeech *livekitEndOfSpeech) worker() {
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
		endOfSpeech.state.confidence = 0
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
				endOfSpeech.fire(command, time.Now())
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
					timer = time.NewTimer(endOfSpeech.fallbackTimeout)
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
				endOfSpeech.fire(command, armedAt)
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
			endOfSpeech.fire(command, armedAt)
			endOfSpeech.mu.Lock()
			resetState()
			endOfSpeech.mu.Unlock()
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

	// Record user turn in conversation history
	endOfSpeech.mu.Lock()
	endOfSpeech.history = append(endOfSpeech.history, chatMessage{Role: "user", Content: speech})
	endOfSpeech.mu.Unlock()

	if confidence < 0 {
		confidence = 0
	}
	if ctx != nil && ctx.Err() != nil {
		ctx = context.Background()
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
				OccurredAt: triggerAt,
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
				OccurredAt: triggerAt,
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

func (endOfSpeech *livekitEndOfSpeech) Close(ctx context.Context) error {
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
				endOfSpeech.onPacket(ctx, internal_type.ObservabilityUsageRecordPacket{
					Scope:  internal_type.ObservabilityRecordScopeConversation,
					Record: observability.NewEOSDurationUsageRecord(endOfSpeech.Name(), time.Since(eosStartedAt), observability.Attributes{}),
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

		if endOfSpeech.stopCh != nil {
			close(endOfSpeech.stopCh)
		}
		endOfSpeech.predictorMu.Lock()
		if predictor, ok := endOfSpeech.predictor.(interface{ Destroy() }); ok {
			predictor.Destroy()
		}
		endOfSpeech.predictor = nil
		endOfSpeech.predictorMu.Unlock()
	})

	return nil
}
