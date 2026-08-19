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

	mu             sync.Mutex
	contextId      string
	ttsConnectedAt time.Time
	textBuffer     strings.Builder

	ttsStartedAt  time.Time
	ttsMetricSent bool

	ttsRequestCancel     context.CancelFunc
	ttsRequestGeneration int64

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
	t.mu.Lock()
	if t.ttsConnectedAt.IsZero() {
		t.ttsConnectedAt = time.Now()
	}
	ctxID := t.contextId
	t.mu.Unlock()
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

func (t *awsTTS) flush() {
	t.mu.Lock()
	text := t.textBuffer.String()
	t.textBuffer.Reset()
	ctxID := t.contextId
	if text == "" || ctxID == "" {
		t.mu.Unlock()
		return
	}
	previousRequestCancel := t.ttsRequestCancel
	requestContext, requestCancel := context.WithCancel(t.ctx)
	t.ttsRequestGeneration++
	ttsRequestGeneration := t.ttsRequestGeneration
	t.ttsRequestCancel = requestCancel
	t.mu.Unlock()

	if previousRequestCancel != nil {
		previousRequestCancel()
	}
	go t.synthesize(requestContext, requestCancel, text, ctxID, ttsRequestGeneration)
}

func (t *awsTTS) synthesize(requestContext context.Context, requestCancel context.CancelFunc, text string, ctxID string, ttsRequestGeneration int64) {
	defer func() {
		t.mu.Lock()
		if t.ttsRequestGeneration == ttsRequestGeneration {
			t.ttsRequestCancel = nil
		}
		t.mu.Unlock()
		requestCancel()
	}()

	region := t.GetRegion()
	endpoint := fmt.Sprintf("https://polly.%s.amazonaws.com/v1/speech", region)
	textType := "text"
	textForPolly := text
	if strings.Contains(text, "<break ") {
		textType = "ssml"
		textForPolly = fmt.Sprintf("<speak>%s</speak>", text)
	}

	payload := map[string]interface{}{
		"Engine":       t.GetEngine(),
		"LanguageCode": t.GetLanguage(),
		"OutputFormat": "pcm",
		"SampleRate":   "16000",
		"Text":         textForPolly,
		"TextType":     textType,
		"VoiceId":      t.GetVoice(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("aws-tts: error marshalling request: %v", err)
		synthesisErr := fmt.Errorf("aws-tts: error marshalling request: %w", err)
		t.onPacket(
			internal_type.TextToSpeechErrorPacket{
				ContextID: ctxID,
				Error:     synthesisErr,
				Type:      internal_type.TTSNetworkTimeout,
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-tts: error while synthesizing",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  t.Name(),
						"error":     observability.AttributeValue(synthesisErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	requestTime := time.Now().UTC()
	req, err := http.NewRequestWithContext(requestContext, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		t.logger.Errorf("aws-tts: error creating request: %v", err)
		synthesisErr := fmt.Errorf("aws-tts: error creating request: %w", err)
		t.onPacket(
			internal_type.TextToSpeechErrorPacket{
				ContextID: ctxID,
				Error:     synthesisErr,
				Type:      internal_type.TTSNetworkTimeout,
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-tts: error while synthesizing",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  t.Name(),
						"error":     observability.AttributeValue(synthesisErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	t.signPollyRequest(req, body, requestTime, region)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if requestContext.Err() != nil {
			return
		}
		t.logger.Errorf("aws-tts: error sending request: %v", err)
		synthesisErr := fmt.Errorf("aws-tts: error sending request: %w", err)
		t.onPacket(
			internal_type.TextToSpeechErrorPacket{
				ContextID: ctxID,
				Error:     synthesisErr,
				Type:      internal_type.TTSNetworkTimeout,
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-tts: error while synthesizing",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  t.Name(),
						"error":     observability.AttributeValue(synthesisErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.logger.Errorf("aws-tts: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		synthesisErr := fmt.Errorf("aws-tts: unexpected status code: %d", resp.StatusCode)
		t.onPacket(
			internal_type.TextToSpeechErrorPacket{
				ContextID: ctxID,
				Error:     synthesisErr,
				Type:      internal_type.TTSNetworkTimeout,
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-tts: error while synthesizing",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  t.Name(),
						"error":     observability.AttributeValue(synthesisErr.Error()),
						"response":  observability.AttributeValue(string(respBody)),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	buf := make([]byte, 4096)
	firstChunk := true
	for {
		select {
		case <-requestContext.Done():
			return
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			audioChunk := make([]byte, n)
			copy(audioChunk, buf[:n])

			var shouldEmitFirstAudioLatencyMetric bool
			t.mu.Lock()
			ttsStartedAt := t.ttsStartedAt
			shouldEmitAudioForRequest := t.ttsRequestGeneration == ttsRequestGeneration && t.contextId == ctxID
			if firstChunk {
				firstChunk = false
				if !t.ttsMetricSent && !ttsStartedAt.IsZero() && shouldEmitAudioForRequest {
					t.ttsMetricSent = true
					shouldEmitFirstAudioLatencyMetric = true
				}
			}
			t.mu.Unlock()
			if !shouldEmitAudioForRequest {
				return
			}
			if shouldEmitFirstAudioLatencyMetric {
				t.onPacket(internal_type.ObservabilityMetricRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": t.Name()}),
				})
			}

			t.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxID, AudioChunk: audioChunk})
		}
		if err != nil {
			if err != io.EOF {
				if requestContext.Err() != nil {
					return
				}
				t.logger.Errorf("aws-tts: error reading response body: %v", err)
				synthesisErr := fmt.Errorf("aws-tts: error reading response body: %w", err)
				t.onPacket(
					internal_type.TextToSpeechErrorPacket{
						ContextID: ctxID,
						Error:     synthesisErr,
						Type:      internal_type.TTSNetworkTimeout,
					},
					internal_type.ObservabilityLogRecordPacket{
						ContextID: ctxID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordLog{
							Level:   observability.LevelError,
							Message: "aws-tts: error while synthesizing",
							Attributes: observability.Attributes{
								"component": observability.ComponentTTS.String(),
								"provider":  t.Name(),
								"error":     observability.AttributeValue(synthesisErr.Error()),
							},
							OccurredAt: time.Now(),
						},
					},
				)
			}
			break
		}
	}

	t.mu.Lock()
	shouldEmitCompletionForRequest := t.ttsRequestGeneration == ttsRequestGeneration && t.contextId == ctxID
	t.mu.Unlock()
	if !shouldEmitCompletionForRequest {
		return
	}
	t.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxID},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
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
	incomingContextID := in.ContextId()
	var requestCancelForPreviousContext context.CancelFunc
	t.mu.Lock()
	if incomingContextID != t.contextId {
		requestCancelForPreviousContext = t.ttsRequestCancel
		t.ttsRequestCancel = nil
		t.ttsRequestGeneration++
		t.contextId = incomingContextID
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
		t.textBuffer.Reset()
	}
	t.mu.Unlock()
	if requestCancelForPreviousContext != nil {
		requestCancelForPreviousContext()
	}

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		t.mu.Lock()
		requestCancelForInterrupt := t.ttsRequestCancel
		t.contextId = ""
		t.ttsRequestCancel = nil
		t.ttsRequestGeneration++
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
		t.textBuffer.Reset()
		t.mu.Unlock()
		if requestCancelForInterrupt != nil {
			requestCancelForInterrupt()
		}
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
		return nil
	case internal_type.TextToSpeechTextPacket:
		normalizedText := input.Text
		if t.normalizer != nil {
			normalizedText = t.normalizer.Normalize(input.Text)
		}
		t.mu.Lock()
		if t.ttsStartedAt.IsZero() {
			t.ttsStartedAt = time.Now()
		}
		t.textBuffer.WriteString(normalizedText)
		t.mu.Unlock()
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSSpeaking,
				Attributes: observability.Attributes{
					"type": "speaking",
					"text": normalizedText,
				},
				OccurredAt: time.Now(),
			},
		})
	case internal_type.TextToSpeechDonePacket:
		t.flush()
		return nil
	default:
		return fmt.Errorf("aws-tts: unsupported input type %T", in)
	}
	return nil
}

func (t *awsTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.mu.Lock()
	ctxID := t.contextId
	connectedAt := t.ttsConnectedAt
	requestCancelForClose := t.ttsRequestCancel
	t.ttsRequestCancel = nil
	t.ttsRequestGeneration++
	t.ttsConnectedAt = time.Time{}
	t.mu.Unlock()
	if requestCancelForClose != nil {
		requestCancelForClose()
	}

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
			ContextID: ctxID,
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
