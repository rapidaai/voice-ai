// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_assemblyai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type assemblyaiSTT struct {
	ctx        context.Context
	cancel     context.CancelFunc
	logger     commons.Logger
	connection *websocket.Conn
	//
	transformerOptions *internal_transformers.SpeechToTextInitializeOptions
	providerOptions    AssemblyaiOption
}

func NewAssemblyaiSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	iOption *internal_transformers.SpeechToTextInitializeOptions,
) (internal_transformers.SpeechToTextTransformer, error) {
	ayOptions, err := NewAssemblyaiOption(
		logger,
		credential,
		iOption.AudioConfig,
		iOption.ModelOptions,
	)
	if err != nil {
		logger.Errorf("Key from credential failed %+v", err)
		return nil, err
	}
	gctx, cancel := context.WithCancel(ctx)
	return &assemblyaiSTT{
		logger:             logger,
		providerOptions:    ayOptions,
		transformerOptions: iOption,
		ctx:                gctx,
		cancel:             cancel,
	}, nil
}

func (aai *assemblyaiSTT) Name() string {
	return "assemblyai-speech-to-text"
}

func (aai *assemblyaiSTT) Initialize() error {
	start := time.Now()

	utils.Go(aai.ctx, func() {
		headers := http.Header{}
		headers.Set("Authorization", aai.providerOptions.GetKey())
		dialer := websocket.Dialer{
			Proxy:            nil,              // Skip proxy for direct connection
			HandshakeTimeout: 10 * time.Second, // Reduced handshake timeout for quick failover
		}
		conenction, _, err := dialer.Dial(aai.providerOptions.GetSpeechToTextConnectionString(), headers)
		if err != nil {
			// return fmt.Errorf("failed to connect to AssemblyAI WebSocket: %w", err)
		}
		aai.connection = conenction
		aai.speech()
	})
	aai.logger.Benchmark("AssemblyaiSTT.Initialize", time.Since(start))
	return nil
}

func (aai *assemblyaiSTT) speech() {
	for {
		select {
		case <-aai.ctx.Done():
			aai.logger.Info("Cartesia STT read goroutine exiting due to context cancellation")
			return
		default:
			_, msg, err := aai.connection.ReadMessage()

			aai.logger.Debug("Other message: ", string(msg))
			if err != nil {
				aai.logger.Error("read error: ", err)
				return
			}
			var transcript TranscriptMessage
			if err := json.Unmarshal(msg, &transcript); err != nil {
				continue
			}
			switch transcript.Type {
			case "Turn":
				confidence := 0.0
				for _, v := range transcript.Words {
					confidence += v.Confidence
				}
				averageConfidence := confidence / float64(len(transcript.Words))
				if transcript.EndOfTurn {
					aai.transformerOptions.OnTranscript(
						transcript.Transcript,
						averageConfidence,
						"en",
						true,
					)
				} else {
					aai.transformerOptions.OnTranscript(
						transcript.Transcript,
						averageConfidence,
						"en",
						false,
					)
				}
			case "Begin":
				aai.logger.Debug("Session began: ", string(msg))
			default:
				aai.logger.Debug("Other message: ", string(msg))
			}
		}
	}
}
func (aai *assemblyaiSTT) Transform(ctx context.Context, in []byte, opts *internal_transformers.SpeechToTextOption) error {
	if aai.connection == nil {
		return fmt.Errorf("WebSocket connection is not initialized")
	}
	if err := aai.connection.WriteMessage(websocket.BinaryMessage, in); err != nil {
		return fmt.Errorf("error sending audio: %w", err)
	}
	return nil
}

func (aai *assemblyaiSTT) Close(ctx context.Context) error {
	if aai.cancel != nil {
		aai.cancel()
	}
	if aai.connection != nil {
		return aai.connection.Close()
	}
	return nil
}
