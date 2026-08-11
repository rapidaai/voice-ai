// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_cartesia

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	cartesia_internal "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type cartesiaSpeechToText struct {
	*cartesiaOption
	mu      sync.Mutex
	writeMu sync.Mutex
	logger  commons.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	connection     *websocket.Conn
	contextId      string
	sttConnectedAt time.Time
	onPacket       func(pkt ...internal_type.Packet) error

	startedAt time.Time
}

func (*cartesiaSpeechToText) Name() string {
	return "cartesia-stt"
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

	cartesiaOpts, err := NewCartesiaOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("cartesia-stt: intializing cartesia failed %+v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(options.ctx)
	return &cartesiaSpeechToText{
		ctx:            ct,
		ctxCancel:      ctxCancel,
		logger:         options.logger,
		cartesiaOption: cartesiaOpts,
		onPacket:       options.onPacket,
	}, nil
}

func (cst *cartesiaSpeechToText) Initialize() error {
	start := time.Now()
	conn, _, err := websocket.DefaultDialer.Dial(cst.GetSpeechToTextConnectionString(), nil)
	if err != nil {
		cst.logger.Errorf("cartesia-stt: failed to connect to Cartesia WebSocket: %v", err)
		cst.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "cartesia-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(cst.GetSpeechToTextConnectionString()),
					"error":     observability.AttributeValue(err.Error()),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	cst.mu.Lock()
	cst.connection = conn
	cst.sttConnectedAt = time.Now()
	cst.mu.Unlock()

	go cst.readLoop(conn)
	cst.logger.Debugf("cartesia-stt: connection established")

	cst.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewMetricSTTInitLatencyMs(time.Since(start), observability.Attributes{"provider": cst.Name()}),
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "cartesia-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(cst.GetSpeechToTextConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// readLoop owns the WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (cst *cartesiaSpeechToText) readLoop(conn *websocket.Conn) {
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
			cst.connection = nil
			contextID := cst.contextId
			cst.mu.Unlock()

			cst.logger.Errorf("cartesia-stt: connection lost: %v", err)
			cst.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("cartesia-stt: connection lost: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "cartesia-stt: connection lost",
						Attributes: observability.Attributes{
							"component": observability.ComponentSTT.String(),
							"provider":  cst.Name(),
							"error":     observability.AttributeValue(err.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return
		}

		var resp cartesia_internal.SpeechToTextOutput
		if err := json.Unmarshal(msg, &resp); err != nil || resp.Text == "" {
			continue
		}
		cst.mu.Lock()
		ctxID := cst.contextId
		cst.mu.Unlock()

		if !resp.IsFinal {
			cst.onPacket(
				internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
				internal_type.SpeechToTextPacket{
					ContextID: ctxID,
					Script:    resp.Text,
					Language:  resp.Language,
					Interim:   true,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentSTT,
						Event:     observability.STTInterim,
						Attributes: observability.Attributes{
							"type":       "interim",
							"script":     resp.Text,
							"confidence": "0.9000",
						},
						OccurredAt: time.Now(),
					},
				},
			)
		} else {
			now := time.Now()
			cst.mu.Lock()
			startedAt := cst.startedAt
			if !cst.startedAt.IsZero() {
				cst.startedAt = time.Time{}
			}
			cst.mu.Unlock()
			packets := []internal_type.Packet{
				internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
				internal_type.SpeechToTextPacket{
					ContextID: ctxID,
					Script:    resp.Text,
					Language:  resp.Language,
					Interim:   false,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentSTT,
						Event:     observability.STTCompleted,
						Attributes: observability.Attributes{
							"type":       "completed",
							"script":     resp.Text,
							"confidence": "0.9000",
							"language":   resp.Language,
							"word_count": fmt.Sprintf("%d", len(strings.Fields(resp.Text))),
							"char_count": fmt.Sprintf("%d", len(resp.Text)),
						},
						OccurredAt: now,
					},
				},
			}
			if !startedAt.IsZero() {
				packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record:    observability.NewMetricSTTLatencyMs(now.Sub(startedAt), observability.Attributes{"provider": cst.Name()}),
				})
			}
			cst.onPacket(packets...)
		}
	}
}

func (cst *cartesiaSpeechToText) Transform(ctx context.Context, in internal_type.Packet) error {
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
		conn := cst.connection
		contextID := cst.contextId
		cst.mu.Unlock()

		if conn == nil {
			return nil
		}

		cst.writeMu.Lock()
		err := conn.WriteMessage(websocket.BinaryMessage, pkt.Audio)
		cst.writeMu.Unlock()
		if err != nil {
			cst.logger.Errorf("cartesia-stt: error sending audio: %v", err)
			cst.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("cartesia-stt: send failed: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "cartesia-stt: send failed",
						Attributes: observability.Attributes{
							"component": observability.ComponentSTT.String(),
							"provider":  cst.Name(),
							"error":     observability.AttributeValue(err.Error()),
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

func (cst *cartesiaSpeechToText) Close(ctx context.Context) error {
	cst.ctxCancel()
	cst.mu.Lock()
	ctxID := cst.contextId
	connectedAt := cst.sttConnectedAt
	cst.sttConnectedAt = time.Time{}

	if cst.connection != nil {
		conn := cst.connection
		cst.connection = nil // mark before Close so readLoop sees intentional
		conn.Close()
	}
	cst.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		cst.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": cst.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewSTTDurationUsageRecord(cst.Name(), duration, observability.Attributes{}),
			},
		)
	}
	cst.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
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
