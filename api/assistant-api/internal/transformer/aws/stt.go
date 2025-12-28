// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_transformer_aws

import (
	"context"

	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

func NewAWSSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	opts *internal_transformer.SpeechToTextInitializeOptions) (internal_transformer.SpeechToTextTransformer, error) {
	return nil, nil
}
