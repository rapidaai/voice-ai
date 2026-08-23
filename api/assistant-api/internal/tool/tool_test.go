// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/tool/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

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

// mockCommunication provides OnPacket for tests.
type mockCommunication struct {
	internal_type.Communication // nil-embed for unimplemented methods
	collector                   *packetCollector
}

func (m *mockCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	return m.collector.collect(ctx, pkts...)
}

// stubToolCaller is a minimal ToolCaller that records whether Call was invoked
// and pushes a result via communication.OnPacket.
type stubToolCaller struct {
	name   string
	called atomic.Bool
}

func (s *stubToolCaller) Id() uint64                                      { return 1 }
func (s *stubToolCaller) Name() string                                    { return s.name }
func (s *stubToolCaller) Definition() (*protos.FunctionDefinition, error) { return nil, nil }
func (s *stubToolCaller) ExecutionMethod() string                         { return "stub" }
func (s *stubToolCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	s.called.Store(true)
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID: toolId, Name: s.name, ContextID: contextID, Arguments: internal_tool.StringifyArgs(args),
	})
	communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
		ToolID:    toolId,
		Name:      s.name,
		ContextID: contextID,
		Result:    internal_tool.Result("ok", true),
	})
}

func newToolCall(id, name, arguments string) *protos.ToolCall {
	return &protos.ToolCall{
		Id: id,
		Function: &protos.FunctionCall{
			Name:      name,
			Arguments: arguments,
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestExecuteAll_UnknownTool_NoPackets(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	executor := &toolExecutor{
		logger: logger,
		tools:  make(map[string]internal_tool.ToolCaller),
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	call := newToolCall("call-1", "nonexistent_tool", `{}`)
	executor.ExecuteAll(context.Background(), "ctx-1", []*protos.ToolCall{call}, comm)

	pkts := collector.all()
	assert.Len(t, pkts, 0, "unknown tool should silently return")
}

func TestExecute_KnownTool_EmitsToolCallEventAndDelegatesToCaller(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	stub := &stubToolCaller{name: "my_tool"}
	executor := &toolExecutor{
		logger: logger,
		tools:  map[string]internal_tool.ToolCaller{"my_tool": stub},
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	call := newToolCall("call-2", "my_tool", `{"key":"value"}`)
	executor.ExecuteAll(context.Background(), "ctx-2", []*protos.ToolCall{call}, comm)
	time.Sleep(50 * time.Millisecond)

	assert.True(t, stub.called.Load(), "expected stub Call() to be invoked")

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + LLMToolResultPacket")

	toolCall, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "first packet should be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "call-2", toolCall.ToolID)
	assert.Equal(t, "my_tool", toolCall.Name)
	assert.Equal(t, "ctx-2", toolCall.ContextID)
	assert.Equal(t, "value", toolCall.Arguments["key"])

	result, ok := pkts[1].(internal_type.LLMToolResultPacket)
	require.True(t, ok, "second packet should be LLMToolResultPacket, got %T", pkts[1])
	assert.Equal(t, "call-2", result.ToolID)
	assert.Equal(t, "my_tool", result.Name)
	assert.Equal(t, "ctx-2", result.ContextID)
	assert.Equal(t, "SUCCESS", result.Result["status"])
}

func TestExecute_MalformedArguments_FallsBackToRaw(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	stub := &stubToolCaller{name: "raw_tool"}
	executor := &toolExecutor{
		logger: logger,
		tools:  map[string]internal_tool.ToolCaller{"raw_tool": stub},
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	call := newToolCall("call-3", "raw_tool", `not-valid-json`)
	executor.ExecuteAll(context.Background(), "ctx-3", []*protos.ToolCall{call}, comm)
	time.Sleep(50 * time.Millisecond)

	assert.True(t, stub.called.Load())

	pkts := collector.all()
	require.GreaterOrEqual(t, len(pkts), 1)

	toolCall, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	assert.Equal(t, "not-valid-json", toolCall.Arguments["raw"])
}

func TestParseArgument_ValidJSON(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	executor := &toolExecutor{logger: logger}

	result := executor.parseArgument(`{"name":"test","count":42}`)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(42), result["count"])
}

func TestParseArgument_InvalidJSON(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	executor := &toolExecutor{logger: logger}

	result := executor.parseArgument("invalid json")
	assert.Equal(t, "invalid json", result["raw"])
}

// ---------------------------------------------------------------------------
// Packet-order tests at executor level
// ---------------------------------------------------------------------------

func TestExecuteAll_KnownTool_PacketOrderPreserved(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	stub := &stubToolCaller{name: "order_tool"}
	executor := &toolExecutor{
		logger: logger,
		tools:  map[string]internal_tool.ToolCaller{"order_tool": stub},
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	call := newToolCall("call-ord", "order_tool", `{"a":"b"}`)
	executor.ExecuteAll(context.Background(), "ctx-ord", []*protos.ToolCall{call}, comm)
	time.Sleep(50 * time.Millisecond)

	pkts := collector.all()
	require.Len(t, pkts, 2, "executor must not add extra packets beyond what the caller emits")

	// Verify strict ordering: [0] LLMToolCallPacket, [1] LLMToolResultPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "call-ord", tc.ToolID)
	assert.Equal(t, "order_tool", tc.Name)
	assert.Equal(t, "ctx-ord", tc.ContextID)

	tr, ok := pkts[1].(internal_type.LLMToolResultPacket)
	require.True(t, ok, "packet[1] must be LLMToolResultPacket, got %T", pkts[1])
	assert.Equal(t, "call-ord", tr.ToolID)
	assert.Equal(t, "order_tool", tr.Name)
	assert.Equal(t, "ctx-ord", tr.ContextID)
}

// actionStubToolCaller simulates a tool that emits a single LLMToolCallPacket with Action
// (like end_of_conversation or transfer_call).
type actionStubToolCaller struct {
	name string
}

func (d *actionStubToolCaller) Id() uint64                                      { return 2 }
func (d *actionStubToolCaller) Name() string                                    { return d.name }
func (d *actionStubToolCaller) Definition() (*protos.FunctionDefinition, error) { return nil, nil }
func (d *actionStubToolCaller) ExecutionMethod() string                         { return "stub" }
func (d *actionStubToolCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID:    toolId,
		Name:      d.name,
		ContextID: contextID,
		Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
		Arguments: internal_tool.StringifyArgs(args),
	})
}

func TestExecuteAll_ActionTool_PacketOrderPreserved(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	stub := &actionStubToolCaller{name: "end_tool"}
	executor := &toolExecutor{
		logger: logger,
		tools:  map[string]internal_tool.ToolCaller{"end_tool": stub},
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	call := newToolCall("call-dir", "end_tool", `{}`)
	executor.ExecuteAll(context.Background(), "ctx-dir", []*protos.ToolCall{call}, comm)
	time.Sleep(50 * time.Millisecond)

	pkts := collector.all()
	require.Len(t, pkts, 1, "action tool emits single LLMToolCallPacket with Action")

	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	assert.True(t, ok, "packet[0] must be LLMToolCallPacket")
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, tc.Action)
}

func TestExecuteAll_MultipleTools_IndependentOrder(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	stubA := &stubToolCaller{name: "tool_a"}
	stubB := &stubToolCaller{name: "tool_b"}
	executor := &toolExecutor{
		logger: logger,
		tools: map[string]internal_tool.ToolCaller{
			"tool_a": stubA,
			"tool_b": stubB,
		},
	}

	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	calls := []*protos.ToolCall{
		newToolCall("call-a", "tool_a", `{"k":"v1"}`),
		newToolCall("call-b", "tool_b", `{"k":"v2"}`),
	}
	executor.ExecuteAll(context.Background(), "ctx-multi", calls, comm)
	time.Sleep(100 * time.Millisecond)

	pkts := collector.all()
	// Each stub emits 2 packets (call+result), so total 4
	require.Len(t, pkts, 4, "expected 2 packets per tool for 2 tools")

	// For each tool, its LLMToolCallPacket must appear before its LLMToolResultPacket.
	// Tools may be interleaved since they run concurrently, but per-tool order is preserved.
	toolCallIdx := map[string]int{}
	toolResultIdx := map[string]int{}
	for i, p := range pkts {
		switch pkt := p.(type) {
		case internal_type.LLMToolCallPacket:
			toolCallIdx[pkt.ToolID] = i
		case internal_type.LLMToolResultPacket:
			toolResultIdx[pkt.ToolID] = i
		}
	}

	assert.Contains(t, toolCallIdx, "call-a")
	assert.Contains(t, toolResultIdx, "call-a")
	assert.Less(t, toolCallIdx["call-a"], toolResultIdx["call-a"],
		"tool_a: LLMToolCallPacket must appear before LLMToolResultPacket")

	assert.Contains(t, toolCallIdx, "call-b")
	assert.Contains(t, toolResultIdx, "call-b")
	assert.Less(t, toolCallIdx["call-b"], toolResultIdx["call-b"],
		"tool_b: LLMToolCallPacket must appear before LLMToolResultPacket")
}
