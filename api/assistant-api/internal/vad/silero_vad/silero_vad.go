package internal_silero_vad

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_vad "github.com/rapidaai/api/assistant-api/internal/vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/streamer45/silero-vad-go/speech"
)

// SileroVAD implements Vad using silero-vad-go
type SileroVAD struct {
	logger       commons.Logger
	inputConfig  *protos.AudioConfig
	detector     *speech.Detector
	onActivity   func(*internal_vad.VadResult) error
	audioSampler *internal_audio.AudioResampler
	vadConfig    *protos.AudioConfig
}

// NewSileroVAD creates a new SileroVAD
func NewSileroVAD(logger commons.Logger,
	inputAudio *protos.AudioConfig,
	callback internal_vad.VADCallback, options utils.Option) (internal_vad.Vad, error) {

	envModelPath := os.Getenv("SILERO_MODEL_PATH")
	if envModelPath == "" {
		_, path, _, _ := runtime.Caller(0)
		envModelPath = filepath.Join(filepath.Dir(path), "models/silero_vad_20251001.onnx")
	}
	vadAudioConfig := internal_audio.NewLinear16khzMonoAudioConfig()
	threshold := 0.5
	if thr, err := options.GetFloat64("microphone.vad.threshold"); err == nil {
		threshold = thr
	}
	config := speech.DetectorConfig{
		ModelPath:            envModelPath,
		SampleRate:           int(vadAudioConfig.SampleRate),
		Threshold:            float32(threshold),
		MinSilenceDurationMs: 100,
		SpeechPadMs:          30,
	}
	detector, err := speech.NewDetector(config)
	if err != nil {
		return nil, err
	}
	return &SileroVAD{
		detector:    detector,
		inputConfig: inputAudio,
		vadConfig:   vadAudioConfig,
		onActivity:  callback,
		logger:      logger,
	}, nil
}

func (s *SileroVAD) Name() string {
	return "silero_vad"
}

// ProcessFrame buffers incoming audio and periodically calls Detect
func (svad *SileroVAD) Process(input []byte) error {
	idi, err := svad.audioSampler.Resample(input, svad.inputConfig, svad.vadConfig)
	if err != nil {
		svad.logger.Debugf("Resampling failed: %+v", err) // Improved logging
		return err
	}

	floatSample, err := svad.audioSampler.ConvertToFloat32Samples(idi, svad.vadConfig)
	if err != nil {
		svad.logger.Debugf("Sample conversion failed: %+v", err) // Improved logging
		return err
	}

	segments, err := svad.detector.Detect(floatSample)
	if err != nil {
		return fmt.Errorf("error during detection: %w", err) // Return error with context
	}

	if len(segments) == 0 { // No speech detected
		return nil
	}
	// Initialize minStart to a large value and maxEnd to a small value
	minStart := math.MaxFloat64
	maxEnd := -math.MaxFloat64

	for _, seg := range segments {
		start := float64(seg.SpeechStartAt)
		end := float64(seg.SpeechEndAt)
		if start < minStart {
			minStart = start
		}
		if end > maxEnd {
			maxEnd = end
		}
	}
	svad.onActivity(&internal_vad.VadResult{StartSec: minStart, EndSec: maxEnd})
	return nil
}
func (s *SileroVAD) Close() error {
	s.detector.Destroy()
	return nil
}
