package internal_llm_agentkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type packetCollector struct {
	mu   sync.Mutex
	pkts []internal_type.Packet
}

func (c *packetCollector) collect(_ context.Context, pkts ...internal_type.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pkts = append(c.pkts, pkts...)
	return nil
}

func (c *packetCollector) all() []internal_type.Packet {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]internal_type.Packet, len(c.pkts))
	copy(out, c.pkts)
	return out
}

func (c *packetCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pkts = nil
}

// =============================================================================
// Mock: mockTalker — grpc.BidiStreamingClient[protos.TalkInput, protos.TalkOutput]
// =============================================================================

type recvResult struct {
	out *protos.TalkOutput
	err error
}

type mockTalker struct {
	mu        sync.Mutex
	sendCalls []*protos.TalkInput
	sendErr   error
	recvCh    chan recvResult
	closeSent atomic.Bool
}

func newMockTalker() *mockTalker {
	return &mockTalker{
		recvCh: make(chan recvResult, 16),
	}
}

func newTestConnection(talker *mockTalker) *AgentkitConnection {
	return &AgentkitConnection{stream: talker}
}

func (m *mockTalker) Send(req *protos.TalkInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, req)
	return m.sendErr
}

func (m *mockTalker) Recv() (*protos.TalkOutput, error) {
	r, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return r.out, r.err
}

func (m *mockTalker) CloseSend() error {
	m.closeSent.Store(true)
	return nil
}

func (m *mockTalker) Header() (metadata.MD, error) { return nil, nil }
func (m *mockTalker) Trailer() metadata.MD         { return nil }
func (m *mockTalker) Context() context.Context     { return context.Background() }
func (m *mockTalker) SendMsg(any) error            { return nil }
func (m *mockTalker) RecvMsg(any) error            { return nil }

// =============================================================================
// Mock: mockCommunication — satisfies internal_type.Communication
// =============================================================================

type mockCommunication struct {
	internal_type.Communication // embedded nil — panics if unoverridden methods called
	collector                   *packetCollector
	assistant                   *internal_assistant_entity.Assistant
	conversation                *internal_conversation_entity.AssistantConversation
	options                     utils.Option
}

func (m *mockCommunication) Assistant() (*internal_assistant_entity.Assistant, error) {
	return m.assistant, nil
}

func (m *mockCommunication) Conversation() (*internal_conversation_entity.AssistantConversation, error) {
	return m.conversation, nil
}

func (m *mockCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	return m.collector.collect(ctx, pkts...)
}

func (m *mockCommunication) GetOptions() utils.Option {
	return m.options
}

type testAgentKitServer struct {
	protos.UnimplementedAgentKitServer
	talkFn func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error
}

func (s *testAgentKitServer) Talk(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
	if s.talkFn == nil {
		return nil
	}
	return s.talkFn(stream)
}

// =============================================================================
// Helpers
// =============================================================================

func newTestExecutor(talkers ...*mockTalker) *agentkitExecutor {
	lgr, _ := commons.NewApplicationLogger()
	ctx, cancel := context.WithCancelCause(context.Background())
	executor := &agentkitExecutor{logger: lgr, ctx: ctx, cancel: cancel}
	if len(talkers) > 0 && talkers[0] != nil {
		executor.connection = newTestConnection(talkers[0])
	}
	return executor
}

func newTestComm() (*mockCommunication, *packetCollector) {
	c := &packetCollector{}
	return &mockCommunication{collector: c}, c
}

func newTestAgentkitComm(provider *internal_assistant_entity.AssistantProviderAgentkit, conversationID uint64) (*mockCommunication, *packetCollector) {
	comm, collector := newTestComm()
	comm.assistant = &internal_assistant_entity.Assistant{
		AssistantProviderAgentkit: provider,
	}
	comm.assistant.Id = 77
	comm.conversation = &internal_conversation_entity.AssistantConversation{}
	comm.conversation.Id = conversationID
	return comm, collector
}

func startAgentKitTestServer(
	t *testing.T,
	talkFn func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error,
) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	protos.RegisterAgentKitServer(server, &testAgentKitServer{talkFn: talkFn})
	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}

func acquireClosedTCPAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

// findPacket returns the first packet of type T from the collector.
func findPacket[T internal_type.Packet](pkts []internal_type.Packet) (T, bool) {
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// findPackets returns all packets of type T from the collector.
func findPackets[T internal_type.Packet](pkts []internal_type.Packet) []T {
	var out []T
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// findActionToolCalls returns LLMToolCallPackets that have a non-UNSPECIFIED Action.
func findActionToolCalls(pkts []internal_type.Packet) []internal_type.LLMToolCallPacket {
	var out []internal_type.LLMToolCallPacket
	for _, p := range pkts {
		if tc, ok := p.(internal_type.LLMToolCallPacket); ok && tc.Action != protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED {
			out = append(out, tc)
		}
	}
	return out
}

// =============================================================================
// Tests: New — 3 cases
// =============================================================================

func TestNew_ReturnsErrorWhenConnectionFails(t *testing.T) {
	transportSecurity := TransportSecurityPlaintext
	comm, _ := newTestAgentkitComm(&internal_assistant_entity.AssistantProviderAgentkit{
		Url:               acquireClosedTCPAddress(t),
		TransportSecurity: &transportSecurity,
	}, 2002)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	e, err := New(
		WithContext(ctx),
		WithLogger(newTestExecutor().logger),
		WithCommunication(comm),
		WithConfiguration(&protos.ConversationInitialization{}),
	)
	require.Error(t, err)
	assert.Nil(t, e)
	assert.True(
		t,
		errors.Is(err, ErrAgentkitInitializationConnect) || errors.Is(err, ErrAgentkitInitializationOpenTalkStream),
		"unexpected New error: %v",
		err,
	)
}

func TestNew_SendsInitializationAndEmitsInitializedEvent(t *testing.T) {
	receivedInitialization := make(chan *protos.TalkInput, 1)
	addr, shutdown := startAgentKitTestServer(t, func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		receivedInitialization <- req

		if err := stream.Send(&protos.TalkOutput{
			Data: &protos.TalkOutput_Initialization{
				Initialization: &protos.ConversationInitialization{
					AssistantConversationId: req.GetInitialization().GetAssistantConversationId(),
				},
			},
		}); err != nil {
			return err
		}

		for {
			if _, err := stream.Recv(); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		}
	})
	defer shutdown()

	transportSecurity := TransportSecurityPlaintext
	comm, collector := newTestAgentkitComm(&internal_assistant_entity.AssistantProviderAgentkit{
		AssistantProvider: internal_assistant_entity.AssistantProvider{
			AssistantId: 5005,
		},
		Url:               addr,
		TransportSecurity: &transportSecurity,
	}, 3003)

	e, err := New(
		WithLogger(newTestExecutor().logger),
		WithCommunication(comm),
		WithConfiguration(&protos.ConversationInitialization{}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = e.Close(context.Background())
	})

	select {
	case req := <-receivedInitialization:
		initReq := req.GetInitialization()
		require.NotNil(t, initReq)
		assert.Equal(t, uint64(3003), initReq.GetAssistantConversationId())
		require.NotNil(t, initReq.GetAssistant())
		assert.Equal(t, uint64(5005), initReq.GetAssistant().GetAssistantId())
	case <-time.After(time.Second):
		t.Fatal("did not receive initialization request on server")
	}

	pkts := collector.all()
	eventPackets := findPackets[internal_type.ObservabilityEventRecordPacket](pkts)
	for _, event := range eventPackets {
		assert.NotEqual(t, observability.AgentStarted, event.Record.Event, "init should not emit agent.started")
	}

	metricPackets := findPackets[internal_type.ObservabilityMetricRecordPacket](pkts)
	require.NotEmpty(t, metricPackets, "expected llm init metric")
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, metricPackets[0].Scope)
	assert.Equal(t, "agentkit", metricPackets[0].Record.Attributes["provider"])

	logPackets := findPackets[internal_type.ObservabilityLogRecordPacket](pkts)
	require.NotEmpty(t, logPackets, "expected llm init log")
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, logPackets[0].Scope)
	assert.Equal(t, "agentkit", logPackets[0].Record.Attributes["provider"])
	assert.Equal(t, addr, logPackets[0].Record.Attributes["url"])

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	assert.NotNil(t, e.connection)
}

func TestAgentkitInitializationOptions_MergesConversationOptionsAndFiltersLocalAgentkitOptions(t *testing.T) {
	_, initializationOptions := withAgentkitOverrides(&internal_assistant_entity.AssistantProviderAgentkit{}, utils.Option{
		"existing":    "conversation",
		"customer_id": "cust-123",
		internal_options.AgentkitOptionTransportSecurity: TransportSecurityPlaintext,
		internal_options.AgentkitOptionCertificate:       "certificate",
		internal_options.AgentkitOptionMetadata:          `{"x-agentkit":"metadata"}`,
	})

	decoded, err := utils.AnyMapToInterfaceMap(initializationOptions)
	require.NoError(t, err)
	assert.Equal(t, "conversation", decoded["existing"])
	assert.Equal(t, "cust-123", decoded["customer_id"])
	assert.NotContains(t, decoded, internal_options.AgentkitOptionCertificate)
	assert.NotContains(t, decoded, internal_options.AgentkitOptionTransportSecurity)
	assert.NotContains(t, decoded, internal_options.AgentkitOptionMetadata)
}

func TestNew_UsesMaxSendMessageBytes(t *testing.T) {
	addr, shutdown := startAgentKitTestServer(t, func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
		_, err := stream.Recv()
		return err
	})
	defer shutdown()

	transportSecurity := TransportSecurityPlaintext
	maxSendMessageBytes := uint32(1)
	comm, _ := newTestAgentkitComm(&internal_assistant_entity.AssistantProviderAgentkit{
		Url:                 addr,
		TransportSecurity:   &transportSecurity,
		MaxSendMessageBytes: &maxSendMessageBytes,
	}, 4004)

	e, err := New(
		WithLogger(newTestExecutor().logger),
		WithCommunication(comm),
		WithConfiguration(&protos.ConversationInitialization{}),
	)
	require.Error(t, err)
	assert.Nil(t, e)
	assert.ErrorIs(t, err, ErrAgentkitInitializationSend)
}

// =============================================================================
// Tests: Concurrency — 2 cases (run with -race)
// =============================================================================

func TestConcurrency_ExecuteAndClose(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      "test",
			})
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond) // let some sends happen
		_ = e.Close(context.Background())
	}()

	wg.Wait()
	// If no race detected (with -race flag), test passes
}

func TestConcurrency_ReadAndClose(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())

	// Start Reader
	go func() {
		e.Read(ctx, comm, e.connection)
	}()

	// Let Reader run briefly then close
	time.Sleep(5 * time.Millisecond)
	cancel()
	err := e.Close(context.Background())
	require.NoError(t, err)
}

// =============================================================================
// Tests: Name
// =============================================================================

func TestName(t *testing.T) {
	e := newTestExecutor()
	assert.Equal(t, "agentkit", e.Name())
}

// =============================================================================
// Tests: Close — 6 cases
// =============================================================================

func TestClose_ClosesStreamWhenPresent(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)

	err := e.Close(context.Background())
	require.NoError(t, err)
	assert.True(t, talker.closeSent.Load(), "CloseSend should have been called")
}

func TestClose_SucceedsWhenConnectionIsEmpty(t *testing.T) {
	e := newTestExecutor()

	err := e.Close(context.Background())
	require.NoError(t, err)
}

func TestClose_ClearsConnectionState(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)

	_ = e.Close(context.Background())

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	assert.Nil(t, e.connection)
}

func TestClose_CancelsExecutorContext(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)

	err := e.Close(context.Background())
	require.NoError(t, err)
	assert.ErrorIs(t, context.Cause(e.ctx), context.Canceled)
}

func TestClose_ResetsActiveContextID(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	e.activeContextID = "active"

	_ = e.Close(context.Background())

	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	assert.Equal(t, "", e.activeContextID)
	assert.Nil(t, e.connection)
}

func TestClose_DisconnectsExecutorForFutureExecute(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	_ = e.Close(context.Background())

	comm, _ := newTestComm()
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "after close",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitExecutorNotConnected)
}

// =============================================================================
// Tests: concurrent user turns are serialized by connection send
// =============================================================================

func TestConcurrency_MultipleUserTurns(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	count := 50
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(index int) {
			defer wg.Done()
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", index),
				Text:      "test",
			})
		}(i)
	}
	wg.Wait()

	talker.mu.Lock()
	defer talker.mu.Unlock()
	assert.Len(t, talker.sendCalls, count)
}

// =============================================================================
// Concurrent read/write on mu — Reader reads, Execute writes
// =============================================================================

func TestConcurrency_ReadExecuteWrite(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(2)

	// Reader: reads from talker
	go func() {
		defer wg.Done()
		e.Read(ctx, comm, e.connection)
	}()

	// Sender: writes concurrently
	var sendCount atomic.Int32
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      "test",
			})
			sendCount.Add(1)
		}
		// After sending, terminate Reader
		talker.recvCh <- recvResult{err: io.EOF}
	}()

	// Wait for Reader to exit
	wg.Wait()
	cancel()
	assert.Equal(t, int32(50), sendCount.Load())
}

// =============================================================================
// End-to-End: full conversation flow
// =============================================================================

func TestE2E_FullConversationTurn(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	// 1. User sends a message
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "turn-1",
		Text:      "What is Go?",
	})
	require.NoError(t, err)

	// Verify: talker received the message, event was emitted
	talker.mu.Lock()
	require.Len(t, talker.sendCalls, 1)
	assert.Equal(t, "What is Go?", talker.sendCalls[0].GetUser().GetText())
	talker.mu.Unlock()

	evs := findPackets[internal_type.ObservabilityEventRecordPacket](collector.all())
	require.Len(t, evs, 1)
	assert.Equal(t, observability.AgentStarted, evs[0].Record.Event)

	// 2. Simulate streaming deltas from agent
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "turn-1", Message: &protos.ConversationAssistantMessage_Text{Text: "Go is"},
		}},
	})
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "turn-1", Message: &protos.ConversationAssistantMessage_Text{Text: " a language"},
		}},
	})

	// 3. Final response
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "turn-1", Completed: true,
			Message: &protos.ConversationAssistantMessage_Text{Text: "Go is a language"},
		}},
	})

	pkts := collector.all()
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	assert.Len(t, deltas, 2)
	assert.Equal(t, "Go is", deltas[0].Text)
	assert.Equal(t, " a language", deltas[1].Text)
	require.Len(t, dones, 1)
	assert.Equal(t, "Go is a language", dones[0].Text)
}

func TestE2E_MultiTurnConversation(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	for turn := 1; turn <= 5; turn++ {
		ctxID := fmt.Sprintf("turn-%d", turn)
		_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
			ContextID: ctxID, Text: fmt.Sprintf("msg-%d", turn),
		})
		e.Write(context.Background(), comm, &protos.TalkOutput{
			Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
				Id: ctxID, Completed: true,
				Message: &protos.ConversationAssistantMessage_Text{Text: fmt.Sprintf("reply-%d", turn)},
			}},
		})
	}

	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	assert.Len(t, dones, 5)
	for i, d := range dones {
		assert.Equal(t, fmt.Sprintf("reply-%d", i+1), d.Text)
	}

	talker.mu.Lock()
	assert.Len(t, talker.sendCalls, 5)
	talker.mu.Unlock()
}

func TestE2E_InterruptDuringStreaming(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	// User sends
	_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1", Text: "tell me a story",
	})

	// Delta arrives
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "ctx-1", Message: &protos.ConversationAssistantMessage_Text{Text: "Once upon"},
		}},
	})

	// Interrupt
	_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	assert.Equal(t, "", e.activeContextID)

	// New context
	_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-2", Text: "new topic",
	})

	// Stale delta from ctx-1 — rejected (activeContextID="ctx-2")
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "ctx-1", Completed: true,
			Message: &protos.ConversationAssistantMessage_Text{Text: "stale"},
		}},
	})

	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	assert.Empty(t, dones, "stale completed response should be dropped")
}

func TestE2E_ToolCallAndResult(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()
	e.activeContextID = "ctx-1"

	// Tool call
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_ToolCall{ToolCall: &protos.ConversationToolCall{
			Id: "ctx-1", ToolId: "tool-1", Name: "get_weather",
		}},
	})

	// Tool result
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_ToolCallResult{ToolCallResult: &protos.ConversationToolCallResult{
			Id: "ctx-1", ToolId: "tool-1", Name: "get_weather",
		}},
	})

	// Final response after tool
	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "ctx-1", Completed: true,
			Message: &protos.ConversationAssistantMessage_Text{Text: "It's 20C"},
		}},
	})

	pkts := collector.all()

	// tool_call is emitted as LLMToolCallPacket.
	toolCalls := findPackets[internal_type.LLMToolCallPacket](pkts)
	require.Len(t, toolCalls, 1, "expected 1 LLMToolCallPacket for tool call")
	assert.Equal(t, "get_weather", toolCalls[0].Name)

	// tool_result is now an observability log record
	logs := findPackets[internal_type.ObservabilityLogRecordPacket](pkts)
	toolResultEvents := make([]string, 0)
	for _, log := range logs {
		if log.Record.Attributes["component"] == observability.ComponentTool.String() && log.Record.Attributes["operation"] == "tool_result" {
			toolResultEvents = append(toolResultEvents, log.Record.Attributes["operation"])
		}
	}
	assert.Equal(t, []string{"tool_result"}, toolResultEvents)

	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	require.Len(t, dones, 1)
	assert.Equal(t, "It's 20C", dones[0].Text)
}

func TestE2E_ErrorEndsConversation(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	e.Write(context.Background(), comm, &protos.TalkOutput{
		Data: &protos.TalkOutput_Error{Error: &protos.Error{
			ErrorCode: 500, ErrorMessage: "agent crashed",
		}},
	})

	pkts := collector.all()
	errPkts := findPackets[internal_type.LLMErrorPacket](pkts)
	dirs := findActionToolCalls(pkts)
	require.Len(t, errPkts, 1)
	assert.Contains(t, errPkts[0].Error.Error(), "agent crashed")
	require.Len(t, dirs, 1)
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dirs[0].Action)
}

// =============================================================================
// Deadlock Detection (run with -timeout 10s and -race)
// =============================================================================

func TestDeadlock_ExecuteAndResponseConcurrent(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = e.Execute(ctx, comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			e.Write(ctx, comm, &protos.TalkOutput{
				Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
					Id: fmt.Sprintf("ctx-%d", i), Completed: true,
					Message: &protos.ConversationAssistantMessage_Text{Text: "resp"},
				}},
			})
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: Execute + Write timed out")
	}
}

func TestDeadlock_ReadAndExecuteAndClose(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(3)

	// Reader
	go func() {
		defer wg.Done()
		e.Read(ctx, comm, e.connection)
	}()

	// Execute
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = e.Execute(ctx, comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	// Close (after brief delay, unblock Reader)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		cancel() // cancel context to unblock Reader
		close(talker.recvCh)
		time.Sleep(time.Millisecond)
		_ = e.Close(context.Background())
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Fatal("DEADLOCK: Read + Execute + Close timed out")
	}
}

// =============================================================================
// Concurrency: race detector stress tests (run with -race)
// =============================================================================

func TestConcurrency_ExecuteAndInterruptRace(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	wg.Wait()
}

func TestConcurrency_ResponseAndInterruptRace(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			e.Write(context.Background(), comm, &protos.TalkOutput{
				Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
					Id:      fmt.Sprintf("ctx-%d", i),
					Message: &protos.ConversationAssistantMessage_Text{Text: "resp"},
				}},
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	wg.Wait()
}

// =============================================================================
// Consistency
// =============================================================================

func TestConsistency_StaleContextDoesNotEmitPackets(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	e.activeContextID = "ctx-active"

	// Stale assistant text emits a discard event; non-text stale responses stay silent.
	staleTypes := []*protos.TalkOutput{
		{Data: &protos.TalkOutput_Assistant{Assistant: &protos.ConversationAssistantMessage{
			Id: "ctx-stale", Completed: true,
			Message: &protos.ConversationAssistantMessage_Text{Text: "ignore"},
		}}},
		{Data: &protos.TalkOutput_Interruption{Interruption: &protos.ConversationInterruption{Id: "ctx-stale"}}},
		{Data: &protos.TalkOutput_ToolCall{ToolCall: &protos.ConversationToolCall{Id: "ctx-stale"}}},
		{Data: &protos.TalkOutput_ToolCallResult{ToolCallResult: &protos.ConversationToolCallResult{Id: "ctx-stale"}}},
		{Data: &protos.TalkOutput_ToolCall{ToolCall: &protos.ConversationToolCall{Id: "ctx-stale", Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION}}},
	}

	for _, resp := range staleTypes {
		e.Write(context.Background(), comm, resp)
	}

	events := findPackets[internal_type.ObservabilityEventRecordPacket](collector.all())
	require.Len(t, events, 1)
	assert.Equal(t, observability.AgentDiscarded, events[0].Record.Event)
	assert.Equal(t, "ignore", events[0].Record.Attributes["script"])
}
