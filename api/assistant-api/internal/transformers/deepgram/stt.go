// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_deepgram

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type deepgramSTT struct {
	logger            commons.Logger
	client            *client.WSCallback
	providerOption    DeepgramOption
	transformerOption *internal_transformers.SpeechToTextInitializeOptions
}

// Name implements internal_transformers.SpeechToTextTransformer.
func (*deepgramSTT) Name() string {
	return "deepgram-speech-to-text"
}

func NewDeepgramSpeechToText(ctx context.Context,
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	opts *internal_transformers.SpeechToTextInitializeOptions,
) (internal_transformers.SpeechToTextTransformer, error) {
	start := time.Now()
	//create deepgram option
	dGoptions, err := NewDeepgramOption(
		logger,
		vaultCredential,
		opts.AudioConfig,
		opts.ModelOptions,
	)
	if err != nil {
		logger.Errorf("Key from credential failed %+v", err)
		return nil, err
	}

	dgClient, err := client.NewWSUsingCallback(
		context.Background(),
		dGoptions.GetKey(),
		&interfaces.ClientOptions{
			APIKey:          dGoptions.GetKey(),
			EnableKeepAlive: true,
		},
		dGoptions.SpeechToTextOptions(),
		NewDeepgramSttCallback(logger, opts.OnTranscript))

	//
	if err != nil {
		logger.Benchmark("deepgram.NewDeepgramSpeechToText", time.Since(start))
		logger.Errorf("unable create dg client with error %+v", err.Error())
		return nil, err
	}

	//
	logger.Benchmark("deepgram.NewDeepgramSpeechToText", time.Since(start))
	return &deepgramSTT{
		client:            dgClient,
		logger:            logger,
		providerOption:    dGoptions,
		transformerOption: opts,
	}, nil
}

// The `Initialize` method in the `deepgram` struct is responsible for establishing a connection to the
// Deepgram service using the WebSocket client `dg.client`.
func (dg *deepgramSTT) Initialize() error {
	start := time.Now()
	if !dg.client.Connect() {
		return errors.New("unable to connect with deepgram")
	}
	dg.logger.Benchmark("deepgram.Initialize", time.Since(start))
	return nil
}

// Transform implements internal_transformers.SpeechToTextTransformer.
// The `Transform` method in the `deepgram` struct is taking an input audio byte array `in`, creating a
// new `bufio.Reader` from it, and then passing that reader to the `Stream` method of the `dg.client`
// WebSocket client. This method is responsible for streaming the audio data to the Deepgram service
// for transcription. If there are any errors during the streaming process, they will be returned by
// the method.
func (dg *deepgramSTT) Transform(ctx context.Context, in []byte, opts *internal_transformers.SpeechToTextOption) error {
	err := dg.client.Stream(bufio.NewReader(bytes.NewReader(in)))
	if err != nil {
		if err.Error() == "EOF" {
			return nil
		}
		dg.logger.Errorf("error while calling deepgram: %v", err)
		return fmt.Errorf("deepgram stream error: %w", err)
	}
	return err
}

func (dg *deepgramSTT) Close(ctx context.Context) error {
	dg.client.Stop()
	return nil
}
