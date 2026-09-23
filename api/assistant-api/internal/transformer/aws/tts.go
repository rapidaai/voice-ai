// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_aws

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	aws_internal "github.com/rapidaai/api/assistant-api/internal/transformer/aws/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type awsTTS struct {
	*awsOption
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

	logger     commons.Logger
	onPacket   func(pkt ...internal_type.Packet) error
	normalizer internal_type.TextNormalizer
}

func NewAWSTextToSpeech(ctx context.Context, logger commons.Logger, vaultCredential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	awsOpts, err := NewAWSOption(logger, vaultCredential, opts)
	if err != nil {
		logger.Errorf("aws-tts: initializing aws failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &awsTTS{
		ctx:        ctx2,
		ctxCancel:  contextCancel,
		onPacket:   onPacket,
		logger:     logger,
		awsOption:  awsOpts,
		normalizer: aws_internal.NewAWSNormalizer(logger, opts),
	}, nil
}

func (t *awsTTS) Initialize() error {
	start := time.Now()
	t.stateMu.Lock()
	if err := t.ctx.Err(); err != nil {
		t.stateMu.Unlock()
		return err
	}
	if t.ttsConnectedAt.IsZero() {
		t.ttsConnectedAt = time.Now()
	}
	ctxID := t.contextId
	t.stateMu.Unlock()
	t.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": t.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "aws-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  t.Name(),
					"region":    t.GetRegion(),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (*awsTTS) Name() string {
	return "aws-tts"
}

func (t *awsTTS) synthesize(ctx context.Context, text, contextID string, startedAt time.Time) {
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

	textType := "text"
	if strings.Contains(text, "<break ") {
		textType = "ssml"
		text = fmt.Sprintf("<speak>%s</speak>", text)
	}
	body, err := json.Marshal(map[string]interface{}{
		"Engine": t.GetEngine(), "LanguageCode": t.GetLanguage(),
		"OutputFormat": "pcm", "SampleRate": "16000",
		"Text": text, "TextType": textType, "VoiceId": t.GetVoice(),
	})
	if err != nil {
		synthesisError = fmt.Errorf("aws-tts: encode request: %w", err)
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://polly.%s.amazonaws.com/v1/speech", t.GetRegion()), bytes.NewReader(body))
	if err != nil {
		synthesisError = fmt.Errorf("aws-tts: create request: %w", err)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	t.signPollyRequest(request, body, time.Now().UTC(), t.GetRegion())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		synthesisError = fmt.Errorf("aws-tts: send request: %w", err)
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
		synthesisError = fmt.Errorf("aws-tts: unexpected status code: %d", response.StatusCode)
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
			synthesisError = fmt.Errorf("aws-tts: read response: %w", err)
			return
		}
	}
}

func (t *awsTTS) signPollyRequest(req *http.Request, payload []byte, now time.Time, region string) {
	service := "polly"
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := ttsSha256Hex(payload)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.URL.Host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		"POST", req.URL.Path, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, ttsSha256Hex([]byte(canonicalRequest)))

	signingKey := ttsGetSignatureKey(t.GetSecretAccessKey(), dateStamp, region, service)
	signature := hex.EncodeToString(ttsHmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.GetAccessKeyId(), credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func ttsSha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func ttsHmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func ttsGetSignatureKey(secret, dateStamp, region, service string) []byte {
	kDate := ttsHmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := ttsHmacSHA256(kDate, []byte(region))
	kService := ttsHmacSHA256(kRegion, []byte(service))
	return ttsHmacSHA256(kService, []byte("aws4_request"))
}

func (t *awsTTS) Transform(ctx context.Context, in internal_type.Packet) error {
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
		if t.normalizer != nil {
			input.Text = t.normalizer.Normalize(input.Text)
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
			t.synthesize(requestCtx, text, input.ContextID, startedAt)
		})
		t.stateMu.Unlock()
		return nil
	default:
		t.stateMu.Unlock()
		return fmt.Errorf("aws-tts: unsupported input type %T", in)
	}
}

func (t *awsTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.stateMu.Lock()
	if t.synthesisCancel != nil {
		t.synthesisCancel()
		t.synthesisCancel = nil
	}
	t.textClosed = true
	t.textBuffer.Reset()
	contextID := t.contextId
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
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
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
