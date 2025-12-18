package internal_audio

type AudioFormat int

const (
	Linear16 AudioFormat = iota
	MuLaw8
)

func (af AudioFormat) Name() string {
	switch af {
	case Linear16:
		return "Linear16"
	case MuLaw8:
		return "MuLaw8"
	default:
		return "Unknown"
	}
}

// AudioConfig holds audio configuration
type AudioConfig struct {
	SampleRate int
	Format     AudioFormat
	Channels   int // typically 1 for mono, 2 for stereo
}

func (ac *AudioConfig) GetSampleRate() int {
	return ac.SampleRate
}

func (ac *AudioConfig) GetFormat() string {
	return ac.Format.Name()
}

func (ac *AudioConfig) IsMono() bool {
	return ac.Channels == 1
}

func NewMulaw8khzMonoAudioConfig() *AudioConfig {
	return &AudioConfig{
		SampleRate: 8000,
		Format:     MuLaw8,
		Channels:   1,
	}
}

func NewLinear24khzMonoAudioConfig() *AudioConfig {
	return &AudioConfig{
		SampleRate: 24000,
		Format:     Linear16,
		Channels:   1,
	}
}

func NewLinear16khzMonoAudioConfig() *AudioConfig {
	return &AudioConfig{
		SampleRate: 16000,
		Format:     Linear16,
		Channels:   1,
	}
}

func NewLinear8khzMonoAudioConfig() *AudioConfig {
	return &AudioConfig{
		SampleRate: 8000,
		Format:     Linear16,
		Channels:   1,
	}
}
