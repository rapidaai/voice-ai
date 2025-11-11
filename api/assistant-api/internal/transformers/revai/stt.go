// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_revai

import (
	"context"

	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

func NewRevaiSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	opts *internal_transformers.SpeechToTextInitializeOptions) (internal_transformers.SpeechToTextTransformer, error) {
	return nil, nil
}
