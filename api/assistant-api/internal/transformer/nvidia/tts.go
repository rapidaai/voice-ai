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
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type nvidiaTTS struct {
	*nvidiaOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	stateMu         sync.Mutex
	workers         sync.WaitGroup
	synthesisCancel context.CancelFunc
	textClosed      bool
	contextId       string
	ttsConnectedAt  time.Time
	textBuffer      strings.Builder

	ttsStartedAt time.Time

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewNvidiaTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	nvidiaOpts, err := NewNvidiaOption(logger, credential, opts)
	if err != nil {
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
	ct.stateMu.Lock()
	if err := ct.ctx.Err(); err != nil {
		ct.stateMu.Unlock()
		return err
	}
	if ct.ttsConnectedAt.IsZero() {
		ct.ttsConnectedAt = time.Now()
	}
	ct.stateMu.Unlock()

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

func (t *nvidiaTTS) streamHTTPTTS(ctx context.Context, text, contextID string, startedAt time.Time) {
	var synthesisError error
	// Response cleanup runs before the message's terminal packet is published.
	defer func() {
		t.stateMu.Lock()
		if t.contextId != contextID {
			t.stateMu.Unlock()
			return
		}
		// Retire under the lock; later interruption cannot revoke this terminal decision.
		t.synthesisCancel = nil
		if ctx.Err() != nil || t.ctx.Err() != nil {
			t.stateMu.Unlock()
			return
		}
		t.stateMu.Unlock()
		if synthesisError != nil {
			t.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID, Error: synthesisError, Type: internal_type.TTSNetworkTimeout,
			})
			return
		}
		t.onPacket(
			internal_type.TextToSpeechEndPacket{ContextID: contextID},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSCompleted,
					Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
				},
			},
		)
	}()

	body, err := json.Marshal(map[string]interface{}{
		"text": text, "voice_name": t.GetVoice(), "language_code": t.GetLanguage(),
		"encoding": "LINEAR_PCM", "sample_rate": 16000,
	})
	if err != nil {
		synthesisError = fmt.Errorf("nvidia-tts: encode request: %w", err)
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/%s", t.GetFunctionId()), bytes.NewReader(body))
	if err != nil {
		synthesisError = fmt.Errorf("nvidia-tts: create request: %w", err)
		return
	}
	request.Header.Set("Authorization", "Bearer "+t.GetKey())
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("NVCF-INPUT-ASSET-REFERENCES", t.GetFunctionId())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		synthesisError = fmt.Errorf("nvidia-tts: send request: %w", err)
		return
	}
	canceled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = response.Body.Close(); close(canceled) })
	defer func() {
		if !stop() {
			<-canceled
			return
		}
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		synthesisError = fmt.Errorf("nvidia-tts: unexpected status code: %d", response.StatusCode)
		return
	}

	buffer := make([]byte, 4096)
	firstChunk := true
	for {
		if ctx.Err() != nil || t.ctx.Err() != nil {
			return
		}
		n, err := response.Body.Read(buffer)
		if ctx.Err() != nil || t.ctx.Err() != nil {
			return
		}
		if n > 0 {
			packets := []internal_type.Packet{}
			if firstChunk {
				firstChunk = false
				packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.NewMetricTTSLatencyMs(time.Since(startedAt), observability.Attributes{"provider": t.Name()}),
				})
			}
			packets = append(packets, internal_type.TextToSpeechAudioPacket{ContextID: contextID, AudioChunk: bytes.Clone(buffer[:n])})
			t.onPacket(packets...)
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			synthesisError = fmt.Errorf("nvidia-tts: read response: %w", err)
			return
		}
	}
}

func (t *nvidiaTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.stateMu.Lock()
	if err := t.ctx.Err(); err != nil {
		t.stateMu.Unlock()
		return err
	}
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		t.stateMu.Unlock()
		return nil
	case internal_type.TextToSpeechInterruptPacket:
		if input.ContextID == "" || input.ContextID != t.contextId || (t.textClosed && t.synthesisCancel == nil) {
			t.stateMu.Unlock()
			return nil
		}
		if t.synthesisCancel != nil {
			t.synthesisCancel()
			t.synthesisCancel = nil
		}
		t.textClosed = true
		t.textBuffer.Reset()
		t.stateMu.Unlock()
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
			},
		})
		return nil
	case internal_type.TextToSpeechTextPacket:
		if input.ContextID == "" || input.Text == "" || (input.ContextID == t.contextId && t.textClosed) {
			t.stateMu.Unlock()
			return nil
		}
		if input.ContextID != t.contextId {
			if t.synthesisCancel != nil {
				t.synthesisCancel()
				t.synthesisCancel = nil
			}
			t.contextId = input.ContextID
			t.textClosed = false
			t.textBuffer.Reset()
			t.ttsStartedAt = time.Now()
		}
		t.textBuffer.WriteString(input.Text)
		t.stateMu.Unlock()
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text}, OccurredAt: time.Now(),
			},
		})
		return nil
	case internal_type.TextToSpeechDonePacket:
		if input.ContextID != t.contextId || t.textClosed || t.textBuffer.Len() == 0 {
			t.stateMu.Unlock()
			return nil
		}
		requestCtx, cancel := context.WithCancel(ctx)
		stopSession := context.AfterFunc(t.ctx, cancel)
		t.synthesisCancel = cancel
		t.textClosed = true
		text := t.textBuffer.String()
		t.textBuffer.Reset()
		startedAt := t.ttsStartedAt
		// Register the worker before Close can pass the state lock and wait for shutdown.
		t.workers.Go(func() {
			defer cancel()
			defer stopSession()
			t.streamHTTPTTS(requestCtx, text, input.ContextID, startedAt)
		})
		t.stateMu.Unlock()
		return nil
	default:
		t.stateMu.Unlock()
		return fmt.Errorf("nvidia-tts: unsupported input type %T", in)
	}
}

func (t *nvidiaTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.stateMu.Lock()
	if t.synthesisCancel != nil {
		t.synthesisCancel()
		t.synthesisCancel = nil
	}
	t.textClosed = true
	t.textBuffer.Reset()
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}
	t.stateMu.Unlock()
	t.workers.Wait()

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
