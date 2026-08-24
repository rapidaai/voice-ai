// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"encoding/json"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

func TestPipelineCallWebhookLifecycleFields(t *testing.T) {
	t.Parallel()

	started := observability.CallStartedWebhookPayload{Status: observability.WebhookCallStatusInProgress}
	if started.Status != observability.WebhookCallStatusInProgress {
		t.Fatalf("started status = %q", started.Status)
	}

	ended := observability.CallEndedWebhookPayload{
		Status:           observability.WebhookCallStatusCompleted,
		DisconnectReason: observability.WebhookCallDisconnectReasonRemoteHangup,
	}
	if ended.Status != observability.WebhookCallStatusCompleted || ended.DisconnectReason != observability.WebhookCallDisconnectReasonRemoteHangup {
		t.Fatalf("unexpected ended lifecycle fields: %+v", ended)
	}

	dispatched := observability.CallOutboundDispatchedWebhookPayload{Status: observability.WebhookCallStatusInProgress}
	serialized, err := json.Marshal(dispatched)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(serialized, &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if fields["status"] != observability.WebhookCallStatusInProgress.String() {
		t.Fatalf("status = %v, want %q", fields["status"], observability.WebhookCallStatusInProgress)
	}
	if _, exists := fields["status_event"]; exists {
		t.Fatalf("outbound dispatched webhook should not include status_event: %+v", fields)
	}
}
