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
		omitReason       bool
	}{
		{name: "call received", payload: CallReceivedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true},
		{name: "call ringing", payload: CallRingingWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusRinging}, status: WebhookCallStatusRinging.String(), omitReason: true},
		{name: "call provider answered", payload: CallProviderAnsweredWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true},
		{name: "call outbound requested", payload: CallOutboundRequestedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusPending}, status: WebhookCallStatusPending.String(), omitReason: true},
		{name: "call outbound dispatched", payload: CallOutboundDispatchedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true},
		{name: "call started", payload: CallStartedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true},
		{name: "call ended", payload: CallEndedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusCompleted, DisconnectReason: WebhookCallDisconnectReasonRemoteHangup}, status: WebhookCallStatusCompleted.String(), disconnectReason: WebhookCallDisconnectReasonRemoteHangup.String(), omitReason: true},
		{name: "call failed", payload: CallFailedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusFailed, DisconnectReason: WebhookCallDisconnectReasonNoAnswer}, status: WebhookCallStatusFailed.String(), disconnectReason: WebhookCallDisconnectReasonNoAnswer.String(), omitReason: true},
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
			if _, exists := payload["reason"]; testCase.omitReason && exists {
				t.Fatalf("reason should be omitted: %+v", payload)
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

func TestWebhookCallEnums(t *testing.T) {
	t.Parallel()

	if WebhookCallStatusCompleted.String() != "completed" {
		t.Fatalf("completed status = %q", WebhookCallStatusCompleted)
	}
	if WebhookCallDirectionOutbound.String() != "outbound" {
		t.Fatalf("outbound direction = %q", WebhookCallDirectionOutbound)
	}
	if WebhookCallDisconnectReasonProviderFailed.String() != "provider_failed" {
		t.Fatalf("provider failure reason = %q", WebhookCallDisconnectReasonProviderFailed)
	}
}
