// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type usagePublisherStub struct {
	auth   *types.Authentication
	usages []*protos.ProductUsage
	err    error
	closed bool
}

func (stub *usagePublisherStub) CreateProductUsages(_ context.Context, auth *types.Authentication, usages []*protos.ProductUsage) (*protos.CreateProductUsagesResponse, error) {
	stub.auth = auth
	stub.usages = usages
	return &protos.CreateProductUsagesResponse{Success: true}, stub.err
}

func (stub *usagePublisherStub) Close() error {
	stub.closed = true
	return nil
}

func TestCollector_ForwardsUsageRecord(t *testing.T) {
	publisher := &usagePublisherStub{}
	collector := New(publisher)
	now := time.Date(2026, 6, 5, 10, 0, 0, 123456789, time.UTC)
	auth := &types.Authentication{}

	err := collector.Collect(context.Background(), observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 10}, ConversationID: 20,
	}, observability.Context{Auth: auth}, observability.RecordUsage{
		ID:         "usage-1",
		Component:  observability.ComponentName(observability.UsageConversationSTTDuration),
		Provider:   "deepgram",
		Duration:   2 * time.Second,
		Attributes: observability.Attributes{"source": "stt"},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("CollectUsage returned error: %v", err)
	}
	if publisher.auth != auth {
		t.Fatal("expected authentication to be forwarded")
	}
	if len(publisher.usages) != 1 {
		t.Fatalf("expected one usage record, got %d", len(publisher.usages))
	}
	got := publisher.usages[0]
	if got.GetUsageId() != "usage-1" || got.GetUsageType() != observability.UsageConversationSTTDuration {
		t.Fatalf("unexpected usage record: %+v", got)
	}
	if got.GetUsages() != int64(2*time.Second) {
		t.Fatalf("unexpected usage quantity: %d", got.GetUsages())
	}
	if got.GetUnit() != string(types.ProductUsageUnitNanosecond) {
		t.Fatalf("unexpected usage unit: %q", got.GetUnit())
	}
	wantOccurredAt := now.Truncate(time.Microsecond)
	if !got.GetOccurredAt().AsTime().Equal(wantOccurredAt) {
		t.Fatalf("unexpected occurredAt: %s", got.GetOccurredAt().AsTime())
	}
}

func TestCollector_ReturnsPublisherError(t *testing.T) {
	publishErr := errors.New("publish failed")
	collector := New(&usagePublisherStub{err: publishErr})

	err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{}, observability.RecordUsage{
		ID:         "usage-1",
		Component:  observability.ComponentName(observability.UsageConversationTTSDuration),
		Duration:   time.Second,
		OccurredAt: time.Now(),
	})
	if !errors.Is(err, publishErr) {
		t.Fatalf("expected publish error, got %v", err)
	}
}

func TestCollector_GeneratesUsageID(t *testing.T) {
	publisher := &usagePublisherStub{}
	collector := New(publisher)

	err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordUsage{
		Component:  observability.ComponentName(observability.UsageConversationVADDuration),
		Duration:   time.Second,
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(publisher.usages) != 1 || publisher.usages[0].GetUsageId() == "" {
		t.Fatal("Collect() did not generate usage ID")
	}
}

func TestCollector_RejectsUnsupportedUsageType(t *testing.T) {
	publisher := &usagePublisherStub{}
	collector := New(publisher)

	err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordUsage{
		Component:  observability.ComponentUsage,
		Duration:   time.Second,
		OccurredAt: time.Now(),
	})
	if !errors.Is(err, types.ErrInvalidProductUsage) {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(publisher.usages) != 0 {
		t.Fatalf("publisher received %d usages", len(publisher.usages))
	}
}

func TestCollector_IgnoresNonUsageRecord(t *testing.T) {
	publisher := &usagePublisherStub{}
	collector := New(publisher)

	if err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordLog{}); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(publisher.usages) != 0 {
		t.Fatalf("publisher received %d usages", len(publisher.usages))
	}
}

func TestCollector_ClosesPublisher(t *testing.T) {
	publisher := &usagePublisherStub{}
	collector := New(publisher)

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !publisher.closed {
		t.Fatal("Close() did not close publisher")
	}
}
