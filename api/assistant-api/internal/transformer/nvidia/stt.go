// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_nvidia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"io"
	"net/http"
	"sync"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type nvidiaSTT struct {
	*nvidiaOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	contextId      string
	sttConnectedAt time.Time
	audioBuffer    bytes.Buffer
	startedAt      time.Time

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
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

	nvidiaOpts, err := NewNvidiaOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("nvidia-stt: initializing nvidia failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(options.ctx)
	return &nvidiaSTT{
		ctx:          ctx2,
		ctxCancel:    contextCancel,
		onPacket:     options.onPacket,
		logger:       options.logger,
		nvidiaOption: nvidiaOpts,
	}, nil
}

func (*nvidiaSTT) Name() string {
	return "nvidia-stt"
}

func (st *nvidiaSTT) Initialize() error {
	start := time.Now()
	st.mu.Lock()
	st.sttConnectedAt = time.Now()
	st.mu.Unlock()

	st.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": st.Name()},
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
				Message: "nvidia-stt: initialization completed",
				Attributes: observability.Attributes{
					"component":   observability.ComponentSTT.String(),
					"provider":    st.Name(),
					"function_id": observability.AttributeValue(st.GetFunctionId()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (st *nvidiaSTT) Transform(ctx context.Context, in internal_type.Packet) error {
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
		st.audioBuffer.Write(pkt.Audio)
		audioData := make([]byte, st.audioBuffer.Len())
		copy(audioData, st.audioBuffer.Bytes())
		st.audioBuffer.Reset()
		ctxId := st.contextId
		st.mu.Unlock()

		go st.transcribe(audioData, ctxId)
		return nil
	default:
		return nil
	}
}

func (st *nvidiaSTT) transcribe(audioData []byte, ctxId string) {
	apiURL := fmt.Sprintf("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/%s", st.GetFunctionId())

	payload := map[string]interface{}{
		"audio":         base64.StdEncoding.EncodeToString(audioData),
		"encoding":      "LINEAR_PCM",
		"sample_rate":   16000,
		"language_code": st.GetLanguage(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		st.logger.Errorf("nvidia-stt: error marshalling request: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("nvidia-stt: marshal failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}

	req, err := http.NewRequestWithContext(st.ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		st.logger.Errorf("nvidia-stt: error creating request: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("nvidia-stt: request creation failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}
	req.Header.Set("Authorization", "Bearer "+st.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("NVCF-INPUT-ASSET-REFERENCES", st.GetFunctionId())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("nvidia-stt: error sending request: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("nvidia-stt: request failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		st.logger.Errorf("nvidia-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("nvidia-stt: status %d", resp.StatusCode), Type: internal_type.STTNetworkTimeout})
		return
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("nvidia-stt: error decoding response: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("nvidia-stt: decode failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}

	if result.Text != "" {
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
				Script:    result.Text,
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
}

func (st *nvidiaSTT) Close(ctx context.Context) error {
	st.ctxCancel()
	st.mu.Lock()
	connectedAt := st.sttConnectedAt
	st.sttConnectedAt = time.Time{}
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
