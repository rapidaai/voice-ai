// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_grpc

import (
	"context"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
)

func TestWebhookCollectorWithoutDependenciesIsNoop(t *testing.T) {
	collector := webhook.New(context.Background(), webhook.Config{})
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected no-op collector, got %T", collector)
	}
}
