package internal_conversation_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type AssistantConversationMetric struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metric
	AssistantConversationId uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
}
