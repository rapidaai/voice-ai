// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_assemblyai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	assemblyai_internal "github.com/rapidaai/api/assistant-api/internal/transformer/assembly-ai/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type assemblyaiSTT struct {
	*assemblyaiOption
	ctx            context.Context
	ctxCancel      context.CancelFunc
	mu             sync.Mutex
	connection     *websocket.Conn
	contextId      string
	logger         commons.Logger
	onPacket       func(pkt ...internal_type.Packet) error
	startedAt      time.Time
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

	ayOptions, err := NewAssemblyaiOption(
		options.logger,
		options.credential,
		options.sttOptions,
	)
	if err != nil {
		options.logger.Errorf("assemblyai-stt: key from credential failed %v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(options.ctx)
	return &assemblyaiSTT{
		ctx:              ct,
		ctxCancel:        ctxCancel,
		logger:           options.logger,
		assemblyaiOption: ayOptions,
		onPacket:         options.onPacket,
	}, nil
}

func (aai *assemblyaiSTT) Name() string {
	return "assemblyai-stt"
}

func (aai *assemblyaiSTT) Initialize() error {
	start := time.Now()
	headers := http.Header{}
	headers.Set("Authorization", aai.GetKey())
	dialer := websocket.Dialer{
		Proxy:            nil,
		HandshakeTimeout: 10 * time.Second,
	}

	connection, _, err := dialer.Dial(aai.GetSpeechToTextConnectionString(), headers)
	if err != nil {
		aai.logger.Errorf("assemblyai-stt: failed to connect to websocket: %v", err)
		aai.onPacket(
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "assemblyai-stt: error while performing connect",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  aai.Name(),
						"path":      observability.AttributeValue(aai.GetSpeechToTextConnectionString()),
					},
					OccurredAt: time.Now(),
				},
			})
		return fmt.Errorf("failed to connect to assemblyai websocket: %w", err)
	}

	aai.mu.Lock()
	aai.connection = connection
	aai.sttConnectedAt = time.Now()
	aai.mu.Unlock()
	go aai.readLoop(connection)

	aai.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewMetricSTTInitLatencyMs(time.Since(start), observability.Attributes{"provider": aai.Name()}),
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "assemblyai-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  aai.Name(),
					"path":      observability.AttributeValue(aai.GetSpeechToTextConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// readLoop owns the WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (aai *assemblyaiSTT) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-aai.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			aai.mu.Lock()
			if aai.connection != conn {
				aai.mu.Unlock()
				return
			}
			// Active connection dropped; next audio packet will be ignored.
			aai.connection = nil
			aai.mu.Unlock()
			aai.logger.Errorf("assemblyai-stt: connection lost: %v", err)
			return
		}

		var transcript assemblyai_internal.TranscriptMessage
		if err := json.Unmarshal(msg, &transcript); err != nil {
			aai.logger.Errorf("assemblyai-stt: error unmarshalling transcript: %v", err)
			continue
		}
		switch transcript.Type {
		case "Turn":
			if len(transcript.Words) == 0 {
				continue
			}

			threshold := 0.0
			if v, err := aai.assemblyaiOption.mdlOpts.GetFloat64(internal_options.ListenOptionThreshold); err == nil {
				threshold = v
			}
			var filteredTranscript string
			var totalConfidence float64
			var wordCount int
			for _, word := range transcript.Words {
				if word.Confidence >= threshold {
					filteredTranscript += word.Text + " "
					totalConfidence += word.Confidence
					wordCount++
				}
			}

			if wordCount == 0 {
				continue
			}

			isInterim := !transcript.EndOfTurn || !transcript.TurnIsFormatted
			confStr := fmt.Sprintf("%.4f", totalConfidence/float64(wordCount))
			aai.mu.Lock()
			ctxID := aai.contextId
			aai.mu.Unlock()
			if isInterim {
				aai.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     filteredTranscript,
						Language:   "en",
						Confidence: totalConfidence / float64(wordCount),
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
								"script":     filteredTranscript,
								"confidence": confStr,
							},
							OccurredAt: time.Now(),
						},
					},
				)
			} else {
				now := time.Now()
				var startedAt time.Time
				aai.mu.Lock()
				if !aai.startedAt.IsZero() {
					startedAt = aai.startedAt
					aai.startedAt = time.Time{}
				}
				aai.mu.Unlock()
				aai.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     filteredTranscript,
						Language:   "en",
						Confidence: totalConfidence / float64(wordCount),
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
								"script":     filteredTranscript,
								"confidence": confStr,
								"language":   "en",
								"word_count": fmt.Sprintf("%d", len(strings.Fields(filteredTranscript))),
								"char_count": fmt.Sprintf("%d", len(filteredTranscript)),
							},
							OccurredAt: now,
						},
					},
				)
				if !startedAt.IsZero() {
					aai.onPacket(
						internal_type.ObservabilityMetricRecordPacket{
							ContextID: ctxID,
							Scope:     internal_type.ObservabilityRecordScopeUserMessage,
							Record:    observability.NewMetricSTTLatencyMs(now.Sub(startedAt), observability.Attributes{"provider": aai.Name()}),
						},
					)
				}
			}

		case "Begin":
			aai.logger.Debugf("assemblyai-stt: received Begin message")
		case "Error":
			aai.mu.Lock()
			ctxID := aai.contextId
			aai.mu.Unlock()
			aai.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: ctxID,
					Error:     fmt.Errorf("assemblyai-stt: error from provider: %s (code %d)", transcript.Error, transcript.ErrorCode),
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: fmt.Sprintf("assemblyai-stt: error while transcribing %s", transcript.Error),
						Attributes: observability.Attributes{
							"component": observability.ComponentSTT.String(),
							"error":     observability.AttributeValue(transcript),
						},
						OccurredAt: time.Now(),
					},
				},
			)

		default:
			aai.logger.Debugf("assemblyai-stt: unhandled message type: %s", transcript.Type)
		}
	}
}

func (aai *assemblyaiSTT) Transform(ctx context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		aai.mu.Lock()
		aai.contextId = pkt.ContextID
		aai.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		aai.mu.Lock()
		if aai.startedAt.IsZero() {
			aai.startedAt = time.Now()
		}
		aai.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		aai.mu.Lock()
		if aai.startedAt.IsZero() {
			aai.startedAt = time.Now()
		}
		connection := aai.connection
		ctxID := aai.contextId
		aai.mu.Unlock()

		if connection == nil {
			return nil
		}
		if err := connection.WriteMessage(websocket.BinaryMessage, pkt.Content()); err != nil {
			aai.logger.Errorf("assemblyai-stt: error sending audio: %v", err)
			aai.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: ctxID,
				Error:     fmt.Errorf("assemblyai-stt: send failed: %w", err),
				Type:      internal_type.STTNetworkTimeout,
			})
			return nil
		}
		return nil
	default:
		return nil
	}
}

func (aai *assemblyaiSTT) Close(ctx context.Context) error {
	aai.ctxCancel()
	aai.mu.Lock()
	ctxID := aai.contextId
	connectedAt := aai.sttConnectedAt
	aai.sttConnectedAt = time.Time{}

	var connection *websocket.Conn
	if aai.connection != nil {
		connection = aai.connection
		aai.connection = nil
	}
	aai.mu.Unlock()
	if connection != nil {
		connection.Close()
	}

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		aai.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": aai.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewSTTDurationUsageRecord(aai.Name(), duration, observability.Attributes{}),
			},
		)
	}
	aai.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": aai.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
