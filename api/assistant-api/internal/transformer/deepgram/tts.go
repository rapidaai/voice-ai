// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_deepgram

import (
	"context"
	"fmt"
	"sync"

	"github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/speak"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	internal_transformer_deepgram_internal "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram/internal"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
)

type DeepgramSpeaking interface {
	Speak(string) error
	Flush() error
	Reset() error
	Connect() bool
}

type deepgramTTS struct {
	*deepgramOption
	ctx       context.Context
	mu        sync.Mutex
	contextId string
	logger    commons.Logger
	client    DeepgramSpeaking
	options   *internal_transformer.TextToSpeechInitializeOptions
}

func NewDeepgramTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	opts *internal_transformer.TextToSpeechInitializeOptions) (internal_transformer.TextToSpeechTransformer, error) {

	//create deepgram option
	dGoptions, err := NewDeepgramOption(
		logger,
		credential,
		opts.AudioConfig,
		opts.ModelOptions,
	)
	if err != nil {
		logger.Errorf("deepgram-tts: error while intializing deepgram text to speech")
		return nil, err
	}
	return &deepgramTTS{
		ctx:            ctx,
		logger:         logger,
		options:        opts,
		deepgramOption: dGoptions,
	}, nil
}

func (dg *deepgramTTS) Name() string {
	return "deepgram-text-to-speech"
}

// Deepgram service using the WebSocket client `dg.client`.
func (dg *deepgramTTS) Initialize() error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	client, err := client.NewWSUsingCallback(dg.ctx,
		dg.GetKey(),
		&interfaces.ClientOptions{
			APIKey:          dg.GetKey(),
			EnableKeepAlive: true,
		},
		dg.TextToSpeechOptions(),
		internal_transformer_deepgram_internal.NewDeepgramSpeakCallback(dg.logger, dg.onspeech, dg.oncomplete),
	)

	if err != nil {
		dg.logger.Errorf("deepgram-tts: unable create dg client with error %+v", err.Error())
		return err
	}

	if !client.Connect() {
		dg.logger.Errorf("deepgram-tts: unable to connect to deepgram service")
		return fmt.Errorf("deepgram-tts: connection failed")
	}
	dg.client = client
	dg.logger.Debugf("deepgram-tts: connection established")
	return nil
}

func (dg *deepgramTTS) onspeech(b []byte) error {
	return dg.options.OnSpeech(dg.contextId, b)
}

func (dg *deepgramTTS) oncomplete() error {
	return dg.options.OnComplete(dg.contextId)
}

func (dg *deepgramTTS) Transform(
	ctx context.Context,
	sentence string,
	opts *internal_transformer.TextToSpeechOption) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if dg.client == nil {
		return fmt.Errorf("deepgram-tts: connection is not initialized")
	}
	dg.contextId = opts.ContextId
	err := dg.client.Speak(sentence)
	if err != nil {
		dg.logger.Errorf("deepgram-tts: unable to speak with error: %v", err)
		return nil
	}
	if opts.IsComplete {
		dg.client.Flush()
	}
	return nil

}

func (dg *deepgramTTS) Close(ctx context.Context) error {
	if dg.client != nil {
		dg.client.Reset()
	}
	return nil
}
