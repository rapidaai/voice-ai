// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"encoding/json"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

func TestProviderStatusWebhookLifecycleFields(t *testing.T) {
	t.Parallel()

	payload := observability.CallFailedWebhookPayload{
		Status:           observability.WebhookCallStatusCancelled,
		DisconnectReason: observability.WebhookCallDisconnectReasonCancelled,
	}
	if payload.Status != observability.WebhookCallStatusCancelled || payload.DisconnectReason != observability.WebhookCallDisconnectReasonCancelled {
		t.Fatalf("unexpected provider status lifecycle fields: %+v", payload)
	}

	ringing := observability.CallRingingWebhookPayload{Status: observability.WebhookCallStatusRinging}
	serialized, err := json.Marshal(ringing)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(serialized, &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if fields["status"] != observability.WebhookCallStatusRinging.String() {
		t.Fatalf("status = %v, want %q", fields["status"], observability.WebhookCallStatusRinging)
	}
	if fields["source"] != "" {
		t.Fatalf("source = %v, want empty", fields["source"])
	}
	if _, exists := fields["status_event"]; exists {
		t.Fatalf("ringing webhook should not include status_event: %+v", fields)
	}
}
