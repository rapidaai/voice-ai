// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_twilio_telephony

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_twilio "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type twilioWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer
	mediaSession *internal_telephony_media.MediaSession
	streamID     string
	connection   *websocket.Conn
	writeMu      sync.Mutex
	closed       atomic.Bool
}

type StreamerOptions struct {
	Logger          commons.Logger
	Connection      *websocket.Conn
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

type FuncOption func(*StreamerOptions)

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

func WithConnection(connection *websocket.Conn) FuncOption {
	return func(options *StreamerOptions) {
		options.Connection = connection
	}
}

func WithCallContext(callContext *callcontext.CallContext) FuncOption {
	return func(options *StreamerOptions) {
		options.CallContext = callContext
	}
}

func WithVaultCredential(vaultCredential *protos.VaultCredential) FuncOption {
	return func(options *StreamerOptions) {
		options.VaultCredential = vaultCredential
	}
}

func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *StreamerOptions) {
		options.Observer = observer
	}
}

func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	audioProcessor, err := internal_twilio.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_twilio.ErrAudioProcessorInitFailed, err)
	}
	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		streamID:   "",
		connection: options.Connection,
	}
	tws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     tws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return tws.sendTwilioMessage(internal_twilio.EventTypeClear, nil)
		},
		StreamSink: tws.Input,
		OutputSink: tws.sendOutputFrame,
		Record:     tws.Record,
	})
	go tws.runWebSocketReader(tws.connection)
	return tws, nil
}

func (tws *twilioWebsocketStreamer) runWebSocketReader(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Twilio websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
					"reason":            "websocket_closed",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "websocket_closed"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "Twilio websocket reader closed",
				}},
			})
			if msg := tws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				tws.Input(msg)
			}
			tws.Cancel()
			return
		}
		var mediaEvent internal_twilio.TwilioMediaEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to unmarshal Twilio media event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Failed to unmarshal Twilio media event",
				}},
			})
			continue
		}
		switch mediaEvent.Event {
		case internal_twilio.EventTypeConnected:
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallSessionConnected,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"provider_event":    string(internal_twilio.EventTypeConnected),
					"conversation_uuid": tws.GetConversationUuid(),
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_twilio.TwilioProvider},
					{Key: observability.MetadataClientProviderCallID, Value: tws.GetConversationUuid()},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Twilio websocket connected",
				}},
			})
		case internal_twilio.EventTypeStart:
			tws.handleStartEvent(mediaEvent)
			if tws.mediaSession != nil {
				tws.mediaSession.Start()
			}
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallMediaStarted,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"provider_event":    string(internal_twilio.EventTypeStart),
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_twilio.TwilioProvider},
					{Key: observability.MetadataClientProviderCallID, Value: tws.GetConversationUuid()},
					{Key: observability.MetadataClientCodec, Value: "mulaw"},
					{Key: observability.MetadataClientSampleRate, Value: "8000"},
					{Key: "twilio.stream_id", Value: tws.streamID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Twilio media stream started",
				}},
			})
			tws.Input(tws.CreateConnectionRequest())
		case internal_twilio.EventTypeMedia:
			if err := tws.handleMediaEvent(mediaEvent); err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Twilio media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"stream_id":         tws.streamID,
						"conversation_uuid": tws.GetConversationUuid(),
						"error":             err.Error(),
					},
				})
			}
		case internal_twilio.EventTypeStop:
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"provider_event":    string(internal_twilio.EventTypeStop),
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
					"reason":            "provider_stop",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "provider_stop"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "Twilio media stream stopped by provider",
				}},
			})
			if msg := tws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				tws.Input(msg)
			}
			tws.Cancel()
			return
		default:
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Twilio event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_twilio.TwilioProvider,
					"provider_event":    string(mediaEvent.Event),
					"stream_id":         tws.streamID,
					"conversation_uuid": tws.GetConversationUuid(),
				},
			})
		}
	}
}

func (tws *twilioWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if tws.mediaSession != nil {
			tws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if tws.mediaSession == nil {
				return nil
			}
			if err := tws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if tws.mediaSession != nil {
			tws.mediaSession.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		// Server-initiated disconnect: the talker already knows the reason
		// (it called Notify with it). No need to round-trip back through
		// CriticalCh — just notify the carrier via Hangup and clean up.
		_ = tws.Disconnect(data.GetType())
		conversationUUID := tws.GetConversationUuid()
		if conversationUUID != "" {
			twilioRestClient, err := twilioClient(tws.VaultCredential())
			if err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to create Twilio client for server-side disconnect",
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_twilio.TwilioProvider,
						"conversation_uuid":  conversationUUID,
						"disconnection_type": data.GetType().String(),
						"error":              err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to create Twilio client for server-side disconnect",
					}},
				})
			} else {
				updateCallParams := &openapi.UpdateCallParams{}
				updateCallParams.SetStatus("completed")
				if _, err := twilioRestClient.Api.UpdateCall(conversationUUID, updateCallParams); err != nil {
					_ = tws.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to end Twilio call on server-side disconnect",
						Attributes: observability.Attributes{
							"component":          observability.ComponentCall.String(),
							"provider":           internal_twilio.TwilioProvider,
							"conversation_uuid":  conversationUUID,
							"disconnection_type": data.GetType().String(),
							"error":              err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to end Twilio call on server-side disconnect",
						}},
					})
				} else {
					_ = tws.Record(observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallHangup,
						Attributes: observability.Attributes{
							"component":          observability.ComponentCall.String(),
							"provider":           internal_twilio.TwilioProvider,
							"conversation_uuid":  conversationUUID,
							"disconnection_type": data.GetType().String(),
							"reason":             "server_side_disconnect",
						},
					}, observability.RecordMetadata{
						Metadata: []*protos.Metadata{
							{Key: observability.MetadataDisconnectReason, Value: "server_side_disconnect"},
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusComplete,
							Description: "Twilio call ended by server-side disconnect",
						}},
					})
				}
			}
		}
		tws.stopAudioProcessing()
		tws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			result := map[string]string{"status": "completed"}
			conversationUUID := tws.GetConversationUuid()
			if conversationUUID != "" {
				client, err := twilioClient(tws.VaultCredential())
				if err != nil {
					_ = tws.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to create Twilio client for end conversation",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_twilio.TwilioProvider,
							"conversation_uuid": conversationUUID,
							"tool_action":       data.GetAction().String(),
							"error":             err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to end Twilio call",
						}},
					})
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("twilio client error: %v", err)}
				} else {
					params := &openapi.UpdateCallParams{}
					params.SetStatus("completed")
					if _, err := client.Api.UpdateCall(conversationUUID, params); err != nil {
						_ = tws.Record(observability.RecordLog{
							Level:   observability.LevelError,
							Message: "Failed to end Twilio call",
							Attributes: observability.Attributes{
								"component":         observability.ComponentCall.String(),
								"provider":          internal_twilio.TwilioProvider,
								"conversation_uuid": conversationUUID,
								"tool_action":       data.GetAction().String(),
								"error":             err.Error(),
							},
						}, observability.RecordMetric{
							Metrics: []*protos.Metric{{
								Name:        observability.MetricCallStatus,
								Value:       observability.MetricCallStatusFailed,
								Description: "Failed to create Twilio client for end conversation",
							}},
						})
						result = map[string]string{"status": "failed", "reason": fmt.Sprintf("end call failed: %v", err)}
					} else {
						_ = tws.Record(observability.RecordEvent{
							Component: observability.ComponentCall,
							Event:     observability.CallHangup,
							Attributes: observability.Attributes{
								"component":         observability.ComponentCall.String(),
								"provider":          internal_twilio.TwilioProvider,
								"conversation_uuid": conversationUUID,
								"tool_action":       data.GetAction().String(),
								"reason":            "tool_end_conversation",
							},
						}, observability.RecordMetadata{
							Metadata: []*protos.Metadata{
								{Key: observability.MetadataDisconnectReason, Value: "tool_end_conversation"},
							},
						}, observability.RecordMetric{
							Metrics: []*protos.Metric{{
								Name:        observability.MetricCallStatus,
								Value:       observability.MetricCallStatusComplete,
								Description: "Twilio call ended by tool action",
							}},
						})
					}
				}
			}
			tws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: result,
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Twilio transfer is one-way: the carrier owns the leg after REST redirect.
			// Only the first transfer target is attempted; resume/failover is unsupported.
			raw := data.GetArgs()["transfer_to"]
			targets := tws.SplitTransferTargets(raw)
			if raw == "" || len(targets) == 0 || tws.GetConversationUuid() == "" {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Twilio transfer failed before dispatch",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"conversation_uuid": tws.GetConversationUuid(),
						"tool_action":       data.GetAction().String(),
						"reason":            "missing target or call ID",
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataFailureReason, Value: "missing target or call ID"},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Twilio transfer failed before dispatch",
					}},
				})
				tws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{"status": "failed", "reason": "missing target or call ID", "next_action": "end_call"},
				})
				return nil
			}
			to := targets[0]
			if len(targets) > 1 {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelDebug,
					Message: "Twilio transfer received multiple targets",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"conversation_uuid": tws.GetConversationUuid(),
						"tool_action":       data.GetAction().String(),
						"transfer_to":       to,
						"ignored_targets":   fmt.Sprintf("%v", targets[1:]),
					},
				})
			}
			client, err := twilioClient(tws.VaultCredential())
			if err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to create Twilio client for transfer",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"conversation_uuid": tws.GetConversationUuid(),
						"tool_action":       data.GetAction().String(),
						"transfer_to":       to,
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to create Twilio client for transfer",
					}},
				})
				tws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{"status": "failed", "reason": fmt.Sprintf("twilio client error: %v", err), "next_action": "end_call"},
				})
				return nil
			}
			params := &openapi.UpdateCallParams{}
			params.SetTwiml(fmt.Sprintf(`<Response><Dial>%s</Dial></Response>`, to))
			if _, err := client.Api.UpdateCall(tws.GetConversationUuid(), params); err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Twilio transfer dispatch failed",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"conversation_uuid": tws.GetConversationUuid(),
						"tool_action":       data.GetAction().String(),
						"transfer_to":       to,
						"error":             err.Error(),
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataFailureReason, Value: err.Error()},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Twilio transfer dispatch failed",
					}},
				})
				tws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{"status": "failed", "reason": fmt.Sprintf("transfer failed: %v", err), "next_action": "end_call"},
				})
			} else {
				_ = tws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_twilio.TwilioProvider,
						"conversation_uuid": tws.GetConversationUuid(),
						"tool_action":       data.GetAction().String(),
						"transfer_to":       to,
						"reason":            "tool_transfer_conversation",
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataBridgeTransferTarget, Value: to},
						{Key: observability.MetadataBridgeTransferStatus, Value: "dispatched"},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusInProgress,
						Description: "Twilio transfer dispatched",
					}},
				})
				tws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{
						"status":      "dispatched",
						"reason":      "transfer dispatched to Twilio; outcome not observed",
						"next_action": "end_call",
					},
				})
			}
		}
	default:
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Twilio Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_twilio.TwilioProvider,
				"conversation_uuid": tws.GetConversationUuid(),
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

func (tws *twilioWebsocketStreamer) handleStartEvent(mediaEvent internal_twilio.TwilioMediaEvent) {
	tws.streamID = mediaEvent.StreamSid
}

func (tws *twilioWebsocketStreamer) GetConversationUuid() string {
	return tws.ChannelUUID
}

func (tws *twilioWebsocketStreamer) Cancel() error {
	if !tws.closed.CompareAndSwap(false, true) {
		return nil
	}
	tws.stopAudioProcessing()
	tws.writeMu.Lock()
	conn := tws.connection
	tws.connection = nil
	tws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	tws.BaseStreamer.Cancel()
	return nil
}

func (tws *twilioWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return tws.sendTwilioMessage(internal_twilio.EventTypeMedia, &internal_twilio.TwilioOutboundMedia{
		Payload: tws.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (tws *twilioWebsocketStreamer) stopAudioProcessing() {
	if tws.mediaSession != nil {
		tws.mediaSession.Shutdown()
	}
}

func (tws *twilioWebsocketStreamer) handleMediaEvent(mediaEvent internal_twilio.TwilioMediaEvent) error {
	if mediaEvent.Media == nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Twilio media event missing media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_twilio.TwilioProvider,
				"stream_id":         tws.streamID,
				"conversation_uuid": tws.GetConversationUuid(),
			},
		})
		return nil
	}
	payloadBytes, err := tws.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to decode Twilio media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_twilio.TwilioProvider,
				"stream_id":         tws.streamID,
				"conversation_uuid": tws.GetConversationUuid(),
				"error":             err.Error(),
			},
		})
		return nil
	}

	if tws.mediaSession == nil {
		return nil
	}
	if err := tws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: time.Now(),
	}); err != nil {
		return err
	}
	return nil
}

func (tws *twilioWebsocketStreamer) sendTwilioMessage(
	eventType internal_twilio.EventType,
	mediaData *internal_twilio.TwilioOutboundMedia,
) error {
	if tws.streamID == "" {
		return nil
	}
	twilioMessageJSON, err := json.Marshal(internal_twilio.TwilioOutboundMessage{
		Event:    eventType,
		StreamID: tws.streamID,
		Media:    mediaData,
	})
	if err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Twilio message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_twilio.TwilioProvider,
				"provider_event":    string(eventType),
				"stream_id":         tws.streamID,
				"conversation_uuid": tws.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to marshal Twilio message",
			}},
		})
		return err
	}

	tws.writeMu.Lock()
	defer tws.writeMu.Unlock()
	if tws.connection == nil {
		return nil
	}
	if err := tws.connection.WriteMessage(websocket.TextMessage, twilioMessageJSON); err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send message to Twilio",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_twilio.TwilioProvider,
				"provider_event":    string(eventType),
				"stream_id":         tws.streamID,
				"conversation_uuid": tws.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to send message to Twilio",
			}},
		})
		return err
	}

	return nil
}
