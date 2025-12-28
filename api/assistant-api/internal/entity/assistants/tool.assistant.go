// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

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
