package internal_voices

type DetectorVoiceSegment struct {
	// The relative timestamp in seconds of when a speech segment begins.
	SpeechStartAt float64
	// The relative timestamp in seconds of when a speech segment ends.
	SpeechEndAt float64
	// The duration of the speech segment in seconds.
	Duration float64
	// The energy level of the speech segment, used to filter out noise.
	Energy float64
	// A confidence score indicating the likelihood of valid speech (0 to 1).
	Confidence float32
}

// GetSpeechStartAt returns the relative timestamp in seconds of when a speech segment begins.
func (d *DetectorVoiceSegment) GetSpeechStartAt() float64 {
	return d.SpeechStartAt
}

// GetSpeechEndAt returns the relative timestamp in seconds of when a speech segment ends.
func (d *DetectorVoiceSegment) GetSpeechEndAt() float64 {
	return d.SpeechEndAt
}

// GetDuration returns the duration of the speech segment in seconds.
func (d *DetectorVoiceSegment) GetDuration() float64 {
	return d.Duration
}

// GetEnergy returns the energy level of the speech segment, used to filter out noise.
func (d *DetectorVoiceSegment) GetEnergy() float64 {
	return d.Energy
}

// GetConfidence returns a confidence score indicating the likelihood of valid speech (0 to 1).
func (d *DetectorVoiceSegment) GetConfidence() float32 {
	return d.Confidence
}

type Detector interface {
	Detect(pcm []float32) ([]DetectorVoiceSegment, error)
}
