// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// handleCallFailed creates a short-lived observer to persist the FAILED status
// metric so the conversation is not left indeterminate. This handles early
// failures (outbound call rejected, setup error) that occur before the main
// SessionEstablished pipeline creates its own observer.
func (d *Dispatcher) handleCallFailed(ctx context.Context, v CallFailedPipeline) {
	if !validator.NonNil(v.Session) {
		d.logger.Warnw("Cannot record failed SIP call",
			"call_id", v.ID,
			"missing", "session",
			"error", fmt.Sprintf("%v", v.Error),
			"sip_code", v.SIPCode)
		return
	}
	auth := v.Session.GetAuth()
	if !validator.NonNil(auth) {
		d.logger.Warnw("Cannot record failed SIP call",
			"call_id", v.ID,
			"missing", "authentication",
			"error", fmt.Sprintf("%v", v.Error),
			"sip_code", v.SIPCode)
		return
	}

	assistant := v.Session.GetAssistant()
	if !validator.NonNil(assistant) {
		d.logger.Warnw("Cannot record failed SIP call",
			"call_id", v.ID,
			"missing", "assistant",
			"error", fmt.Sprintf("%v", v.Error),
			"sip_code", v.SIPCode)
		return
	}

	conversationID := v.Session.GetConversationID()
	if !validator.NonZero(conversationID) {
		d.logger.Warnw("Cannot record failed SIP call",
			"call_id", v.ID,
			"missing", "conversation",
			"assistant_id", assistant.Id,
			"error", fmt.Sprintf("%v", v.Error),
			"sip_code", v.SIPCode)
		return
	}

	errorMessage := "call_failed"
	if v.Error != nil {
		errorMessage = v.Error.Error()
	}

	callSetupResult := &CallSetupResult{
		AssistantID:    assistant.Id,
		ConversationID: conversationID,
		Auth:           auth,
	}
	if projectContext, err := auth.ProjectContext(); err == nil {
		callSetupResult.OrganizationID = projectContext.OrganizationID
		callSetupResult.ProjectID = projectContext.ProjectID
	} else if organizationContext, err := auth.OrganizationContext(); err == nil {
		callSetupResult.OrganizationID = organizationContext.OrganizationID
	}

	observer := d.createObserver(ctx, callSetupResult, auth)
	contextID := v.Session.GetContextID()
	if contextID == "" {
		contextID = v.ID
	}
	observer.Record(
		ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{
				AssistantID: assistant.Id,
			},
			ConversationID: conversationID,
		},
		observability.RecordLog{
			Level:   observability.LevelError,
			Message: "SIP call failed",
			Attributes: observability.Attributes{
				"provider":  "sip",
				"direction": string(v.Session.GetInfo().Direction),
				"call_id":   v.ID,
				"error":     errorMessage,
				"sip_code":  fmt.Sprintf("%d", v.SIPCode),
			},
		},
		observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallFailed,
			Attributes: observability.Attributes{
				"provider":  "sip",
				"direction": string(v.Session.GetInfo().Direction),
				"call_id":   v.ID,
				"error":     errorMessage,
				"sip_code":  fmt.Sprintf("%d", v.SIPCode),
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallFailed,
			ContextID: contextID,
			Payload: observability.CallFailedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
					"sip_code": fmt.Sprintf("%d", v.SIPCode),
				}),
				Provider:         "sip",
				CallID:           v.ID,
				Direction:        observability.WebhookCallDirection(v.Session.GetInfo().Direction),
				ContextID:        contextID,
				Error:            errorMessage,
				Status:           observability.WebhookCallStatusFailed,
				DisconnectReason: observability.WebhookCallDisconnectReasonInternalError,
			},
		},
		observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusFailed,
				Description: errorMessage,
			}},
		},
	)
	observer.Close(ctx)
}
