package internal_message_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

// CREATE TABLE assistant_conversation_message_metrics (
//     id BIGINT PRIMARY KEY NOT NULL,
//     assistant_conversation_id BIGINT NOT NULL,
//     assistant_conversation_message_id VARCHAR(50) NOT NULL,
//     name VARCHAR(200) NOT NULL,
//     value TEXT NOT NULL,
//     description TEXT,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT NOT NULL,
//     updated_by BIGINT,
//     created_date TIMESTAMP NOT NULL DEFAULT NOW(),
//     updated_date TIMESTAMP DEFAULT NULL,
//     CONSTRAINT uk_assistant_conversation_message_metrics UNIQUE (assistant_conversation_message_id, name)
// );
// CREATE INDEX idx_assistant_conversation_message_id
// ON assistant_conversation_message_metrics (assistant_conversation_message_id);

type AssistantConversationMessageMetric struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metric
	AssistantConversationId        uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
	AssistantConversationMessageId string `json:"assistantConversationMessageId" gorm:"type:string;not null"`
}
