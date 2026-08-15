// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_smallest

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
	smallest_internal "github.com/rapidaai/api/assistant-api/internal/transformer/smallest/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type smallestSpeechToText struct {
	*smallestOption
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

func (*smallestSpeechToText) Name() string {
	return "smallest-stt"
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

	smallestOpts, err := NewSmallestOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("smallest-stt: intializing smallest failed %+v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(options.ctx)
	return &smallestSpeechToText{
		ctx:            ct,
		ctxCancel:      ctxCancel,
		logger:         options.logger,
		smallestOption: smallestOpts,
		onPacket:       options.onPacket,
	}, nil
}

func (cst *smallestSpeechToText) Initialize() error {
	start := time.Now()
	connectionString := cst.GetSpeechToTextConnectionString()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+cst.GetKey())
	setSourceHeaders(header)

	conn, _, err := websocket.DefaultDialer.Dial(connectionString, header)
	if err != nil {
		err = fmt.Errorf("smallest-stt: dial %s: %w", connectionString, err)
		cst.logger.Errorf("smallest-stt: failed to connect to Smallest WebSocket: %v", err)
		cst.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "smallest-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(connectionString),
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
	cst.logger.Debugf("smallest-stt: connection established")

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
				Message: "smallest-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  cst.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// meanWordConfidence averages per-word confidence scores. Pulse only reports
// confidence at the word level (populated when word_timestamps=true), so
// there is no top-level value to report as-is.
func meanWordConfidence(words []smallest_internal.SpeechToTextWord) (float64, bool) {
	if len(words) == 0 {
		return 0, false
	}
	var sum float64
	for _, w := range words {
		sum += w.Confidence
	}
	return sum / float64(len(words)), true
}

// readLoop owns the WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (cst *smallestSpeechToText) readLoop(conn *websocket.Conn) {
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

			cst.logger.Errorf("smallest-stt: connection lost: %v", err)
			cst.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("smallest-stt: connection lost: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "smallest-stt: connection lost",
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

		var resp smallest_internal.SpeechToTextOutput
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}

		// speech_started / speech_ended are VAD-style onset/offset markers —
		// Rapida's own VAD/EOS stages own that responsibility, so we only log them.
		if resp.Type == "speech_started" || resp.Type == "speech_ended" {
			cst.logger.Debugf("smallest-stt: %s at %.3fs", resp.Type, resp.Timestamp)
			continue
		}

		if resp.Transcript == "" {
			continue
		}

		cst.mu.Lock()
		ctxID := cst.contextId
		cst.mu.Unlock()

		if !resp.IsFinal {
			interimAttrs := observability.Attributes{
				"type":   "interim",
				"script": resp.Transcript,
			}
			if conf, ok := meanWordConfidence(resp.Words); ok {
				interimAttrs["confidence"] = fmt.Sprintf("%.4f", conf)
			}
			cst.onPacket(
				internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
				internal_type.SpeechToTextPacket{
					ContextID: ctxID,
					Script:    resp.Transcript,
					Language:  resp.Language,
					Interim:   true,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component:  observability.ComponentSTT,
						Event:      observability.STTInterim,
						Attributes: interimAttrs,
						OccurredAt: time.Now(),
					},
				},
			)
			continue
		}

		now := time.Now()
		cst.mu.Lock()
		startedAt := cst.startedAt
		if !cst.startedAt.IsZero() {
			cst.startedAt = time.Time{}
		}
		cst.mu.Unlock()

		attrs := observability.Attributes{
			"type":       "completed",
			"script":     resp.Transcript,
			"language":   resp.Language,
			"word_count": fmt.Sprintf("%d", len(strings.Fields(resp.Transcript))),
			"char_count": fmt.Sprintf("%d", len(resp.Transcript)),
		}
		// Populated only when the corresponding feature (word_timestamps,
		// sentence_timestamps, diarize, redact_pii/redact_pci) was requested
		// via listen.* options — see GetSpeechToTextConnectionString.
		if len(resp.Words) > 0 {
			attrs["word_timestamp_count"] = fmt.Sprintf("%d", len(resp.Words))
		}
		// Pulse has no top-level confidence; derive one from per-word
		// confidence when word_timestamps was requested, rather than
		// fabricating a constant.
		if conf, ok := meanWordConfidence(resp.Words); ok {
			attrs["confidence"] = fmt.Sprintf("%.4f", conf)
		}
		if len(resp.Utterances) > 0 {
			attrs["utterance_count"] = fmt.Sprintf("%d", len(resp.Utterances))
		}
		if len(resp.RedactedEntities) > 0 {
			attrs["redacted_entities"] = strings.Join(resp.RedactedEntities, ",")
		}

		packets := []internal_type.Packet{
			internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
			internal_type.SpeechToTextPacket{
				ContextID: ctxID,
				Script:    resp.Transcript,
				Language:  resp.Language,
				Interim:   false,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentSTT,
					Event:      observability.STTCompleted,
					Attributes: attrs,
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

func (cst *smallestSpeechToText) Transform(ctx context.Context, in internal_type.Packet) error {
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
			cst.logger.Errorf("smallest-stt: error sending audio: %v", err)
			cst.onPacket(
				internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("smallest-stt: send failed: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "smallest-stt: send failed",
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

func (cst *smallestSpeechToText) Close(ctx context.Context) error {
	cst.ctxCancel()
	cst.mu.Lock()
	ctxID := cst.contextId
	connectedAt := cst.sttConnectedAt
	cst.sttConnectedAt = time.Time{}

	if cst.connection != nil {
		conn := cst.connection
		cst.connection = nil // mark before Close so readLoop sees intentional
		// Best-effort graceful session end; ignore errors — conn.Close() below
		// tears down the socket regardless.
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"close_stream"}`))
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
