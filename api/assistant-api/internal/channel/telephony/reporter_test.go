// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
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
}
