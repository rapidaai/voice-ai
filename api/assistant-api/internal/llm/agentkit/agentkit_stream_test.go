package internal_llm_agentkit

import (
	"context"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRead_ContextCancelled(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(ctx, comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not exit after context cancellation")
	}
}

func TestRead_RecvEOFExitsGracefully(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	talker.recvCh <- recvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(context.Background(), comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not exit on EOF")
	}

	errPkts := findPackets[internal_type.LLMErrorPacket](collector.all())
	assert.Empty(t, errPkts)
}

func TestRead_RecvCanceledExitsGracefully(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	talker.recvCh <- recvResult{err: status.Error(codes.Canceled, "client closed")}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(context.Background(), comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not exit on Canceled")
	}

	errPkts := findPackets[internal_type.LLMErrorPacket](collector.all())
	assert.Empty(t, errPkts)
}

func TestRead_RecvUnavailable(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	talker.recvCh <- recvResult{err: status.Error(codes.Unavailable, "gone")}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(context.Background(), comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not exit on Unavailable")
	}

	errPkts := findPackets[internal_type.LLMErrorPacket](collector.all())
	require.Len(t, errPkts, 1)
	assert.Equal(t, internal_type.LLMSystemPanic, errPkts[0].Type)
	assert.Contains(t, errPkts[0].Error.Error(), "server unavailable")
}

func TestRead_ProcessesMultipleMessages(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	talker.recvCh <- recvResult{out: &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:      "m1",
				Message: &protos.ConversationAssistantMessage_Text{Text: "hi"},
			},
		},
	}}
	talker.recvCh <- recvResult{out: &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:      "m1",
				Message: &protos.ConversationAssistantMessage_Text{Text: " there"},
			},
		},
	}}
	talker.recvCh <- recvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(context.Background(), comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not exit")
	}

	pkts := collector.all()
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	assert.Len(t, deltas, 2)
	errPkts := findPackets[internal_type.LLMErrorPacket](pkts)
	assert.Empty(t, errPkts)
}

func TestE2E_ReadProcessesAndExitsGracefullyOnEOF(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	talker.recvCh <- recvResult{out: &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "r1", Message: &protos.ConversationAssistantMessage_Text{Text: "chunk"},
		}},
	}}
	talker.recvCh <- recvResult{out: &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "r1", Completed: true,
			Message: &protos.ConversationAssistantMessage_Text{Text: "done"},
		}},
	}}
	talker.recvCh <- recvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Read(context.Background(), comm, e.connection)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Read did not exit")
	}

	pkts := collector.all()
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	errPkts := findPackets[internal_type.LLMErrorPacket](pkts)
	assert.Len(t, deltas, 1)
	assert.Len(t, dones, 1)
	assert.Empty(t, errPkts)
}

func TestWrite_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		resp     *protos.TalkOutput
		wantFunc func(t *testing.T, pkts []internal_type.Packet)
	}{
		{
			name: "initialization_ack",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Initialization{
					Initialization: &protos.ConversationInitialization{
						AssistantConversationId: 42,
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				log, ok := pkts[0].(internal_type.ObservabilityLogRecordPacket)
				require.True(t, ok)
				assert.Equal(t, observability.LevelDebug, log.Record.Level)
				assert.Equal(t, "initialization_ack", log.Record.Attributes["operation"])
				assert.Equal(t, "42", log.Record.Attributes["conversation_id"])
			},
		},
		{
			name: "interruption",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Interruption{
					Interruption: &protos.ConversationInterruption{Id: "ctx-1"},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 3)
				ip, ok := pkts[0].(internal_type.InterruptionDetectedPacket)
				require.True(t, ok)
				assert.Equal(t, "ctx-1", ip.ContextID)
				assert.Equal(t, internal_type.InterruptionSourceWord, ip.Source)
				ev, ok := pkts[1].(internal_type.ObservabilityEventRecordPacket)
				require.True(t, ok)
				assert.Equal(t, observability.LLMDiscarded, ev.Record.Event)
				log, ok := pkts[2].(internal_type.ObservabilityLogRecordPacket)
				require.True(t, ok)
				assert.Equal(t, "interrupt", log.Record.Attributes["operation"])
			},
		},
		{
			name: "user_message",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_User{
					User: &protos.ConversationUserMessage{
						Id:        "user-1",
						Completed: true,
						Message:   &protos.ConversationUserMessage_Text{Text: "synthetic user"},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				userInput, ok := pkts[0].(internal_type.UserInputPacket)
				require.True(t, ok)
				assert.Equal(t, "synthetic user", userInput.Text)
			},
		},
		{
			name: "control_block",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Control{
					Control: &protos.ConversationControl{
						Id:     "ctx-control",
						Action: protos.ConversationControl_CONTROL_ACTION_BLOCK,
						Types: []protos.ConversationControl_Type{
							protos.ConversationControl_CONTROL_TYPE_USER_AUDIO,
							protos.ConversationControl_CONTROL_TYPE_USER_TEXT,
							protos.ConversationControl_CONTROL_TYPE_BARGE_IN,
						},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 3)
				audioPolicy, ok := pkts[0].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, "ctx-control", audioPolicy.ContextID)
				assert.Equal(t, internal_type.PacketNameUserAudioReceived, audioPolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionIgnore, audioPolicy.Policy.Action)

				textPolicy, ok := pkts[1].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.PacketNameUserTextReceived, textPolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionIgnore, textPolicy.Policy.Action)

				bargePolicy, ok := pkts[2].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.PacketNameInterruptionDetected, bargePolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionIgnore, bargePolicy.Policy.Action)
			},
		},
		{
			name: "control_unblock",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Control{
					Control: &protos.ConversationControl{
						Id:     "ctx-control",
						Action: protos.ConversationControl_CONTROL_ACTION_UNBLOCK,
						Types: []protos.ConversationControl_Type{
							protos.ConversationControl_CONTROL_TYPE_USER_AUDIO,
							protos.ConversationControl_CONTROL_TYPE_USER_TEXT,
							protos.ConversationControl_CONTROL_TYPE_BARGE_IN,
						},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 3)
				audioPolicy, ok := pkts[0].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.PacketNameUserAudioReceived, audioPolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionPassthrough, audioPolicy.Policy.Action)

				textPolicy, ok := pkts[1].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.PacketNameUserTextReceived, textPolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionPassthrough, textPolicy.Policy.Action)

				bargePolicy, ok := pkts[2].(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.PacketNameInterruptionDetected, bargePolicy.Policy.Target)
				assert.Equal(t, internal_type.DispatchActionPassthrough, bargePolicy.Policy.Action)
			},
		},
		{
			name: "control_unspecified_ignored",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Control{
					Control: &protos.ConversationControl{
						Id:     "ctx-control",
						Action: protos.ConversationControl_CONTROL_ACTION_UNSPECIFIED,
						Types: []protos.ConversationControl_Type{
							protos.ConversationControl_CONTROL_TYPE_USER_AUDIO,
						},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				assert.Empty(t, pkts)
			},
		},
		{
			name: "control_nil_ignored",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Control{},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				assert.Empty(t, pkts)
			},
		},
		{
			name: "text_delta",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Assistant{
					Assistant: &protos.ConversationAssistantMessage{
						Id:        "msg-1",
						Completed: false,
						Message:   &protos.ConversationAssistantMessage_Text{Text: "hello "},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				delta, ok := pkts[0].(internal_type.LLMResponseDeltaPacket)
				require.True(t, ok)
				assert.Equal(t, "msg-1", delta.ContextID)
				assert.Equal(t, "hello ", delta.Text)
			},
		},
		{
			name: "text_completed",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Assistant{
					Assistant: &protos.ConversationAssistantMessage{
						Id:        "msg-2",
						Completed: true,
						Message:   &protos.ConversationAssistantMessage_Text{Text: "world"},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 3)
				done, ok := pkts[0].(internal_type.LLMResponseDonePacket)
				require.True(t, ok)
				assert.Equal(t, "msg-2", done.ContextID)
				assert.Equal(t, "world", done.Text)
				ev, ok := pkts[1].(internal_type.ObservabilityEventRecordPacket)
				require.True(t, ok)
				assert.Equal(t, observability.LLMCompleted, ev.Record.Event)
				assert.Equal(t, "5", ev.Record.Attributes["response_char_count"])
				metric, ok := pkts[2].(internal_type.ObservabilityMetricRecordPacket)
				require.True(t, ok)
				require.Len(t, metric.Record.Metrics, 1)
				assert.Equal(t, "llm_response_char_count", metric.Record.Metrics[0].Name)
			},
		},
		{
			name: "audio_noop",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Assistant{
					Assistant: &protos.ConversationAssistantMessage{
						Id:      "msg-3",
						Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0x01}},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				assert.Empty(t, pkts)
			},
		},
		{
			name: "tool_call",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_ToolCall{
					ToolCall: &protos.ConversationToolCall{
						Id:     "tc-1",
						ToolId: "tool-42",
						Name:   "get_weather",
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
				require.True(t, ok)
				assert.Equal(t, "tool-42", tc.ToolID)
				assert.Equal(t, "get_weather", tc.Name)
			},
		},
		{
			name: "tool_call_generates_empty_tool_id",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_ToolCall{
					ToolCall: &protos.ConversationToolCall{
						Id:     "tc-1",
						Name:   "get_weather",
						Action: protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED,
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
				require.True(t, ok)
				assert.Equal(t, "tc-1", tc.ContextID)
				assert.Regexp(t, "^agentkit-tool-", tc.ToolID)
				assert.Equal(t, "get_weather", tc.Name)
			},
		},
		{
			name: "tool_call_preserves_empty_name",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_ToolCall{
					ToolCall: &protos.ConversationToolCall{
						Id:     "tc-1",
						Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
				require.True(t, ok)
				assert.Regexp(t, "^agentkit-tool-", tc.ToolID)
				assert.Empty(t, tc.Name)
				assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, tc.Action)
			},
		},
		{
			name: "tool_call_result",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_ToolCallResult{
					ToolCallResult: &protos.ConversationToolCallResult{
						Id:     "tr-1",
						ToolId: "tool-42",
						Name:   "get_weather",
						Action: protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED,
						Result: map[string]string{"ok": "true"},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 2)

				toolResult, ok := pkts[0].(internal_type.LLMToolResultPacket)
				require.True(t, ok)
				assert.Equal(t, "tr-1", toolResult.ContextID)
				assert.Equal(t, "tool-42", toolResult.ToolID)
				assert.Equal(t, "get_weather", toolResult.Name)
				assert.Equal(t, map[string]string{"ok": "true"}, toolResult.Result)

				log, ok := pkts[1].(internal_type.ObservabilityLogRecordPacket)
				require.True(t, ok)
				assert.Equal(t, "tool_result", log.Record.Attributes["operation"])
				assert.Equal(t, "tool-42", log.Record.Attributes["tool_id"])
			},
		},
		{
			name: "error",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Error{
					Error: &protos.Error{
						ErrorCode:    500,
						ErrorMessage: "agent crashed",
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 4)
				errPkt, ok := pkts[0].(internal_type.LLMErrorPacket)
				require.True(t, ok)
				assert.Contains(t, errPkt.Error.Error(), "agent crashed")

				ev, ok := pkts[1].(internal_type.ObservabilityEventRecordPacket)
				require.True(t, ok)
				assert.Equal(t, observability.LLMError, ev.Record.Event)
				assert.Equal(t, "500", ev.Record.Attributes["code"])
				log, ok := pkts[2].(internal_type.ObservabilityLogRecordPacket)
				require.True(t, ok)
				assert.Equal(t, observability.LevelError, log.Record.Level)

				dir, ok := pkts[3].(internal_type.LLMToolCallPacket)
				require.True(t, ok)
				assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dir.Action)
			},
		},
		{
			name: "observability_log",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Observability{
					Observability: &protos.ObservabilityRecord{
						Record: &protos.ObservabilityRecord_Log{Log: &protos.ObservabilityLogRecord{
							Id:         "log-1",
							Scope:      internal_type.ObservabilityRecordScopeConversation.String(),
							Level:      string(observability.LevelInfo),
							Message:    "remote.log",
							Attributes: map[string]string{"source": "agentkit"},
							Context:    map[string]string{"context_id": "ctx-log"},
							OccurredAt: timestamppb.Now(),
						}},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				log, ok := pkts[0].(internal_type.ObservabilityLogRecordPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.ObservabilityRecordScopeAssistantMessage, log.Scope)
				assert.Equal(t, "log-1", log.Record.ID)
				assert.Equal(t, observability.LevelInfo, log.Record.Level)
				assert.Equal(t, "remote.log", log.Record.Message)
				assert.Equal(t, "agentkit", log.Record.Attributes["source"])
			},
		},
		{
			name: "observability_event",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Observability{
					Observability: &protos.ObservabilityRecord{
						Record: &protos.ObservabilityRecord_Event{Event: &protos.ObservabilityEventRecord{
							Id:         "event-1",
							Scope:      internal_type.ObservabilityRecordScopeAssistantMessage.String(),
							Component:  "llm",
							Event:      "response.created",
							Attributes: map[string]string{"phase": "stream"},
							Context:    map[string]string{"context_id": "ctx-event"},
							OccurredAt: timestamppb.Now(),
						}},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				event, ok := pkts[0].(internal_type.ObservabilityEventRecordPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.ObservabilityRecordScopeAssistantMessage, event.Scope)
				assert.Equal(t, "event-1", event.Record.ID)
				assert.Equal(t, observability.ComponentName("agentkit.llm"), event.Record.Component)
				assert.Equal(t, observability.EventName("agentkit.response.created"), event.Record.Event)
				assert.Equal(t, "stream", event.Record.Attributes["phase"])
			},
		},
		{
			name: "observability_metric",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_Observability{
					Observability: &protos.ObservabilityRecord{
						Record: &protos.ObservabilityRecord_Metric{Metric: &protos.ObservabilityMetricRecord{
							Id:          "metric-1",
							Scope:       internal_type.ObservabilityRecordScopeAssistantMessage.String(),
							Name:        "latency_ms",
							Value:       "42",
							Description: "latency",
							Attributes:  map[string]string{"unit": "ms"},
							Context:     map[string]string{"context_id": "ctx-metric"},
							OccurredAt:  timestamppb.Now(),
						}},
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				metric, ok := pkts[0].(internal_type.ObservabilityMetricRecordPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.ObservabilityRecordScopeAssistantMessage, metric.Scope)
				assert.Equal(t, "metric-1", metric.Record.ID)
				require.Len(t, metric.Record.Metrics, 1)
				assert.Equal(t, "agentkit.latency_ms", metric.Record.Metrics[0].Name)
				assert.Equal(t, "42", metric.Record.Metrics[0].Value)
				assert.Equal(t, "latency", metric.Record.Metrics[0].Description)
				assert.Equal(t, "ms", metric.Record.Attributes["unit"])
			},
		},
		{
			name: "tool_call_with_action",
			resp: &protos.TalkOutput{
				Data: &protos.TalkOutput_ToolCall{
					ToolCall: &protos.ConversationToolCall{
						Id:     "d-1",
						Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
					},
				},
			},
			wantFunc: func(t *testing.T, pkts []internal_type.Packet) {
				require.Len(t, pkts, 1)
				tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
				require.True(t, ok)
				assert.Equal(t, "d-1", tc.ContextID)
				assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, tc.Action)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestExecutor()
			comm, collector := newTestComm()
			e.Write(context.Background(), comm, tt.resp)
			tt.wantFunc(t, collector.all())
		})
	}
}

func TestWrite_CompletedTextContextID(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        "unique-ctx",
				Completed: true,
				Message:   &protos.ConversationAssistantMessage_Text{Text: "done"},
			},
		},
	}
	e.Write(context.Background(), comm, resp)

	pkts := collector.all()
	done, ok := findPacket[internal_type.LLMResponseDonePacket](pkts)
	require.True(t, ok)
	assert.Equal(t, "unique-ctx", done.ContextID)

	ev, ok := findPacket[internal_type.ObservabilityEventRecordPacket](pkts)
	require.True(t, ok)
	assert.Equal(t, "unique-ctx", ev.ContextID)
}

func TestWrite_FirstDeltaAddsTTFTAndCompletedAddsTRT(t *testing.T) {
	e := newTestExecutor()
	e.requestStartedAt = time.Now().Add(-25 * time.Millisecond)
	e.waitingForFirstResponse = true
	comm, collector := newTestComm()

	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        "ctx-latency",
				Completed: false,
				Message:   &protos.ConversationAssistantMessage_Text{Text: "he"},
			},
		},
	})
	pkts := collector.all()
	require.Len(t, pkts, 2)
	delta, ok := pkts[0].(internal_type.LLMResponseDeltaPacket)
	require.True(t, ok)
	assert.Equal(t, "he", delta.Text)
	ttftMetric, ok := pkts[1].(internal_type.ObservabilityMetricRecordPacket)
	require.True(t, ok)
	require.Len(t, ttftMetric.Record.Metrics, 1)
	assert.Equal(t, observability.MetricAgentTTFTMs, ttftMetric.Record.Metrics[0].Name)
	ttftMs, err := strconv.ParseInt(ttftMetric.Record.Metrics[0].Value, 10, 64)
	require.NoError(t, err)
	assert.Positive(t, ttftMs)

	collector.reset()

	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        "ctx-latency",
				Completed: true,
				Message:   &protos.ConversationAssistantMessage_Text{Text: "hello"},
			},
		},
	})

	metrics := findPackets[internal_type.ObservabilityMetricRecordPacket](collector.all())
	require.Len(t, metrics, 1)
	require.Len(t, metrics[0].Record.Metrics, 2)
	assert.Equal(t, "llm_response_char_count", metrics[0].Record.Metrics[0].Name)
	assert.Equal(t, observability.MetricAgentTRTMs, metrics[0].Record.Metrics[1].Name)
	ms, err := strconv.ParseInt(metrics[0].Record.Metrics[1].Value, 10, 64)
	require.NoError(t, err)
	assert.Positive(t, ms)
}

func TestWrite_CompletedTextFirstPacketAddsTTFTAndTRT(t *testing.T) {
	e := newTestExecutor()
	e.requestStartedAt = time.Now().Add(-25 * time.Millisecond)
	e.waitingForFirstResponse = true
	comm, collector := newTestComm()

	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        "ctx-completed-latency",
				Completed: true,
				Message:   &protos.ConversationAssistantMessage_Text{Text: "hello"},
			},
		},
	})

	metrics := findPackets[internal_type.ObservabilityMetricRecordPacket](collector.all())
	require.Len(t, metrics, 1)
	require.Len(t, metrics[0].Record.Metrics, 3)
	assert.Equal(t, "llm_response_char_count", metrics[0].Record.Metrics[0].Name)
	assert.Equal(t, observability.MetricAgentTTFTMs, metrics[0].Record.Metrics[1].Name)
	assert.Equal(t, observability.MetricAgentTRTMs, metrics[0].Record.Metrics[2].Name)
}

func TestWrite_ToolResultFailed(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.TalkOutput{
		Data: &protos.TalkOutput_ToolCallResult{
			ToolCallResult: &protos.ConversationToolCallResult{
				Id:     "tr-2",
				ToolId: "tool-99",
				Name:   "calculator",
			},
		},
	}
	e.Write(context.Background(), comm, resp)

	logs := findPackets[internal_type.ObservabilityLogRecordPacket](collector.all())
	require.Len(t, logs, 1)
	assert.Equal(t, "tool_result", logs[0].Record.Attributes["operation"])
}

func TestWrite_ErrorMessageFormat(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.TalkOutput{
		Data: &protos.TalkOutput_Error{
			Error: &protos.Error{
				ErrorCode:    403,
				ErrorMessage: "forbidden",
			},
		},
	}
	e.Write(context.Background(), comm, resp)

	errPkts := findPackets[internal_type.LLMErrorPacket](collector.all())
	require.Len(t, errPkts, 1)
	assert.ErrorIs(t, errPkts[0].Error, ErrAgentkitResponse)
	assert.Contains(t, errPkts[0].Error.Error(), "agentkit error 403: forbidden")
}

func TestWrite_StaleContext_Dropped(t *testing.T) {
	e := newTestExecutor()
	e.activeContextID = "ctx-active"
	comm, collector := newTestComm()

	resp := &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        "ctx-stale",
				Completed: true,
				Message:   &protos.ConversationAssistantMessage_Text{Text: "ignore"},
			},
		},
	}
	e.Write(context.Background(), comm, resp)
	assert.Empty(t, collector.all())
}
