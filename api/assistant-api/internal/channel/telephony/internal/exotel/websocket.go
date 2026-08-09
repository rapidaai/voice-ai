// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel_telephony

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_exotel "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel/internal"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type exotelWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer
	mediaSession *internal_telephony_media.MediaSession
	connection   *websocket.Conn
	writeMu      sync.Mutex
	closed       atomic.Bool
	streamID     string
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
	audioProcessor, err := internal_exotel.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_exotel.ErrAudioProcessorInitFailed, err)
	}
	exotel := &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		streamID:   "",
		connection: options.Connection,
	}
	exotel.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     exotel.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return exotel.sendExotelMessage(internal_exotel.EventTypeClear, nil)
		},
		StreamSink: exotel.Input,
		OutputSink: exotel.sendOutputFrame,
		Record:     exotel.Record,
	})
	go exotel.runWebSocketReader()
	return exotel, nil
}

func (exotel *exotelWebsocketStreamer) runWebSocketReader() {
	conn := exotel.connection
	if conn == nil {
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			exotel.stopAudioProcessing()
			_ = exotel.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Exotel websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
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
					Description: "Exotel websocket reader closed",
				}},
			})
			if msg := exotel.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				exotel.Input(msg)
			}
			exotel.BaseStreamer.Cancel()
			return
		}
		var mediaEvent internal_exotel.ExotelMediaEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			_ = exotel.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to unmarshal Exotel media event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Failed to unmarshal Exotel media event",
				}},
			})
			continue
		}
		switch mediaEvent.Event {
		case internal_exotel.EventTypeConnected:
			exotel.Input(exotel.CreateConnectionRequest())
			_ = exotel.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallSessionConnected,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"provider_event":    string(internal_exotel.EventTypeConnected),
					"conversation_uuid": exotel.ChannelUUID,
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_exotel.Provider},
					{Key: observability.MetadataClientProviderCallID, Value: exotel.ChannelUUID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Exotel websocket connected",
				}},
			})
		case internal_exotel.EventTypeStart:
			exotel.handleStartEvent(mediaEvent)
			if exotel.mediaSession != nil {
				exotel.mediaSession.Start()
			}
			_ = exotel.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallMediaStarted,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"provider_event":    string(internal_exotel.EventTypeStart),
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_exotel.Provider},
					{Key: observability.MetadataClientProviderCallID, Value: exotel.ChannelUUID},
					{Key: observability.MetadataClientCodec, Value: "linear16"},
					{Key: observability.MetadataClientSampleRate, Value: "16000"},
					{Key: "exotel.stream_id", Value: exotel.streamID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Exotel media stream started",
				}},
			})
		case internal_exotel.EventTypeMedia:
			if err := exotel.handleMediaEvent(mediaEvent); err != nil {
				_ = exotel.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Exotel media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_exotel.Provider,
						"stream_id":         exotel.streamID,
						"conversation_uuid": exotel.ChannelUUID,
						"error":             err.Error(),
					},
				})
			}
		case internal_exotel.EventTypeDTMF:
			_ = exotel.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallStatus,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"provider_event":    string(internal_exotel.EventTypeDTMF),
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
					"status":            internal_exotel.ChannelEventDTMF,
				},
			})
		case internal_exotel.EventTypeStop:
			_ = exotel.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"provider_event":    string(internal_exotel.EventTypeStop),
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
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
					Description: "Exotel media stream stopped by provider",
				}},
			})
			if msg := exotel.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				exotel.Input(msg)
			}
			exotel.Cancel()
			return
		default:
			_ = exotel.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Exotel event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"provider_event":    string(mediaEvent.Event),
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
				},
			})
		}
	}
}

func (exotel *exotelWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if exotel.mediaSession != nil {
			exotel.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if exotel.mediaSession == nil {
				return nil
			}
			if err := exotel.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if exotel.mediaSession != nil {
			exotel.mediaSession.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		// Server-initiated disconnect: the talker already knows the reason
		// (it called Notify with it). No need to round-trip back through
		// CriticalCh — Exotel has no REST API to terminate a call; closing
		// the WebSocket via Cancel is the only way to release the call leg.
		_ = exotel.Disconnect(data.GetType())
		_ = exotel.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallHangup,
			Attributes: observability.Attributes{
				"component":          observability.ComponentCall.String(),
				"provider":           internal_exotel.Provider,
				"stream_id":          exotel.streamID,
				"conversation_uuid":  exotel.ChannelUUID,
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
				Description: "Exotel call ended by server-side disconnect",
			}},
		})
		exotel.stopAudioProcessing()
		exotel.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			_ = exotel.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
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
					Description: "Exotel call ended by tool action",
				}},
			})
			exotel.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Exotel transfer is NOT supported. Exotel exposes call-flow level
			// "Connect" applets but no live mid-call transfer API on the
			// streaming WebSocket leg. A blind transfer would require building
			// an out-of-band Connect/Dial app and redirecting via the Exotel
			// HTTP API; resume_ai is not feasible without a B2BUA bridge
			// (Exotel does not provide an SDP/RTP path to bridge against).
			_ = exotel.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Exotel call transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_exotel.Provider,
					"stream_id":         exotel.streamID,
					"conversation_uuid": exotel.ChannelUUID,
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for Exotel"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Exotel call transfer is not supported",
				}},
			})
			exotel.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Exotel", "next_action": "end_call"},
			})
		}
	default:
		_ = exotel.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Exotel Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_exotel.Provider,
				"stream_id":         exotel.streamID,
				"conversation_uuid": exotel.ChannelUUID,
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) handleStartEvent(mediaEvent internal_exotel.ExotelMediaEvent) {
	exotel.streamID = mediaEvent.StreamSid
}

func (exotel *exotelWebsocketStreamer) handleMediaEvent(mediaEvent internal_exotel.ExotelMediaEvent) error {
	if mediaEvent.Media == nil {
		_ = exotel.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Exotel media event missing media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_exotel.Provider,
				"stream_id":         exotel.streamID,
				"conversation_uuid": exotel.ChannelUUID,
			},
		})
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := exotel.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		_ = exotel.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to decode Exotel media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_exotel.Provider,
				"stream_id":         exotel.streamID,
				"conversation_uuid": exotel.ChannelUUID,
				"error":             err.Error(),
			},
		})
		return nil
	}

	if exotel.mediaSession == nil {
		return nil
	}
	if err := exotel.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: receivedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) sendExotelMessage(eventType internal_exotel.EventType, mediaData *internal_exotel.ExotelOutboundMedia) error {
	if exotel.streamID == "" {
		return nil
	}
	message := internal_exotel.ExotelOutboundMessage{
		Event:    eventType,
		StreamID: exotel.streamID,
		Media:    mediaData,
	}
	exotelMessageJSON, err := json.Marshal(message)
	if err != nil {
		_ = exotel.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Exotel message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_exotel.Provider,
				"provider_event":    string(eventType),
				"stream_id":         exotel.streamID,
				"conversation_uuid": exotel.ChannelUUID,
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to marshal Exotel message",
			}},
		})
		return err
	}
	exotel.writeMu.Lock()
	defer exotel.writeMu.Unlock()
	if exotel.connection == nil {
		return nil
	}
	if err := exotel.connection.WriteMessage(websocket.TextMessage, exotelMessageJSON); err != nil {
		_ = exotel.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send message to Exotel",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_exotel.Provider,
				"provider_event":    string(eventType),
				"stream_id":         exotel.streamID,
				"conversation_uuid": exotel.ChannelUUID,
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) Cancel() error {
	if !exotel.closed.CompareAndSwap(false, true) {
		return nil
	}
	exotel.stopAudioProcessing()
	exotel.writeMu.Lock()
	conn := exotel.connection
	exotel.connection = nil
	exotel.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	exotel.BaseStreamer.Cancel()
	return nil
}

func (exotel *exotelWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return exotel.sendExotelMessage(internal_exotel.EventTypeMedia, &internal_exotel.ExotelOutboundMedia{
		Payload: exotel.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (exotel *exotelWebsocketStreamer) stopAudioProcessing() {
	if exotel.mediaSession != nil {
		exotel.mediaSession.Shutdown()
	}
}
