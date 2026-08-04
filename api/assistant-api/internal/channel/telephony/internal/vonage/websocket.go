// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vonage_telephony

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
	internal_vonage "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vonage/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
	"github.com/vonage/vonage-go-sdk"
)

type vonageWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer
	mediaSession *internal_telephony_media.MediaSession
	connection   *websocket.Conn
	writeMu      sync.Mutex
	closed       atomic.Bool
}

// Vonage sends linear16 16kHz — same as the internal Rapida format, so no
// resampling is needed (nil source audio config defaults to linear16 16kHz).
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
	audioProcessor, err := internal_vonage.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_vonage.ErrAudioProcessorInitFailed, err)
	}
	vng := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		connection: options.Connection,
	}
	vng.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:           vng.Ctx,
		Logger:            options.Logger,
		MediaEngine:       audioProcessor,
		SendProviderClear: vng.sendProviderClear,
		StreamSink:        vng.Input,
		OutputSink:        vng.sendOutputFrame,
		Record:            vng.Record,
	})
	go vng.runWebSocketReader()
	return vng, nil
}

func (vng *vonageWebsocketStreamer) runWebSocketReader() {
	conn := vng.connection
	if conn == nil {
		return
	}
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			vng.stopAudioProcessing()
			_ = vng.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Vonage websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vonage.Provider,
					"conversation_uuid": vng.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vonage.Provider,
					"conversation_uuid": vng.GetConversationUuid(),
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
					Description: "Vonage websocket reader closed",
				}},
			})
			if msg := vng.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				vng.Input(msg)
			}
			vng.BaseStreamer.Cancel()
			return
		}
		switch messageType {
		case websocket.TextMessage:
			var textEvent internal_vonage.VonageWebSocketEvent
			if err := json.Unmarshal(message, &textEvent); err != nil {
				_ = vng.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to unmarshal Vonage text event",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vonage.Provider,
						"conversation_uuid": vng.GetConversationUuid(),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to unmarshal Vonage text event",
					}},
				})
				continue
			}
			switch textEvent.Event {
			case internal_vonage.EventTypeWebSocketConnected:
				if vng.mediaSession != nil {
					vng.mediaSession.Start()
				}
				vng.Input(vng.CreateConnectionRequest())
				_ = vng.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallSessionConnected,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vonage.Provider,
						"provider_event":    string(internal_vonage.EventTypeWebSocketConnected),
						"conversation_uuid": vng.GetConversationUuid(),
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataClientChannel, Value: internal_vonage.Provider},
						{Key: observability.MetadataClientProviderCallID, Value: vng.GetConversationUuid()},
						{Key: observability.MetadataClientCodec, Value: "linear16"},
						{Key: observability.MetadataClientSampleRate, Value: "16000"},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusInProgress,
						Description: "Vonage websocket connected",
					}},
				})
			case internal_vonage.EventTypeStop:
				_ = vng.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vonage.Provider,
						"provider_event":    string(internal_vonage.EventTypeStop),
						"conversation_uuid": vng.GetConversationUuid(),
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
						Description: "Vonage websocket stopped by provider",
					}},
				})
				if msg := vng.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
					vng.Input(msg)
				}
				vng.Cancel()
				return
			default:
				_ = vng.Record(observability.RecordLog{
					Level:   observability.LevelDebug,
					Message: "Unhandled Vonage event",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vonage.Provider,
						"provider_event":    string(textEvent.Event),
						"conversation_uuid": vng.GetConversationUuid(),
					},
				})
			}
		case websocket.BinaryMessage:
			if err := vng.handleMediaEvent(message); err != nil {
				_ = vng.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Vonage media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_vonage.Provider,
						"conversation_uuid": vng.GetConversationUuid(),
						"payload_bytes":     fmt.Sprintf("%d", len(message)),
						"error":             err.Error(),
					},
				})
			}
		default:
			_ = vng.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Vonage websocket message type",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vonage.Provider,
					"conversation_uuid": vng.GetConversationUuid(),
					"message_type":      fmt.Sprintf("%d", messageType),
				},
			})
		}
	}
}

func (vng *vonageWebsocketStreamer) Send(response internal_type.Stream) error {
	if vng.connection == nil {
		return nil
	}
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if vng.mediaSession != nil {
			vng.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if vng.mediaSession == nil {
				return nil
			}
			if err := vng.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if vng.mediaSession != nil {
			vng.mediaSession.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		_ = vng.Disconnect(data.GetType())
		conversationUUID := vng.GetConversationUuid()
		if conversationUUID != "" {
			vonageClientAuth, err := vonageAuth(vng.VaultCredential())
			if err != nil {
				_ = vng.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to create Vonage client for server-side disconnect",
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_vonage.Provider,
						"conversation_uuid":  conversationUUID,
						"disconnection_type": data.GetType().String(),
						"error":              err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to create Vonage client for server-side disconnect",
					}},
				})
			} else {
				vonageVoiceClient := vonage.NewVoiceClient(vonageClientAuth)
				if _, _, err := vonageVoiceClient.Hangup(conversationUUID); err != nil {
					_ = vng.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to end Vonage call on server-side disconnect",
						Attributes: observability.Attributes{
							"component":          observability.ComponentCall.String(),
							"provider":           internal_vonage.Provider,
							"conversation_uuid":  conversationUUID,
							"disconnection_type": data.GetType().String(),
							"error":              err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to end Vonage call on server-side disconnect",
						}},
					})
				} else {
					_ = vng.Record(observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallHangup,
						Attributes: observability.Attributes{
							"component":          observability.ComponentCall.String(),
							"provider":           internal_vonage.Provider,
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
							Description: "Vonage call ended by server-side disconnect",
						}},
					})
				}
			}
		}
		vng.stopAudioProcessing()
		vng.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			result := map[string]string{"status": "completed"}
			if vng.GetConversationUuid() != "" {
				cAuth, err := vonageAuth(vng.VaultCredential())
				if err != nil {
					_ = vng.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to create Vonage client for end conversation",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_vonage.Provider,
							"conversation_uuid": vng.GetConversationUuid(),
							"tool_action":       data.GetAction().String(),
							"error":             err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to create Vonage client for end conversation",
						}},
					})
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("vonage client error: %v", err)}
				} else if _, _, err := vonage.NewVoiceClient(cAuth).Hangup(vng.GetConversationUuid()); err != nil {
					_ = vng.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to end Vonage call",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_vonage.Provider,
							"conversation_uuid": vng.GetConversationUuid(),
							"tool_action":       data.GetAction().String(),
							"error":             err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: "Failed to end Vonage call",
						}},
					})
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("hangup failed: %v", err)}
				} else {
					_ = vng.Record(observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallHangup,
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_vonage.Provider,
							"conversation_uuid": vng.GetConversationUuid(),
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
							Description: "Vonage call ended by tool action",
						}},
					})
				}
			}
			vng.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: result,
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Vonage transfer is NOT implemented. A blind transfer would be
			// possible via the Voice API "Transfer Call" PUT
			// (https://api.nexmo.com/v1/calls/{uuid}) with an NCCO containing a
			// `connect` action — equivalent to Twilio `<Dial>`. That path would
			// support post_transfer_action=end_call only. resume_ai would
			// require a B2BUA bridge (separate outbound call + WebSocket
			// reconnect on hangup) which Vonage does not natively support.
			_ = vng.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Vonage call transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_vonage.Provider,
					"conversation_uuid": vng.GetConversationUuid(),
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for Vonage"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Vonage call transfer is not supported",
				}},
			})
			vng.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Vonage", "next_action": "end_call"},
			})
		}
	default:
		_ = vng.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Vonage Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vonage.Provider,
				"conversation_uuid": vng.GetConversationUuid(),
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

func (vng *vonageWebsocketStreamer) handleMediaEvent(message []byte) error {
	if vng.mediaSession == nil {
		return nil
	}
	if err := vng.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      message,
		ReceivedAt: time.Now(),
	}); err != nil {
		return err
	}
	return nil
}

func (vng *vonageWebsocketStreamer) GetConversationUuid() string {
	return vng.ChannelUUID
}

func (vng *vonageWebsocketStreamer) Cancel() error {
	if !vng.closed.CompareAndSwap(false, true) {
		return nil
	}
	vng.stopAudioProcessing()
	vng.writeMu.Lock()
	conn := vng.connection
	vng.connection = nil
	vng.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	vng.BaseStreamer.Cancel()
	return nil
}

func (vng *vonageWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	vng.writeMu.Lock()
	defer vng.writeMu.Unlock()
	if vng.connection == nil {
		return nil
	}
	if err := vng.connection.WriteMessage(websocket.BinaryMessage, frame.ProviderAudio); err != nil {
		_ = vng.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send audio frame to Vonage",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vonage.Provider,
				"conversation_uuid": vng.GetConversationUuid(),
				"payload_bytes":     fmt.Sprintf("%d", len(frame.ProviderAudio)),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to send audio frame to Vonage",
			}},
		})
		return err
	}
	return nil
}

func (vng *vonageWebsocketStreamer) sendProviderClear() error {
	message, err := json.Marshal(internal_vonage.VonageClearMessage{Action: internal_vonage.ClearAction})
	if err != nil {
		_ = vng.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Vonage clear message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vonage.Provider,
				"conversation_uuid": vng.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "Failed to marshal Vonage clear message",
			}},
		})
		return err
	}
	vng.writeMu.Lock()
	defer vng.writeMu.Unlock()
	if vng.connection == nil {
		return nil
	}
	if err := vng.connection.WriteMessage(websocket.TextMessage, message); err != nil {
		_ = vng.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send clear message to Vonage",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_vonage.Provider,
				"conversation_uuid": vng.GetConversationUuid(),
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

func (vng *vonageWebsocketStreamer) stopAudioProcessing() {
	if vng.mediaSession != nil {
		vng.mediaSession.Shutdown()
	}
}
