package internal_voice_silero_vad

// #cgo CFLAGS: -Wall -Werror -std=c99
// #cgo LDFLAGS:
// #include "ort_bridge.h"
import "C"

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
)

const (
	stateLen   = 2 * 1 * 128
	contextLen = 64
)

type LogLevel int

func (l LogLevel) OrtLoggingLevel() C.OrtLoggingLevel {
	switch l {
	case LevelVerbose:
		return C.ORT_LOGGING_LEVEL_VERBOSE
	case LogLevelInfo:
		return C.ORT_LOGGING_LEVEL_INFO
	case LogLevelWarn:
		return C.ORT_LOGGING_LEVEL_WARNING
	case LogLevelError:
		return C.ORT_LOGGING_LEVEL_ERROR
	case LogLevelFatal:
		return C.ORT_LOGGING_LEVEL_FATAL
	default:
		return C.ORT_LOGGING_LEVEL_WARNING
	}
}

const (
	LevelVerbose LogLevel = iota + 1
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

type DetectorConfig struct {
	// The path to the ONNX Silero VAD model file to load.
	ModelPath string
	// The sampling rate of the input audio samples. Supported values are 8000 and 16000.
	SampleRate int
	// The probability threshold above which we detect speech. A good default is 0.5.
	Threshold float32
	// The duration of silence to wait for each speech segment before separating it.
	MinSilenceDurationMs int
	// The padding to add to speech segments to avoid aggressive cutting.
	SpeechPadMs int
	// The loglevel for the onnx environment, by default it is set to LogLevelWarn.
	LogLevel LogLevel
}

type sileroDetector struct {
	api         *C.OrtApi
	env         *C.OrtEnv
	sessionOpts *C.OrtSessionOptions
	session     *C.OrtSession
	memoryInfo  *C.OrtMemoryInfo
	cStrings    map[string]*C.char

	cfg DetectorConfig

	state [stateLen]float32
	ctx   [contextLen]float32

	currSample int
	triggered  bool
	tempEnd    int
}

func NewSileroDetector(logger commons.Logger, cfg *DetectorConfig) (voices.Detector, error) {
	if cfg == nil {
		cfg = &DetectorConfig{}
	}
	envModelPath := os.Getenv("SILERO_MODEL_PATH")
	if envModelPath != "" {
		cfg.ModelPath = envModelPath
	} else {
		_, path, _, _ := runtime.Caller(0)
		cfg.ModelPath = filepath.Join(filepath.Dir(path), "models/silero_vad_20251001.onnx")
	}

	if cfg.SampleRate != 8000 && cfg.SampleRate != 16000 {
		cfg.SampleRate = 16000
	}

	if cfg.Threshold <= 0 || cfg.Threshold >= 1 {
		cfg.Threshold = 0.5
	}

	if cfg.MinSilenceDurationMs < 0 {
		cfg.MinSilenceDurationMs = 100
	}

	if cfg.SpeechPadMs < 0 {
		cfg.SpeechPadMs = 30
	}
	logger.Debugf("model config = %+v", cfg)
	sd := sileroDetector{
		cfg:      *cfg,
		cStrings: map[string]*C.char{},
	}

	sd.api = C.OrtGetApi()
	if sd.api == nil {
		return nil, fmt.Errorf("failed to get API")
	}

	sd.cStrings["loggerName"] = C.CString("vad")
	status := C.OrtApiCreateEnv(sd.api, cfg.LogLevel.OrtLoggingLevel(), sd.cStrings["loggerName"], &sd.env)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to create env: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiCreateSessionOptions(sd.api, &sd.sessionOpts)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to create session options: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiSetIntraOpNumThreads(sd.api, sd.sessionOpts, 1)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to set intra threads: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiSetInterOpNumThreads(sd.api, sd.sessionOpts, 1)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to set inter threads: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiSetSessionGraphOptimizationLevel(sd.api, sd.sessionOpts, C.ORT_ENABLE_ALL)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to set session graph optimization level: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	sd.cStrings["modelPath"] = C.CString(sd.cfg.ModelPath)
	status = C.OrtApiCreateSession(sd.api, sd.env, sd.cStrings["modelPath"], sd.sessionOpts, &sd.session)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to create session: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	status = C.OrtApiCreateCpuMemoryInfo(sd.api, C.OrtArenaAllocator, C.OrtMemTypeDefault, &sd.memoryInfo)
	defer C.OrtApiReleaseStatus(sd.api, status)
	if status != nil {
		return nil, fmt.Errorf("failed to create memory info: %s", C.GoString(C.OrtApiGetErrorMessage(sd.api, status)))
	}

	sd.cStrings["input"] = C.CString("input")
	sd.cStrings["sr"] = C.CString("sr")
	sd.cStrings["state"] = C.CString("state")
	sd.cStrings["stateN"] = C.CString("stateN")
	sd.cStrings["output"] = C.CString("output")

	return &sd, nil
}

// type Segment struct {
// 	// The relative timestamp in seconds of when a speech segment begins.
// 	SpeechStartAt float64
// 	// The relative timestamp in seconds of when a speech segment ends.
// 	SpeechEndAt float64
// 	// The duration of the speech segment in seconds.
// 	Duration float64
// 	// The energy level of the speech segment, used to filter out noise.
// 	Energy float64
// 	// A confidence score indicating the likelihood of valid speech (0 to 1).
// 	Confidence float32
// }

func (sd *sileroDetector) Detect(pcm []float32) ([]voices.DetectorVoiceSegment, error) {
	if sd == nil {
		return nil, fmt.Errorf("invalid nil detector")
	}

	windowSize := 512
	if sd.cfg.SampleRate == 8000 {
		windowSize = 256
	}

	if len(pcm) < windowSize {
		return nil, fmt.Errorf("not enough samples")
	}

	slog.Debug("starting speech detection", slog.Int("samplesLen", len(pcm)))

	minSilenceSamples := sd.cfg.MinSilenceDurationMs * sd.cfg.SampleRate / 1000
	speechPadSamples := sd.cfg.SpeechPadMs * sd.cfg.SampleRate / 1000

	var segments []voices.DetectorVoiceSegment
	var energySum float64
	for i := 0; i < len(pcm)-windowSize; i += windowSize {
		speechProb, err := sd.infer(pcm[i : i+windowSize])
		if err != nil {
			return nil, fmt.Errorf("infer failed: %w", err)
		}

		// Calculate energy of the current window
		windowEnergy := float64(0)
		for _, sample := range pcm[i : i+windowSize] {
			windowEnergy += float64(sample * sample)
		}
		energySum += windowEnergy

		sd.currSample += windowSize

		if speechProb >= sd.cfg.Threshold && sd.tempEnd != 0 {
			sd.tempEnd = 0
		}

		if speechProb >= sd.cfg.Threshold && !sd.triggered {
			sd.triggered = true
			speechStartAt := (float64(sd.currSample-windowSize-speechPadSamples) / float64(sd.cfg.SampleRate))

			// We clamp at zero since due to padding the starting position could be negative.
			if speechStartAt < 0 {
				speechStartAt = 0
			}

			slog.Debug("speech start", slog.Float64("startAt", speechStartAt))
			segments = append(segments, voices.DetectorVoiceSegment{
				SpeechStartAt: speechStartAt,
				Confidence:    speechProb,   // Initial confidence
				Energy:        windowEnergy, // Initial energy
			})
		}

		if speechProb < (sd.cfg.Threshold-0.15) && sd.triggered {
			if sd.tempEnd == 0 {
				sd.tempEnd = sd.currSample
			}

			// Not enough silence yet to split, we continue.
			if sd.currSample-sd.tempEnd < minSilenceSamples {
				continue
			}

			// Finalize the speech segment
			speechEndAt := (float64(sd.tempEnd+speechPadSamples) / float64(sd.cfg.SampleRate))
			sd.tempEnd = 0
			sd.triggered = false
			slog.Debug("speech end", slog.Float64("endAt", speechEndAt))

			if len(segments) < 1 {
				return nil, fmt.Errorf("unexpected speech end")
			}

			// Update the last segment with the end time and calculate duration.
			lastSegment := &segments[len(segments)-1]
			lastSegment.SpeechEndAt = speechEndAt
			lastSegment.Duration = speechEndAt - lastSegment.SpeechStartAt
			lastSegment.Energy = energySum / lastSegment.Duration              // Average energy
			lastSegment.Confidence = (lastSegment.Confidence + speechProb) / 2 // Average confidence
			energySum = 0                                                      // Reset energy for the next segment
		}
	}

	slog.Debug("speech detection done", slog.Int("segmentsLen", len(segments)))

	return segments, nil
}

func (sd *sileroDetector) Flush() error {
	if sd == nil {
		return fmt.Errorf("invalid nil detector")
	}

	sd.currSample = 0
	sd.triggered = false
	sd.tempEnd = 0
	for i := 0; i < stateLen; i++ {
		sd.state[i] = 0
	}
	for i := 0; i < contextLen; i++ {
		sd.ctx[i] = 0
	}
	return nil
}

func (sd *sileroDetector) Close() error {
	if sd == nil {
		return fmt.Errorf("invalid nil detector")
	}

	C.OrtApiReleaseMemoryInfo(sd.api, sd.memoryInfo)
	C.OrtApiReleaseSession(sd.api, sd.session)
	C.OrtApiReleaseSessionOptions(sd.api, sd.sessionOpts)
	C.OrtApiReleaseEnv(sd.api, sd.env)
	for _, ptr := range sd.cStrings {
		C.free(unsafe.Pointer(ptr))
	}

	return nil
}
