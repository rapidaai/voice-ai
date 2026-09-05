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

type productUsageClientStub struct {
	auth  *types.Authentication
	usage *protos.CreateProductUsageRequest
	err   error
}

func (stub *productUsageClientStub) CreateProductUsage(_ context.Context, auth *types.Authentication, usage *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error) {
	stub.auth = auth
	stub.usage = usage
	return &protos.GetProductUsageResponse{Success: true, Data: &protos.ProductUsage{Id: 42}}, stub.err
}

func (stub *productUsageClientStub) Close() error {
	return nil
}

func TestCollector_ForwardsUsageRecord(t *testing.T) {
	productUsageClient := &productUsageClientStub{}
	collector := New(Config{ProductUsageClient: productUsageClient})
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
	if productUsageClient.auth != auth {
		t.Fatal("expected authentication to be forwarded")
	}
	if productUsageClient.usage == nil {
		t.Fatal("expected one usage record")
	}
	got := productUsageClient.usage
	if got.GetUsageType() != observability.UsageConversationSTTDuration {
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
	collector := New(Config{ProductUsageClient: &productUsageClientStub{err: publishErr}})

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

func TestCollector_PublishesUsageWithoutClientID(t *testing.T) {
	productUsageClient := &productUsageClientStub{}
	collector := New(Config{ProductUsageClient: productUsageClient})

	err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordUsage{
		Component:  observability.ComponentName(observability.UsageConversationVADDuration),
		Duration:   time.Second,
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if productUsageClient.usage == nil {
		t.Fatal("Collect() did not publish usage")
	}
}

func TestCollector_RejectsUnsupportedUsageType(t *testing.T) {
	productUsageClient := &productUsageClientStub{}
	collector := New(Config{ProductUsageClient: productUsageClient})

	err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordUsage{
		Component:  observability.ComponentUsage,
		Duration:   time.Second,
		OccurredAt: time.Now(),
	})
	if !errors.Is(err, types.ErrInvalidProductUsage) {
		t.Fatalf("Collect() error = %v", err)
	}
	if productUsageClient.usage != nil {
		t.Fatalf("product usage client received usage: %+v", productUsageClient.usage)
	}
}

func TestCollector_IgnoresNonUsageRecord(t *testing.T) {
	productUsageClient := &productUsageClientStub{}
	collector := New(Config{ProductUsageClient: productUsageClient})

	if err := collector.Collect(context.Background(), observability.ProjectScope{}, observability.Context{}, observability.RecordLog{}); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if productUsageClient.usage != nil {
		t.Fatalf("product usage client received usage: %+v", productUsageClient.usage)
	}
}

func TestCollector_CloseDoesNotCloseProductUsageClient(t *testing.T) {
	if err := New(Config{ProductUsageClient: &productUsageClientStub{}}).Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
