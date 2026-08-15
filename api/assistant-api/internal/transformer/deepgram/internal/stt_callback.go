// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package deepgram_internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/websocket/interfaces"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// Implement the LiveMessageCallback interface
type deepgramSttCallback struct {
	logger       commons.Logger
	onPacket     func(pkt ...internal_type.Packet) error
	options      utils.Option
	contextID    func() string
	providerName string
	metrics      *SttSessionMetrics
}

func NewDeepgramSttCallback(
	logger commons.Logger,
	onPacket func(pkt ...internal_type.Packet) error,
	options utils.Option,
	contextID func() string,
	providerName string,
	metrics *SttSessionMetrics,
) msginterfaces.LiveMessageCallback {
	return &deepgramSttCallback{
		logger:       logger,
		onPacket:     onPacket,
		options:      options,
		contextID:    contextID,
		providerName: providerName,
		metrics:      metrics,
	}
}

// Handle when the WebSocket is opened
func (d *deepgramSttCallback) Open(or *msginterfaces.OpenResponse) error {
	return nil
}

// Handle incoming transcription messages from Deepgram
func (d *deepgramSttCallback) Message(mr *msginterfaces.MessageResponse) error {
	ctxID := d.contextID()
	for _, alternative := range mr.Channel.Alternatives {
		if alternative.Transcript == "" {
			continue
		}

		now := time.Now()
		transcriptLatency := time.Duration(0)
		ttftValue := ""
		ttltValue := ""
		latencyValue := ""
		d.metrics.mu.Lock()
		if !d.metrics.speechStartedAt.IsZero() && !now.Before(d.metrics.speechStartedAt) {
			speechStartedMs := strconv.FormatInt(now.Sub(d.metrics.speechStartedAt).Milliseconds(), 10)
			if !d.metrics.ttftReported {
				d.metrics.ttftReported = true
				ttftValue = speechStartedMs
			}
			if mr.IsFinal && !d.metrics.ttltReported {
				d.metrics.ttltReported = true
				ttltValue = speechStartedMs
			}
		}
		if mr.IsFinal && !d.metrics.speechEndedAt.IsZero() && !now.Before(d.metrics.speechEndedAt) {
			transcriptLatency = now.Sub(d.metrics.speechEndedAt)
			if !d.metrics.latencyReported {
				d.metrics.latencyReported = true
				latencyValue = strconv.FormatInt(transcriptLatency.Milliseconds(), 10)
			}
		}
		d.metrics.mu.Unlock()

		if ttftValue != "" {
			d.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTTimeToFirstTokenMs,
						Value:       ttftValue,
						Description: STTTimeToFirstTokenMetricDescription,
					}},
					Attributes: observability.Attributes{
						"provider":  d.providerName,
						"messageId": ctxID,
					},
				},
			})
		}
		if ttltValue != "" {
			d.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTTimeToLastTokenMs,
						Value:       ttltValue,
						Description: STTTimeToLastTokenMetricDescription,
					}},
					Attributes: observability.Attributes{
						"provider":  d.providerName,
						"messageId": ctxID,
					},
				},
			})
		}
		if latencyValue != "" {
			d.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeUserMessage,
				Record: observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricSTTLatencyMs,
						Value:       latencyValue,
						Description: STTLatencyMetricDescription,
					}},
					Attributes: observability.Attributes{
						"provider":  d.providerName,
						"messageId": ctxID,
					},
				},
			})
		}

		if mr.IsFinal {
			if v, err := d.options.GetFloat64(internal_options.ListenOptionThreshold); err == nil {
				if alternative.Confidence < v {
					d.onPacket(internal_type.ObservabilityEventRecordPacket{
						ContextID: ctxID,
						Scope:     internal_type.ObservabilityRecordScopeUserMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentSTT,
							Event:     observability.STTLowConfidence,
							Attributes: observability.Attributes{
								"type":       "low_confidence",
								"script":     alternative.Transcript,
								"confidence": fmt.Sprintf("%.4f", alternative.Confidence),
								"threshold":  fmt.Sprintf("%.4f", v),
							},
							OccurredAt: now,
						},
					})
					return nil
				}
			}
			lang := d.GetMostUsedLanguage(alternative.Languages)
			d.onPacket(
				internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: "word"},
				internal_type.SpeechToTextPacket{
					ContextID:  ctxID,
					Script:     alternative.Transcript,
					Confidence: alternative.Confidence,
					Language:   lang,
					Interim:    false,
					Latency:    transcriptLatency,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentSTT,
						Event:     observability.STTCompleted,
						Attributes: observability.Attributes{
							"type":       "completed",
							"script":     alternative.Transcript,
							"confidence": fmt.Sprintf("%.4f", alternative.Confidence),
							"language":   lang,
							"word_count": fmt.Sprintf("%d", len(strings.Fields(alternative.Transcript))),
							"messageId":  ctxID,
							"char_count": fmt.Sprintf("%d", len(alternative.Transcript)),
						},
						OccurredAt: now,
					},
				},
			)
			return nil
		}

		if v, err := d.options.GetFloat64(internal_options.ListenOptionThreshold); err == nil {
			if alternative.Confidence < v {
				d.onPacket(internal_type.ObservabilityEventRecordPacket{
					ContextID: ctxID,
					Scope:     internal_type.ObservabilityRecordScopeUserMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentSTT,
						Event:     observability.STTLowConfidence,
						Attributes: observability.Attributes{
							"type":       "low_confidence",
							"script":     alternative.Transcript,
							"confidence": fmt.Sprintf("%.4f", alternative.Confidence),
							"threshold":  fmt.Sprintf("%.4f", v),
						},
						OccurredAt: now,
					},
				})
				return nil
			}
		}

		lang := d.GetMostUsedLanguage(alternative.Languages)
		d.onPacket(
			internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: "word"},
			internal_type.SpeechToTextPacket{
				ContextID:  ctxID,
				Script:     alternative.Transcript,
				Confidence: alternative.Confidence,
				Language:   lang,
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
						"script":     alternative.Transcript,
						"messageId":  ctxID,
						"confidence": fmt.Sprintf("%.4f", alternative.Confidence),
					},
					OccurredAt: now,
				},
			},
		)
		return nil
	}
	return nil
}

// Handle utterance end event - this signals the end of a sentence
func (d *deepgramSttCallback) UtteranceEnd(ur *msginterfaces.UtteranceEndResponse) error {
	return nil
}

// Handle metadata emitted by Deepgram for the active STT stream.
func (d *deepgramSttCallback) Metadata(md *msginterfaces.MetadataResponse) error {
	if md != nil {
		d.onPacket(internal_type.ObservabilityMetadataRecordPacket{
			ContextID: d.contextID(),
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewConversationMetadataRecord([]*protos.Metadata{{
				Key:   observability.MetadataSTTRequestID,
				Value: md.RequestID,
			}}),
		})
	}
	return nil
}

// Handle speech started event — no-op; timing is driven by the conversation STT packets.
func (d *deepgramSttCallback) SpeechStarted(ssr *msginterfaces.SpeechStartedResponse) error {
	return nil
}

// Handle when the WebSocket is closed
func (d *deepgramSttCallback) Close(cr *msginterfaces.CloseResponse) error {
	// d.logger.Debugf("Deepgram WebSocket closed")
	return nil
}

// Handle errors from Deepgram
func (d *deepgramSttCallback) Error(er *msginterfaces.ErrorResponse) error {
	ctxID := d.contextID()
	d.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSTTError,
					Value:       "1",
					Description: STTProviderErrorMetricDescription,
				}},
				Attributes: observability.Attributes{
					"provider": d.providerName,
				},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: er.ErrMsg,
				Attributes: observability.Attributes{
					"component": observability.ComponentSTT.String(),
					"messageId": ctxID,
					"error":     observability.AttributeValue(er),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// Handle unhandled events (optional, can be left empty)
func (d *deepgramSttCallback) UnhandledEvent(byData []byte) error {
	d.logger.Errorf(STTUnhandledEventLogMessage, byData)
	return nil
}

func (d *deepgramSttCallback) GetMostUsedLanguage(languages []string) string {
	if len(languages) == 0 {
		return "en"
	}

	languageCount := make(map[string]int)
	for _, lang := range languages {
		languageCount[lang]++
	}

	mostUsedLang := ""
	maxCount := 0
	for lang, count := range languageCount {
		if count > maxCount {
			maxCount = count
			mostUsedLang = lang
		}
	}
	return mostUsedLang
}
