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
	"golang.org/x/sync/semaphore"
)

type googleTextToSpeech struct {
	*googleOption
	stateMu   sync.Mutex
	writeLock *semaphore.Weighted
	workers   sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc

	synthesis      *googleSynthesis
	ttsConnectedAt time.Time
	logger         commons.Logger
	client         *texttospeech.Client
	onPacket       func(pkt ...internal_type.Packet) error
	normalizer     internal_type.TextNormalizer
}

// A retired synthesis retains its ID to reject late input; cancel is nil after retirement.
type googleSynthesis struct {
	contextID  string
	stream     texttospeechpb.TextToSpeech_StreamingSynthesizeClient
	cancel     context.CancelFunc
	startedAt  time.Time
	textClosed bool
}

func (*googleTextToSpeech) Name() string { return "google-tts" }

func NewGoogleTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error, opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	googleOption, err := NewGoogleOption(logger, credential, opts)
	if err != nil {
		return nil, err
	}
	client, err := texttospeech.NewClient(ctx, googleOption.GetClientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("google-tts: create client: %w", err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	return &googleTextToSpeech{
		ctx: sessionCtx, ctxCancel: cancel, writeLock: semaphore.NewWeighted(1),
		logger: logger, onPacket: onPacket, client: client, googleOption: googleOption,
		normalizer: google_internal.NewGoogleNormalizer(logger, opts),
	}, nil
}

// Initialize starts session accounting. Synthesis RPCs open only when text arrives.
func (g *googleTextToSpeech) Initialize() error {
	g.stateMu.Lock()
	defer g.stateMu.Unlock()
	if err := g.ctx.Err(); err != nil {
		return err
	}
	if g.ttsConnectedAt.IsZero() {
		g.ttsConnectedAt = time.Now()
	}
	return nil
}

func (g *googleTextToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.ctx.Err(); err != nil {
		return err
	}
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		return nil
	case internal_type.TextToSpeechInterruptPacket:
		g.stateMu.Lock()
		if g.synthesis == nil || g.synthesis.contextID != input.ContextID || g.synthesis.cancel == nil {
			g.stateMu.Unlock()
			return nil
		}
		cancel := g.synthesis.cancel
		g.synthesis.cancel = nil
		g.stateMu.Unlock()
		cancel()
		g.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
			},
		})
		return nil
	}

	writeCtx, cancelWrite := context.WithCancel(ctx)
	defer cancelWrite()
	stopSession := context.AfterFunc(g.ctx, cancelWrite)
	defer stopSession()
	if err := g.writeLock.Acquire(writeCtx, 1); err != nil {
		return err
	}
	var packets []internal_type.Packet
	defer func() {
		g.writeLock.Release(1)
		if len(packets) > 0 {
			g.onPacket(packets...)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.ctx.Err(); err != nil {
		return err
	}

	g.stateMu.Lock()
	var streamCtx context.Context
	var previousCancel context.CancelFunc
	var request *texttospeechpb.StreamingSynthesizeRequest
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		if input.ContextID == "" || input.Text == "" {
			g.stateMu.Unlock()
			return nil
		}
		if g.synthesis == nil || g.synthesis.contextID != input.ContextID {
			if g.synthesis != nil {
				previousCancel = g.synthesis.cancel
				g.synthesis.cancel = nil
			}
			var cancel context.CancelFunc
			streamCtx, cancel = context.WithCancel(g.ctx)
			g.synthesis = &googleSynthesis{contextID: input.ContextID, cancel: cancel, startedAt: time.Now()}
		} else if g.synthesis.cancel == nil || g.synthesis.textClosed {
			g.stateMu.Unlock()
			return nil
		}
		request = &texttospeechpb.StreamingSynthesizeRequest{
			StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
				Input: &texttospeechpb.StreamingSynthesisInput{
					InputSource: &texttospeechpb.StreamingSynthesisInput_Text{Text: g.normalizer.Normalize(input.Text)},
				},
			},
		}
	case internal_type.TextToSpeechDonePacket:
		if g.synthesis == nil || g.synthesis.contextID != input.ContextID || g.synthesis.cancel == nil || g.synthesis.textClosed {
			g.stateMu.Unlock()
			return nil
		}
		g.synthesis.textClosed = true
	default:
		g.stateMu.Unlock()
		return fmt.Errorf("google-tts: unsupported packet type %T", in)
	}
	synthesis := g.synthesis
	cancel := synthesis.cancel
	g.stateMu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}

	writeCanceled := make(chan struct{})
	stopWrite := context.AfterFunc(ctx, func() {
		g.stateMu.Lock()
		synthesis.cancel = nil
		g.stateMu.Unlock()
		cancel()
		close(writeCanceled)
	})
	defer func() {
		if !stopWrite() {
			<-writeCanceled
		}
	}()

	var err error
	if streamCtx != nil {
		start := time.Now()
		synthesis.stream, err = g.client.StreamingSynthesize(streamCtx)
		if err == nil {
			err = synthesis.stream.Send(&texttospeechpb.StreamingSynthesizeRequest{
				StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_StreamingConfig{
					StreamingConfig: g.TextToSpeechOptions(),
				},
			})
		}
		if err == nil {
			g.workers.Go(func() { g.recvLoop(synthesis) })
			packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name: observability.MetricTTSInitLatencyMs, Value: strconv.FormatInt(time.Since(start).Milliseconds(), 10),
						Description: "TTS initialization latency in milliseconds",
					}},
					Attributes: observability.Attributes{"provider": g.Name()},
				},
			})
		}
	}
	if err == nil {
		if request != nil {
			err = synthesis.stream.Send(request)
		} else {
			err = synthesis.stream.CloseSend()
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if g.ctx.Err() != nil {
		return g.ctx.Err()
	}
	if err != nil {
		g.stateMu.Lock()
		emitError := synthesis.cancel != nil
		synthesis.cancel = nil
		g.stateMu.Unlock()
		cancel()
		if emitError {
			packets = append(packets, internal_type.TextToSpeechErrorPacket{
				ContextID: synthesis.contextID, Error: fmt.Errorf("google-tts: send: %w", err), Type: internal_type.TTSNetworkTimeout,
			})
		}
		return nil
	}
	if input, ok := in.(internal_type.TextToSpeechTextPacket); ok {
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text}, OccurredAt: time.Now(),
			},
		})
	}
	return nil
}

func (g *googleTextToSpeech) recvLoop(synthesis *googleSynthesis) {
	for {
		response, err := synthesis.stream.Recv()
		waitForClose := false
		if err == io.EOF {
			g.stateMu.Lock()
			waitForClose = g.synthesis == synthesis && synthesis.cancel != nil && synthesis.textClosed
			g.stateMu.Unlock()
		}
		if waitForClose {
			// CloseSend and Send share one writer. Observe their outcome before accepting EOF.
			if err := g.writeLock.Acquire(g.ctx, 1); err != nil {
				return
			}
		}
		g.stateMu.Lock()
		if g.synthesis != synthesis || synthesis.cancel == nil {
			g.stateMu.Unlock()
			if waitForClose {
				g.writeLock.Release(1)
			}
			return
		}
		if err != nil {
			completed := err == io.EOF && waitForClose
			cancel := synthesis.cancel
			synthesis.cancel = nil
			emitTerminal := g.ctx.Err() == nil
			g.stateMu.Unlock()
			if waitForClose {
				g.writeLock.Release(1)
			}
			cancel()
			if !emitTerminal {
				return
			}
			if completed {
				g.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: synthesis.contextID},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: synthesis.contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentTTS, Event: observability.TTSCompleted,
							Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
						},
					},
				)
			} else {
				g.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: synthesis.contextID, Error: fmt.Errorf("google-tts: receive: %w", err), Type: internal_type.TTSNetworkTimeout,
				})
			}
			return
		}
		if g.ctx.Err() != nil || len(response.GetAudioContent()) == 0 {
			g.stateMu.Unlock()
			continue
		}
		packets := []internal_type.Packet{}
		if !synthesis.startedAt.IsZero() {
			packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
				ContextID: synthesis.contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMetricTTSLatencyMs(time.Since(synthesis.startedAt), observability.Attributes{"provider": g.Name()}),
			})
			synthesis.startedAt = time.Time{}
		}
		packets = append(packets, internal_type.TextToSpeechAudioPacket{ContextID: synthesis.contextID, AudioChunk: response.GetAudioContent()})
		g.stateMu.Unlock()
		g.onPacket(packets...)
	}
}

func (g *googleTextToSpeech) Close(ctx context.Context) error {
	g.ctxCancel()
	_ = g.writeLock.Acquire(context.WithoutCancel(ctx), 1)
	g.stateMu.Lock()
	connectedAt := g.ttsConnectedAt
	g.ttsConnectedAt = time.Time{}
	if g.synthesis != nil && g.synthesis.cancel != nil {
		g.synthesis.cancel()
		g.synthesis.cancel = nil
	}
	g.stateMu.Unlock()
	var err error
	if g.client != nil {
		err = g.client.Close()
		g.client = nil
	}
	g.writeLock.Release(1)
	g.workers.Wait()
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
	g.onPacket(internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS, Event: observability.TTSClosed,
			Attributes: observability.Attributes{"type": "closed", "provider": g.Name()}, OccurredAt: time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("google-tts: close client: %w", err)
	}
	return nil
}
