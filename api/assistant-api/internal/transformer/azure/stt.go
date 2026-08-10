// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"context"
	"encoding/json"
	"fmt"
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

const defaultConfidence = 0.9

type azureSpeechToText struct {
	*azureOption
	mu sync.Mutex

	// context management
	ctx       context.Context
	ctxCancel context.CancelFunc

	logger           commons.Logger
	client           *speech.SpeechRecognizer
	azureAudioConfig *audio.AudioConfig
	inputstream      *audio.PushAudioInputStream
	onPacket         func(pkt ...internal_type.Packet) error

	// observability: time when speech started
	startedAt      time.Time
	contextId      string
	sttConnectedAt time.Time
}

type options struct {
	ctx        context.Context
	logger     commons.Logger
	credential *protos.VaultCredential
	onPacket   func(pkt ...internal_type.Packet) error
	sttOptions utils.Option

	assistantID    uint64
	conversationID uint64
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithCredential(credential *protos.VaultCredential) Option {
	return func(options *options) {
		options.credential = credential
	}
}

func WithOnPacket(onPacket func(pkt ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(sttOptions utils.Option) Option {
	return func(options *options) {
		options.sttOptions = sttOptions
	}
}

func WithAssistantID(assistantID uint64) Option {
	return func(options *options) {
		options.assistantID = assistantID
	}
}

func WithConversationID(conversationID uint64) Option {
	return func(options *options) {
		options.conversationID = conversationID
	}
}

func NewSpeechToText(opts ...Option) (internal_type.SpeechToTextTransformer, error) {
	options := &options{ctx: context.Background(), sttOptions: utils.Option{}}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}

	azureOpt, err := NewAzureOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("azure-stt: unable to initialize azure option: %v", err)
		return nil, err
	}

	childCtx, cancel := context.WithCancel(options.ctx)
	return &azureSpeechToText{
		ctx:         childCtx,
		ctxCancel:   cancel,
		logger:      options.logger,
		onPacket:    options.onPacket,
		azureOption: azureOpt,
	}, nil
}

// Initialize sets up the Azure Speech-to-Text recognizer with audio stream and event handlers.
func (s *azureSpeechToText) Initialize() error {
	start := time.Now()
	emitInitializationErrorLog := func(initializationErr error) {
		s.onPacket(internal_type.ObservabilityLogRecordPacket{
			ContextID: s.contextId,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "azure-stt: initialization failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  s.Name(),
					"error":     observability.AttributeValue(initializationErr.Error()),
				},
				OccurredAt: time.Now(),
			},
		})
	}
	inputStream, err := audio.CreatePushAudioInputStreamFromFormat(s.GetAudioStreamFormat())
	if err != nil {
		s.logger.Errorf("azure-stt: failed to create push audio input stream: %v", err)
		initializationErr := fmt.Errorf("failed to create push audio input stream: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	audioConfig, err := audio.NewAudioConfigFromStreamInput(inputStream)
	if err != nil {
		s.logger.Errorf("azure-stt: failed to create audio config from stream input: %v", err)
		initializationErr := fmt.Errorf("failed to create audio config from stream input: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	speechConfig, err := s.SpeechToTextOption()
	if err != nil {
		s.logger.Errorf("azure-stt: failed to create speech config from subscription: %v", err)
		initializationErr := fmt.Errorf("failed to create speech config from subscription: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	client, err := speech.NewSpeechRecognizerFromConfig(speechConfig, audioConfig)
	if err != nil {
		s.logger.Errorf("azure-stt: failed to create speech recognizer from config: %v", err)
		initializationErr := fmt.Errorf("failed to create speech recognizer from config: %w", err)
		emitInitializationErrorLog(initializationErr)
		return initializationErr
	}

	s.mu.Lock()
	s.client = client
	s.azureAudioConfig = audioConfig
	s.inputstream = inputStream
	s.sttConnectedAt = time.Now()
	s.mu.Unlock()

	s.registerEventHandlers()
	s.client.StartContinuousRecognitionAsync()

	s.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: s.contextId,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record:    observability.NewMetricSTTInitLatencyMs(time.Since(start), observability.Attributes{"provider": s.Name()}),
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: s.contextId,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "azure-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  s.Name(),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// registerEventHandlers sets up all the speech recognition event callbacks.
func (s *azureSpeechToText) registerEventHandlers() {
	s.client.SessionStarted(s.OnSessionStarted)
	s.client.SessionStopped(s.OnSessionStopped)
	s.client.Recognizing(s.OnRecognizing)
	s.client.Recognized(s.OnRecognized)
	s.client.Canceled(s.OnCancelled)
}

// Name returns the transformer identifier.
func (s *azureSpeechToText) Name() string {
	return "azure-stt"
}

// Transform writes audio data to the input stream for recognition.
func (s *azureSpeechToText) Transform(_ context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		s.mu.Lock()
		s.contextId = pkt.ContextID
		s.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		s.mu.Lock()
		if s.startedAt.IsZero() {
			s.startedAt = time.Now()
		}
		s.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		s.mu.Lock()
		if s.startedAt.IsZero() {
			s.startedAt = time.Now()
		}
		stream := s.inputstream
		ctxID := s.contextId
		s.mu.Unlock()

		if stream == nil {
			return nil
		}

		if err := stream.Write(pkt.Content()); err != nil {
			s.logger.Errorf("azure-stt: error sending audio: %v", err)
			sendErr := fmt.Errorf("azure-stt: send failed: %w", err)
			s.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: ctxID,
					Error:     sendErr,
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "azure-stt: error while sending audio",
						Attributes: observability.Attributes{
							"component": observability.ComponentSTT.String(),
							"provider":  s.Name(),
							"error":     observability.AttributeValue(sendErr.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return nil
		}

		return nil
	default:
		return nil
	}
}

func (s *azureSpeechToText) OnSessionStarted(event speech.SessionEventArgs) {
	defer event.Close()
}

func (s *azureSpeechToText) OnSessionStopped(event speech.SessionEventArgs) {
	defer event.Close()
}

// OnRecognizing handles interim speech recognition results.
func (s *azureSpeechToText) OnRecognizing(event speech.SpeechRecognitionEventArgs) {
	defer event.Close()

	jsonResult := event.Result.Properties.GetProperty(common.SpeechServiceResponseJSONResult, "{}")

	var result azure_internal.AzureRecognizingResult
	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		s.logger.Warnf("failed to parse recognizing result: %v", err)
		return
	}

	if result.Text == "" {
		return
	}

	s.mu.Lock()
	ctxID := s.contextId
	s.mu.Unlock()

	language := result.PrimaryLanguage.Language
	if language == "" {
		language = "en-US"
	}

	s.onPacket(
		internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
		internal_type.SpeechToTextPacket{
			ContextID:  ctxID,
			Script:     result.Text,
			Confidence: defaultConfidence,
			Language:   language,
			Interim:    true,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTInterim,
				Attributes: observability.Attributes{
					"type":       "interim",
					"script":     result.Text,
					"confidence": "0.9000",
				},
				OccurredAt: time.Now(),
			},
		},
	)
}

// OnRecognized handles final speech recognition results.
func (s *azureSpeechToText) OnRecognized(event speech.SpeechRecognitionEventArgs) {
	defer event.Close()
	jsonResult := event.Result.Properties.GetProperty(common.SpeechServiceResponseJSONResult, "{}")

	var result azure_internal.AzureRecognizedResult
	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		s.logger.Warnf("failed to parse recognized result: %v", err)
		return
	}
	if result.RecognitionStatus != "Success" {
		return
	}

	text := result.DisplayText
	confidence := defaultConfidence

	if len(result.NBest) > 0 {
		confidence = result.NBest[0].Confidence
		if threshold, err := s.mdlOpts.GetFloat64(internal_options.ListenOptionThreshold); err == nil {
			if confidence < threshold {
				s.logger.Debugf("confidence %.4f below threshold %.4f, skipping", confidence, threshold)
				s.mu.Lock()
				ctxID := s.contextId
				s.mu.Unlock()
				s.onPacket(
					internal_type.ObservabilityEventRecordPacket{
						ContextID: ctxID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentSTT,
							Event:     observability.STTLowConfidence,
							Attributes: observability.Attributes{
								"type":       "low_confidence",
								"script":     text,
								"confidence": fmt.Sprintf("%.4f", confidence),
								"threshold":  fmt.Sprintf("%.4f", threshold),
							},
							OccurredAt: time.Now(),
						},
					},
				)
				return
			}
		}
		if result.NBest[0].Display != "" {
			text = result.NBest[0].Display
		}
	}

	if text == "" {
		return
	}

	now := time.Now()
	var startedAt time.Time
	s.mu.Lock()
	if !s.startedAt.IsZero() {
		startedAt = s.startedAt
		s.startedAt = time.Time{}
	}
	ctxID := s.contextId
	s.mu.Unlock()

	confStr := fmt.Sprintf("%.4f", confidence)
	s.onPacket(
		internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
		internal_type.SpeechToTextPacket{
			ContextID:  ctxID,
			Script:     text,
			Confidence: confidence,
			Language:   "en-US",
			Interim:    false,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTCompleted,
				Attributes: observability.Attributes{
					"type":       "completed",
					"script":     text,
					"confidence": confStr,
					"language":   "en-US",
					"word_count": fmt.Sprintf("%d", len(strings.Fields(text))),
					"char_count": fmt.Sprintf("%d", len(text)),
				},
				OccurredAt: now,
			},
		},
	)
	if !startedAt.IsZero() {
		s.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record:    observability.NewMetricSTTLatencyMs(now.Sub(startedAt), observability.Attributes{"provider": s.Name()}),
			},
		)
	}
}

func (s *azureSpeechToText) OnCancelled(event speech.SpeechRecognitionCanceledEventArgs) {
	defer event.Close()
	s.mu.Lock()
	ctxID := s.contextId
	s.mu.Unlock()
	cancelErr := fmt.Errorf("azure-stt: recognition cancelled")
	s.onPacket(
		internal_type.SpeechToTextErrorPacket{
			ContextID: ctxID,
			Error:     cancelErr,
			Type:      internal_type.STTNetworkTimeout,
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "azure-stt: recognition cancelled",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  s.Name(),
					"error":     observability.AttributeValue(cancelErr.Error()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
}

// Close stops recognition and releases all Azure Speech SDK resources.
func (s *azureSpeechToText) Close(_ context.Context) error {
	s.ctxCancel()

	s.mu.Lock()
	ctxID := s.contextId
	connectedAt := s.sttConnectedAt
	s.sttConnectedAt = time.Time{}

	if s.client != nil {
		s.client.StopContinuousRecognitionAsync()
		s.client.Close()
	}
	if s.inputstream != nil {
		s.inputstream.Close()
	}
	if s.azureAudioConfig != nil {
		s.azureAudioConfig.Close()
	}
	s.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		s.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": s.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewSTTDurationUsageRecord(s.Name(), duration, observability.Attributes{}),
			},
		)
	}
	s.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": s.Name(),
				},
				OccurredAt: time.Now(),
			},
		})

	return nil
}
