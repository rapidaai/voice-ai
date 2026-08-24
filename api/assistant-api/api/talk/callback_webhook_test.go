// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_talk_api

import (
	"encoding/json"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

func TestCallbackWebhookLifecycleFields(t *testing.T) {
	t.Parallel()

	payload := observability.CallFailedWebhookPayload{
		Status:           observability.WebhookCallStatusFailed,
		DisconnectReason: observability.WebhookCallDisconnectReasonProviderFailed,
	}
	if payload.Status != observability.WebhookCallStatusFailed || payload.DisconnectReason != observability.WebhookCallDisconnectReasonProviderFailed {
		t.Fatalf("unexpected callback lifecycle fields: %+v", payload)
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(serialized, &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if fields["status"] != observability.WebhookCallStatusFailed.String() {
		t.Fatalf("status = %v, want %q", fields["status"], observability.WebhookCallStatusFailed)
	}
	if _, exists := fields["status_event"]; exists {
		t.Fatalf("callback webhook should not include status_event: %+v", fields)
	}
}
