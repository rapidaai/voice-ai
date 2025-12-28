package internal_vad_factory

import (
	internal_vad "github.com/rapidaai/api/assistant-api/internal/vad"
	internal_vad_silero "github.com/rapidaai/api/assistant-api/internal/vad/silero_vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type VADIdentifier string

const (
	SILERO_VAD VADIdentifier = "silero_vad"
	TEN_VAD    VADIdentifier = "ten_vad"
)

// logger, audioConfig, opts
func GetVAD(aa VADIdentifier, logger commons.Logger, intputAudio *protos.AudioConfig,
	callback internal_vad.VADCallback,
	options utils.Option) (internal_vad.Vad, error) {
	switch aa {
	case SILERO_VAD:
		return internal_vad_silero.NewSileroVAD(logger, intputAudio, callback, options)
	case TEN_VAD:
		return internal_vad_silero.NewSileroVAD(logger, intputAudio, callback, options)
	default:
		return internal_vad_silero.NewSileroVAD(logger, intputAudio, callback, options)
	}
}
