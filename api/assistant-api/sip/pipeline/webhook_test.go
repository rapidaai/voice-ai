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

	endedPayload := observability.CallEndedWebhookPayload{
		Status:           observability.WebhookCallStatusCompleted,
		DisconnectReason: observability.WebhookCallDisconnectReasonAssistantEnded,
	}
	if endedPayload.Status != observability.WebhookCallStatusCompleted || endedPayload.DisconnectReason != observability.WebhookCallDisconnectReasonAssistantEnded {
		t.Fatalf("unexpected SIP ended lifecycle fields: %+v", endedPayload)
	}

	failedPayload := observability.CallFailedWebhookPayload{
		Status:           observability.WebhookCallStatusFailed,
		DisconnectReason: observability.WebhookCallDisconnectReasonInternalError,
	}
	if failedPayload.Status != observability.WebhookCallStatusFailed || failedPayload.DisconnectReason != observability.WebhookCallDisconnectReasonInternalError {
		t.Fatalf("unexpected SIP failed lifecycle fields: %+v", failedPayload)
	}
}
