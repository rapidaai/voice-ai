// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type speechToText struct {
	config *Config
	engine *dslEngine

	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	mu                 sync.Mutex
	contextID          string
	connectedAt        time.Time
	speechStartedAt    time.Time
	speechAudioBuffer  bytes.Buffer
	activeRequestCount int

	resampler         internal_type.AudioResampler
	sourceAudioConfig *protos.AudioConfig
	targetAudioConfig *protos.AudioConfig
}

func NewSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	config, err := NewConfig(credential, opts)
	if err != nil {
		return nil, err
	}
	resampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)
	transformerContext, cancel := context.WithCancel(ctx)
	return &speechToText{
		config:            config,
		engine:            config.newEngine(),
		ctx:               transformerContext,
		cancel:            cancel,
		httpClient:        &http.Client{Timeout: 60 * time.Second},
		logger:            logger,
		onPacket:          onPacket,
		resampler:         resampler,
		sourceAudioConfig: internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		targetAudioConfig: &protos.AudioConfig{
			SampleRate:  uint32(config.SampleRate),
			AudioFormat: protos.AudioConfig_LINEAR16,
			Channels:    1,
		},
	}, nil
}

func (*speechToText) Name() string {
	return "custom-stt-http-v1"
}

func (transformer *speechToText) Initialize() error {
	start := time.Now()
	transformer.mu.Lock()
	transformer.connectedAt = time.Now()
	contextID := transformer.contextID
	transformer.mu.Unlock()

	transformer.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: contextID,
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
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "custom-stt http_v1: initialization completed",
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
		return nil
	case internal_type.SpeechToTextEndPacket:
		transformer.flushBufferedSpeech(input.ContextID)
		return nil
	case internal_type.SpeechToTextStartPacket:
		transformer.mu.Lock()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		transformer.speechStartedAt = time.Now()
		transformer.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		if len(input.Audio) == 0 {
			return nil
		}
		chunk, err := transformer.prepareAudioChunk(input.Audio)
		if err != nil {
			transformer.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTInvalidInput,
			})
			return nil
		}
		transformer.mu.Lock()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		_, _ = transformer.speechAudioBuffer.Write(chunk)
		transformer.mu.Unlock()
		return nil
	default:
		return nil
	}
}

func (transformer *speechToText) Close(_ context.Context) error {
	transformer.cancel()

	transformer.mu.Lock()
	contextID := transformer.contextID
	connectedAt := transformer.connectedAt
	transformer.contextID = ""
	transformer.connectedAt = time.Time{}
	transformer.speechStartedAt = time.Time{}
	transformer.speechAudioBuffer.Reset()
	transformer.mu.Unlock()

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

func (transformer *speechToText) flushBufferedSpeech(contextID string) {
	transformer.mu.Lock()
	if contextID != "" {
		transformer.contextID = contextID
	}
	effectiveContextID := transformer.contextID
	startedAt := transformer.speechStartedAt
	audioData := make([]byte, transformer.speechAudioBuffer.Len())
	copy(audioData, transformer.speechAudioBuffer.Bytes())
	transformer.speechAudioBuffer.Reset()
	transformer.speechStartedAt = time.Time{}
	if len(audioData) > 0 {
		transformer.activeRequestCount++
	}
	transformer.mu.Unlock()
	if len(audioData) == 0 {
		return
	}
	utils.Go(transformer.ctx, func() {
		defer func() {
			transformer.mu.Lock()
			transformer.activeRequestCount--
			transformer.mu.Unlock()
		}()
		transformer.transcribe(effectiveContextID, audioData, startedAt)
	})
}

func (transformer *speechToText) transcribe(contextID string, pcmAudio []byte, startedAt time.Time) {
	requestURL, err := transformer.engine.BuildRequestURL(transformer.config.newQueryScope())
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	requests, err := transformer.engine.EvaluateRequestRules(
		requestPacketAudio,
		transformer.config.newRequestScope(contextID, pcmAudio),
	)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to evaluate request rules: %w", err),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	if len(requests) == 0 {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: request rules produced no audio request"),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	requestBody := requests[0].Body
	if requests[0].Frame != frameTypeJSON {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: audio request rule send.frame must be json"),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to marshal request body: %w", err),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}

	request, err := http.NewRequestWithContext(transformer.ctx, http.MethodPost, requestURL, bytes.NewReader(requestBodyBytes))
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to create request: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	for key, value := range transformer.config.Headers {
		request.Header.Set(key, value)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := transformer.httpClient.Do(request)
	if err != nil {
		if transformer.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: request failed: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to read response: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		sttErr := fmt.Errorf("custom-stt http_v1: status %d: %s", response.StatusCode, string(responseBody))
		errorType := classifyHTTPStatus(response.StatusCode)
		transformer.onPacket(
			internal_type.ObservabilityLogRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: fmt.Sprintf("stt: %s", sttErr.Error()),
					Attributes: observability.Attributes{
						"component":      observability.ComponentSTT.String(),
						"provider":       transformer.Name(),
						"operation":      "http_transcribe",
						"context_id":     contextID,
						"http_status":    fmt.Sprintf("%d", response.StatusCode),
						"recoverable":    fmt.Sprintf("%t", errorType == internal_type.STTRateLimit || errorType == internal_type.STTNetworkTimeout),
						"stt_error_type": fmt.Sprintf("%d", errorType),
						"error":          fmt.Sprintf("stt: %s", sttErr.Error()),
						"error_type":     fmt.Sprintf("%T", sttErr),
					},
				},
			},
			internal_type.SpeechToTextErrorPacket{
				ContextID: contextID,
				Error:     sttErr,
				Type:      errorType,
			},
		)
		return
	}

	frame, err := transformer.engine.ParseHTTPResponse(responseBody)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	outcome, err := transformer.engine.EvaluateResponse(frame)
	if err != nil {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	if !outcome.Matched {
		return
	}
	if strings.TrimSpace(outcome.ErrorText) != "" {
		transformer.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     errors.New(strings.TrimSpace(outcome.ErrorText)),
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	if strings.TrimSpace(outcome.Script) == "" {
		return
	}

	transformer.emitTranscript(contextID, outcome, startedAt)
}

func (transformer *speechToText) emitTranscript(contextID string, outcome responseOutcome, startedAt time.Time) {
	now := time.Now()
	language := strings.TrimSpace(outcome.Language)
	if language == "" {
		language = transformer.config.Language
	}

	eventData := map[string]string{
		"type":       "completed",
		"script":     outcome.Script,
		"confidence": fmt.Sprintf("%.4f", outcome.Confidence),
		"word_count": fmt.Sprintf("%d", len(strings.Fields(outcome.Script))),
		"char_count": fmt.Sprintf("%d", len(outcome.Script)),
	}
	if language != "" {
		eventData["language"] = language
	}

	packets := []internal_type.Packet{
		internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceWord,
		},
		internal_type.SpeechToTextPacket{
			ContextID:  contextID,
			Script:     outcome.Script,
			Confidence: outcome.Confidence,
			Language:   language,
			Interim:    false,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentSTT,
				Event:      observability.STTCompleted,
				Attributes: eventData,
				OccurredAt: now,
			},
		},
	}
	if !startedAt.IsZero() {
		packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record:    observability.NewMetricSTTLatencyMs(now.Sub(startedAt), observability.Attributes{"provider": transformer.Name()}),
		})
	}

	transformer.onPacket(packets...)
}

func (transformer *speechToText) prepareAudioChunk(audio []byte) ([]byte, error) {
	chunk := audio
	if transformer.resampler != nil {
		resampled, err := transformer.resampler.Resample(chunk, transformer.sourceAudioConfig, transformer.targetAudioConfig)
		if err != nil {
			return nil, fmt.Errorf("custom-stt http_v1: failed to resample audio: %w", err)
		}
		chunk = resampled
	}
	return chunk, nil
}

func (transformer *speechToText) currentContextID() string {
	transformer.mu.Lock()
	defer transformer.mu.Unlock()
	return transformer.contextID
}

func classifyHTTPStatus(statusCode int) internal_type.STTErrorType {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return internal_type.STTAuthentication
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return internal_type.STTInvalidInput
	default:
		return internal_type.STTNetworkTimeout
	}
}
