// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import (
	"encoding/binary"
	"fmt"
	"sync"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	resampling "github.com/tphakala/go-audio-resampler"
	"github.com/zaf/g711"
)

// cachedEngine holds a stateful polyphase FIR resampler for one rate-pair.
// The mutex serialises access so the filter's internal state is never
// corrupted by concurrent callers and carries over between consecutive
// audio chunks, eliminating per-chunk startup transients.
type cachedEngine struct {
	mu sync.Mutex
	rs resampling.Resampler
}

// libsoxrResampler provides high-quality polyphase FIR audio resampling.
// One engine is created per (srcRate/dstRate) pair and reused across calls
// so the filter state is continuous for streaming audio.
type libsoxrResampler struct {
	logger  commons.Logger
	quality resampling.QualityPreset
	engines sync.Map // key "srcRate/dstRate" → *cachedEngine
}

type options struct {
	logger  commons.Logger
	quality resampling.QualityPreset
}

type Option func(*options)

// WithLogger sets the resampler logger.
func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithHighQuality selects the higher-quality streaming filter.
func WithHighQuality() Option {
	return func(options *options) {
		options.quality = resampling.QualityHigh
	}
}

// WithQuickQuality selects the low-latency streaming filter.
func WithQuickQuality() Option {
	return func(options *options) {
		options.quality = resampling.QualityQuick
	}
}

// New creates a streaming audio resampler with high-quality output by default.
func New(options ...Option) internal_type.AudioResampler {
	config := newOptions(options)
	return &libsoxrResampler{logger: config.logger, quality: config.quality}
}

func newOptions(opts []Option) options {
	config := options{quality: resampling.QualityHigh}
	for _, option := range opts {
		if option != nil {
			option(&config)
		}
	}
	return config
}

// Resample converts audio data using high-quality resampling.
// The polyphase FIR engine is cached per rate-pair and reused across calls
// so filter state is continuous without per-chunk startup transients.
func (r *libsoxrResampler) Resample(
	data []byte,
	source, target *protos.AudioConfig,
) ([]byte, error) {

	if source == nil || target == nil {
		return nil, fmt.Errorf("source and target configs are required")
	}

	if len(data) == 0 {
		return []byte{}, nil
	}

	// No-op when all parameters already match.
	if source.SampleRate == target.SampleRate &&
		source.Channels == target.Channels &&
		source.AudioFormat == target.AudioFormat {
		return data, nil
	}

	// Convert input to LINEAR16 so the FIR engine always works in PCM.
	pcm := data
	if source.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = r.convertToLinear16(data, source)
		if err != nil {
			return nil, err
		}
	}

	// Resample the sample rate while still mono / same channel count.
	if source.SampleRate != target.SampleRate {
		var err error
		pcm, err = r.resamplePCM16(pcm, source.SampleRate, target.SampleRate)
		if err != nil {
			return nil, err
		}
	}

	// Channel conversion after rate change (cheaper to convert fewer samples).
	if source.Channels != target.Channels {
		pcm = r.convertChannels(pcm, source.Channels, target.Channels)
	}

	// Convert to the target encoding if it differs from LINEAR16.
	if target.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = r.convertFromLinear16(pcm, target)
		if err != nil {
			return nil, err
		}
	}

	return pcm, nil
}

// =======================
// Resampling
// =======================

// resamplePCM16 resamples mono LINEAR16 PCM using a cached polyphase FIR engine.
// No Flush() is called between chunks: the filter state carries over so
// consecutive chunks from a streaming TTS source produce seamless output.
func (r *libsoxrResampler) resamplePCM16(pcm []byte, srcRate, dstRate uint32) ([]byte, error) {
	eng, err := r.getOrCreateEngine(srcRate, dstRate)
	if err != nil {
		return nil, err
	}

	eng.mu.Lock()
	defer eng.mu.Unlock()

	out, err := eng.rs.Process(pcm16ToFloat64(pcm))
	if err != nil {
		return nil, fmt.Errorf("resample failed: %w", err)
	}

	return float64ToPCM16(out), nil
}

// getOrCreateEngine returns the cached engine for (srcRate, dstRate),
// creating it on first access. LoadOrStore ensures only one engine is
// created even under concurrent first-time calls for the same key.
func (r *libsoxrResampler) getOrCreateEngine(srcRate, dstRate uint32) (*cachedEngine, error) {
	key := fmt.Sprintf("%d/%d", srcRate, dstRate)

	if v, ok := r.engines.Load(key); ok {
		return v.(*cachedEngine), nil
	}

	rs, err := resampling.New(&resampling.Config{
		InputRate:  float64(srcRate),
		OutputRate: float64(dstRate),
		Channels:   1, // Process() is mono-only; multi-channel needs ProcessMulti
		EnableSIMD: true,
		Quality:    resampling.QualitySpec{Preset: r.quality},
	})
	if err != nil {
		return nil, fmt.Errorf("resampler init failed: %w", err)
	}

	ce := &cachedEngine{rs: rs}
	actual, _ := r.engines.LoadOrStore(key, ce)
	return actual.(*cachedEngine), nil
}

// =======================
// PCM16 ↔ float64
// =======================

func pcm16ToFloat64(data []byte) []float64 {
	n := len(data) &^ 1 // round down to even byte boundary
	out := make([]float64, n/2)
	for i := 0; i < n; i += 2 {
		s := int16(binary.LittleEndian.Uint16(data[i : i+2]))
		out[i/2] = float64(s) / 32768.0
	}
	return out
}

func float64ToPCM16(data []float64) []byte {
	out := make([]byte, len(data)*2)
	for i, v := range data {
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		s := int16(v * 32767.0)
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s))
	}
	return out
}

// =======================
// Format Conversion
// =======================

func (r *libsoxrResampler) convertToLinear16(data []byte, cfg *protos.AudioConfig) ([]byte, error) {
	switch cfg.AudioFormat {
	case protos.AudioConfig_LINEAR16:
		return data, nil
	case protos.AudioConfig_MuLaw8:
		return g711.DecodeUlaw(data), nil
	default:
		return nil, fmt.Errorf("unsupported input format: %v", cfg.AudioFormat)
	}
}

func (r *libsoxrResampler) convertFromLinear16(data []byte, cfg *protos.AudioConfig) ([]byte, error) {
	switch cfg.AudioFormat {
	case protos.AudioConfig_LINEAR16:
		return data, nil
	case protos.AudioConfig_MuLaw8:
		return g711.EncodeUlaw(data), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %v", cfg.AudioFormat)
	}
}

// =======================
// Channel Conversion
// =======================

func (r *libsoxrResampler) convertChannels(data []byte, src, dst uint32) []byte {
	if src == dst {
		return data
	}

	// Mono → Stereo: duplicate each sample into both channels.
	if src == 1 && dst == 2 {
		out := make([]byte, len(data)*2)
		for i := 0; i < len(data); i += 2 {
			copy(out[i*2:], data[i:i+2])
			copy(out[i*2+2:], data[i:i+2])
		}
		return out
	}

	// Stereo → Mono: average the two channels.
	if src == 2 && dst == 1 {
		out := make([]byte, len(data)/2)
		for i := 0; i < len(data); i += 4 {
			l := int16(binary.LittleEndian.Uint16(data[i:]))
			r := int16(binary.LittleEndian.Uint16(data[i+2:]))
			m := int16((int32(l) + int32(r)) / 2)
			binary.LittleEndian.PutUint16(out[i/2:], uint16(m))
		}
		return out
	}

	return data
}
