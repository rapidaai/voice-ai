package internal_denoiser_krisp

import (
	"context"

	internal_denoiser "github.com/rapidaai/api/assistant-api/internal/denoiser"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type krispDenoiser struct {
	logger commons.Logger
}

func NewKrispDenoiser(logger commons.Logger, inCfg *protos.AudioConfig, options utils.Option) (internal_denoiser.Denoiser, error) {
	return &krispDenoiser{logger: logger}, nil
}

func (krisp *krispDenoiser) Denoise(ctx context.Context, input []byte) ([]byte, float64, error) {
	panic("not yet implimented")
}

func (krisp *krispDenoiser) Flush() {
	panic("not yet implimented")
}
