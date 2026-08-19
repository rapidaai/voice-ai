// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	groq_internal "github.com/rapidaai/api/assistant-api/internal/transformer/groq/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type groqSTT struct {
	*groqOption
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

	groqOpts, err := NewGroqOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("groq-stt: initializing groq failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(options.ctx)
	return &groqSTT{
		ctx:        ctx2,
		ctxCancel:  contextCancel,
		onPacket:   options.onPacket,
		logger:     options.logger,
		groqOption: groqOpts,
	}, nil
}

func (*groqSTT) Name() string {
	return "groq-stt"
}

func (st *groqSTT) Initialize() error {
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
				Message: "groq-stt: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"provider":  st.Name(),
					"model":     observability.AttributeValue(st.GetSTTModel()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (st *groqSTT) Transform(ctx context.Context, in internal_type.Packet) error {
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

func (st *groqSTT) transcribe(audioData []byte, ctxId string) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		st.logger.Errorf("groq-stt: error creating form file: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("groq-stt: form file failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}

	// Write WAV header for raw PCM
	wavHeader := createWAVHeader(len(audioData), 16000, 1, 16)
	part.Write(wavHeader)
	part.Write(audioData)

	writer.WriteField("model", st.GetSTTModel())
	writer.WriteField("response_format", "verbose_json")
	writer.WriteField("language", st.GetLanguage())
	writer.Close()

	req, err := http.NewRequestWithContext(st.ctx, "POST", GROQ_STT_URL, &body)
	if err != nil {
		st.logger.Errorf("groq-stt: error creating request: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("groq-stt: request creation failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}
	req.Header.Set("Authorization", "Bearer "+st.GetKey())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("groq-stt: error sending request: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("groq-stt: request failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		st.logger.Errorf("groq-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("groq-stt: status %d", resp.StatusCode), Type: internal_type.STTNetworkTimeout})
		return
	}

	var result groq_internal.GroqTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("groq-stt: error decoding response: %v", err)
		st.onPacket(internal_type.SpeechToTextErrorPacket{ContextID: ctxId, Error: fmt.Errorf("groq-stt: decode failed: %w", err), Type: internal_type.STTNetworkTimeout})
		return
	}

	if result.Text != "" {
		now := time.Now()
		var sttStartedAt time.Time
		st.mu.Lock()
		if !st.startedAt.IsZero() {
			sttStartedAt = st.startedAt
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
		if !sttStartedAt.IsZero() {
			packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxId,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record:    observability.NewMetricSTTLatencyMs(now.Sub(sttStartedAt), observability.Attributes{"provider": st.Name()}),
			})
		}
		st.onPacket(packets...)
	}
}

func createWAVHeader(dataSize, sampleRate, numChannels, bitsPerSample int) []byte {
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	totalSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	header[4] = byte(totalSize)
	header[5] = byte(totalSize >> 8)
	header[6] = byte(totalSize >> 16)
	header[7] = byte(totalSize >> 24)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	header[16] = 16 // chunk size
	header[20] = 1  // PCM format
	header[22] = byte(numChannels)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	header[32] = byte(blockAlign)
	header[34] = byte(bitsPerSample)
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	return header
}

func (st *groqSTT) Close(ctx context.Context) error {
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
