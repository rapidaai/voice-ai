package internal_denoiser_factory

import (
	internal_denoiser "github.com/rapidaai/api/assistant-api/internal/denoiser"
	internal_denoiser_krisp "github.com/rapidaai/api/assistant-api/internal/denoiser/krisp"
	internal_denoiser_rnnoise "github.com/rapidaai/api/assistant-api/internal/denoiser/rn_noise"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type DenoiserIdentifier string

const (
	RN_NOISE DenoiserIdentifier = "rn_noise"
	KRISP    DenoiserIdentifier = "krisp"
)

// logger, audioConfig, opts
func GetDenoiser(aa DenoiserIdentifier, logger commons.Logger, inCfg *protos.AudioConfig, options utils.Option) (internal_denoiser.Denoiser, error) {
	switch aa {
	case KRISP:
		return internal_denoiser_krisp.NewKrispDenoiser(logger, inCfg, options)
	default:
		return internal_denoiser_rnnoise.NewRnnoiseDenoiser(logger, inCfg, options)
	}
}
