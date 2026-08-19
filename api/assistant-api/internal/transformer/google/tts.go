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
	"strconv"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	google_internal "github.com/rapidaai/api/assistant-api/internal/transformer/google/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// googleTextToSpeech is the main struct handling Google Text-to-Speech functionality.
type googleTextToSpeech struct {
	*googleOption
	mu sync.Mutex // Ensures thread-safe operations.

	ctx       context.Context
	ctxCancel context.CancelFunc

	contextId      string // Tracks context ID for audio synthesis.
	ttsConnectedAt time.Time
	logger         commons.Logger                                        // Logger for debugging and error reporting.
	client         *texttospeech.Client                                  // Google TTS client.
	streamClient   texttospeechpb.TextToSpeech_StreamingSynthesizeClient // Streaming client for real-time TTS.
	onPacket       func(pkt ...internal_type.Packet) error               // Callback for handling audio packets.
	normalizer     internal_type.TextNormalizer                          // Text normalizer for preprocessing.

	// TTS latency tracking
	ttsStartedAt  time.Time
	ttsMetricSent bool
}

// Name returns the name of this transformer implementation.
func (*googleTextToSpeech) Name() string {
	return "google-tts"
}

// NewGoogleTextToSpeech creates a new instance of googleTextToSpeech.
func NewGoogleTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	// Initialize Google TTS options.
	googleOption, err := NewGoogleOption(logger, credential, opts)
	if err != nil {
		// Log and return error if initialization fails.
		logger.Errorf("intializing google failed %+v", err)
		return nil, err
	}

	// Create Google TTS client with options.
	client, err := texttospeech.NewClient(ctx, googleOption.GetClientOptions()...)
	if err != nil {
		// Log and return error if client creation fails.
		logger.Errorf("error while creating client for google tts %+v", err)
		return nil, err
	}

	xctx, contextCancel := context.WithCancel(ctx)
	// Return configured TTS instance.
	return &googleTextToSpeech{
		ctx:       xctx,
		ctxCancel: contextCancel,

		logger:       logger,
		onPacket:     onPacket,
		client:       client,
		googleOption: googleOption,
		normalizer:   google_internal.NewGoogleNormalizer(logger, opts),
	}, nil
}

// Initialize sets up the streaming synthesis functionality.
func (google *googleTextToSpeech) Initialize() error {
	start := time.Now()
	// Start a streaming synthesis session.
	stream, err := google.client.StreamingSynthesize(google.ctx)
	if err != nil {
		google.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "google-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  google.Name(),
					"options":   observability.AttributeValue(google.TextToSpeechOptions()),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to create bidirectional stream: %w", err)
	}

	req := texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.
			StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: google.TextToSpeechOptions(),
		},
	}

	google.mu.Lock()
	if google.streamClient != nil {
		_ = google.streamClient.CloseSend()
	}
	google.streamClient = stream
	currentContextId := google.contextId
	google.mu.Unlock()

	// Send the initial configuration request.
	if err = stream.Send(&req); err != nil {
		google.mu.Lock()
		if google.streamClient == stream {
			google.streamClient = nil
		}
		google.mu.Unlock()
		google.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("google-tts: error while initialization %s", err.Error()),
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  google.Name(),
					"options":   observability.AttributeValue(google.TextToSpeechOptions()),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to send config request: %w", err)
	}

	google.mu.Lock()
	if google.ttsConnectedAt.IsZero() {
		google.ttsConnectedAt = time.Now()
	}
	google.mu.Unlock()

	go google.recvLoop(stream, currentContextId)
	google.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": google.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "google-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  google.Name(),
					"options":   observability.AttributeValue(google.TextToSpeechOptions()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// Transform handles streaming synthesis requests for input text.
func (google *googleTextToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	google.mu.Lock()
	currentCtx := google.contextId
	if in.ContextId() != google.contextId {
		google.contextId = in.ContextId()
		google.ttsStartedAt = time.Time{}
		google.ttsMetricSent = false
	}
	sCli := google.streamClient
	google.mu.Unlock()
	if sCli == nil {
		google.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: in.ContextId(),
			Error:     fmt.Errorf("google-tts: calling transform without initialize"),
			Type:      internal_type.TTSNetworkTimeout,
		})
		return nil
	}

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		if currentCtx != "" {
			google.mu.Lock()
			google.ttsStartedAt = time.Time{}
			google.ttsMetricSent = false
			google.mu.Unlock()
			google.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component:  observability.ComponentTTS,
					Event:      observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"},
					OccurredAt: time.Now(),
				},
			})
			if err := google.Initialize(); err != nil {
				google.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("google-tts: failed to reinitialize stream on context change: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
				return nil
			}
			google.mu.Lock()
			sCli = google.streamClient
			google.mu.Unlock()
		}
		return nil
	case internal_type.TextToSpeechTextPacket:
		google.mu.Lock()
		if google.ttsStartedAt.IsZero() {
			google.ttsStartedAt = time.Now()
		}
		google.mu.Unlock()
		normalized := google.normalizer.Normalize(input.Text)
		if err := sCli.Send(&texttospeechpb.StreamingSynthesizeRequest{
			StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
				Input: &texttospeechpb.StreamingSynthesisInput{
					InputSource: &texttospeechpb.StreamingSynthesisInput_Text{Text: normalized},
				},
			},
		}); err != nil {
			google.logger.Errorf("google-tts: failed to synthesize text: %v", err)
			google.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("google-tts: failed to synthesize text: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		google.onPacket(internal_type.ObservabilityEventRecordPacket{
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
		return nil
	case internal_type.TextToSpeechDonePacket:
		// Signal to the server that no more input will be sent.
		// This triggers server-side EOF → recvLoop emits TextToSpeechEndPacket.
		if err := sCli.CloseSend(); err != nil {
			google.logger.Errorf("google-tts: failed to close send: %v", err)
			google.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("google-tts: failed to close send: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		return nil
	default:
		return fmt.Errorf("google-tts: unsupported input type %T", in)
	}
}

// recvLoop reads audio from the gRPC stream for the lifetime of the synthesis session.
// It exits when the stream ends (EOF, cancellation, or error).
func (g *googleTextToSpeech) recvLoop(streamClient texttospeechpb.TextToSpeech_StreamingSynthesizeClient, initialContextId string) {
	for {
		select {
		case <-g.ctx.Done():
			return
		default:
		}

		resp, err := streamClient.Recv()
		if err != nil {
			if err == io.EOF {
				g.mu.Lock()
				effectiveCtx := g.contextId
				if effectiveCtx == "" {
					effectiveCtx = initialContextId
				}
				g.mu.Unlock()
				g.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: effectiveCtx},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: effectiveCtx,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentTTS,
							Event:      observability.TTSCompleted,
							Attributes: observability.Attributes{"type": "completed"},
							OccurredAt: time.Now(),
						},
					},
				)
				return
			}
			if strings.Contains(err.Error(), "Stream aborted due to long duration elapsed without input sent") {
				g.logger.Debugf("google-tts: stream aborted due to timeout, reinitializing")
				g.mu.Lock()
				effectiveCtx := g.contextId
				if effectiveCtx == "" {
					effectiveCtx = initialContextId
				}
				g.mu.Unlock()
				g.onPacket(internal_type.TextToSpeechEndPacket{ContextID: effectiveCtx})
				go g.Initialize()
				return
			}
			g.mu.Lock()
			effectiveCtx := g.contextId
			if effectiveCtx == "" {
				effectiveCtx = initialContextId
			}
			g.mu.Unlock()
			g.onPacket(internal_type.TextToSpeechEndPacket{ContextID: effectiveCtx})
			g.logger.Errorf("google-tts: error receiving from stream: %v", err)
			return
		}

		if resp == nil {
			continue
		}

		g.mu.Lock()
		currentContextId := g.contextId
		currentStreamClient := g.streamClient
		g.mu.Unlock()

		if currentStreamClient != streamClient {
			g.logger.Debugf("google-tts: interrupted, stream replaced - stopping old callback")
			return
		}

		effectiveContextId := currentContextId
		if effectiveContextId == "" {
			effectiveContextId = initialContextId
		}

		audioContent := resp.GetAudioContent()
		var shouldEmitFirstAudioLatencyMetric bool
		g.mu.Lock()
		ttsStartedAt := g.ttsStartedAt
		if !g.ttsMetricSent && !ttsStartedAt.IsZero() {
			g.ttsMetricSent = true
			shouldEmitFirstAudioLatencyMetric = true
		}
		g.mu.Unlock()
		if shouldEmitFirstAudioLatencyMetric {
			g.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: effectiveContextId,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": g.Name()}),
			})
		}
		if err := g.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: effectiveContextId, AudioChunk: audioContent}); err != nil {
			g.logger.Errorf("google-tts: failed to send packet: %v", err)
		}
	}
}

// Close safely shuts down the TTS client and streaming client.
func (g *googleTextToSpeech) Close(ctx context.Context) error {
	g.ctxCancel()

	g.mu.Lock()
	connectedAt := g.ttsConnectedAt
	g.ttsConnectedAt = time.Time{}
	var combinedErr error
	if g.streamClient != nil {
		// Attempt to close the streaming client.
		if err := g.streamClient.CloseSend(); err != nil {
			// Log the error if closure fails.
			combinedErr = fmt.Errorf("error closing StreamClient: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}

	if g.client != nil {
		// Attempt to close the client.
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
				Record: observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": g.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewTTSDurationUsageRecord(g.Name(), duration, observability.Attributes{}),
			},
		)
	}
	g.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": g.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return combinedErr
}
