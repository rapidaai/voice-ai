// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_google

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type googleSpeechToText struct {
	*googleOption
	mu      sync.Mutex // Ensures thread-safe operations.
	logger  commons.Logger
	client  *speech.Client
	stream  speechpb.Speech_StreamingRecognizeClient
	options *internal_transformer.SpeechToTextInitializeOptions
	ctx     context.Context
}

// Name implements internal_transformer.SpeechToTextTransformer.
func (g *googleSpeechToText) Name() string {
	return "google-speech-to-text"
}

// Transform implements internal_transformer.SpeechToTextTransformer.
func (google *googleSpeechToText) Transform(c context.Context, byf []byte, opts *internal_transformer.SpeechToTextOption) error {
	google.mu.Lock()
	defer google.mu.Unlock()

	return google.stream.Send(&speechpb.StreamingRecognizeRequest{
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
			g.logger.Infof("google-stt: context cancelled, stopping response listener")
			return
		default:
			resp, err := g.stream.Recv()
			if err == io.EOF {
				g.logger.Infof("google-stt: stream ended (EOF)")
				return
			}
			if err != nil {
				g.logger.Errorf("google-stt: recv error: %v", err)
				return
			}
			if resp == nil {
				g.logger.Warnf("google-stt: received nil response")
				return
			}
			if resp.Error != nil {
				switch resp.Error.Code {
				case 3, 11:
					// reconnect
					g.Initialize()
					g.logger.Warnf("google-stt: stream duration limit reached (code=%d): %s", resp.Error.Code, resp.Error.Message)
				default:
					g.logger.Errorf("google-stt: recognition error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
				}
				return
			}
			for _, result := range resp.Results {
				if len(result.Alternatives) == 0 {
					continue
				}
				alt := result.Alternatives[0]
				if g.options.OnTranscript != nil {
					g.options.OnTranscript(
						alt.GetTranscript(),
						float64(alt.GetConfidence()),
						result.GetLanguageCode(),
						result.GetIsFinal())
				}
			}
		}
	}
}

func NewGoogleSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	opts *internal_transformer.SpeechToTextInitializeOptions,
) (internal_transformer.SpeechToTextTransformer, error) {
	start := time.Now()
	googleOption, err := NewGoogleOption(logger, credential, opts.AudioConfig, opts.ModelOptions)
	if err != nil {
		logger.Errorf("google-stt: Error while GoogleOption err: %v", err)
		return nil, err
	}
	client, err := speech.NewClient(ctx,
		googleOption.GetClientOptions()...,
	)

	if err != nil {
		logger.Errorf("google-stt: Error creating Google client: %v", err)
		return nil, err
	}

	// Context for callback management
	logger.Benchmark("google.NewGoogleSpeechToText", time.Since(start))
	return &googleSpeechToText{
		ctx:          ctx,
		logger:       logger,
		client:       client,
		googleOption: googleOption,
		options:      opts,
	}, nil
}

func (google *googleSpeechToText) Initialize() error {
	google.mu.Lock()
	defer google.mu.Unlock()

	stream, err := google.client.StreamingRecognize(google.ctx)
	if err != nil {
		google.logger.Errorf("google-stt: error creating google-stt stream: %v", err)
		return err
	}
	google.stream = stream
	err = google.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: google.SpeechToTextOptions(),
		},
	})
	if err != nil {
		return err
	}
	// Launch callback listener
	go google.SpeechToTextCallback(google.ctx)
	return nil
}

func (g *googleSpeechToText) Close(ctx context.Context) error {
	var combinedErr error
	if g.stream != nil {
		// Attempt to close the streaming client.
		err := g.stream.CloseSend()
		if err != nil {
			// Log the error if closure fails.
			combinedErr = fmt.Errorf("error closing StreamClient: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}

	if g.client != nil {
		// Attempt to close the client.
		err := g.client.Close()
		if err != nil {
			// Log the error if closure fails.
			combinedErr = fmt.Errorf("error closing Client: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}
	return combinedErr
}
