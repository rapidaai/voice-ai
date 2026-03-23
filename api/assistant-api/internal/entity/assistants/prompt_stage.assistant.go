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
)

type AssistantPromptStage struct {
	gorm_model.Audited
	gorm_model.Mutable
	AssistantProviderModelId uint64               `json:"assistantProviderModelId" gorm:"type:bigint;size:20;not null"`
	Name                     string               `json:"name" gorm:"type:string;size:100;not null"`
	Description              string               `json:"description" gorm:"type:string"`
	Template                 gorm_types.PromptMap `json:"template" gorm:"type:jsonb"`
	IsDefault                bool                 `json:"isDefault" gorm:"type:boolean;default:false"`
	Order                    int                  `json:"order" gorm:"type:int;default:0"`
	TransitionRules          string               `json:"transitionRules" gorm:"type:text"`
}

func (aps *AssistantPromptStage) SetPrompt(promptString string) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(promptString), &jsonData)
	if err != nil {
		return
	}
	aps.Template = jsonData
}

func (aps *AssistantPromptStage) GetTemplate() map[string]interface{} {
	return aps.Template
}
