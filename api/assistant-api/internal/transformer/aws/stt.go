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
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type awsSTT struct {
	*awsOption
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

	awsOpts, err := NewAWSOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("aws-stt: initializing aws failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(options.ctx)
	return &awsSTT{
		ctx:       ctx2,
		ctxCancel: contextCancel,
		onPacket:  options.onPacket,
		logger:    options.logger,
		awsOption: awsOpts,
	}, nil
}

func (*awsSTT) Name() string {
	return "aws-stt"
}

func (st *awsSTT) Initialize() error {
	start := time.Now()
	st.mu.Lock()
	st.sttConnectedAt = time.Now()
	ctxID := st.contextId
	st.mu.Unlock()
	st.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
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
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "aws-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"region":    st.GetRegion(),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (st *awsSTT) Transform(ctx context.Context, in internal_type.Packet) error {
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
		startedAt := st.startedAt
		st.startedAt = time.Time{}
		st.audioBuffer.Write(pkt.Audio)
		audioData := make([]byte, st.audioBuffer.Len())
		copy(audioData, st.audioBuffer.Bytes())
		st.audioBuffer.Reset()
		ctxID := st.contextId
		st.mu.Unlock()

		go st.transcribe(audioData, ctxID, startedAt)
		return nil
	default:
		return nil
	}
}

func (st *awsSTT) transcribe(audioData []byte, ctxID string, startedAt time.Time) {
	region := st.GetRegion()
	language := st.GetLanguage()
	endpoint := fmt.Sprintf("https://transcribe.%s.amazonaws.com", region)

	payload := map[string]interface{}{
		"AudioStream": map[string]interface{}{
			"AudioEvent": map[string]interface{}{
				"AudioChunk": audioData,
			},
		},
		"LanguageCode":         language,
		"MediaEncoding":        "pcm",
		"MediaSampleRateHertz": 16000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		st.logger.Errorf("aws-stt: error marshalling request: %v", err)
		transcribeErr := fmt.Errorf("aws-stt: marshal failed: %w", err)
		st.onPacket(
			internal_type.SpeechToTextErrorPacket{ContextID: ctxID, Error: transcribeErr, Type: internal_type.STTNetworkTimeout},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-stt: error while transcribing",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  st.Name(),
						"error":     observability.AttributeValue(transcribeErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	requestTime := time.Now().UTC()
	req, err := http.NewRequestWithContext(st.ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		st.logger.Errorf("aws-stt: error creating request: %v", err)
		transcribeErr := fmt.Errorf("aws-stt: request creation failed: %w", err)
		st.onPacket(
			internal_type.SpeechToTextErrorPacket{ContextID: ctxID, Error: transcribeErr, Type: internal_type.STTNetworkTimeout},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-stt: error while transcribing",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  st.Name(),
						"error":     observability.AttributeValue(transcribeErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "com.amazonaws.transcribe.Transcribe.StartTranscriptionJob")

	st.signRequest(req, body, requestTime, region, "transcribe")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("aws-stt: error sending request: %v", err)
		transcribeErr := fmt.Errorf("aws-stt: request failed: %w", err)
		st.onPacket(
			internal_type.SpeechToTextErrorPacket{ContextID: ctxID, Error: transcribeErr, Type: internal_type.STTNetworkTimeout},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-stt: error while transcribing",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  st.Name(),
						"error":     observability.AttributeValue(transcribeErr.Error()),
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
		st.logger.Errorf("aws-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		transcribeErr := fmt.Errorf("aws-stt: status %d", resp.StatusCode)
		st.onPacket(
			internal_type.SpeechToTextErrorPacket{ContextID: ctxID, Error: transcribeErr, Type: internal_type.STTNetworkTimeout},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-stt: error while transcribing",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  st.Name(),
						"error":     observability.AttributeValue(transcribeErr.Error()),
						"response":  observability.AttributeValue(string(respBody)),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	var result struct {
		Results struct {
			Transcripts []struct {
				Transcript string `json:"Transcript"`
			} `json:"Transcripts"`
		} `json:"Results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("aws-stt: error decoding response: %v", err)
		transcribeErr := fmt.Errorf("aws-stt: decode failed: %w", err)
		st.onPacket(
			internal_type.SpeechToTextErrorPacket{ContextID: ctxID, Error: transcribeErr, Type: internal_type.STTNetworkTimeout},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "aws-stt: error while transcribing",
					Attributes: observability.Attributes{
						"component": observability.ComponentSTT.String(),
						"provider":  st.Name(),
						"error":     observability.AttributeValue(transcribeErr.Error()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	transcript := ""
	if len(result.Results.Transcripts) > 0 {
		transcript = result.Results.Transcripts[0].Transcript
	}

	if transcript != "" {
		completedAt := time.Now()
		st.onPacket(
			internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
			internal_type.SpeechToTextPacket{
				ContextID: ctxID,
				Script:    transcript,
				Language:  language,
				Interim:   false,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentSTT,
					Event:     observability.STTCompleted,
					Attributes: observability.Attributes{
						"type":       "completed",
						"script":     transcript,
						"language":   language,
						"word_count": fmt.Sprintf("%d", len(strings.Fields(transcript))),
						"char_count": fmt.Sprintf("%d", len(transcript)),
					},
					OccurredAt: completedAt,
				},
			},
		)
		if !startedAt.IsZero() {
			st.onPacket(
				internal_type.ObservabilityMetricRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record:    observability.NewMetricSTTLatencyMs(completedAt.Sub(startedAt), observability.Attributes{"provider": st.Name()}),
				},
			)
		}
	}
}

func (st *awsSTT) signRequest(req *http.Request, payload []byte, now time.Time, region, service string) {
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := sha256Hex(payload)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.URL.Host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		"POST", req.URL.Path, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)))

	signingKey := getSignatureKey(st.GetSecretAccessKey(), dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		st.GetAccessKeyId(), credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func (st *awsSTT) Close(ctx context.Context) error {
	st.ctxCancel()
	st.mu.Lock()
	ctxID := st.contextId
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
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
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
