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

type cachedEngine struct {
	mutex         sync.Mutex
	goResampler   resampling.Resampler
	soxrResampler *nativePCM16Resampler
	inputSamples  []float64
	outputBytes   []byte
}

type ratePair struct {
	sourceRate uint32
	targetRate uint32
}

// Resampler reuses one streaming engine for each sample-rate pair.
type Resampler struct {
	logger               commons.Logger
	quality              resampling.QualityPreset
	useNativeHighQuality bool
	engines              sync.Map
	lifecycleMu          sync.RWMutex
	closed               bool
}

type options struct {
	logger               commons.Logger
	quality              resampling.QualityPreset
	useNativeHighQuality bool
}

type Option func(*options)

var (
	_ internal_type.AudioResampler       = (*Resampler)(nil)
	_ internal_type.AudioStreamResampler = (*Writer)(nil)
)

type Writer struct {
	resampler *Resampler
	source    protos.AudioConfig
	target    protos.AudioConfig
	sink      internal_type.AudioResampleSink
}

// WithLogger sets the resampler logger.
func WithLogger(logger commons.Logger) Option {
	return func(configuration *options) {
		configuration.logger = logger
	}
}

// WithHighQuality selects the higher-quality streaming filter.
func WithHighQuality() Option {
	return func(configuration *options) {
		configuration.quality = resampling.QualityHigh
		configuration.useNativeHighQuality = true
	}
}

// WithQuickQuality selects the low-latency streaming filter.
func WithQuickQuality() Option {
	return func(configuration *options) {
		configuration.quality = resampling.QualityQuick
	}
}

// New creates a streaming audio resampler with high-quality output by default.
func New(optionFunctions ...Option) *Resampler {
	configuration := options{quality: defaultQuality}
	for _, option := range optionFunctions {
		if option != nil {
			option(&configuration)
		}
	}
	return &Resampler{
		logger:               configuration.logger,
		quality:              configuration.quality,
		useNativeHighQuality: configuration.useNativeHighQuality,
	}
}

func NewWriter(
	source, target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
	optionFunctions ...Option,
) (*Writer, error) {
	return New(optionFunctions...).NewWriter(source, target, sink)
}

func (resampler *Resampler) NewWriter(
	source, target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
) (*Writer, error) {
	if source == nil || target == nil {
		return nil, ErrAudioConfigRequired
	}
	if sink == nil {
		return nil, ErrResampleSinkRequired
	}
	return &Writer{
		resampler: resampler,
		source:    *source,
		target:    *target,
		sink:      sink,
	}, nil
}

func (writer *Writer) Write(data []byte) error {
	if writer == nil || writer.resampler == nil {
		return ErrResamplerClosed
	}
	return writer.resampler.write(data, &writer.source, &writer.target, writer.sink)
}

func (writer *Writer) Flush() error {
	if writer == nil || writer.resampler == nil {
		return ErrResamplerClosed
	}
	return writer.resampler.flush(&writer.source, &writer.target, writer.sink)
}

func (writer *Writer) Close() {
	if writer == nil || writer.resampler == nil {
		return
	}
	writer.resampler.Close()
}

// Resample converts audio while preserving streaming filter state.
func (resampler *Resampler) Resample(
	data []byte,
	source, target *protos.AudioConfig,
) ([]byte, error) {
	resampler.lifecycleMu.RLock()
	defer resampler.lifecycleMu.RUnlock()
	if resampler.closed {
		return nil, ErrResamplerClosed
	}

	if source == nil || target == nil {
		return nil, ErrAudioConfigRequired
	}

	if len(data) == 0 {
		return []byte{}, nil
	}

	if source.SampleRate == target.SampleRate &&
		source.Channels == target.Channels &&
		source.AudioFormat == target.AudioFormat {
		return data, nil
	}
	if source.SampleRate != target.SampleRate &&
		source.Channels == monoChannelCount &&
		target.Channels == monoChannelCount {
		return resampler.resampleMono(data, source, target)
	}

	pcm := data
	if source.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = resampler.convertToLinear16(data, source)
		if err != nil {
			return nil, err
		}
	}

	if source.SampleRate != target.SampleRate {
		var err error
		pcm, err = resampler.resamplePCM16(pcm, source.SampleRate, target.SampleRate)
		if err != nil {
			return nil, err
		}
	}

	if source.Channels != target.Channels {
		var err error
		pcm, err = resampler.convertChannels(pcm, source.Channels, target.Channels)
		if err != nil {
			return nil, err
		}
	}

	if target.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = resampler.convertFromLinear16(pcm, target)
		if err != nil {
			return nil, err
		}
	}

	return pcm, nil
}

func (resampler *Resampler) write(
	data []byte,
	source, target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
) error {
	resampler.lifecycleMu.RLock()
	defer resampler.lifecycleMu.RUnlock()
	if resampler.closed {
		return ErrResamplerClosed
	}
	if source == nil || target == nil {
		return ErrAudioConfigRequired
	}
	if sink == nil {
		return ErrResampleSinkRequired
	}
	if len(data) == 0 {
		return nil
	}
	if source.SampleRate == target.SampleRate &&
		source.Channels == target.Channels &&
		source.AudioFormat == target.AudioFormat {
		return sink(data)
	}
	if source.SampleRate != target.SampleRate &&
		source.Channels == monoChannelCount &&
		target.Channels == monoChannelCount {
		return resampler.resampleMonoToSink(data, source, target, sink)
	}

	pcm := data
	if source.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = resampler.convertToLinear16(data, source)
		if err != nil {
			return err
		}
	}

	if source.SampleRate != target.SampleRate {
		var err error
		pcm, err = resampler.resamplePCM16(pcm, source.SampleRate, target.SampleRate)
		if err != nil {
			return err
		}
	}

	if source.Channels != target.Channels {
		var err error
		pcm, err = resampler.convertChannels(pcm, source.Channels, target.Channels)
		if err != nil {
			return err
		}
	}

	if target.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = resampler.convertFromLinear16(pcm, target)
		if err != nil {
			return err
		}
	}

	return sink(pcm)
}

func (resampler *Resampler) flush(
	source, target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
) error {
	resampler.lifecycleMu.RLock()
	defer resampler.lifecycleMu.RUnlock()
	if resampler.closed {
		return ErrResamplerClosed
	}
	if source == nil || target == nil {
		return ErrAudioConfigRequired
	}
	if sink == nil {
		return ErrResampleSinkRequired
	}
	if source.SampleRate == target.SampleRate ||
		source.Channels != monoChannelCount {
		return nil
	}

	ratePair := ratePair{sourceRate: source.SampleRate, targetRate: target.SampleRate}
	existingEngine, exists := resampler.engines.Load(ratePair)
	if !exists {
		return nil
	}
	engine := existingEngine.(*cachedEngine)
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if engine.soxrResampler == nil {
		return nil
	}
	err := engine.soxrResampler.FlushTo(func(pcm []byte) error {
		if source.Channels != target.Channels {
			var err error
			pcm, err = resampler.convertChannels(pcm, source.Channels, target.Channels)
			if err != nil {
				return err
			}
		}
		return engine.writeLinear16ToTarget(pcm, target, sink)
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrResamplingFailed, err)
	}
	engine.soxrResampler.Close()
	resampler.engines.Delete(ratePair)
	return nil
}

// Close releases native resampling state.
func (resampler *Resampler) Close() {
	if resampler == nil {
		return
	}
	resampler.lifecycleMu.Lock()
	defer resampler.lifecycleMu.Unlock()
	if resampler.closed {
		return
	}
	resampler.closed = true
	resampler.engines.Range(func(key, value any) bool {
		engine := value.(*cachedEngine)
		if engine.soxrResampler != nil {
			engine.soxrResampler.Close()
		}
		resampler.engines.Delete(key)
		return true
	})
}

func (resampler *Resampler) resampleMono(data []byte, source, target *protos.AudioConfig) ([]byte, error) {
	engine, err := resampler.getOrCreateEngine(source.SampleRate, target.SampleRate)
	if err != nil {
		return nil, err
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if engine.soxrResampler != nil {
		pcm := data
		if source.AudioFormat != protos.AudioConfig_LINEAR16 {
			pcm, err = resampler.convertToLinear16(data, source)
			if err != nil {
				return nil, err
			}
		}
		pcm, err = engine.soxrResampler.Resample(pcm)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrResamplingFailed, err)
		}
		if target.AudioFormat != protos.AudioConfig_LINEAR16 {
			return resampler.convertFromLinear16(pcm, target)
		}
		return pcm, nil
	}

	engine.inputSamples, err = decodeMono(engine.inputSamples, data, source.AudioFormat)
	if err != nil {
		return nil, err
	}
	output, err := engine.goResampler.Process(engine.inputSamples)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResamplingFailed, err)
	}
	return encodeMono(output, target.AudioFormat)
}

func (resampler *Resampler) resampleMonoToSink(
	data []byte,
	source, target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
) error {
	engine, err := resampler.getOrCreateEngine(source.SampleRate, target.SampleRate)
	if err != nil {
		return err
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if engine.soxrResampler == nil {
		engine.inputSamples, err = decodeMono(engine.inputSamples, data, source.AudioFormat)
		if err != nil {
			return err
		}
		resampled, err := engine.goResampler.Process(engine.inputSamples)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrResamplingFailed, err)
		}
		output, err := encodeMono(resampled, target.AudioFormat)
		if err != nil {
			return err
		}
		if len(output) == 0 {
			return nil
		}
		return sink(output)
	}
	pcm := data
	if source.AudioFormat != protos.AudioConfig_LINEAR16 {
		pcm, err = resampler.convertToLinear16(data, source)
		if err != nil {
			return err
		}
	}
	err = engine.soxrResampler.WriteTo(pcm, func(output []byte) error {
		return engine.writeLinear16ToTarget(output, target, sink)
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrResamplingFailed, err)
	}
	return nil
}

func (engine *cachedEngine) writeLinear16ToTarget(
	pcm []byte,
	target *protos.AudioConfig,
	sink internal_type.AudioResampleSink,
) error {
	if len(pcm) == 0 {
		return nil
	}
	switch target.AudioFormat {
	case protos.AudioConfig_LINEAR16:
		return sink(pcm)
	case protos.AudioConfig_MuLaw8:
		sampleCount := len(pcm) / pcm16BytesPerSample
		if cap(engine.outputBytes) < sampleCount {
			engine.outputBytes = make([]byte, sampleCount)
		} else {
			engine.outputBytes = engine.outputBytes[:sampleCount]
		}
		for index := range engine.outputBytes {
			engine.outputBytes[index] = g711.EncodeUlawFrame(int16(binary.LittleEndian.Uint16(pcm[index*pcm16BytesPerSample:])))
		}
		return sink(engine.outputBytes)
	default:
		return fmt.Errorf("%w: %v", ErrUnsupportedOutputFormat, target.AudioFormat)
	}
}

func (resampler *Resampler) resamplePCM16(pcm []byte, sourceRate, targetRate uint32) ([]byte, error) {
	engine, err := resampler.getOrCreateEngine(sourceRate, targetRate)
	if err != nil {
		return nil, err
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if engine.soxrResampler != nil {
		output, err := engine.soxrResampler.Resample(pcm)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrResamplingFailed, err)
		}
		return output, nil
	}

	output, err := engine.goResampler.Process(pcm16ToFloat64(pcm))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResamplingFailed, err)
	}

	return float64ToPCM16(output), nil
}

func (resampler *Resampler) getOrCreateEngine(sourceRate, targetRate uint32) (*cachedEngine, error) {
	ratePair := ratePair{sourceRate: sourceRate, targetRate: targetRate}

	if existingEngine, exists := resampler.engines.Load(ratePair); exists {
		return existingEngine.(*cachedEngine), nil
	}

	nativeQuality, useNative := resampler.nativeQuality()
	if useNative {
		soxrResampler, err := newNativePCM16Resampler(sourceRate, targetRate, nativeQuality)
		if err == nil {
			newEngine := &cachedEngine{soxrResampler: soxrResampler}
			cachedValue, alreadyExists := resampler.engines.LoadOrStore(ratePair, newEngine)
			if alreadyExists {
				soxrResampler.Close()
			}
			return cachedValue.(*cachedEngine), nil
		}
		if resampler.logger != nil {
			resampler.logger.Warnw("Native SOXR unavailable, using Go resampler", "error", err)
		}
	}

	goResampler, err := resampling.New(&resampling.Config{
		InputRate:  float64(sourceRate),
		OutputRate: float64(targetRate),
		Channels:   monoChannelCount,
		EnableSIMD: true,
		Quality:    resampling.QualitySpec{Preset: resampler.quality},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResamplerInitializationFailed, err)
	}

	newEngine := &cachedEngine{goResampler: goResampler}
	cachedValue, _ := resampler.engines.LoadOrStore(ratePair, newEngine)
	return cachedValue.(*cachedEngine), nil
}

func (resampler *Resampler) nativeQuality() (nativeSOXRQuality, bool) {
	switch resampler.quality {
	case resampling.QualityQuick:
		return nativeSOXRQualityQuick, true
	case resampling.QualityHigh:
		return nativeSOXRQualityLiveKit, resampler.useNativeHighQuality
	default:
		return nativeSOXRQualityQuick, false
	}
}

func decodeMono(samples []float64, data []byte, format protos.AudioConfig_AudioFormat) ([]float64, error) {
	var sampleCount int
	switch format {
	case protos.AudioConfig_LINEAR16:
		sampleCount = len(data) / pcm16BytesPerSample
	case protos.AudioConfig_MuLaw8:
		sampleCount = len(data)
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedInputFormat, format)
	}

	if cap(samples) < sampleCount {
		samples = make([]float64, sampleCount)
	} else {
		samples = samples[:sampleCount]
	}
	switch format {
	case protos.AudioConfig_LINEAR16:
		for index := range samples {
			// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
			sample := int16(binary.LittleEndian.Uint16(data[index*pcm16BytesPerSample:]))
			samples[index] = float64(sample) / pcm16Scale
		}
	case protos.AudioConfig_MuLaw8:
		for index := range samples {
			samples[index] = float64(g711.DecodeUlawFrame(data[index])) / pcm16Scale
		}
	}
	return samples, nil
}

func encodeMono(samples []float64, format protos.AudioConfig_AudioFormat) ([]byte, error) {
	switch format {
	case protos.AudioConfig_LINEAR16:
		return float64ToPCM16(samples), nil
	case protos.AudioConfig_MuLaw8:
		encoded := make([]byte, len(samples))
		for index, sample := range samples {
			encoded[index] = g711.EncodeUlawFrame(float64ToInt16(sample))
		}
		return encoded, nil
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedOutputFormat, format)
	}
}

func pcm16ToFloat64(data []byte) []float64 {
	evenByteCount := len(data) - len(data)%pcm16BytesPerSample
	output := make([]float64, evenByteCount/pcm16BytesPerSample)
	for byteIndex := 0; byteIndex < evenByteCount; byteIndex += pcm16BytesPerSample {
		// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
		sample := int16(binary.LittleEndian.Uint16(data[byteIndex:]))
		output[byteIndex/pcm16BytesPerSample] = float64(sample) / pcm16Scale
	}
	return output
}

func float64ToPCM16(data []float64) []byte {
	output := make([]byte, len(data)*pcm16BytesPerSample)
	for sampleIndex, sample := range data {
		pcm16Sample := float64ToInt16(sample)
		// #nosec G115, PCM16 encoding preserves the signed sample bits.
		binary.LittleEndian.PutUint16(output[sampleIndex*pcm16BytesPerSample:], uint16(pcm16Sample))
	}
	return output
}

func float64ToInt16(sample float64) int16 {
	if sample > 1 {
		sample = 1
	} else if sample < -1 {
		sample = -1
	}
	return int16(sample * pcm16PositiveLimit)
}

func (resampler *Resampler) convertToLinear16(data []byte, config *protos.AudioConfig) ([]byte, error) {
	switch config.AudioFormat {
	case protos.AudioConfig_LINEAR16:
		return data, nil
	case protos.AudioConfig_MuLaw8:
		return g711.DecodeUlaw(data), nil
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedInputFormat, config.AudioFormat)
	}
}

func (resampler *Resampler) convertFromLinear16(data []byte, config *protos.AudioConfig) ([]byte, error) {
	switch config.AudioFormat {
	case protos.AudioConfig_LINEAR16:
		return data, nil
	case protos.AudioConfig_MuLaw8:
		return g711.EncodeUlaw(data), nil
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedOutputFormat, config.AudioFormat)
	}
}

func (resampler *Resampler) convertChannels(data []byte, sourceChannels, targetChannels uint32) ([]byte, error) {
	if sourceChannels == targetChannels {
		return data, nil
	}

	if sourceChannels == monoChannelCount && targetChannels == stereoChannelCount {
		if len(data)%pcm16BytesPerSample != 0 {
			return nil, fmt.Errorf("%w: bytes=%d channels=%d", ErrInvalidPCM16ChannelFrame, len(data), sourceChannels)
		}
		output := make([]byte, len(data)*stereoChannelCount)
		for byteIndex := 0; byteIndex < len(data); byteIndex += pcm16BytesPerSample {
			copy(output[byteIndex*stereoChannelCount:], data[byteIndex:byteIndex+pcm16BytesPerSample])
			copy(output[byteIndex*stereoChannelCount+pcm16BytesPerSample:], data[byteIndex:byteIndex+pcm16BytesPerSample])
		}
		return output, nil
	}

	if sourceChannels == stereoChannelCount && targetChannels == monoChannelCount {
		stereoFrameBytes := pcm16BytesPerSample * stereoChannelCount
		if len(data)%stereoFrameBytes != 0 {
			return nil, fmt.Errorf("%w: bytes=%d channels=%d", ErrInvalidPCM16ChannelFrame, len(data), sourceChannels)
		}
		output := make([]byte, len(data)/stereoChannelCount)
		for byteIndex := 0; byteIndex < len(data); byteIndex += stereoFrameBytes {
			// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
			leftSample := int16(binary.LittleEndian.Uint16(data[byteIndex:]))
			// #nosec G115, PCM16 decoding preserves the source two's-complement bits.
			rightSample := int16(binary.LittleEndian.Uint16(data[byteIndex+pcm16BytesPerSample:]))
			// #nosec G115, averaging two int16 samples remains within int16 range.
			monoSample := int16((int32(leftSample) + int32(rightSample)) / 2)
			// #nosec G115, PCM16 encoding preserves the signed sample bits.
			binary.LittleEndian.PutUint16(output[byteIndex/stereoChannelCount:], uint16(monoSample))
		}
		return output, nil
	}

	return nil, fmt.Errorf("%w: %d to %d", ErrUnsupportedChannelConversion, sourceChannels, targetChannels)
}
