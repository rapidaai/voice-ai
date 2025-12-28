// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_message_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
)

type AssistantConversationMessage struct {
	gorm_model.Audited
	gorm_model.Mutable
	MessageId                string                  `json:"messageId" gorm:"type:string;not null"`
	AssistantConversationId  uint64                  `json:"assistantConversationId" gorm:"type:bigint;not null"`
	AssistantId              uint64                  `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantProviderModelId uint64                  `json:"assistantProviderModelId" gorm:"type:bigint;not null"`
	Request                  gorm_types.InterfaceMap `json:"request" gorm:"type:jsonb"`
	Response                 gorm_types.InterfaceMap `json:"response" gorm:"type:jsonb"`
	Source                   string                  `json:"source" gorm:"type:string;size:50;not null"`

	Metadatas []*AssistantConversationMessageMetadata `json:"metadata" gorm:"foreignKey:AssistantConversationMessageId;references:MessageId"`
	Metrics   []*AssistantConversationMessageMetric   `json:"metrics" gorm:"foreignKey:AssistantConversationMessageId;references:MessageId"`
}

func (acm *AssistantConversationMessage) SetRequest(message *types.Message) {
	acm.Request = map[string]interface{}{
		"role":      message.GetRole(),
		"contents":  message.GetContents(),
		"toolCalls": message.GetToolCalls(),
	}
}

func (acm *AssistantConversationMessage) SetResponse(message *types.Message) {
	acm.Response = map[string]interface{}{
		"role":      message.GetRole(),
		"contents":  message.GetContents(),
		"toolCalls": message.GetToolCalls(),
	}
}
