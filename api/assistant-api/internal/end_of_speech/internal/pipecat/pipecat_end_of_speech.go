// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"encoding/binary"
	"errors"
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

type vadState uint8

type transcriptState uint8

type turnState uint8

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
	deadline        time.Time
	segment         speechSegment
	confidence      float64
	fireImmediately bool
}

type predictionRequest struct {
	ctx             context.Context
	packetContext   context.Context
	cancel          context.CancelFunc
	contextID       string
	vadRevision     uint64
	audioGeneration uint64
	audio           []float32
}

type endOfSpeechState struct {
	segment            speechSegment
	confidence         float64
	started            bool
	vadState           vadState
	transcript         transcriptState
	turnState          turnState
	vadRevision        uint64
	transcriptDeadline time.Time
	turnStopDeadline   time.Time
	silenceSamples     uint64
}

type turnPredictor interface {
	PredictContext(context.Context, []float32) (float64, error)
}

type pipecatEndOfSpeech struct {
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	opts     utils.Option

	predictor   turnPredictor
	predictorMu sync.Mutex

	threshold       float64
	extendedTimeout time.Duration
	fallbackTimeout time.Duration
	turnStopTimeout time.Duration

	audioBuffer       []float32
	audioStartSample  uint64
	audioNextSample   uint64
	hasSpeechStart    bool
	speechStartSample uint64

	audioGeneration      uint64
	predictedGeneration  uint64
	predictedProbability float64
	hasPredictedResult   bool

	commandCh chan workerCommand
	stopCh    chan struct{}
	closeOnce sync.Once

	mu               sync.RWMutex
	state            *endOfSpeechState
	eosStartedAt     time.Time
	cancelPrediction context.CancelFunc
	predictionCh     chan predictionRequest
	predictionDone   chan struct{}
	closed           bool
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
		return nil, fmt.Errorf("%s: %w", pipecatEndOfSpeechName, errPipecatOnPacketRequired)
	}
	start := time.Now()

	detectorConfig := PipecatDetectorConfig{}
	if modelPath, err := options.options.GetString(optPctModelPath); err == nil {
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
		return nil, fmt.Errorf("%w: %w", errPipecatInitDetector, err)
	}

	endOfSpeech := &pipecatEndOfSpeech{
		logger:          options.logger,
		onPacket:        options.onPacket,
		opts:            options.options,
		predictor:       detector,
		threshold:       defaultPctThreshold,
		extendedTimeout: time.Duration(defaultPctExtendedTimeout) * time.Millisecond,
		fallbackTimeout: time.Duration(defaultPctFallbackTimeout) * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
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
	}
	if fallbackTimeout, err := options.options.GetFloat64(optPctFallbackTimeout); err == nil {
		endOfSpeech.fallbackTimeout = time.Duration(fallbackTimeout) * time.Millisecond
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
		endOfSpeech.appendAudio(packet.Audio)
		endOfSpeech.mu.Lock()
		if endOfSpeech.state.vadState != vadStateEnded || endOfSpeech.state.transcript == transcriptStateUserText ||
			endOfSpeech.state.turnState == turnStateComplete {
			endOfSpeech.mu.Unlock()
			return nil
		}
		// Slice lengths produce non-negative PCM16 sample counts.
		audioSampleCount, _ := utils.IntToUint64(len(packet.Audio) / 2)
		endOfSpeech.state.silenceSamples += audioSampleCount
		silenceSamples, conversionErr := utils.Uint64ToInt64(endOfSpeech.state.silenceSamples)
		if conversionErr == nil && endOfSpeech.state.silenceSamples <= maxAudioDurationSamples &&
			time.Duration(silenceSamples)*pipecatAudioSampleDuration < endOfSpeech.extendedTimeout {
			endOfSpeech.mu.Unlock()
			return nil
		}
		endOfSpeech.state.turnState = turnStateComplete
		endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[:0]
		endOfSpeech.audioStartSample = endOfSpeech.audioNextSample
		endOfSpeech.hasSpeechStart = false
		endOfSpeech.audioGeneration++
		endOfSpeech.hasPredictedResult = false
		command := workerCommand{
			ctx:        ctx,
			segment:    endOfSpeech.state.segment,
			confidence: endOfSpeech.state.confidence,
			deadline:   endOfSpeech.state.transcriptDeadline,
		}
		command.segment.Text = command.segment.FinalText
		endOfSpeech.mu.Unlock()
		if command.segment.Text != "" {
			endOfSpeech.enqueueCommand(command)
		}
	case internal_type.UserTextReceivedPacket:
		return endOfSpeech.handleUserTextPacket(ctx, packet)
	case internal_type.EndOfSpeechInterruptionPacket:
		endOfSpeech.mu.RLock()
		if packet.ContextID != "" && endOfSpeech.state.segment.ContextID != "" &&
			packet.ContextID != endOfSpeech.state.segment.ContextID {
			endOfSpeech.mu.RUnlock()
			return nil
		}
		if endOfSpeech.state.vadState != vadStateIdle || endOfSpeech.state.transcript == transcriptStateUserText {
			endOfSpeech.mu.RUnlock()
			return nil
		}
		command := workerCommand{
			ctx:        ctx,
			segment:    endOfSpeech.state.segment,
			confidence: endOfSpeech.state.confidence,
			timeout:    endOfSpeech.extendedTimeout,
		}
		command.segment.Text = command.segment.FinalText
		endOfSpeech.mu.RUnlock()
		if command.segment.Text != "" {
			endOfSpeech.enqueueCommand(command)
		}
	case internal_type.InterruptionDetectedPacket:
		if packet.Source != internal_type.InterruptionSourceVad {
			return nil
		}
		endOfSpeech.mu.Lock()
		if endOfSpeech.closed {
			endOfSpeech.mu.Unlock()
			return nil
		}
		switch packet.Event {
		case internal_type.InterruptionEventStart:
			if endOfSpeech.cancelPrediction != nil {
				endOfSpeech.cancelPrediction()
				endOfSpeech.cancelPrediction = nil
			}
			endOfSpeech.state.vadState = vadStateSpeaking
			endOfSpeech.state.turnState = turnStatePending
			endOfSpeech.state.confidence = 0
			endOfSpeech.state.transcriptDeadline = time.Time{}
			endOfSpeech.state.turnStopDeadline = time.Time{}
			endOfSpeech.state.silenceSamples = 0
			endOfSpeech.state.vadRevision++
			endOfSpeech.state.segment.Revision++
			if endOfSpeech.state.transcript == transcriptStateUserText {
				endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision}
				endOfSpeech.state.transcript = transcriptStateIdle
				endOfSpeech.state.started = false
			}
			if !endOfSpeech.hasSpeechStart {
				endOfSpeech.speechStartSample = endOfSpeech.audioNextSample
				if packet.StartAt > 0 {
					endOfSpeech.speechStartSample = uint64(packet.StartAt * float64(pipecatAudioSampleRate))
				}
				endOfSpeech.hasSpeechStart = true
			}
			endOfSpeech.audioGeneration++
			endOfSpeech.hasPredictedResult = false
		case internal_type.InterruptionEventEnd:
			if endOfSpeech.state.vadState == vadStateEnded {
				endOfSpeech.mu.Unlock()
				return nil
			}
			endOfSpeech.state.vadState = vadStateEnded
			endOfSpeech.state.turnState = turnStatePending
			endOfSpeech.state.silenceSamples = 0
			endOfSpeech.state.vadRevision++
			endOfSpeech.state.segment.Revision++
			endOfSpeech.state.turnStopDeadline = time.Now().Add(endOfSpeech.turnStopTimeout)
			if endOfSpeech.state.transcript == transcriptStateUserText {
				endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision}
				endOfSpeech.state.transcript = transcriptStateIdle
				endOfSpeech.state.started = false
			}
			// STT packets carry committed chunks, not Pipecat's explicit finalization acknowledgment.
			// The configured safety budget substitutes for STT P99 metadata absent from this contract.
			endOfSpeech.state.transcriptDeadline = time.Now().Add(endOfSpeech.fallbackTimeout)
			if packet.EndAt > 0 && packet.EndAt <= float64(endOfSpeech.audioNextSample)/pipecatAudioSampleRate {
				vadStopSamples, _ := utils.Uint64ToInt64(min(
					endOfSpeech.audioNextSample-uint64(packet.EndAt*pipecatAudioSampleRate), maxAudioDurationSamples,
				))
				endOfSpeech.state.transcriptDeadline = endOfSpeech.state.transcriptDeadline.Add(
					-time.Duration(vadStopSamples) * pipecatAudioSampleDuration,
				)
			}
			// Transcription can extend turn recovery without extending the native inference budget.
			predictionDeadline := endOfSpeech.state.turnStopDeadline
			if endOfSpeech.state.transcriptDeadline.After(predictionDeadline) {
				predictionDeadline = endOfSpeech.state.transcriptDeadline
			}
			predictionContext, cancelPrediction := context.WithDeadline(ctx, predictionDeadline)
			endOfSpeech.cancelPrediction = cancelPrediction
			// Snapshot stopped speech before later packets can change the audio window.
			audioStartSample := endOfSpeech.audioStartSample
			if endOfSpeech.hasSpeechStart {
				if endOfSpeech.speechStartSample > preSpeechAudioSamples {
					audioStartSample = endOfSpeech.speechStartSample - preSpeechAudioSamples
				} else {
					audioStartSample = 0
				}
				if audioStartSample < endOfSpeech.audioStartSample {
					audioStartSample = endOfSpeech.audioStartSample
				}
			}
			audioStartIndex := len(endOfSpeech.audioBuffer)
			audioStartOffset, conversionErr := utils.Uint64ToInt64(audioStartSample - endOfSpeech.audioStartSample)
			if conversionErr == nil && audioStartOffset < int64(len(endOfSpeech.audioBuffer)) {
				audioStartIndex, _ = utils.Int64ToInt(audioStartOffset)
			}
			audio := make([]float32, len(endOfSpeech.audioBuffer)-audioStartIndex)
			copy(audio, endOfSpeech.audioBuffer[audioStartIndex:])
			request := predictionRequest{
				ctx:             predictionContext,
				packetContext:   ctx,
				cancel:          cancelPrediction,
				contextID:       packet.ContextID,
				vadRevision:     endOfSpeech.state.vadRevision,
				audioGeneration: endOfSpeech.audioGeneration,
				audio:           audio,
			}
			command := workerCommand{
				ctx:      ctx,
				segment:  endOfSpeech.state.segment,
				deadline: endOfSpeech.state.transcriptDeadline,
			}
			command.segment.Text = command.segment.FinalText
			endOfSpeech.mu.Unlock()
			if command.segment.Text != "" {
				endOfSpeech.enqueueCommand(command)
			}
			endOfSpeech.mu.Lock()
			if endOfSpeech.closed || endOfSpeech.state.vadRevision != request.vadRevision {
				request.cancel()
				endOfSpeech.mu.Unlock()
				return nil
			}
			if endOfSpeech.predictionCh == nil {
				endOfSpeech.predictionCh = make(chan predictionRequest, 1)
				endOfSpeech.predictionDone = make(chan struct{})
				go endOfSpeech.predictionWorker()
			}
			// Keep only the latest pending stop while one native prediction is active.
			select {
			case pending := <-endOfSpeech.predictionCh:
				pending.cancel()
			default:
			}
			endOfSpeech.predictionCh <- request
			endOfSpeech.mu.Unlock()
			return nil
		}
		endOfSpeech.mu.Unlock()
	case internal_type.SpeechToTextPacket:
		return endOfSpeech.handleSpeechToTextPacket(ctx, packet)
	}
	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) handleUserTextPacket(ctx context.Context, packet internal_type.UserTextReceivedPacket) error {
	if packet.Text == "" {
		return nil
	}

	endOfSpeech.mu.Lock()
	if endOfSpeech.cancelPrediction != nil {
		endOfSpeech.cancelPrediction()
		endOfSpeech.cancelPrediction = nil
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
	}
	endOfSpeech.state.segment = segment
	endOfSpeech.state.confidence = 0
	endOfSpeech.state.transcript = transcriptStateUserText
	endOfSpeech.state.vadRevision++
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

func (endOfSpeech *pipecatEndOfSpeech) handleSpeechToTextPacket(ctx context.Context, packet internal_type.SpeechToTextPacket) error {
	if strings.TrimSpace(packet.Script) == "" {
		return nil
	}
	endOfSpeech.mu.Lock()
	if endOfSpeech.state.vadState == vadStateEnded {
		endOfSpeech.state.turnStopDeadline = time.Now().Add(endOfSpeech.turnStopTimeout)
	}
	if endOfSpeech.state.transcript == transcriptStateUserText {
		endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision}
		endOfSpeech.state.started = false
	}
	segment := speechSegment{
		Revision:  endOfSpeech.state.segment.Revision + 1,
		ContextID: packet.ContextId(),
		FinalText: endOfSpeech.state.segment.FinalText,
		Timestamp: time.Now(),
		Chunks:    append([]internal_type.SpeechToTextPacket(nil), endOfSpeech.state.segment.Chunks...),
	}
	segment.Chunks = append(segment.Chunks, packet)
	segment.Text = segment.FinalText
	if segment.Text != "" {
		segment.Text += packet.GetConcat()
	}
	segment.Text += packet.Script
	if packet.Interim {
		endOfSpeech.state.transcript = transcriptStateInterimPending
		if segment.FinalText != "" {
			segment.Timestamp = endOfSpeech.state.segment.Timestamp
			endOfSpeech.state.transcript = transcriptStateFinalizedWithPendingInterim
		}
	} else {
		segment.FinalText = segment.Text
		endOfSpeech.state.transcript = transcriptStateFinalized
	}
	endOfSpeech.state.segment = segment
	emitStarted := !endOfSpeech.state.started
	endOfSpeech.state.started = true
	command := workerCommand{
		ctx:        ctx,
		segment:    segment,
		confidence: endOfSpeech.state.confidence,
	}
	command.segment.Text = segment.FinalText
	shouldSchedule := false
	if segment.FinalText != "" {
		switch endOfSpeech.state.vadState {
		case vadStateIdle:
			if !packet.Interim || endOfSpeech.state.transcriptDeadline.IsZero() {
				endOfSpeech.state.transcriptDeadline = time.Now().Add(endOfSpeech.fallbackTimeout)
			}
			shouldSchedule = true
		case vadStateEnded:
			shouldSchedule = true
		}
	}
	command.deadline = endOfSpeech.state.transcriptDeadline
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
	if shouldSchedule {
		endOfSpeech.enqueueCommand(command)
	}
	return nil
}

func (endOfSpeech *pipecatEndOfSpeech) appendAudio(pcm16 []byte) {
	if len(pcm16) < 2 {
		return
	}

	pcmSampleCount := len(pcm16) / 2
	pcmSamples := make([]float32, pcmSampleCount)
	for sampleIndex := 0; sampleIndex < pcmSampleCount; sampleIndex++ {
		// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
		linearSample := int16(binary.LittleEndian.Uint16(pcm16[sampleIndex*2:]))
		pcmSamples[sampleIndex] = float32(linearSample) / 32768.0
	}

	endOfSpeech.mu.Lock()
	bufferEnd := endOfSpeech.audioStartSample + uint64(len(endOfSpeech.audioBuffer))
	if endOfSpeech.audioNextSample < bufferEnd {
		endOfSpeech.audioNextSample = bufferEnd
	}
	endOfSpeech.audioBuffer = append(endOfSpeech.audioBuffer, pcmSamples...)
	endOfSpeech.audioNextSample += uint64(len(pcmSamples))
	if len(endOfSpeech.audioBuffer) > maxAudioSamples {
		excess := len(endOfSpeech.audioBuffer) - maxAudioSamples
		endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[excess:]
		evictedSampleCount, _ := utils.IntToUint64(excess)
		endOfSpeech.audioStartSample += evictedSampleCount
	}
	endOfSpeech.audioGeneration++
	endOfSpeech.hasPredictedResult = false
	endOfSpeech.mu.Unlock()
}

func (endOfSpeech *pipecatEndOfSpeech) predictEOU(ctx context.Context, audio []float32, generation uint64) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	endOfSpeech.mu.RLock()
	if endOfSpeech.hasPredictedResult && endOfSpeech.predictedGeneration == generation {
		probability := endOfSpeech.predictedProbability
		endOfSpeech.mu.RUnlock()
		return probability, nil
	}
	endOfSpeech.mu.RUnlock()

	if len(audio) == 0 {
		endOfSpeech.debugf("pipecat_eos: inference skipped: empty audio buffer")
		return 0, nil
	}

	endOfSpeech.predictorMu.Lock()
	defer endOfSpeech.predictorMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if endOfSpeech.predictor == nil {
		return 0, errPipecatDetectorNil
	}

	probability, err := endOfSpeech.predictor.PredictContext(ctx, audio)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errPipecatDetectorRunInference, err)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
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

	return probability, nil
}

func (endOfSpeech *pipecatEndOfSpeech) predictionWorker() {
	defer close(endOfSpeech.predictionDone)
	for {
		select {
		case <-endOfSpeech.stopCh:
			return
		case request := <-endOfSpeech.predictionCh:
			probability, predictionErr := endOfSpeech.predictEOU(request.ctx, request.audio, request.audioGeneration)
			request.cancel()
			if predictionErr != nil && !errors.Is(predictionErr, context.Canceled) && !errors.Is(predictionErr, context.DeadlineExceeded) {
				_ = endOfSpeech.onPacket(request.packetContext, internal_type.ObservabilityLogRecordPacket{
					ContextID: request.contextID,
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:      observability.LevelError,
						Message:    predictionErr.Error(),
						OccurredAt: time.Now(),
						Attributes: observability.Attributes{
							"component": observability.ComponentEOS.String(),
							"provider":  endOfSpeech.Name(),
							"operation": "predict_end_of_turn",
						},
					},
				})
			}
			endOfSpeech.mu.Lock()
			if endOfSpeech.closed || endOfSpeech.state.vadRevision != request.vadRevision || endOfSpeech.state.vadState != vadStateEnded {
				endOfSpeech.mu.Unlock()
				continue
			}
			endOfSpeech.cancelPrediction = nil
			endOfSpeech.state.confidence = probability
			if endOfSpeech.state.turnState != turnStateComplete {
				if predictionErr == nil && probability > endOfSpeech.threshold {
					endOfSpeech.state.turnState = turnStateComplete
				} else {
					endOfSpeech.state.turnState = turnStateIncomplete
				}
			}
			if endOfSpeech.state.turnState == turnStateComplete {
				endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[:0]
				endOfSpeech.audioStartSample = endOfSpeech.audioNextSample
				endOfSpeech.hasSpeechStart = false
				endOfSpeech.audioGeneration++
				endOfSpeech.hasPredictedResult = false
			}
			command := workerCommand{
				ctx:        request.packetContext,
				segment:    endOfSpeech.state.segment,
				confidence: endOfSpeech.state.confidence,
				deadline:   endOfSpeech.state.transcriptDeadline,
			}
			command.segment.Text = command.segment.FinalText
			endOfSpeech.mu.Unlock()
			if command.segment.Text != "" {
				endOfSpeech.enqueueCommand(command)
			}
		}
	}
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
	var timer *time.Timer
	var timerCh <-chan time.Time
	var timerArmedAt time.Time
	var currentCommand workerCommand

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerCh = nil
		}
		timerArmedAt = time.Time{}
	}
	resetState := func() {
		if endOfSpeech.cancelPrediction != nil {
			endOfSpeech.cancelPrediction()
			endOfSpeech.cancelPrediction = nil
		}
		endOfSpeech.state.segment = speechSegment{Revision: endOfSpeech.state.segment.Revision + 1}
		endOfSpeech.state.confidence = 0
		endOfSpeech.state.started = false
		endOfSpeech.state.vadState = vadStateIdle
		endOfSpeech.state.transcript = transcriptStateIdle
		endOfSpeech.state.turnState = turnStatePending
		endOfSpeech.state.vadRevision++
		endOfSpeech.state.transcriptDeadline = time.Time{}
		endOfSpeech.state.turnStopDeadline = time.Time{}
		endOfSpeech.state.silenceSamples = 0
		endOfSpeech.audioBuffer = endOfSpeech.audioBuffer[:0]
		endOfSpeech.audioStartSample = endOfSpeech.audioNextSample
		endOfSpeech.hasSpeechStart = false
		endOfSpeech.speechStartSample = 0
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
			if command.fireImmediately {
				if command.segment.Revision == endOfSpeech.state.segment.Revision {
					stopTimer()
					resetState()
				}
				endOfSpeech.mu.Unlock()
				endOfSpeech.emitEndOfSpeech(command, time.Now())
				continue
			}
			if command.segment.Revision != endOfSpeech.state.segment.Revision {
				endOfSpeech.mu.Unlock()
				continue
			}
			// The inactivity watchdog only recovers turns that model or audio completion has not released.
			if endOfSpeech.state.vadState == vadStateEnded && endOfSpeech.state.turnState != turnStateComplete &&
				command.deadline.Before(endOfSpeech.state.turnStopDeadline) {
				command.deadline = endOfSpeech.state.turnStopDeadline
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
			if currentCommand.segment.Revision != endOfSpeech.state.segment.Revision ||
				endOfSpeech.state.vadState == vadStateSpeaking ||
				endOfSpeech.state.transcript == transcriptStateInterimPending {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			if endOfSpeech.state.vadState == vadStateEnded && endOfSpeech.state.turnState != turnStateComplete &&
				(endOfSpeech.state.turnStopDeadline.IsZero() ||
					time.Now().Before(endOfSpeech.state.turnStopDeadline)) {
				stopTimer()
				endOfSpeech.mu.Unlock()
				continue
			}
			turnStopDeadlineExpired := endOfSpeech.state.vadState == vadStateEnded && endOfSpeech.state.turnState != turnStateComplete
			command := currentCommand
			command.confidence = endOfSpeech.state.confidence
			armedAt := timerArmedAt
			stopTimer()
			resetState()
			endOfSpeech.mu.Unlock()
			if turnStopDeadlineExpired {
				_ = endOfSpeech.onPacket(command.ctx, internal_type.ObservabilityLogRecordPacket{
					ContextID: command.segment.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:      observability.LevelInfo,
						Message:    "Pipecat inactivity watchdog released committed text without a complete prediction",
						OccurredAt: time.Now(),
						Attributes: observability.Attributes{
							"component":  observability.ComponentEOS.String(),
							"provider":   endOfSpeech.Name(),
							"confidence": fmt.Sprintf("%.4f", command.confidence),
						},
					},
				})
			}
			endOfSpeech.emitEndOfSpeech(command, armedAt)
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
				OccurredAt: time.Now(),
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
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider": endOfSpeech.Name(),
				},
				Metrics: []*protos.Metric{
					{Name: observability.MetricEOSLatencyMs, Value: fmt.Sprintf("%d", waitToTriggerMs)},
					{Name: observability.MetricEOSTextToTriggerMs, Value: fmt.Sprintf("%d", textToTriggerMs)},
					{Name: observability.MetricEOSWordCount, Value: fmt.Sprintf("%d", wordCount)},
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
		endOfSpeech.closed = true
		if endOfSpeech.stopCh != nil {
			close(endOfSpeech.stopCh)
		}
		if endOfSpeech.cancelPrediction != nil {
			endOfSpeech.cancelPrediction()
			endOfSpeech.cancelPrediction = nil
		}
		endOfSpeech.state.vadRevision++
		endOfSpeech.state.segment.Revision++
		eosStartedAt := endOfSpeech.eosStartedAt
		endOfSpeech.eosStartedAt = time.Time{}
		predictionDone := endOfSpeech.predictionDone
		endOfSpeech.mu.Unlock()
		if predictionDone != nil {
			<-predictionDone
		}

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
		endOfSpeech.predictorMu.Lock()
		if predictor, ok := endOfSpeech.predictor.(interface{ Destroy() }); ok {
			predictor.Destroy()
		}
		endOfSpeech.predictor = nil
		endOfSpeech.predictorMu.Unlock()
	})

	return nil
}
