// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_firered_vad

// framesPerSecond is the number of fbank frames per second (1000 / frameShiftMs).
const framesPerSecond = 1000 / frameShiftMs // 100

// vadState represents the state machine states for streaming VAD.
type vadState int

const (
	stateSilence         vadState = 0
	statePossibleSpeech  vadState = 1
	stateSpeech          vadState = 2
	statePossibleSilence vadState = 3
)

// StreamVadFrameResult holds the result of processing a single frame.
type StreamVadFrameResult struct {
	FrameIdx         int
	IsSpeech         bool
	RawProb          float32
	SmoothedProb     float32
	IsSpeechStart    bool
	IsSpeechEnd      bool
	SpeechStartFrame int // 1-based, -1 if not applicable
	SpeechEndFrame   int // 1-based, -1 if not applicable
}

// PostprocessorConfig holds configuration for the stream VAD postprocessor.
type PostprocessorConfig struct {
	SmoothWindowSize int
	SpeechThreshold  float32
	MinSpeechFrame   int
	MaxSpeechFrame   int
	MinSilenceFrame  int
}

// DefaultPostprocessorConfig returns the default configuration exposed by the
// provider options.
func DefaultPostprocessorConfig() PostprocessorConfig {
	return PostprocessorConfig{
		SmoothWindowSize: 5,
		SpeechThreshold:  defaultConfidence,
		MinSpeechFrame:   vadDurationFrames(defaultStartSecs),
		MaxSpeechFrame:   2000,
		MinSilenceFrame:  vadDurationFrames(defaultStopSecs),
	}
}

// Postprocessor implements the FireRedVAD stream VAD state machine
// that converts per-frame speech probabilities into speech start/end events.
type Postprocessor struct {
	cfg PostprocessorConfig

	// Smooth window (ring buffer)
	smoothWindow    []float32
	smoothWindowSum float64
	smoothWindowIdx int
	smoothWindowLen int

	// State machine
	frameCnt             int
	state                vadState
	speechCnt            int
	silenceCnt           int
	hitMaxSpeech         bool
	lastSpeechStartFrame int
	lastSpeechEndFrame   int
}

// NewPostprocessor creates a new postprocessor with the given config.
func NewPostprocessor(cfg PostprocessorConfig) *Postprocessor {
	if cfg.SmoothWindowSize < 1 {
		cfg.SmoothWindowSize = 1
	}
	return &Postprocessor{
		cfg:                  cfg,
		smoothWindow:         make([]float32, 0, cfg.SmoothWindowSize),
		lastSpeechStartFrame: -1,
		lastSpeechEndFrame:   -1,
	}
}

// InSpeech returns true when the state machine has confirmed an active speech
// segment — i.e. the state is stateSpeech or statePossibleSilence. Frames in
// statePossibleSpeech (not yet past MinSpeechFrame) are treated as unconfirmed
// and should not trigger interruptions or heartbeats.
func (p *Postprocessor) InSpeech() bool {
	return p.state == stateSpeech || p.state == statePossibleSilence
}

// Reset clears all state for reuse with a new audio stream.
func (p *Postprocessor) Reset() {
	p.frameCnt = 0
	p.smoothWindow = p.smoothWindow[:0]
	p.smoothWindowSum = 0
	p.smoothWindowIdx = 0
	p.smoothWindowLen = 0
	p.state = stateSilence
	p.speechCnt = 0
	p.silenceCnt = 0
	p.hitMaxSpeech = false
	p.lastSpeechStartFrame = -1
	p.lastSpeechEndFrame = -1
}

// ProcessFrame processes a single frame's raw probability and returns
// the frame result with speech start/end events.
func (p *Postprocessor) ProcessFrame(rawProb float32) StreamVadFrameResult {
	p.frameCnt++

	smoothedProb := p.smoothProb(rawProb)
	isSpeech := smoothedProb >= p.cfg.SpeechThreshold

	result := StreamVadFrameResult{
		FrameIdx:         p.frameCnt,
		IsSpeech:         isSpeech,
		RawProb:          rawProb,
		SmoothedProb:     smoothedProb,
		SpeechStartFrame: -1,
		SpeechEndFrame:   -1,
	}

	p.stateTransition(isSpeech, &result)
	return result
}

// smoothProb applies a moving average smoothing window to the probability.
func (p *Postprocessor) smoothProb(prob float32) float32 {
	if p.cfg.SmoothWindowSize <= 1 {
		return prob
	}

	if p.smoothWindowLen < p.cfg.SmoothWindowSize {
		p.smoothWindow = append(p.smoothWindow, prob)
		p.smoothWindowLen++
		p.smoothWindowSum += float64(prob)
	} else {
		p.smoothWindowSum -= float64(p.smoothWindow[p.smoothWindowIdx])
		p.smoothWindow[p.smoothWindowIdx] = prob
		p.smoothWindowSum += float64(prob)
		p.smoothWindowIdx = (p.smoothWindowIdx + 1) % p.cfg.SmoothWindowSize
	}

	return float32(p.smoothWindowSum / float64(p.smoothWindowLen))
}

// stateTransition implements the 4-state machine from FireRedVAD.
func (p *Postprocessor) stateTransition(isSpeech bool, result *StreamVadFrameResult) {
	if p.hitMaxSpeech {
		result.IsSpeechStart = true
		result.SpeechStartFrame = p.frameCnt
		p.lastSpeechStartFrame = result.SpeechStartFrame
		p.hitMaxSpeech = false
	}

	switch p.state {
	case stateSilence:
		if isSpeech {
			p.state = statePossibleSpeech
			p.speechCnt++
		} else {
			p.silenceCnt++
			p.speechCnt = 0
		}

	case statePossibleSpeech:
		if isSpeech {
			p.speechCnt++
			if p.speechCnt >= p.cfg.MinSpeechFrame {
				p.state = stateSpeech
				result.IsSpeechStart = true
				startFrame := p.frameCnt - p.speechCnt + 1
				if startFrame < 1 {
					startFrame = 1
				}
				if p.lastSpeechEndFrame+1 > startFrame {
					startFrame = p.lastSpeechEndFrame + 1
				}
				result.SpeechStartFrame = startFrame
				p.lastSpeechStartFrame = result.SpeechStartFrame
				p.silenceCnt = 0
			}
		} else {
			p.state = stateSilence
			p.silenceCnt = 1
			p.speechCnt = 0
		}

	case stateSpeech:
		p.speechCnt++
		if isSpeech {
			p.silenceCnt = 0
			if p.speechCnt >= p.cfg.MaxSpeechFrame {
				p.hitMaxSpeech = true
				p.speechCnt = 0
				result.IsSpeechEnd = true
				result.SpeechEndFrame = p.frameCnt
				result.SpeechStartFrame = p.lastSpeechStartFrame
				p.lastSpeechStartFrame = -1
				p.lastSpeechEndFrame = result.SpeechEndFrame
			}
		} else {
			p.state = statePossibleSilence
			p.silenceCnt++
		}

	case statePossibleSilence:
		p.speechCnt++
		if isSpeech {
			p.state = stateSpeech
			p.silenceCnt = 0
			if p.speechCnt >= p.cfg.MaxSpeechFrame {
				p.hitMaxSpeech = true
				p.speechCnt = 0
				result.IsSpeechEnd = true
				result.SpeechEndFrame = p.frameCnt
				result.SpeechStartFrame = p.lastSpeechStartFrame
				p.lastSpeechStartFrame = -1
				p.lastSpeechEndFrame = result.SpeechEndFrame
			}
		} else {
			p.silenceCnt++
			if p.silenceCnt >= p.cfg.MinSilenceFrame {
				p.state = stateSilence
				result.IsSpeechEnd = true
				result.SpeechEndFrame = p.frameCnt
				result.SpeechStartFrame = p.lastSpeechStartFrame
				p.lastSpeechEndFrame = result.SpeechEndFrame
				p.lastSpeechStartFrame = -1
				p.speechCnt = 0
			}
		}
	}
}
