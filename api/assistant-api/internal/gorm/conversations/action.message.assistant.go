package internal_conversation_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

//

// ALTER TABLE assistant_conversation_actions rename column message_id to assistant_conversation_message_id;
// "assistant_conversation_message_id" of relation "assistant_conversation_actions" does
type AssistantConversationAction struct {
	gorm_model.Audited
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

func (acm *AssistantConversationAction) RequestMessage() *lexatic_backend.Message {
	out := &lexatic_backend.Message{}
	_ = utils.Cast(acm.Request, &out)
	return out
}

func (acm *AssistantConversationAction) ResponseMessage() *lexatic_backend.Message {
	out := &lexatic_backend.Message{}
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
