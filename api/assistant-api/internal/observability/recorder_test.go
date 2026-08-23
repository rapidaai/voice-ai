// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type recordingCollector struct {
	mu          sync.Mutex
	key         string
	contexts    []Context
	scopes      []Scope
	logs        []RecordLog
	events      []RecordEvent
	metrics     []RecordMetric
	metadata    []RecordMetadata
	usage       []RecordUsage
	webhooks    []RecordWebhook
	toolLogs    []RecordToolLog
	requestLogs []RecordRequestLog
	collectErr  error
	closeErr    error
	closed      bool
}

func (c *recordingCollector) Key() string {
	return c.key
}

func (c *recordingCollector) Collect(_ context.Context, scope Scope, observationContext Context, record Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contexts = append(c.contexts, observationContext)
	c.scopes = append(c.scopes, scope)
	switch typed := record.(type) {
	case RecordLog:
		c.logs = append(c.logs, typed)
	case RecordEvent:
		c.events = append(c.events, typed)
	case RecordMetric:
		c.metrics = append(c.metrics, typed)
	case RecordMetadata:
		c.metadata = append(c.metadata, typed)
	case RecordUsage:
		c.usage = append(c.usage, typed)
	case RecordWebhook:
		c.webhooks = append(c.webhooks, typed)
	case RecordToolLog:
		c.toolLogs = append(c.toolLogs, typed)
	case RecordRequestLog:
		c.requestLogs = append(c.requestLogs, typed)
	}
	return c.collectErr
}

func (c *recordingCollector) eventCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

func (c *recordingCollector) Close(context.Context) error {
	c.closed = true
	return c.closeErr
}

type blockingCollector struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingCollector) Key() string {
	return ""
}

func (c *blockingCollector) Collect(context.Context, Scope, Context, Record) error {
	c.once.Do(func() { close(c.started) })
	<-c.release
	return nil
}

func (c *blockingCollector) Close(context.Context) error {
	select {
	case <-c.release:
	default:
		close(c.release)
	}
	return nil
}

type blockingKeyCollector struct {
	key     string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingKeyCollector) Key() string {
	c.once.Do(func() { close(c.started) })
	<-c.release
	return c.key
}

func (c *blockingKeyCollector) Collect(context.Context, Scope, Context, Record) error {
	return nil
}

func (c *blockingKeyCollector) Close(context.Context) error {
	return nil
}

func waitForRecorderDone(t *testing.T, observabilityRecorder Recorder) error {
	t.Helper()
	concreteRecorder, ok := observabilityRecorder.(*recorder)
	if !ok {
		t.Fatalf("unexpected recorder type: %T", observabilityRecorder)
	}
	select {
	case <-concreteRecorder.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for recorder close")
	}
	return concreteRecorder.errors()
}

func TestRecorder_RecordMetric_FansOutAndInjectsGlobalScope(t *testing.T) {
	now := time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC)
	first := &recordingCollector{key: "first"}
	second := &recordingCollector{key: "second"}
	recorder := New(
		WithGlobalScope(GlobalScope{OrganizationID: 7, ProjectID: 8}),
		WithContext(context.WithValue(context.Background(), types.REQUEST_ID_KEY, "trace-1")),
		WithClock(func() time.Time { return now }),
		WithCollectors(first, second),
	)
	scope := ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}

	err := recorder.Record(context.Background(), scope, RecordMetric{
		ID:      "metric-1",
		Metrics: []*protos.Metric{{Name: MetricConversationDuration, Value: "1000"}},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(first.metrics) != 1 || len(second.metrics) != 1 {
		t.Fatalf("expected fanout to both collectors, got first=%d second=%d", len(first.metrics), len(second.metrics))
	}
	got := first.metrics[0]
	if got.ID != "metric-1" {
		t.Fatalf("unexpected id: %s", got.ID)
	}
	if first.contexts[0].TraceID != "trace-1" {
		t.Fatalf("unexpected trace id: %s", first.contexts[0].TraceID)
	}
	if !got.OccurredAt.Equal(now) {
		t.Fatalf("unexpected occurred_at: %s", got.OccurredAt)
	}
	if observabilityGlobal := first.scopes[0].GlobalScopeValue(); observabilityGlobal.OrganizationID != 7 || observabilityGlobal.ProjectID != 8 {
		t.Fatalf("unexpected global scope: %+v", observabilityGlobal)
	}
}

func TestRecorder_RecordInjectsAuthIntoObservationContext(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	userID := uint64(9)
	auth := &types.ServiceScope{
		UserId:         &userID,
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
	}
	collector := &recordingCollector{key: "collector"}
	recorder := New(
		WithAuth(auth),
		WithCollector(collector),
	)

	err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordMetric{
		Metrics: []*protos.Metric{{Name: MetricConversationStatus, Value: "FAILED"}},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.contexts) != 1 || collector.contexts[0].Auth != auth {
		t.Fatalf("expected recorder auth in observation context, got %+v", collector.contexts)
	}
	if observabilityGlobal := collector.scopes[0].GlobalScopeValue(); observabilityGlobal.OrganizationID != organizationID || observabilityGlobal.ProjectID != projectID {
		t.Fatalf("unexpected global scope: %+v", observabilityGlobal)
	}
}

func TestRecorder_RecordUsesRequestIDFromRecordContext(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(
		WithGlobalScope(GlobalScope{OrganizationID: 7, ProjectID: 8}),
		WithCollector(collector),
	)

	err := recorder.Record(
		context.WithValue(context.Background(), types.REQUEST_ID_KEY, "record-trace"),
		ProjectScope{},
		RecordLog{Message: "hello"},
	)
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if collector.contexts[0].TraceID != "record-trace" {
		t.Fatalf("unexpected trace id: %s", collector.contexts[0].TraceID)
	}
}

func TestRecorder_RecordUsesCanceledContextValues(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(
		WithGlobalScope(GlobalScope{OrganizationID: 7, ProjectID: 8}),
		WithCollector(collector),
	)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), types.REQUEST_ID_KEY, "canceled-trace"))
	cancel()

	err := recorder.Record(ctx, ProjectScope{}, RecordLog{Message: "client disconnected"})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.logs) != 1 {
		t.Fatalf("expected canceled-context record to be collected, got %d", len(collector.logs))
	}
	if collector.contexts[0].TraceID != "canceled-trace" {
		t.Fatalf("unexpected trace id: %s", collector.contexts[0].TraceID)
	}
}

func TestRecorder_RecordFallsBackToGeneratedTraceID(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(
		WithGlobalScope(GlobalScope{OrganizationID: 7, ProjectID: 8}),
		WithCollector(collector),
	)

	err := recorder.Record(context.Background(), ProjectScope{}, RecordLog{Message: "hello"})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if collector.contexts[0].TraceID == "" {
		t.Fatal("expected generated trace id")
	}
}

func TestRecorder_RecordLog_ProjectScope(t *testing.T) {
	now := time.Date(2026, 6, 7, 9, 0, 0, 0, time.UTC)
	collector := &recordingCollector{key: "collector"}
	recorder := New(
		WithGlobalScope(GlobalScope{OrganizationID: 7, ProjectID: 8}),
		WithClock(func() time.Time { return now }),
		WithCollector(collector),
	)

	err := recorder.Record(context.Background(), ProjectScope{}, RecordLog{
		ID:      "log-1",
		Level:   LevelError,
		Message: "webrtc constructor failed",
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.logs) != 1 {
		t.Fatalf("expected project log, got %d", len(collector.logs))
	}
	if collector.scopes[0].ScopeType() != ScopeProject {
		t.Fatalf("unexpected scope type: %s", collector.scopes[0].ScopeType())
	}
	global := collector.scopes[0].GlobalScopeValue()
	if global.OrganizationID != 7 || global.ProjectID != 8 {
		t.Fatalf("unexpected global scope: %+v", global)
	}
	projectScope, ok := collector.scopes[0].(ProjectScope)
	if !ok {
		t.Fatalf("unexpected scope type: %T", collector.scopes[0])
	}
	if projectScope.ContextID() != "8" {
		t.Fatalf("unexpected context id: %s", projectScope.ContextID())
	}
	if !collector.logs[0].OccurredAt.Equal(now) {
		t.Fatalf("unexpected occurred_at: %s", collector.logs[0].OccurredAt)
	}
}

func TestRecorder_RecordEvent_UsesExplicitMessageScope(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(WithCollector(collector))
	scope := MessageScope{
		ConversationScope: ConversationScope{
			AssistantScope: AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "msg-1",
		Role:      MessageRoleUser,
	}

	err := recorder.Record(context.Background(), scope, NewMessageEventRecord(
		"msg-1",
		MessageRoleUser,
		EOSCompleted,
		Attributes{"speech": "hello"},
	))
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.events) != 1 {
		t.Fatalf("expected one event, got %d", len(collector.events))
	}
	gotScope := collector.scopes[0]
	messageScope, ok := gotScope.(MessageScope)
	if !ok {
		t.Fatalf("unexpected scope type: %T", gotScope)
	}
	if messageScope.AssistantScopeID() != 10 || messageScope.ConversationScopeID() != 20 || messageScope.ContextID() != "msg-1" || messageScope.MessageScopeRole() != MessageRoleUser {
		t.Fatalf("unexpected message scope: assistant=%d conversation=%d context=%q role=%q",
			messageScope.AssistantScopeID(), messageScope.ConversationScopeID(), messageScope.ContextID(), messageScope.MessageScopeRole())
	}
}

func TestRecorder_RecordRequiresScope(t *testing.T) {
	recorder := New(WithCollector(&recordingCollector{key: "collector"}))
	defer recorder.Close(context.Background())

	err := recorder.Record(context.Background(), nil, NewMessageEventRecord(
		"msg-1",
		MessageRoleUser,
		EOSCompleted,
		nil,
	))
	if err == nil {
		t.Fatal("expected missing scope error")
	}
}

func TestRecorder_RecordWebhook_FansOut(t *testing.T) {
	first := &recordingCollector{key: "first"}
	second := &recordingCollector{key: "second"}
	recorder := New(WithCollectors(first, second))

	err := recorder.Record(context.Background(), AssistantScope{AssistantID: 10}, RecordWebhook{
		ID:    "wh-1",
		Event: CallReceived,
		Payload: CallReceivedWebhookPayload{
			V1WebhookPayloadBase: NewV1WebhookPayload(nil),
			Provider:             "test",
			CallID:               "call-1",
			Direction:            "inbound",
		},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
	if len(first.webhooks) != 1 || len(second.webhooks) != 1 {
		t.Fatalf("expected both collectors to receive webhook record, got first=%d second=%d", len(first.webhooks), len(second.webhooks))
	}
}

func TestRecorder_RecordUsage_AllowsMessageScope(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(WithCollector(collector))
	defer recorder.Close(context.Background())

	err := recorder.Record(context.Background(), MessageScope{
		ConversationScope: ConversationScope{
			AssistantScope: AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "user-ctx-1",
		Role:      MessageRoleUser,
	}, RecordUsage{
		Component: ComponentUsage,
		Duration:  time.Second,
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
	if len(collector.usage) != 1 {
		t.Fatalf("expected usage record, got %d", len(collector.usage))
	}
}

func TestRecorder_RecordVariadicRecords_AllCollected(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New(WithCollector(collector))
	scope := ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}

	err := recorder.Record(context.Background(), scope,
		RecordEvent{
			Component: ComponentConversation,
			Event:     ConversationInitializing,
		},
		RecordMetric{
			Metrics: []*protos.Metric{{Name: MetricCallStatus, Value: "started"}},
		},
		RecordMetadata{
			Metadata: []*protos.Metadata{{Key: MetadataDisconnectReason, Value: "started"}},
		},
	)
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.events) != 1 {
		t.Fatalf("expected event to be collected, got %d", len(collector.events))
	}
	if len(collector.metrics) != 1 {
		t.Fatalf("expected metric to be collected, got %d", len(collector.metrics))
	}
	if len(collector.metadata) != 1 {
		t.Fatalf("expected metadata to be collected, got %d", len(collector.metadata))
	}
}

func TestRecorder_RecordSnapshotsMutablePayloads(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New()
	scope := ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}
	metric := &protos.Metric{Name: MetricConversationDuration, Value: "1000", Description: "before"}
	metadata := &protos.Metadata{Key: MetadataLanguage, Value: "en"}
	webhookNestedPayload := map[string]interface{}{"value": "before"}
	webhookListPayload := []interface{}{map[string]interface{}{"value": "before"}}
	webhookBytesPayload := []byte("before")
	webhookPayload := CallRingingWebhookPayload{
		V1WebhookPayloadBase: NewV1WebhookPayload(map[string]interface{}{
			"status": "before",
			"nested": webhookNestedPayload,
			"list":   webhookListPayload,
			"bytes":  webhookBytesPayload,
		}),
		Provider:    "test",
		CallID:      "call-1",
		Direction:   "outbound",
		StatusEvent: "ringing",
	}
	toolRequestPayload := []byte("tool-request-before")
	toolResponsePayload := []byte("tool-response-before")
	requestLogError := "request-log-before"
	requestLogRequestPayload := []byte("request-log-request-before")
	requestLogResponsePayload := []byte("request-log-response-before")

	if err := recorder.Record(context.Background(), scope,
		RecordMetric{Metrics: []*protos.Metric{metric}},
		RecordMetadata{Metadata: []*protos.Metadata{metadata}},
		RecordWebhook{Event: CallRinging, Payload: webhookPayload},
		RecordToolLog{
			Operation:       ToolLogOperationCreate,
			ToolCallID:      "tool-1",
			ToolName:        "lookup",
			Status:          "in_progress",
			RequestPayload:  toolRequestPayload,
			ResponsePayload: toolResponsePayload,
		},
		RecordRequestLog{
			Source:          "webhook",
			SourceEvent:     CallRinging.String(),
			ContextID:       "ctx-1",
			Status:          "complete",
			ErrorMessage:    &requestLogError,
			RequestPayload:  requestLogRequestPayload,
			ResponsePayload: requestLogResponsePayload,
		},
	); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	metric.Value = "2000"
	metric.Description = "after"
	metadata.Value = "fr"
	webhookPayload.Extra["status"] = "after"
	webhookNestedPayload["value"] = "after"
	webhookListPayload[0].(map[string]interface{})["value"] = "after"
	webhookBytesPayload[0] = 'x'
	toolRequestPayload[0] = 'x'
	toolResponsePayload[0] = 'x'
	requestLogError = "request-log-after"
	requestLogRequestPayload[0] = 'x'
	requestLogResponsePayload[0] = 'x'

	if err := recorder.AddCollectors(collector); err != nil {
		t.Fatalf("AddCollectors returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if got := collector.metrics[0].Metrics[0]; got.GetValue() != "1000" || got.GetDescription() != "before" {
		t.Fatalf("metric was not snapshotted: %+v", got)
	}
	if got := collector.metadata[0].Metadata[0]; got.GetValue() != "en" {
		t.Fatalf("metadata was not snapshotted: %+v", got)
	}
	webhookRecordPayload, ok := collector.webhooks[0].Payload.(CallRingingWebhookPayload)
	if !ok {
		t.Fatalf("expected CallRingingWebhookPayload, got %T", collector.webhooks[0].Payload)
	}
	if got := webhookRecordPayload.Extra["status"]; got != "before" {
		t.Fatalf("webhook payload was not snapshotted: %v", got)
	}
	if got := webhookRecordPayload.Extra["nested"].(map[string]interface{})["value"]; got != "before" {
		t.Fatalf("webhook nested payload was not snapshotted: %v", got)
	}
	if got := webhookRecordPayload.Extra["list"].([]interface{})[0].(map[string]interface{})["value"]; got != "before" {
		t.Fatalf("webhook list payload was not snapshotted: %v", got)
	}
	if got := string(webhookRecordPayload.Extra["bytes"].([]byte)); got != "before" {
		t.Fatalf("webhook bytes payload was not snapshotted: %s", got)
	}
	if got := string(collector.toolLogs[0].RequestPayload); got != "tool-request-before" {
		t.Fatalf("tool request payload was not snapshotted: %s", got)
	}
	if got := string(collector.toolLogs[0].ResponsePayload); got != "tool-response-before" {
		t.Fatalf("tool response payload was not snapshotted: %s", got)
	}
	if got := *collector.requestLogs[0].ErrorMessage; got != "request-log-before" {
		t.Fatalf("request log error was not snapshotted: %s", got)
	}
	if got := string(collector.requestLogs[0].RequestPayload); got != "request-log-request-before" {
		t.Fatalf("request log request payload was not snapshotted: %s", got)
	}
	if got := string(collector.requestLogs[0].ResponsePayload); got != "request-log-response-before" {
		t.Fatalf("request log response payload was not snapshotted: %s", got)
	}
}

func TestRecorder_NewDoesNotWaitForCollectorRegistration(t *testing.T) {
	collector := &blockingKeyCollector{
		key:     "collector",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}

	startedAt := time.Now()
	recorder := New(WithCollector(collector))
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("New waited for collector registration: %s", elapsed)
	}

	select {
	case <-collector.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for collector registration")
	}
	close(collector.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_AddCollectorsDoesNotWaitForCollectorRegistration(t *testing.T) {
	recorder := New()
	collector := &blockingKeyCollector{
		key:     "collector",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}

	startedAt := time.Now()
	if err := recorder.AddCollectors(collector); err != nil {
		t.Fatalf("AddCollectors returned error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("AddCollectors waited for collector registration: %s", elapsed)
	}

	select {
	case <-collector.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for collector registration")
	}
	close(collector.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_RecordDoesNotWaitForCollectorCollect(t *testing.T) {
	blocked := &blockingCollector{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	recorder := New(WithCollector(blocked))

	startedAt := time.Now()
	if err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	}); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("Record waited for collector collect: %s", elapsed)
	}

	select {
	case <-blocked.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for collector collect")
	}
	close(blocked.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_AddCollectors_ReplaysBufferedRecords(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	recorder := New()

	err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.AddCollectors(collector); err != nil {
		t.Fatalf("AddCollectors returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
	if len(collector.events) != 1 {
		t.Fatalf("expected replayed event, got %d", len(collector.events))
	}
}

func TestRecorder_CollectorWorkersDoNotBlockEachOther(t *testing.T) {
	blocked := &blockingCollector{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	fast := &recordingCollector{key: "fast"}
	recorder := New(WithCollectors(blocked, fast))

	if err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	}); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	select {
	case <-blocked.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocking collector")
	}
	deadline := time.After(time.Second)
	for fast.eventCount() == 0 {
		select {
		case <-deadline:
			t.Fatal("fast collector did not receive event while slow collector was blocked")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	close(blocked.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_CloseGraceAcceptsLateRecords(t *testing.T) {
	collector := &recordingCollector{key: "collector"}
	observabilityRecorder := New(WithGracePeriod(), WithCollector(collector))
	concreteRecorder, ok := observabilityRecorder.(*recorder)
	if !ok {
		t.Fatalf("unexpected recorder type: %T", observabilityRecorder)
	}
	concreteRecorder.closeGracePeriod = 200 * time.Millisecond
	scope := ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}

	closeStarted := time.Now()
	closeContext, cancelCloseContext := context.WithCancel(context.Background())
	cancelCloseContext()
	if err := observabilityRecorder.Close(closeContext); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if elapsed := time.Since(closeStarted); elapsed > 100*time.Millisecond {
		t.Fatalf("Close waited for grace period: %s", elapsed)
	}
	time.Sleep(10 * time.Millisecond)

	if err := observabilityRecorder.Record(context.Background(), scope, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationCleanup,
	}); err != nil {
		t.Fatalf("late record during close grace failed: %v", err)
	}
	if err := waitForRecorderDone(t, observabilityRecorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}

	if len(collector.events) != 1 {
		t.Fatalf("expected late event to be collected, got %d", len(collector.events))
	}
	if err := observabilityRecorder.Record(context.Background(), scope, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationCompleted,
	}); !errors.Is(err, ErrRecorderClosed) {
		t.Fatalf("expected ErrRecorderClosed after close, got %v", err)
	}
}

func TestRecorder_DefaultGracePeriodIsZero(t *testing.T) {
	observabilityRecorder := New()
	concreteRecorder, ok := observabilityRecorder.(*recorder)
	if !ok {
		t.Fatalf("unexpected recorder type: %T", observabilityRecorder)
	}
	if concreteRecorder.closeGracePeriod != 0 {
		t.Fatalf("unexpected default grace period: %s", concreteRecorder.closeGracePeriod)
	}
	if err := concreteRecorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, observabilityRecorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_WithGracePeriodUsesConfiguredGracePeriod(t *testing.T) {
	observabilityRecorder := New(WithGracePeriod())
	concreteRecorder, ok := observabilityRecorder.(*recorder)
	if !ok {
		t.Fatalf("unexpected recorder type: %T", observabilityRecorder)
	}
	if concreteRecorder.closeGracePeriod != recorderCloseGracePeriod {
		t.Fatalf("unexpected grace period: %s", concreteRecorder.closeGracePeriod)
	}
	concreteRecorder.closeGracePeriod = 0
	if err := concreteRecorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, observabilityRecorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_Record_ReturnsBufferFullWhenOperationQueueIsFull(t *testing.T) {
	blocked := &blockingKeyCollector{
		key:     "collector",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	recorder := New(WithCollector(blocked))
	scope := ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}
	record := RecordMetric{
		Metrics: []*protos.Metric{{Name: MetricConversationDuration, Value: "1"}},
	}

	<-blocked.started

	for recordIndex := 0; recordIndex < recorderQueueSize; recordIndex++ {
		if err := recorder.Record(context.Background(), scope, record); err != nil {
			t.Fatalf("queued record %d failed: %v", recordIndex, err)
		}
	}
	err := recorder.Record(context.Background(), scope, record)
	if !errors.Is(err, ErrBufferFull) {
		t.Fatalf("expected ErrBufferFull, got %v", err)
	}

	close(blocked.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_CloseDoesNotWaitForInflightCollect(t *testing.T) {
	blocked := &blockingCollector{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	recorder := New(WithCollector(blocked))

	if err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	<-blocked.started

	closeStarted := time.Now()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if elapsed := time.Since(closeStarted); elapsed > 100*time.Millisecond {
		t.Fatalf("Close waited for inflight collect: %s", elapsed)
	}

	close(blocked.release)
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
}

func TestRecorder_Close_JoinsCollectorErrors(t *testing.T) {
	collectErr := errors.New("collect failed")
	closeErr := errors.New("close failed")
	recorder := New(WithCollector(&recordingCollector{
		key:        "collector",
		collectErr: collectErr,
		closeErr:   closeErr,
	}))

	err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	})
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}

	err = recorder.Close(context.Background())
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	err = waitForRecorderDone(t, recorder)
	if !errors.Is(err, collectErr) || !errors.Is(err, closeErr) {
		t.Fatalf("expected joined collect+close errors, got %v", err)
	}
}

func TestRecorder_AddCollectors_DeduplicatesByKey(t *testing.T) {
	first := &recordingCollector{key: "telemetry"}
	duplicate := &recordingCollector{key: "telemetry"}
	second := &recordingCollector{key: "timeline"}
	recorder := New(WithCollector(first))

	if err := recorder.AddCollectors(duplicate, second); err != nil {
		t.Fatalf("AddCollectors returned error: %v", err)
	}
	err := recorder.Record(context.Background(), ConversationScope{
		AssistantScope: AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, RecordEvent{
		Component: ComponentConversation,
		Event:     ConversationInitializing,
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
	if len(first.events) != 1 || len(second.events) != 1 {
		t.Fatalf("expected original and new collectors to receive event, got first=%d second=%d", len(first.events), len(second.events))
	}
	if len(duplicate.events) != 0 {
		t.Fatalf("expected duplicate collector to be skipped, got %d events", len(duplicate.events))
	}
}

func TestRecorder_AddCollectors_ReturnsClosedError(t *testing.T) {
	recorder := New()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := waitForRecorderDone(t, recorder); err != nil {
		t.Fatalf("recorder close returned error: %v", err)
	}
	if err := recorder.AddCollectors(&recordingCollector{key: "collector"}); !errors.Is(err, ErrRecorderClosed) {
		t.Fatalf("expected ErrRecorderClosed, got %v", err)
	}
}

func TestValidateScope(t *testing.T) {
	if err := ValidateScope(AssistantScope{}); err == nil {
		t.Fatal("expected assistant scope error")
	}
	if err := ValidateScope(ConversationScope{AssistantScope: AssistantScope{AssistantID: 10}}); err == nil {
		t.Fatal("expected conversation scope error")
	}
	if err := ValidateScope(MessageScope{
		ConversationScope: ConversationScope{
			AssistantScope: AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "msg-1",
		Role:      MessageRoleUser,
	}); err != nil {
		t.Fatalf("expected valid message scope, got %v", err)
	}
}
