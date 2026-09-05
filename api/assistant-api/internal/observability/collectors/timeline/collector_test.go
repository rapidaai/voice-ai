// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
)

type openSearchStub struct {
	bodies []string
	err    error
}

func (o *openSearchStub) Connect(context.Context) error { return nil }
func (o *openSearchStub) Name() string                  { return "opensearch-stub" }
func (o *openSearchStub) IsConnected(context.Context) bool {
	return true
}
func (o *openSearchStub) Disconnect(context.Context) error { return nil }
func (o *openSearchStub) VectorSearch(context.Context, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchStub) HybridSearch(context.Context, string, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchStub) TextSearch(context.Context, string, string, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchStub) Search(context.Context, []string, string) *connectors.SearchResponse {
	return nil
}
func (o *openSearchStub) SearchWithCount(context.Context, []string, string) *connectors.SearchResponseWithCount {
	return nil
}
func (o *openSearchStub) Persist(context.Context, string, string, string) error { return nil }
func (o *openSearchStub) Update(context.Context, string, string, string) error  { return nil }
func (o *openSearchStub) Bulk(_ context.Context, body string) error {
	o.bodies = append(o.bodies, body)
	return o.err
}

func TestNew_ReturnsNoopWithoutOpenSearch(t *testing.T) {
	collector, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected noop collector, got %T", collector)
	}
}

func TestNew_ReturnsErrorForIncompleteOpenSearchConfig(t *testing.T) {
	_, err := New(context.Background(), Config{
		OpenSearchConfig: &configs.OpenSearchConfig{Schema: "http"},
	})
	if err == nil {
		t.Fatal("expected opensearch config error")
	}
}

func TestCollector_PushesEventDocumentToOpenSearchBulk(t *testing.T) {
	opensearch := &openSearchStub{}
	collector, err := New(context.Background(), Config{OpenSearch: opensearch})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	now := time.Date(2026, 6, 5, 12, 30, 0, 0, time.UTC)

	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{
			GlobalScope: observability.GlobalScope{
				OrganizationID: 1,
				ProjectID:      2,
			},
			AssistantID: 3,
		},
		ConversationID: 4,
	}
	err = collector.Collect(context.Background(), scope, observability.Context{}, observability.RecordEvent{
		ID:         "evt-1",
		Component:  observability.ComponentCall,
		Event:      observability.CallRinging,
		Attributes: observability.Attributes{"status": "ringing"},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("CollectEvent returned error: %v", err)
	}
	if len(opensearch.bodies) != 1 {
		t.Fatalf("expected one bulk body, got %d", len(opensearch.bodies))
	}

	lines := strings.Split(strings.TrimSpace(opensearch.bodies[0]), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected bulk metadata+document lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"rapida-timeline-1-20260605"`) || !strings.Contains(lines[0], `"evt-1"`) {
		t.Fatalf("unexpected bulk metadata line: %s", lines[0])
	}

	var doc document
	if err := json.Unmarshal([]byte(lines[1]), &doc); err != nil {
		t.Fatalf("failed to unmarshal document: %v", err)
	}
	if doc.ID != "evt-1" || doc.Kind != "event" || doc.Name != observability.CallRinging.String() {
		t.Fatalf("unexpected doc identity: %+v", doc)
	}
	if doc.OrganizationID != 1 || doc.ProjectID != 2 || doc.AssistantID != 3 || doc.AssistantConversationID != 4 {
		t.Fatalf("unexpected doc scope: %+v", doc)
	}
	if doc.Attributes["status"] != "ringing" {
		t.Fatalf("unexpected doc attributes: %+v", doc.Attributes)
	}
}

func TestCollector_UsesCustomIndexPrefix(t *testing.T) {
	opensearch := &openSearchStub{}
	collector, err := New(context.Background(), Config{OpenSearch: opensearch, IndexPrefix: "custom-timeline"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)

	err = collector.Collect(context.Background(), observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 3},
		ConversationID: 4,
	}, observability.Context{}, observability.RecordMetric{
		ID:         "metric-1",
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}
	if !strings.Contains(opensearch.bodies[0], `"custom-timeline-global-20260605"`) {
		t.Fatalf("unexpected bulk body: %s", opensearch.bodies[0])
	}
}

func TestCollector_ReturnsBulkError(t *testing.T) {
	bulkErr := errors.New("bulk failed")
	collector, err := New(context.Background(), Config{OpenSearch: &openSearchStub{err: bulkErr}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 3}, observability.Context{}, observability.RecordLog{
		Message: "hello",
	})
	if !errors.Is(err, bulkErr) {
		t.Fatalf("expected bulk error, got %v", err)
	}
}

func TestNewDocumentFallsBackToNowWhenOccurredAtMissing(t *testing.T) {
	doc := newDocument("event", observability.AssistantScope{AssistantID: 1}, "", time.Time{})
	if doc.OccurredAt.IsZero() {
		t.Fatal("expected occurred_at to be set")
	}
}

var _ connectors.OpenSearchConnector = (*openSearchStub)(nil)
