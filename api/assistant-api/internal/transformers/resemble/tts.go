// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_resemble

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type resembleTTS struct {
	ctx       context.Context
	contextId string
	mu        sync.Mutex

	logger            commons.Logger
	connection        *websocket.Conn
	providerOption    ResembleOption
	transformerOption *internal_transformers.TextToSpeechInitializeOptions
}

func NewResembleTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	options *internal_transformers.TextToSpeechInitializeOptions,
) (internal_transformers.TextToSpeechTransformer, error) {
	wsURL := "wss://websocket.cluster.resemble.ai/stream"
	headers := map[string][]string{
		"Authorization": {"Bearer " + RESEMBLE_API_KEY},
	}
	cOptions, err := NewResembleOption(
		logger,
		credential,
		options.AudioConfig,
		options.ModelOptions)
	if err != nil {
		logger.Errorf("intializing resembleai failed %+v", err)
		return nil, err
	}
	//
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}
	return &resembleTTS{
		ctx:               ctx,
		logger:            logger,
		connection:        conn,
		transformerOption: options,
		providerOption:    cOptions,
	}, nil
}

// Initialize implements internal_transformers.OutputAudioTransformer.
func (rt *resembleTTS) Initialize() error {
	utils.Go(context.Background(), func() {
		rt.processAudio()
	})
	return nil
}

// Name implements internal_transformers.OutputAudioTransformer.
func (*resembleTTS) Name() string {
	return "resemble-text-to-speech"
}

func (rt *resembleTTS) Cleanup(ctx context.Context, opts *internal_transformers.TextToSpeechOption) error {
	return nil
}

func (rt *resembleTTS) processAudio() {
	for {
		_, audioChunk, err := rt.connection.ReadMessage()
		if err != nil {
			rt.logger.Errorf("Error reading from Resemble WebSocket: %v", err)
			continue
		}

		var audioData map[string]interface{}
		if err := json.Unmarshal(audioChunk, &audioData); err != nil {
			rt.logger.Errorf("Error parsing audio chunk: %v", err)
			continue
		}
		if audioData["type"] == "audio_end" {
			break
		}
		if audioData["type"] == "audio" {
			payload, ok := audioData["audio_content"].(string)
			if !ok {
				continue
			}
			rawAudioData, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				log.Fatalf("Error decoding base64 string: %v", err)
			}
			rt.transformerOption.OnSpeech(rt.contextId, rawAudioData)
		}
	}
}

func (rt *resembleTTS) Transform(ctx context.Context, in string, opts *internal_transformers.TextToSpeechOption) error {
	rt.logger.Infof("resemble-tts: speak %s with context id = %s and completed = %t", in, opts.ContextId, opts.IsComplete)
	rt.mu.Lock()
	rt.contextId = opts.ContextId
	rt.mu.Unlock()

	if err := rt.connection.WriteJSON(rt.providerOption.GetTextToSpeechRequest(opts.ContextId, in)); err != nil {
		rt.logger.Debugf("error while writing request to websocket %v", err)
		return err
	}
	return nil
}

func (rt *resembleTTS) Close(ctx context.Context) error {
	rt.connection.Close()
	return nil
}
