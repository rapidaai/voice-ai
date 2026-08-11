// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_speechmatics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	speechmatics_internal "github.com/rapidaai/api/assistant-api/internal/transformer/speechmatics/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type speechmaticsSTT struct {
	*speechmaticsOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	writeMu        sync.Mutex // serializes all WebSocket writes
	contextId      string
	sttConnectedAt time.Time
	startedAt      time.Time

	logger     commons.Logger
	connection *websocket.Conn
	onPacket   func(pkt ...internal_type.Packet) error
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

	smOpts, err := NewSpeechmaticsOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("speechmatics-stt: initializing speechmatics failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(options.ctx)
	return &speechmaticsSTT{
		ctx:                ctx2,
		ctxCancel:          contextCancel,
		onPacket:           options.onPacket,
		logger:             options.logger,
		speechmaticsOption: smOpts,
	}, nil
}

func (*speechmaticsSTT) Name() string {
	return "speechmatics-stt"
}

func (st *speechmaticsSTT) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+st.GetKey())
	conn, resp, err := websocket.DefaultDialer.Dial(SPEECHMATICS_STT_URL, header)
	if err != nil {
		st.logger.Errorf("speechmatics-stt: error while connecting %s with response %v", err, resp)
		st.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "speechmatics-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"path":      observability.AttributeValue(SPEECHMATICS_STT_URL),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	st.mu.Lock()
	st.connection = conn
	if st.sttConnectedAt.IsZero() {
		st.sttConnectedAt = time.Now()
	}
	st.mu.Unlock()

	transcriptionConfig := map[string]interface{}{
		"language":        st.GetLanguage(),
		"operating_point": "enhanced",
		"enable_partials": true,
		"max_delay":       2.0,
	}
	if op, err := st.mdlOpts.GetString(internal_options.ListenOptionOperatingPoint); err == nil && op != "" {
		transcriptionConfig["operating_point"] = op
	}

	startMsg := map[string]interface{}{
		"message": "StartRecognition",
		"audio_format": map[string]interface{}{
			"type":        "raw",
			"encoding":    "pcm_s16le",
			"sample_rate": 16000,
		},
		"transcription_config": transcriptionConfig,
	}

	st.writeMu.Lock()
	err = conn.WriteJSON(startMsg)
	st.writeMu.Unlock()
	if err != nil {
		conn.Close()
		st.logger.Errorf("speechmatics-stt: error sending start recognition: %v", err)
		st.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "speechmatics-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"path":      observability.AttributeValue(SPEECHMATICS_STT_URL),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	// Speechmatics requires the client to wait for RecognitionStarted before sending audio.
	if err := st.waitForRecognitionStarted(conn); err != nil {
		conn.Close()
		st.logger.Errorf("speechmatics-stt: error waiting for RecognitionStarted: %v", err)
		st.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "speechmatics-stt: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"path":      observability.AttributeValue(SPEECHMATICS_STT_URL),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	go st.readLoop(conn)
	st.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewMetricSTTInitLatencyMs(time.Since(start), observability.Attributes{"provider": st.Name()}),
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "speechmatics-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"path":      observability.AttributeValue(SPEECHMATICS_STT_URL),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// waitForRecognitionStarted reads messages from the WebSocket until it receives
// a RecognitionStarted message or an error. This must be called before the
// readLoop goroutine starts and before any audio is sent.
func (st *speechmaticsSTT) waitForRecognitionStarted(conn *websocket.Conn) error {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{}) // clear deadline for readLoop

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("speechmatics-stt: failed reading RecognitionStarted: %w", err)
		}
		var response speechmatics_internal.SpeechmaticsSTTResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			return fmt.Errorf("speechmatics-stt: failed parsing RecognitionStarted: %w", err)
		}
		if response.Message == "RecognitionStarted" {
			st.logger.Debugf("speechmatics-stt: RecognitionStarted received")
			return nil
		}
		if response.Message == "Error" {
			return fmt.Errorf("speechmatics-stt: server error during init: %s", string(msg))
		}
		st.logger.Debugf("speechmatics-stt: ignoring pre-start message: %s", response.Message)
	}
}

// readLoop owns the WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (st *speechmaticsSTT) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-st.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			st.mu.Lock()
			if st.connection != conn {
				st.mu.Unlock()
				return
			}
			ctxID := st.contextId
			st.connection = nil
			st.mu.Unlock()
			st.logger.Errorf("speechmatics-stt: connection lost: %v", err)
			st.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: ctxID,
				Error:     fmt.Errorf("speechmatics-stt: connection lost: %w", err),
				Type:      internal_type.STTNetworkTimeout,
			})
			return
		}

		var response speechmatics_internal.SpeechmaticsSTTResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			st.logger.Errorf("speechmatics-stt: error parsing response: %v", err)
			continue
		}

		st.mu.Lock()
		ctxId := st.contextId
		st.mu.Unlock()

		switch response.Message {
		case "AddPartialTranscript":
			transcript := response.Metadata.Transcript
			if transcript != "" && ctxId != "" {
				st.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxId, Source: "word"},
					internal_type.SpeechToTextPacket{
						ContextID: ctxId,
						Script:    transcript,
						Interim:   true,
					},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: ctxId,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentSTT,
							Event:      observability.STTInterim,
							Attributes: observability.Attributes{"type": "interim"},
							OccurredAt: time.Now(),
						},
					},
				)
			}
		case "AddTranscript":
			transcript := response.Metadata.Transcript
			if transcript != "" && ctxId != "" {
				now := time.Now()
				var startedAt time.Time
				st.mu.Lock()
				if !st.startedAt.IsZero() {
					startedAt = st.startedAt
					st.startedAt = time.Time{}
				}
				st.mu.Unlock()
				packets := []internal_type.Packet{
					internal_type.InterruptionDetectedPacket{ContextID: ctxId, Source: "word"},
					internal_type.SpeechToTextPacket{
						ContextID: ctxId,
						Script:    transcript,
						Interim:   false,
					},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: ctxId,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentSTT,
							Event:      observability.STTCompleted,
							Attributes: observability.Attributes{"type": "completed"},
							OccurredAt: now,
						},
					},
				}
				if !startedAt.IsZero() {
					packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
						ContextID: ctxId,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record:    observability.NewMetricSTTLatencyMs(time.Since(startedAt), observability.Attributes{"provider": st.Name()}),
					})
				}
				st.onPacket(packets...)
			}
		case "Error":
			st.logger.Errorf("speechmatics-stt: server error: %s", string(msg))
			st.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: ctxId,
				Error:     fmt.Errorf("speechmatics-stt: server error"),
				Type:      internal_type.STTNetworkTimeout,
			})
		case "AudioAdded", "EndOfTranscript", "Info":
			// Acknowledged — no action needed.
		default:
			st.logger.Debugf("speechmatics-stt: unhandled message type: %s", response.Message)
		}
	}
}

func (st *speechmaticsSTT) Transform(ctx context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		st.mu.Lock()
		st.contextId = pkt.ContextID
		st.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		st.mu.Lock()
		if st.startedAt.IsZero() {
			st.startedAt = time.Now()
		}
		st.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		st.mu.Lock()
		if st.startedAt.IsZero() {
			st.startedAt = time.Now()
		}
		st.mu.Unlock()
		st.mu.Lock()
		conn := st.connection
		st.mu.Unlock()

		if conn == nil {
			return nil
		}

		st.writeMu.Lock()
		err := conn.WriteMessage(websocket.BinaryMessage, pkt.Audio)
		st.writeMu.Unlock()
		if err != nil {
			st.logger.Errorf("speechmatics-stt: error sending audio: %v", err)
			st.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: st.contextId,
				Error:     fmt.Errorf("speechmatics-stt: send failed: %w", err),
				Type:      internal_type.STTNetworkTimeout,
			})
			return nil
		}
		return nil
	default:
		return nil
	}
}

func (st *speechmaticsSTT) Close(ctx context.Context) error {
	st.ctxCancel()
	st.mu.Lock()
	connectedAt := st.sttConnectedAt
	st.sttConnectedAt = time.Time{}

	if st.connection != nil {
		conn := st.connection
		st.connection = nil // mark before Close so readLoop sees intentional

		// Send EndOfStream so the server flushes any pending transcripts.
		st.writeMu.Lock()
		_ = conn.WriteJSON(map[string]interface{}{
			"message":     "EndOfStream",
			"last_seq_no": 0,
		})
		st.writeMu.Unlock()

		conn.Close()
	}
	st.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		st.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": st.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewSTTDurationUsageRecord(st.Name(), duration, observability.Attributes{}),
			},
		)
	}
	st.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": st.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
