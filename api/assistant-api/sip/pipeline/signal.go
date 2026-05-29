// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emiago/sipgo"
	obs "github.com/rapidaai/api/assistant-api/internal/observe"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

const (
	reasonNoAnswer      = "no_answer"
	reasonBusy          = "busy"
	reasonRejected      = "rejected"
	reasonCancelled     = "cancelled"
	reasonAnsweredOther = "answered_by_other"
	reasonSIPError      = "sip_error"
	reasonMediaError    = "media_error"
	reasonInternalError = "internal_error"
	reasonUnknown       = "unknown"
	// maxExtensionLength is the maximum digit length treated as a SIP extension.
	maxExtensionLength = 6
)

var sipCodePattern = regexp.MustCompile(`\b([1-6][0-9]{2})\b`)

func (d *Dispatcher) emitLifecycleWebhook(ctx context.Context, eventType string, session *sip_infra.Session, payload map[string]interface{}) {
	if d.onLifecycleWebhook == nil {
		return
	}
	go d.onLifecycleWebhook(ctx, eventType, session, payload)
}

func parseSIPCode(err error, fallback int) int {
	if fallback > 0 {
		return fallback
	}
	if err == nil {
		return 0
	}
	var dialogErr *sipgo.ErrDialogResponse
	if errors.As(err, &dialogErr) && dialogErr != nil && dialogErr.Res != nil {
		return int(dialogErr.Res.StatusCode)
	}
	var sipErr *sip_infra.SIPError
	if errors.As(err, &sipErr) && sipErr != nil {
		if sipErr.Code > 0 {
			return sipErr.Code
		}
		if sipErr.Err != nil {
			return parseSIPCode(sipErr.Err, 0)
		}
	}
	matches := sipCodePattern.FindStringSubmatch(err.Error())
	if len(matches) == 2 {
		if code, convErr := strconv.Atoi(matches[1]); convErr == nil {
			return code
		}
	}
	return 0
}

func classifyCallFailure(err error, sipCode int) (string, string) {
	code := parseSIPCode(err, sipCode)
	if code == 486 || code == 600 {
		return "call.busy", reasonBusy
	}
	if code == 480 || code == 408 {
		return "call.no_answer", reasonNoAnswer
	}
	if code == 603 || code == 403 {
		return "call.rejected", reasonRejected
	}

	msg := ""
	if err != nil {
		msg = strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(msg, "busy"):
		return "call.busy", reasonBusy
	case strings.Contains(msg, "decline") || strings.Contains(msg, "reject"):
		return "call.rejected", reasonRejected
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") || strings.Contains(msg, "no answer") || strings.Contains(msg, "unavailable"):
		return "call.no_answer", reasonNoAnswer
	case strings.Contains(msg, "rtp") || strings.Contains(msg, "media"):
		return "call.failed", reasonMediaError
	case strings.Contains(msg, "cancel"):
		return "call.cancelled", reasonCancelled
	case strings.Contains(msg, "sip"):
		return "call.failed", reasonSIPError
	case msg != "":
		return "call.failed", reasonInternalError
	default:
		return "call.failed", reasonUnknown
	}
}

func classifyTransferAttemptFailure(err error) (string, string) {
	if err == nil {
		return "failed", reasonUnknown
	}
	code := parseSIPCode(err, 0)
	if code == 486 || code == 600 {
		return "busy", reasonBusy
	}
	if code == 480 || code == 408 {
		return "no_answer", reasonNoAnswer
	}
	if code == 603 || code == 403 {
		return "rejected", reasonRejected
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "busy"):
		return "busy", reasonBusy
	case strings.Contains(msg, "decline") || strings.Contains(msg, "reject"):
		return "rejected", reasonRejected
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") || strings.Contains(msg, "no answer") || strings.Contains(msg, "unavailable"):
		return "no_answer", reasonNoAnswer
	case strings.Contains(msg, "cancel"):
		return "cancelled", reasonCancelled
	default:
		return "failed", reasonSIPError
	}
}

func sessionURIs(session *sip_infra.Session) (string, string) {
	if session == nil {
		return "", ""
	}
	var fromURI, toURI string
	if from, ok := session.GetMetadata(sip_infra.MetadataCallFromURI); ok {
		if s, ok := from.(string); ok {
			fromURI = s
		}
	}
	if to, ok := session.GetMetadata(sip_infra.MetadataCallToURI); ok {
		if s, ok := to.(string); ok {
			toURI = s
		}
	}
	return fromURI, toURI
}

func transferDirection(session *sip_infra.Session) string {
	if session == nil {
		return ""
	}
	return string(session.GetInfo().Direction)
}

func transferCallPayload(session *sip_infra.Session, state, reason string) map[string]interface{} {
	fromURI, toURI := sessionURIs(session)
	payload := map[string]interface{}{
		"provider":        "sip",
		"call_id":         "",
		"conversation_id": nil,
		"assistant_id":    nil,
		"project_id":      nil,
		"organization_id": nil,
		"direction":       transferDirection(session),
		"from_uri":        fromURI,
		"to_uri":          toURI,
		"from_number":     sip_infra.ExtractDIDFromURI(fromURI),
		"to_number":       sip_infra.ExtractDIDFromURI(toURI),
		"session_id":      "",
		"state":           state,
		"reason":          reason,
		"metadata":        map[string]interface{}{},
	}
	if session == nil {
		return payload
	}
	payload["call_id"] = session.GetCallID()
	payload["session_id"] = session.GetCallID()
	if convID := session.GetConversationID(); convID > 0 {
		payload["conversation_id"] = fmt.Sprintf("%d", convID)
	}
	if assistant := session.GetAssistant(); assistant != nil {
		payload["assistant_id"] = fmt.Sprintf("%d", assistant.Id)
	}
	if auth := session.GetAuth(); auth != nil {
		if pid := auth.GetCurrentProjectId(); pid != nil {
			payload["project_id"] = fmt.Sprintf("%d", *pid)
		}
		if oid := auth.GetCurrentOrganizationId(); oid != nil {
			payload["organization_id"] = fmt.Sprintf("%d", *oid)
		}
	}
	return payload
}

func targetExtensionFromURI(uri string) string {
	if uri == "" {
		return ""
	}
	raw := strings.TrimPrefix(strings.TrimPrefix(uri, "sip:"), "sips:")
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	user := parts[0]
	if idx := strings.IndexByte(user, ';'); idx >= 0 {
		user = user[:idx]
	}
	user = strings.TrimPrefix(user, "+")
	for _, ch := range user {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	if len(user) > 0 && len(user) <= maxExtensionLength {
		return user
	}
	return ""
}

func transferEventPayload(session *sip_infra.Session, state, reason, transferID, routingMode string) map[string]interface{} {
	if routingMode == "" {
		routingMode = "unknown"
	}
	payload := transferCallPayload(session, state, reason)
	payload["transfer_id"] = transferID
	payload["attempt_id"] = nil
	payload["outbound_call_id"] = nil
	payload["routing_mode"] = routingMode
	payload["target_uri"] = nil
	payload["target_number"] = nil
	payload["target_extension"] = nil
	payload["target_user_id"] = nil
	payload["attempt"] = nil
	payload["total_attempts"] = nil
	payload["answered_by"] = nil
	return payload
}

func (d *Dispatcher) handleCallCreated(ctx context.Context, v sip_infra.CallCreatedPipeline) {
	d.logger.Infow("Pipeline: CallCreated", "call_id", v.ID)
	payload := transferCallPayload(v.Session, "created", "")
	payload["from_uri"] = v.FromURI
	payload["to_uri"] = v.ToURI
	payload["from_number"] = sip_infra.ExtractDIDFromURI(v.FromURI)
	payload["to_number"] = sip_infra.ExtractDIDFromURI(v.ToURI)
	d.emitLifecycleWebhook(ctx, "call.created", v.Session, payload)
}

func (d *Dispatcher) handleCallRinging(ctx context.Context, v sip_infra.CallRingingPipeline) {
	d.logger.Infow("Pipeline: CallRinging", "call_id", v.ID)
	payload := transferCallPayload(v.Session, "ringing", "")
	payload["from_uri"] = v.FromURI
	payload["to_uri"] = v.ToURI
	payload["from_number"] = sip_infra.ExtractDIDFromURI(v.FromURI)
	payload["to_number"] = sip_infra.ExtractDIDFromURI(v.ToURI)
	d.emitLifecycleWebhook(ctx, "call.ringing", v.Session, payload)
}

func (d *Dispatcher) handleCallAnswered(ctx context.Context, v sip_infra.CallAnsweredPipeline) {
	d.logger.Infow("Pipeline: CallAnswered", "call_id", v.ID)
	payload := transferCallPayload(v.Session, "answered", "")
	payload["from_uri"] = v.FromURI
	payload["to_uri"] = v.ToURI
	payload["from_number"] = sip_infra.ExtractDIDFromURI(v.FromURI)
	payload["to_number"] = sip_infra.ExtractDIDFromURI(v.ToURI)
	d.emitLifecycleWebhook(ctx, "call.answered", v.Session, payload)
}

func (d *Dispatcher) handleCallMediaStarted(ctx context.Context, v sip_infra.CallMediaStartedPipeline) {
	d.logger.Infow("Pipeline: CallMediaStarted", "call_id", v.ID)
	payload := transferCallPayload(v.Session, "media_started", "")
	d.emitLifecycleWebhook(ctx, "call.media.started", v.Session, payload)
}

func (d *Dispatcher) handleByeReceived(ctx context.Context, v sip_infra.ByeReceivedPipeline) {
	d.logger.Infow("Pipeline: ByeReceived",
		"call_id", v.ID,
		"reason", v.Reason,
		"direction", v.Session.GetInfo().Direction)
	reason := v.Reason
	if reason == "" {
		reason = "remote_bye"
	}
	d.emitLifecycleWebhook(ctx, "call.ended", v.Session, transferCallPayload(v.Session, "ended", reason))
}

func (d *Dispatcher) handleCancelReceived(ctx context.Context, v sip_infra.CancelReceivedPipeline) {
	d.logger.Infow("Pipeline: CancelReceived",
		"call_id", v.ID,
		"direction", v.Session.GetInfo().Direction)
	d.emitLifecycleWebhook(ctx, "call.cancelled", v.Session, transferCallPayload(v.Session, "ended", "cancelled"))
}

func (d *Dispatcher) handleCallEnded(ctx context.Context, v sip_infra.CallEndedPipeline) {
	d.logger.Infow("Pipeline: CallEnded",
		"call_id", v.ID,
		"duration", v.Duration,
		"reason", v.Reason)
}

// handleCallFailed creates a short-lived observer to persist the FAILED status
// metric so the conversation is not left indeterminate. This handles early
// failures (outbound call rejected, setup error) that occur before the main
// SessionEstablished pipeline creates its own observer.
func (d *Dispatcher) handleCallFailed(ctx context.Context, v sip_infra.CallFailedPipeline) {
	d.logger.Warnw("Pipeline: CallFailed",
		"call_id", v.ID,
		"error", fmt.Sprintf("%v", v.Error),
		"sip_code", v.SIPCode)

	// Emit failure metric via observer if session has enough context
	sipCode := parseSIPCode(v.Error, v.SIPCode)
	if v.Session == nil {
		eventType, normalizedReason := classifyCallFailure(v.Error, sipCode)
		meta := map[string]interface{}{}
		if sipCode > 0 {
			meta["sip_code"] = fmt.Sprintf("%d", sipCode)
		}
		d.emitLifecycleWebhook(ctx, eventType, nil, map[string]interface{}{
			"provider":        "sip",
			"call_id":         v.ID,
			"conversation_id": nil,
			"assistant_id":    nil,
			"project_id":      nil,
			"organization_id": nil,
			"direction":       "",
			"from_uri":        "",
			"to_uri":          "",
			"from_number":     "",
			"to_number":       "",
			"session_id":      v.ID,
			"state":           "failed",
			"reason":          normalizedReason,
			"metadata":        meta,
		})
		return
	}
	eventType, normalizedReason := classifyCallFailure(v.Error, sipCode)
	callFailedPayload := transferCallPayload(v.Session, "failed", normalizedReason)
	if sipCode > 0 {
		meta := callFailedPayload["metadata"].(map[string]interface{})
		meta["sip_code"] = fmt.Sprintf("%d", sipCode)
	}
	if v.Error != nil {
		meta := callFailedPayload["metadata"].(map[string]interface{})
		meta["error"] = v.Error.Error()
	}
	d.emitLifecycleWebhook(ctx, eventType, v.Session, callFailedPayload)
	auth := v.Session.GetAuth()
	convID := v.Session.GetConversationID()
	if auth == nil || convID == 0 {
		return
	}

	var assistantID uint64
	if assistant := v.Session.GetAssistant(); assistant != nil {
		assistantID = assistant.Id
	}

	setup := &CallSetupResult{
		AssistantID:    assistantID,
		ConversationID: convID,
	}
	if auth.GetCurrentProjectId() != nil {
		setup.ProjectID = *auth.GetCurrentProjectId()
	}
	if auth.GetCurrentOrganizationId() != nil {
		setup.OrganizationID = *auth.GetCurrentOrganizationId()
	}

	if d.onCreateObserver != nil {
		observer := d.onCreateObserver(ctx, setup, auth)
		if observer != nil {
			eventData := map[string]string{
				obs.DataType:      obs.EventCallFailed,
				obs.DataProvider:  "sip",
				obs.DataReason:    normalizedReason,
				obs.DataDirection: string(v.Session.GetInfo().Direction),
			}
			if sipCode > 0 {
				eventData["sip_code"] = fmt.Sprintf("%d", sipCode)
			}
			if v.Error != nil {
				eventData[obs.DataError] = v.Error.Error()
			}
			observer.EmitMetric(ctx, obs.CallStatusMetric("FAILED", normalizedReason))
			observer.EmitEvent(ctx, obs.ComponentTelephony, eventData)
			observer.Shutdown(ctx)
		}
	}
}

func (d *Dispatcher) handleTransferRequested(ctx context.Context, v sip_infra.TransferRequestedPipeline) {
	routingMode := v.RoutingMode
	if routingMode == "" {
		routingMode = "unknown"
	}
	payload := transferEventPayload(v.Session, "requested", "", v.TransferID, routingMode)
	payload["attempt"] = 0
	payload["total_attempts"] = len(v.Targets)
	payload["metadata"] = map[string]interface{}{
		"post_transfer_action": v.PostTransferAction,
	}
	d.emitLifecycleWebhook(ctx, "transfer.requested", v.Session, payload)
}

func (d *Dispatcher) handleTransferAttemptStarted(ctx context.Context, v sip_infra.TransferAttemptStartedPipeline) {
	payload := transferEventPayload(v.Session, "attempting", "", v.TransferID, v.RoutingMode)
	payload["attempt_id"] = fmt.Sprintf("%s:%d", v.TransferID, v.Attempt)
	payload["target_uri"] = v.TargetURI
	payload["target_number"] = sip_infra.ExtractDIDFromURI(v.TargetURI)
	payload["target_extension"] = targetExtensionFromURI(v.TargetURI)
	payload["target_user_id"] = nil
	payload["attempt"] = v.Attempt
	payload["total_attempts"] = v.Total
	d.emitLifecycleWebhook(ctx, "transfer.attempt.started", v.Session, payload)
}

func (d *Dispatcher) handleTransferTargetRinging(ctx context.Context, v sip_infra.TransferTargetRingingPipeline) {
	payload := transferEventPayload(v.Session, "ringing", "", v.TransferID, v.RoutingMode)
	payload["attempt_id"] = fmt.Sprintf("%s:%d", v.TransferID, v.Attempt)
	payload["target_uri"] = v.TargetURI
	payload["target_number"] = sip_infra.ExtractDIDFromURI(v.TargetURI)
	payload["target_extension"] = targetExtensionFromURI(v.TargetURI)
	payload["attempt"] = v.Attempt
	payload["total_attempts"] = v.Total
	d.emitLifecycleWebhook(ctx, "transfer.target.ringing", v.Session, payload)
}

func (d *Dispatcher) handleTransferConnected(ctx context.Context, v sip_infra.TransferConnectedPipeline) {
	outboundInfo := v.OutboundSession.GetInfo()
	d.logger.Infow("Pipeline: transfer_connected",
		"call_id", v.ID,
		"outbound_call_id", v.OutboundSession.GetCallID(),
		"target_uri", outboundInfo.RemoteURI,
		"codec", outboundInfo.Codec)
	targetURI := v.TargetURI
	if targetURI == "" {
		targetURI = outboundInfo.RemoteURI
	}
	payload := transferEventPayload(v.InboundSession, "connected", "target_answered", v.TransferID, v.RoutingMode)
	payload["outbound_call_id"] = v.OutboundSession.GetCallID()
	payload["target_uri"] = targetURI
	payload["target_number"] = sip_infra.ExtractDIDFromURI(targetURI)
	payload["target_extension"] = targetExtensionFromURI(targetURI)
	payload["target_user_id"] = nil
	payload["attempt"] = v.Attempt
	payload["total_attempts"] = v.TotalAttempts
	if v.Attempt > 0 {
		payload["attempt_id"] = fmt.Sprintf("%s:%d", v.TransferID, v.Attempt)
	}
	d.emitLifecycleWebhook(ctx, "transfer.connected", v.InboundSession, payload)
}

func (d *Dispatcher) handleTransferAttemptEnded(ctx context.Context, v sip_infra.TransferAttemptEndedPipeline) {
	payload := transferEventPayload(v.Session, v.State, v.Reason, v.TransferID, v.RoutingMode)
	payload["attempt_id"] = v.AttemptID
	payload["outbound_call_id"] = v.OutboundCallID
	payload["target_uri"] = v.TargetURI
	payload["target_number"] = sip_infra.ExtractDIDFromURI(v.TargetURI)
	payload["target_extension"] = targetExtensionFromURI(v.TargetURI)
	payload["attempt"] = v.Attempt
	payload["total_attempts"] = v.Total
	payload["answered_by"] = v.AnsweredBy
	if v.Metadata != nil {
		payload["metadata"] = v.Metadata
	}
	d.emitLifecycleWebhook(ctx, "transfer.attempt.ended", v.Session, payload)
}

func (d *Dispatcher) handleTransferCancelled(ctx context.Context, v sip_infra.TransferCancelledPipeline) {
	payload := transferEventPayload(v.Session, "cancelled", v.Reason, v.TransferID, v.RoutingMode)
	if v.TransferID != "" && v.Attempt > 0 {
		payload["attempt_id"] = fmt.Sprintf("%s:%d", v.TransferID, v.Attempt)
	}
	payload["target_uri"] = v.TargetURI
	payload["target_number"] = sip_infra.ExtractDIDFromURI(v.TargetURI)
	payload["target_extension"] = targetExtensionFromURI(v.TargetURI)
	payload["attempt"] = v.Attempt
	payload["total_attempts"] = v.Total
	payload["answered_by"] = v.AnsweredBy
	if v.Metadata != nil {
		payload["metadata"] = v.Metadata
	}
	d.emitLifecycleWebhook(ctx, "transfer.cancelled", v.Session, payload)
}

func (d *Dispatcher) handleTransferFailed(ctx context.Context, v sip_infra.TransferFailedPipeline) {
	// Categorize the failure for structured alerting
	category := categorizeTransferError(v.Reason, v.Error)
	d.logger.Warnw("Pipeline: transfer_failed",
		"call_id", v.ID,
		"reason", v.Reason,
		"category", category,
		"error", v.Error)
	reason := strings.TrimSpace(v.Reason)
	if reason == "" && v.Error != nil {
		reason = v.Error.Error()
	}
	payload := transferEventPayload(v.Session, "failed", reason, v.TransferID, v.RoutingMode)
	payload["outbound_call_id"] = nil
	payload["target_uri"] = nil
	payload["target_number"] = nil
	payload["target_extension"] = nil
	payload["target_user_id"] = nil
	payload["attempt"] = nil
	payload["total_attempts"] = nil
	d.emitLifecycleWebhook(ctx, "transfer.failed", v.Session, payload)
}
