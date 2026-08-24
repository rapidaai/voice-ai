// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

func TestSIPWebhookLifecycleFields(t *testing.T) {
	t.Parallel()

	payload := observability.CallEndedWebhookPayload{
		Status:           observability.WebhookCallStatusCompleted,
		DisconnectReason: observability.WebhookCallDisconnectReasonAssistantEnded,
	}
	if payload.Status != observability.WebhookCallStatusCompleted || payload.DisconnectReason != observability.WebhookCallDisconnectReasonAssistantEnded {
		t.Fatalf("unexpected SIP lifecycle fields: %+v", payload)
	}
}
