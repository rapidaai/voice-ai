// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_transformer_sarvam

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	sarvam_internal "github.com/rapidaai/api/assistant-api/internal/transformer/sarvam/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type sarvamSpeechToText struct {
	*sarvamOption

	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	connection     *websocket.Conn
	startedAt      time.Time
	contextId      string
	sttConnectedAt time.Time

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func (*sarvamSpeechToText) Name() string {
	return "sarvam-stt"
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

	sarvamOpts, err := NewSarvamOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("sarvam-stt: failed to initialize options: %v", err)
		return nil, err
	}

	ct, ctxCancel := context.WithCancel(options.ctx)
	return &sarvamSpeechToText{
		ctx:          ct,
		ctxCancel:    ctxCancel,
		logger:       options.logger,
		sarvamOption: sarvamOpts,
		onPacket:     options.onPacket,
	}, nil
}

func (cst *sarvamSpeechToText) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Api-Subscription-Key", cst.GetKey())
	connectionString := cst.speechToTextUrl()
	conn, _, err := websocket.DefaultDialer.Dial(connectionString, header)
	if err != nil {
		cst.logger.Errorf("sarvam-stt: dial failed: %v", err)
		cst.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "sarvam-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("sarvam-stt: dial failed: %w", err)
	}

	cst.mu.Lock()
	cst.connection = conn
	if cst.sttConnectedAt.IsZero() {
		cst.sttConnectedAt = time.Now()
	}
	cst.mu.Unlock()

	go cst.readLoop(conn)

	cst.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": cst.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSTTInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "STT initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "sarvam-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// readLoop owns a single WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (cst *sarvamSpeechToText) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-cst.ctx.Done():
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			cst.mu.Lock()
			if cst.connection != conn {
				cst.mu.Unlock()
				return
			}
			ctxID := cst.contextId
			cst.connection = nil
			cst.mu.Unlock()
			cst.logger.Errorf("sarvam-stt: connection lost: %v", err)
			cst.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: ctxID,
				Error:     fmt.Errorf("sarvam-stt: connection lost: %w", err),
				Type:      internal_type.STTNetworkTimeout,
			})
			return
		}

		var response sarvam_internal.SarvamSpeechToTextResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			cst.logger.Errorf("sarvam-stt: failed to parse message: %v", err)
			continue
		}

		switch response.Type {
		case "data":
			cst.handleTranscription(response)
		case "error":
			cst.handleServerError(response)
		case "events":
			cst.logger.Infof("sarvam-stt: vad event: %s", string(response.Data))
		default:
			cst.logger.Warnf("sarvam-stt: unknown message type: %s", response.Type)
		}
	}
}

func (cst *sarvamSpeechToText) handleTranscription(response sarvam_internal.SarvamSpeechToTextResponse) {
	transcriptionData, err := response.AsTranscription()
	if err != nil {
		cst.logger.Errorf("sarvam-stt: invalid transcription payload: %v", err)
		return
	}

	now := time.Now()
	var startedAt time.Time
	cst.mu.Lock()
	if !cst.startedAt.IsZero() {
		startedAt = cst.startedAt
		cst.startedAt = time.Time{}
	}
	ctxID := cst.contextId
	cst.mu.Unlock()

	langCode := ""
	if transcriptionData.LanguageCode != nil {
		langCode = *transcriptionData.LanguageCode
	}

	packets := []internal_type.Packet{
		internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
		internal_type.SpeechToTextPacket{
			ContextID:  ctxID,
			Script:     transcriptionData.Transcript,
			Confidence: 0.9,
			Language:   langCode,
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
					"script":     transcriptionData.Transcript,
					"confidence": "0.9000",
					"language":   langCode,
					"word_count": fmt.Sprintf("%d", len(strings.Fields(transcriptionData.Transcript))),
					"char_count": fmt.Sprintf("%d", len(transcriptionData.Transcript)),
				},
				OccurredAt: now,
			},
		},
	}
	if !startedAt.IsZero() {
		packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record:    observability.NewMetricSTTLatencyMs(time.Since(startedAt), observability.Attributes{"provider": cst.Name()}),
		})
	}
	cst.onPacket(packets...)
}

func (cst *sarvamSpeechToText) handleServerError(response sarvam_internal.SarvamSpeechToTextResponse) {
	errorData, err := response.AsError()
	if err != nil {
		cst.logger.Errorf("sarvam-stt: could not parse error payload: %v", err)
		return
	}
	cst.logger.Errorf("sarvam-stt: server error code=%s message=%s", errorData.Code, errorData.Error)
	cst.onPacket(internal_type.SpeechToTextErrorPacket{
		ContextID: cst.contextId,
		Error:     fmt.Errorf("sarvam-stt: server error: %s (code=%s)", errorData.Error, errorData.Code),
		Type:      internal_type.STTNetworkTimeout,
	})
}

func (cst *sarvamSpeechToText) Transform(ctx context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		cst.mu.Lock()
		cst.contextId = pkt.ContextID
		cst.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		cst.mu.Lock()
		if cst.startedAt.IsZero() {
			cst.startedAt = time.Now()
		}
		cst.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		cst.mu.Lock()
		if cst.startedAt.IsZero() {
			cst.startedAt = time.Now()
		}
		cst.mu.Unlock()
		vl, err := cst.speechToTextMessage(pkt.Audio)
		if err != nil {
			return fmt.Errorf("sarvam-stt: failed to encode audio: %w", err)
		}
		cst.mu.Lock()
		connection := cst.connection
		ctxID := cst.contextId
		if connection == nil {
			cst.mu.Unlock()
			return nil
		}
		err = connection.WriteMessage(websocket.TextMessage, vl)
		cst.mu.Unlock()
		if err != nil {
			cst.logger.Errorf("sarvam-stt: error sending audio: %v", err)
			cst.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: ctxID,
				Error:     fmt.Errorf("sarvam-stt: send failed: %w", err),
				Type:      internal_type.STTSystemPanic,
			})
			return nil
		}
		return nil
	default:
		return nil
	}
}

func (cst *sarvamSpeechToText) Close(ctx context.Context) error {
	cst.ctxCancel()
	cst.mu.Lock()
	connectedAt := cst.sttConnectedAt
	cst.sttConnectedAt = time.Time{}
	if cst.connection != nil {
		conn := cst.connection
		cst.connection = nil // mark nil before Close so readLoop sees intentional
		conn.Close()
	}
	cst.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		cst.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": cst.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewSTTDurationUsageRecord(cst.Name(), duration, observability.Attributes{}),
			},
		)
	}
	cst.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": cst.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
