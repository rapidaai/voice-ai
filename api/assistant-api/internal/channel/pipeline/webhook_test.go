// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

func TestPipelineCallWebhookLifecycleFields(t *testing.T) {
	t.Parallel()

	started := observability.CallStartedWebhookPayload{Status: observability.MetricCallStatusInProgress}
	if started.Status != observability.MetricCallStatusInProgress {
		t.Fatalf("started status = %q", started.Status)
	}

	ended := observability.CallEndedWebhookPayload{
		Status:           observability.MetricCallStatusComplete,
		DisconnectReason: "remote_hangup",
	}
	if ended.Status != observability.MetricCallStatusComplete || ended.DisconnectReason != "remote_hangup" {
		t.Fatalf("unexpected ended lifecycle fields: %+v", ended)
	}
}
