// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_sip

import (
	"context"
	"fmt"
	"slices"
	"time"

	internal_condition "github.com/rapidaai/api/assistant-api/internal/condition"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_webhook "github.com/rapidaai/api/assistant-api/internal/webhook"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

type sipWebhookCallback struct {
	httpLogService internal_services.AssistantHTTPLogService
	auth           types.SimplePrinciple
	assistantID    uint64
	conversationID *uint64
}

func (c *sipWebhookCallback) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	for _, pkt := range pkts {
		httpPkt, ok := pkt.(internal_type.HTTPLogCreatePacket)
		if !ok {
			continue
		}
		_, err := c.httpLogService.CreateLog(
			ctx,
			c.auth,
			httpPkt.Source,
			httpPkt.SourceRefID,
			httpPkt.SourceEvent,
			httpPkt.ContextID,
			c.assistantID,
			c.conversationID,
			httpPkt.HTTPURL,
			httpPkt.HTTPMethod,
			httpPkt.ResponseStatus,
			httpPkt.TimeTaken,
			httpPkt.RetryCount,
			httpPkt.Status,
			httpPkt.ErrorMessage,
			httpPkt.RequestPayload,
			httpPkt.ResponsePayload,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *SIPEngine) publishSIPLifecycleWebhook(ctx context.Context, eventType string, session *sip_infra.Session, payload map[string]interface{}) {
	if session == nil {
		return
	}

	auth := session.GetAuth()
	assistant := session.GetAssistant()
	if auth == nil || assistant == nil {
		return
	}

	if len(assistant.AssistantWebhooks) == 0 {
		loaded, err := m.assistantService.Get(ctx, auth, assistant.Id, utils.GetVersionDefinition("latest"),
			&internal_services.GetAssistantOption{InjectWebhook: true})
		if err == nil && loaded != nil {
			assistant = loaded
			session.SetAssistant(loaded)
		}
	}
	if len(assistant.AssistantWebhooks) == 0 {
		return
	}

	data := payload
	if data == nil {
		data = map[string]interface{}{}
	}
	envelope := map[string]interface{}{
		"id":        fmt.Sprintf("%d", gorm_generator.ID()),
		"type":      eventType,
		"timestamp": time.Now().UnixMilli(),
		"data":      data,
	}

	var conversationID *uint64
	if convID := session.GetConversationID(); convID > 0 {
		conversationID = &convID
	}
	cb := &sipWebhookCallback{
		httpLogService: m.assistantHTTPLogService,
		auth:           auth,
		assistantID:    assistant.Id,
		conversationID: conversationID,
	}

	for _, webhook := range assistant.AssistantWebhooks {
		if webhook == nil {
			continue
		}
		if !slices.Contains(webhook.GetAssistantEvents(), eventType) {
			continue
		}
		if !allowSIPWebhookCondition(webhook, string(session.GetInfo().Direction)) {
			continue
		}

		exec, err := internal_webhook.NewExecutor(m.logger, ctx, webhook, cb, nil)
		if err != nil {
			m.logger.Warnw("SIP lifecycle webhook: executor creation failed",
				"event", eventType,
				"webhook_id", webhook.Id,
				"error", err)
			continue
		}
		err = exec.Execute(ctx, internal_type.ExecuteWebhookPacket{
			ContextID: session.GetCallID(),
			Event:     utils.AssistantWebhookEvent(eventType),
			Arguments: envelope,
		})
		if err != nil {
			m.logger.Warnw("SIP lifecycle webhook execution failed",
				"event", eventType,
				"webhook_id", webhook.Id,
				"error", err)
		}
		_ = exec.Close(ctx)
	}
}

func allowSIPWebhookCondition(webhook *internal_assistant_entity.AssistantWebhook, direction string) bool {
	if webhook == nil {
		return false
	}
	raw, err := webhook.GetOptions().GetString("webhook.condition")
	if err != nil {
		return true
	}
	parsed, parseErr := internal_condition.Parse(raw)
	if parseErr != nil {
		return false
	}
	allowed, evalErr := parsed.Run(
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeSource, Value: utils.PhoneCall.Get()},
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeMode, Value: type_enums.AudioMode.String()},
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeDirection, Value: direction},
	)
	if evalErr != nil {
		return false
	}
	return allowed
}

