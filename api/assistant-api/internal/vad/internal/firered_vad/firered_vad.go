// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_firered_vad

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	vadName          = "firered_vad"
	envModelPathKey  = "FIRERED_VAD_MODEL_PATH"
	defaultModelFile = "models/fireredvad_stream_vad_with_cache.onnx"

	// Default configuration values aligned with Pipecat-style VAD options.
	defaultConfidence = 0.7
	defaultStartSecs  = 0.2
	defaultStopSecs   = 0.2
)

// -----------------------------------------------------------------------------
// FireRedVAD — Voice Activity Detection using FireRedVAD DFSMN model
// -----------------------------------------------------------------------------

// FireRedVAD implements the VoiceActivityDetectorExecutor interface using the FireRedVAD ONNX streaming
// model. It performs Kaldi-compatible fbank feature extraction, CMVN
// normalisation, ONNX inference, and postprocessing on incoming 16 kHz
// LINEAR16 mono audio.
type FireRedVAD struct {
	logger   commons.Logger
	onPacket func(ctx context.Context, pkt ...internal_type.Packet) error
	opts     utils.Option

	detector      *Detector
	fbank         *FbankExtractor
	postprocessor *Postprocessor

	// Audio sample buffer for frame extraction
	audioBuf []int16

	mu           sync.RWMutex
	isTerminated bool
	vadStartedAt time.Time
}

type options struct {
	ctx      context.Context
	logger   commons.Logger
	onPacket func(ctx context.Context, pkt ...internal_type.Packet) error
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

func WithOnPacket(onPacket func(ctx context.Context, pkt ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(opts utils.Option) Option {
	return func(options *options) {
		options.options = opts
	}
}

// New creates a new FireRedVAD instance.
// Input audio must be 16 kHz LINEAR16 mono — the platform's internal format.
func New(opts ...Option) (internal_type.VoiceActivityDetectorExecutor, error) {
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
		return nil, fmt.Errorf("%s: onPacket is required", vadName)
	}
	start := time.Now()

	ppCfg := DefaultPostprocessorConfig()
	if options.options != nil {
		if v, err := options.options.GetFloat64(internal_options.MicrophoneVADOptionConfidence); err == nil {
			ppCfg.SpeechThreshold = float32(v)
		}
		if v, err := options.options.GetFloat64(internal_options.MicrophoneVADOptionStartSecs); err == nil {
			if v < 0 {
				return nil, fmt.Errorf("invalid %s: should be a positive number", internal_options.MicrophoneVADOptionStartSecs)
			}
			ppCfg.MinSpeechFrame = vadDurationFrames(v)
		}
		if v, err := options.options.GetFloat64(internal_options.MicrophoneVADOptionStopSecs); err == nil {
			if v < 0 {
				return nil, fmt.Errorf("invalid %s: should be a positive number", internal_options.MicrophoneVADOptionStopSecs)
			}
			ppCfg.MinSilenceFrame = vadDurationFrames(v)
		}
	}
	if ppCfg.SpeechThreshold < 0 || ppCfg.SpeechThreshold > 1 {
		return nil, fmt.Errorf("invalid %s: should be in range [0, 1]", internal_options.MicrophoneVADOptionConfidence)
	}

	modelPath := resolveModelPath()
	detector, err := NewDetector(modelPath)
	if err != nil {
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: fmt.Sprintf("%s: error while initialization %s", vadName, err.Error()),
					Attributes: observability.Attributes{
						"component": observability.ComponentVAD.String(),
						"provider":  vadName,
						"options":   observability.AttributeValue(options.options),
					},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil, fmt.Errorf("firered_vad: failed to create detector: %w", err)
	}

	vad := &FireRedVAD{
		logger:        options.logger,
		onPacket:      options.onPacket,
		opts:          options.options,
		detector:      detector,
		fbank:         NewFbankExtractor(),
		postprocessor: NewPostprocessor(ppCfg),
		audioBuf:      make([]int16, 0, frameLenSample*2),
		isTerminated:  false,
		vadStartedAt:  time.Now(),
	}

	go func() {
		<-options.ctx.Done()
		_ = vad.Close(context.Background())
	}()

	if options.onPacket != nil {
		_ = options.onPacket(options.ctx,
			internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{"provider": vad.Name()},
					Metrics: []*protos.Metric{{
						Name:        observability.MetricVADInitLatencyMs,
						Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
						Description: "VAD initialization latency in milliseconds",
					}},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: fmt.Sprintf("%s: initialization completed", vad.Name()),
					Attributes: observability.Attributes{
						"component": observability.ComponentVAD.String(),
						"provider":  vad.Name(),
						"options":   observability.AttributeValue(vad.Options()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return vad, nil
}

// -----------------------------------------------------------------------------
// Public Interface Methods
// -----------------------------------------------------------------------------

func (v *FireRedVAD) Name() string {
	return vadName
}

func (v *FireRedVAD) Options() utils.Option {
	return v.opts
}

func (v *FireRedVAD) Arguments() (map[string]string, error) {
	return nil, nil
}

// Execute analyses an audio packet for voice activity.
// The packet must contain 16 kHz LINEAR16 mono audio.
func (v *FireRedVAD) Execute(ctx context.Context, pkt internal_type.UserAudioReceivedPacket) error {
	if !v.isActive() {
		return nil
	}

	// Convert LINEAR16 bytes to int16 samples
	samples := internal_audio.Linear16ToInt16(pkt.Audio)

	// Append to buffer
	v.mu.Lock()
	v.audioBuf = append(v.audioBuf, samples...)

	// Process complete frames (400 samples each, 160-sample shift)
	var speechStartAt, speechEndAt float64
	hasSpeechStart := false
	hasSpeechEnd := false

	for len(v.audioBuf) >= frameLenSample {
		frame := v.audioBuf[:frameLenSample]

		// Extract fbank features for this frame
		var feat [featDim]float32
		v.fbank.Extract(frame, feat[:])

		// Apply CMVN normalisation
		applyCMVN(feat[:])

		// Run ONNX inference
		if v.isTerminated || v.detector == nil {
			v.mu.Unlock()
			return nil
		}
		prob, err := v.detector.Infer(feat[:])
		if err != nil {
			v.mu.Unlock()
			if v.onPacket != nil {
				_ = v.onPacket(ctx,
					internal_type.ObservabilityEventRecordPacket{
						ContextID: pkt.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentVAD,
							Event:     observability.VADError,
							Attributes: observability.Attributes{
								"provider":   vadName,
								"context_id": pkt.ContextID,
								"error":      "firered_vad inference failed",
							},
							OccurredAt: time.Now(),
						},
					},
					internal_type.ObservabilityLogRecordPacket{
						ContextID: pkt.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordLog{
							Level:   observability.LevelError,
							Message: "firered_vad: inference failed",
							Attributes: observability.Attributes{
								"component":   observability.ComponentVAD.String(),
								"operation":   "infer",
								"provider":    vadName,
								"context_id":  pkt.ContextID,
								"audio_bytes": fmt.Sprintf("%d", len(pkt.Audio)),
								"error":       err.Error(),
								"error_type":  fmt.Sprintf("%T", err),
							},
							OccurredAt: time.Now(),
						},
					},
				)
			}
			return fmt.Errorf("firered_vad: inference failed: %w", err)
		}

		// Postprocess
		result := v.postprocessor.ProcessFrame(prob)

		if result.IsSpeechStart {
			startAt := float64(result.SpeechStartFrame-1) / float64(framesPerSecond)
			if startAt < 0 {
				startAt = 0
			}
			if !hasSpeechStart || startAt < speechStartAt {
				speechStartAt = startAt
			}
			hasSpeechStart = true
		}
		if result.IsSpeechEnd {
			endAt := float64(result.SpeechEndFrame-1) / float64(framesPerSecond)
			if endAt < 0 {
				endAt = 0
			}
			if !hasSpeechEnd || endAt > speechEndAt {
				speechEndAt = endAt
			}
			hasSpeechEnd = true
		}

		// Shift by frameShiftSamp (160 samples)
		v.audioBuf = v.audioBuf[frameShiftSamp:]
	}
	v.mu.Unlock()

	// Emit explicit interruption lifecycle events from VAD transitions.
	if hasSpeechStart {
		v.onPacket(ctx,
			internal_type.InterruptionDetectedPacket{
				ContextID: pkt.ContextID,
				Source:    internal_type.InterruptionSourceVad,
				Event:     internal_type.InterruptionEventStart,
				StartAt:   speechStartAt,
				EndAt:     speechEndAt,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: pkt.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentVAD,
					Event:     observability.VADSpeechStarted,
					Attributes: observability.Attributes{
						"provider": vadName,
						"event":    string(internal_type.InterruptionEventStart),
						"start_at": fmt.Sprintf("%f", speechStartAt),
						"end_at":   fmt.Sprintf("%f", speechEndAt),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}
	if hasSpeechEnd {
		v.onPacket(ctx,
			internal_type.InterruptionDetectedPacket{
				ContextID: pkt.ContextID,
				Source:    internal_type.InterruptionSourceVad,
				Event:     internal_type.InterruptionEventEnd,
				StartAt:   speechStartAt,
				EndAt:     speechEndAt,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: pkt.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentVAD,
					Event:     observability.VADSpeechEnded,
					Attributes: observability.Attributes{
						"provider": vadName,
						"event":    string(internal_type.InterruptionEventEnd),
						"start_at": fmt.Sprintf("%f", speechStartAt),
						"end_at":   fmt.Sprintf("%f", speechEndAt),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return nil
}

// Close terminates the VAD and releases all resources.
func (v *FireRedVAD) Close(ctx context.Context) error {
	v.mu.Lock()
	if v.isTerminated {
		v.mu.Unlock()
		return nil
	}
	v.isTerminated = true
	vadStartedAt := v.vadStartedAt
	v.vadStartedAt = time.Time{}

	if v.detector != nil {
		v.detector.Destroy()
		v.detector = nil
	}
	v.mu.Unlock()

	if v.onPacket != nil {
		packets := []internal_type.Packet{}
		if !vadStartedAt.IsZero() {
			duration := time.Since(vadStartedAt)
			packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewVADDurationUsageRecord(v.Name(), duration, observability.Attributes{}),
			})
		}
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentVAD,
				Event:     observability.VADClosed,
				Attributes: observability.Attributes{
					"provider": v.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
		_ = v.onPacket(ctx, packets...)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Private Methods
// -----------------------------------------------------------------------------

func resolveModelPath() string {
	if envPath := os.Getenv(envModelPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultModelFile)
}

func vadDurationFrames(durationSecs float64) int {
	frameSecs := float64(frameShiftMs) / 1000
	return int(math.RoundToEven(durationSecs / frameSecs))
}

func (v *FireRedVAD) isActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return !v.isTerminated && v.detector != nil
}
