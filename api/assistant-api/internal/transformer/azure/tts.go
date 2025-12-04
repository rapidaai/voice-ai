// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"context"
	"fmt"
	"sync"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
)

type azureTextToSpeech struct {
	*azureOption
	mu          sync.Mutex
	contextId   string
	logger      commons.Logger
	audioConfig *audio.AudioConfig
	client      *speech.SpeechSynthesizer
	options     *internal_transformer.TextToSpeechInitializeOptions
}

func NewAzureTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	iOption *internal_transformer.TextToSpeechInitializeOptions) (internal_transformer.TextToSpeechTransformer, error) {
	azureOption, err := NewAzureOption(logger, credential,
		iOption.AudioConfig,
		iOption.ModelOptions)
	if err != nil {
		logger.Errorf("azure-tts: Unable to initilize azure option", err)
		return nil, err
	}
	return &azureTextToSpeech{
		azureOption: azureOption,
		logger:      logger,
		options:     iOption,
	}, nil
}

func (azure *azureTextToSpeech) Name() string {
	return "azure-text-to-speech"
}

func (azure *azureTextToSpeech) Close(ctx context.Context) error {
	if azure.client != nil {
		azure.client.Close()
	}
	if azure.audioConfig != nil {
		azure.audioConfig.Close()
	}
	return nil
}

func (azure *azureTextToSpeech) Initialize() (err error) {
	stream, err := audio.CreatePullAudioOutputStream()
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to create audio stream:", err)
		return fmt.Errorf("azure-tts: failed to create audio stream: %w", err)
	}
	azure.audioConfig, err = audio.NewAudioConfigFromStreamOutput(stream)
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to create audio config:", err)
		return fmt.Errorf("azure-tts: failed to create audio config: %w", err)
	}

	speechConfig, err := azure.TextToSpeechOption()
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to get speech configuration:", err)
		return fmt.Errorf("azure-tts: failed to get speech configuration: %w", err)
	}

	azure.client, err = speech.NewSpeechSynthesizerFromConfig(speechConfig, azure.audioConfig)
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to initialize speech synthesizer:", err)
		return fmt.Errorf("azure-tts: failed to initialize speech synthesizer: %w", err)
	}
	azure.client.SynthesisStarted(azure.OnStart)
	azure.client.Synthesizing(azure.OnSpeech)
	azure.client.SynthesisCompleted(azure.OnComplete)
	azure.client.SynthesisCanceled(azure.OnCancel)
	return nil
}

func (azure *azureTextToSpeech) Transform(ctx context.Context, text string, opts *internal_transformer.TextToSpeechOption) error {
	azure.mu.Lock()
	defer azure.mu.Unlock()
	azure.contextId = opts.ContextId
	if azure.client == nil {
		return fmt.Errorf("azure-tts: you are calling transform without initilize")
	}

	res := <-azure.client.SpeakTextAsync(text)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (azCallback *azureTextToSpeech) OnStart(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
}

func (azCallback *azureTextToSpeech) OnSpeech(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	azCallback.options.OnSpeech(azCallback.contextId, event.Result.AudioData)
}

func (azCallback *azureTextToSpeech) OnComplete(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	azCallback.options.OnComplete(azCallback.contextId)
}

func (azCallback *azureTextToSpeech) OnCancel(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
}
