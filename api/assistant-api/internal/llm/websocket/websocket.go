// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_llm_websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type websocketExecutor struct {
	logger                  commons.Logger
	conn                    *websocket.Conn
	writeMu                 sync.Mutex
	contextMu               sync.RWMutex
	currentID               string
	requestStartedAt        time.Time
	waitingForFirstResponse bool
}

type options struct {
	ctx           context.Context
	logger        commons.Logger
	communication internal_type.Communication
	configuration *protos.ConversationInitialization
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithCommunication(communication internal_type.Communication) Option {
	return func(options *options) {
		options.communication = communication
	}
}

func WithConfiguration(configuration *protos.ConversationInitialization) Option {
	return func(options *options) {
		options.configuration = configuration
	}
}

// New creates and initializes a WebSocket-based assistant executor.
func New(opts ...Option) (*websocketExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.communication == nil {
		return nil, errors.New("websocket: communication is required")
	}
	if options.configuration == nil {
		return nil, errors.New("websocket: configuration is required")
	}
	assistant, err := options.communication.Assistant()
	if err != nil || assistant == nil {
		return nil, errors.New("websocket: assistant is required")
	}
	if assistant.AssistantProviderWebsocket == nil {
		return nil, errors.New("websocket: provider configuration is required")
	}
	executor := &websocketExecutor{logger: options.logger}
	if err := executor.initialize(options.ctx, options.communication); err != nil {
		_ = executor.Close(options.ctx)
		return nil, err
	}
	return executor, nil
}

// Name returns the executor name identifier.
func (e *websocketExecutor) Name() string {
	return "websocket"
}

func (e *websocketExecutor) initialize(ctx context.Context, comm internal_type.Communication) error {
	start := time.Now()
	assistant, err := comm.Assistant()
	if err != nil || assistant == nil {
		return errors.New("websocket: assistant is required")
	}
	conversation, err := comm.Conversation()
	if err != nil || conversation == nil {
		return errors.New("websocket: conversation is required")
	}
	provider := assistant.AssistantProviderWebsocket
	if provider == nil {
		comm.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", e.Name(), "websocket provider is not enabled"),
				Attributes: observability.Attributes{
					"component": observability.ComponentAgent.String(),
					"provider":  e.Name(),
					"options":   observability.AttributeValue(comm.GetOptions()),
					"error":     "websocket provider is not enabled",
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("websocket provider is not enabled")
	}

	// Connect
	if err := e.connect(ctx, provider); err != nil {
		comm.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", e.Name(), err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentAgent.String(),
					"provider":   e.Name(),
					"options":    observability.AttributeValue(comm.GetOptions()),
					"url":        provider.Url,
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	// Start listener - stops on context cancel or server close
	utils.Go(ctx, func() {
		if err := e.listen(ctx, comm.OnPacket); err != nil && ctx.Err() == nil {
			comm.OnPacket(ctx, internal_type.LLMToolCallPacket{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, Arguments: map[string]string{"reason": err.Error()}})
		}
	})

	// Send initial configuration
	if err := e.sendConfiguration(provider.AssistantId, conversation.Id); err != nil {
		comm.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", e.Name(), err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentAgent.String(),
					"provider":   e.Name(),
					"options":    observability.AttributeValue(comm.GetOptions()),
					"url":        provider.Url,
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		return fmt.Errorf("failed to send configuration: %w", err)
	}
	comm.OnPacket(ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": e.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricLLMInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "LLM initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: fmt.Sprintf("%s: initialization completed", e.Name()),
				Attributes: observability.Attributes{
					"component": observability.ComponentAgent.String(),
					"provider":  e.Name(),
					"url":       provider.Url,
					"options":   observability.AttributeValue(comm.GetOptions()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// connect establishes the WebSocket connection.
func (e *websocketExecutor) connect(ctx context.Context, provider *internal_assistant_entity.AssistantProviderWebsocket) error {
	headers := http.Header{}
	for k, v := range provider.Headers {
		headers.Set(k, v)
	}

	wsURL, err := url.Parse(provider.Url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	query := wsURL.Query()
	for k, v := range provider.Parameters {
		query.Set(k, v)
	}
	wsURL.RawQuery = query.Encode()

	dialer := websocket.Dialer{HandshakeTimeout: 30 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL.String(), headers)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	conn.SetReadLimit(10 * 1024 * 1024)
	e.conn = conn
	return nil
}

// send writes a message to the WebSocket.
func (e *websocketExecutor) send(msg Request) error {
	e.writeMu.Lock()
	defer e.writeMu.Unlock()
	if e.conn == nil {
		return fmt.Errorf("not connected")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return e.conn.WriteMessage(websocket.TextMessage, data)
}

// sendConfiguration sends the initial configuration.
func (e *websocketExecutor) sendConfiguration(assistantId uint64, conversationID uint64) error {
	return e.send(Request{
		Type:      TypeConfiguration,
		Timestamp: time.Now().UnixMilli(),
		Data: ConfigurationData{
			AssistantID:    assistantId,
			ConversationID: conversationID,
		},
	})
}

func (e *websocketExecutor) setCurrentContextID(id string) {
	e.contextMu.Lock()
	e.currentID = id
	if strings.TrimSpace(id) != "" {
		e.requestStartedAt = time.Now()
		e.waitingForFirstResponse = true
	} else {
		e.requestStartedAt = time.Time{}
		e.waitingForFirstResponse = false
	}
	e.contextMu.Unlock()
}

func (e *websocketExecutor) isCurrentContextID(id string) bool {
	clean := strings.TrimSpace(id)
	e.contextMu.RLock()
	defer e.contextMu.RUnlock()
	current := strings.TrimSpace(e.currentID)
	// Preserve historical behavior for id-less packets while still gating stale ids.
	if clean == "" || current == "" {
		return true
	}
	return clean == current
}

func (e *websocketExecutor) sendUserMessage(contextID string, text string) error {
	if strings.TrimSpace(contextID) == "" {
		return nil
	}
	e.setCurrentContextID(contextID)
	return e.send(Request{
		Type:      TypeUserMessage,
		Timestamp: time.Now().UnixMilli(),
		Data:      UserMessageData{ID: contextID, Content: text},
	})
}

// listen reads messages from WebSocket until context is cancelled or connection closes.
func (e *websocketExecutor) listen(ctx context.Context, onPacket func(ctx context.Context, packet ...internal_type.Packet) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Allow periodic context checks
		e.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		_, data, err := e.conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				onPacket(ctx, internal_type.LLMToolCallPacket{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, Arguments: map[string]string{"reason": "websocket closed the connection"}})
				return nil
			}
			e.contextMu.RLock()
			currentID := e.currentID
			e.contextMu.RUnlock()
			onPacket(ctx,
				internal_type.ObservabilityLogRecordPacket{
					ContextID: currentID,
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "websocket read failed",
						Attributes: observability.Attributes{
							"component":  observability.ComponentAgent.String(),
							"operation":  "listen",
							"provider":   e.Name(),
							"context_id": currentID,
							"error":      err.Error(),
							"error_type": fmt.Sprintf("%T", err),
						},
						OccurredAt: time.Now(),
					},
				},
				internal_type.LLMToolCallPacket{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, Arguments: map[string]string{"reason": err.Error()}},
			)
			return nil
		}

		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			e.logger.Errorf("Invalid response: %v", err)
			e.contextMu.RLock()
			currentID := e.currentID
			e.contextMu.RUnlock()
			onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: currentID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "websocket response decode failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentAgent.String(),
						"operation":  "decode_response",
						"provider":   e.Name(),
						"context_id": currentID,
						"error":      err.Error(),
						"error_type": fmt.Sprintf("%T", err),
					},
					OccurredAt: time.Now(),
				},
			})
			continue
		}

		e.handleResponse(ctx, &resp, onPacket)
	}
}

// handleResponse processes a single response from the server.
func (e *websocketExecutor) handleResponse(ctx context.Context, resp *Response, onPacket func(ctx context.Context, packet ...internal_type.Packet) error) {
	switch resp.Type {
	case TypeError:
		var d ErrorData
		json.Unmarshal(resp.Data, &d)
		e.logger.Errorf("Error: %d - %s", d.Code, d.Message)
		e.contextMu.Lock()
		currentID := e.currentID
		e.requestStartedAt = time.Time{}
		e.waitingForFirstResponse = false
		e.contextMu.Unlock()
		onPacket(ctx,
			internal_type.LLMErrorPacket{
				ContextID: currentID,
				Error:     fmt.Errorf("websocket error %d: %s", d.Code, d.Message),
				Type:      internal_type.LLMSystemPanic,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: currentID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(currentID, observability.ComponentAgent, observability.AgentError, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   e.Name(),
					"context_id": currentID,
					"code":       fmt.Sprintf("%d", d.Code),
					"error":      d.Message,
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: currentID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "websocket llm response failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentAgent.String(),
						"operation":  "response",
						"provider":   e.Name(),
						"context_id": currentID,
						"code":       fmt.Sprintf("%d", d.Code),
						"error":      d.Message,
					},
					OccurredAt: time.Now(),
				},
			},
		)

	case TypeStream:
		var d StreamData
		json.Unmarshal(resp.Data, &d)
		if !e.isCurrentContextID(d.ID) {
			return
		}
		now := time.Now()
		e.contextMu.Lock()
		requestStartedAt := e.requestStartedAt
		publishTTFT := e.waitingForFirstResponse
		e.waitingForFirstResponse = false
		e.contextMu.Unlock()

		if publishTTFT && !requestStartedAt.IsZero() {
			onPacket(ctx,
				internal_type.LLMResponseDeltaPacket{ContextID: d.ID, Text: d.Content},
				internal_type.ObservabilityMetricRecordPacket{
					ContextID: d.ID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{"provider": e.Name()},
						Metrics: []*protos.Metric{{
							Name:        observability.MetricAgentTTFTMs,
							Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
							Description: "Agent time to first token in milliseconds",
						}},
					},
				},
			)
			return
		}
		onPacket(ctx, internal_type.LLMResponseDeltaPacket{ContextID: d.ID, Text: d.Content})

	case TypeComplete:
		var d CompleteData
		json.Unmarshal(resp.Data, &d)
		if !e.isCurrentContextID(d.ID) {
			return
		}
		if d.Content != "" {
			now := time.Now()
			e.contextMu.Lock()
			requestStartedAt := e.requestStartedAt
			publishTTFT := e.waitingForFirstResponse
			e.requestStartedAt = time.Time{}
			e.waitingForFirstResponse = false
			e.contextMu.Unlock()
			packets := []internal_type.Packet{
				internal_type.LLMResponseDonePacket{
					ContextID: d.ID,
					Text:      d.Content,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: d.ID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.NewMessageRecord(d.ID, observability.ComponentAgent, observability.AgentCompleted, observability.MessageRoleAssistant, observability.Attributes{
						"provider":            e.Name(),
						"context_id":          d.ID,
						"response_char_count": fmt.Sprintf("%d", len(d.Content)),
					}),
				},
			}
			metrics := []*protos.Metric{{
				Name:        observability.MetricAgentResponseCharCount,
				Value:       fmt.Sprintf("%d", len(d.Content)),
				Description: "Agent response character count",
			}}
			if !requestStartedAt.IsZero() {
				if publishTTFT {
					metrics = append(metrics, &protos.Metric{
						Name:        observability.MetricAgentTTFTMs,
						Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
						Description: "Agent time to first token in milliseconds",
					})
				}
				metrics = append(metrics, &protos.Metric{
					Name:        observability.MetricAgentTRTMs,
					Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
					Description: "Agent total response time in milliseconds",
				})
			}
			packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
				ContextID: d.ID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{"provider": e.Name()},
					Metrics:    metrics,
				},
			})
			if !requestStartedAt.IsZero() {
				packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
					ContextID: d.ID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.NewLLMDurationUsageRecord(e.Name(), now.Sub(requestStartedAt), observability.Attributes{
						"context_id":          d.ID,
						"response_char_count": fmt.Sprintf("%d", len(d.Content)),
					}),
				})
			}
			onPacket(ctx, packets...)
		}

	case TypeInterruption:
		var d InterruptionData
		json.Unmarshal(resp.Data, &d)
		if !e.isCurrentContextID(d.ID) {
			return
		}
		source := internal_type.InterruptionSourceWord
		if d.Source == "vad" {
			source = internal_type.InterruptionSourceVad
		}
		onPacket(ctx,
			internal_type.InterruptionDetectedPacket{ContextID: d.ID, Source: source},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: d.ID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(d.ID, observability.ComponentAgent, observability.AgentDiscarded, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   e.Name(),
					"context_id": d.ID,
					"reason":     "interruption",
					"source":     d.Source,
				}),
			},
		)

	case TypeClose:
		var d CloseData
		json.Unmarshal(resp.Data, &d)
		onPacket(ctx, internal_type.LLMToolCallPacket{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, Arguments: map[string]string{"reason": d.Reason}})

	case TypePing:
		e.send(Request{Type: TypePong, Timestamp: time.Now().UnixMilli()})
	}
}

// mapToolAction maps tool names from websocket to conversation actions.
// func (e *websocketExecutor) mapToolAction(name string) protos.AssistantConversationAction_ActionType {
// 	switch name {
// 	case "disconnect", "end_conversation", "hangup":
// 		return protos.AssistantConversationAction_END_CONVERSATION
// 	default:
// 		return protos.AssistantConversationAction_ACTION_UNSPECIFIED
// 	}
// }

// Execute sends a packet to the WebSocket server.
func (e *websocketExecutor) Execute(ctx context.Context, comm internal_type.Communication, packet internal_type.Packet) error {
	switch p := packet.(type) {
	case internal_type.UserInputPacket:
		return e.sendUserMessage(p.ContextID, p.Text)
	case internal_type.UserTextReceivedPacket:
		return e.sendUserMessage(p.ContextID, p.Text)
	case internal_type.InjectMessagePacket:
		return nil
	case internal_type.InterruptionDetectedPacket:
		e.setCurrentContextID("")
		return nil
	default:
		return fmt.Errorf("unsupported packet: %T", packet)
	}
}

// Close terminates the WebSocket connection.
func (e *websocketExecutor) Close(ctx context.Context) error {
	e.writeMu.Lock()
	defer e.writeMu.Unlock()
	if e.conn != nil {
		e.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		e.conn.Close()
		e.conn = nil
	}
	e.contextMu.Lock()
	e.currentID = ""
	e.requestStartedAt = time.Time{}
	e.waitingForFirstResponse = false
	e.contextMu.Unlock()
	return nil
}
