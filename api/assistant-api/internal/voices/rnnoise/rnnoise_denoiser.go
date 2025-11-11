package internal_voice_rnnoise

import voices "github.com/rapidaai/api/assistant-api/internal/voices"

type rnnoiseDenoiser struct {
	state *DenoiseState
}

// NewDenoiser creates a new denoiser instance
func NewRnnoiseDenoiser() (voices.Denoiser, error) {
	state, err := Create(nil)
	if err != nil {
		return nil, err
	}
	return &rnnoiseDenoiser{state: state}, nil
}

// ProcessStream processes a continuous audio stream
func (d *rnnoiseDenoiser) Denoise(input []float32) (float64, []float32, error) {
	return d.state.ProcessFrame(input)
}

// Close releases resources
func (d *rnnoiseDenoiser) Flush() {
	if d.state != nil {
		d.state.Destroy()
		d.state = nil
	}
}
