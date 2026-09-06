// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type textToSpeech struct {
	config *Config
	engine *dslEngine

	ctx    context.Context
	cancel context.CancelFunc

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	stateMu        sync.Mutex
	connectMu      sync.Mutex
	writeMu        sync.Mutex
	connection     *websocket.Conn
	currentContext string
	connectedAt    time.Time
	turnStartedAt  time.Time
	metricEmitted  bool

	resampler         internal_type.AudioResampler
	sourceAudioConfig *protos.AudioConfig
	targetAudioConfig *protos.AudioConfig
}

type readErrorDisposition int

const (
	readErrorIgnore readErrorDisposition = iota
	readErrorComplete
	readErrorFail
)

func NewTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	config, err := NewConfig(credential, opts)
	if err != nil {
		if packetErr := onPacket(internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSError,
				Attributes: observability.Attributes{
					"provider":   "custom-tts-websocket-v1",
					"operation":  "load_config",
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		}); packetErr != nil {
			logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", packetErr)
		}
		return nil, err
	}
	resampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)
	ctx2, cancel := context.WithCancel(ctx)
	return &textToSpeech{
		config:    config,
		engine:    config.newEngine(),
		ctx:       ctx2,
		cancel:    cancel,
		logger:    logger,
		onPacket:  onPacket,
		resampler: resampler,
		sourceAudioConfig: &protos.AudioConfig{
			SampleRate:  uint32(config.SampleRate),
			AudioFormat: parseAudioEncoding(config.Encoding),
			Channels:    1,
		},
		targetAudioConfig: internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
	}, nil
}

func (*textToSpeech) Name() string {
	return "custom-tts-websocket-v1"
}

func (transformer *textToSpeech) Initialize() error {
	return nil
}

func (transformer *textToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		return transformer.handleText(input.ContextID, input.Text)
	case internal_type.TextToSpeechDonePacket:
		return transformer.handleDone(input.ContextID, input.Text)
	case internal_type.TextToSpeechInterruptPacket:
		transformer.handleInterrupt(input.ContextID)
		return nil
	default:
		return fmt.Errorf("custom-tts websocket_v1: unsupported input type %T", in)
	}
}

func (transformer *textToSpeech) Close(ctx context.Context) error {
	transformer.cancel()
	transformer.connectMu.Lock()
	defer transformer.connectMu.Unlock()
	transformer.stateMu.Lock()
	conn := transformer.connection
	contextID := transformer.currentContext
	connectedAt := transformer.connectedAt
	transformer.connection = nil
	transformer.currentContext = ""
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.stateMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		if err := transformer.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": transformer.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewTTSDurationUsageRecord(transformer.Name(), duration, observability.Attributes{}),
			},
		); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
	}
	if err := transformer.onPacket(internal_type.ObservabilityEventRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS,
			Event:     observability.TTSClosed,
			Attributes: observability.Attributes{
				"type":     "closed",
				"provider": transformer.Name(),
			},
			OccurredAt: time.Now(),
		},
	}); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}

	transformer.stateMu.Lock()
	transformer.connectedAt = time.Time{}
	transformer.stateMu.Unlock()

	return nil
}

func (transformer *textToSpeech) handleText(contextID, text string) error {
	scope := transformer.config.newQueryScope(contextID, text)
	conn, err := transformer.getOrOpenConnection(scope)
	if err != nil {
		if transformer.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return nil
		}
		if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-tts websocket_v1: failed to connect: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		}); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
		return nil
	}

	requests, err := transformer.engine.EvaluateRequestRules(
		requestPacketText,
		transformer.config.newRequestScope(requestPacketText, contextID, text),
	)
	if err != nil {
		if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.TTSInvalidInput,
		}); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
		return nil
	}

	transformer.stateMu.Lock()
	if transformer.turnStartedAt.IsZero() {
		transformer.turnStartedAt = time.Now()
	}
	transformer.stateMu.Unlock()

	if err := transformer.writeRequests(conn, requests); err != nil {
		if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-tts websocket_v1: failed to write text request: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		}); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
		transformer.dropConnection(conn)
		return nil
	}

	if err := transformer.onPacket(internal_type.ObservabilityEventRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS,
			Event:     observability.TTSSpeaking,
			Attributes: observability.Attributes{
				"type": "speaking",
				"text": text,
			},
			OccurredAt: time.Now(),
		},
	}); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}

	return nil
}

func (transformer *textToSpeech) handleDone(contextID, text string) error {
	if !transformer.engine.HasRequestRules(requestPacketDone) {
		return nil
	}

	conn, active := transformer.getActiveConnection(contextID)
	if !active || conn == nil {
		return nil
	}

	requests, err := transformer.engine.EvaluateRequestRules(
		requestPacketDone,
		transformer.config.newRequestScope(requestPacketDone, contextID, text),
	)
	if err != nil {
		if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.TTSInvalidInput,
		}); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
		return nil
	}
	if len(requests) == 0 {
		return nil
	}

	if err := transformer.writeRequests(conn, requests); err != nil {
		if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-tts websocket_v1: failed to write done request: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		}); err != nil {
			transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
		}
		transformer.dropConnection(conn)
	}

	return nil
}

func (transformer *textToSpeech) handleInterrupt(contextID string) {
	transformer.connectMu.Lock()
	defer transformer.connectMu.Unlock()

	transformer.stateMu.Lock()
	if transformer.currentContext != contextID {
		transformer.stateMu.Unlock()
		return
	}
	conn := transformer.connection
	transformer.stateMu.Unlock()

	if conn != nil && transformer.engine.HasRequestRules(requestPacketInterrupt) {
		requests, err := transformer.engine.EvaluateRequestRules(
			requestPacketInterrupt,
			transformer.config.newRequestScope(requestPacketInterrupt, contextID, ""),
		)
		if err != nil {
			if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID,
				Error:     err,
				Type:      internal_type.TTSInvalidInput,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
		} else if len(requests) > 0 {
			if err := transformer.writeRequests(conn, requests); err != nil {
				if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("custom-tts websocket_v1: failed to write interrupt request: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				}); err != nil {
					transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
				}
			}
		}
	}

	transformer.stateMu.Lock()
	if transformer.currentContext != contextID || transformer.connection != conn {
		transformer.stateMu.Unlock()
		return
	}
	transformer.connection = nil
	transformer.currentContext = ""
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.stateMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	if err := transformer.onPacket(internal_type.ObservabilityEventRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.RecordEvent{
			Component:  observability.ComponentTTS,
			Event:      observability.TTSInterrupted,
			Attributes: observability.Attributes{"type": "interrupted"},
			OccurredAt: time.Now(),
		},
	}); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}
}

func (transformer *textToSpeech) getOrOpenConnection(scope queryScope) (*websocket.Conn, error) {
	transformer.connectMu.Lock()
	defer transformer.connectMu.Unlock()

	transformer.stateMu.Lock()
	if transformer.connection != nil && transformer.currentContext == scope.MessageID {
		conn := transformer.connection
		transformer.stateMu.Unlock()
		return conn, nil
	}
	oldConn := transformer.connection
	transformer.connection = nil
	transformer.currentContext = ""
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.stateMu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}

	connectionURL, err := transformer.engine.BuildConnectionURL(scope)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	for key, value := range transformer.config.Headers {
		headers.Set(key, value)
	}

	start := time.Now()
	conn, response, err := websocket.DefaultDialer.DialContext(transformer.ctx, connectionURL, headers)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	connectedAt := time.Now()
	transformer.stateMu.Lock()
	transformer.connection = conn
	transformer.currentContext = scope.MessageID
	if transformer.connectedAt.IsZero() {
		transformer.connectedAt = connectedAt
	}
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.stateMu.Unlock()

	go transformer.readLoop(conn, scope.MessageID)
	if err := transformer.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: scope.MessageID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": transformer.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: scope.MessageID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "custom-tts websocket_v1: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  transformer.Name(),
					"path":      observability.AttributeValue(transformer.config.BaseURL),
				},
				OccurredAt: time.Now(),
			},
		},
	); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}

	return conn, nil
}

func (transformer *textToSpeech) getActiveConnection(contextID string) (*websocket.Conn, bool) {
	transformer.stateMu.Lock()
	defer transformer.stateMu.Unlock()
	if transformer.connection == nil || transformer.currentContext != contextID {
		return nil, false
	}
	return transformer.connection, true
}

func (transformer *textToSpeech) writeRequests(conn *websocket.Conn, requests []outboundRequest) error {
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
			// transformer.logger.Debugf("custom-tts websocket_v1: sending text payload: %s", payload)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				return err
			}
		case frameTypeJSON:
			// transformer.logger.Debugf("custom-tts websocket_v1: sending JSON payload: %s", request.Body)
			if err := conn.WriteJSON(request.Body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported request frame %q", request.Frame)
		}
	}

	return nil
}

func (transformer *textToSpeech) readLoop(conn *websocket.Conn, contextID string) {
	for {
		select {
		case <-transformer.ctx.Done():
			return
		default:
		}
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			switch transformer.classifyReadError(conn, err) {
			case readErrorIgnore:
				return
			case readErrorComplete:
				if err := transformer.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: contextID},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: contextID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentTTS,
							Event:      observability.TTSCompleted,
							Attributes: observability.Attributes{"type": "completed"},
							OccurredAt: time.Now(),
						},
					},
				); err != nil {
					transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
				}
				return
			case readErrorFail:
			default:
			}
			if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID,
				Error:     fmt.Errorf("custom-tts websocket_v1: read failed: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
			return
		}
		frame, err := transformer.engine.ParseFrame(messageType, payload)
		if err != nil {
			if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID,
				Error:     err,
				Type:      internal_type.TTSUnknownError,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
			continue
		}

		outcome, err := transformer.engine.EvaluateResponse(frame, contextID)
		if err != nil {
			if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID,
				Error:     err,
				Type:      internal_type.TTSUnknownError,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
			continue
		}
		if !outcome.Matched {
			continue
		}

		resolvedContextID := outcome.MessageID
		if resolvedContextID == "" {
			resolvedContextID = contextID
		}

		if len(outcome.Audio) > 0 {
			audio, err := transformer.normalizeAudioChunk(outcome.Audio)
			if err != nil {
				if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: resolvedContextID,
					Error:     err,
					Type:      internal_type.TTSUnknownError,
				}); err != nil {
					transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
				}
				continue
			}

			transformer.emitFirstAudioMetric(resolvedContextID)
			if err := transformer.onPacket(internal_type.TextToSpeechAudioPacket{
				ContextID:  resolvedContextID,
				AudioChunk: audio,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
		}

		if outcome.ErrorText != "" {
			if err := transformer.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: resolvedContextID,
				Error:     errors.New(outcome.ErrorText),
				Type:      internal_type.TTSUnknownError,
			}); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
		}

		if outcome.Done {
			transformer.dropConnection(conn)
			if err := transformer.onPacket(
				internal_type.TextToSpeechEndPacket{ContextID: resolvedContextID},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: resolvedContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordEvent{
						Component:  observability.ComponentTTS,
						Event:      observability.TTSCompleted,
						Attributes: observability.Attributes{"type": "completed"},
						OccurredAt: time.Now(),
					},
				},
			); err != nil {
				transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
			}
			return
		}
	}
}

func (transformer *textToSpeech) classifyReadError(conn *websocket.Conn, err error) readErrorDisposition {
	transformer.stateMu.Lock()
	active := transformer.connection == conn
	turnStarted := !transformer.turnStartedAt.IsZero()
	if active {
		transformer.connection = nil
		transformer.currentContext = ""
		transformer.turnStartedAt = time.Time{}
		transformer.metricEmitted = false
	}
	transformer.stateMu.Unlock()
	if active && conn != nil {
		_ = conn.Close()
	}

	if !active {
		return readErrorIgnore
	}
	if transformer.ctx.Err() != nil {
		return readErrorIgnore
	}
	if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		if turnStarted {
			return readErrorComplete
		}
		return readErrorFail
	}
	return readErrorFail
}

func (transformer *textToSpeech) dropConnection(conn *websocket.Conn) {
	transformer.stateMu.Lock()
	if transformer.connection == conn {
		transformer.connection = nil
		transformer.currentContext = ""
		transformer.turnStartedAt = time.Time{}
		transformer.metricEmitted = false
	}
	transformer.stateMu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

func (transformer *textToSpeech) emitFirstAudioMetric(contextID string) {
	transformer.stateMu.Lock()
	startedAt := transformer.turnStartedAt
	alreadySent := transformer.metricEmitted
	if !alreadySent && !startedAt.IsZero() {
		transformer.metricEmitted = true
	}
	transformer.stateMu.Unlock()

	if alreadySent || startedAt.IsZero() {
		return
	}

	if err := transformer.onPacket(internal_type.ObservabilityMetricRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record:    observability.NewMetricTTSLatencyMs(time.Since(startedAt), observability.Attributes{"provider": transformer.Name()}),
	}); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}
}

func (transformer *textToSpeech) normalizeAudioChunk(audio []byte) ([]byte, error) {
	if transformer.resampler == nil {
		return audio, nil
	}
	audio, err := transformer.resampler.Resample(audio, transformer.sourceAudioConfig, transformer.targetAudioConfig)
	if err != nil {
		return nil, fmt.Errorf("custom-tts websocket_v1: failed to resample audio: %w", err)
	}
	return audio, nil
}

func parseAudioEncoding(encoding string) protos.AudioConfig_AudioFormat {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "mulaw", "mu-law", "mulaw8", "mu_law", "ulaw", "u-law", "pcmu", "g711_ulaw":
		return protos.AudioConfig_MuLaw8
	default:
		return protos.AudioConfig_LINEAR16
	}
}
