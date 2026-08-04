// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_audiosocket

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// Streamer implements AudioSocket media streaming over TCP.
type Streamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	conn           net.Conn
	reader         *bufio.Reader
	writer         *bufio.Writer
	writeMu        sync.Mutex
	closed         atomic.Bool
	audioProcessor *internal_asterisk.AudioProcessor
	mediaSession   *internal_telephony_media.MediaSession

	ctx    context.Context
	cancel context.CancelFunc

	initialUUID string
}

type StreamerOptions struct {
	Logger          commons.Logger
	Connection      net.Conn
	Reader          *bufio.Reader
	Writer          *bufio.Writer
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

func WithConnection(connection net.Conn) FuncOption {
	return func(options *StreamerOptions) {
		options.Connection = connection
	}
}

func WithReader(reader *bufio.Reader) FuncOption {
	return func(options *StreamerOptions) {
		options.Reader = reader
	}
}

func WithWriter(writer *bufio.Writer) FuncOption {
	return func(options *StreamerOptions) {
		options.Writer = writer
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
		AsteriskConfig:   internal_audio.NewLinear8khzMonoAudioConfig(),
		DownstreamConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		SilenceByte:      0x00, // SLIN silence
		FrameSize:        320,  // 20ms at 8kHz 16-bit
	})
	if err != nil {
		return nil, err
	}

	if options.Reader == nil {
		options.Reader = bufio.NewReader(options.Connection)
	}
	if options.Writer == nil {
		options.Writer = bufio.NewWriter(options.Connection)
	}

	as := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		conn:           options.Connection,
		reader:         options.Reader,
		writer:         options.Writer,
		audioProcessor: audioProcessor,
		initialUUID:    options.CallContext.ContextID,
	}
	as.ctx, as.cancel = context.WithCancel(as.Ctx)
	as.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     as.ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		StreamSink:  as.Input,
		OutputSink:  as.sendOutputFrame,
		Record:      as.Record,
	})
	as.mediaSession.Start()
	go as.runFrameReader()
	return as, nil
}

func (as *Streamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if as.conn == nil || len(frame.ProviderAudio) == 0 {
		return nil
	}
	if err := as.writeFrame(FrameTypeAudio, frame.ProviderAudio); err != nil {
		_ = as.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to send Asterisk AudioSocket audio frame",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          "asterisk_as",
				"conversation_uuid": as.ChannelUUID,
				"payload_bytes":     fmt.Sprintf("%d", len(frame.ProviderAudio)),
				"error":             err.Error(),
			},
		})
		// Connection dead — stop media session output sender.
		if as.mediaSession != nil {
			as.mediaSession.Shutdown()
		}
		return err
	}
	return nil
}

func (as *Streamer) writeFrame(frameType byte, payload []byte) error {
	as.writeMu.Lock()
	defer as.writeMu.Unlock()

	if err := WriteFrame(as.writer, frameType, payload); err != nil {
		return err
	}
	return as.writer.Flush()
}

func (as *Streamer) Context() context.Context {
	return as.ctx
}

func (as *Streamer) runFrameReader() {
	_ = as.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallSessionConnected,
		Attributes: observability.Attributes{
			"component":         observability.ComponentCall.String(),
			"provider":          "asterisk_as",
			"conversation_uuid": as.ChannelUUID,
		},
	}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{
			{Key: observability.MetadataClientChannel, Value: "asterisk_as"},
			{Key: observability.MetadataClientProviderCallID, Value: as.ChannelUUID},
		},
	}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricCallStatus,
			Value:       observability.MetricCallStatusInProgress,
			Description: "Asterisk AudioSocket connected",
		}},
	})
	if as.initialUUID != "" {
		as.Input(as.CreateConnectionRequest())
	}
	for {
		select {
		case <-as.ctx.Done():
			return
		default:
		}
		frame, err := ReadFrame(as.reader)
		if err != nil {
			disconnectType := protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED
			if err == io.EOF {
				disconnectType = protos.ConversationDisconnection_DISCONNECTION_TYPE_USER
			}
			_ = as.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Asterisk AudioSocket reader closed",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
					"error":             err.Error(),
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
					"reason":            "reader_closed",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "reader_closed"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "Asterisk AudioSocket reader closed",
				}},
			})
			if msg := as.Disconnect(disconnectType); msg != nil {
				as.Input(msg)
			}
			as.Cancel()
			return
		}
		switch frame.Type {
		case FrameTypeUUID:
			if as.initialUUID == "" {
				as.initialUUID = strings.TrimSpace(string(frame.Payload))
				as.Input(as.CreateConnectionRequest())
			}
		case FrameTypeAudio:
			if as.mediaSession == nil {
				continue
			}
			if err := as.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
				Audio:      frame.Payload,
				ReceivedAt: time.Now(),
			}); err != nil {
				_ = as.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Failed to process Asterisk AudioSocket input audio",
					Attributes: observability.Attributes{
						"component":         observability.ComponentCall.String(),
						"provider":          "asterisk_as",
						"conversation_uuid": as.ChannelUUID,
						"payload_bytes":     fmt.Sprintf("%d", len(frame.Payload)),
						"error":             err.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "Asterisk AudioSocket input audio processing failed",
					}},
				})
				continue
			}
		case FrameTypeSilence:
			// no-op
		case FrameTypeHangup:
			_ = as.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
					"reason":            "provider_hangup",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "provider_hangup"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "Asterisk AudioSocket hangup received",
				}},
			})
			if msg := as.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				as.Input(msg)
			}
			as.Cancel()
			return
		case FrameTypeError:
			_ = as.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Asterisk AudioSocket error frame received",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Asterisk AudioSocket error frame received",
				}},
			})
			if msg := as.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED); msg != nil {
				as.Input(msg)
			}
			as.Cancel()
			return
		}
	}
}

func (as *Streamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if as.mediaSession != nil {
			as.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.GetMessage().(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if as.mediaSession == nil {
				return nil
			}
			if err := as.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
		}
	case *protos.ConversationInterruption:
		if as.mediaSession != nil {
			as.mediaSession.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		// Server-initiated disconnect: the talker already knows the reason
		// (it called Notify with it). No need to round-trip back through
		// CriticalCh — just signal hangup over AudioSocket and clean up.
		_ = as.Disconnect(data.GetType())
		_ = as.writeFrame(FrameTypeHangup, nil)
		_ = as.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallHangup,
			Attributes: observability.Attributes{
				"component":          observability.ComponentCall.String(),
				"provider":           "asterisk_as",
				"conversation_uuid":  as.ChannelUUID,
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
				Description: "Asterisk AudioSocket call ended by server-side disconnect",
			}},
		})
		as.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			_ = as.writeFrame(FrameTypeHangup, nil)
			_ = as.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
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
					Description: "Asterisk AudioSocket call ended by tool action",
				}},
			})
			as.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// AudioSocket transfer is NOT supported. AudioSocket is a raw
			// audio-only TCP protocol with no signalling channel back to
			// Asterisk — there is no way to instruct Asterisk to redirect /
			// bridge from inside the AudioSocket session. To implement
			// transfer for an AudioSocket-attached channel, route the call
			// through the Asterisk WS/ARI streamer instead, which can issue
			// `channels/{id}/redirect` (blind transfer; end_call only).
			_ = as.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Asterisk AudioSocket transfer is not supported",
				Attributes: observability.Attributes{
					"component":         observability.ComponentCall.String(),
					"provider":          "asterisk_as",
					"conversation_uuid": as.ChannelUUID,
					"tool_action":       data.GetAction().String(),
					"transfer_to":       data.GetArgs()["transfer_to"],
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataFailureReason, Value: "transfer not supported for AudioSocket"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "Asterisk AudioSocket transfer is not supported",
				}},
			})
			as.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for AudioSocket", "next_action": "end_call"},
			})
		}
	default:
		_ = as.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Asterisk AudioSocket Send unknown message type",
			Attributes: observability.Attributes{
				"component":         observability.ComponentCall.String(),
				"provider":          "asterisk_as",
				"conversation_uuid": as.ChannelUUID,
				"type":              fmt.Sprintf("%T", response),
			},
		})
	}

	return nil
}

func (as *Streamer) Cancel() error {
	if !as.closed.CompareAndSwap(false, true) {
		return nil
	}
	if as.mediaSession != nil {
		as.mediaSession.Shutdown()
	}
	as.cancel()
	as.writeMu.Lock()
	conn := as.conn
	as.conn = nil
	as.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	as.BaseStreamer.Cancel()
	return nil
}
