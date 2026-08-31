package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/telemetry"
)

type openSearchConnectorStub struct {
	bodies []string
}

func (o *openSearchConnectorStub) Connect(context.Context) error { return nil }
func (o *openSearchConnectorStub) Name() string                  { return "opensearch-stub" }
func (o *openSearchConnectorStub) IsConnected(context.Context) bool {
	return true
}
func (o *openSearchConnectorStub) Disconnect(context.Context) error { return nil }
func (o *openSearchConnectorStub) VectorSearch(context.Context, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchConnectorStub) HybridSearch(context.Context, string, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchConnectorStub) TextSearch(context.Context, string, string, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}
func (o *openSearchConnectorStub) Search(context.Context, []string, string) *connectors.SearchResponse {
	return nil
}
func (o *openSearchConnectorStub) SearchWithCount(context.Context, []string, string) *connectors.SearchResponseWithCount {
	return nil
}
func (o *openSearchConnectorStub) Persist(context.Context, string, string, string) error {
	return nil
}
func (o *openSearchConnectorStub) Update(context.Context, string, string, string) error {
	return nil
}
func (o *openSearchConnectorStub) Bulk(_ context.Context, body string) error {
	o.bodies = append(o.bodies, body)
	return nil
}

func TestOpenSearchExporter_ExportsContextToDocuments(t *testing.T) {
	opensearch := &openSearchConnectorStub{}
	exporter := NewOpenSearchExporter(nil, OpenSearchConfig{IndexPrefix: "test"}, opensearch)
	scope := telemetry.Scope{
		ProjectID:      10,
		OrganizationID: 20,
		Name:           "project",
	}
	trace := map[string]string{"traceId": "trace-1"}
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)

	records := []telemetry.Record{
		telemetry.LogRecord{ID: "log-1", Context: trace, Level: "error", Message: "failed", OccurredAt: now},
		telemetry.EventRecord{ID: "event-1", Context: trace, Event: "session.streamer_created", Component: "session", OccurredAt: now},
		telemetry.MetricRecord{ID: "metric-1", Context: trace, Name: "duration", Value: "1", OccurredAt: now},
	}
	expectedIndices := []string{
		`"test-logs-20-20260607"`,
		`"test-events-20-20260607"`,
		`"test-metrics-20-20260607"`,
	}
	for _, record := range records {
		if err := exporter.Export(context.Background(), scope, record); err != nil {
			t.Fatalf("Export returned error: %v", err)
		}
	}

	if len(opensearch.bodies) != len(records) {
		t.Fatalf("expected %d bulk bodies, got %d", len(records), len(opensearch.bodies))
	}
	for index, body := range opensearch.bodies {
		lines := strings.Split(strings.TrimSpace(body), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected bulk metadata+document lines, got %d", len(lines))
		}
		var doc struct {
			Context map[string]string `json:"context"`
		}
		if err := json.Unmarshal([]byte(lines[1]), &doc); err != nil {
			t.Fatalf("failed to unmarshal document: %v", err)
		}
		if doc.Context["traceId"] != "trace-1" {
			t.Fatalf("unexpected trace id: %s", doc.Context["traceId"])
		}
		if !strings.Contains(lines[0], expectedIndices[index]) {
			t.Fatalf("unexpected bulk metadata line: %s", lines[0])
		}
	}
}

func TestOpenSearchExporter_UsesGlobalIndexWithoutOrganization(t *testing.T) {
	opensearch := &openSearchConnectorStub{}
	exporter := NewOpenSearchExporter(nil, OpenSearchConfig{IndexPrefix: "test"}, opensearch)
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)

	records := []telemetry.Record{
		telemetry.LogRecord{ID: "log-1", OccurredAt: now},
		telemetry.EventRecord{ID: "event-1", OccurredAt: now},
		telemetry.MetricRecord{ID: "metric-1", OccurredAt: now},
	}
	expectedIndices := []string{
		`"test-logs-global-20260607"`,
		`"test-events-global-20260607"`,
		`"test-metrics-global-20260607"`,
	}

	for _, record := range records {
		if err := exporter.Export(context.Background(), telemetry.Scope{}, record); err != nil {
			t.Fatalf("Export returned error: %v", err)
		}
	}

	if len(opensearch.bodies) != len(records) {
		t.Fatalf("expected %d bulk bodies, got %d", len(records), len(opensearch.bodies))
	}
	for index, body := range opensearch.bodies {
		if !strings.Contains(body, expectedIndices[index]) {
			t.Fatalf("unexpected bulk body: %s", body)
		}
	}
}
