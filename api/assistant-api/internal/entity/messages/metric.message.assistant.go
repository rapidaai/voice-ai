// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_message_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type AssistantConversationMessageMetric struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metric
	AssistantConversationId        uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
	AssistantConversationMessageId string `json:"assistantConversationMessageId" gorm:"type:string;not null"`
}
