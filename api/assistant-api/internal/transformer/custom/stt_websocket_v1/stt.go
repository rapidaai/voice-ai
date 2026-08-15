// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type speechToText struct {
	config *Config
	engine *dslEngine

	ctx    context.Context
	cancel context.CancelFunc

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	mu         sync.Mutex
	connectMu  sync.Mutex
	writeMu    sync.Mutex
	connection *websocket.Conn

	contextID             string
	connectedAt           time.Time
	interruptionStartedAt time.Time

	resampler         internal_type.AudioResampler
	sourceAudioConfig *protos.AudioConfig
	targetAudioConfig *protos.AudioConfig
}

type readErrorDisposition int

const (
	readErrorIgnore readErrorDisposition = iota
	readErrorFail
)

func NewSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	config, err := NewConfig(credential, opts)
	if err != nil {
		if packetErr := onPacket(internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTError,
				Attributes: observability.Attributes{
					"provider":   "custom-stt-websocket-v1",
					"operation":  "load_config",
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		}); packetErr != nil {
			logger.Errorf("custom-stt websocket_v1: onPacket failed: %v", packetErr)
		}
		return nil, err
	}
	resampler, err := internal_audio_resampler.GetResampler(logger)
	if err != nil {
		return nil, fmt.Errorf("custom-stt websocket_v1: failed to initialize audio resampler: %w", err)
	}
	transformerContext, cancel := context.WithCancel(ctx)
	return &speechToText{
		config:            config,
		engine:            config.newEngine(),
		ctx:               transformerContext,
		cancel:            cancel,
		logger:            logger,
		onPacket:          onPacket,
		resampler:         resampler,
		sourceAudioConfig: internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		targetAudioConfig: &protos.AudioConfig{
			SampleRate:  uint32(config.SampleRate),
			AudioFormat: parseAudioEncoding(config.Encoding),
			Channels:    1,
		},
	}, nil
}

func (*speechToText) Name() string {
	return "custom-stt-websocket-v1"
}

func (transformer *speechToText) Initialize() error {
	start := time.Now()
	if _, err := transformer.getOrOpenConnection(); err != nil {
		return fmt.Errorf("custom-stt websocket_v1: failed to connect: %w", err)
	}

	transformer.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: transformer.currentContextID(),
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": transformer.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSTTInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "STT initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: transformer.currentContextID(),
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "custom-stt websocket_v1: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  transformer.Name(),
					"path":      observability.AttributeValue(transformer.config.BaseURL),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (transformer *speechToText) Transform(_ context.Context, in internal_type.Packet) error {
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		transformer.mu.Lock()
		transformer.contextID = input.ContextID
		transformer.mu.Unlock()
		if err := transformer.handlePacketRequests(requestPacketTurnChange, input.ContextID, nil, true); err != nil {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTNetworkTimeout,
			})
		}
		return nil
	case internal_type.SpeechToTextStartPacket:
		transformer.mu.Lock()
		transformer.interruptionStartedAt = time.Now()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		transformer.mu.Unlock()
		return nil
	case internal_type.SpeechToTextEndPacket:
		transformer.mu.Lock()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		transformer.mu.Unlock()
		if err := transformer.handlePacketRequests(requestPacketInterrupt, input.ContextID, nil, false); err != nil {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTNetworkTimeout,
			})
		}
		return nil
	case internal_type.SpeechToTextAudioPacket:
		if len(input.Audio) == 0 {
			return nil
		}
		return transformer.handleAudio(input.ContextID, input.Audio)
	default:
		return nil
	}
}

func (transformer *speechToText) Close(_ context.Context) error {
	transformer.cancel()
	transformer.connectMu.Lock()
	defer transformer.connectMu.Unlock()

	transformer.mu.Lock()
	conn := transformer.connection
	contextID := transformer.contextID
	connectedAt := transformer.connectedAt
	transformer.connection = nil
	transformer.contextID = ""
	transformer.connectedAt = time.Time{}
	transformer.interruptionStartedAt = time.Time{}
	transformer.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		transformer.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": transformer.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewSTTDurationUsageRecord(transformer.Name(), duration, observability.Attributes{}),
			},
		)
	}
	transformer.onPacket(internal_type.ObservabilityEventRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentSTT,
			Event:     observability.STTClosed,
			Attributes: observability.Attributes{
				"type":     "closed",
				"provider": transformer.Name(),
			},
			OccurredAt: time.Now(),
		},
	})

	return nil
}

func (transformer *speechToText) handleAudio(contextID string, audio []byte) error {
	transformer.mu.Lock()
	if contextID != "" {
		transformer.contextID = contextID
	}
	if transformer.interruptionStartedAt.IsZero() {
		transformer.interruptionStartedAt = time.Now()
	}
	effectiveContextID := transformer.contextID
	transformer.mu.Unlock()

	chunk, err := transformer.prepareAudioChunk(audio)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: effectiveContextID,
			Error:     err,
			Type:      internal_type.STTInvalidInput,
		})
		return nil
	}

	err = transformer.handlePacketRequests(requestPacketAudio, effectiveContextID, chunk, true)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: effectiveContextID,
			Error:     err,
			Type:      internal_type.STTNetworkTimeout,
		})
		return nil
	}

	return nil
}

func (transformer *speechToText) getOrOpenConnection() (*websocket.Conn, error) {
	transformer.connectMu.Lock()
	defer transformer.connectMu.Unlock()

	transformer.mu.Lock()
	if transformer.connection != nil {
		conn := transformer.connection
		transformer.mu.Unlock()
		return conn, nil
	}
	transformer.mu.Unlock()

	connectionURL, err := transformer.engine.BuildConnectionURL(transformer.config.newQueryScope())
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	for key, value := range transformer.config.Headers {
		headers.Set(key, value)
	}

	conn, response, err := websocket.DefaultDialer.DialContext(transformer.ctx, connectionURL, headers)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	transformer.mu.Lock()
	if transformer.connection != nil {
		existing := transformer.connection
		transformer.mu.Unlock()
		_ = conn.Close()
		return existing, nil
	}
	transformer.connection = conn
	if transformer.connectedAt.IsZero() {
		transformer.connectedAt = time.Now()
	}
	transformer.mu.Unlock()

	go transformer.readLoop(conn)
	return conn, nil
}

func (transformer *speechToText) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-transformer.ctx.Done():
			return
		default:
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if transformer.classifyReadError(conn, err) == readErrorFail {
				transformer.onPacket(internal_type.SpeechToTextErrorPacket{
					ContextID: transformer.currentContextID(),
					Error:     fmt.Errorf("custom-stt websocket_v1: read failed: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				})
			}
			return
		}
		frame, err := transformer.engine.ParseFrame(messageType, payload)
		if err != nil {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTSystemPanic,
			})
			continue
		}

		outcome, err := transformer.engine.EvaluateResponse(frame)
		if err != nil {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTSystemPanic,
			})
			continue
		}
		if !outcome.Matched {
			continue
		}
		if strings.TrimSpace(outcome.ErrorText) != "" {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     errors.New(strings.TrimSpace(outcome.ErrorText)),
				Type:      internal_type.STTSystemPanic,
			})
			continue
		}
		if strings.TrimSpace(outcome.Script) == "" {
			continue
		}

		transformer.emitTranscript(outcome)
	}
}

func (transformer *speechToText) classifyReadError(conn *websocket.Conn, err error) readErrorDisposition {
	transformer.mu.Lock()
	active := transformer.connection == conn
	if active {
		transformer.connection = nil
	}
	transformer.mu.Unlock()

	if !active {
		return readErrorIgnore
	}
	if conn != nil {
		_ = conn.Close()
	}
	if transformer.ctx.Err() != nil {
		return readErrorIgnore
	}
	if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return readErrorIgnore
	}
	return readErrorFail
}

func (transformer *speechToText) dropConnection(conn *websocket.Conn) {
	transformer.mu.Lock()
	if transformer.connection == conn {
		transformer.connection = nil
	}
	transformer.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (transformer *speechToText) emitTranscript(outcome responseOutcome) {
	now := time.Now()
	contextID := transformer.currentContextID()
	language := strings.TrimSpace(outcome.Language)
	if language == "" {
		language = transformer.config.Language
	}

	confidenceValue := outcome.Confidence
	confidenceText := fmt.Sprintf("%.4f", confidenceValue)
	eventType := "completed"
	eventName := observability.STTCompleted
	if outcome.Interim {
		eventType = "interim"
		eventName = observability.STTInterim
	}

	eventData := map[string]string{
		"type":       eventType,
		"script":     outcome.Script,
		"confidence": confidenceText,
	}
	if language != "" {
		eventData["language"] = language
	}
	if !outcome.Interim {
		eventData["word_count"] = fmt.Sprintf("%d", len(strings.Fields(outcome.Script)))
		eventData["char_count"] = fmt.Sprintf("%d", len(outcome.Script))
	}

	transformer.onPacket(internal_type.InterruptionDetectedPacket{
		ContextID: contextID,
		Source:    internal_type.InterruptionSourceWord,
	},
		internal_type.SpeechToTextPacket{
			ContextID:  contextID,
			Script:     outcome.Script,
			Concat:     utils.Ptr(""),
			Confidence: confidenceValue,
			Language:   language,
			Interim:    outcome.Interim,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentSTT,
				Event:      eventName,
				Attributes: eventData,
				OccurredAt: now,
			},
		})

	if !outcome.Interim {
		var interruptionStartedAt time.Time
		transformer.mu.Lock()
		if !transformer.interruptionStartedAt.IsZero() {
			interruptionStartedAt = transformer.interruptionStartedAt
			transformer.interruptionStartedAt = time.Time{}
		}
		transformer.mu.Unlock()

		if !interruptionStartedAt.IsZero() {
			transformer.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record:    observability.NewMetricSTTLatencyMs(now.Sub(interruptionStartedAt), observability.Attributes{"provider": transformer.Name()}),
			})
		}
	}
}

func (transformer *speechToText) prepareAudioChunk(audio []byte) ([]byte, error) {
	chunk := audio
	if transformer.resampler != nil {
		resampled, err := transformer.resampler.Resample(chunk, transformer.sourceAudioConfig, transformer.targetAudioConfig)
		if err != nil {
			return nil, fmt.Errorf("custom-stt websocket_v1: failed to resample audio: %w", err)
		}
		chunk = resampled
	}

	return chunk, nil
}

func (transformer *speechToText) handlePacketRequests(packet string, contextID string, audio []byte, openIfNeeded bool) error {
	if !transformer.engine.HasRequestRules(packet) {
		return nil
	}
	if transformer.ctx.Err() != nil {
		return nil
	}

	var (
		conn *websocket.Conn
		err  error
	)

	if openIfNeeded {
		conn, err = transformer.getOrOpenConnection()
		if err != nil {
			if transformer.ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("custom-stt websocket_v1: failed to connect: %w", err)
		}
	} else {
		transformer.mu.Lock()
		conn = transformer.connection
		transformer.mu.Unlock()
		if conn == nil {
			return nil
		}
	}

	scope := transformer.config.newRequestScope(packet, contextID, audio)
	requests, err := transformer.engine.EvaluateRequestRules(packet, scope)
	if err != nil {
		return fmt.Errorf("custom-stt websocket_v1: failed to evaluate %s request rules: %w", packet, err)
	}
	if len(requests) == 0 {
		return nil
	}

	if err := transformer.writeRequests(conn, requests); err != nil {
		transformer.dropConnection(conn)
		if transformer.ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("custom-stt websocket_v1: failed to write %s request: %w", packet, err)
	}

	return nil
}

func (transformer *speechToText) writeRequests(conn *websocket.Conn, requests []outboundRequest) error {
	transformer.writeMu.Lock()
	defer transformer.writeMu.Unlock()

	for _, request := range requests {
		switch request.Frame {
		case frameTypeBinary:
			payload, ok := request.Body.([]byte)
			if !ok {
				return fmt.Errorf("expected binary payload")
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
				return err
			}
		case frameTypeText:
			payload, ok := request.Body.(string)
			if !ok {
				return fmt.Errorf("expected text payload")
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				return err
			}
		case frameTypeJSON:
			if err := conn.WriteJSON(request.Body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported request frame %q", request.Frame)
		}
	}

	return nil
}

func (transformer *speechToText) currentContextID() string {
	transformer.mu.Lock()
	defer transformer.mu.Unlock()
	return transformer.contextID
}

func parseAudioEncoding(encoding string) protos.AudioConfig_AudioFormat {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "mulaw", "mu-law", "mulaw8", "mu_law", "ulaw", "u-law", "pcmu", "g711_ulaw":
		return protos.AudioConfig_MuLaw8
	default:
		return protos.AudioConfig_LINEAR16
	}
}
