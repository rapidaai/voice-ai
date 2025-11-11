package internal_assistant_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

type AssistantTool struct {
	gorm_model.Audited
	gorm_model.Mutable
	AssistantId      uint64                  `json:"assistantId" gorm:"type:bigint;not null"`
	Name             string                  `json:"name" gorm:"type:bigint;not null"`
	Description      string                  `json:"description" gorm:"type:bigint;not null"`
	Fields           gorm_types.InterfaceMap `json:"fields" gorm:"type:string;size:200;not null;"`
	ExecutionMethod  string                  `json:"executionMethod" gorm:"type:string;size:50;not null;"`
	ExecutionOptions []*AssistantToolOption  `json:"executionOptions" gorm:"foreignKey:AssistantToolId"`
}

// CREATE TABLE assistant_tools (
//     id bigint NOT NULL,
//     created_date timestamp without time zone NOT NULL DEFAULT now(),
//     updated_date timestamp without time zone,
//     status character varying(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
//     created_by bigint NOT NULL,
//     updated_by bigint,
//     assistant_id bigint NOT NULL,
//     name character varying(50) NOT NULL,
// 	description character varying(400) NOT NULL,
// 	fields TEXT NOT NULL,
// 	execution_method character varying(200) NOT NULL,
//     CONSTRAINT assistant_tools_pkey PRIMARY KEY (id),
//     CONSTRAINT assistant_tools_assistant_id_name_key UNIQUE (assistant_id, name)
// );

type AssistantToolOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantToolId uint64 `json:"assistantToolId" gorm:"type:bigint;size:20"`
}

func (a *AssistantTool) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	for _, v := range a.ExecutionOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

// CREATE TABLE assistant_tool_options (
//     id BIGINT PRIMARY KEY NOT NULL,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT NOT NULL,
//     updated_by BIGINT,
//     created_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     updated_date TIMESTAMP DEFAULT NULL,
//     key VARCHAR(200) NOT NULL,
//     value TEXT NOT NULL,
//     assistant_tool_id BIGINT NOT NULL,
//     CONSTRAINT uk_assistant_tool_id UNIQUE (key, assistant_tool_id)
// );

type AssistantToolLog struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	AssistantId                    uint64         `json:"assistantId" gorm:"type:bigint"`
	AssistantConversationId        uint64         `json:"assistantConversationId" gorm:"type:bigint"`
	AssistantConversationMessageId string         `json:"assistantConversationMessageId" gorm:"type:string;not null"`
	AssistantToolId                uint64         `json:"assistantToolId" gorm:"type:bigint"`
	AssistantToolName              string         `json:"assistantToolName" gorm:"type:string"`
	ExecutionMethod                string         `json:"executionMethod" gorm:"type:string"`
	AssetPrefix                    string         `json:"assetPrefix" gorm:"type:string;size:200;not null"`
	TimeTaken                      int64          `json:"timeTaken" gorm:"type:bigint;size:20"`
	AssistantTool                  *AssistantTool `json:"assistantTool" gorm:"foreignKey:AssistantToolId"`
}

//  CREATE TABLE assistant_tool_logs (
//     id BIGINT PRIMARY KEY,
//     created_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     updated_date TIMESTAMP WITH TIME ZONE,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT,
//     updated_by BIGINT,
//     project_id BIGINT NOT NULL,
//     organization_id BIGINT NOT NULL,
// 		assistant_tool_id BIGINT NOT NULL,
// 		assistant_tool_name VARCHAR(255) NOT NULL,
//     assistant_id BIGINT,
//     assistant_conversation_id BIGINT,
//     assistant_conversation_message_id VARCHAR(255) NOT NULL,
//     execution_method VARCHAR(20),
//     asset_prefix VARCHAR(200) NOT NULL,
//     time_taken BIGINT
// );
