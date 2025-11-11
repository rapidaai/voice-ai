package internal_analyzers

import (
	"context"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	internal_voice_rnnoise "github.com/rapidaai/api/assistant-api/internal/voices/rnnoise"
	voice_silero_vad "github.com/rapidaai/api/assistant-api/internal/voices/silero_vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type silerovadAnalyzer struct {
	detector    internal_voices.Detector
	logger      commons.Logger
	opts        *VoiceAnalyzerOptions
	resampler   *internal_voices.AudioResampler
	inputConfig *internal_voices.AudioConfig
}

func (*silerovadAnalyzer) Name() string {
	return "silero-vad-analyzer"
}

func NewSileroVadAnalyzer(logger commons.Logger, audioConfig *internal_voices.AudioConfig, opts *VoiceAnalyzerOptions) (AudioAnalyzer, error) {
	options := &voice_silero_vad.
		DetectorConfig{
		SampleRate:           16000,
		MinSilenceDurationMs: 10,
		SpeechPadMs:          10,
	}

	sd, err := voice_silero_vad.NewSileroDetector(logger, options)
	if err != nil {
		logger.Errorf("failed to create speech detector: %s", err)
		return nil, err
	}
	return &silerovadAnalyzer{
		logger:      logger,
		detector:    sd,
		resampler:   internal_voices.NewAudioResampler(),
		opts:        opts,
		inputConfig: audioConfig,
	}, nil
}

func (sva *silerovadAnalyzer) Close() error {
	return nil
}

func (sva *silerovadAnalyzer) Analyze(ctx context.Context, wavBytes []byte) error {
	utils.Go(ctx, func() {
		select {
		case <-ctx.Done():
			return
		default:
			sva.analyze(ctx, wavBytes)
		}
	})
	return nil
}

func (sva *silerovadAnalyzer) analyze(
	ctx context.Context,
	wavBytes []byte,
) ([]byte, error) {

	rnnoiseConfig := &internal_voices.AudioConfig{
		SampleRate: 48000,
		Format:     internal_voices.Linear16,
		Channels:   1,
	}

	float32Samples48k, err := sva.resampler.ConvertToFloat32WithResample(wavBytes, sva.inputConfig, 48000)
	if err != nil {
		sva.logger.Error("Error converting to 48kHz float32 for RNNoise: %v", err)
		return wavBytes, err
	}

	ds, err := internal_voice_rnnoise.NewRnnoiseDenoiser()
	if err != nil {
		sva.logger.Error("Error initializing RNNoise: %v", err)
		return wavBytes, err
	}
	const RNNOISE_FRAME_SIZE = 480
	denoisedSamples := make([]float32, 0, len(float32Samples48k))
	totalConfidence := 0.0
	frameCount := 0
	for i := 0; i < len(float32Samples48k); i += RNNOISE_FRAME_SIZE {
		end := i + RNNOISE_FRAME_SIZE
		if end > len(float32Samples48k) {
			frame := make([]float32, RNNOISE_FRAME_SIZE)
			copy(frame, float32Samples48k[i:])
			confidence, denoisedFrame, err := ds.Denoise(frame)
			if err != nil {
				sva.logger.Errorf("error while processing RNNoise frame %d: %v", frameCount, err)
				denoisedSamples = append(denoisedSamples, float32Samples48k[i:]...)
			} else {
				totalConfidence += confidence
				frameCount++
				validSamples := len(float32Samples48k) - i
				denoisedSamples = append(denoisedSamples, denoisedFrame[:validSamples]...)
			}
			break
		} else {
			frame := float32Samples48k[i:end]
			confidence, denoisedFrame, err := ds.Denoise(frame)
			if err != nil {
				sva.logger.Errorf("error while processing RNNoise frame %d: %v", frameCount, err)
				denoisedSamples = append(denoisedSamples, frame...)
			} else {
				totalConfidence += confidence
				frameCount++
				denoisedSamples = append(denoisedSamples, denoisedFrame...)
			}
		}
	}

	denoisedBytes48k, err := sva.resampler.ConvertFromFloat32(denoisedSamples, rnnoiseConfig)
	if err != nil {
		sva.logger.Errorf("error converting denoised samples to bytes: %v", err)
		return wavBytes, err
	}

	float32Samples16k, err := sva.resampler.ConvertToFloat32WithResample(denoisedBytes48k, rnnoiseConfig, 16000)
	if err != nil {
		sva.logger.Errorf("error converting to 16kHz float32 for Silero: %v", err)
		return wavBytes, err
	}

	segments, err := sva.detector.Detect(float32Samples16k)
	if err != nil {
		sva.logger.Errorf("error while detecting segments: %v", err)
		return wavBytes, nil
	}

	for _, segment := range segments {
		sva.logger.Debugf("interrupt :vad interrupted with segments %+v", segment)
		sva.opts.OnAnalyze(ctx, &segment)
	}
	return wavBytes, nil
}
