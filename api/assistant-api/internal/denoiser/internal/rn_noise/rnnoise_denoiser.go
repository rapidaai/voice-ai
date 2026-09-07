// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser_rnnoise

import (
	"context"
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	resampler_linear "github.com/rapidaai/api/assistant-api/internal/audio/resampler/linear"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const rnNoiseDenoiserName = "rn_noise"

type rnnoiseDenoiser struct {
	rnNoise        *RNNoise
	logger         commons.Logger
	opts           utils.Option
	denoiserConfig *protos.AudioConfig
	// default rapida input config
	inputConfig      *protos.AudioConfig
	audioResampler   internal_type.AudioResampler
	audioConverter   internal_type.AudioConverter
	onPacket         func(context.Context, ...internal_type.Packet) error
	denoiseStartedAt time.Time
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

func New(opts ...Option) (internal_type.VoiceDenoiserExecutor, error) {
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
		return nil, fmt.Errorf("%s: onPacket is required", rnNoiseDenoiserName)
	}
	start := time.Now()
	rn, err := NewRNNoise()
	if err != nil {
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: fmt.Sprintf("%s: error while initialization %s", rnNoiseDenoiserName, err.Error()),
					Attributes: observability.Attributes{
						"component":  observability.ComponentDenoise.String(),
						"provider":   rnNoiseDenoiserName,
						"options":    observability.AttributeValue(options.options),
						"error":      err.Error(),
						"error_type": fmt.Sprintf("%T", err),
					},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil, err
	}

	d := &rnnoiseDenoiser{
		audioResampler: resampler_soxr.NewChunk(
			resampler_soxr.WithLogger(options.logger),
			resampler_soxr.WithQuickQuality(),
		),
		audioConverter: resampler_linear.NewConverter(
			resampler_linear.WithLogger(options.logger),
		),
		rnNoise: rn,
		denoiserConfig: &protos.AudioConfig{
			SampleRate:  48000,
			AudioFormat: protos.AudioConfig_LINEAR16,
			Channels:    1,
		},
		inputConfig:      internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		logger:           options.logger,
		opts:             options.options,
		onPacket:         options.onPacket,
		denoiseStartedAt: time.Now(),
	}
	_ = options.onPacket(options.ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": d.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricDenoiseInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "Denoise initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: fmt.Sprintf("%s: initialization completed", d.Name()),
				Attributes: observability.Attributes{
					"component": observability.ComponentDenoise.String(),
					"provider":  d.Name(),
					"options":   observability.AttributeValue(options.options),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return d, nil
}

func (rnd *rnnoiseDenoiser) Name() string {
	return rnNoiseDenoiserName
}

func (rnd *rnnoiseDenoiser) Options() utils.Option {
	return rnd.opts
}

func (rnd *rnnoiseDenoiser) Arguments() (map[string]string, error) {
	return nil, nil
}

// Denoise processes the audio in pkt and pushes a DenoisedAudioPacket via
// onPacket instead of returning bytes to the caller. On error it falls back
// to the original audio and still emits the packet with NoiseReduced=false.
func (rnd *rnnoiseDenoiser) Execute(ctx context.Context, pkt internal_type.DenoiseAudioPacket) error {
	input := pkt.Audio
	if rnd.inputConfig == nil || rnd.denoiserConfig == nil {
		_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
			ContextID: pkt.ContextID,
			Audio:     input,
		}, internal_type.ObservabilityEventRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentDenoise,
				Event:      observability.DenoiseError,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   rnd.Name(),
					"context_id": pkt.ContextID,
					"error":      "audio config is not initialized",
				},
			},
		}, internal_type.ObservabilityLogRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "denoise failed",
				Attributes: observability.Attributes{
					"component":   observability.ComponentDenoise.String(),
					"operation":   "validate_config",
					"provider":    rnd.Name(),
					"context_id":  pkt.ContextID,
					"audio_bytes": fmt.Sprintf("%d", len(input)),
					"error":       "audio config is not initialized",
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("audio config is not initialized")
	}

	resampledInput, err := rnd.audioResampler.Resample(input, rnd.inputConfig, rnd.denoiserConfig)
	if err != nil {
		_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
			ContextID: pkt.ContextID,
			Audio:     input,
		}, internal_type.ObservabilityEventRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentDenoise,
				Event:      observability.DenoiseError,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   rnd.Name(),
					"context_id": pkt.ContextID,
					"error":      "failed to resample audio",
				},
			},
		}, internal_type.ObservabilityLogRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "denoise failed",
				Attributes: observability.Attributes{
					"component":   observability.ComponentDenoise.String(),
					"operation":   "resample_input",
					"provider":    rnd.Name(),
					"context_id":  pkt.ContextID,
					"audio_bytes": fmt.Sprintf("%d", len(input)),
					"error":       err.Error(),
					"error_type":  fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to resample audio: %w", err)
	}

	floatSamples, err := rnd.audioConverter.ConvertToFloat32Samples(resampledInput, rnd.denoiserConfig)
	if err != nil {

		_ = rnd.onPacket(ctx,
			internal_type.DenoisedAudioPacket{
				ContextID: pkt.ContextID,
				Audio:     input,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: pkt.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentDenoise,
					Event:      observability.DenoiseError,
					OccurredAt: time.Now(),
					Attributes: observability.Attributes{
						"provider":   rnd.Name(),
						"context_id": pkt.ContextID,
						"error":      "failed to convert audio to float32 samples",
					},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: pkt.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "denoise failed",
					Attributes: observability.Attributes{
						"component":   observability.ComponentDenoise.String(),
						"operation":   "convert_to_float32",
						"provider":    rnd.Name(),
						"context_id":  pkt.ContextID,
						"audio_bytes": fmt.Sprintf("%d", len(resampledInput)),
						"error":       err.Error(),
						"error_type":  fmt.Sprintf("%T", err),
					},
					OccurredAt: time.Now(),
				},
			})
		return fmt.Errorf("failed to convert audio to float32 samples: %w", err)
	}

	confidence, cleanedSamples, err := rnd.rnNoise.ProcessAudio(floatSamples)
	if err != nil {
		_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
			ContextID: pkt.ContextID,
			Audio:     input,
		}, internal_type.ObservabilityEventRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentDenoise,
				Event:      observability.DenoiseError,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   rnd.Name(),
					"context_id": pkt.ContextID,
					"error":      "failed to process audio",
				},
			},
		}, internal_type.ObservabilityLogRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "denoise failed",
				Attributes: observability.Attributes{
					"component":    observability.ComponentDenoise.String(),
					"operation":    "process_audio",
					"provider":     rnd.Name(),
					"context_id":   pkt.ContextID,
					"sample_count": fmt.Sprintf("%d", len(floatSamples)),
					"error":        err.Error(),
					"error_type":   fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to process audio: %w", err)
	}

	denoisedBytes, err := rnd.audioConverter.ConvertToByteSamples(cleanedSamples, rnd.denoiserConfig)
	if err != nil {
		_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
			ContextID: pkt.ContextID,
			Audio:     input,
		}, internal_type.ObservabilityEventRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentDenoise,
				Event:      observability.DenoiseError,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   rnd.Name(),
					"context_id": pkt.ContextID,
					"error":      "failed to convert audio to byte samples",
				},
			},
		}, internal_type.ObservabilityLogRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "denoise failed",
				Attributes: observability.Attributes{
					"component":    observability.ComponentDenoise.String(),
					"operation":    "convert_to_bytes",
					"provider":     rnd.Name(),
					"context_id":   pkt.ContextID,
					"sample_count": fmt.Sprintf("%d", len(cleanedSamples)),
					"confidence":   fmt.Sprintf("%.4f", confidence),
					"error":        err.Error(),
					"error_type":   fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to convert audio to byte samples: %w", err)
	}

	restoredInputRate, err := rnd.audioResampler.Resample(denoisedBytes, rnd.denoiserConfig, rnd.inputConfig)
	if err != nil {
		_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
			ContextID: pkt.ContextID,
			Audio:     input,
		}, internal_type.ObservabilityEventRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentDenoise,
				Event:      observability.DenoiseError,
				OccurredAt: time.Now(),
				Attributes: observability.Attributes{
					"provider":   rnd.Name(),
					"context_id": pkt.ContextID,
					"error":      "failed to resample denoised audio",
				},
			},
		}, internal_type.ObservabilityLogRecordPacket{
			ContextID: pkt.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "denoise failed",
				Attributes: observability.Attributes{
					"component":   observability.ComponentDenoise.String(),
					"operation":   "resample_output",
					"provider":    rnd.Name(),
					"context_id":  pkt.ContextID,
					"audio_bytes": fmt.Sprintf("%d", len(denoisedBytes)),
					"confidence":  fmt.Sprintf("%.4f", confidence),
					"error":       err.Error(),
					"error_type":  fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to resample denoised audio: %w", err)
	}

	_ = rnd.onPacket(ctx, internal_type.DenoisedAudioPacket{
		ContextID:  pkt.ContextID,
		Audio:      restoredInputRate,
		Confidence: confidence,
	})
	return nil
}

// Close releases resources
func (d *rnnoiseDenoiser) Close(ctx context.Context) error {
	denoiseStartedAt := d.denoiseStartedAt
	d.denoiseStartedAt = time.Time{}
	if !denoiseStartedAt.IsZero() {
		_ = d.onPacket(ctx, internal_type.ObservabilityUsageRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewDenoiseDurationUsageRecord(d.Name(), time.Since(denoiseStartedAt), observability.Attributes{}),
		})
	}
	_ = d.onPacket(ctx, internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentDenoise,
			Event:     observability.DenoiseClosed,
			Attributes: observability.Attributes{
				"provider": d.Name(),
			},
			OccurredAt: time.Now(),
		},
	})

	if d.rnNoise != nil {
		d.rnNoise.Close()
	}
	return nil
}
