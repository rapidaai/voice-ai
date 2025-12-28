// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import (
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

type AssistantProvider struct {
	gorm_model.Audited
	AssistantId uint64 `json:"assistantId" gorm:"type:bigint;size:20"`
	CreatedBy   uint64 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	Status      string `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
	Description string `json:"description" gorm:"type:string"`
}

type AssistantProviderAgentkit struct {
	AssistantProvider
	//
	Url         string               `json:"url" gorm:"type:string"`
	Certificate string               `json:"certificate" gorm:"type:string;size:400;not null;"`
	Metadata    gorm_types.StringMap `json:"metadata" gorm:"type:string;size:400;not null;"`
}

type AssistantProviderWebsocket struct {
	AssistantProvider

	//
	Url        string               `json:"url" gorm:"type:string"`
	Headers    gorm_types.StringMap `json:"headers" gorm:"type:string;size:400;not null;"`
	Parameters gorm_types.StringMap `json:"parameters" gorm:"type:string;size:400;not null;"`
}

type AssistantProviderModel struct {
	AssistantProvider
	//
	Template              gorm_types.PromptMap            `json:"template" gorm:"type:jsonb"`
	AssistantId           uint64                          `json:"assistantId" gorm:"type:bigint;size:20"`
	ModelProviderName     string                          `json:"modelProviderName" gorm:"type:string"`
	AssistantModelOptions []*AssistantProviderModelOption `json:"assistantModelOptions" gorm:"foreignKey:AssistantProviderModelId"`
}

type AssistantProviderModelOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantProviderModelId uint64 `json:"AssistantProviderModelId" gorm:"type:bigint;size:20"`
}

func (a *AssistantProviderModel) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	for _, v := range a.AssistantModelOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

func (epm *AssistantProviderModel) SetPrompt(promptString string) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(promptString), &jsonData)
	if err != nil {
		return
	}
	epm.Template = jsonData
}
