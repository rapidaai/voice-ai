// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_sip

import (
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

func TestAllowSIPWebhookCondition_NoCondition_Allows(t *testing.T) {
	webhook := &internal_assistant_entity.AssistantWebhook{}
	if !allowSIPWebhookCondition(webhook, "inbound") {
		t.Fatal("expected webhook with no condition to be allowed")
	}
}

func TestAllowSIPWebhookCondition_ValidDirectionCondition(t *testing.T) {
	webhook := &internal_assistant_entity.AssistantWebhook{
		AssistantWebhookOption: []*internal_assistant_entity.AssistantWebhookOption{
			{
				Metadata: gorm_model.Metadata{
					Key:   "webhook.condition",
					Value: `[{"key":"source","condition":"=","value":"phone"},{"key":"mode","condition":"=","value":"voice"},{"key":"direction","condition":"=","value":"inbound"}]`,
				},
			},
		},
	}
	if !allowSIPWebhookCondition(webhook, "inbound") {
		t.Fatal("expected inbound direction to satisfy webhook.condition")
	}
	if allowSIPWebhookCondition(webhook, "outbound") {
		t.Fatal("expected outbound direction to fail webhook.condition")
	}
}

func TestAllowSIPWebhookCondition_InvalidCondition_Blocks(t *testing.T) {
	webhook := &internal_assistant_entity.AssistantWebhook{
		AssistantWebhookOption: []*internal_assistant_entity.AssistantWebhookOption{
			{
				Metadata: gorm_model.Metadata{
					Key:   "webhook.condition",
					Value: `{invalid_json`,
				},
			},
		},
	}
	if allowSIPWebhookCondition(webhook, "inbound") {
		t.Fatal("expected invalid webhook.condition to be blocked")
	}
}
