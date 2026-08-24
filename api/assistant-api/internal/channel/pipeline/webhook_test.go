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
}
