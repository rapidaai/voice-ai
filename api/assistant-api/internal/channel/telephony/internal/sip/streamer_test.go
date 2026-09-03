package internal_sip_telephony

import (
	"context"
	"sync"
	"testing"
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSIPStreamer(t *testing.T) *Streamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	return &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, &callcontext.CallContext{}, nil, nil),
	}
}

type testSIPCollector struct {
	mu      sync.Mutex
	events  []observability.RecordEvent
	metrics []observability.RecordMetric
}

func (c *testSIPCollector) Key() string {
	return "test-sip"
}

func (c *testSIPCollector) Collect(_ context.Context, _ observability.Scope, _ observability.Context, record observability.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch typed := record.(type) {
	case observability.RecordEvent:
		c.events = append(c.events, typed)
	case observability.RecordMetric:
		c.metrics = append(c.metrics, typed)
	}
	return nil
}

func (c *testSIPCollector) Close(context.Context) error {
	return nil
}

func (c *testSIPCollector) hasEvent(event observability.EventName, reason string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, recorded := range c.events {
		if recorded.Event == event && recorded.Attributes["reason"] == reason {
			return true
		}
	}
	return false
}

func (c *testSIPCollector) metricValue(name string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, recorded := range c.metrics {
		for _, metric := range recorded.Metrics {
			if metric.GetName() == name {
				return metric.GetValue(), true
			}
		}
	}
	return "", false
}

func newTestSIPStreamerWithCollector(t *testing.T) (*Streamer, *testSIPCollector) {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	collector := &testSIPCollector{}
	observer := observability.New(
		observability.WithGlobalScope(observability.GlobalScope{OrganizationID: 1, ProjectID: 1}),
		observability.WithCollector(collector),
	)
	t.Cleanup(func() { _ = observer.Close(context.Background()) })
	return &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, &callcontext.CallContext{
			AssistantID:    1,
			ConversationID: 1,
		}, nil, observer),
	}, collector
}

func newTestInboundSIPSession(t *testing.T, callID string) *sip_runtime.Session {
	t.Helper()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10100,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionInbound),
		sip_runtime.WithSessionCallID(callID),
	)
	require.NoError(t, err)
	return session
}

type fakeSIPLifecycleController struct {
	endReasons []sip_runtime.LifecycleReason
}

func (f *fakeSIPLifecycleController) TransitionCall(session *sip_runtime.Session, next sip_runtime.CallState, _ sip_runtime.LifecycleReason) bool {
	session.SetState(next)
	return true
}

func (f *fakeSIPLifecycleController) EndCallWithReason(session *sip_runtime.Session, reason sip_runtime.LifecycleReason) error {
	f.endReasons = append(f.endReasons, reason)
	session.SetState(sip_runtime.CallStateEnded)
	session.ClearOnDisconnect()
	session.End()
	return nil
}

func (f *fakeSIPLifecycleController) FailCall(session *sip_runtime.Session, reason sip_runtime.LifecycleReason, _ error) error {
	session.SetState(sip_runtime.CallStateFailed)
	return f.EndCallWithReason(session, reason)
}

func (f *fakeSIPLifecycleController) CancelCall(session *sip_runtime.Session, reason sip_runtime.LifecycleReason) error {
	session.SetState(sip_runtime.CallStateCancelled)
	return f.EndCallWithReason(session, reason)
}

func TestSend_EndConversation_PushesToolResult(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-1",
		ToolId: "tool-1",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "ctx-1", result.GetId())
		assert.Equal(t, "tool-1", result.GetToolId())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	select {
	case <-s.Context().Done():
		t.Fatal("streamer context should remain open; teardown is owned by Talk loop")
	default:
	}
}

func TestSend_AssistantAudioQueuesUntilOutputActivated(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{
			Audio: []byte{1, 2, 3},
		},
		Completed: true,
	})
	require.NoError(t, err)

	require.Len(t, s.pendingAssistantAudioFrames, 1)
	assert.Equal(t, []byte{1, 2, 3}, s.pendingAssistantAudioFrames[0].audio)
	assert.True(t, s.pendingAssistantAudioFrames[0].completed)

	s.StartAssistantOutput()
	assert.True(t, s.assistantOutputActive.Load())
	assert.Empty(t, s.pendingAssistantAudioFrames)
}

func TestSend_InboundAssistantAudioMarksReadyBeforeOutputActivated(t *testing.T) {
	s := newTestSIPStreamer(t)
	session := newTestInboundSIPSession(t, "sip-audio-ready")
	s.session = session

	err := s.Send(&protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{
			Audio: []byte{1, 2, 3},
		},
	})
	require.NoError(t, err)

	timings := session.GetInboundSetupTimings()
	assert.False(t, timings.FirstAssistantAudioReadyAt.IsZero())
	assert.True(t, timings.FirstAssistantAudioSentAt.IsZero())
	require.Len(t, s.pendingAssistantAudioFrames, 1)
}

func TestShouldEndSessionOnClose_SkipsPreAnswerStates(t *testing.T) {
	assert.False(t, shouldEndSessionOnClose(sip_runtime.CallStateInitializing))
	assert.False(t, shouldEndSessionOnClose(sip_runtime.CallStateRinging))
	assert.True(t, shouldEndSessionOnClose(sip_runtime.CallStateConnected))
}

func TestSend_ConversationDisconnection_RecordsEventAndClosesStreamer(t *testing.T) {
	s, collector := newTestSIPStreamerWithCollector(t)
	session := newTestInboundSIPSession(t, "sip-streamer-disconnect")
	lifecycle := &fakeSIPLifecycleController{}
	s.session = session
	s.lifecycle = lifecycle

	err := s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT,
	})
	require.NoError(t, err)

	// Server-initiated Send no longer requeues the disconnect onto CriticalCh —
	// the server callsite already knows the reason. The talker exits via the
	// Recv-err path once Close cancels s.Ctx.
	select {
	case msg := <-s.CriticalCh:
		t.Fatalf("server-initiated Send must not push to CriticalCh; got %T", msg)
	default:
	}

	require.Eventually(t, func() bool {
		return collector.hasEvent(observability.CallHangup, protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT.String())
	}, time.Second, 10*time.Millisecond)

	// Streamer should be closed: s.Ctx cancelled so Talker.Recv returns EOF.
	select {
	case <-s.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("expected streamer context to be cancelled after disconnect")
	}

	require.Equal(t, []sip_runtime.LifecycleReason{sip_runtime.LifecycleReasonStreamerEndSession}, lifecycle.endReasons)
}

func TestRecordTransferDurationMetric_RecordsCallTransferMetric(t *testing.T) {
	s, collector := newTestSIPStreamerWithCollector(t)

	s.RecordTransferDurationMetric("1234")

	var value string
	require.Eventually(t, func() bool {
		var ok bool
		value, ok = collector.metricValue(observability.MetricCallTransferDurationMs)
		return ok
	}, time.Second, 10*time.Millisecond, "expected SIP transfer teardown to record transfer duration metric")
	assert.Equal(t, "1234", value)
}

func TestSend_ConversationDisconnection_PreservesExplicitReason(t *testing.T) {
	s, collector := newTestSIPStreamerWithCollector(t)

	err := s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return collector.hasEvent(observability.CallHangup, protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION.String())
	}, time.Second, 10*time.Millisecond)
}

func TestSend_ConversationDisconnection_SubsequentSendErrors(t *testing.T) {
	s := newTestSIPStreamer(t)

	// First disconnect closes the streamer.
	require.NoError(t, s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
	}))

	select {
	case <-s.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("expected streamer context to be cancelled")
	}

	// Subsequent Send must error (streamer is closed) without panicking or
	// duplicating cleanup.
	err := s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
	})
	require.Error(t, err, "Send on closed streamer should return an error")
}

func TestSend_TransferConversation_UsesTransferToKey(t *testing.T) {
	s := newTestSIPStreamer(t)

	var gotTargets []string
	var gotPostTransferAction string
	s.SetTransferRequestHandler(func(targets []string, postTransferAction string) {
		gotTargets = append([]string(nil), targets...)
		gotPostTransferAction = postTransferAction
	})

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-transfer",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args: map[string]string{
			"transfer_to":          "+15550001111" + commons.SEPARATOR + "sip:agent@example.com",
			"post_transfer_action": "end_call",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"+15550001111", "sip:agent@example.com"}, gotTargets)
	assert.Equal(t, "end_call", gotPostTransferAction)
}

func TestSend_TransferConversation_MissingTransferTarget(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-transfer-missing",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""},
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "ctx-transfer-missing", result.GetId())
		assert.Equal(t, "tool-transfer", result.GetToolId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing transfer target")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}
