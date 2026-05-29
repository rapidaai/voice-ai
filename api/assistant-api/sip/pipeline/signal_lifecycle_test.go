// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type lifecycleEventCapture struct {
	eventType string
	session   *sip_infra.Session
	payload   map[string]interface{}
}

func newLifecycleTestDispatcher(t *testing.T, ch chan lifecycleEventCapture) *Dispatcher {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(false),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return NewDispatcher(&DispatcherConfig{
		Logger: logger,
		OnLifecycleWebhook: func(ctx context.Context, eventType string, session *sip_infra.Session, payload map[string]interface{}) {
			ch <- lifecycleEventCapture{
				eventType: eventType,
				session:   session,
				payload:   payload,
			}
		},
	})
}

func newLifecycleTestSession(t *testing.T, callID string) *sip_infra.Session {
	t.Helper()
	session, err := sip_infra.NewSession(context.Background(), &sip_infra.SessionConfig{
		Config: &sip_infra.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10100,
		},
		Direction: sip_infra.CallDirectionInbound,
		CallID:    callID,
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	session.SetMetadata(sip_infra.MetadataCallFromURI, "sip:+15551234567@pbx.example.com")
	session.SetMetadata(sip_infra.MetadataCallToURI, "sip:+18005550100@pbx.example.com")
	session.SetConversationID(42)

	projectID := uint64(1001)
	organizationID := uint64(2002)
	session.SetAuth(&types.ProjectScope{
		ProjectId:      &projectID,
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	})

	assistant := &internal_assistant_entity.Assistant{}
	assistant.Id = 77
	session.SetAssistant(assistant)
	return session
}

func waitLifecycleEvent(t *testing.T, ch <-chan lifecycleEventCapture) lifecycleEventCapture {
	t.Helper()
	select {
	case evt := <-ch:
		return evt
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lifecycle event")
		return lifecycleEventCapture{}
	}
}

func TestDispatch_CallRinging_EmitsLifecycleWebhook(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-ring-1")

	d.dispatch(context.Background(), sip_infra.CallRingingPipeline{
		ID:      "call-ring-1",
		Session: session,
		FromURI: "sip:+15551234567@pbx.example.com",
		ToURI:   "sip:+18005550100@pbx.example.com",
	})

	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "call.ringing" {
		t.Fatalf("expected event type call.ringing, got %s", evt.eventType)
	}
	if got := evt.payload["call_id"]; got != "call-ring-1" {
		t.Fatalf("expected call_id call-ring-1, got %v", got)
	}
	if got := evt.payload["state"]; got != "ringing" {
		t.Fatalf("expected state ringing, got %v", got)
	}
	if got := evt.payload["assistant_id"]; got != "77" {
		t.Fatalf("expected assistant_id 77, got %v", got)
	}
}

func TestDispatch_CallCreated_EmitsLifecycleWebhook(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-created-1")

	d.dispatch(context.Background(), sip_infra.CallCreatedPipeline{
		ID:      "call-created-1",
		Session: session,
		FromURI: "sip:+15551234567@pbx.example.com",
		ToURI:   "sip:+18005550100@pbx.example.com",
	})

	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "call.created" {
		t.Fatalf("expected event type call.created, got %s", evt.eventType)
	}
	if got := evt.payload["state"]; got != "created" {
		t.Fatalf("expected state created, got %v", got)
	}
}

func TestDispatch_TransferAttemptStarted_EmitsAttemptPayload(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-transfer-1")

	d.dispatch(context.Background(), sip_infra.TransferAttemptStartedPipeline{
		ID:          "call-transfer-1",
		Session:     session,
		TransferID:  "transfer:call-transfer-1",
		TargetURI:   "sip:101@pbx.example.com",
		Attempt:     1,
		Total:       3,
		RoutingMode: "sequential",
	})

	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "transfer.attempt.started" {
		t.Fatalf("expected event type transfer.attempt.started, got %s", evt.eventType)
	}
	if got := evt.payload["target_uri"]; got != "sip:101@pbx.example.com" {
		t.Fatalf("expected target_uri sip:101@pbx.example.com, got %v", got)
	}
	if got := evt.payload["attempt"]; got != 1 {
		t.Fatalf("expected attempt 1, got %v", got)
	}
	if got := evt.payload["total_attempts"]; got != 3 {
		t.Fatalf("expected total 3, got %v", got)
	}
	if got := evt.payload["state"]; got != "attempting" {
		t.Fatalf("expected state attempting, got %v", got)
	}
}

func TestDispatch_CallFailedWithoutSession_MapsNoAnswer(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)

	d.dispatch(context.Background(), sip_infra.CallFailedPipeline{
		ID:      "call-failed-no-answer",
		Session: nil,
		Error:   context.DeadlineExceeded,
		SIPCode: 408,
	})

	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "call.no_answer" {
		t.Fatalf("expected event type call.no_answer, got %s", evt.eventType)
	}
	if got := evt.payload["reason"]; got != reasonNoAnswer {
		t.Fatalf("expected reason no_answer, got %v", got)
	}
}

func TestDispatch_CallFailedWithoutSession_MapsBusyAndRejected(t *testing.T) {
	tcs := []struct {
		name      string
		err       error
		sipCode   int
		eventType string
		reason    string
	}{
		{
			name:      "busy",
			err:       errors.New("486 Busy Here"),
			sipCode:   486,
			eventType: "call.busy",
			reason:    reasonBusy,
		},
		{
			name:      "rejected",
			err:       errors.New("603 Decline"),
			sipCode:   603,
			eventType: "call.rejected",
			reason:    reasonRejected,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan lifecycleEventCapture, 1)
			d := newLifecycleTestDispatcher(t, ch)
			d.dispatch(context.Background(), sip_infra.CallFailedPipeline{
				ID:      "call-failed-" + tc.name,
				Session: nil,
				Error:   tc.err,
				SIPCode: tc.sipCode,
			})
			evt := waitLifecycleEvent(t, ch)
			if evt.eventType != tc.eventType {
				t.Fatalf("expected event type %s, got %s", tc.eventType, evt.eventType)
			}
			if got := evt.payload["reason"]; got != tc.reason {
				t.Fatalf("expected reason %s, got %v", tc.reason, got)
			}
		})
	}
}

func TestDispatch_TransferRequested_MapsPayload(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-transfer-requested")

	d.dispatch(context.Background(), sip_infra.TransferRequestedPipeline{
		ID:                 "call-transfer-requested",
		Session:            session,
		TransferID:         "transfer:call-transfer-requested",
		Targets:            []string{"sip:101@pbx.example.com", "sip:102@pbx.example.com"},
		RoutingMode:        "sequential",
		PostTransferAction: "resume_ai",
	})
	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "transfer.requested" {
		t.Fatalf("expected transfer.requested, got %s", evt.eventType)
	}
	if got := evt.payload["transfer_id"]; got != "transfer:call-transfer-requested" {
		t.Fatalf("expected transfer id, got %v", got)
	}
	if got := evt.payload["total_attempts"]; got != 2 {
		t.Fatalf("expected total attempts 2, got %v", got)
	}
}

func TestDispatch_TransferAttemptEnded_MapsTerminalState(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-transfer-ended")

	d.dispatch(context.Background(), sip_infra.TransferAttemptEndedPipeline{
		ID:          "call-transfer-ended",
		Session:     session,
		TransferID:  "transfer:call-transfer-ended",
		AttemptID:   "transfer:call-transfer-ended:1",
		TargetURI:   "sip:101@pbx.example.com",
		Attempt:     1,
		Total:       3,
		RoutingMode: "sequential",
		State:       "busy",
		Reason:      reasonBusy,
	})
	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "transfer.attempt.ended" {
		t.Fatalf("expected transfer.attempt.ended, got %s", evt.eventType)
	}
	if got := evt.payload["state"]; got != "busy" {
		t.Fatalf("expected state busy, got %v", got)
	}
	if got := evt.payload["reason"]; got != reasonBusy {
		t.Fatalf("expected reason busy, got %v", got)
	}
}

func TestDispatch_TransferCancelled_AnsweredByOther(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)
	session := newLifecycleTestSession(t, "call-transfer-cancel")

	d.dispatch(context.Background(), sip_infra.TransferCancelledPipeline{
		ID:          "call-transfer-cancel",
		Session:     session,
		TransferID:  "transfer:call-transfer-cancel",
		TargetURI:   "sip:102@pbx.example.com",
		Attempt:     2,
		Total:       3,
		RoutingMode: "parallel",
		Reason:      reasonAnsweredOther,
		AnsweredBy:  "sip:101@pbx.example.com",
	})
	evt := waitLifecycleEvent(t, ch)
	if evt.eventType != "transfer.cancelled" {
		t.Fatalf("expected transfer.cancelled, got %s", evt.eventType)
	}
	if got := evt.payload["reason"]; got != reasonAnsweredOther {
		t.Fatalf("expected reason answered_by_other, got %v", got)
	}
	if got := evt.payload["answered_by"]; got != "sip:101@pbx.example.com" {
		t.Fatalf("expected answered_by to be set, got %v", got)
	}
}

func TestDispatch_CallFailedWithoutSession_EmitsMinimalFailurePayload(t *testing.T) {
	ch := make(chan lifecycleEventCapture, 1)
	d := newLifecycleTestDispatcher(t, ch)

	d.dispatch(context.Background(), sip_infra.CallFailedPipeline{
		ID:      "call-failed-unknown",
		Session: nil,
		Error:   errors.New("unclassified"),
	})

	evt := waitLifecycleEvent(t, ch)
	if got := evt.payload["call_id"]; got != "call-failed-unknown" {
		t.Fatalf("expected call_id call-failed-unknown, got %v", got)
	}
	if got := evt.payload["state"]; got != "failed" {
		t.Fatalf("expected state failed, got %v", got)
	}
	if got := evt.payload["reason"]; got == nil || got == "" {
		t.Fatalf("expected non-empty normalized failure reason")
	}
}
