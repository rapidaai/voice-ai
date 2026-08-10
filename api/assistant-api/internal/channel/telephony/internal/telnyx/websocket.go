// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx_telephony

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
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type telnyxWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mediaSession *internal_telephony_media.MediaSession

	streamID      string
	callControlID string
	connection    *websocket.Conn
	writeMu       sync.Mutex
	closed        atomic.Bool
	telephony     *telnyxTelephony
}

// Telnyx sends PCMU 8kHz, matching Twilio's provider audio format.
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
	audioProcessor, err := internal_telnyx.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_telnyx.ErrAudioProcessorInitFailed, err)
	}

	tws := &telnyxWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		streamID:   "",
		connection: options.Connection,
		telephony: &telnyxTelephony{
			logger: options.Logger,
		},
	}

	tws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     tws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return tws.sendTelnyxMessage(internal_telnyx.EventTypeClear, nil)
		},
		StreamSink: tws.Input,
		OutputSink: tws.sendOutputFrame,
		Record:     tws.Record,
	})

	go tws.runWebSocketReader()
	return tws, nil
}

func (tws *telnyxWebsocketStreamer) runWebSocketReader() {
	conn := tws.connection
	if conn == nil {
		return
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			tws.stopAudioProcessing()
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Telnyx websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
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
					Description: "Telnyx websocket reader closed",
				}},
			})
			if msg := tws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				tws.Input(msg)
			}
			tws.BaseStreamer.Cancel()
			return
		}

		if messageType != websocket.TextMessage {
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Telnyx websocket message type",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
					"message_type":      fmt.Sprintf("%d", messageType),
				},
			})
			continue
		}

		var mediaEvent internal_telnyx.TelnyxWebSocketEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to unmarshal Telnyx media event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Failed to unmarshal Telnyx media event",
				}},
			})
			continue
		}

		switch mediaEvent.Event {
		case internal_telnyx.EventTypeConnected:
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallSessionConnected,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"provider_event":    string(internal_telnyx.EventTypeConnected),
					"conversation_uuid": tws.GetConversationUuid(),
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_telnyx.Provider},
					{Key: observability.MetadataClientProviderCallID, Value: tws.GetConversationUuid()},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Telnyx websocket connected",
				}},
			})
		case internal_telnyx.EventTypeStart:
			tws.handleStartEvent(mediaEvent)
			if tws.mediaSession != nil {
				tws.mediaSession.Start()
			}
			tws.Input(tws.CreateConnectionRequest())
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallMediaStarted,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"provider_event":    string(internal_telnyx.EventTypeStart),
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_telnyx.Provider},
					{Key: observability.MetadataClientProviderCallID, Value: tws.GetConversationUuid()},
					{Key: observability.MetadataClientCodec, Value: "mulaw"},
					{Key: observability.MetadataClientSampleRate, Value: "8000"},
					{Key: "telnyx.stream_id", Value: tws.streamID},
					{Key: "telnyx.call_control_id", Value: tws.callControlID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Telnyx media stream started",
				}},
			})
		case internal_telnyx.EventTypeMedia:
			if err := tws.handleMediaEvent(mediaEvent); err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Telnyx media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_telnyx.Provider,
						"stream_id":         tws.streamID,
						"call_control_id":   tws.callControlID,
						"conversation_uuid": tws.GetConversationUuid(),
						"error":             err.Error(),
					},
				})
			}
		case internal_telnyx.EventTypeDTMF:
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallStatus,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"provider_event":    string(internal_telnyx.EventTypeDTMF),
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
					"status":            internal_telnyx.ChannelEventDTMF,
				},
			})
		case internal_telnyx.EventTypeStop:
			_ = tws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"provider_event":    string(internal_telnyx.EventTypeStop),
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
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
					Description: "Telnyx media stream stopped by provider",
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
				Message: "Unhandled Telnyx event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"provider_event":    string(mediaEvent.Event),
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
				},
			})
		}
	}
}

func (tws *telnyxWebsocketStreamer) Send(response internal_type.Stream) error {
	if tws.connection == nil {
		return nil
	}
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
		_ = tws.Disconnect(data.GetType())
		if tws.GetConversationUuid() != "" {
			if err := tws.telephony.HangupCall(tws.GetConversationUuid(), tws.VaultCredential()); err != nil {
				_ = tws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to end Telnyx call on server-side disconnect",
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_telnyx.Provider,
						"stream_id":          tws.streamID,
						"call_control_id":    tws.callControlID,
						"conversation_uuid":  tws.GetConversationUuid(),
						"disconnection_type": data.GetType().String(),
						"error":              err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to end Telnyx call on server-side disconnect",
					}},
				})
			} else {
				_ = tws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_telnyx.Provider,
						"stream_id":          tws.streamID,
						"call_control_id":    tws.callControlID,
						"conversation_uuid":  tws.GetConversationUuid(),
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
						Description: "Telnyx call ended by server-side disconnect",
					}},
				})
			}
		}
		tws.stopAudioProcessing()
		tws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			result := map[string]string{"status": "completed"}
			if tws.GetConversationUuid() != "" {
				if err := tws.telephony.HangupCall(tws.GetConversationUuid(), tws.VaultCredential()); err != nil {
					_ = tws.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to end Telnyx call",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_telnyx.Provider,
							"stream_id":         tws.streamID,
							"call_control_id":   tws.callControlID,
							"conversation_uuid": tws.GetConversationUuid(),
							"tool_action":       data.GetAction().String(),
							"error":             err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to end Telnyx call",
						}},
					})
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("hangup failed: %v", err)}
				} else {
					_ = tws.Record(observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallHangup,
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_telnyx.Provider,
							"stream_id":         tws.streamID,
							"call_control_id":   tws.callControlID,
							"conversation_uuid": tws.GetConversationUuid(),
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
							Description: "Telnyx call ended by tool action",
						}},
					})
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
			_ = tws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Telnyx call transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_telnyx.Provider,
					"stream_id":         tws.streamID,
					"call_control_id":   tws.callControlID,
					"conversation_uuid": tws.GetConversationUuid(),
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for Telnyx"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Telnyx call transfer is not supported",
				}},
			})
			tws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Telnyx", "next_action": "end_call"},
			})
		}
	default:
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Telnyx Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_telnyx.Provider,
				"stream_id":         tws.streamID,
				"call_control_id":   tws.callControlID,
				"conversation_uuid": tws.GetConversationUuid(),
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) handleStartEvent(mediaEvent internal_telnyx.TelnyxWebSocketEvent) {
	tws.streamID = mediaEvent.StreamID
	if mediaEvent.Start == nil {
		return
	}
	tws.callControlID = mediaEvent.Start.CallControlID
	tws.ChannelUUID = mediaEvent.Start.CallControlID
}

func (tws *telnyxWebsocketStreamer) handleMediaEvent(mediaEvent internal_telnyx.TelnyxWebSocketEvent) error {
	if mediaEvent.Media == nil {
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := tws.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to decode Telnyx media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_telnyx.Provider,
				"stream_id":         tws.streamID,
				"call_control_id":   tws.callControlID,
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
		ReceivedAt: receivedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return tws.sendTelnyxMessage(internal_telnyx.EventTypeMedia, &internal_telnyx.TelnyxOutboundMedia{
		Payload: tws.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (tws *telnyxWebsocketStreamer) sendTelnyxMessage(eventType internal_telnyx.EventType, mediaData *internal_telnyx.TelnyxOutboundMedia) error {
	if tws.connection == nil || tws.streamID == "" {
		return nil
	}
	message := internal_telnyx.TelnyxOutboundMessage{
		Event:    eventType,
		StreamID: tws.streamID,
		Media:    mediaData,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Telnyx message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_telnyx.Provider,
				"provider_event":    string(eventType),
				"stream_id":         tws.streamID,
				"call_control_id":   tws.callControlID,
				"conversation_uuid": tws.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to marshal Telnyx message",
			}},
		})
		return err
	}

	tws.writeMu.Lock()
	defer tws.writeMu.Unlock()
	if tws.connection == nil {
		return nil
	}
	if err := tws.connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
		_ = tws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send message to Telnyx",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_telnyx.Provider,
				"provider_event":    string(eventType),
				"stream_id":         tws.streamID,
				"call_control_id":   tws.callControlID,
				"conversation_uuid": tws.GetConversationUuid(),
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) stopAudioProcessing() {
	if tws.mediaSession != nil {
		tws.mediaSession.Shutdown()
	}
}

func (tws *telnyxWebsocketStreamer) GetConversationUuid() string {
	return tws.ChannelUUID
}

func (tws *telnyxWebsocketStreamer) Cancel() error {
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
