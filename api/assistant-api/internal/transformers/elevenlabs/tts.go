// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_elevenlabs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type elevenlabsTTS struct {
	logger            commons.Logger
	connection        *websocket.Conn
	ctx               context.Context
	providerOption    ElevenLabsOption
	transformerOption *internal_transformers.TextToSpeechInitializeOptions
}

func NewElevenlabsTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	opts *internal_transformers.TextToSpeechInitializeOptions) (internal_transformers.TextToSpeechTransformer, error) {
	cOptions, err := NewElevenLabsOption(
		logger,
		credential,
		opts.AudioConfig,
		opts.ModelOptions)
	if err != nil {
		logger.Errorf("intializing elevenlabs failed %+v", err)
		return nil, err
	}

	//
	header := http.Header{}
	header.Set("xi-api-key", cOptions.GetKey())
	logger.Debugf("connecting with elevenlabs %v", cOptions.GetTextToSpeechConnectionString())
	conn, resp, err := websocket.DefaultDialer.Dial(cOptions.GetTextToSpeechConnectionString(), header)
	if err != nil {
		logger.Errorf("error while elevenlabs %s with response %v", err, resp)
		return nil, err
	}
	return &elevenlabsTTS{
		connection:        conn,
		logger:            logger,
		ctx:               ctx,
		providerOption:    cOptions,
		transformerOption: opts,
	}, nil
}

// Initialize implements internal_transformers.OutputAudioTransformer.
func (ct *elevenlabsTTS) Initialize() error {
	utils.Go(context.Background(), func() {
		ct.speech()
	})
	return nil
}

// Name implements internal_transformers.OutputAudioTransformer.
func (*elevenlabsTTS) Name() string {
	return "elevenlabs-text-to-speech"
}

func (t *elevenlabsTTS) speech() {
	for {
		_, audioChunk, err := t.connection.ReadMessage()
		if err != nil {
			t.logger.Errorf("Error reading from TTS WebSocket: %v", err)
		}

		var audioData map[string]interface{}
		if err := json.Unmarshal(audioChunk, &audioData); err != nil {
			t.logger.Errorf("Error parsing audio chunk: %v", err)
			break
		}

		contextId, ok := audioData["contextId"].(string)
		if !ok {
			continue
		}
		done, ok := audioData["isFinal"].(bool)
		if ok && done {
			t.transformerOption.OnComplete(contextId)
		}
		payload, ok := audioData["audio"].(string)
		if !ok {
			continue
		}
		rawAudioData, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			log.Fatalf("Error decoding base64 string: %v", err)
		}
		t.transformerOption.OnSpeech(contextId, rawAudioData)
	}
}

func (t *elevenlabsTTS) Transform(ctx context.Context, in string, opts *internal_transformers.TextToSpeechOption) error {
	t.logger.Infof("elevenlabs-tts: speak %s with context id = %s and completed = %t", in, opts.ContextId, opts.IsComplete)
	ttsMessage := map[string]interface{}{
		"text":       in,
		"context_id": opts.ContextId,
		"flush":      !opts.IsComplete,
	}

	if err := t.connection.WriteJSON(ttsMessage); err != nil {
		return err
	}
	return nil
}

func (t *elevenlabsTTS) Close(ctx context.Context) error {
	t.connection.Close()
	return nil
}
