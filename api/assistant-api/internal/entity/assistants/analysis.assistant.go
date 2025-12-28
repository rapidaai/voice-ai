// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type AssistantAnalysis struct {
	gorm_model.Audited
	gorm_model.Mutable

	AssistantId uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	Name        string `json:"name" gorm:"type:text"`
	Description string `json:"description" gorm:"type:text"`

	EndpointId         uint64               `json:"endpointId" gorm:"type:bigint;not null"`
	EndpointVersion    string               `json:"endpointVersion" gorm:"type:text"`
	EndpointParameters gorm_types.StringMap `json:"endpointParameters" gorm:"type:string;"`
	ExecutionPriority  uint32               `json:"executionPriority" gorm:"type:int"`
}

func (aa *AssistantAnalysis) GetName() string {
	return aa.Name
}

func (aa *AssistantAnalysis) GetEndpointId() uint64 {
	return aa.EndpointId

}

func (aa *AssistantAnalysis) GetEndpointVersion() string {
	return aa.EndpointVersion
}

func (aa *AssistantAnalysis) GetExecutionPriority() uint32 {
	return aa.ExecutionPriority
}

func (aa *AssistantAnalysis) GetParameters() map[string]string {
	return aa.EndpointParameters
}
