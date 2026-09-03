// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
)

// inboundDialog owns SIP dialog responses for an inbound INVITE.
// It sends provisional/final responses and tracks final-response state; call setup state stays in HandleInvite.
type inboundDialog struct {
	server        *Server
	session       *Session
	request       *sip.Request
	transaction   sip.ServerTransaction
	dialogSession *sipgo.DialogServerSession
	callID        string
	inviteKey     inboundInviteKey

	mu                   sync.Mutex
	finalResponseStarted bool
	ringingCancel        context.CancelFunc
	ringingStopped       chan struct{}
}

func NewInboundDialog(
	server *Server,
	session *Session,
	request *sip.Request,
	transaction sip.ServerTransaction,
	inviteKey inboundInviteKey,
) (*inboundDialog, *inboundFailure) {
	if session == nil {
		err := fmt.Errorf("inbound dialog requires a session")
		return nil, &inboundFailure{
			statusCode:      500,
			class:           inboundFailureDialog,
			responseClass:   internal_inbound.FailureDialog,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_dialog"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             err,
		}
	}

	dialogSession, err := server.dialogServerCache.ReadInvite(request, transaction)
	if err != nil {
		dialogErr := fmt.Errorf("failed to create inbound dialog session: %w", err)
		return nil, &inboundFailure{
			statusCode:      500,
			class:           inboundFailureDialog,
			responseClass:   internal_inbound.FailureDialog,
			reason:          dialogErr.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_dialog"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             dialogErr,
		}
	}

	return &inboundDialog{
		server:        server,
		session:       session,
		request:       request,
		transaction:   transaction,
		dialogSession: dialogSession,
		callID:        inviteKey.callID,
		inviteKey:     inviteKey,
	}, nil
}

func (dialog *inboundDialog) DialogSession() *sipgo.DialogServerSession {
	if dialog == nil {
		return nil
	}
	return dialog.dialogSession
}

func (dialog *inboundDialog) StartRinging(ctx context.Context) error {
	if dialog.dialogSession == nil {
		return fmt.Errorf("inbound dialog session is required before ringing")
	}
	dialog.mu.Lock()
	if dialog.ringingCancel != nil {
		dialog.mu.Unlock()
		return nil
	}
	dialog.mu.Unlock()
	if err := dialog.sendRingingResponse(); err != nil {
		return err
	}

	ringingContext, ringingCancel := context.WithCancel(ctx)
	ringingStopped := make(chan struct{})
	dialog.mu.Lock()
	dialog.ringingCancel = ringingCancel
	dialog.ringingStopped = ringingStopped
	dialog.mu.Unlock()
	go dialog.runRingingLoop(ringingContext, ringingStopped)
	return nil
}

func (dialog *inboundDialog) StopRinging() {
	dialog.mu.Lock()
	cancel := dialog.ringingCancel
	stopped := dialog.ringingStopped
	dialog.ringingCancel = nil
	dialog.ringingStopped = nil
	dialog.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-stopped
}

func (dialog *inboundDialog) sendRingingResponse() error {
	if dialog.dialogSession == nil {
		return fmt.Errorf("inbound dialog session is required before ringing")
	}
	if err := dialog.dialogSession.Respond(180, "Ringing", nil); err != nil {
		return fmt.Errorf("failed to send 180 Ringing: %w", err)
	}
	return nil
}

func (dialog *inboundDialog) runRingingLoop(ctx context.Context, stopped chan<- struct{}) {
	defer close(stopped)
	ticker := time.NewTicker(dialog.server.effectiveInboundRingingInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dialog.server.ctx.Done():
			return
		case <-dialog.session.Context().Done():
			return
		case <-ticker.C:
		}
		dialog.mu.Lock()
		finalResponseStarted := dialog.finalResponseStarted
		dialog.mu.Unlock()
		if finalResponseStarted {
			return
		}
		if err := dialog.sendRingingResponse(); err != nil {
			dialog.server.logger.Warnw("Inbound SIP ringing retransmit failed",
				"error", err,
				"call_id", dialog.callID)
			return
		}
	}
}

func (dialog *inboundDialog) AnswerAndWaitACK(ctx context.Context, sdpBody string, ackTimeout time.Duration, onFinalResponseSent func(time.Time)) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dialog.StopRinging()
	if !dialog.server.beginPendingInviteFinalResponse(dialog.inviteKey) {
		return ErrInboundInviteCancelled
	}
	if err := dialog.server.sendSDPResponseAndWaitACK(
		dialog.transaction,
		dialog.request,
		dialog.session,
		sdpBody,
		LifecycleReasonInboundInviteACKReceived,
		ackTimeout,
		func(answeredAt time.Time) {
			dialog.mu.Lock()
			dialog.finalResponseStarted = true
			dialog.mu.Unlock()
			if onFinalResponseSent != nil {
				onFinalResponseSent(answeredAt)
			}
		},
	); err != nil {
		if err == ErrInboundACKTimeout {
			return fmt.Errorf("%w: initial INVITE ACK not received", ErrInboundACKTimeout)
		}
		return fmt.Errorf("failed to send inbound 200 OK: %w", err)
	}
	return nil
}

func (dialog *inboundDialog) CancelBeforeAnswer() {
	if dialog == nil {
		return
	}
	dialog.StopRinging()
	terminated := dialog.server.terminatePendingInvite(dialog.inviteKey, 487)
	if terminated {
		dialog.mu.Lock()
		dialog.finalResponseStarted = true
		dialog.mu.Unlock()
	}
}

func (dialog *inboundDialog) RejectBeforeAnswer(statusCode int) {
	if dialog == nil {
		return
	}
	dialog.mu.Lock()
	if dialog.finalResponseStarted {
		dialog.mu.Unlock()
		return
	}
	dialog.mu.Unlock()
	dialog.StopRinging()
	dialog.sendSessionFinalResponse(statusCode)
}

func (dialog *inboundDialog) sendSessionFinalResponse(statusCode int) {
	dialog.mu.Lock()
	if dialog.finalResponseStarted {
		dialog.mu.Unlock()
		return
	}
	dialog.mu.Unlock()
	if dialog.dialogSession == nil || dialog.dialogSession.InviteRequest == nil {
		dialog.server.logger.Errorw("Inbound session final response skipped without dialog ownership",
			"call_id", dialog.callID,
			"status_code", statusCode)
		dialog.mu.Lock()
		dialog.finalResponseStarted = true
		dialog.mu.Unlock()
		return
	}

	response := sip.NewResponseFromRequest(dialog.dialogSession.InviteRequest, statusCode, "", nil)
	if response.Contact() == nil && dialog.server.listenConfig != nil {
		contactHeader := dialog.server.listenConfig.SIPContactHeader()
		response.AppendHeader(&contactHeader)
	}
	dialog.dialogSession.InviteResponse = response
	if err := dialog.transaction.Respond(response); err != nil {
		dialog.server.logger.Errorw("Failed to send inbound dialog final response",
			"error", err,
			"call_id", dialog.callID,
			"status_code", statusCode)
	}
	dialog.server.recordRejectedInboundInvite(dialog.request, response)
	dialog.mu.Lock()
	dialog.finalResponseStarted = true
	dialog.mu.Unlock()
}
