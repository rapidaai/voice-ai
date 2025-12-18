package internal_audio

import (
	"encoding/binary"
	"fmt"
)

const muLawBias = 33

// AudioInfo holds information about audio data
type AudioInfo struct {
	SampleRate        int
	Format            AudioFormat
	Channels          int
	SamplesPerChannel int
	BytesPerSample    int
	TotalBytes        int
	DurationSeconds   float64
}

// String returns a formatted string representation of AudioInfo
func (info AudioInfo) String() string {
	formatName := "Unknown"
	switch info.Format {
	case Linear16:
		formatName = "Linear16"
	case MuLaw8:
		formatName = "μ-law 8-bit"
	}

	channelName := "Mono"
	if info.Channels == 2 {
		channelName = "Stereo"
	} else if info.Channels > 2 {
		channelName = fmt.Sprintf("%d channels", info.Channels)
	}

	return fmt.Sprintf("Audio: %s, %d Hz, %s, %.2f seconds (%d samples, %d bytes)",
		formatName, info.SampleRate, channelName, info.DurationSeconds,
		info.SamplesPerChannel, info.TotalBytes)
}

// AudioResampler handles audio resampling operations
type AudioResampler struct{}

// NewAudioResampler creates a new audio resampler instance
func NewAudioResampler() *AudioResampler {
	return &AudioResampler{}
}

// Resample converts audio data from source format to target format
func (r *AudioResampler) Resample(data []byte, source, target *AudioConfig) ([]byte, error) {

	if source.SampleRate == target.SampleRate && source.Channels == target.Channels {
		return data, nil // No resampling needed
	}

	samples, err := r.decodeToFloat64(data, source)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	if source.SampleRate != target.SampleRate {
		samples = r.resampleFloat64(samples, source.SampleRate, target.SampleRate)
	}

	if source.Channels != target.Channels {
		samples = r.convertChannels(samples, source.Channels, target.Channels)
	}

	result, err := r.encodeFromFloat64(samples, target)
	if err != nil {
		return nil, fmt.Errorf("failed to encode audio: %w", err)
	}

	return result, nil
}

// ConvertToFloat32 converts byte audio data to float32 samples with specified sample rate
func (r *AudioResampler) ConvertToFloat32Samples(data []byte, config *AudioConfig) ([]float32, error) {
	// First convert to float64 (our internal format)
	float64Samples, err := r.decodeToFloat64(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode to float64: %w", err)
	}
	// Convert float64 to float32
	float32Samples := make([]float32, len(float64Samples))
	for i, sample := range float64Samples {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		float32Samples[i] = float32(sample)
	}

	return float32Samples, nil
}

// ConvertToFloat32WithResample converts byte audio to float32 with resampling to target sample rate
func (r *AudioResampler) ConvertToFloat32WithResample(data []byte, source *AudioConfig, targetSampleRate int) ([]float32, error) {
	// Create target config with Linear16 format and target sample rate
	target := &AudioConfig{
		SampleRate: targetSampleRate,
		Format:     Linear16,
		Channels:   source.Channels,
	}

	// Resample and convert to Linear16 if needed
	resampledData, err := r.Resample(data, source, target)
	if err != nil {
		return nil, fmt.Errorf("failed to resample: %w", err)
	}

	// Convert Linear16 to float32
	return r.ConvertToFloat32Samples(resampledData, target)
}

// ConvertFromFloat32 converts float32 samples to byte audio data
func (r *AudioResampler) ConvertToByteSamples(samples []float32, config *AudioConfig) ([]byte, error) {
	float64Samples := make([]float64, len(samples))
	for i, sample := range samples {
		float64Samples[i] = float64(sample)
	}
	return r.encodeFromFloat64(float64Samples, config)
}

// GetAudioInfo returns information about the byte audio data
func (r *AudioResampler) GetAudioInfo(data []byte, config AudioConfig) AudioInfo {
	var samplesPerChannel int
	var bytesPerSample int

	switch config.Format {
	case Linear16:
		bytesPerSample = 2
		samplesPerChannel = len(data) / (bytesPerSample * config.Channels)
	case MuLaw8:
		bytesPerSample = 1
		samplesPerChannel = len(data) / (bytesPerSample * config.Channels)
	}

	duration := float64(samplesPerChannel) / float64(config.SampleRate)

	return AudioInfo{
		SampleRate:        config.SampleRate,
		Format:            config.Format,
		Channels:          config.Channels,
		SamplesPerChannel: samplesPerChannel,
		BytesPerSample:    bytesPerSample,
		TotalBytes:        len(data),
		DurationSeconds:   duration,
	}
}

// decodeToFloat64 converts various audio formats to normalized float64 samples
func (r *AudioResampler) decodeToFloat64(data []byte, config *AudioConfig) ([]float64, error) {
	switch config.Format {
	case Linear16:
		return r.decodePCM16ToFloat64(data), nil
	case MuLaw8:
		return r.decodeMuLawToFloat64(data), nil
	default:
		return nil, fmt.Errorf("unsupported input format: %v", config.Format)
	}
}

// encodeFromFloat64 converts normalized float64 samples to target format
func (r *AudioResampler) encodeFromFloat64(samples []float64, config *AudioConfig) ([]byte, error) {
	switch config.Format {
	case Linear16:
		return r.encodeFloat64ToPCM16(samples), nil
	case MuLaw8:
		return r.encodeFloat64ToMuLaw(samples), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %v", config.Format)
	}
}

// decodePCM16ToFloat64 converts 16-bit PCM to normalized float64
func (r *AudioResampler) decodePCM16ToFloat64(data []byte) []float64 {
	samples := make([]float64, len(data)/2)
	for i := 0; i < len(samples); i++ {
		sample := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		samples[i] = float64(sample) / 32768.0
	}
	return samples
}

// encodeFloat64ToPCM16 converts normalized float64 to 16-bit PCM
func (r *AudioResampler) encodeFloat64ToPCM16(samples []float64) []byte {
	data := make([]byte, len(samples)*2)
	const maxInt16 = float64(32767.0)

	for i, sample := range samples {
		// Clamp to [-1.0, 1.0] and convert to int16
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		value := int16(sample * maxInt16)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(value))
	}
	return data
}

// decodeMuLawToFloat64 converts μ-law to normalized float64 using G.711 standard
func (r *AudioResampler) decodeMuLawToFloat64(data []byte) []float64 {
	samples := make([]float64, len(data))
	for i, b := range data {
		sample := r.muLawDecode(b)
		samples[i] = float64(sample) / 32768.0
	}
	return samples
}

// encodeFloat64ToMuLaw converts normalized float64 to μ-law using G.711 standard
func (r *AudioResampler) encodeFloat64ToMuLaw(samples []float64) []byte {
	data := make([]byte, len(samples))
	for i, sample := range samples {
		// Clamp to [-1.0, 1.0] and convert to int16
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		pcmSample := int16(sample * 32767.0)
		data[i] = r.muLawEncode(pcmSample)
	}
	return data
}

// muLawDecode converts μ-law byte to 16-bit PCM using G.711 standard
func (r *AudioResampler) muLawDecode(mulaw byte) int16 {
	mulaw = ^mulaw
	sign := int16(1)
	if (mulaw & 0x80) != 0 {
		mulaw &= 0x7F
		sign = -1
	}
	exponent := (mulaw >> 4) & 0x07
	mantissa := int16(mulaw & 0x0F)
	sample := ((mantissa << 3) + 0x84) << (exponent & 0x07)
	return sign * (sample - muLawBias)
}

// Update the muLawEncode function
func (r *AudioResampler) muLawEncode(sample int16) byte {
	var sign byte
	if sample < 0 {
		sign = 0x80
		sample = -sample
	}
	sample += muLawBias
	exponent := byte(7)
	for i := byte(7); i > 0; i-- {
		if sample > 255 {
			exponent = i
			break
		}
		sample <<= 1
	}
	mantissa := byte((sample >> 4) & 0x0F)
	return ^(sign | (exponent << 4) | mantissa)
}

// resampleFloat64 performs high-quality resampling using linear interpolation
func (r *AudioResampler) resampleFloat64(samples []float64, sourceSR, targetSR int) []float64 {
	if sourceSR == targetSR {
		return samples
	}

	ratio := float64(sourceSR) / float64(targetSR)
	outputLength := int(float64(len(samples)) / ratio)
	resampled := make([]float64, outputLength)

	for i := 0; i < outputLength; i++ {
		sourceIndex := float64(i) * ratio

		// Linear interpolation
		index := int(sourceIndex)
		frac := sourceIndex - float64(index)

		if index >= len(samples)-1 {
			resampled[i] = samples[len(samples)-1]
		} else {
			resampled[i] = samples[index]*(1-frac) + samples[index+1]*frac
		}
	}

	return resampled
}

// convertChannels handles mono/stereo conversions
func (r *AudioResampler) convertChannels(samples []float64, sourceChannels, targetChannels int) []float64 {
	if sourceChannels == targetChannels {
		return samples
	}

	if sourceChannels == 1 && targetChannels == 2 {
		// Mono to stereo - duplicate each sample
		stereo := make([]float64, len(samples)*2)
		for i, sample := range samples {
			stereo[i*2] = sample
			stereo[i*2+1] = sample
		}
		return stereo
	} else if sourceChannels == 2 && targetChannels == 1 {
		// Stereo to mono - average left and right channels
		mono := make([]float64, len(samples)/2)
		for i := 0; i < len(mono); i++ {
			mono[i] = (samples[i*2] + samples[i*2+1]) / 2.0
		}
		return mono
	}

	return samples // Return unchanged if unsupported conversion
}
