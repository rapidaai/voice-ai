// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"encoding/json"
	"testing"
)

func TestAssistantWebhookPayloadLifecycleFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		payload          V1WebhookPayload
		status           string
		disconnectReason string
	}{
		{name: "call received", payload: CallReceivedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusInProgress}, status: MetricCallStatusInProgress},
		{name: "call ringing", payload: CallRingingWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusRinging}, status: MetricCallStatusRinging},
		{name: "call provider answered", payload: CallProviderAnsweredWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusInProgress}, status: MetricCallStatusInProgress},
		{name: "call outbound requested", payload: CallOutboundRequestedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusInProgress}, status: MetricCallStatusInProgress},
		{name: "call outbound dispatched", payload: CallOutboundDispatchedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusInProgress}, status: MetricCallStatusInProgress},
		{name: "call started", payload: CallStartedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusInProgress}, status: MetricCallStatusInProgress},
		{name: "call ended", payload: CallEndedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusComplete, DisconnectReason: "remote_hangup"}, status: MetricCallStatusComplete, disconnectReason: "remote_hangup"},
		{name: "call failed", payload: CallFailedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: MetricCallStatusFailed, DisconnectReason: "no_answer"}, status: MetricCallStatusFailed, disconnectReason: "no_answer"},
		{name: "conversation begin", payload: ConversationBeginWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: "in_progress"}, status: "in_progress"},
		{name: "conversation resume", payload: ConversationResumeWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: "in_progress"}, status: "in_progress"},
		{name: "conversation completed", payload: ConversationCompletedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: "complete", DisconnectReason: "USER"}, status: "complete", disconnectReason: "USER"},
		{name: "conversation error", payload: ConversationErrorWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Status: "error", DisconnectReason: "ERROR"}, status: "error", disconnectReason: "ERROR"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serialized, err := json.Marshal(testCase.payload)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(serialized, &payload); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if payload["status"] != testCase.status {
				t.Fatalf("status = %v, want %q", payload["status"], testCase.status)
			}
			if testCase.disconnectReason == "" {
				if _, exists := payload["disconnect_reason"]; exists {
					t.Fatalf("disconnect_reason should be omitted: %+v", payload)
				}
				return
			}
			if payload["disconnect_reason"] != testCase.disconnectReason {
				t.Fatalf("disconnect_reason = %v, want %q", payload["disconnect_reason"], testCase.disconnectReason)
			}
		})
	}
}
