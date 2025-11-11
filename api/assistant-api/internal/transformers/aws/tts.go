// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_aws

import (
	"context"

	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type awsSpeechToTextTransformer struct {
	cfg    *config.AppConfig
	logger commons.Logger
}

func NewAWSTextToSpeech(ctx context.Context, logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	opts *internal_transformers.TextToSpeechInitializeOptions) (internal_transformers.TextToSpeechTransformer, error) {
	return nil, nil
	// return &awsSpeechToTextTransformer{
	// 	cfg:    cfg,
	// 	logger: logger,
	// }
}

// a implements internal_transformers.SpeechToTextTransformer.
func (*awsSpeechToTextTransformer) Name() string {
	panic("unimplemented")
}

// trf implements internal_transformers.SpeechToTextTransformer.
func (*awsSpeechToTextTransformer) Transform(a []byte) (string, error) {
	panic("unimplemented")
}
