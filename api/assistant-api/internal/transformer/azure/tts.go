// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	azure_internal "github.com/rapidaai/api/assistant-api/internal/transformer/azure/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/semaphore"
)

type azureTextToSpeech struct {
	*azureOption
	stateMu   sync.Mutex
	writeLock *semaphore.Weighted
	ctx       context.Context
	ctxCancel context.CancelFunc

	synthesis      *azureSynthesis
	ttsConnectedAt time.Time
	client         azureSynthesisClient
	onPacket       func(pkt ...internal_type.Packet) error
	normalizer     internal_type.TextNormalizer
}

type azureSynthesis struct {
	contextID string
	startedAt time.Time
	finished  bool
	cancel    context.CancelFunc
}

// These contracts isolate native SDK handles from message ownership.
type azureSynthesisClient interface {
	StartSpeaking(string, bool) (azureSynthesisStream, error)
	StopSpeaking() error
	Close()
}

type azureSynthesisStream interface {
	Read([]byte) (int, error)
	GetStatus() (common.StreamStatus, error)
	Close()
}

func NewAzureTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error, opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	azureOption, err := NewAzureOption(logger, credential, opts)
	if err != nil {
		return nil, err
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	return &azureTextToSpeech{
		ctx: sessionCtx, ctxCancel: cancel, writeLock: semaphore.NewWeighted(1),
		azureOption: azureOption, onPacket: onPacket,
		normalizer: azure_internal.NewAzureNormalizer(logger, opts),
	}, nil
}

func (*azureTextToSpeech) Name() string { return "azure-tts" }

func (azure *azureTextToSpeech) Initialize() error {
	if err := azure.writeLock.Acquire(azure.ctx, 1); err != nil {
		return err
	}
	var packets []internal_type.Packet
	defer func() {
		azure.writeLock.Release(1)
		if len(packets) > 0 {
			azure.onPacket(packets...)
		}
	}()
	if err := azure.ctx.Err(); err != nil {
		return err
	}
	if azure.client != nil {
		return nil
	}
	start := time.Now()
	speechConfig, err := azure.TextToSpeechOption()
	if err != nil {
		return fmt.Errorf("azure-tts: speech configuration: %w", err)
	}
	defer speechConfig.Close()
	client, err := speech.NewSpeechSynthesizerFromConfig(speechConfig, nil)
	if err != nil {
		return fmt.Errorf("azure-tts: create synthesizer: %w", err)
	}
	if err := azure.ctx.Err(); err != nil {
		client.Close()
		return err
	}
	azure.client = &azureSDKClient{SpeechSynthesizer: client}
	azure.ttsConnectedAt = time.Now()
	packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name: observability.MetricTTSInitLatencyMs, Value: strconv.FormatInt(time.Since(start).Milliseconds(), 10),
				Description: "TTS initialization latency in milliseconds",
			}},
			Attributes: observability.Attributes{"provider": azure.Name()},
		},
	})
	return nil
}

func (azure *azureTextToSpeech) Transform(ctx context.Context, in internal_type.Packet) (returnErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := azure.ctx.Err(); err != nil {
		return err
	}
	var synthesis *azureSynthesis
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		return nil
	case internal_type.TextToSpeechTextPacket:
		if input.ContextID == "" || input.Text == "" {
			return nil
		}
		azure.stateMu.Lock()
		var cancel context.CancelFunc
		if azure.synthesis == nil || azure.synthesis.contextID != input.ContextID {
			if azure.synthesis != nil {
				azure.synthesis.finished = true
				cancel = azure.synthesis.cancel
			}
			azure.synthesis = &azureSynthesis{contextID: input.ContextID, startedAt: time.Now()}
		}
		synthesis = azure.synthesis
		azure.stateMu.Unlock()
		if cancel != nil {
			cancel()
		}
	case internal_type.TextToSpeechInterruptPacket:
		azure.stateMu.Lock()
		if azure.synthesis == nil || azure.synthesis.contextID != input.ContextID || azure.synthesis.finished {
			azure.stateMu.Unlock()
			return nil
		}
		azure.synthesis.finished = true
		cancel := azure.synthesis.cancel
		azure.stateMu.Unlock()
		if cancel != nil {
			cancel()
		}
		azure.onPacket(internal_type.ObservabilityEventRecordPacket{
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
	stopSession := context.AfterFunc(azure.ctx, cancelWrite)
	defer stopSession()
	if err := azure.writeLock.Acquire(writeCtx, 1); err != nil {
		if synthesis != nil {
			azure.stateMu.Lock()
			synthesis.finished = true
			cancel := synthesis.cancel
			azure.stateMu.Unlock()
			if cancel != nil {
				cancel()
			}
		}
		return err
	}
	var packets []internal_type.Packet
	defer func() {
		azure.writeLock.Release(1)
		if len(packets) > 0 {
			azure.onPacket(packets...)
		}
	}()
	if err := ctx.Err(); err != nil {
		if synthesis != nil {
			azure.stateMu.Lock()
			synthesis.finished = true
			azure.stateMu.Unlock()
		}
		return err
	}
	if err := azure.ctx.Err(); err != nil {
		return err
	}

	azure.stateMu.Lock()
	var text string
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		if azure.synthesis != synthesis || synthesis.finished {
			azure.stateMu.Unlock()
			return nil
		}
		text = input.Text
		if azure.normalizer != nil {
			text = azure.normalizer.Normalize(text)
		}
	case internal_type.TextToSpeechDonePacket:
		if azure.synthesis == nil || azure.synthesis.contextID != input.ContextID || azure.synthesis.finished {
			azure.stateMu.Unlock()
			return nil
		}
		// The writer permit ensures every preceding request has drained successfully.
		azure.synthesis.finished = true
		azure.stateMu.Unlock()
		packets = append(packets,
			internal_type.TextToSpeechEndPacket{ContextID: input.ContextID},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSCompleted,
					Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
				},
			},
		)
		return nil
	default:
		azure.stateMu.Unlock()
		return fmt.Errorf("azure-tts: unsupported packet type %T", in)
	}
	operationCtx, cancel := context.WithCancel(azure.ctx)
	synthesis.cancel = cancel
	azure.stateMu.Unlock()
	stopCaller := context.AfterFunc(ctx, cancel)
	defer stopCaller()
	defer cancel()
	if azure.client == nil {
		azure.stateMu.Lock()
		synthesis.finished = true
		synthesis.cancel = nil
		azure.stateMu.Unlock()
		packets = append(packets, internal_type.TextToSpeechErrorPacket{
			ContextID: synthesis.contextID, Error: fmt.Errorf("azure-tts: synthesizer is not initialized"), Type: internal_type.TTSNetworkTimeout,
		})
		return nil
	}

	client := azure.client
	stopDone := make(chan struct{})
	var stopErr error
	stopSpeaking := context.AfterFunc(operationCtx, func() {
		stopErr = client.StopSpeaking()
		close(stopDone)
	})
	var stream azureSynthesisStream
	defer func() {
		if operationCtx.Err() != nil || !stopSpeaking() {
			<-stopDone
		}
		if stream != nil {
			stream.Close()
		}
		azure.stateMu.Lock()
		synthesis.cancel = nil
		if operationCtx.Err() != nil || stopErr != nil {
			synthesis.finished = true
		}
		azure.stateMu.Unlock()
		if stopErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("azure-tts: stop synthesis: %w", stopErr))
		}
	}()

	ssml := strings.Contains(text, "<break ")
	if ssml {
		language := "en-US"
		if configuredLanguage, err := azure.mdlOpts.GetString(internal_options.SpeakOptionLanguage); err == nil && configuredLanguage != "" {
			language = configuredLanguage
		}
		if voice, err := azure.mdlOpts.GetString(internal_options.SpeakOptionVoiceID); err == nil && voice != "" {
			text = fmt.Sprintf(`<voice name="%s">%s</voice>`, voice, text)
		}
		text = fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s">%s</speak>`, language, text)
	}
	var err error
	stream, err = client.StartSpeaking(text, ssml)
	if operationCtx.Err() != nil {
		// Stop may have raced the SDK's start acknowledgment. Stop again after start settles.
		<-stopDone
		stopErr = errors.Join(stopErr, client.StopSpeaking())
	} else if err == nil {
		audio := make([]byte, 2048)
		for {
			var count int
			count, err = stream.Read(audio)
			azure.stateMu.Lock()
			if synthesis.finished || operationCtx.Err() != nil {
				azure.stateMu.Unlock()
				break
			}
			var audioPackets []internal_type.Packet
			if count > 0 {
				if !synthesis.startedAt.IsZero() {
					audioPackets = append(audioPackets, internal_type.ObservabilityMetricRecordPacket{
						ContextID: synthesis.contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.NewMetricTTSLatencyMs(time.Since(synthesis.startedAt), observability.Attributes{"provider": azure.Name()}),
					})
					synthesis.startedAt = time.Time{}
				}
				audioPackets = append(audioPackets, internal_type.TextToSpeechAudioPacket{ContextID: synthesis.contextID, AudioChunk: bytes.Clone(audio[:count])})
			}
			azure.stateMu.Unlock()
			if len(audioPackets) > 0 {
				azure.onPacket(audioPackets...)
			}
			if err != nil {
				break
			}
		}
		if err == io.EOF && operationCtx.Err() == nil {
			var status common.StreamStatus
			status, err = stream.GetStatus()
			if err == nil && status != common.StreamStatusAllData {
				err = fmt.Errorf("stream ended with status %s", status)
			}
		}
	}

	azure.stateMu.Lock()
	if synthesis.finished || operationCtx.Err() != nil || err != nil {
		emitError := !synthesis.finished && operationCtx.Err() == nil
		synthesis.finished = true
		azure.stateMu.Unlock()
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if azure.ctx.Err() != nil {
			return azure.ctx.Err()
		}
		if emitError {
			packets = append(packets, internal_type.TextToSpeechErrorPacket{
				ContextID: synthesis.contextID, Error: fmt.Errorf("azure-tts: synthesis: %w", err), Type: internal_type.TTSNetworkTimeout,
			})
		}
		return nil
	}
	azure.stateMu.Unlock()
	packets = append(packets, internal_type.ObservabilityEventRecordPacket{
		ContextID: synthesis.contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
			Attributes: observability.Attributes{"type": "speaking", "text": text}, OccurredAt: time.Now(),
		},
	})
	return nil
}

func (azure *azureTextToSpeech) Close(ctx context.Context) error {
	azure.ctxCancel()
	_ = azure.writeLock.Acquire(context.WithoutCancel(ctx), 1)
	connectedAt := azure.ttsConnectedAt
	azure.ttsConnectedAt = time.Time{}
	azure.stateMu.Lock()
	if azure.synthesis != nil {
		azure.synthesis.finished = true
	}
	azure.stateMu.Unlock()
	if azure.client != nil {
		azure.client.Close()
		azure.client = nil
	}
	azure.writeLock.Release(1)
	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		azure.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": azure.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewTTSDurationUsageRecord(azure.Name(), duration, observability.Attributes{}),
			},
		)
	}
	azure.onPacket(internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS, Event: observability.TTSClosed,
			Attributes: observability.Attributes{"type": "closed", "provider": azure.Name()}, OccurredAt: time.Now(),
		},
	})
	return nil
}

type azureSDKClient struct {
	*speech.SpeechSynthesizer
}

func (client *azureSDKClient) StartSpeaking(text string, ssml bool) (azureSynthesisStream, error) {
	var outcome speech.SpeechSynthesisOutcome
	if ssml {
		outcome = <-client.StartSpeakingSsmlAsync(text)
	} else {
		outcome = <-client.StartSpeakingTextAsync(text)
	}
	if outcome.Error != nil {
		outcome.Close()
		return nil, outcome.Error
	}
	if outcome.Result == nil {
		return nil, fmt.Errorf("SDK returned no synthesis result")
	}
	if outcome.Result.Reason != common.SynthesizingAudioStarted && outcome.Result.Reason != common.SynthesizingAudioCompleted {
		defer outcome.Close()
		if outcome.Result.Reason == common.Canceled {
			if details, err := speech.NewCancellationDetailsFromSpeechSynthesisResult(outcome.Result); err == nil {
				return nil, fmt.Errorf("SDK canceled synthesis: %s (code=%s)", details.ErrorDetails, details.ErrorCode)
			}
		}
		return nil, fmt.Errorf("SDK returned synthesis reason %s", outcome.Result.Reason)
	}
	stream, err := speech.NewAudioDataStreamFromSpeechSynthesisResult(outcome.Result)
	if err != nil {
		outcome.Close()
		return nil, err
	}
	return &azureSDKStream{AudioDataStream: stream, result: outcome.Result}, nil
}

func (client *azureSDKClient) StopSpeaking() error {
	return <-client.StopSpeakingAsync()
}

type azureSDKStream struct {
	*speech.AudioDataStream
	result *speech.SpeechSynthesisResult
}

func (stream *azureSDKStream) Close() {
	stream.AudioDataStream.Close()
	stream.result.Close()
}
