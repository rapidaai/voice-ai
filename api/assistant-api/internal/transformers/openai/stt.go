// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_openai

import (
	"context"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
)

type openaiSpeechToText struct {
	logger commons.Logger
	client openai.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func (o *openaiSpeechToText) Initialize() error {
	o.ctx, o.cancel = context.WithCancel(context.Background())
	o.client = openai.NewClient(option.WithAPIKey("YOUR_API_KEY"))
	return nil
}

func (o *openaiSpeechToText) Close(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	o.logger.Infof("OpenAI SpeechToText connection closed.")
	return nil
}

func (o *openaiSpeechToText) Name() string {
	return "openai-speech-to-text"
}

// Transform receives a stream of bytes (audioStream) and prints transcribed text in realtime.
func (o *openaiSpeechToText) Transform(ctx context.Context,
	byt []byte,
	opt *internal_transformers.SpeechToTextOption) error {
	return nil
}

func NewOpenaiSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	opts internal_transformers.SpeechToTextTransformer,
) (internal_transformers.SpeechToTextTransformer, error) {
	stt := &openaiSpeechToText{
		logger: logger,
	}
	return stt, nil
}
