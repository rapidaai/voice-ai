// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_nvidia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type nvidiaTTS struct {
	*nvidiaOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	contextId      string
	ttsConnectedAt time.Time
	textBuffer     strings.Builder

	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewNvidiaTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	nvidiaOpts, err := NewNvidiaOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("nvidia-tts: initializing nvidia failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &nvidiaTTS{
		ctx:          ctx2,
		ctxCancel:    contextCancel,
		onPacket:     onPacket,
		logger:       logger,
		nvidiaOption: nvidiaOpts,
	}, nil
}

func (ct *nvidiaTTS) Initialize() error {
	start := time.Now()
	ct.mu.Lock()
	if ct.ttsConnectedAt.IsZero() {
		ct.ttsConnectedAt = time.Now()
	}
	ct.mu.Unlock()

	ct.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": ct.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "nvidia-tts: initialization completed",
				Attributes: observability.Attributes{
					"component":   observability.ComponentTTS.String(),
					"provider":    ct.Name(),
					"function_id": observability.AttributeValue(ct.GetFunctionId()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (*nvidiaTTS) Name() string {
	return "nvidia-tts"
}

func (t *nvidiaTTS) flush() {
	t.mu.Lock()
	text := t.textBuffer.String()
	t.textBuffer.Reset()
	ctxId := t.contextId
	t.mu.Unlock()

	if text == "" || ctxId == "" {
		return
	}

	go t.streamHTTPTTS(text, ctxId)
}

func (t *nvidiaTTS) streamHTTPTTS(text string, ctxId string) {
	apiURL := fmt.Sprintf("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/%s", t.GetFunctionId())

	payload := map[string]interface{}{
		"text":          text,
		"voice_name":    t.GetVoice(),
		"language_code": t.GetLanguage(),
		"encoding":      "LINEAR_PCM",
		"sample_rate":   16000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("nvidia-tts: error marshalling request: %v", err)
		t.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: ctxId,
			Error:     fmt.Errorf("nvidia-tts: error marshalling request: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		})
		return
	}

	req, err := http.NewRequestWithContext(t.ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		t.logger.Errorf("nvidia-tts: error creating request: %v", err)
		t.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: ctxId,
			Error:     fmt.Errorf("nvidia-tts: error creating request: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		})
		return
	}
	req.Header.Set("Authorization", "Bearer "+t.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("NVCF-INPUT-ASSET-REFERENCES", t.GetFunctionId())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.logger.Errorf("nvidia-tts: error sending request: %v", err)
		t.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: ctxId,
			Error:     fmt.Errorf("nvidia-tts: error sending request: %w", err),
			Type:      internal_type.TTSNetworkTimeout,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.logger.Errorf("nvidia-tts: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		t.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: ctxId,
			Error:     fmt.Errorf("nvidia-tts: unexpected status code: %d", resp.StatusCode),
			Type:      internal_type.TTSNetworkTimeout,
		})
		return
	}

	buf := make([]byte, 4096)
	firstChunk := true
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			audioChunk := make([]byte, n)
			copy(audioChunk, buf[:n])

			if firstChunk {
				firstChunk = false
				var shouldEmitFirstAudioLatencyMetric bool
				t.mu.Lock()
				ttsStartedAt := t.ttsStartedAt
				if !t.ttsMetricSent && !ttsStartedAt.IsZero() {
					t.ttsMetricSent = true
					shouldEmitFirstAudioLatencyMetric = true
				}
				t.mu.Unlock()
				if shouldEmitFirstAudioLatencyMetric {
					t.onPacket(internal_type.ObservabilityMetricRecordPacket{
						ContextID: ctxId,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": t.Name()}),
					})
				}
			}

			t.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxId, AudioChunk: audioChunk})
		}
		if err != nil {
			if err != io.EOF {
				t.logger.Errorf("nvidia-tts: error reading response body: %v", err)
				t.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: ctxId,
					Error:     fmt.Errorf("nvidia-tts: error reading response body: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
			}
			break
		}
	}

	t.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxId},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxId,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSCompleted,
				Attributes: observability.Attributes{"type": "completed"},
				OccurredAt: time.Now(),
			},
		},
	)
}

func (t *nvidiaTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	t.mu.Lock()
	currentCtx := t.contextId
	if in.ContextId() != t.contextId {
		t.contextId = in.ContextId()
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
		t.textBuffer.Reset()
	}
	t.mu.Unlock()

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		if currentCtx != "" {
			t.mu.Lock()
			t.ttsStartedAt = time.Time{}
			t.ttsMetricSent = false
			t.textBuffer.Reset()
			t.mu.Unlock()
			t.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentTTS,
					Event:      observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil
	case internal_type.TextToSpeechTextPacket:
		t.mu.Lock()
		if t.ttsStartedAt.IsZero() {
			t.ttsStartedAt = time.Now()
		}
		t.textBuffer.WriteString(input.Text)
		t.mu.Unlock()
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSSpeaking,
				Attributes: observability.Attributes{
					"type": "speaking",
					"text": input.Text,
				},
				OccurredAt: time.Now(),
			},
		})
	case internal_type.TextToSpeechDonePacket:
		t.flush()
		return nil
	default:
		return fmt.Errorf("nvidia-tts: unsupported input type %T", in)
	}
	return nil
}

func (t *nvidiaTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.mu.Lock()
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}
	t.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		t.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": t.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewTTSDurationUsageRecord(t.Name(), duration, observability.Attributes{}),
			},
		)
	}
	t.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": t.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
