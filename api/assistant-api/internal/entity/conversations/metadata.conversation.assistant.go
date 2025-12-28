// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_conversation_gorm

import gorm "github.com/rapidaai/pkg/models/gorm"

type AssistantConversationMetadata struct {
	gorm.Audited
	gorm.Mutable
	gorm.Metadata
	AssistantId             uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantConversationId uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
}
