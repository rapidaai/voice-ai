// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_deepgram

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	deepgram_internal "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	utils "github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type deepgramSTT struct {
	*deepgram_internal.DeepgramOption
	mu             sync.Mutex
	ctx            context.Context
	ctxCancel      context.CancelFunc
	logger         commons.Logger
	client         *client.WSCallback
	onPacket       func(pkt ...internal_type.Packet) error
	contextId      string
	sttConnectedAt time.Time
	metrics        deepgram_internal.SttSessionMetrics
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

func (*deepgramSTT) Name() string {
	return deepgram_internal.SpeechToTextTransformerName
}

func (dg *deepgramSTT) Initialize() error {
	return nil
}

func NewSpeechToText(opts ...Option) (*deepgramSTT, error) {
	options := &options{ctx: context.Background(), sttOptions: utils.Option{}}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.credential == nil {
		return nil, fmt.Errorf(deepgram_internal.STTCredentialRequiredErrorMessage)
	}
	if options.onPacket == nil {
		return nil, fmt.Errorf(deepgram_internal.STTOnPacketRequiredErrorMessage)
	}

	deepgramOpts, err := deepgram_internal.NewDeepgramOption(
		options.logger,
		options.credential,
		options.sttOptions,
		fmt.Sprintf(deepgram_internal.STTAgentIDTag, options.assistantID),
		fmt.Sprintf(deepgram_internal.STTConversationIDTag, options.conversationID),
	)
	if err != nil {
		options.logger.Errorf(deepgram_internal.STTCredentialFailedLogMessage, err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(options.ctx)
	stt := &deepgramSTT{
		ctx:            ct,
		ctxCancel:      ctxCancel,
		logger:         options.logger,
		DeepgramOption: deepgramOpts,
		onPacket:       options.onPacket,
	}

	start := time.Now()
	dgClient, err := client.NewWSUsingCallback(
		stt.ctx,
		stt.GetKey(),
		stt.ClientOptions(),
		stt.SpeechToTextOptions(),
		deepgram_internal.NewDeepgramSttCallback(
			stt.logger,
			stt.onPacket,
			options.sttOptions,
			stt.getContextID,
			stt.Name(),
			&stt.metrics,
		))
	if err != nil {
		stt.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTError,
						Value:       "1",
						Description: deepgram_internal.STTInitializationFailureMetricDescription,
					}},
					Attributes: observability.Attributes{"provider": stt.Name()},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: fmt.Sprintf(deepgram_internal.STTInitializationErrorLogMessage, err.Error()),
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  stt.Name(),
						"options":   observability.AttributeValue(stt.SpeechToTextOptions()),
					},
					OccurredAt: time.Now(),
				},
			})
		return nil, err
	}
	if !dgClient.Connect() {
		stt.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTError,
						Value:       "1",
						Description: deepgram_internal.STTConnectionFailureMetricDescription,
					}},
					Attributes: observability.Attributes{"provider": stt.Name()},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: deepgram_internal.STTConnectErrorLogMessage,
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  stt.Name(),
						"options":   observability.AttributeValue(stt.SpeechToTextOptions()),
					},
					OccurredAt: time.Now(),
				},
			})
		return nil, fmt.Errorf(deepgram_internal.STTConnectionFailedErrorMessage)
	}

	stt.client = dgClient
	stt.sttConnectedAt = time.Now()
	stt.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSTTInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: deepgram_internal.STTInitializationLatencyMetricDescription,
				}},
				Attributes: observability.Attributes{"provider": stt.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: deepgram_internal.STTInitializationCompletedMessage,
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  stt.Name(),
					"options":   observability.AttributeValue(stt.SpeechToTextOptions()),
				},
				OccurredAt: time.Now(),
			},
		})

	return stt, nil
}

// Transform implements internal_transformer.SpeechToTextTransformer.
// The `Transform` method in the `deepgram` struct is taking an input audio byte array `in`, creating a
// new `bufio.Reader` from it, and then passing that reader to the `Stream` method of the `dg.client`
// WebSocket client. This method is responsible for streaming the audio data to the Deepgram service
// for transcription. If there are any errors during the streaming process, they will be returned by
// the method.
func (dg *deepgramSTT) Transform(ctx context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		dg.mu.Lock()
		dg.contextId = pkt.ContextID
		dg.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		dg.mu.Lock()
		if pkt.ContextID != "" {
			dg.contextId = pkt.ContextID
		}
		dg.mu.Unlock()
		dg.metrics.ResetSpeech()
		return nil
	case internal_type.SpeechToTextEndPacket:
		dg.mu.Lock()
		if pkt.ContextID != "" {
			dg.contextId = pkt.ContextID
		}
		contextID := dg.contextId
		client := dg.client
		dg.mu.Unlock()

		dg.metrics.SetSpeechEndedAt(time.Now())

		if client == nil {
			return fmt.Errorf(deepgram_internal.STTConnectionNotInitializedErrorMessage)
		}
		if err := client.Finalize(); err != nil {
			dg.logger.Errorf(deepgram_internal.STTFinalizeErrorLogMessage, err)
			dg.onPacket(
				internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricSTTError,
							Value:       "1",
							Description: deepgram_internal.STTFinalizeFailureMetricDescription,
						}},
						Attributes: observability.Attributes{"provider": dg.Name()},
					},
				},
				internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf(deepgram_internal.STTFinalizeErrorMessage, err),
					Type:      internal_type.STTNetworkTimeout,
				})
			return fmt.Errorf(deepgram_internal.STTFinalizeErrorMessage, err)
		}
		return nil
	case internal_type.SpeechToTextAudioPacket:
		dg.mu.Lock()
		contextID := dg.contextId
		client := dg.client
		dg.mu.Unlock()

		if client == nil {
			return fmt.Errorf(deepgram_internal.STTConnectionNotInitializedErrorMessage)
		}
		err := client.Stream(bufio.NewReader(bytes.NewReader(pkt.Audio)))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			dg.logger.Errorf(deepgram_internal.STTStreamErrorLogMessage, err)
			dg.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTError,
						Value:       "1",
						Description: deepgram_internal.STTStreamFailureMetricDescription,
					}},
					Attributes: observability.Attributes{"provider": dg.Name()},
				},
			})
			return fmt.Errorf(deepgram_internal.STTStreamErrorMessage, err)
		}
		return err
	default:
		return nil
	}
}

func (dg *deepgramSTT) getContextID() string {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	return dg.contextId
}

func (dg *deepgramSTT) Close(ctx context.Context) error {
	dg.ctxCancel()
	dg.mu.Lock()
	connectedAt := dg.sttConnectedAt
	dg.sttConnectedAt = time.Time{}
	if dg.client != nil {
		dg.client.Stop()
	}
	dg.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		dg.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": dg.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewSTTDurationUsageRecord(dg.Name(), duration, observability.Attributes{}),
			},
		)
	}
	dg.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": dg.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
