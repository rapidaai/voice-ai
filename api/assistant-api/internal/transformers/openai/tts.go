// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_openai

import (
	"context"

	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
)

func NewOpenaiTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	onSpeech func([]byte) error) (internal_transformers.TextToSpeechTransformer, error) {

	return nil, nil
}
