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
		omitStatusEvent  bool
	}{
		{name: "call received", payload: CallReceivedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true, omitStatusEvent: true},
		{name: "call ringing", payload: CallRingingWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusRinging}, status: WebhookCallStatusRinging.String(), omitReason: true, omitStatusEvent: true},
		{name: "call provider answered", payload: CallProviderAnsweredWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true, omitStatusEvent: true},
		{name: "call outbound requested", payload: CallOutboundRequestedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusPending}, status: WebhookCallStatusPending.String(), omitReason: true, omitStatusEvent: true},
		{name: "call outbound dispatched", payload: CallOutboundDispatchedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true, omitStatusEvent: true},
		{name: "call started", payload: CallStartedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusInProgress}, status: WebhookCallStatusInProgress.String(), omitReason: true, omitStatusEvent: true},
		{name: "call ended", payload: CallEndedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionInbound, Status: WebhookCallStatusCompleted, DisconnectReason: WebhookCallDisconnectReasonRemoteHangup}, status: WebhookCallStatusCompleted.String(), disconnectReason: WebhookCallDisconnectReasonRemoteHangup.String(), omitReason: true, omitStatusEvent: true},
		{name: "call failed", payload: CallFailedWebhookPayload{V1WebhookPayloadBase: NewV1WebhookPayload(nil), Direction: WebhookCallDirectionOutbound, Status: WebhookCallStatusFailed, DisconnectReason: WebhookCallDisconnectReasonNoAnswer}, status: WebhookCallStatusFailed.String(), disconnectReason: WebhookCallDisconnectReasonNoAnswer.String(), omitReason: true, omitStatusEvent: true},
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
			if _, exists := payload["status_event"]; testCase.omitStatusEvent && exists {
				t.Fatalf("status_event should be omitted: %+v", payload)
			}
			if testCase.disconnectReason == "" {
				if _, exists := payload["disconnectReason"]; exists {
					t.Fatalf("disconnectReason should be omitted: %+v", payload)
				}
				if _, exists := payload["disconnect_reason"]; exists {
					t.Fatalf("disconnect_reason should be omitted: %+v", payload)
				}
				return
			}
			if payload["disconnectReason"] != testCase.disconnectReason {
				t.Fatalf("disconnectReason = %v, want %q", payload["disconnectReason"], testCase.disconnectReason)
			}
			if _, exists := payload["disconnect_reason"]; exists {
				t.Fatalf("disconnect_reason should be omitted: %+v", payload)
			}
		})
	}
}

func TestWebhookPayloadJSONFieldNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		payload       V1WebhookPayload
		presentFields map[string]interface{}
		absentFields  []string
	}{
		{
			name: "call payload uses camelcase lifecycle keys",
			payload: CallFailedWebhookPayload{
				V1WebhookPayloadBase: NewV1WebhookPayload(nil),
				CallID:               "call-1",
				ContextID:            "ctx-1",
				DurationMs:           "1200",
				Status:               WebhookCallStatusFailed,
				DisconnectReason:     WebhookCallDisconnectReasonProviderFailed,
			},
			presentFields: map[string]interface{}{
				"callId":           "call-1",
				"contextId":        "ctx-1",
				"durationMs":       "1200",
				"disconnectReason": WebhookCallDisconnectReasonProviderFailed.String(),
			},
			absentFields: []string{"call_id", "context_id", "duration_ms", "disconnect_reason"},
		},
		{
			name: "conversation payload uses camelcase message count",
			payload: ConversationResumeWebhookPayload{
				V1WebhookPayloadBase: NewV1WebhookPayload(nil),
				MessageCount:         "3",
				Status:               "in_progress",
			},
			presentFields: map[string]interface{}{"messageCount": "3"},
			absentFields:  []string{"message_count"},
		},
		{
			name: "webrtc payload uses camelcase session identifiers",
			payload: WebRTCConnectedWebhookPayload{
				V1WebhookPayloadBase: NewV1WebhookPayload(nil),
				SessionID:            "session-1",
				MediaSessionID:       42,
				ICELatencyMs:         15,
				PeerConnectionState:  "connected",
			},
			presentFields: map[string]interface{}{
				"sessionId":           "session-1",
				"mediaSessionId":      float64(42),
				"iceLatencyMs":        float64(15),
				"peerConnectionState": "connected",
			},
			absentFields: []string{"session_id", "media_session_id", "ice_latency_ms", "peer_connection_state"},
		},
		{
			name: "webrtc payload uses camelcase restart counters",
			payload: WebRTCReconnectingWebhookPayload{
				V1WebhookPayloadBase: NewV1WebhookPayload(nil),
				SessionID:            "session-2",
				MediaSessionID:       84,
				RestartAttempt:       2,
				RestartLimit:         5,
			},
			presentFields: map[string]interface{}{
				"restartAttempt": float64(2),
				"restartLimit":   float64(5),
			},
			absentFields: []string{"restart_attempt", "restart_limit"},
		},
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

			for key, want := range testCase.presentFields {
				if payload[key] != want {
					t.Fatalf("%s = %v, want %v", key, payload[key], want)
				}
			}
			for _, key := range testCase.absentFields {
				if _, exists := payload[key]; exists {
					t.Fatalf("%s should be omitted: %+v", key, payload)
				}
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
