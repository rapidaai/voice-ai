// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_google

import (
	"context"
	"io"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type googleSpeechToText struct {
	logger             commons.Logger
	client             *speech.Client
	stream             speechpb.Speech_StreamingRecognizeClient
	providerOptions    GoogleOption
	transformerOptions *internal_transformers.SpeechToTextInitializeOptions
	ctx                context.Context
	cancel             context.CancelFunc
}

// Name implements internal_transformers.SpeechToTextTransformer.
func (g *googleSpeechToText) Name() string {
	return "google-speech-to-text"
}

func (g *googleSpeechToText) Initialize() error {
	err := g.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: g.providerOptions.SpeechToTextOptions(),
		},
	})
	if err != nil {
		return err
	}

	// Launch callback listener
	go g.SpeechToTextCallback(g.ctx)
	return nil
}

func (g *googleSpeechToText) Close(ctx context.Context) error {
	if g.cancel != nil {
		g.cancel()
	}
	return g.client.Close()
}

// Transform implements internal_transformers.SpeechToTextTransformer.
func (g *googleSpeechToText) Transform(c context.Context, byf []byte, opts *internal_transformers.SpeechToTextOption) error {
	return g.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
			AudioContent: byf,
		},
	})
}

// SpeechToTextCallback processes streaming responses with context awareness.
func (g *googleSpeechToText) SpeechToTextCallback(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.logger.Infof("Google STT: context cancelled, stopping response listener")
			return
		default:
			resp, err := g.stream.Recv()
			if err == io.EOF {
				g.logger.Infof("Google STT: stream ended (EOF)")
				return
			}
			if err != nil {
				g.logger.Errorf("Google STT: recv error: %v", err)
				return
			}
			if resp == nil {
				g.logger.Warnf("Google STT: received nil response")
				return
			}
			if resp.Error != nil {
				switch resp.Error.Code {
				case 3, 11:
					g.logger.Warnf("Google STT: stream duration limit reached (code=%d): %s", resp.Error.Code, resp.Error.Message)
				default:
					g.logger.Errorf("Google STT: recognition error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
				}
				return
			}
			for _, result := range resp.Results {
				if len(result.Alternatives) == 0 {
					continue
				}
				alt := result.Alternatives[0]
				if g.transformerOptions.OnTranscript != nil {
					g.transformerOptions.OnTranscript(
						alt.GetTranscript(),
						float64(alt.GetConfidence()),
						result.GetLanguageCode(),
						result.GetIsFinal())
				}
			}
		}
	}
}

// NewGoogleSpeechToText initializes the transformer with context and stream.
func NewGoogleSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	opts *internal_transformers.SpeechToTextInitializeOptions,
) (internal_transformers.SpeechToTextTransformer, error) {
	start := time.Now()
	cOptions, err := NewGoogleOption(logger, credential, opts.AudioConfig, opts.ModelOptions)
	if err != nil {
		logger.Errorf("intializing google failed %+v", err)
		return nil, err
	}

	//
	client, err := speech.NewClient(ctx,
		cOptions.GetClientOptions()...,
	)
	if err != nil {
		logger.Errorf("error creating Google STT client: %+v", err)
		return nil, err
	}

	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		logger.Errorf("error creating Google STT stream: %+v", err)
		return nil, err
	}

	// Context for callback management
	sttCtx, cancel := context.WithCancel(ctx)
	logger.Benchmark("google.NewGoogleSpeechToText", time.Since(start))
	return &googleSpeechToText{
		logger:             logger,
		client:             client,
		stream:             stream,
		providerOptions:    cOptions,
		transformerOptions: opts,
		ctx:                sttCtx,
		cancel:             cancel,
	}, nil
}
