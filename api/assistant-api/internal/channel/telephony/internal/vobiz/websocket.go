// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vobiz_telephony

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
	internal_vobiz "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type vobizWebsocketStreamer struct {
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
	return func(o *StreamerOptions) { o.Logger = logger }
}
func WithConnection(connection *websocket.Conn) FuncOption {
	return func(o *StreamerOptions) { o.Connection = connection }
}
func WithCallContext(callContext *callcontext.CallContext) FuncOption {
	return func(o *StreamerOptions) { o.CallContext = callContext }
}
func WithVaultCredential(vaultCredential *protos.VaultCredential) FuncOption {
	return func(o *StreamerOptions) { o.VaultCredential = vaultCredential }
}
func WithObserver(observer observability.Recorder) FuncOption {
	return func(o *StreamerOptions) { o.Observer = observer }
}

func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	audioProcessor, err := internal_vobiz.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_vobiz.ErrAudioProcessorInitFailed, err)
	}
	vws := &vobizWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		connection: options.Connection,
	}
	vws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     vws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return vws.sendVobizMessage(internal_vobiz.EventTypeClearAudio, nil)
		},
		StreamSink: vws.Input,
		OutputSink: vws.sendOutputFrame,
		Record:     vws.Record,
	})
	go vws.runWebSocketReader()
	return vws, nil
}

func (vws *vobizWebsocketStreamer) runWebSocketReader() {
	conn := vws.connection
	if conn == nil {
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			// Vobiz sends no JSON `stop` — the WebSocket close is the
			// authoritative end-of-stream signal.
			vws.stopAudioProcessing()
			_ = vws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Vobiz websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
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
					Description: "Vobiz websocket reader closed",
				}},
			})
			if msg := vws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				vws.Input(msg)
			}
			vws.BaseStreamer.Cancel()
			return
		}
		var mediaEvent internal_vobiz.VobizMediaEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			_ = vws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to unmarshal Vobiz media event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Failed to unmarshal Vobiz media event",
				}},
			})
			continue
		}
		switch mediaEvent.Event {
		case internal_vobiz.EventTypeStart:
			vws.handleStartEvent(mediaEvent)
			if vws.mediaSession != nil {
				vws.mediaSession.Start()
			}
			vws.Input(vws.CreateConnectionRequest())
			_ = vws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallMediaStarted,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"provider_event":    string(internal_vobiz.EventTypeStart),
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_vobiz.VobizProvider},
					{Key: observability.MetadataClientProviderCallID, Value: vws.ChannelUUID},
					{Key: observability.MetadataClientCodec, Value: "mulaw"},
					{Key: observability.MetadataClientSampleRate, Value: "8000"},
					{Key: "vobiz.stream_id", Value: vws.streamID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "Vobiz media stream started",
				}},
			})
		case internal_vobiz.EventTypeMedia:
			if err := vws.handleMediaEvent(mediaEvent); err != nil {
				_ = vws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Vobiz media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vobiz.VobizProvider,
						"stream_id":         vws.streamID,
						"conversation_uuid": vws.ChannelUUID,
						"error":             err.Error(),
					},
				})
			}
		case internal_vobiz.EventTypePlayedStream, internal_vobiz.EventTypeClearedAudio:
			// playback / clear acknowledgements — no action needed.
		case internal_vobiz.EventTypeStop:
			_ = vws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"provider_event":    string(internal_vobiz.EventTypeStop),
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
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
					Description: "Vobiz media stream stopped by provider",
				}},
			})
			if msg := vws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				vws.Input(msg)
			}
			vws.Cancel()
			return
		default:
			_ = vws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Vobiz event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"provider_event":    string(mediaEvent.Event),
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
				},
			})
		}
	}
}

func (vws *vobizWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if vws.mediaSession != nil {
			vws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if vws.mediaSession == nil {
				return nil
			}
			return vws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted())
		}
	case *protos.ConversationInterruption:
		if vws.mediaSession != nil {
			vws.mediaSession.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		_ = vws.Disconnect(data.GetType())
		_ = vws.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallHangup,
			Attributes: observability.Attributes{
				"component":          observability.ComponentCall.String(),
				"provider":           internal_vobiz.VobizProvider,
				"stream_id":          vws.streamID,
				"conversation_uuid":  vws.ChannelUUID,
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
				Description: "Vobiz call ended by server-side disconnect",
			}},
		})
		_ = vws.sendVobizMessage(internal_vobiz.EventTypeStop, nil)
		vws.stopAudioProcessing()
		vws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			_ = vws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
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
					Description: "Vobiz call ended by tool action",
				}},
			})
			vws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Mid-call transfer is not supported over the vobiz websocket stream.
			_ = vws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Vobiz call transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vobiz.VobizProvider,
					"stream_id":         vws.streamID,
					"conversation_uuid": vws.ChannelUUID,
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for Vobiz"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Vobiz call transfer is not supported",
				}},
			})
			vws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Vobiz", "next_action": "end_call"},
			})
		}
	default:
		_ = vws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Vobiz Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vobiz.VobizProvider,
				"stream_id":         vws.streamID,
				"conversation_uuid": vws.ChannelUUID,
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

func (vws *vobizWebsocketStreamer) handleStartEvent(mediaEvent internal_vobiz.VobizMediaEvent) {
	if mediaEvent.Start != nil && mediaEvent.Start.StreamId != "" {
		vws.streamID = mediaEvent.Start.StreamId
		return
	}
	vws.streamID = mediaEvent.StreamId
}

func (vws *vobizWebsocketStreamer) handleMediaEvent(mediaEvent internal_vobiz.VobizMediaEvent) error {
	if mediaEvent.Media == nil {
		_ = vws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Vobiz media event missing media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vobiz.VobizProvider,
				"stream_id":         vws.streamID,
				"conversation_uuid": vws.ChannelUUID,
			},
		})
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := vws.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		_ = vws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to decode Vobiz media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vobiz.VobizProvider,
				"stream_id":         vws.streamID,
				"conversation_uuid": vws.ChannelUUID,
				"error":             err.Error(),
			},
		})
		return nil
	}
	if vws.mediaSession == nil {
		return nil
	}
	return vws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: receivedAt,
	})
}

func (vws *vobizWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return vws.sendVobizMessage(internal_vobiz.EventTypePlayAudio, &internal_vobiz.VobizOutboundMedia{
		ContentType: internal_vobiz.OutputContentType,
		SampleRate:  internal_vobiz.OutputSampleRate,
		Payload:     vws.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (vws *vobizWebsocketStreamer) sendVobizMessage(eventType internal_vobiz.EventType, mediaData *internal_vobiz.VobizOutboundMedia) error {
	if vws.streamID == "" {
		return nil
	}
	var vobizMessageJSON []byte
	var err error
	if mediaData != nil {
		vobizMessageJSON, err = json.Marshal(internal_vobiz.VobizPlayAudioMessage{
			Event: eventType,
			Media: *mediaData,
		})
	} else {
		vobizMessageJSON, err = json.Marshal(internal_vobiz.VobizControlMessage{
			Event:    eventType,
			StreamID: vws.streamID,
		})
	}
	if err != nil {
		_ = vws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Vobiz message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vobiz.VobizProvider,
				"provider_event":    string(eventType),
				"stream_id":         vws.streamID,
				"conversation_uuid": vws.ChannelUUID,
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to marshal Vobiz message",
			}},
		})
		return err
	}
	vws.writeMu.Lock()
	defer vws.writeMu.Unlock()
	if vws.connection == nil {
		return nil
	}
	if err := vws.connection.WriteMessage(websocket.TextMessage, vobizMessageJSON); err != nil {
		_ = vws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send message to Vobiz",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vobiz.VobizProvider,
				"provider_event":    string(eventType),
				"stream_id":         vws.streamID,
				"conversation_uuid": vws.ChannelUUID,
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

func (vws *vobizWebsocketStreamer) stopAudioProcessing() {
	if vws.mediaSession != nil {
		vws.mediaSession.Shutdown()
	}
}

func (vws *vobizWebsocketStreamer) Cancel() error {
	if !vws.closed.CompareAndSwap(false, true) {
		return nil
	}
	vws.stopAudioProcessing()
	vws.writeMu.Lock()
	conn := vws.connection
	vws.connection = nil
	vws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	vws.BaseStreamer.Cancel()
	return nil
}
