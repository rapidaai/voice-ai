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
	"strings"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

// googleTextToSpeech is the main struct handling Google Text-to-Speech functionality.
type googleTextToSpeech struct {
	*googleOption
	mu                 sync.Mutex // Ensures thread-safe operations.
	ctx                context.Context
	contextId          string                                                // Tracks context ID for audio synthesis.
	logger             commons.Logger                                        // Logger for debugging and error reporting.
	client             *texttospeech.Client                                  // Google TTS client.
	streamClient       texttospeechpb.TextToSpeech_StreamingSynthesizeClient // Streaming client for real-time TTS.
	transformerOptions *internal_transformer.TextToSpeechInitializeOptions   // Options for TTS initialization.
}

// NewGoogleTextToSpeech creates a new instance of googleTextToSpeech.
func NewGoogleTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	opts *internal_transformer.TextToSpeechInitializeOptions) (internal_transformer.TextToSpeechTransformer, error) {
	// Initialize Google TTS options.
	googleOption, err := NewGoogleOption(logger, credential, opts.AudioConfig, opts.ModelOptions)
	if err != nil {
		// Log and return error if initialization fails.
		logger.Errorf("intializing google failed %+v", err)
		return nil, err
	}

	// Create Google TTS client with options.
	client, err := texttospeech.NewClient(
		ctx,
		googleOption.GetClientOptions()...)
	if err != nil {
		// Log and return error if client creation fails.
		logger.Errorf("error while creating client for google tts %+v", err)
		return nil, err
	}

	// Return configured TTS instance.
	return &googleTextToSpeech{
		ctx:                ctx,
		logger:             logger,
		transformerOptions: opts,
		client:             client,
		googleOption:       googleOption,
	}, nil
}

// Close safely shuts down the TTS client and streaming client.
func (g *googleTextToSpeech) Close(ctx context.Context) error {
	var combinedErr error
	if g.streamClient != nil {
		// Attempt to close the streaming client.
		err := g.streamClient.CloseSend()
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

// Initialize sets up the streaming synthesis functionality.
func (google *googleTextToSpeech) Initialize() error {
	google.mu.Lock()
	defer google.mu.Unlock()

	// Start a streaming synthesis session.
	stream, err := google.client.StreamingSynthesize(google.ctx)
	if err != nil {
		// Log any initialization errors.
		google.logger.Errorf("error while creating by directional for google tts %+v", err)
		return err
	}

	req := texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.
			StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: google.TextToSpeechOptions(),
		},
	}
	google.streamClient = stream
	// Send the initial configuration request.
	if err = stream.Send(&req); err != nil {
		// Log errors in sending initialization request.
		google.logger.Errorf("error while intiializing google text to speech")
		return err
	}
	// Launch callback goroutine for processing streaming responses.
	go google.textToSpeechCallback(google.ctx)
	google.logger.Debugf("google-tts: connection established")
	return nil
}

// Name returns the name of this transformer implementation.
func (*googleTextToSpeech) Name() string {
	return "google-text-to-speech"
}

// Transform handles streaming synthesis requests for input text.
func (google *googleTextToSpeech) Transform(ctx context.Context, in string, opts *internal_transformer.TextToSpeechOption) error {
	google.logger.Infof("google-tts: speak %s with context id = %s and completed = %t", in, opts.ContextId, opts.IsComplete)
	google.mu.Lock()
	defer google.mu.Unlock()

	if google.streamClient == nil {
		return fmt.Errorf("you are calling transform without initilize")
	}
	google.contextId = opts.ContextId
	// Construct synthesis request with input text.
	req := texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
			Input: &texttospeechpb.StreamingSynthesisInput{
				InputSource: &texttospeechpb.StreamingSynthesisInput_Text{Text: in},
			},
		},
	}
	// Send synthesis request to the streaming client.
	if err := google.streamClient.Send(&req); err != nil {
		// Log any errors during synthesis.
		google.logger.Errorf("unable to Synthesize text %v", err)
	}
	return nil
}

// textToSpeechCallback processes streaming responses asynchronously.
func (g *googleTextToSpeech) textToSpeechCallback(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.logger.Infof("google-tts: context cancelled, stopping response listener")
			return
		default:
			// Receive audio content from the stream.
			resp, err := g.streamClient.Recv()
			if err != nil {
				if err == io.EOF {
					// Handle end-of-file scenario gracefully.
					g.logger.Infof("google-tts: stream ended (EOF)")
					continue
				}
				if strings.Contains(err.Error(), "Stream aborted due to long duration elapsed without input sent") {
					// Restart initialization if stream is aborted.
					g.Initialize()
					return
				}
				// Log errors during response retrieval.
			}
			if resp != nil {
				// Pass audio content to the speech handler.
				g.transformerOptions.OnSpeech(
					g.contextId,
					resp.GetAudioContent())
			}
		}
	}
}
