// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"errors"
	"fmt"
	"io"

	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	observability_collector_requestlog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/requestlog"
	observability_collector_toollog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/toollog"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// buildGRPCResponse wraps stream messages for WebTalk gRPC.
func (s *webrtcStreamer) buildGRPCResponse(msg internal_type.Stream) *protos.WebTalkResponse {
	resp := &protos.WebTalkResponse{Code: webrtc_internal.WebTalkSuccessCode, Success: true}
	switch m := msg.(type) {
	case *protos.ConversationAssistantMessage:
		resp.Data = &protos.WebTalkResponse_Assistant{Assistant: m}
	case *protos.ConversationConfiguration:
		resp.Data = &protos.WebTalkResponse_Configuration{Configuration: m}
	case *protos.ConversationInitialization:
		resp.Data = &protos.WebTalkResponse_Initialization{Initialization: m}
	case *protos.ConversationUserMessage:
		resp.Data = &protos.WebTalkResponse_User{User: m}
	case *protos.ConversationInterruption:
		resp.Data = &protos.WebTalkResponse_Interruption{Interruption: m}
	case *protos.ConversationToolCall:
		resp.Data = &protos.WebTalkResponse_ToolCall{ToolCall: m}
	case *protos.ConversationDisconnection:
		resp.Data = &protos.WebTalkResponse_Disconnection{Disconnection: m}
	case *protos.ConversationError:
		resp.Data = &protos.WebTalkResponse_Error{Error: m}
	case *protos.ConversationEvent:
		resp.Data = &protos.WebTalkResponse_Event{Event: m}
	case *protos.ConversationMetadata:
		resp.Data = &protos.WebTalkResponse_Metadata{Metadata: m}
	case *protos.ConversationMetric:
		resp.Data = &protos.WebTalkResponse_Metric{Metric: m}
	case *protos.ServerSignaling:
		resp.Data = &protos.WebTalkResponse_Signaling{Signaling: m}
	default:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC output message skipped",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataType:      fmt.Sprintf("%T", msg),
			},
		})
		return nil
	}
	return resp
}

// dispatchOutput writes a WebTalk response to the client stream.
func (s *webrtcStreamer) dispatchOutput(resp *protos.WebTalkResponse) bool {
	if err := s.grpcStream.Send(resp); err != nil {
		if s.Ctx.Err() != nil || errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled || status.Code(err) == codes.Unavailable {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
				observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "WebRTC gRPC stream closed during send",
					Attributes: observability.Attributes{
						"component":                   observability.ComponentWebRTC.String(),
						webrtc_internal.DataSessionID: s.sessionID,
						"grpc_code":                   status.Code(err).String(),
						"error":                       err.Error(),
					},
				},
				observability.RecordMetadata{
					Metadata: observability.DisconnectMetadata(
						protos.ConversationDisconnection_DISCONNECTION_TYPE_USER.String(),
						err.Error(),
					),
				})
			if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
				s.Input(disc)
			}
		} else {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to send WebRTC gRPC response",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"grpc_code":                   status.Code(err).String(),
					"error":                       err.Error(),
				},
			})
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(
					protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
					err.Error(),
				),
			})
			if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR); disc != nil {
				s.Input(disc)
			}
		}
		s.Close()
		return false
	}
	return true
}

// runGrpcReader routes client gRPC messages into the conversation stream.
func (s *webrtcStreamer) runGrpcReader() {
	for {
		msg, err := s.grpcStream.Recv()
		if err != nil {
			if s.Ctx.Err() != nil || errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled || status.Code(err) == codes.Unavailable {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
					observability.RecordLog{
						Level:   observability.LevelInfo,
						Message: "WebRTC gRPC stream closed",
						Attributes: observability.Attributes{
							"component":                   observability.ComponentWebRTC.String(),
							webrtc_internal.DataSessionID: s.sessionID,
							"grpc_code":                   status.Code(err).String(),
							"error":                       err.Error(),
						},
					},
					observability.RecordMetadata{
						Metadata: observability.DisconnectMetadata(
							protos.ConversationDisconnection_DISCONNECTION_TYPE_USER.String(),
							err.Error(),
						),
					})
				if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
					s.Input(disc)
				}
			} else {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
					observability.RecordLog{
						Level:   observability.LevelError,
						Message: "WebRTC gRPC receive failed",
						Attributes: observability.Attributes{
							"component":                   observability.ComponentWebRTC.String(),
							webrtc_internal.DataSessionID: s.sessionID,
							"grpc_code":                   status.Code(err).String(),
							"error":                       err.Error(),
						},
					},
					observability.RecordMetadata{
						Metadata: observability.DisconnectMetadata(
							protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
							err.Error(),
						),
					})
				if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR); disc != nil {
					s.Input(disc)
				}
			}
			s.Close()
			return
		}
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC gRPC request received",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataType:      fmt.Sprintf("%T", msg.GetRequest()),
			},
		})
		switch msg.GetRequest().(type) {
		case *protos.WebTalkRequest_Initialization:
			initialization := msg.GetInitialization()
			if !validator.NonNil(initialization) || !validator.OfAssistantDefinition(initialization.GetAssistant()) {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Invalid WebRTC initialization",
					Attributes: observability.Attributes{
						"component":                   observability.ComponentWebRTC.String(),
						webrtc_internal.DataSessionID: s.sessionID,
					},
				})
				s.Input(&protos.ConversationError{Message: "invalid conversation initialization"})
				if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR); disc != nil {
					s.Input(disc)
				}
				s.Close()
				return
			}
			assistantID := initialization.GetAssistant().GetAssistantId()
			if err := s.observer.AddCollectors(
				observability_collector_requestlog.New(observability_collector_requestlog.Config{
					Logger:         s.Logger,
					HTTPLogService: s.httpLogService,
				}),
				observability_collector_toollog.New(observability_collector_toollog.Config{
					Logger:      s.Logger,
					ToolService: s.assistantToolService,
				}),
				webhook.New(s.Ctx, webhook.Config{
					Logger:                        s.Logger,
					Auth:                          s.auth,
					AssistantID:                   assistantID,
					AssistantConfigurationService: s.configurationService,
					HTTPLogService:                s.httpLogService,
				}),
			); err != nil {
				s.Logger.Warnw("observability collector registration failed",
					"component", "webrtc",
					"operation", "add_assistant_collectors",
					"assistant_id", assistantID,
					webrtc_internal.DataSessionID, s.sessionID,
					"error", err,
				)
			}
			s.Input(initialization)
		case *protos.WebTalkRequest_Configuration:
			s.Input(msg.GetConfiguration())
		case *protos.WebTalkRequest_Message:
			s.Input(msg.GetMessage())
		case *protos.WebTalkRequest_Metadata:
			s.Input(msg.GetMetadata())
		case *protos.WebTalkRequest_Metric:
			s.Input(msg.GetMetric())
		case *protos.WebTalkRequest_ToolCallResult:
			s.Input(msg.GetToolCallResult())
		case *protos.WebTalkRequest_Disconnection:
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(
					msg.GetDisconnection().GetType().String(),
					"client_disconnection_request",
				),
			})
			if disc := s.Disconnect(msg.GetDisconnection().GetType()); disc != nil {
				s.Input(disc)
			}
			s.Close()
			return
		case *protos.WebTalkRequest_Signaling:
			s.queueClientSignal(msg.GetSignaling())
		default:
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unknown WebRTC gRPC request type",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataType:      fmt.Sprintf("%T", msg.GetRequest()),
				},
			})
		}
	}
}
