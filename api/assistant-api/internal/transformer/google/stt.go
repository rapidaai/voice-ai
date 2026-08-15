// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_google

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type googleSpeechToText struct {
	*googleOption
	mu sync.Mutex

	logger commons.Logger

	client        *speech.Client
	stream        speechpb.Speech_StreamingRecognizeClient
	streamFactory func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error)
	onPacket      func(pkt ...internal_type.Packet) error

	// context management
	ctx       context.Context
	ctxCancel context.CancelFunc

	// observability: time when speech started
	startedAt      time.Time
	contextId      string
	sttConnectedAt time.Time
}

// Name implements internal_transformer.SpeechToTextTransformer.
func (g *googleSpeechToText) Name() string {
	return "google-stt"
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

	start := time.Now()
	googleOption, err := NewGoogleOption(options.logger, options.credential, options.sttOptions)
	if err != nil {
		options.logger.Errorf("google-stt: Error while GoogleOption err: %v", err)
		return nil, err
	}
	client, err := speech.NewClient(options.ctx, googleOption.GetSpeechToTextClientOptions()...)

	if err != nil {
		options.logger.Errorf("google-stt: Error creating Google client: %v", err)
		return nil, err
	}

	xctx, contextCancel := context.WithCancel(options.ctx)
	// Context for callback management
	options.logger.Benchmark("google.NewSpeechToText", time.Since(start))
	g := &googleSpeechToText{
		ctx:          xctx,
		ctxCancel:    contextCancel,
		logger:       options.logger,
		client:       client,
		googleOption: googleOption,
		onPacket:     options.onPacket,
	}
	g.streamFactory = func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error) {
		return client.StreamingRecognize(ctx)
	}
	return g, nil
}

// Transform implements internal_transformer.SpeechToTextTransformer.
func (google *googleSpeechToText) Transform(c context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		google.mu.Lock()
		google.contextId = pkt.ContextID
		google.mu.Unlock()
		return nil
	case internal_type.SpeechToTextStartPacket:
		google.mu.Lock()
		if google.startedAt.IsZero() {
			google.startedAt = time.Now()
		}
		google.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		google.mu.Lock()
		if google.startedAt.IsZero() {
			google.startedAt = time.Now()
		}
		google.mu.Unlock()
		google.mu.Lock()
		strm := google.stream
		google.mu.Unlock()

		if strm == nil {
			google.logger.Infof("google-stt: stream not available, re-initializing")
			start := time.Now()
			google.mu.Lock()
			if err := google.initializeStreamLocked(); err != nil {
				google.mu.Unlock()
				google.onPacket(internal_type.ObservabilityLogRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: fmt.Sprintf("google-stt: error while initialization %s", err.Error()),
						Attributes: observability.Attributes{
							"component":  observability.ComponentSTT.String(),
							"provider":   google.Name(),
							"recognizer": observability.AttributeValue(google.GetRecognizer()),
							"options":    observability.AttributeValue(google.SpeechToTextOptions()),
						},
						OccurredAt: time.Now(),
					},
				})
				google.onPacket(internal_type.SpeechToTextErrorPacket{
					ContextID: google.contextId,
					Error:     fmt.Errorf("google-stt: re-initialize failed: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				})
				return nil
			}
			if google.sttConnectedAt.IsZero() {
				google.sttConnectedAt = time.Now()
			}
			strm = google.stream
			google.mu.Unlock()
			google.onPacket(
				internal_type.ObservabilityMetricRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{"provider": google.Name()},
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
						Message: "google-stt: initialization completed",
						Attributes: observability.Attributes{
							"component":  observability.ComponentSTT.String(),
							"provider":   google.Name(),
							"recognizer": observability.AttributeValue(google.GetRecognizer()),
							"options":    observability.AttributeValue(google.SpeechToTextOptions()),
						},
						OccurredAt: time.Now(),
					},
				})
			if strm == nil {
				return nil
			}
		}

		if err := strm.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
				Audio: pkt.Audio,
			},
		}); err != nil {
			google.logger.Errorf("google-stt: error sending audio: %v", err)
			google.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: google.contextId,
				Error:     fmt.Errorf("google-stt: send failed: %w", err),
				Type:      internal_type.STTNetworkTimeout,
			})
			return nil
		}
		return nil
	default:
		return nil
	}
}

// recvLoop reads responses from the gRPC stream for the lifetime of the STT session.
// It exits when the stream ends (EOF, cancellation, or error).
func (g *googleSpeechToText) recvLoop(stream speechpb.Speech_StreamingRecognizeClient) {
	for {
		select {
		case <-g.ctx.Done():
			return
		default:
		}

		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			g.logger.Errorf("google-stt: recv error: %v", err)

			start := time.Now()
			g.mu.Lock()
			g.stream = nil
			if reinitErr := g.initializeStreamLocked(); reinitErr != nil {
				g.mu.Unlock()
				g.logger.Errorf("google-stt: re-initialize failed: %v", reinitErr)
				g.onPacket(internal_type.ObservabilityLogRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: fmt.Sprintf("google-stt: error while initialization %s", reinitErr.Error()),
						Attributes: observability.Attributes{
							"component":  observability.ComponentSTT.String(),
							"provider":   g.Name(),
							"recognizer": observability.AttributeValue(g.GetRecognizer()),
							"options":    observability.AttributeValue(g.SpeechToTextOptions()),
						},
						OccurredAt: time.Now(),
					},
				})
				g.onPacket(internal_type.SpeechToTextErrorPacket{
					ContextID: g.contextId,
					Error:     fmt.Errorf("google-stt: stream error: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				})
				return
			}
			if g.sttConnectedAt.IsZero() {
				g.sttConnectedAt = time.Now()
			}
			g.mu.Unlock()
			g.onPacket(
				internal_type.ObservabilityMetricRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{"provider": g.Name()},
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
						Message: "google-stt: initialization completed",
						Attributes: observability.Attributes{
							"component":  observability.ComponentSTT.String(),
							"provider":   g.Name(),
							"recognizer": observability.AttributeValue(g.GetRecognizer()),
							"options":    observability.AttributeValue(g.SpeechToTextOptions()),
						},
						OccurredAt: time.Now(),
					},
				})
			g.logger.Infof("google-stt: stream re-initialized after error")
			// New recvLoop was started by initializeStreamLocked, exit this one
			return
		}
		if resp == nil {
			continue
		}

		for _, result := range resp.Results {
			if len(result.Alternatives) == 0 {
				continue
			}
			g.mu.Lock()
			ctxID := g.contextId
			g.mu.Unlock()
			alt := result.Alternatives[0]
			if len(alt.GetTranscript()) == 0 {
				continue
			}
			confStr := fmt.Sprintf("%.4f", float64(alt.GetConfidence()))
			transcript := alt.GetTranscript()

			if result.GetIsFinal() {
				if v, err := g.mdlOpts.GetFloat64(internal_options.ListenOptionThreshold); err == nil {
					if alt.GetConfidence() < float32(v) {
						g.onPacket(
							internal_type.ObservabilityEventRecordPacket{
								ContextID: ctxID,
								Scope:     internal_type.ObservabilityRecordScopeUserMessage,
								Record: observability.RecordEvent{
									Component: observability.ComponentSTT,
									Event:     observability.STTLowConfidence,
									Attributes: observability.Attributes{
										"type":       "low_confidence",
										"script":     transcript,
										"confidence": confStr,
										"threshold":  fmt.Sprintf("%.4f", v),
									},
									OccurredAt: time.Now(),
								},
							},
						)
						continue
					}
				}
				now := time.Now()
				var sttStartedAt time.Time
				g.mu.Lock()
				if !g.startedAt.IsZero() {
					sttStartedAt = g.startedAt
					g.startedAt = time.Time{}
				}
				g.mu.Unlock()
				packets := []internal_type.Packet{
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     transcript,
						Confidence: float64(alt.GetConfidence()),
						Language:   result.GetLanguageCode(),
						Interim:    false,
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
								"confidence": confStr,
								"language":   result.GetLanguageCode(),
								"word_count": fmt.Sprintf("%d", len(strings.Fields(transcript))),
								"char_count": fmt.Sprintf("%d", len(transcript)),
							},
							OccurredAt: now,
						},
					},
				}
				if !sttStartedAt.IsZero() {
					packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
						ContextID: ctxID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record:    observability.NewMetricSTTLatencyMs(now.Sub(sttStartedAt), observability.Attributes{"provider": g.Name()}),
					})
				}
				g.onPacket(packets...)
			} else {
				g.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     transcript,
						Confidence: float64(result.GetStability()),
						Language:   result.GetLanguageCode(),
						Interim:    true,
					},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: ctxID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentSTT,
							Event:     observability.STTInterim,
							Attributes: observability.Attributes{
								"type":       "interim",
								"script":     transcript,
								"confidence": confStr,
							},
							OccurredAt: time.Now(),
						},
					},
				)
			}
		}
	}
}

func (google *googleSpeechToText) Initialize() error {
	start := time.Now()
	google.mu.Lock()
	err := google.initializeStreamLocked()
	if err == nil {
		google.sttConnectedAt = time.Now()
	}
	google.mu.Unlock()
	if err != nil {
		google.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("google-stt: error while initialization %s", err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentSTT.String(),
					"provider":   google.Name(),
					"recognizer": observability.AttributeValue(google.GetRecognizer()),
					"options":    observability.AttributeValue(google.SpeechToTextOptions()),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}
	google.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": google.Name()},
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
				Message: "google-stt: initialization completed",
				Attributes: observability.Attributes{
					"component":  observability.ComponentSTT.String(),
					"provider":   google.Name(),
					"recognizer": observability.AttributeValue(google.GetRecognizer()),
					"options":    observability.AttributeValue(google.SpeechToTextOptions()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// initializeStreamLocked opens a new StreamingRecognize gRPC stream, sends the
// config, and starts recvLoop. Caller MUST hold google.mu.
func (google *googleSpeechToText) initializeStreamLocked() error {
	stream, err := google.streamFactory(google.ctx)
	if err != nil {
		google.logger.Errorf("google-stt: error creating google-stt stream: %v", err)
		return err
	}

	if google.stream != nil {
		_ = google.stream.CloseSend()
	}
	google.stream = stream

	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		Recognizer: google.GetRecognizer(),
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: google.SpeechToTextOptions(),
		},
	}); err != nil {
		google.logger.Errorf("google-stt: error sending config: %v", err)
		google.stream = nil
		return err
	}

	go google.recvLoop(stream)
	google.logger.Debugf("google-stt: connection established")
	return nil
}

func (g *googleSpeechToText) Close(ctx context.Context) error {
	g.ctxCancel()

	g.mu.Lock()
	connectedAt := g.sttConnectedAt
	g.sttConnectedAt = time.Time{}

	var combinedErr error
	if g.stream != nil {
		if err := g.stream.CloseSend(); err != nil {
			combinedErr = fmt.Errorf("error closing StreamClient: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}

	if g.client != nil {
		if err := g.client.Close(); err != nil {
			// Log the error if closure fails.
			combinedErr = fmt.Errorf("error closing Client: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}
	g.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		g.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricSTTDuration(duration, observability.Attributes{"provider": g.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewSTTDurationUsageRecord(g.Name(), duration, observability.Attributes{}),
			},
		)
	}
	g.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentSTT,
				Event:     observability.STTClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": g.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return combinedErr
}
