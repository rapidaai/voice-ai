// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	azure_internal "github.com/rapidaai/api/assistant-api/internal/transformer/azure/internal"
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

	contextId      string
	ttsConnectedAt time.Time

	// TTS latency tracking
	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger      commons.Logger
	stream      *audio.PullAudioOutputStream
	audioConfig *audio.AudioConfig
	client      *speech.SpeechSynthesizer
	onPacket    func(pkt ...internal_type.Packet) error
	normalizer  internal_type.TextNormalizer
}

func NewAzureTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {

	azureOption, err := NewAzureOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("azure-tts: unable to initialize azure option: %v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &azureTextToSpeech{
		ctx:       ct,
		ctxCancel: ctxCancel,

		azureOption: azureOption,
		logger:      logger,
		onPacket:    onPacket,
		normalizer:  azure_internal.NewAzureNormalizer(logger, opts),
	}, nil
}

func (azure *azureTextToSpeech) Name() string {
	return "azure-tts"
}

func (azure *azureTextToSpeech) Close(ctx context.Context) error {
	azure.ctxCancel()
	azure.mu.Lock()
	ctxID := azure.contextId
	connectedAt := azure.ttsConnectedAt
	azure.ttsConnectedAt = time.Time{}

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
	azure.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		azure.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": azure.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewTTSDurationUsageRecord(azure.Name(), duration, observability.Attributes{}),
			},
		)
	}
	azure.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": azure.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (azure *azureTextToSpeech) Initialize() (err error) {
	start := time.Now()
	emitInitializationErrorLog := func(initializationErr error) {
		azure.onPacket(internal_type.ObservabilityLogRecordPacket{
			ContextID: azure.contextId,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "azure-tts: initialization failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  azure.Name(),
					"error":     observability.AttributeValue(initializationErr.Error()),
				},
				OccurredAt: time.Now(),
			},
		})
	}
	stream, err := audio.CreatePullAudioOutputStream()
	if err != nil {
		azure.logger.Errorf("azure-tts: failed to create audio stream: %v", err)
		initializationErr := fmt.Errorf("azure-tts: failed to create audio stream: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}
	audioConfig, err := audio.NewAudioConfigFromStreamOutput(stream)
	if err != nil {
		stream.Close()
		azure.logger.Errorf("azure-tts: failed to create audio config: %v", err)
		initializationErr := fmt.Errorf("azure-tts: failed to create audio config: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	speechConfig, err := azure.TextToSpeechOption()
	if err != nil {
		stream.Close()
		audioConfig.Close()
		azure.logger.Errorf("azure-tts: failed to get speech configuration: %v", err)
		initializationErr := fmt.Errorf("azure-tts: failed to get speech configuration: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}
	// Close speechConfig after creating synthesizer as it's no longer needed
	defer speechConfig.Close()

	client, err := speech.NewSpeechSynthesizerFromConfig(speechConfig, audioConfig)
	if err != nil {
		stream.Close()
		audioConfig.Close()
		azure.logger.Errorf("azure-tts: failed to initialize speech synthesizer: %v", err)
		initializationErr := fmt.Errorf("azure-tts: failed to initialize speech synthesizer: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	azure.mu.Lock()
	azure.stream = stream
	azure.client = client
	azure.audioConfig = audioConfig
	if azure.ttsConnectedAt.IsZero() {
		azure.ttsConnectedAt = time.Now()
	}
	azure.mu.Unlock()

	azure.client.SynthesisStarted(azure.OnStart)
	azure.client.Synthesizing(azure.OnSpeech)
	azure.client.SynthesisCompleted(azure.OnComplete)
	azure.client.SynthesisCanceled(azure.OnCancel)
	azure.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": azure.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "azure-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  azure.Name(),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (azure *azureTextToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	azure.mu.Lock()
	cl := azure.client
	previousContextID := azure.contextId
	if in.ContextId() != azure.contextId {
		azure.contextId = in.ContextId()
		azure.ttsStartedAt = time.Time{}
		azure.ttsMetricSent = false
	}
	azure.mu.Unlock()
	if cl == nil {
		return nil
	}

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		if previousContextID != "" {
			azure.mu.Lock()
			azure.contextId = ""
			azure.ttsStartedAt = time.Time{}
			azure.ttsMetricSent = false
			azure.mu.Unlock()
			<-cl.StopSpeakingAsync()
			azure.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentTTS,
					Event:      observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil
	case internal_type.TextToSpeechTextPacket:
		normalizedText := input.Text
		if azure.normalizer != nil {
			normalizedText = azure.normalizer.Normalize(input.Text)
		}
		azure.mu.Lock()
		if azure.ttsStartedAt.IsZero() {
			azure.ttsStartedAt = time.Now()
		}
		azure.mu.Unlock()
		var res speech.SpeechSynthesisOutcome
		if strings.Contains(normalizedText, "<break ") {
			language := "en-US"
			if configuredLanguage, err := azure.mdlOpts.GetString(internal_options.SpeakOptionLanguage); err == nil && configuredLanguage != "" {
				language = configuredLanguage
			}
			voiceName := ""
			if configuredVoice, err := azure.mdlOpts.GetString(internal_options.SpeakOptionVoiceID); err == nil && configuredVoice != "" {
				voiceName = configuredVoice
			}
			textForAzure := fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s">%s</speak>`, language, normalizedText)
			if voiceName != "" {
				textForAzure = fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s"><voice name="%s">%s</voice></speak>`, language, voiceName, normalizedText)
			}
			res = <-cl.StartSpeakingSsmlAsync(textForAzure)
		} else {
			res = <-cl.StartSpeakingTextAsync(normalizedText)
		}
		if res.Error != nil {
			synthesisErr := fmt.Errorf("azure-tts: synthesis failed: %w", res.Error)
			azure.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     synthesisErr,
					Type:      internal_type.TTSNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: input.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "azure-tts: synthesis failed",
						Attributes: observability.Attributes{
							"component": observability.ComponentTTS.String(),
							"provider":  azure.Name(),
							"error":     observability.AttributeValue(synthesisErr.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return nil
		}
		azure.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSSpeaking,
				Attributes: observability.Attributes{
					"type": "speaking",
					"text": normalizedText,
				},
				OccurredAt: time.Now(),
			},
		})
		return nil
	case internal_type.TextToSpeechDonePacket:
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
	var shouldEmitFirstAudioLatencyMetric bool
	azCallback.mu.Lock()
	ctxID := azCallback.contextId
	startedAt := azCallback.ttsStartedAt
	if ctxID == "" {
		azCallback.mu.Unlock()
		return
	}
	if !azCallback.ttsMetricSent && !startedAt.IsZero() {
		azCallback.ttsMetricSent = true
		shouldEmitFirstAudioLatencyMetric = true
	}
	azCallback.mu.Unlock()
	if shouldEmitFirstAudioLatencyMetric {
		azCallback.onPacket(internal_type.ObservabilityMetricRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record:    observability.NewMetricTTSLatencyMs(time.Since(startedAt), observability.Attributes{"provider": azCallback.Name()}),
		})
	}
	azCallback.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxID, AudioChunk: event.Result.AudioData})
}

func (azCallback *azureTextToSpeech) OnComplete(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	azCallback.mu.Lock()
	ctxID := azCallback.contextId
	azCallback.mu.Unlock()
	if ctxID == "" {
		return
	}
	azCallback.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxID},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSCompleted,
				Attributes: observability.Attributes{"type": "completed"},
				OccurredAt: time.Now(),
			},
		},
	)
}

func (azCallback *azureTextToSpeech) OnCancel(event speech.SpeechSynthesisEventArgs) {
	defer event.Close()
	if event.Result.Reason == common.Canceled {
		cancellation, _ := speech.NewCancellationDetailsFromSpeechSynthesisResult(&event.Result)
		azCallback.logger.Warnf("azure-tts: synthesis canceled: reason=%v, errorCode=%v, errorDetails=%v", cancellation.Reason, cancellation.ErrorCode, cancellation.ErrorDetails)
		azCallback.mu.Lock()
		ctxID := azCallback.contextId
		azCallback.mu.Unlock()
		if ctxID == "" {
			return
		}
		cancelErr := fmt.Errorf("azure-tts: synthesis canceled: %v (code=%v)", cancellation.ErrorDetails, cancellation.ErrorCode)
		azCallback.onPacket(
			internal_type.TextToSpeechErrorPacket{
				ContextID: ctxID,
				Error:     cancelErr,
				Type:      internal_type.TTSNetworkTimeout,
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "azure-tts: synthesis canceled",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  azCallback.Name(),
						"error":     observability.AttributeValue(cancelErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}
}
