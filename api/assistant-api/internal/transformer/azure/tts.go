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
	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type azureTextToSpeech struct {
	*azureOption
	mu sync.Mutex
	// context management
	ctx       context.Context
	ctxCancel context.CancelFunc

	contextId   string
	logger      commons.Logger
	stream      *audio.PullAudioOutputStream
	audioConfig *audio.AudioConfig
	client      *speech.SpeechSynthesizer
	onPacket    func(pkt ...internal_type.Packet) error
}

func NewAzureTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	audioConfig *protos.AudioConfig,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {

	azureOption, err := NewAzureOption(logger, credential, audioConfig, opts)
	if err != nil {
		logger.Errorf("azure-tts: Unable to initialize azure option: %v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &azureTextToSpeech{
		ctx:       ct,
		ctxCancel: ctxCancel,

		azureOption: azureOption,
		logger:      logger,
		onPacket:    onPacket,
	}, nil
}

func (azure *azureTextToSpeech) Name() string {
	return "azure-text-to-speech"
}

func (azure *azureTextToSpeech) Close(ctx context.Context) error {
	azure.ctxCancel()
	azure.mu.Lock()
	defer azure.mu.Unlock()

	if azure.client != nil {
		// Stop any ongoing synthesis before closing
		<-azure.client.StopSpeakingAsync()
		azure.client.Close()
		azure.client = nil
	}
	if azure.audioConfig != nil {
		azure.audioConfig.Close()
		azure.audioConfig = nil
	}
	if azure.stream != nil {
		azure.stream.Close()
		azure.stream = nil
	}
	return nil
}

func (azure *azureTextToSpeech) Initialize() (err error) {
	stream, err := audio.CreatePullAudioOutputStream()
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to create audio stream: %v", err)
		return fmt.Errorf("azure-tts: failed to create audio stream: %w", err)
	}
	audioConfig, err := audio.NewAudioConfigFromStreamOutput(stream)
	if err != nil {
		stream.Close()
		azure.logger.Errorf("azure-tts: failed to create audio config: %v", err)
		return fmt.Errorf("azure-tts: failed to create audio config: %w", err)
	}

	speechConfig, err := azure.TextToSpeechOption()
	if err != nil {
		stream.Close()
		audioConfig.Close()
		azure.logger.Errorf("azure-tts: failed to get speech configuration: %v", err)
		return fmt.Errorf("azure-tts: failed to get speech configuration: %w", err)
	}
	// Close speechConfig after creating synthesizer as it's no longer needed
	defer speechConfig.Close()

	client, err := speech.NewSpeechSynthesizerFromConfig(speechConfig, audioConfig)
	if err != nil {
		stream.Close()
		audioConfig.Close()
		azure.logger.Errorf("azure-tts: failed to initialize speech synthesizer: %v", err)
		return fmt.Errorf("azure-tts: failed to initialize speech synthesizer: %w", err)
	}

	azure.mu.Lock()
	azure.stream = stream
	azure.client = client
	azure.audioConfig = audioConfig
	azure.mu.Unlock()

	azure.client.SynthesisStarted(azure.OnStart)
	azure.client.Synthesizing(azure.OnSpeech)
	azure.client.SynthesisCompleted(azure.OnComplete)
	azure.client.SynthesisCanceled(azure.OnCancel)
	return nil
}

func (azure *azureTextToSpeech) Transform(ctx context.Context, in internal_type.LLMPacket) error {
	azure.mu.Lock()
	cl := azure.client
	azure.mu.Unlock()

	if cl == nil {
		return fmt.Errorf("azure-tts: client not initialized")
	}

	azure.mu.Lock()
	currentCtx := azure.contextId
	azure.contextId = in.ContextId()
	azure.mu.Unlock()

	if currentCtx != azure.contextId && currentCtx != "" {
		<-cl.StopSpeakingAsync()
	}

	switch input := in.(type) {
	case internal_type.LLMStreamPacket:
		res := <-cl.StartSpeakingTextAsync(input.Text)
		if res.Error != nil {
			return res.Error
		}

		return nil
	case internal_type.LLMMessagePacket:
		return nil
	default:
		return fmt.Errorf("azure-tts: unsupported input type %T", in)
	}

}

func (azCallback *azureTextToSpeech) OnStart(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
}

func (azCallback *azureTextToSpeech) OnSpeech(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	azCallback.mu.Lock()
	ctxId := azCallback.contextId
	azCallback.mu.Unlock()

	azCallback.onPacket(internal_type.TextToSpeechAudioPacket{
		ContextID:  ctxId,
		AudioChunk: event.Result.AudioData,
	})
}

func (azCallback *azureTextToSpeech) OnComplete(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	azCallback.mu.Lock()
	ctxId := azCallback.contextId
	azCallback.mu.Unlock()

	azCallback.onPacket(internal_type.TextToSpeechEndPacket{
		ContextID: ctxId,
	})
}

func (azCallback *azureTextToSpeech) OnCancel(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	if event.Result.Reason == common.Canceled {
		cancellation, _ := speech.NewCancellationDetailsFromSpeechSynthesisResult(&event.Result)
		azCallback.logger.Warnf("azure-tts: synthesis canceled: reason=%v, errorCode=%v, errorDetails=%v",
			cancellation.Reason, cancellation.ErrorCode, cancellation.ErrorDetails)
	}
}
