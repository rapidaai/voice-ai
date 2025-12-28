// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_conversation_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

//

type AssistantConversationAction struct {
	gorm_model.Audited
	AssistantId                    uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantConversationId        uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
	AssistantConversationMessageId string `json:"assistantConversationMessageId" gorm:"type:string;not null"`

	//
	Metrics []*AssistantConversationActionMetric `json:"metrics" gorm:"foreignKey:AssistantConversationActionId"`

	//
	ExternalId string `json:"externalId" gorm:"external_id"`
	// type will be llm
	ActionType type_enums.MessageAction `json:"actionType" gorm:"type:string;not null"`
	Request    gorm_types.InterfaceMap  `json:"request" gorm:"type:text;not null"`
	Response   gorm_types.InterfaceMap  `json:"response" gorm:"type:text"`
	Status     type_enums.RecordState   `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
}

type AssistantConversationActionMetric struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metric
	AssistantConversationActionId  uint64 `json:"assistantConversationActionId" gorm:"type:bigint;not null"`
	AssistantConversationId        uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
	AssistantConversationMessageId string `json:"assistantConversationMessageId" gorm:"type:string;not null"`
}

func (acm *AssistantConversationAction) RequestMessage() *protos.Message {
	out := &protos.Message{}
	_ = utils.Cast(acm.Request, &out)
	return out
}

func (acm *AssistantConversationAction) ResponseMessage() *protos.Message {
	out := &protos.Message{}
	_ = utils.Cast(acm.Response, &out)
	return out
}

func (acm *AssistantConversationAction) SetLLMCall(in, out *types.Message) {
	acm.ActionType = type_enums.ACTION_LLM_CALL
	acm.Request = map[string]interface{}{
		"role":      in.GetRole(),
		"contents":  in.GetContents(),
		"toolCalls": in.GetToolCalls(),
	}
	acm.Response = map[string]interface{}{
		"role":      out.GetRole(),
		"contents":  out.GetContents(),
		"toolCalls": out.GetToolCalls(),
	}
}

func (acm *AssistantConversationAction) SetToolCall(in, out map[string]interface{}) {
	acm.ActionType = type_enums.ACTION_TOOL_CALL
	acm.Request = in
	acm.Response = out
}
