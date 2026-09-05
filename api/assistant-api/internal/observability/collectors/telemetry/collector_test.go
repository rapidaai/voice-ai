// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/configs"
	telemetry "github.com/rapidaai/pkg/telemetry"
	"github.com/rapidaai/protos"
)

type exporterStub struct {
	events  []telemetry.EventRecord
	metrics []telemetry.MetricRecord
	logs    []telemetry.LogRecord
	scopes  []telemetry.Scope
	closed  bool
	err     error
}

func TestNewWithAssistantConfig_ReturnsConfiguredCollector(t *testing.T) {
	collectors := NewWithAssistantConfig(context.Background(), nil, &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			Logging:       &configs.TelemetryLoggingConfig{},
		},
	})
	if len(collectors) != 1 {
		t.Fatalf("expected one collector, got %d", len(collectors))
	}
	if collectors[0].Key() != "telemetry:logging" {
		t.Fatalf("unexpected collector key: %q", collectors[0].Key())
	}
}

func TestNewWithAssistantConfig_ReturnsEmptyWithoutTelemetry(t *testing.T) {
	if collectors := NewWithAssistantConfig(context.Background(), nil, nil); len(collectors) != 0 {
		t.Fatalf("expected no collectors, got %d", len(collectors))
	}
	if collectors := NewWithAssistantConfig(context.Background(), nil, &assistant_config.AssistantConfig{}); len(collectors) != 0 {
		t.Fatalf("expected no collectors, got %d", len(collectors))
	}
}

func TestNewWithAssistantConfig_ReturnsEmptyForUnsupportedTelemetry(t *testing.T) {
	collectors := NewWithAssistantConfig(context.Background(), nil, &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{TelemetryType: "unsupported"},
	})
	if len(collectors) != 0 {
		t.Fatalf("expected no collectors, got %d", len(collectors))
	}
}

func (e *exporterStub) Export(_ context.Context, scope telemetry.Scope, rec telemetry.Record) error {
	e.scopes = append(e.scopes, scope)
	switch typed := rec.(type) {
	case telemetry.LogRecord:
		e.logs = append(e.logs, typed)
	case telemetry.EventRecord:
		e.events = append(e.events, typed)
	case telemetry.MetricRecord:
		e.metrics = append(e.metrics, typed)
	}
	return e.err
}

func (e *exporterStub) Close(context.Context) error {
	e.closed = true
	return e.err
}

func TestNew_ReturnsNoopWithoutExporter(t *testing.T) {
	collector, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected noop collector, got %T", collector)
	}
}

func TestCollector_ExportsEventsAndMetrics(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{
			GlobalScope: observability.GlobalScope{
				ProjectID:      30,
				OrganizationID: 40,
			},
			AssistantID: 10,
		},
		ConversationID: 20,
	}

	err = collector.Collect(context.Background(), scope, observability.Context{TraceID: "trace-1"}, observability.RecordEvent{
		Component:  observability.ComponentCall,
		Event:      observability.CallRinging,
		Attributes: observability.Attributes{"status": "ringing"},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("CollectEvent returned error: %v", err)
	}

	err = collector.Collect(context.Background(), scope, observability.Context{TraceID: "trace-1"}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricConversationDuration,
			Value:       "1000",
			Description: "duration",
		}},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}

	if len(exporter.events) != 1 {
		t.Fatalf("expected one event, got %d", len(exporter.events))
	}
	event := exporter.events[0]
	if event.Event != observability.CallRinging.String() || !event.OccurredAt.Equal(now) {
		t.Fatalf("unexpected event record: %+v", event)
	}
	if event.Component != observability.ComponentCall.String() {
		t.Fatalf("unexpected event component: %+v", event)
	}
	if event.Context["traceId"] != "trace-1" {
		t.Fatalf("unexpected event trace id: %s", event.Context["traceId"])
	}
	if len(exporter.scopes) < 2 {
		t.Fatalf("expected exported scopes, got %d", len(exporter.scopes))
	}
	eventScope := exporter.scopes[0]
	if eventScope.ProjectID != 30 || eventScope.OrganizationID != 40 ||
		eventScope.ScopeAttributes["assistantId"] != "10" ||
		eventScope.ScopeAttributes["assistantConversationId"] != "20" {
		t.Fatalf("unexpected event scope: %+v", eventScope)
	}
	if event.Attributes["status"] != "ringing" {
		t.Fatalf("unexpected event attributes: %+v", event.Attributes)
	}

	if len(exporter.metrics) != 1 {
		t.Fatalf("expected one metric, got %d", len(exporter.metrics))
	}
	metric := exporter.metrics[0]
	metricScope := exporter.scopes[1]
	if metricScope.ScopeAttributes["assistantConversationId"] != "20" ||
		metric.Name != observability.MetricConversationDuration ||
		metric.Value != "1000" ||
		metric.Context["traceId"] != "trace-1" {
		t.Fatalf("unexpected metric record: scope=%+v metric=%+v", metricScope, metric)
	}
}

func TestCollector_ExportsMessageMetrics(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	scope := observability.MessageScope{
		ConversationScope: observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "user-ctx-1",
		Role:      observability.MessageRoleUser,
	}
	err = collector.Collect(context.Background(), scope, observability.Context{}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:  observability.MetricUserTurn,
			Value: "complete",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}
	metric := exporter.metrics[0]
	metricScope := exporter.scopes[0]
	if metricScope.ScopeAttributes["messageId"] != "user-ctx-1" ||
		metricScope.ScopeAttributes["messageRole"] != "user" ||
		metricScope.ScopeAttributes["assistantConversationId"] != "20" {
		t.Fatalf("unexpected message metric: scope=%+v metric=%+v", metricScope, metric)
	}
}

func TestCollector_ReturnsExporterErrors(t *testing.T) {
	exportErr := errors.New("export failed")
	collector, err := New(context.Background(), Config{Exporters: &exporterStub{err: exportErr}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}, observability.Context{}, observability.RecordEvent{
		Event: observability.CallRinging,
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("expected export error, got %v", err)
	}
}

func TestCollector_CloseExporter(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !exporter.closed {
		t.Fatal("expected exporter to close")
	}
}

var _ telemetry.Exporter = (*exporterStub)(nil)
