// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

type Streamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mu                    sync.RWMutex
	closed                atomic.Bool
	assistantOutputActive atomic.Bool

	session   *sip_runtime.Session
	lifecycle sip_runtime.LifecycleController
	mediaPort *MediaPort

	outputMu                    sync.Mutex
	pendingAssistantAudioFrames []assistantAudioFrame
	onTransferInitiated         func(targets []string, postTransferAction string)
}

type assistantAudioFrame struct {
	responseID string
	audio      []byte
	completed  bool
}

type StreamerOptions struct {
	Context         context.Context
	Logger          commons.Logger
	Session         *sip_runtime.Session
	Lifecycle       sip_runtime.LifecycleController
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

type FuncOption func(*StreamerOptions)

func WithContext(ctx context.Context) FuncOption {
	return func(options *StreamerOptions) {
		options.Context = ctx
	}
}

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

func WithSession(session *sip_runtime.Session) FuncOption {
	return func(options *StreamerOptions) {
		options.Session = session
	}
}

func WithLifecycle(lifecycle sip_runtime.LifecycleController) FuncOption {
	return func(options *StreamerOptions) {
		options.Lifecycle = lifecycle
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

func New(opts ...FuncOption) (internal_type.SIPCallStreamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.Session == nil {
		return nil, ErrSessionRequired
	}
	if options.Lifecycle == nil {
		return nil, ErrLifecycleControllerRequired
	}

	s := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger,
			options.CallContext,
			options.VaultCredential,
			options.Observer,
			channel_base.WithInputChannelCapacity(RealtimeInputChannelCapacity),
		),
		session:   options.Session,
		lifecycle: options.Lifecycle,
	}
	mediaPort, err := NewMediaPort(MediaPortConfig{
		Context:    s.Ctx,
		Logger:     options.Logger,
		Session:    options.Session,
		StreamSink: s.Input,
		RecordSink: s.Record,
	})
	if err != nil {
		return nil, err
	}
	s.mediaPort = mediaPort
	s.mediaPort.StartInput()
	s.Input(s.CreateConnectionRequest())
	_ = s.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallSessionConnected,
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  Provider,
			"call_id":   options.Session.GetCallID(),
		},
	}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{
			{Key: observability.MetadataClientChannel, Value: Provider},
		},
	}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricCallStatus,
			Value:       observability.MetricCallStatusInProgress,
			Description: "SIP streamer connected",
		}},
	})

	localIP, localPort := mediaPort.LocalAddr()
	_ = s.Record(observability.RecordLog{
		Level:   observability.LevelDebug,
		Message: "SIP streamer created",
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  Provider,
			"call_id":   options.Session.GetCallID(),
			"codec":     mediaPort.CodecName(),
			"rtp_port":  fmt.Sprintf("%d", localPort),
			"local_ip":  localIP,
		},
	})

	// Watchers start only after every owned resource has a cleanup path.
	go func() {
		select {
		case <-options.Session.ByeReceived():
			_ = s.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "SIP user BYE received",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  Provider,
					"call_id":   options.Session.GetCallID(),
					"reason":    "bye_received",
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  Provider,
					"call_id":   options.Session.GetCallID(),
					"reason":    "bye_received",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "bye_received"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "SIP user BYE received",
				}},
			})
			if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				s.Input(msg)
			}
		case <-s.Ctx.Done():
		}
	}()

	go func() {
		reason := ""
		select {
		case <-options.Session.Context().Done():
			reason = "session_context_cancelled"
		case <-options.Context.Done():
			reason = "caller_context_cancelled"
		case <-s.Ctx.Done():
			return
		}
		_ = s.Record(observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "SIP context cancelled",
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   options.Session.GetCallID(),
				"reason":    reason,
			},
		})
		if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
			s.Input(msg)
		}
		s.Close()
	}()

	return s, nil
}

func (s *Streamer) Context() context.Context {
	return s.Ctx
}

func (s *Streamer) Send(response proto.Message) error {
	switch response.(type) {
	case *protos.ConversationPlaybackPause, *protos.ConversationPlaybackContinue, *protos.ConversationPlaybackFlush:
		s.outputMu.Lock()
		defer s.outputMu.Unlock()
		if s.closed.Load() {
			return sip_runtime.ErrSessionClosed
		}
		if _, isFlushOutput := response.(*protos.ConversationPlaybackFlush); isFlushOutput {
			s.pendingAssistantAudioFrames = nil
		}
		if s.mediaPort == nil {
			return nil
		}
		_, outputControlError := s.mediaPort.HandleOutputControl(response)
		return outputControlError
	}
	if s.closed.Load() {
		return sip_runtime.ErrSessionClosed
	}
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if s.mediaPort != nil {
			s.mediaPort.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			s.outputMu.Lock()
			defer s.outputMu.Unlock()
			if s.closed.Load() {
				return sip_runtime.ErrSessionClosed
			}
			if s.assistantOutputActive.Load() {
				if s.mediaPort == nil {
					return nil
				}
				assistantAudioAccepted, assistantAudioError := s.mediaPort.HandleAssistantAudio(data.GetId(), data.GetAudio(), data.GetCompleted())
				if assistantAudioAccepted {
					s.markAssistantAudioReady(data.GetAudio())
				}
				return assistantAudioError
			}
			if s.mediaPort != nil {
				assistantAudioAccepted, assistantAudioError := s.mediaPort.HandleAssistantAudio(data.GetId(), nil, false)
				if assistantAudioError != nil {
					return assistantAudioError
				}
				if !assistantAudioAccepted {
					return nil
				}
			}
			s.markAssistantAudioReady(data.GetAudio())
			s.pendingAssistantAudioFrames = append(s.pendingAssistantAudioFrames, assistantAudioFrame{
				responseID: data.GetId(),
				audio:      append([]byte(nil), data.GetAudio()...),
				completed:  data.GetCompleted(),
			})
			return nil
		}
	case *protos.ConversationInterruption:
		return nil
	case *protos.ConversationDisconnection:
		_ = s.Disconnect(data.GetType())
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallHangup,
			Attributes: observability.Attributes{
				"component":          observability.ComponentCall.String(),
				"provider":           Provider,
				"disconnection_type": data.GetType().String(),
				"reason":             data.GetType().String(),
			},
		}, observability.RecordMetadata{
			Metadata: []*protos.Metadata{
				{Key: observability.MetadataDisconnectReason, Value: data.GetType().String()},
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusComplete,
				Description: "SIP call ended by server-side disconnect",
			}},
		})
		s.endSession()
		s.Close()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			_ = s.Record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallHangup,
				Attributes: observability.Attributes{
					"component":   observability.ComponentCall.String(),
					"provider":    Provider,
					"tool_action": data.GetAction().String(),
					"reason":      "tool_end_conversation",
				},
			}, observability.RecordMetadata{
				Metadata: []*protos.Metadata{
					{Key: observability.MetadataDisconnectReason, Value: "tool_end_conversation"},
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "SIP call ended by tool action",
				}},
			})
			s.SendTransferToolResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
				"status": "completed",
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			raw := data.GetArgs()["transfer_to"]
			if raw == "" {
				_ = s.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP transfer missing target",
					Attributes: observability.Attributes{
						"component":   observability.ComponentCall.String(),
						"provider":    Provider,
						"tool_action": data.GetAction().String(),
						"reason":      "missing transfer target",
					},
				}, observability.RecordMetadata{
					Metadata: []*protos.Metadata{
						{Key: observability.MetadataFailureReason, Value: "missing transfer target"},
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "SIP transfer missing target",
					}},
				})
				s.SendTransferToolResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
					"status": "failed", "reason": "missing transfer target",
				})
				return nil
			}
			targets := s.SplitTransferTargets(raw)
			postTransferAction := data.GetArgs()["post_transfer_action"]
			ringtone := data.GetArgs()["ringtone"]
			s.mu.RLock()
			if s.session != nil {
				s.session.SetMetadata(sip_runtime.MetadataBridgeTransferTarget, strings.Join(targets, commons.SEPARATOR))
				s.session.SetMetadata("tool_id", data.GetToolId())
				s.session.SetMetadata("tool_context_id", data.GetId())
			}
			s.mu.RUnlock()
			s.EnterTransferMode(targets, postTransferAction, ringtone)
			return nil
		}
	}
	return nil
}

func (s *Streamer) StartAssistantOutput() {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	if s.closed.Load() {
		return
	}
	if !s.assistantOutputActive.CompareAndSwap(false, true) {
		return
	}
	pendingAssistantAudioFrames := s.pendingAssistantAudioFrames
	s.pendingAssistantAudioFrames = nil
	if s.mediaPort != nil {
		s.mediaPort.StartBridgeRecorder()
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallMediaStarted,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
			},
		}, observability.RecordMetadata{
			Metadata: []*protos.Metadata{},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusInProgress,
				Description: "SIP media started",
			}},
		})
	}
	for _, pendingAssistantAudioFrame := range pendingAssistantAudioFrames {
		if s.mediaPort != nil {
			if _, assistantAudioError := s.mediaPort.HandleAssistantAudio(pendingAssistantAudioFrame.responseID, pendingAssistantAudioFrame.audio, pendingAssistantAudioFrame.completed); assistantAudioError != nil {
				_ = s.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP queued assistant audio delivery failed",
					Attributes: observability.Attributes{
						"component": observability.ComponentCall.String(),
						"provider":  Provider,
						"error":     assistantAudioError.Error(),
					},
				}, observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: "SIP queued assistant audio delivery failed",
					}},
				})
			}
		}
	}
	if s.mediaPort != nil {
		s.mediaPort.StartOutput()
	}
}

func (s *Streamer) markAssistantAudioReady(audio []byte) {
	if len(audio) == 0 {
		return
	}
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()
	if session == nil || session.GetInfo().Direction != sip_runtime.CallDirectionInbound {
		return
	}
	if session.MarkInboundAssistantAudioReady() {
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallStatus,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   session.GetCallID(),
				"status":    "assistant_audio_ready",
			},
		})
	}
}

func (s *Streamer) EnterTransferMode(targets []string, postTransferAction, ringtoneEnum string) {
	s.outputMu.Lock()
	if s.mediaPort != nil && !s.mediaPort.EnterTransferMode(ringtoneEnum) {
		s.outputMu.Unlock()
		return
	}
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()

	s.mu.RLock()
	session := s.session
	callback := s.onTransferInitiated
	s.mu.RUnlock()

	if session != nil {
		s.transitionCall(session, sip_runtime.CallStateTransferring, sip_runtime.LifecycleReasonTransferModeStarted)
	}

	if callback != nil {
		callback(targets, postTransferAction)
	}
}

func (s *Streamer) ResumeAssistant() {
	if s.mediaPort != nil && !s.mediaPort.ResumeAssistant() {
		return
	}

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session != nil {
		s.transitionCall(session, sip_runtime.CallStateConnected, sip_runtime.LifecycleReasonTransferModeEnded)
	}

	_ = s.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallStatus,
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  Provider,
			"status":    "transfer_resumed",
		},
	})
}

func (s *Streamer) StopTransferRingback() {
	if s.mediaPort != nil {
		s.mediaPort.StopTransferRingback()
	}
}

func (s *Streamer) ConnectTransferMedia(target internal_type.SIPRTPBridgeTarget, outputCodecName string) {
	if s.mediaPort != nil {
		s.mediaPort.ConnectTransferMedia(target, outputCodecName)
	}
}

func (s *Streamer) DisconnectTransferMedia() {
	if s.mediaPort != nil {
		s.mediaPort.DisconnectTransferMedia()
	}
}

func (s *Streamer) RecordTransferOperatorAudio(audio []byte) {
	if s.mediaPort != nil {
		s.mediaPort.RecordTransferOperatorAudio(audio)
	}
}

func (s *Streamer) RecordTransferDurationMetric(durationMs string) {
	if durationMs == "" {
		return
	}
	_ = s.Record(observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricCallTransferDurationMs,
			Value:       durationMs,
			Description: "SIP transfer bridge duration in milliseconds",
		}},
	})
}

func (s *Streamer) SendTransferToolResult(contextID, toolID, toolName string, action protos.ToolCallAction, result map[string]string) {
	s.Input(&protos.ConversationToolCallResult{
		Id:     contextID,
		ToolId: toolID,
		Name:   toolName,
		Action: action,
		Result: result,
	})
}

func (s *Streamer) SendTransferEvent(event proto.Message) {
	s.Input(event)
}

func (s *Streamer) SetTransferRequestHandler(fn func(targets []string, postTransferAction string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTransferInitiated = fn
}

func (s *Streamer) endSession() {
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()
	if session != nil {
		s.endCall(session, sip_runtime.LifecycleReasonStreamerEndSession)
	}
}

func (s *Streamer) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.outputMu.Lock()
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()
	_ = s.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallStatus,
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  Provider,
			"status":    "media_stopped",
		},
	}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{},
	}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricCallStatus,
			Value:       observability.MetricCallStatusComplete,
			Description: "SIP media stopped",
		}},
	})

	if s.mediaPort != nil {
		if err := s.mediaPort.Close(); err != nil {
			_ = s.Record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "SIP media port close failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  Provider,
					"error":     err.Error(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: "SIP media port close failed",
				}},
			})
		}
	}
	s.BaseStreamer.Cancel()

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session != nil && shouldEndSessionOnClose(session.GetState()) {
		s.endCall(session, sip_runtime.LifecycleReasonStreamerClosed)
	}

	_ = s.Record(observability.RecordLog{
		Level:   observability.LevelDebug,
		Message: "SIP streamer closed",
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  Provider,
		},
	})
	return nil
}

func shouldEndSessionOnClose(state sip_runtime.CallState) bool {
	switch state {
	case sip_runtime.CallStateInitializing, sip_runtime.CallStateRinging, sip_runtime.CallStateCancelled, sip_runtime.CallStateFailed, sip_runtime.CallStateEnded:
		return false
	default:
		return true
	}
}

func (s *Streamer) transitionCall(session *sip_runtime.Session, next sip_runtime.CallState, reason sip_runtime.LifecycleReason) {
	if s.lifecycle == nil {
		_ = s.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "SIP lifecycle transition skipped",
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   session.GetCallID(),
				"to":        string(next),
				"reason":    string(reason),
			},
		})
		return
	}
	s.lifecycle.TransitionCall(session, next, reason)
}

func (s *Streamer) endCall(session *sip_runtime.Session, reason sip_runtime.LifecycleReason) {
	if s.lifecycle == nil {
		_ = s.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "SIP lifecycle end skipped",
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   session.GetCallID(),
				"reason":    string(reason),
			},
		})
		return
	}
	if err := s.lifecycle.EndCallWithReason(session, reason); err != nil {
		_ = s.Record(observability.RecordLog{
			Level:   observability.LevelError,
			Message: "SIP lifecycle end failed",
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   session.GetCallID(),
				"reason":    string(reason),
				"error":     err.Error(),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: "SIP lifecycle end failed",
			}},
		})
	}
}
