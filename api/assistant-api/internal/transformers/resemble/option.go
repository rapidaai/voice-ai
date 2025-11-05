// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_resemble

import (
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

const (
	RESEMBLE_URL     = "f.cluster.resemble.ai/stream"
	RESEMBLE_API_KEY = "MeSE5fr4a1yMblzzLYWMzgtt"
	PROJECT_ID       = "665d946d"
	VOICE_ID         = "1dcf0222"
)

type ResembleOption interface {
	GetTextToSpeechRequest(contextId, text string) map[string]interface{}
}

type resembleOption struct {
	logger      commons.Logger
	audioConfig *internal_voices.AudioConfig
	option      utils.Option
}

func NewResembleOption(logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	audioConfig *internal_voices.AudioConfig, option utils.Option) (ResembleOption, error) {
	return &resembleOption{
		logger:      logger,
		audioConfig: audioConfig,
		option:      option,
	}, nil
}

func (ro *resembleOption) GetTextToSpeechFormat(format string) string {
	switch format {
	case "PCM_16", "Linear16":
		return "PCM_16"
	case "MuLaw8", "MULAW":
		return "MULAW"
	}
	return "PCM_16"
}

func (ro *resembleOption) GetTextToSpeechRequest(contextId, text string) map[string]interface{} {
	return map[string]interface{}{
		"voice_uuid":      VOICE_ID,
		"request_id":      contextId,
		"project_uuid":    PROJECT_ID,
		"data":            text,
		"binary_response": true,
		"precision":       ro.GetTextToSpeechFormat(ro.audioConfig.GetFormat()),
		"sample_rate":     ro.audioConfig.GetSampleRate(),
	}

}
