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
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_sip "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type Streamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mu                    sync.RWMutex
	closed                atomic.Bool
	assistantOutputActive atomic.Bool

	session   *sip_runtime.Session
	lifecycle sip_runtime.LifecycleController
	mediaPort *internal_sip.MediaPort

	outputMu                    sync.Mutex
	pendingAssistantAudioFrames []assistantAudioFrame
	onTransferInitiated         func(targets []string, postTransferAction string)
}

type assistantAudioFrame struct {
	audio     []byte
	completed bool
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
		return nil, fmt.Errorf("SIP session is required; standalone server mode is not supported")
	}
	if options.Lifecycle == nil {
		return nil, fmt.Errorf("SIP lifecycle controller is required")
	}

	s := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
	}

	// Peer BYE is reported to Talk; MediaPort owns bridge teardown safety.
	go func() {
		select {
		case <-options.Session.ByeReceived():
			_ = s.Record(observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "SIP user BYE received",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  internal_sip.Provider,
					"call_id":   options.Session.GetCallID(),
					"reason":    "bye_received",
				},
			}, observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  internal_sip.Provider,
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

	// Context cancellation is a safety net when Talk cannot drive teardown.
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
				"provider":  internal_sip.Provider,
				"call_id":   options.Session.GetCallID(),
				"reason":    reason,
			},
		})
		if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
			s.Input(msg)
		}
		s.Close()
	}()

	s.session = options.Session
	s.lifecycle = options.Lifecycle
	isInbound := options.Session.GetInfo().Direction == sip_runtime.CallDirectionInbound
	if !isInbound {
		s.assistantOutputActive.Store(true)
	}
	mediaPort, err := internal_sip.NewMediaPort(internal_sip.MediaPortConfig{
		Context:    s.Ctx,
		Logger:     options.Logger,
		Session:    options.Session,
		Resampler:  s.Resampler(),
		StreamSink: s.Input,
		Record:     s.Record,
	})
	if err != nil {
		return nil, err
	}
	s.mediaPort = mediaPort
	if isInbound {
		s.mediaPort.StartInput()
	} else {
		s.mediaPort.Start()
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallMediaStarted,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  internal_sip.Provider,
				"call_id":   options.Session.GetCallID(),
			},
		}, observability.RecordMetadata{
			Metadata: []*protos.Metadata{
				{Key: observability.MetadataClientChannel, Value: internal_sip.Provider},
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusInProgress,
				Description: "SIP media started",
			}},
		})
	}
	s.Input(s.CreateConnectionRequest())
	_ = s.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallSessionConnected,
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  internal_sip.Provider,
			"call_id":   options.Session.GetCallID(),
		},
	}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{
			{Key: observability.MetadataClientChannel, Value: internal_sip.Provider},
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
			"provider":  internal_sip.Provider,
			"call_id":   options.Session.GetCallID(),
			"codec":     mediaPort.CodecName(),
			"rtp_port":  fmt.Sprintf("%d", localPort),
			"local_ip":  localIP,
		},
	})

	return s, nil
}

func (s *Streamer) Context() context.Context {
	return s.Ctx
}

func (s *Streamer) Send(response internal_type.Stream) error {
	if s.closed.Load() {
		return sip_runtime.ErrSessionClosed
	}
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if s.mediaPort != nil {
			s.mediaPort.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			s.markAssistantAudioReady(content.Audio)
			if !s.assistantOutputActive.Load() {
				s.queueAssistantAudio(content.Audio, data.GetCompleted())
				return nil
			}
			if s.mediaPort == nil {
				return nil
			}
			return s.mediaPort.HandleAssistantAudio(content.Audio, data.GetCompleted())
		}
	case *protos.ConversationInterruption:
		if s.mediaPort != nil {
			s.mediaPort.HandleInterrupt()
		}
	case *protos.ConversationDisconnection:
		_ = s.Disconnect(data.GetType())
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallHangup,
			Attributes: observability.Attributes{
				"component":          observability.ComponentCall.String(),
				"provider":           internal_sip.Provider,
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
					"provider":    internal_sip.Provider,
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
						"provider":    internal_sip.Provider,
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
	if !s.assistantOutputActive.CompareAndSwap(false, true) {
		return
	}
	if s.mediaPort != nil {
		s.mediaPort.StartOutput()
		s.mediaPort.StartBridgeRecorder()
		_ = s.Record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallMediaStarted,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  internal_sip.Provider,
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
	s.outputMu.Lock()
	frames := s.pendingAssistantAudioFrames
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()
	for _, frame := range frames {
		if s.mediaPort != nil {
			if err := s.mediaPort.HandleAssistantAudio(frame.audio, frame.completed); err != nil {
				_ = s.Record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP queued assistant audio delivery failed",
					Attributes: observability.Attributes{
						"component": observability.ComponentCall.String(),
						"provider":  internal_sip.Provider,
						"error":     err.Error(),
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
}

func (s *Streamer) queueAssistantAudio(audio []byte, completed bool) {
	audioCopy := make([]byte, len(audio))
	copy(audioCopy, audio)
	s.outputMu.Lock()
	s.pendingAssistantAudioFrames = append(s.pendingAssistantAudioFrames, assistantAudioFrame{
		audio:     audioCopy,
		completed: completed,
	})
	s.outputMu.Unlock()
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
				"provider":  internal_sip.Provider,
				"call_id":   session.GetCallID(),
				"status":    "assistant_audio_ready",
			},
		})
	}
}

func (s *Streamer) EnterTransferMode(targets []string, postTransferAction, ringtoneEnum string) {
	if s.mediaPort != nil && !s.mediaPort.EnterTransferMode(ringtoneEnum) {
		return
	}

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
			"provider":  internal_sip.Provider,
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

func (s *Streamer) SendTransferEvent(event internal_type.Stream) {
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
	_ = s.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallStatus,
		Attributes: observability.Attributes{
			"component": observability.ComponentCall.String(),
			"provider":  internal_sip.Provider,
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
					"provider":  internal_sip.Provider,
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
	s.outputMu.Lock()
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()
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
			"provider":  internal_sip.Provider,
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
				"provider":  internal_sip.Provider,
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
				"provider":  internal_sip.Provider,
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
				"provider":  internal_sip.Provider,
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
