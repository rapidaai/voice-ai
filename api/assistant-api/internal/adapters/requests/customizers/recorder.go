package internal_adapter_request_customizers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/pkg/commons"
)

// AudioChunk represents a timestamped audio chunk
type AudioChunk struct {
	Data      []byte
	Timestamp time.Time
	IsSystem  bool // true for system audio, false for user audio
	Config    *internal_audio.AudioConfig
	ID        int64 // Add unique identifier for each chunk
}

type Recorder interface {
	Init(userAudioConfig, systemAudioConfig *internal_audio.AudioConfig) error
	User(in []byte) error
	Interrupt() error
	System(out []byte) error
	Persist() ([]byte, error)
}

type recorder struct {
	logger           commons.Logger
	mu               sync.Mutex   // Ensure thread-safe access
	audioChunks      []AudioChunk // All audio chunks with timestamps
	interruptionTime *time.Time   // When interruption occurred (nil if no interruption)
	userConfig       *internal_audio.AudioConfig
	systemConfig     *internal_audio.AudioConfig
	chunkIDCounter   int64 // Counter for unique chunk IDs
}

func NewRecorder(logger commons.Logger) Recorder {
	return &recorder{
		logger:      logger,
		audioChunks: []AudioChunk{},
		mu:          sync.Mutex{},
	}
}

func (r *recorder) Init(userConfig, systemConfig *internal_audio.AudioConfig) error {
	r.userConfig = userConfig
	r.systemConfig = systemConfig
	return nil
}

func (r *recorder) User(in []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.chunkIDCounter++
	chunk := AudioChunk{
		Data:      make([]byte, len(in)),
		Timestamp: time.Now(),
		IsSystem:  false,
		Config:    r.userConfig,
		ID:        r.chunkIDCounter,
	}
	copy(chunk.Data, in)
	r.audioChunks = append(r.audioChunks, chunk)
	return nil
}
func (r *recorder) Interrupt() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	interruptTime := time.Now()
	// Check if the last interruption occurred very recently
	if r.interruptionTime != nil {
		elapsed := interruptTime.Sub(*r.interruptionTime)
		if elapsed <= 100*time.Millisecond && elapsed >= 50*time.Millisecond {
			// r.logger.Info("Interrupt ignored. Too soon after the previous interruption: ", elapsed)
			// Update interruption time regardless of ignoring
			r.interruptionTime = &interruptTime
			return nil
		}
	}

	// Update interruption time
	r.interruptionTime = &interruptTime
	// r.logger.Info("User interruption detected at ", interruptTime)

	// Remove system audio chunks that would "play" after interruption
	r.removeInterruptedSystemAudio(interruptTime)

	return nil
}

func (r *recorder) System(out []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.chunkIDCounter++
	chunk := AudioChunk{
		Data:      make([]byte, len(out)),
		Timestamp: time.Now(),
		IsSystem:  true,
		Config:    r.systemConfig,
		ID:        r.chunkIDCounter,
	}
	copy(chunk.Data, out)
	r.audioChunks = append(r.audioChunks, chunk)
	return nil
}

func (r *recorder) removeInterruptedSystemAudio(interruptTime time.Time) {
	silencedCount := 0
	modifiedCount := 0

	// r.logger.Info("Processing interruption at", interruptTime)
	adjustedInterruptTime := interruptTime

	for i := range r.audioChunks {
		chunk := &r.audioChunks[i]
		if !chunk.IsSystem {
			continue // Only process system audio chunks
		}

		// Calculate playback times for the chunk
		playStartTime := r.calculateSystemAudioPlayTime(*chunk)
		chunkDuration := r.calculateChunkDuration(*chunk)
		playEndTime := playStartTime.Add(chunkDuration)

		// r.logger.Debug(fmt.Sprintf(
		// 	"Chunk ID=%d: Plays from %v to %v (Duration: %.2fms, Format: %v, Sample Rate: %d)",
		// 	chunk.ID, playStartTime, playEndTime, chunkDuration.Seconds()*1000, chunk.Config.Format, chunk.Config.SampleRate,
		// ))

		// Analyze overlap with the interruption time
		if playStartTime.After(adjustedInterruptTime) || playStartTime.Equal(adjustedInterruptTime) {
			// r.logger.Debug(fmt.Sprintf("Chunk ID=%d: No interruption - plays after %v", chunk.ID, adjustedInterruptTime))
			continue // Chunk starts playing after the interruption, keep it unchanged
		}

		if playEndTime.Before(adjustedInterruptTime) {
			// r.logger.Debug(fmt.Sprintf("Chunk ID=%d: No interruption - ends before %v", chunk.ID, adjustedInterruptTime))
			continue // Chunk ends before the interruption, keep it unchanged
		}

		if playStartTime.Before(adjustedInterruptTime) && playEndTime.After(adjustedInterruptTime) {
			// Overlap with interruption: trim or silence the chunk
			keepDuration := adjustedInterruptTime.Sub(playStartTime)

			if keepDuration > 0 {
				// Calculate bytes to keep based on duration, sample rate, and format
				trimmedData := r.trimAudioChunkData(chunk.Data, keepDuration, chunk.Config)
				// r.logger.Debug(fmt.Sprintf(
				// 	"Chunk ID=%d: Partially silenced. Keeping %.2fms, silencing %.2fms",
				// 	chunk.ID,
				// 	keepDuration.Seconds()*1000,
				// 	(chunkDuration-keepDuration).Seconds()*1000,
				// ))
				chunk.Data = trimmedData
				modifiedCount++
			} else {
				// Fully silence the chunk if no valid duration remains
				r.logger.Debug(fmt.Sprintf("Chunk ID=%d: Fully silenced", chunk.ID))
				chunk.Data = r.createSilentAudioData(len(chunk.Data))
				silencedCount++
			}
			continue
		}
	}

	r.logger.Info(fmt.Sprintf("Interruption processed: silenced %d chunks, modified %d chunks", silencedCount, modifiedCount))
}

func (r *recorder) createSilentAudioData(byteLength int) []byte {
	// Generate silence: zero bytes for the specified length
	return make([]byte, byteLength)
}

func (r *recorder) trimAudioChunkData(data []byte, keepDuration time.Duration, config *internal_audio.AudioConfig) []byte {
	if config == nil || keepDuration <= 0 {
		return []byte{}
	}

	// Calculate bytes per sample based on audio format
	var bytesPerSample int
	switch config.Format {
	case internal_audio.Linear16:
		bytesPerSample = 2
	case internal_audio.MuLaw8:
		bytesPerSample = 1
	default:
		bytesPerSample = 2 // Default to 16-bit PCM
	}

	// Calculate bytes per frame (sample + channels)
	bytesPerFrame := bytesPerSample * config.Channels

	// Calculate total samples to keep based on duration
	samplesToKeep := int(keepDuration.Seconds() * float64(config.SampleRate))

	// Calculate bytes to keep (ensure frame alignment)
	bytesToKeep := samplesToKeep * bytesPerFrame
	bytesToKeep = (bytesToKeep / bytesPerFrame) * bytesPerFrame // Round down to nearest frame boundary

	if bytesToKeep > len(data) {
		bytesToKeep = len(data) // Ensure data length is not exceeded
	}

	if bytesToKeep <= 0 {
		return []byte{} // Return empty if no valid data remains
	}

	// Return trimmed data
	trimmedData := make([]byte, bytesToKeep)
	copy(trimmedData, data[:bytesToKeep])

	return trimmedData
}

func (r *recorder) calculateSystemAudioPlayTime(targetChunk AudioChunk) time.Time {
	var latestEndTime time.Time
	isFirstSystemChunk := true

	for _, chunk := range r.audioChunks {
		if chunk.ID == targetChunk.ID {
			break // Stop as soon as target chunk is reached
		}

		if chunk.IsSystem {
			playStartTime := latestEndTime
			if chunk.Timestamp.After(latestEndTime) {
				playStartTime = chunk.Timestamp
			}
			latestEndTime = playStartTime.Add(r.calculateChunkDuration(chunk))
			isFirstSystemChunk = false
		}
	}

	playStartTime := targetChunk.Timestamp
	if !isFirstSystemChunk && latestEndTime.After(targetChunk.Timestamp) {
		playStartTime = latestEndTime
	}

	return playStartTime
}

func (r *recorder) Persist() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.audioChunks) == 0 {
		// r.logger.Info("No audio chunks to persist")
		return nil, fmt.Errorf("empty chunk of audio")
	}

	// Sort chunks by timestamp to maintain chronological order
	sort.Slice(r.audioChunks, func(i, j int) bool {
		return r.audioChunks[i].Timestamp.Before(r.audioChunks[j].Timestamp)
	})

	// Determine the target audio configuration
	targetConfig := r.getTargetAudioConfig()
	if targetConfig == nil {
		return nil, fmt.Errorf("no valid audio configuration found")
	}

	// Convert and merge all chunks
	mergedAudio, err := r.mergeAudioChunks(targetConfig)
	if err != nil {
		r.logger.Error("Failed to merge audio chunks", err)
		return nil, err
	}

	// Create WAV file
	wavData, err := r.createWAVFile(mergedAudio, targetConfig)
	if err != nil {
		r.logger.Error("Failed to create WAV file", err)
		return nil, err
	}

	// r.logger.Info(fmt.Sprintf("Persisted audio with %d chunks", len(r.audioChunks)))
	return wavData, nil
}

func (r *recorder) getTargetAudioConfig() *internal_audio.AudioConfig {
	if r.userConfig != nil {
		return r.userConfig
	}
	if r.systemConfig != nil {
		return r.systemConfig
	}
	if len(r.audioChunks) > 0 && r.audioChunks[0].Config != nil {
		return r.audioChunks[0].Config
	}
	return nil
}

func (r *recorder) mergeAudioChunks(targetConfig *internal_audio.AudioConfig) ([]byte, error) {
	if len(r.audioChunks) == 0 {
		return nil, fmt.Errorf("no audio chunks to merge")
	}

	// Calculate total duration and create timeline
	startTime := r.audioChunks[0].Timestamp
	var endTime time.Time
	for _, chunk := range r.audioChunks {
		chunkDuration := r.calculateChunkDuration(chunk)
		chunkEndTime := chunk.Timestamp.Add(chunkDuration)
		if chunkEndTime.After(endTime) {
			endTime = chunkEndTime
		}
	}

	totalDuration := endTime.Sub(startTime)
	totalSamples := int(totalDuration.Seconds() * float64(targetConfig.SampleRate))

	// Create output buffer (16-bit samples)
	outputSamples := make([]int32, totalSamples*targetConfig.Channels)

	// Process each chunk and add to the output buffer
	for _, chunk := range r.audioChunks {
		err := r.addChunkToOutput(chunk, outputSamples, startTime, targetConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to add chunk to output: %v", err)
		}
	}

	// Convert int32 samples to 16-bit PCM
	pcmData := make([]byte, len(outputSamples)*2)
	for i, sample := range outputSamples {
		// Clamp to 16-bit range
		if sample > 32767 {
			sample = 32767
		} else if sample < -32768 {
			sample = -32768
		}
		binary.LittleEndian.PutUint16(pcmData[i*2:], uint16(sample))
	}

	return pcmData, nil
}

func (r *recorder) calculateChunkDuration(chunk AudioChunk) time.Duration {
	if chunk.Config == nil {
		return 0
	}
	var bytesPerSample int
	switch chunk.Config.Format {
	case internal_audio.Linear16:
		bytesPerSample = 2
	case internal_audio.MuLaw8:
		bytesPerSample = 1
	default:
		bytesPerSample = 2 // default to 16-bit
	}

	samples := len(chunk.Data) / (bytesPerSample * chunk.Config.Channels)
	duration := float64(samples) / float64(chunk.Config.SampleRate)
	return time.Duration(duration * float64(time.Second))
}

func (r *recorder) addChunkToOutput(chunk AudioChunk, output []int32, startTime time.Time, targetConfig *internal_audio.AudioConfig) error {
	if chunk.Config == nil {
		return fmt.Errorf("chunk has no audio configuration")
	}

	// Convert chunk data to int32 samples
	chunkSamples, err := r.convertToSamples(chunk.Data, chunk.Config)
	if err != nil {
		return err
	}

	// Find the last non-zero sample in the output buffer
	lastNonZeroIndex := len(output) - 1
	for lastNonZeroIndex >= 0 && output[lastNonZeroIndex] == 0 {
		lastNonZeroIndex--
	}

	// Determine where to start adding the new samples
	var startIndex int
	if chunk.IsSystem {
		// For system audio, start right after the last non-zero sample
		startIndex = lastNonZeroIndex + 1
	} else {
		// For user audio, use the timestamp-based offset
		offsetDuration := chunk.Timestamp.Sub(startTime)
		startIndex = int(offsetDuration.Seconds()*float64(targetConfig.SampleRate)) * targetConfig.Channels
	}

	// Add samples to output buffer
	for i, sample := range chunkSamples {
		outputIndex := startIndex + i
		if outputIndex >= 0 && outputIndex < len(output) {
			if chunk.IsSystem {
				// For system audio, simply place it
				output[outputIndex] = sample
			} else {
				// For user audio, mix with existing audio
				output[outputIndex] += sample
			}
		}
	}

	return nil
}

func (r *recorder) convertToSamples(data []byte, config *internal_audio.AudioConfig) ([]int32, error) {
	var samples []int32

	switch config.Format {
	case internal_audio.Linear16:
		samples = make([]int32, len(data)/2)
		for i := 0; i < len(samples); i++ {
			sample := int16(binary.LittleEndian.Uint16(data[i*2:]))
			samples[i] = int32(sample)
		}
	case internal_audio.MuLaw8:
		samples = make([]int32, len(data))
		for i, b := range data {
			samples[i] = r.muLawToLinear(b)
		}
	default:
		return nil, fmt.Errorf("unsupported audio format: %v", config.Format)
	}

	return samples, nil
}

func (r *recorder) muLawToLinear(muLawByte byte) int32 {
	// μ-law to linear PCM conversion
	muLawByte = ^muLawByte
	sign := muLawByte & 0x80
	exponent := (muLawByte >> 4) & 0x07
	mantissa := muLawByte & 0x0F

	sample := int32(mantissa<<1 + 33)
	sample <<= exponent
	sample -= 33

	if sign != 0 {
		sample = -sample
	}

	return sample << 2 // Scale to 16-bit range
}

func (r *recorder) createWAVFile(pcmData []byte, config *internal_audio.AudioConfig) ([]byte, error) {
	var buf bytes.Buffer

	// WAV header
	// RIFF header
	buf.Write([]byte("RIFF"))
	binary.Write(&buf, binary.LittleEndian, uint32(36+len(pcmData))) // File size - 8
	buf.Write([]byte("WAVE"))

	// fmt chunk
	buf.Write([]byte("fmt "))
	binary.Write(&buf, binary.LittleEndian, uint32(16))                                  // fmt chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))                                   // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(config.Channels))                     // Number of channels
	binary.Write(&buf, binary.LittleEndian, uint32(config.SampleRate))                   // Sample rate
	binary.Write(&buf, binary.LittleEndian, uint32(config.SampleRate*config.Channels*2)) // Byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(config.Channels*2))                   // Block align
	binary.Write(&buf, binary.LittleEndian, uint16(16))                                  // Bits per sample

	// data chunk
	buf.Write([]byte("data"))
	binary.Write(&buf, binary.LittleEndian, uint32(len(pcmData)))
	buf.Write(pcmData)

	return buf.Bytes(), nil
}
