// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_transformer_aws

import (
	"context"

	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type awsSpeechToTextTransformer struct {
	cfg    *config.AppConfig
	logger commons.Logger
}

func NewAWSTextToSpeech(ctx context.Context, logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	opts *internal_transformer.TextToSpeechInitializeOptions) (internal_transformer.TextToSpeechTransformer, error) {
	return nil, nil
	// return &awsSpeechToTextTransformer{
	// 	cfg:    cfg,
	// 	logger: logger,
	// }
}

// a implements internal_transformer.SpeechToTextTransformer.
func (*awsSpeechToTextTransformer) Name() string {
	panic("unimplemented")
}

// trf implements internal_transformer.SpeechToTextTransformer.
func (*awsSpeechToTextTransformer) Transform(a []byte) (string, error) {
	panic("unimplemented")
}
