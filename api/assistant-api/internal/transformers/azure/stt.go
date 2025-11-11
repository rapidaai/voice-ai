// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type azureSpeechToText struct {
	logger      commons.Logger
	client      *speech.SpeechRecognizer
	audioConfig *audio.AudioConfig
	inputstream *audio.PushAudioInputStream

	//
	providerOption    AzureOption
	transformerOption *internal_transformers.SpeechToTextInitializeOptions
}

// Cancel implements internal_transformers.SpeechToTextTransformer.
func (azure *azureSpeechToText) Close(ctx context.Context) error {
	azure.client.StopContinuousRecognitionAsync()
	azure.inputstream.Close()
	azure.client.Close()
	azure.audioConfig.Close()
	return nil
}

func (azure *azureSpeechToText) Initialize() (err error) {
	azure.inputstream, err = audio.CreatePushAudioInputStream()
	if err != nil {
		azure.logger.Debugf("Failed to create push audio input stream: %v", err)

		return fmt.Errorf("failed to create push audio input stream: %w", err)
	}

	azure.audioConfig, err = audio.NewAudioConfigFromStreamInput(azure.inputstream)
	if err != nil {
		azure.logger.Debugf("Failed to create audio config from stream input: %v", err)

		return fmt.Errorf("failed to create audio config from stream input: %w", err)
	}

	speechConfig, err := azure.providerOption.SpeechToTextOption()
	if err != nil {
		azure.logger.Debugf("Failed to create speech config from subscription: %v", err)

		return fmt.Errorf("failed to create speech config from subscription: %w", err)
	}

	azure.client, err = speech.NewSpeechRecognizerFromConfig(speechConfig, azure.audioConfig)
	if err != nil {
		azure.logger.Debugf("Failed to create speech recognizer from config: %v", err)

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

// Name implements internal_transformers.SpeechToTextTransformer.
func (a *azureSpeechToText) Name() string {
	return "azure-speech-to-text"
}

// Transform implements internal_transformers.SpeechToTextTransformer.
func (azure *azureSpeechToText) Transform(ctx context.Context, ad []byte, opts *internal_transformers.SpeechToTextOption) (err error) {
	if err := azure.inputstream.Write(ad); err != nil {
		return fmt.Errorf("failed to write audio data to push stream: %w", err)
	}
	return nil
}

func NewAzureSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *lexatic_backend.VaultCredential,
	iOptions *internal_transformers.SpeechToTextInitializeOptions) (internal_transformers.SpeechToTextTransformer, error) {
	providerOption, err := NewAzureOption(logger,
		credential,
		iOptions.AudioConfig,
		iOptions.ModelOptions)
	if err != nil {
		logger.Errorf("Unable to initilize azure option", err)
		return nil, err
	}

	logger.Benchmark("azure.NewAzureSpeechToText", time.Since(time.Now()))
	return &azureSpeechToText{
		logger:            logger,
		transformerOption: iOptions,
		providerOption:    providerOption,
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
	azCallback.logger.Debugf("azure got %+v", event)
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
