// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silero_vad

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	// vadName is the identifier for this VAD implementation
	vadName = "silero_vad"

	// Default configuration values — aligned with FireRedVAD defaults
	// (20 frames × 10 ms = 200 ms silence, 8 frames × 10 ms = 80 ms pad)
	defaultThreshold            = 0.5
	defaultMinSilenceDurationMs = 200
	defaultSpeechPadMs          = 80

	// Environment variable for model path
	envModelPathKey = "SILERO_MODEL_PATH"

	// Default model filename
	defaultModelFile = "models/silero_vad_20251001.onnx"
)

// -----------------------------------------------------------------------------
// SileroVAD - Voice Activity Detection using Silero
// -----------------------------------------------------------------------------

// SileroVAD implements the VoiceActivityDetectorExecutor interface using the Silero ONNX model
// with native ONNX Runtime inference. It provides thread-safe voice
// activity detection with automatic cleanup on context cancellation.
//
// Input audio is expected to be 16 kHz LINEAR16 mono (the platform's
// internal audio format). No resampling is performed.
type SileroVAD struct {
	// Core dependencies
	logger   commons.Logger
	onPacket func(ctx context.Context, pkt ...internal_type.Packet) error
	opts     utils.Option

	// Silero detector (CGO-backed, requires careful lifecycle management)
	detector *Detector
	// Shared audio converter for LINEAR16 -> float32 conversion
	converter internal_type.AudioConverter

	// Thread-safety for CGO resource protection
	mu           sync.RWMutex
	isTerminated bool
	vadStartedAt time.Time
}

// -----------------------------------------------------------------------------
// Constructor
// -----------------------------------------------------------------------------

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

// New creates a new SileroVAD instance.
// Input audio must be 16 kHz LINEAR16 mono — the platform's internal format.
// The VAD will automatically close when the provided context is cancelled,
// ensuring safe cleanup of CGO resources.
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

	// Initialize detector
	detector, err := createDetector(options.options)
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
		return nil, fmt.Errorf("failed to create silero detector: %w", err)
	}
	converter, err := internal_audio_resampler.GetConverter(options.logger)
	if err != nil {
		detector.Destroy()
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
		return nil, fmt.Errorf("failed to create audio converter: %w", err)
	}

	svad := &SileroVAD{
		logger:       options.logger,
		onPacket:     options.onPacket,
		opts:         options.options,
		detector:     detector,
		converter:    converter,
		isTerminated: false,
		vadStartedAt: time.Now(),
	}

	go func() {
		<-options.ctx.Done()
		_ = svad.Close(context.Background())
	}()

	if options.onPacket != nil {
		_ = options.onPacket(options.ctx,
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricVADInitLatencyMs(time.Since(start), observability.Attributes{"provider": svad.Name()}),
			},
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: fmt.Sprintf("%s: initialization completed", svad.Name()),
					Attributes: observability.Attributes{
						"component": observability.ComponentVAD.String(),
						"provider":  svad.Name(),
						"options":   observability.AttributeValue(svad.Options()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return svad, nil
}

// -----------------------------------------------------------------------------
// Public Interface Methods
// -----------------------------------------------------------------------------

// Name returns the identifier for this VAD implementation.
func (s *SileroVAD) Name() string {
	return vadName
}

func (s *SileroVAD) Options() utils.Option {
	return s.opts
}

func (s *SileroVAD) Arguments() (map[string]string, error) {
	return nil, nil
}

// Execute analyzes an audio packet for voice activity.
// The packet must contain 16 kHz LINEAR16 mono audio.
// Returns immediately if the VAD has been terminated.
// Thread-safe for concurrent calls.
func (s *SileroVAD) Execute(ctx context.Context, pkt internal_type.UserAudioReceivedPacket) error {
	// Early termination check
	if !s.isActive() {
		return nil
	}

	// Convert LINEAR16 bytes to float32 samples via shared audio converter.
	samples, err := s.converter.ConvertToFloat32Samples(pkt.Audio, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG)
	if err != nil {
		if s.onPacket != nil {
			_ = s.onPacket(ctx,
				internal_type.ObservabilityEventRecordPacket{
					ContextID: pkt.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentVAD,
						Event:     observability.VADError,
						Attributes: observability.Attributes{
							"provider":   vadName,
							"context_id": pkt.ContextID,
							"error":      "failed to convert audio to float32",
						},
						OccurredAt: time.Now(),
					},
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: pkt.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "silero_vad: audio conversion failed",
						Attributes: observability.Attributes{
							"component":   observability.ComponentVAD.String(),
							"operation":   "convert_audio",
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
		return fmt.Errorf("failed to convert audio to float32: %w", err)
	}

	// Perform detection with CGO safety
	segments, _, err := s.detectSafely(samples)
	if err != nil {
		if s.onPacket != nil {
			_ = s.onPacket(ctx,
				internal_type.ObservabilityEventRecordPacket{
					ContextID: pkt.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentVAD,
						Event:     observability.VADError,
						Attributes: observability.Attributes{
							"provider":   vadName,
							"context_id": pkt.ContextID,
							"error":      "detection failed",
						},
						OccurredAt: time.Now(),
					},
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: pkt.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "silero_vad: detection failed",
						Attributes: observability.Attributes{
							"component":    observability.ComponentVAD.String(),
							"operation":    "detect",
							"provider":     vadName,
							"context_id":   pkt.ContextID,
							"sample_count": fmt.Sprintf("%d", len(samples)),
							"error":        err.Error(),
							"error_type":   fmt.Sprintf("%T", err),
						},
						OccurredAt: time.Now(),
					},
				},
			)
		}
		return err
	}

	hasSpeechStart := false
	hasSpeechEnd := false
	var speechStartAt, speechEndAt float64
	for _, seg := range segments {
		if seg.SpeechStartAt >= 0 && (!hasSpeechStart || seg.SpeechStartAt < speechStartAt) {
			speechStartAt = seg.SpeechStartAt
			hasSpeechStart = true
		}
		if seg.SpeechEndAt >= 0 && (!hasSpeechEnd || seg.SpeechEndAt > speechEndAt) {
			speechEndAt = seg.SpeechEndAt
			hasSpeechEnd = true
		}
	}

	// Emit explicit interruption lifecycle events from VAD transitions.
	if hasSpeechStart {
		s.notifyInterruption(ctx, pkt.ContextID, internal_type.InterruptionEventStart, speechStartAt, len(segments))
	}
	if hasSpeechEnd {
		s.notifyInterruption(ctx, pkt.ContextID, internal_type.InterruptionEventEnd, speechEndAt, len(segments))
	}

	return nil
}

// Close terminates the VAD and releases all CGO resources.
// Safe to call multiple times; subsequent calls are no-ops.
// Thread-safe.
func (s *SileroVAD) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.isTerminated {
		s.mu.Unlock()
		return nil
	}
	s.isTerminated = true
	vadStartedAt := s.vadStartedAt
	s.vadStartedAt = time.Time{}

	if s.detector != nil {
		s.detector.Destroy()
		s.detector = nil
	}
	s.mu.Unlock()

	if s.onPacket != nil {
		packets := []internal_type.Packet{}
		if !vadStartedAt.IsZero() {
			duration := time.Since(vadStartedAt)
			packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewVADDurationUsageRecord(s.Name(), duration, observability.Attributes{}),
			})
		}
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentVAD,
				Event:     observability.VADClosed,
				Attributes: observability.Attributes{
					"provider": s.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
		_ = s.onPacket(ctx, packets...)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Private Helper Methods - Initialization
// -----------------------------------------------------------------------------

// createDetector initializes the Silero speech detector with configuration.
func createDetector(options utils.Option) (*Detector, error) {
	modelPath := resolveModelPath()
	threshold := resolveThreshold(options)

	config := DetectorConfig{
		ModelPath:            modelPath,
		SampleRate:           16000, // Silero requires 16kHz
		Threshold:            float32(threshold),
		MinSilenceDurationMs: resolveMinSilenceDurationMs(options),
		SpeechPadMs:          resolveSpeechPadMs(options),
	}
	return NewDetector(config)
}

// resolveModelPath determines the ONNX model file path.
func resolveModelPath() string {
	if envPath := os.Getenv(envModelPathKey); envPath != "" {
		return envPath
	}

	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultModelFile)
}

// resolveThreshold extracts threshold from options or returns default.
func resolveThreshold(options utils.Option) float64 {
	if options == nil {
		return defaultThreshold
	}

	if threshold, err := options.GetFloat64("microphone.vad.threshold"); err == nil {
		return threshold
	}

	return defaultThreshold
}

// resolveMinSilenceDurationMs extracts min silence duration from options.
// The option key uses frame count (consistent with FireRedVAD config);
// each frame is 10 ms, so we multiply by 10 to get milliseconds.
func resolveMinSilenceDurationMs(options utils.Option) int {
	if options == nil {
		return defaultMinSilenceDurationMs
	}
	if v, err := options.GetFloat64("microphone.vad.min_silence_frame"); err == nil {
		return int(v) * 10
	}
	return defaultMinSilenceDurationMs
}

// resolveSpeechPadMs extracts speech pad duration from options.
// The option key uses frame count (consistent with FireRedVAD config);
// each frame is 10 ms, so we multiply by 10 to get milliseconds.
func resolveSpeechPadMs(options utils.Option) int {
	if options == nil {
		return defaultSpeechPadMs
	}
	if v, err := options.GetFloat64("microphone.vad.min_speech_frame"); err == nil {
		return int(v) * 10
	}
	return defaultSpeechPadMs
}

// isActive checks if the VAD is still operational.
// Thread-safe.
func (s *SileroVAD) isActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.isTerminated && s.detector != nil
}

// -----------------------------------------------------------------------------
// Private Helper Methods - Audio Processing
// -----------------------------------------------------------------------------

// detectSafely performs voice activity detection with CGO resource protection.
// Holds the write lock for the duration of the CGO call: Detector
// mutates internal ONNX state and is not safe for concurrent use.
// Returns segments and whether the detector is currently in a triggered (speech active) state.
func (s *SileroVAD) detectSafely(samples []float32) ([]Segment, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isTerminated || s.detector == nil {
		return nil, false, nil
	}

	segments, err := s.detector.Detect(samples)
	if err != nil {
		return nil, false, fmt.Errorf("detection failed: %w", err)
	}

	return segments, s.detector.triggered, nil
}

// notifyInterruption emits a VAD interruption lifecycle event packet.
func (s *SileroVAD) notifyInterruption(ctx context.Context, contextID string, event internal_type.InterruptionEvent, at float64, segmentCount int) {
	if s.onPacket != nil {
		eventName := observability.VADSpeechStarted
		if event == internal_type.InterruptionEventEnd {
			eventName = observability.VADSpeechEnded
		}
		_ = s.onPacket(ctx,
			internal_type.InterruptionDetectedPacket{
				ContextID: contextID,
				Source:    internal_type.InterruptionSourceVad,
				Event:     event,
				StartAt:   at,
				EndAt:     at,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentVAD,
					Event:     eventName,
					Attributes: observability.Attributes{
						"provider":      vadName,
						"event":         string(event),
						"start_at":      fmt.Sprintf("%f", at),
						"end_at":        fmt.Sprintf("%f", at),
						"segment_count": fmt.Sprintf("%d", segmentCount),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}
}
