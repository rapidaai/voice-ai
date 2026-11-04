// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo_telephony

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
	internal_plivo "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/plivo/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

// plivoWebsocketStreamer transports bidirectional mu-law 8kHz media over Plivo's
// Audio Streaming WebSocket. Inbound "media" frames are decoded and handed to the
// media session; assistant audio is emitted as Plivo "playAudio" messages, and
// barge-in flushes the caller-side buffer with a "clearAudio" message.
type plivoWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mediaSession *internal_telephony_media.MediaSession

	streamID   string
	connection *websocket.Conn
	writeMu    sync.Mutex
	closed     atomic.Bool
	telephony  *plivoTelephony
}

// StreamerOptions carries the dependencies for a Plivo streamer.
type StreamerOptions struct {
	Logger          commons.Logger
	Connection      *websocket.Conn
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

// FuncOption mutates StreamerOptions.
type FuncOption func(*StreamerOptions)

// WithLogger sets the logger.
func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

// WithConnection sets the WebSocket connection.
func WithConnection(connection *websocket.Conn) FuncOption {
	return func(options *StreamerOptions) {
		options.Connection = connection
	}
}

// WithCallContext sets the call context.
func WithCallContext(callContext *callcontext.CallContext) FuncOption {
	return func(options *StreamerOptions) {
		options.CallContext = callContext
	}
}

// WithVaultCredential sets the vault credential.
func WithVaultCredential(vaultCredential *protos.VaultCredential) FuncOption {
	return func(options *StreamerOptions) {
		options.VaultCredential = vaultCredential
	}
}

// WithObserver sets the observability recorder.
func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *StreamerOptions) {
		options.Observer = observer
	}
}

// New constructs a Plivo media streamer and starts its WebSocket reader.
func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	audioProcessor, err := internal_plivo.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_plivo.ErrAudioProcessorInitFailed, err)
	}

	pws := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		streamID:   "",
		connection: options.Connection,
		telephony: &plivoTelephony{
			logger: options.Logger,
		},
	}

	pws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     pws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return pws.sendClearAudio()
		},
		StreamSink: pws.Input,
		OutputSink: pws.sendOutputFrame,
		Record:     pws.Record,
	})

	// Pass the connection in rather than reading pws.connection from the reader
	// goroutine — Cancel() nils it under writeMu, so an unsynchronized read here
	// would race.
	go pws.runWebSocketReader(pws.connection)
	return pws, nil
}

func (pws *plivoWebsocketStreamer) runWebSocketReader(conn *websocket.Conn) {
	if conn == nil {
		return
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			pws.stopAudioProcessing()
			_ = pws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Plivo websocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"reason":            "websocket_closed",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "websocket_closed"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       "COMPLETE",
					Description: "Plivo websocket reader closed",
				}},
			})
			if msg := pws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				pws.Input(msg)
			}
			// pws.Cancel() (not the embedded BaseStreamer.Cancel field) closes the
			// socket and marks the streamer closed; the field alone only cancels ctx.
			pws.Cancel()
			return
		}

		if messageType != websocket.TextMessage {
			_ = pws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Plivo websocket message type",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"message_type":      fmt.Sprintf("%d", messageType),
				},
			})
			continue
		}

		var mediaEvent internal_plivo.PlivoMediaEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			_ = pws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to unmarshal Plivo media event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"error":             err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       "FAILED",
					Description: "Failed to unmarshal Plivo media event",
				}},
			})
			continue
		}

		switch mediaEvent.Event {
		case internal_plivo.EventTypeStart:
			pws.handleStartEvent(mediaEvent)
			if pws.mediaSession != nil {
				pws.mediaSession.Start()
			}
			pws.Input(pws.CreateConnectionRequest())
			_ = pws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallMediaStarted,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"provider_event":    string(internal_plivo.EventTypeStart),
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataClientChannel, Value: internal_plivo.PlivoProvider},
					{Key: observability.MetadataClientProviderCallID, Value: pws.GetConversationUuid()},
					{Key: observability.MetadataClientCodec, Value: "mulaw"},
					{Key: observability.MetadataClientSampleRate, Value: "8000"},
					{Key: "plivo.stream_id", Value: pws.streamID},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       "INPROGRESS",
					Description: "Plivo media stream started",
				}},
			})
		case internal_plivo.EventTypeMedia:
			if err := pws.handleMediaEvent(mediaEvent); err != nil {
				_ = pws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Plivo media frame",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          internal_plivo.PlivoProvider,
						"stream_id":         pws.streamID,
						"conversation_uuid": pws.GetConversationUuid(),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       "FAILED",
						Description: "Plivo media frame processing failed",
					}},
				})
			}
		case internal_plivo.EventTypeStop:
			_ = pws.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"provider_event":    string(internal_plivo.EventTypeStop),
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"reason":            "provider_stop",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "provider_stop"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       "COMPLETE",
					Description: "Plivo media stream stopped by provider",
				}},
			})
			if msg := pws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				pws.Input(msg)
			}
			pws.Cancel()
			return
		default:
			_ = pws.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unhandled Plivo event",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"provider_event":    string(mediaEvent.Event),
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
				},
			})
		}
	}
}

// Send delivers an outbound conversation message to the caller over the stream.
// Writes go through writeMessage, which re-checks the connection under writeMu —
// so there's no unsynchronized pws.connection read here.
func (pws *plivoWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if pws.mediaSession != nil {
			pws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if pws.mediaSession == nil {
				return nil
			}
			if err := pws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			if pws.mediaSession != nil {
				pws.mediaSession.HandleInterrupt()
			}
		}
	case *protos.ConversationDisconnection:
		_ = pws.Disconnect(data.GetType())
		if pws.GetConversationUuid() != "" {
			if err := pws.hangupWithRetry(pws.GetConversationUuid()); err != nil {
				_ = pws.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to end Plivo call on server-side disconnect",
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_plivo.PlivoProvider,
						"stream_id":          pws.streamID,
						"conversation_uuid":  pws.GetConversationUuid(),
						"disconnection_type": data.GetType().String(),
						"error":              err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       "FAILED",
						Description: "Failed to end Plivo call on server-side disconnect",
					}},
				})
			} else {
				_ = pws.Record(observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallHangup,
					Attributes: observability.Attributes{
						"component":          observability.ComponentCall.String(),
						"provider":           internal_plivo.PlivoProvider,
						"stream_id":          pws.streamID,
						"conversation_uuid":  pws.GetConversationUuid(),
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
						Value:       "COMPLETE",
						Description: "Plivo call ended by server-side disconnect",
					}},
				})
			}
		}
		pws.stopAudioProcessing()
		pws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			result := map[string]string{"status": "completed"}
			if pws.GetConversationUuid() != "" {
				if err := pws.hangupWithRetry(pws.GetConversationUuid()); err != nil {
					_ = pws.Record(observability.RecordLog{
						Level:   observability.LevelError,
						Message: "Failed to end Plivo call",
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_plivo.PlivoProvider,
							"stream_id":         pws.streamID,
							"conversation_uuid": pws.GetConversationUuid(),
							"tool_action":       data.GetAction().String(),
							"error":             err.Error(),
						},
					}, observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       "FAILED",
							Description: "Failed to end Plivo call",
						}},
					})
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("hangup failed: %v", err)}
				} else {
					_ = pws.Record(observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallHangup,
						Attributes: observability.Attributes{
							"component":         observability.ComponentCall.String(),
							"provider":          internal_plivo.PlivoProvider,
							"stream_id":         pws.streamID,
							"conversation_uuid": pws.GetConversationUuid(),
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
							Value:       "COMPLETE",
							Description: "Plivo call ended by tool action",
						}},
					})
				}
			}
			pws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: result,
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Native call transfer is not implemented for Plivo. Report the tool
			// result as failed so the assistant can fall back to ending the call.
			_ = pws.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Plivo call transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          internal_plivo.PlivoProvider,
					"stream_id":         pws.streamID,
					"conversation_uuid": pws.GetConversationUuid(),
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for Plivo"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       "FAILED",
					Description: "Plivo call transfer is not supported",
				}},
			})
			pws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Plivo", "next_action": "end_call"},
			})
		}
	default:
		_ = pws.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Plivo Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_plivo.PlivoProvider,
				"stream_id":         pws.streamID,
				"conversation_uuid": pws.GetConversationUuid(),
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

// handleStartEvent records the stream id and adopts the Plivo CallUUID as the
// channel/conversation identifier used for call control and correlation.
func (pws *plivoWebsocketStreamer) handleStartEvent(mediaEvent internal_plivo.PlivoMediaEvent) {
	pws.streamID = mediaEvent.StreamID
	if mediaEvent.Start == nil {
		return
	}
	if mediaEvent.Start.StreamID != "" {
		pws.streamID = mediaEvent.Start.StreamID
	}
	if mediaEvent.Start.CallID != "" {
		pws.ChannelUUID = mediaEvent.Start.CallID
	}
}

// handleMediaEvent decodes an inbound base64 mu-law frame and forwards it to the
// media session for resampling into the pipeline.
func (pws *plivoWebsocketStreamer) handleMediaEvent(mediaEvent internal_plivo.PlivoMediaEvent) error {
	if mediaEvent.Media == nil {
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := pws.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		_ = pws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to decode Plivo media payload",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_plivo.PlivoProvider,
				"stream_id":         pws.streamID,
				"conversation_uuid": pws.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       "FAILED",
				Description: "Failed to decode Plivo media payload",
			}},
		})
		return nil
	}

	if pws.mediaSession == nil {
		return nil
	}
	if err := pws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: receivedAt,
	}); err != nil {
		return err
	}
	return nil
}

// sendOutputFrame emits one mu-law output frame as a Plivo "playAudio" message.
func (pws *plivoWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return pws.sendPlayAudio(pws.Encoder().EncodeToString(frame.ProviderAudio))
}

// sendPlayAudio writes a Plivo playAudio message carrying base64 mu-law audio.
// The codec is declared on the media object (contentType/sampleRate). Per Plivo's
// protocol the playAudio frame carries no streamId (only clearAudio does).
func (pws *plivoWebsocketStreamer) sendPlayAudio(payload string) error {
	return pws.writeMessage(internal_plivo.PlivoOutboundMessage{
		Event: internal_plivo.EventTypePlayAudio,
		Media: &internal_plivo.PlivoOutboundMedia{
			ContentType: internal_plivo.OutboundContentType,
			SampleRate:  internal_plivo.OutboundSampleRate,
			Payload:     payload,
		},
	})
}

// sendClearAudio flushes the caller-side playback buffer on barge-in. It must
// carry the streamId so Plivo clears the correct stream.
func (pws *plivoWebsocketStreamer) sendClearAudio() error {
	return pws.writeMessage(internal_plivo.PlivoOutboundMessage{
		Event:    internal_plivo.EventTypeClearAudio,
		StreamID: pws.streamID,
	})
}

// writeMessage marshals and writes an outbound message under the write lock.
func (pws *plivoWebsocketStreamer) writeMessage(message internal_plivo.PlivoOutboundMessage) error {
	// No outbound frame is meaningful before the stream is established; mirror
	// the Twilio streamer and drop it rather than emit an unaddressed message.
	if pws.streamID == "" {
		return nil
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		_ = pws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to marshal Plivo message",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_plivo.PlivoProvider,
				"provider_event":    string(message.Event),
				"stream_id":         pws.streamID,
				"conversation_uuid": pws.GetConversationUuid(),
				"error":             err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       "FAILED",
				Description: "Failed to marshal Plivo message",
			}},
		})
		return err
	}

	pws.writeMu.Lock()
	defer pws.writeMu.Unlock()
	if pws.connection == nil {
		return nil
	}
	if err := pws.connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
		_ = pws.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send message to Plivo",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          internal_plivo.PlivoProvider,
				"provider_event":    string(message.Event),
				"stream_id":         pws.streamID,
				"conversation_uuid": pws.GetConversationUuid(),
				"error":             err.Error(),
			},
		})
		return err
	}
	return nil
}

// hangupWithRetry ends the Plivo call over REST, retrying briefly on transient
// failures. The answer XML sets keepCallAlive, so closing the media WebSocket
// alone does not end the PSTN leg — only this REST hangup does.
func (pws *plivoWebsocketStreamer) hangupWithRetry(conversationUUID string) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if err = pws.telephony.HangupCall(conversationUUID, pws.VaultCredential()); err == nil {
			return nil
		}
		if attempt < 3 {
			// Back off, but bail out early if the stream is already being torn down.
			select {
			case <-pws.Ctx.Done():
				return fmt.Errorf("plivo hangup canceled after %d attempt(s): %w", attempt, err)
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return fmt.Errorf("plivo hangup failed after 3 attempts: %w", err)
}

func (pws *plivoWebsocketStreamer) stopAudioProcessing() {
	if pws.mediaSession != nil {
		pws.mediaSession.Shutdown()
	}
}

// GetConversationUuid returns the Plivo CallUUID adopted as the channel identifier.
func (pws *plivoWebsocketStreamer) GetConversationUuid() string {
	return pws.ChannelUUID
}

// Cancel closes the WebSocket, stops audio processing, and cancels the stream.
func (pws *plivoWebsocketStreamer) Cancel() error {
	if !pws.closed.CompareAndSwap(false, true) {
		return nil
	}
	pws.stopAudioProcessing()
	pws.writeMu.Lock()
	conn := pws.connection
	pws.connection = nil
	pws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	pws.BaseStreamer.Cancel()
	return nil
}
