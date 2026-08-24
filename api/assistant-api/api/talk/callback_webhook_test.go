// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_talk_api

import (
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
}
