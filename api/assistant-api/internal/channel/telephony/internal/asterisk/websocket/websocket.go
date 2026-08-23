// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_websocket

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_asterisk "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk/internal"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type asteriskWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	audioProcessor *internal_asterisk.AudioProcessor
	mediaSession   *internal_telephony_media.MediaSession
	connection     *websocket.Conn
	writeMu        sync.Mutex // guards all writes to connection (gorilla WS is not concurrent-write safe)
	closed         atomic.Bool
	channelName    string

	mediaBuffering bool
	mediaBufferMu  sync.Mutex
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
	audioProcessor, err := internal_asterisk.NewAudioProcessor(options.Logger, internal_asterisk.AudioProcessorConfig{
		AsteriskConfig:   internal_audio.NewMulaw8khzMonoAudioConfig(),
		DownstreamConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		SilenceByte:      0xFF, // mu-law silence
		FrameSize:        160,  // 20ms at 8kHz 8-bit
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create audio processor: %w", err)
	}

	aws := &asteriskWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		audioProcessor: audioProcessor,
		connection:     options.Connection,
	}
	aws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     aws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			if aws.isMediaBuffering() {
				return aws.sendCommand("STOP_MEDIA_BUFFERING")
			}
			return nil
		},
		StreamSink: aws.Input,
		OutputSink: aws.sendOutputFrame,
		Record:     aws.Record,
	})

	go aws.runWebSocketReader(aws.connection)
	return aws, nil
}

func (aws *asteriskWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if aws.connection == nil || len(frame.ProviderAudio) == 0 {
		return nil
	}
	aws.writeMu.Lock()
	defer aws.writeMu.Unlock()
	if err := aws.connection.WriteMessage(websocket.BinaryMessage, frame.ProviderAudio); err != nil {
		_ = aws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send audio frame to Asterisk websocket",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          "asterisk_ws",
				"channel_name":      aws.channelName,
				"conversation_uuid": aws.ChannelUUID,
				"payload_bytes":     fmt.Sprintf("%d", len(frame.ProviderAudio)),
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

func (aws *asteriskWebsocketStreamer) runWebSocketReader(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			_ = aws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Asterisk websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_ws",
					"channel_name":      aws.channelName,
					"conversation_uuid": aws.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_ws",
					"channel_name":      aws.channelName,
					"conversation_uuid": aws.ChannelUUID,
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
					Description: "Asterisk websocket reader closed",
				}},
			})
			if msg := aws.Disconnect(disconnectTypeFromReadError(err)); msg != nil {
				aws.Input(msg)
			}
			aws.Cancel()
			return
		}
		switch messageType {
		case websocket.BinaryMessage:
			if err := aws.handleAudioData(message); err != nil {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Asterisk media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"payload_bytes":     fmt.Sprintf("%d", len(message)),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Asterisk media frame processing failed",
					}},
				})
			}
		case websocket.TextMessage:
			event, err := internal_asterisk.ParseAsteriskEvent(string(message))
			if err != nil {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to parse Asterisk event",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"message":           string(message),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to parse Asterisk event",
					}},
				})
				continue
			}
			switch event.Event {
			case "MEDIA_START":
				aws.channelName = event.Channel
				aws.ChannelUUID = event.Channel
				if event.OptimalFrameSize > 0 {
					aws.audioProcessor.SetOptimalFrameSize(event.OptimalFrameSize)
				}
				aws.mediaSession.Start()
				aws.Input(aws.CreateConnectionRequest())
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallMediaStarted,
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           "asterisk_ws",
						"provider_event":     event.Event,
						"channel_name":       aws.channelName,
						"conversation_uuid":  aws.ChannelUUID,
						"optimal_frame_size": fmt.Sprintf("%d", event.OptimalFrameSize),
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataClientChannel, Value: "asterisk_ws"},
						{Key: observability.MetadataClientProviderCallID, Value: event.Channel},
						{Key: observability.MetadataClientCodec, Value: "mulaw"},
						{Key: observability.MetadataClientSampleRate, Value: "8000"},
						{Key: "asterisk.channel_name", Value: aws.channelName},
						{Key: "asterisk.optimal_frame_size", Value: fmt.Sprintf("%d", event.OptimalFrameSize)},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusInProgress,
						Description: "Asterisk media stream started",
					}},
				})
			case "MEDIA_STOP":
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"provider_event":    event.Event,
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
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
						Description: "Asterisk media stream stopped by provider",
					}},
				})
				if msg := aws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
					aws.Input(msg)
				}
				aws.Cancel()
				return
			case "MEDIA_XON":
				aws.audioProcessor.SetXON()
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallStatus,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"provider_event":    event.Event,
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"status":            "flow_control",
						"state":             "xon",
					},
				})
			case "MEDIA_XOFF":
				aws.audioProcessor.SetXOFF()
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallStatus,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"provider_event":    event.Event,
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"status":            "flow_control",
						"state":             "xoff",
					},
				})
			case "MEDIA_BUFFERING_COMPLETED":
				aws.setMediaBuffering(false)
			default:
				if event.Command != "" {
					_ = aws.Record(observability.RecordLog{
						Level:   observability.LevelDebug,
						Message: "Received Asterisk command response",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          "asterisk_ws",
							"command":           event.Command,
							"channel_name":      aws.channelName,
							"conversation_uuid": aws.ChannelUUID,
						},
					})
				} else if event.RawMessage != "" {
					_ = aws.Record(observability.RecordLog{
						Level:   observability.LevelDebug,
						Message: "Received unhandled Asterisk message",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          "asterisk_ws",
							"message":           event.RawMessage,
							"channel_name":      aws.channelName,
							"conversation_uuid": aws.ChannelUUID,
						},
					})
				}
			}
		case websocket.CloseMessage:
			aws.Cancel()
			return
		default:
			_ = aws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Received unsupported Asterisk websocket message type",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_ws",
					"message_type":      fmt.Sprintf("%d", messageType),
					"channel_name":      aws.channelName,
					"conversation_uuid": aws.ChannelUUID,
				},
			})
		}
	}
}

func (aws *asteriskWebsocketStreamer) handleAudioData(audio []byte) error {
	if aws.mediaSession == nil {
		return nil
	}
	if err := aws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      audio,
		ReceivedAt: time.Now(),
	}); err != nil {
		return err
	}
	return nil
}

func (aws *asteriskWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if aws.mediaSession != nil {
			aws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if aws.mediaSession == nil {
				return nil
			}
			if err := aws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Asterisk output audio",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to process Asterisk output audio",
					}},
				})
				return err
			}
		}

	case *protos.ConversationInterruption:
		if aws.mediaSession != nil {
			aws.mediaSession.HandleInterrupt()
		}

	case *protos.ConversationDisconnection:
		// Server-initiated disconnect: the talker already knows the reason
		// (it called Notify with it). No need to round-trip back through
		// CriticalCh — just hang up the Asterisk channel and clean up.
		_ = aws.Disconnect(data.GetType())
		aws.stopAudioProcessing()
		if err := aws.hangupCall(); err != nil {
			_ = aws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to hang up Asterisk call for disconnection",
				Attributes: observability.Attributes{
					"component":          observability.ComponentCall.String(),
					"provider":           "asterisk_ws",
					"channel_name":       aws.channelName,
					"conversation_uuid":  aws.ChannelUUID,
					"disconnection_type": data.GetType().String(),
					"error":              err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Failed to hang up Asterisk call for disconnection",
				}},
			})
		} else {
			_ = aws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":          observability.ComponentCall.String(),
					"provider":           "asterisk_ws",
					"channel_name":       aws.channelName,
					"conversation_uuid":  aws.ChannelUUID,
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
					Description: "Asterisk call ended by server-side disconnect",
				}},
			})
		}
		aws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			aws.stopAudioProcessing()
			result := map[string]string{"status": "completed"}
			if err := aws.hangupCall(); err != nil {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to hang up Asterisk call",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"tool_action":       data.GetAction().String(),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Failed to hang up Asterisk call",
					}},
				})
				result = map[string]string{"status": "failed", "reason": fmt.Sprintf("hangup failed: %v", err)}
			} else {
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
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
						Description: "Asterisk call ended by tool action",
					}},
				})
			}
			aws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: result,
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Asterisk transfer is one-way: redirect leaves Stasis and closes the AI leg.
			// Only the first transfer target is attempted; resume/failover is unsupported.
			raw := data.GetArgs()["transfer_to"]
			targets := aws.SplitTransferTargets(raw)
			if raw == "" || len(targets) == 0 || aws.channelName == "" {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Asterisk transfer failed before dispatch",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"tool_action":       data.GetAction().String(),
						"reason":            "missing target or channel name",
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataFailureReason, Value: "missing target or channel name"},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Asterisk transfer failed before dispatch",
					}},
				})
				aws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{"status": "failed", "reason": "missing target or channel name", "next_action": "end_call"},
				})
				return nil
			}
			to := targets[0]
			if len(targets) > 1 {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelDebug,
					Message: "Asterisk transfer received multiple targets",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
						"tool_action":       data.GetAction().String(),
						"transfer_to":       to,
						"ignored_targets":   fmt.Sprintf("%v", targets[1:]),
					},
				})
			}
			aws.stopAudioProcessing()
			if err := aws.redirectViaARI(to); err != nil {
				_ = aws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Asterisk ARI redirect failed",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
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
						Description: "Asterisk ARI redirect failed",
					}},
				})
				aws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{"status": "failed", "reason": fmt.Sprintf("ARI redirect failed: %v", err), "next_action": "end_call"},
				})
			} else {
				_ = aws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_ws",
						"channel_name":      aws.channelName,
						"conversation_uuid": aws.ChannelUUID,
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
						Description: "Asterisk transfer dispatched",
					}},
				})
				aws.Input(&protos.ConversationToolCallResult{
					Id:     data.GetId(),
					ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
					Result: map[string]string{
						"status":      "dispatched",
						"reason":      "transfer dispatched via ARI redirect; outcome not observed",
						"next_action": "end_call",
					},
				})
			}
		}
	}

	return nil
}

func disconnectTypeFromReadError(err error) protos.ConversationDisconnection_DisconnectionType {
	if err == nil {
		return protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED
	}
	if errors.Is(err, io.EOF) {
		return protos.ConversationDisconnection_DISCONNECTION_TYPE_USER
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		return protos.ConversationDisconnection_DISCONNECTION_TYPE_USER
	}
	return protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED
}

func (aws *asteriskWebsocketStreamer) hangupCall() error {
	if wsErr := aws.sendCommand("HANGUP"); wsErr != nil {
		_ = aws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send Asterisk HANGUP over websocket",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          "asterisk_ws",
				"channel_name":      aws.channelName,
				"conversation_uuid": aws.ChannelUUID,
				"error":             wsErr.Error(),
			},
		})
		if aws.channelName == "" {
			return wsErr
		}
		if ariErr := aws.hangupViaARI(); ariErr != nil {
			return fmt.Errorf("ws hangup failed: %w; ari hangup failed: %w", wsErr, ariErr)
		}
	}
	return nil
}

func (aws *asteriskWebsocketStreamer) stopAudioProcessing() {
	if aws.mediaSession != nil {
		aws.mediaSession.Shutdown()
	}
}

func (aws *asteriskWebsocketStreamer) sendCommand(command string) error {
	if aws.connection == nil {
		return nil
	}
	aws.writeMu.Lock()
	defer aws.writeMu.Unlock()
	return aws.connection.WriteMessage(websocket.TextMessage, []byte(command))
}

func (aws *asteriskWebsocketStreamer) setMediaBuffering(buffering bool) {
	aws.mediaBufferMu.Lock()
	aws.mediaBuffering = buffering
	aws.mediaBufferMu.Unlock()
}

func (aws *asteriskWebsocketStreamer) isMediaBuffering() bool {
	aws.mediaBufferMu.Lock()
	defer aws.mediaBufferMu.Unlock()
	return aws.mediaBuffering
}

func (aws *asteriskWebsocketStreamer) hangupViaARI() error {
	vaultCredential := aws.VaultCredential()
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return fmt.Errorf("vault credential is nil")
	}

	credMap := vaultCredential.GetValue().AsMap()

	ariURL, _ := credMap["ari_url"].(string)
	ariURL = fmt.Sprintf("%s/ari/channels/%s", ariURL, aws.channelName)
	user, _ := credMap["ari_user"].(string)
	password, _ := credMap["ari_password"].(string)

	req, err := http.NewRequest("DELETE", ariURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(user, password)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ARI API returned status: %d", resp.StatusCode)
	}

	_ = aws.Record(observability.RecordLog{
		Level:   observability.LevelDebug,
		Message: "Asterisk call hung up via ARI API",
		Attributes: observability.Attributes{
			"component":         observability.ComponentCall.String(),
			"provider":          "asterisk_ws",
			"channel_name":      aws.channelName,
			"conversation_uuid": aws.ChannelUUID,
		},
	})
	return nil
}

// redirectViaARI transfers a call by redirecting the Asterisk channel to a new extension.
func (aws *asteriskWebsocketStreamer) redirectViaARI(target string) error {
	vaultCredential := aws.VaultCredential()
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return fmt.Errorf("vault credential is nil")
	}
	credMap := vaultCredential.GetValue().AsMap()
	ariURL, _ := credMap["ari_url"].(string)
	user, _ := credMap["ari_user"].(string)
	password, _ := credMap["ari_password"].(string)

	redirectURL := fmt.Sprintf("%s/ari/channels/%s/redirect", ariURL, aws.channelName)
	req, err := http.NewRequest("POST", redirectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create redirect request: %w", err)
	}
	req.SetBasicAuth(user, password)
	q := req.URL.Query()
	q.Set("endpoint", fmt.Sprintf("PJSIP/%s", target))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ARI redirect failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ARI redirect returned status: %d", resp.StatusCode)
	}
	_ = aws.Record(observability.RecordLog{
		Level:   observability.LevelDebug,
		Message: "Asterisk call redirected via ARI",
		Attributes: observability.Attributes{
			"component":         observability.ComponentCall.String(),
			"provider":          "asterisk_ws",
			"channel_name":      aws.channelName,
			"conversation_uuid": aws.ChannelUUID,
			"transfer_to":       target,
		},
	})
	return nil
}

func (aws *asteriskWebsocketStreamer) Cancel() error {
	if !aws.closed.CompareAndSwap(false, true) {
		return nil
	}
	aws.stopAudioProcessing()
	aws.writeMu.Lock()
	conn := aws.connection
	aws.connection = nil
	aws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	aws.BaseStreamer.Cancel()
	return nil
}
