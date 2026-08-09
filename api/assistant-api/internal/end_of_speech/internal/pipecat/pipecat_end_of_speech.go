// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"encoding/binary"
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
	pipecatEndOfSpeechName = "pipecatSmartTurnEndOfSpeech"
	optPctThreshold        = "microphone.eos.threshold"
	optPctExtendedTimeout  = "microphone.eos.extended_timeout"
	optPctQuickTimeout     = "microphone.eos.quick_timeout"
	optPctFallbackTimeout  = "microphone.eos.fallback_timeout"

	optPctLegacySilenceTimeout = "microphone.eos.silence_timeout"
	optPctLegacyTimeout        = "microphone.eos.timeout"

	defaultPctThreshold       = 0.5
	defaultPctQuickTimeout    = 250.0
	defaultPctExtendedTimeout = 2000.0
	defaultPctFallbackTimeout = 500.0

	maxAudioSamples = whisperMaxSamples
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
	// Timestamp tracks the last text-bearing update for this segment.
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
	Predict([]float32) (float64, error)
}

type pipecatEndOfSpeech struct {
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	opts     utils.Option

	predictor   turnPredictor
	predictorMu sync.Mutex

	threshold       float64
	quickTimeout    time.Duration
	extendedTimeout time.Duration
	fallbackTimeout time.Duration

	audioBuffer []float32

	audioGeneration      uint64
	predictedGeneration  uint64
	predictedProbability float64
	hasPredictedResult   bool

	commandCh chan workerCommand
	stopCh    chan struct{}
	closeOnce sync.Once

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
		return nil, fmt.Errorf("%s: onPacket is required", pipecatEndOfSpeechName)
	}
	start := time.Now()

	detectorConfig := PipecatDetectorConfig{}
	if modelPath, err := options.options.GetString("microphone.eos.pipecat.model_path"); err == nil {
		detectorConfig.ModelPath = modelPath
	}

	detector, err := NewPipecatDetector(detectorConfig)
	if err != nil {
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: fmt.Sprintf("%s: error while initialization %s", pipecatEndOfSpeechName, err.Error()),
					Attributes: observability.Attributes{
						"component": observability.ComponentEOS.String(),
						"provider":  pipecatEndOfSpeechName,
						"options":   observability.AttributeValue(options.options),
					},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil, fmt.Errorf("pipecat_eos: init detector: %w", err)
	}

	endOfSpeech := &pipecatEndOfSpeech{
		logger:          options.logger,
		onPacket:        options.onPacket,
		opts:            options.options,
		predictor:       detector,
		threshold:       defaultPctThreshold,
		quickTimeout:    time.Duration(defaultPctQuickTimeout) * time.Millisecond,
		extendedTimeout: time.Duration(defaultPctExtendedTimeout) * time.Millisecond,
		fallbackTimeout: time.Duration(defaultPctFallbackTimeout) * time.Millisecond,
		audioBuffer:     make([]float32, 0, maxAudioSamples),
		commandCh:       make(chan workerCommand, 32),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
		eosStartedAt:    time.Now(),
	}

	if threshold, err := options.options.GetFloat64(optPctThreshold); err == nil {
		endOfSpeech.threshold = threshold
	}
	if extendedTimeout, err := options.options.GetFloat64(optPctExtendedTimeout); err == nil {
		endOfSpeech.extendedTimeout = time.Duration(extendedTimeout) * time.Millisecond
	} else if extendedTimeout, err := options.options.GetFloat64(optPctLegacySilenceTimeout); err == nil {
		endOfSpeech.extendedTimeout = time.Duration(extendedTimeout) * time.Millisecond
	}
	if quickTimeout, err := options.options.GetFloat64(optPctQuickTimeout); err == nil {
		endOfSpeech.quickTimeout = time.Duration(quickTimeout) * time.Millisecond
	}
	if fallbackTimeout, err := options.options.GetFloat64(optPctFallbackTimeout); err == nil {
		endOfSpeech.fallbackTimeout = time.Duration(fallbackTimeout) * time.Millisecond
	} else if fallbackTimeout, err := options.options.GetFloat64(optPctLegacyTimeout); err == nil {
		endOfSpeech.fallbackTimeout = time.Duration(fallbackTimeout) * time.Millisecond
	}

	go endOfSpeech.worker()
	_ = endOfSpeech.onPacket(options.ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewMetricEOSInitLatencyMs(time.Since(start), observability.Attributes{"provider": endOfSpeech.Name()}),
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

func (endOfSpeech *pipecatEndOfSpeech) Name() string {
	return pipecatEndOfSpeechName
}

func (endOfSpeech *pipecatEndOfSpeech) Options() utils.Option {
	return endOfSpeech.opts
}

func (endOfSpeech *pipecatEndOfSpeech) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (endOfSpeech *pipecatEndOfSpeech) Execute(ctx context.Context, packet internal_type.Packet) error {
	switch packet := packet.(type) {
	case internal_type.EndOfSpeechAudioPacket:
		endOfSpeech.handleAudioPacket(packet)
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
		return endOfSpeech.handleSpeechToTextPacket(ctx, packet)
	}

	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) handleAudioPacket(packet internal_type.EndOfSpeechAudioPacket) {
	endOfSpeech.appendAudio(packet.Audio)
}

func (endOfSpeech *pipecatEndOfSpeech) handleUserTextPacket(ctx context.Context, packet internal_type.UserTextReceivedPacket) error {
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

	endOfSpeech.emitInterimSpeech(ctx, command.segment)
	endOfSpeech.enqueueCommand(command)

	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) handleInterruptionPacket(ctx context.Context) error {
	return endOfSpeech.extendCurrentSegment(ctx, endOfSpeech.extendedTimeout)
}

func (endOfSpeech *pipecatEndOfSpeech) handleSpeechToTextPacket(ctx context.Context, packet internal_type.SpeechToTextPacket) error {
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

			endOfSpeech.emitInterimSpeech(ctx, command.segment)
			endOfSpeech.enqueueCommand(command)
			return nil
		}
		endOfSpeech.mu.Unlock()

		endOfSpeech.emitInterimSpeech(ctx, segment)
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
	endOfSpeech.state.transcript = transcriptStateFinalized
	if segment.Text != segment.FinalText {
		endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
	}
	endOfSpeech.state.segment = segment
	endOfSpeech.state.confidence = 0
	if endOfSpeech.state.vadState == vadStateEnded {
		timeout := endOfSpeech.quickTimeout
		if endOfSpeech.state.transcript == transcriptStateFinalizedWithPendingInterim {
			timeout = endOfSpeech.fallbackTimeout
		}
		command := workerCommand{
			ctx:        ctx,
			segment:    segment,
			confidence: 0,
			timeout:    timeout,
		}
		endOfSpeech.mu.Unlock()

		if command.segment.Text == "" {
			return nil
		}
		endOfSpeech.emitInterimSpeech(ctx, command.segment)
		endOfSpeech.enqueueCommand(command)
		return nil
	}
	endOfSpeech.mu.Unlock()

	if segment.Text == "" {
		return nil
	}

	endOfSpeech.emitInterimSpeech(ctx, segment)

	probability := endOfSpeech.predictEOU()
	confidence := 0.0
	if probability >= 0 {
		confidence = probability
		endOfSpeech.mu.Lock()
		if endOfSpeech.state.segment.Revision == segment.Revision {
			endOfSpeech.state.confidence = confidence
		}
		endOfSpeech.mu.Unlock()
	}

	switch {
	case probability < 0:
		endOfSpeech.enqueueCommand(workerCommand{
			ctx:        ctx,
			segment:    segment,
			confidence: confidence,
			timeout:    endOfSpeech.fallbackTimeout,
		})
	case probability >= endOfSpeech.threshold:
		endOfSpeech.enqueueCommand(workerCommand{
			ctx:        ctx,
			segment:    segment,
			confidence: confidence,
			timeout:    endOfSpeech.quickTimeout,
		})
	default:
		endOfSpeech.enqueueCommand(workerCommand{
			ctx:        ctx,
			segment:    segment,
			confidence: confidence,
			timeout:    endOfSpeech.extendedTimeout,
		})
	}

	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) extendCurrentSegment(ctx context.Context, timeout time.Duration) error {
	endOfSpeech.mu.RLock()
	command := workerCommand{
		ctx:        ctx,
		segment:    endOfSpeech.state.segment,
		confidence: endOfSpeech.state.confidence,
		timeout:    timeout,
	}
	endOfSpeech.mu.RUnlock()

	if command.segment.Text == "" {
		return nil
	}

	endOfSpeech.enqueueCommand(command)

	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) emitInterimSpeech(ctx context.Context, segment speechSegment) {
	packets := []internal_type.Packet{internal_type.InterimEndOfSpeechPacket{
		Speech:    segment.Text,
		ContextID: segment.ContextID,
	}}
	_ = endOfSpeech.onPacket(ctx, packets...)
}

func (endOfSpeech *pipecatEndOfSpeech) appendAudio(pcm16 []byte) {
	if len(pcm16) < 2 {
		return
	}

	sampleCount := len(pcm16) / 2
	samples := make([]float32, sampleCount)
	for i := 0; i < sampleCount; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm16[i*2:]))
		samples[i] = float32(sample) / 32768.0
	}

	endOfSpeech.mu.Lock()
	endOfSpeech.audioBuffer = append(endOfSpeech.audioBuffer, samples...)
	if len(endOfSpeech.audioBuffer) > maxAudioSamples {
		excess := len(endOfSpeech.audioBuffer) - maxAudioSamples
		endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[excess:]
	}
	endOfSpeech.audioGeneration++
	endOfSpeech.mu.Unlock()
}

func (endOfSpeech *pipecatEndOfSpeech) predictEOU() float64 {
	endOfSpeech.mu.RLock()
	generation := endOfSpeech.audioGeneration
	if endOfSpeech.hasPredictedResult && endOfSpeech.predictedGeneration == generation {
		probability := endOfSpeech.predictedProbability
		endOfSpeech.mu.RUnlock()
		return probability
	}
	audio := make([]float32, len(endOfSpeech.audioBuffer))
	copy(audio, endOfSpeech.audioBuffer)
	endOfSpeech.mu.RUnlock()

	if len(audio) == 0 {
		endOfSpeech.debugf("pipecat_eos: inference skipped: empty audio buffer")
		return -1
	}

	endOfSpeech.predictorMu.Lock()
	defer endOfSpeech.predictorMu.Unlock()

	if endOfSpeech.predictor == nil {
		endOfSpeech.debugf("pipecat_eos: inference skipped: detector unavailable")
		return -1
	}

	probability, err := endOfSpeech.predictor.Predict(audio)
	if err != nil {
		endOfSpeech.debugf("pipecat_eos: inference failed: %v", err)
		return -1
	}

	endOfSpeech.debugf(
		"pipecat_eos: P(complete)=%.4f threshold=%.4f audio_samples=%d",
		probability,
		endOfSpeech.threshold,
		len(audio),
	)

	endOfSpeech.mu.Lock()
	if endOfSpeech.audioGeneration == generation {
		endOfSpeech.predictedGeneration = generation
		endOfSpeech.predictedProbability = probability
		endOfSpeech.hasPredictedResult = true
	}
	endOfSpeech.mu.Unlock()

	return probability
}

func (endOfSpeech *pipecatEndOfSpeech) debugf(format string, args ...interface{}) {
	if endOfSpeech.logger == nil {
		return
	}
	endOfSpeech.logger.Debugf(format, args...)
}

func (endOfSpeech *pipecatEndOfSpeech) enqueueCommand(command workerCommand) {
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

func (endOfSpeech *pipecatEndOfSpeech) worker() {
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
		endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[:0]
		endOfSpeech.audioGeneration++
		endOfSpeech.predictedGeneration = 0
		endOfSpeech.predictedProbability = 0
		endOfSpeech.hasPredictedResult = false
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

func (endOfSpeech *pipecatEndOfSpeech) emitEndOfSpeech(command workerCommand, timerArmedAt time.Time) {
	if endOfSpeech == nil || endOfSpeech.onPacket == nil {
		return
	}

	ctx := command.ctx
	segment := command.segment
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
				OccurredAt: triggerAt,
				Attributes: observability.Attributes{
					"provider":           endOfSpeech.Name(),
					"context_id":         segment.ContextID,
					"speech":             segment.Text,
					"confidence":         fmt.Sprintf("%.4f", command.confidence),
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
				OccurredAt: triggerAt,
				Attributes: observability.Attributes{
					"provider": endOfSpeech.Name(),
				},
				Metrics: []*protos.Metric{
					{Name: observability.MetricEOSLatencyMs, Value: fmt.Sprintf("%d", waitToTriggerMs)},
					{Name: observability.MetricEOSTextToTriggerMs, Value: fmt.Sprintf("%d", textToTriggerMs)},
					{Name: observability.MetricEOSWordCount, Value: fmt.Sprintf("%d", wordCount)},
					{Name: observability.MetricEOSCharCount, Value: fmt.Sprintf("%d", len(segment.Text))},
					{Name: observability.MetricEOSConfidence, Value: fmt.Sprintf("%.4f", command.confidence)},
				},
			},
		},
	)
}

func (endOfSpeech *pipecatEndOfSpeech) Close(ctx context.Context) error {
	if endOfSpeech == nil {
		return nil
	}

	endOfSpeech.closeOnce.Do(func() {
		endOfSpeech.mu.Lock()
		eosStartedAt := endOfSpeech.eosStartedAt
		endOfSpeech.eosStartedAt = time.Time{}
		endOfSpeech.mu.Unlock()

		if endOfSpeech.onPacket != nil {
			packets := []internal_type.Packet{}
			if !eosStartedAt.IsZero() {
				packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
					Scope:  internal_type.ObservabilityRecordScopeConversation,
					Record: observability.NewEOSDurationUsageRecord(endOfSpeech.Name(), time.Since(eosStartedAt), observability.Attributes{}),
				})
			}
			packets = append(packets, internal_type.ObservabilityEventRecordPacket{
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
			_ = endOfSpeech.onPacket(ctx, packets...)
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
