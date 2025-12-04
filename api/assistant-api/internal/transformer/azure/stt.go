// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"context"
	"fmt"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type azureSpeechToText struct {
	*azureOption
	logger            commons.Logger
	client            *speech.SpeechRecognizer
	azureAudioConfig  *audio.AudioConfig
	inputstream       *audio.PushAudioInputStream
	transformerOption *internal_transformer.SpeechToTextInitializeOptions
}

func (azure *azureSpeechToText) Initialize() (err error) {
	azure.inputstream, err = audio.CreatePushAudioInputStreamFromFormat(azure.GetAudioStreamFormat())
	if err != nil {
		azure.logger.Errorf("azure-stt: failed to create push audio input stream: %v", err)
		return fmt.Errorf("failed to create push audio input stream: %w", err)
	}

	azure.azureAudioConfig, err = audio.NewAudioConfigFromStreamInput(azure.inputstream)
	if err != nil {
		azure.logger.Errorf("azure-stt: failed to create audio config from stream input: %v", err)
		return fmt.Errorf("failed to create audio config from stream input: %w", err)
	}

	speechConfig, err := azure.SpeechToTextOption()
	if err != nil {
		azure.logger.Errorf("azure-stt: failed to create speech config from subscription: %v", err)
		return fmt.Errorf("failed to create speech config from subscription: %w", err)
	}

	azure.client, err = speech.NewSpeechRecognizerFromConfig(speechConfig, azure.azureAudioConfig)
	if err != nil {
		azure.logger.Errorf("azure-stt: failed to create speech recognizer from config: %v", err)
		return fmt.Errorf("failed to create speech recognizer from config: %w", err)
	}

	azure.client.SessionStarted(azure.OnSessionStarted)
	azure.client.SessionStopped(azure.OnSessionStopped)
	azure.client.Recognizing(azure.OnRecognizing)
	azure.client.Recognized(azure.OnRecognized)
	azure.client.Canceled(azure.OnCancelled)
	azure.client.StartContinuousRecognitionAsync()
	return nil
}

// Name implements internal_transformer.SpeechToTextTransformer.
func (a *azureSpeechToText) Name() string {
	return "azure-speech-to-text"
}

// Transform implements internal_transformer.SpeechToTextTransformer.
func (azure *azureSpeechToText) Transform(ctx context.Context, ad []byte, opts *internal_transformer.SpeechToTextOption) (err error) {
	if azure.inputstream == nil {
		return fmt.Errorf("azure-stt: you are calling transform without initilize")
	}

	if err := azure.inputstream.Write(ad); err != nil {
		azure.logger.Debugf("azure-stt: error writing audio bytes %v", err)
		return fmt.Errorf("failed to write audio data to push stream: %w", err)
	}
	return nil
}

func NewAzureSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	iOptions *internal_transformer.SpeechToTextInitializeOptions) (internal_transformer.SpeechToTextTransformer, error) {
	azure, err := NewAzureOption(logger,
		credential,
		iOptions.AudioConfig,
		iOptions.ModelOptions)
	if err != nil {
		logger.Errorf("azure-stt: Unable to initilize azure option", err)
		return nil, err
	}
	return &azureSpeechToText{
		logger:            logger,
		transformerOption: iOptions,
		azureOption:       azure,
	}, nil
}

func (azCallback *azureSpeechToText) OnSessionStarted(event speech.SessionEventArgs) {
	defer event.Close()
}

func (azCallback *azureSpeechToText) OnSessionStopped(event speech.SessionEventArgs) {
	defer event.Close()
}

func (azCallback *azureSpeechToText) OnRecognizing(event speech.SpeechRecognitionEventArgs) {
	defer event.Close()
	azCallback.transformerOption.OnTranscript(
		event.Result.Text,
		0.9,
		"en",
		false,
	)
}

func (azCallback *azureSpeechToText) OnRecognized(event speech.SpeechRecognitionEventArgs) {
	defer event.Close()
	azCallback.transformerOption.OnTranscript(
		event.Result.Text,
		0.9,
		"en",
		true)
}

func (azCallback *azureSpeechToText) OnCancelled(event speech.SpeechRecognitionCanceledEventArgs) {
	defer event.Close()
}

// Cancel implements internal_transformer.SpeechToTextTransformer.
func (azure *azureSpeechToText) Close(ctx context.Context) error {
	if azure.client != nil {
		azure.client.StopContinuousRecognitionAsync()
		azure.client.Close()
	}
	if azure.inputstream != nil {
		azure.inputstream.Close()
	}
	if azure.azureAudioConfig != nil {
		azure.azureAudioConfig.Close()
	}
	return nil
}
