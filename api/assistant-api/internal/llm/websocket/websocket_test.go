package internal_llm_websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

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

func findPackets[T internal_type.Packet](pkts []internal_type.Packet) []T {
	var out []T
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

func newTestExecutor(t *testing.T) *websocketExecutor {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	return &websocketExecutor{logger: logger}
}

func TestHandleResponse_Stream_StaleContextDropped(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-active"

	collected := make([]internal_type.Packet, 0)
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: []byte(`{"id":"ctx-stale","content":"hello","index":0}`),
	}, func(_ context.Context, packet ...internal_type.Packet) error {
		collected = append(collected, packet...)
		return nil
	})

	assert.Empty(t, collected)
}

func TestHandleResponse_Stream_CurrentContextEmits(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"

	collected := make([]internal_type.Packet, 0)
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: []byte(`{"id":"ctx-1","content":"hello","index":0}`),
	}, func(_ context.Context, packet ...internal_type.Packet) error {
		collected = append(collected, packet...)
		return nil
	})

	require.Len(t, collected, 1)
	delta, ok := collected[0].(internal_type.LLMResponseDeltaPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-1", delta.ContextID)
	assert.Equal(t, "hello", delta.Text)
}

func TestExecute_UserInputPacket_EmptyContextNoop(t *testing.T) {
	e := newTestExecutor(t)
	err := e.Execute(context.Background(), nil, internal_type.UserInputPacket{Text: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "", e.currentID)
}

func TestExecute_InterruptionDetectedPacket_ClearsContext(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"
	err := e.Execute(context.Background(), nil, internal_type.InterruptionDetectedPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	assert.Equal(t, "", e.currentID)
}

func TestExecute_UnsupportedPacket(t *testing.T) {
	e := newTestExecutor(t)
	err := e.Execute(context.Background(), nil, internal_type.EndOfSpeechPacket{ContextID: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported packet")
}

func TestExecute_InjectMessagePacket_Noop(t *testing.T) {
	e := newTestExecutor(t)
	err := e.Execute(context.Background(), nil, internal_type.InjectMessagePacket{ContextID: "x", Text: "hi"})
	require.NoError(t, err)
}

func TestName(t *testing.T) {
	e := newTestExecutor(t)
	assert.Equal(t, "websocket", e.Name())
}

// =============================================================================
// Tests: handleResponse — all message types
// =============================================================================

func TestHandleResponse_Complete_EmitsLLMResponseDonePacket(t *testing.T) {
	e := newTestExecutor(t)
	e.setCurrentContextID("ctx-1")
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeComplete,
		Data: json.RawMessage(`{"id":"ctx-1","content":"done text","metrics":[{"name":"tokens","value":10}]}`),
	}, onPacket)

	require.Len(t, collected, 4)
	done, ok := collected[0].(internal_type.LLMResponseDonePacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-1", done.ContextID)
	assert.Equal(t, "done text", done.Text)
	ev, ok := collected[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	assert.Equal(t, observability.AgentCompleted, ev.Record.Event)
	metric, ok := collected[2].(internal_type.ObservabilityMetricRecordPacket)
	require.True(t, ok)
	require.Len(t, metric.Record.Metrics, 3)
	assert.Equal(t, observability.MetricAgentResponseCharCount, metric.Record.Metrics[0].Name)
	assert.Equal(t, observability.MetricAgentTTFTMs, metric.Record.Metrics[1].Name)
	assert.Equal(t, observability.MetricAgentTRTMs, metric.Record.Metrics[2].Name)
	usage, ok := collected[3].(internal_type.ObservabilityUsageRecordPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-1", usage.ContextID)
}

func TestHandleResponse_Complete_StaleContextDropped(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-active"
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeComplete,
		Data: json.RawMessage(`{"id":"ctx-stale","content":"ignore"}`),
	}, onPacket)

	assert.Empty(t, collected)
}

func TestHandleResponse_Complete_EmptyContentNoPacket(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeComplete,
		Data: json.RawMessage(`{"id":"ctx-1","content":""}`),
	}, onPacket)

	assert.Empty(t, collected, "empty content should not emit done packet")
}

func TestHandleResponse_Interruption(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeInterruption,
		Data: json.RawMessage(`{"id":"ctx-1","source":"vad"}`),
	}, onPacket)

	require.Len(t, collected, 2)
	ip, ok := collected[0].(internal_type.InterruptionDetectedPacket)
	require.True(t, ok)
	assert.Equal(t, internal_type.InterruptionSourceVad, ip.Source)
	ev, ok := collected[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	assert.Equal(t, observability.AgentDiscarded, ev.Record.Event)
}

func TestHandleResponse_Interruption_WordSource(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeInterruption,
		Data: json.RawMessage(`{"id":"ctx-1","source":"word"}`),
	}, onPacket)

	require.Len(t, collected, 2)
	ip, ok := collected[0].(internal_type.InterruptionDetectedPacket)
	require.True(t, ok)
	assert.Equal(t, internal_type.InterruptionSourceWord, ip.Source)
	ev, ok := collected[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	assert.Equal(t, observability.AgentDiscarded, ev.Record.Event)
}

func TestHandleResponse_Close(t *testing.T) {
	e := newTestExecutor(t)
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeClose,
		Data: json.RawMessage(`{"reason":"session ended","code":1000}`),
	}, onPacket)

	require.Len(t, collected, 1)
	dir, ok := collected[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dir.Action)
	assert.Equal(t, "session ended", dir.Arguments["reason"])
}

func TestHandleResponse_Error(t *testing.T) {
	e := newTestExecutor(t)
	e.currentID = "ctx-1"
	collected := make([]internal_type.Packet, 0)
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		collected = append(collected, pkts...)
		return nil
	}

	e.handleResponse(context.Background(), &Response{
		Type: TypeError,
		Data: json.RawMessage(`{"code":500,"message":"server error"}`),
	}, onPacket)

	require.Len(t, collected, 3)
	errPkt, ok := collected[0].(internal_type.LLMErrorPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-1", errPkt.ContextID)

	ev, ok := collected[1].(internal_type.ObservabilityEventRecordPacket)
	require.True(t, ok)
	assert.Equal(t, observability.AgentError, ev.Record.Event)
	assert.Equal(t, "websocket", ev.Record.Attributes["provider"])

	log, ok := collected[2].(internal_type.ObservabilityLogRecordPacket)
	require.True(t, ok)
	assert.Equal(t, observability.LevelError, log.Record.Level)
	assert.Equal(t, "response", log.Record.Attributes["operation"])
	assert.Equal(t, "websocket", log.Record.Attributes["provider"])
}

// =============================================================================
// Tests: context ID management
// =============================================================================

func TestSetCurrentContextID(t *testing.T) {
	e := newTestExecutor(t)
	e.setCurrentContextID("ctx-1")
	assert.Equal(t, "ctx-1", e.currentID)

	e.setCurrentContextID("ctx-2")
	assert.Equal(t, "ctx-2", e.currentID)
}

func TestIsCurrentContextID_EmptyIDsAccepted(t *testing.T) {
	e := newTestExecutor(t)
	// Both empty — accepted
	assert.True(t, e.isCurrentContextID(""))
	assert.True(t, e.isCurrentContextID("  "))

	// Current empty, incoming non-empty — accepted
	assert.True(t, e.isCurrentContextID("ctx-1"))

	// Current set, incoming empty — accepted
	e.currentID = "ctx-1"
	assert.True(t, e.isCurrentContextID(""))

	// Current set, incoming matches — accepted
	assert.True(t, e.isCurrentContextID("ctx-1"))

	// Current set, incoming differs — rejected
	assert.False(t, e.isCurrentContextID("ctx-stale"))
}

// =============================================================================
// End-to-End: full conversation flow through handleResponse
// =============================================================================

func TestE2E_FullConversationFlow(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	// 1. Set context (simulating Execute)
	e.setCurrentContextID("turn-1")

	// 2. Stream deltas
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"turn-1","content":"Hello","index":0}`),
	}, onPacket)
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"turn-1","content":" world","index":1}`),
	}, onPacket)

	// 3. Complete
	e.handleResponse(context.Background(), &Response{
		Type: TypeComplete,
		Data: json.RawMessage(`{"id":"turn-1","content":"Hello world"}`),
	}, onPacket)

	pkts := collector.all()
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	assert.Len(t, deltas, 2)
	assert.Equal(t, "Hello", deltas[0].Text)
	assert.Equal(t, " world", deltas[1].Text)
	require.Len(t, dones, 1)
	assert.Equal(t, "Hello world", dones[0].Text)
}

func TestE2E_InterruptDuringStreaming(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	e.setCurrentContextID("ctx-1")

	// Delta arrives
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":"hello","index":0}`),
	}, onPacket)

	// Interrupt
	_ = e.Execute(context.Background(), nil, internal_type.InterruptionDetectedPacket{ContextID: "ctx-1"})
	assert.Equal(t, "", e.currentID)

	// Late delta from old context — isCurrentContextID("ctx-1") with current="" → true (empty current accepts all)
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":" late","index":1}`),
	}, onPacket)

	// New context
	e.setCurrentContextID("ctx-2")

	// Stale delta from ctx-1 — now rejected
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":" stale","index":2}`),
	}, onPacket)

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	assert.Len(t, deltas, 2, "pre-interrupt + post-interrupt(empty current), not the stale one")
}

func TestE2E_MultiTurn(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	for turn := 1; turn <= 3; turn++ {
		ctxID := fmt.Sprintf("turn-%d", turn)
		e.setCurrentContextID(ctxID)

		e.handleResponse(context.Background(), &Response{
			Type: TypeStream,
			Data: json.RawMessage(fmt.Sprintf(`{"id":"%s","content":"chunk-%d","index":0}`, ctxID, turn)),
		}, onPacket)

		e.handleResponse(context.Background(), &Response{
			Type: TypeComplete,
			Data: json.RawMessage(fmt.Sprintf(`{"id":"%s","content":"reply-%d"}`, ctxID, turn)),
		}, onPacket)
	}

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	assert.Len(t, deltas, 3)
	assert.Len(t, dones, 3)
}

func TestE2E_ServerClose(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	e.setCurrentContextID("ctx-1")

	// Some streaming, then server sends close
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":"partial","index":0}`),
	}, onPacket)

	e.handleResponse(context.Background(), &Response{
		Type: TypeClose,
		Data: json.RawMessage(`{"reason":"timeout","code":1001}`),
	}, onPacket)

	dirs := findPackets[internal_type.LLMToolCallPacket](collector.all())
	require.Len(t, dirs, 1)
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dirs[0].Action)
}

// =============================================================================
// Deadlock Detection (run with -timeout 10s and -race)
// =============================================================================

func TestDeadlock_SetContextAndIsContextConcurrent(t *testing.T) {
	e := newTestExecutor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			e.setCurrentContextID(fmt.Sprintf("ctx-%d", i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			e.isCurrentContextID(fmt.Sprintf("ctx-%d", i))
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: setCurrentContextID + isCurrentContextID timed out")
	}
}

func TestDeadlock_HandleResponseAndExecuteConcurrent(t *testing.T) {
	e := newTestExecutor(t)
	onPacket := func(_ context.Context, _ ...internal_type.Packet) error { return nil }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			e.handleResponse(ctx, &Response{
				Type: TypeStream,
				Data: json.RawMessage(fmt.Sprintf(`{"id":"ctx-%d","content":"c","index":0}`, i)),
			}, onPacket)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(ctx, nil, internal_type.InterruptionDetectedPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: handleResponse + Execute timed out")
	}
}

// =============================================================================
// Concurrency (run with -race)
// =============================================================================

func TestConcurrency_ExecuteAndInterruptRace(t *testing.T) {
	e := newTestExecutor(t)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = e.Execute(context.Background(), nil, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = e.Execute(context.Background(), nil, internal_type.InterruptionDetectedPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	wg.Wait()
	// No assertion — point is no panic/race under -race flag.
}

func TestConcurrency_HandleResponseMultipleGoroutines(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	var wg sync.WaitGroup
	const N = 10
	wg.Add(N)

	for g := 0; g < N; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				e.handleResponse(context.Background(), &Response{
					Type: TypeStream,
					Data: json.RawMessage(fmt.Sprintf(`{"id":"ctx-%d","content":"g%d-c%d","index":%d}`, g, g, i, i)),
				}, onPacket)
			}
		}()
	}

	wg.Wait()
	// All 500 deltas should have been collected
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	assert.Len(t, deltas, N*50)
}

// =============================================================================
// Consistency
// =============================================================================

func TestConsistency_StaleAfterContextSwitch(t *testing.T) {
	e := newTestExecutor(t)
	collector := &packetCollector{}
	onPacket := func(ctx context.Context, pkts ...internal_type.Packet) error {
		return collector.collect(ctx, pkts...)
	}

	e.setCurrentContextID("ctx-1")

	// Delta for ctx-1 — accepted
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":"ok","index":0}`),
	}, onPacket)

	// Switch context
	e.setCurrentContextID("ctx-2")

	// Delta for ctx-1 — rejected
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-1","content":"stale","index":1}`),
	}, onPacket)

	// Delta for ctx-2 — accepted
	e.handleResponse(context.Background(), &Response{
		Type: TypeStream,
		Data: json.RawMessage(`{"id":"ctx-2","content":"new","index":0}`),
	}, onPacket)

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	require.Len(t, deltas, 2)
	assert.Equal(t, "ok", deltas[0].Text)
	assert.Equal(t, "new", deltas[1].Text)
}
